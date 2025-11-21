## 0. High-Level Behavior

Binary name (example): `helm`

Commands:

* `helm scaffold`
* `helm run`
* `helm spec` (splitting)
* `helm status`

All four commands are **TUI-first** (Bubble Tea). They may accept flags to run in non-interactive mode later, but v1 focuses on interactive flows.

Specs live under a configurable root (default `docs/specs`).

---

## 1. Project Structure (Go + Bubble Tea)

Proposed repo layout for the CLI (inside some project, or as a separate module):

```text
.
├─ cmd/
│  └─ helm/
│     └─ main.go          # root CLI, subcommands
├─ internal/
│  ├─ config/             # .cli-settings.json handling
│  │  └─ config.go
│  ├─ fs/                 # filesystem utilities (paths, discovery)
│  │  └─ paths.go
│  ├─ metadata/           # metadata.json structs + IO
│  │  └─ metadata.go
│  ├─ specs/              # operations on spec folders (create, split, etc.)
│  │  └─ specs.go
│  ├─ runner/             # wrapper around implement-spec.mjs and codex exec
│  │  └─ runner.go
│  ├─ tui/
│  │  ├─ common/          # shared TUI widgets/components
│  │  │  ├─ status_badge.go
│  │  │  ├─ layout.go
│  │  │  └─ styles.go
│  │  ├─ scaffold/        # `scaffold` TUI model, view, update
│  │  │  └─ model.go
│  │  ├─ run/             # `run` spec selection + streaming logs
│  │  │  └─ model.go
│  │  ├─ status/          # `status` dependency graph + table view
│  │  │  └─ model.go
│  │  └─ specsplit/       # `spec` splitting TUI (paste input, summary)
│  │     └─ model.go
└─ package.json / etc.    # only if needed for implement-spec.mjs
```

Libraries:

* CLI flags: either standard `flag` or `spf13/cobra`. Spec below assumes **Cobra** for clean subcommands.
* TUI: `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles`, `github.com/charmbracelet/lipgloss`.

---

## 2. Core Domain Types

### 2.1 Metadata

File: `docs/specs/spec-XX-*/metadata.json`

Go struct:

```go
type SpecStatus string

const (
    StatusTodo       SpecStatus = "todo"
    StatusInProgress SpecStatus = "in-progress"
    StatusDone       SpecStatus = "done"
    StatusBlocked    SpecStatus = "blocked"
)

type SpecMetadata struct {
    ID                 string      `json:"id"`
    Name               string      `json:"name"`
    Status             SpecStatus  `json:"status"`
    DependsOn          []string    `json:"dependsOn"`
    LastRun            *time.Time  `json:"lastRun,omitempty"`
    Notes              string      `json:"notes,omitempty"`
    AcceptanceCommands []string    `json:"acceptanceCommands"`
}
```

Helpers in `internal/metadata`:

* `LoadMetadata(path string) (*SpecMetadata, error)`
* `SaveMetadata(path string, md *SpecMetadata) error`
* `UpdateStatus(path string, status SpecStatus, notesAppend string, lastRun *time.Time) error`

### 2.2 Config (.cli-settings.json)

File: `docs/specs/.cli-settings.json`

Go struct:

```go
type Mode string

const (
    ModeParallel Mode = "parallel"
    ModeStrict   Mode = "strict"
)

type Settings struct {
    SpecsRoot          string   `json:"specsRoot"`          // default "docs/specs"
    Mode               Mode     `json:"mode"`               // "parallel" or "strict"
    DefaultMaxAttempts int      `json:"defaultMaxAttempts"` // default 2
    CodexModelScaffold string   `json:"codexModelScaffold"`
    CodexModelRunImpl  string   `json:"codexModelRunImpl"`
    CodexModelRunVer   string   `json:"codexModelRunVer"`
    CodexModelSplit    string   `json:"codexModelSplit"`
    AcceptanceCommands []string `json:"acceptanceCommands"` // defaults collected at scaffold
}
```

Helpers in `internal/config`:

* `LoadSettings(root string) (*Settings, error)`:

  * Determine `root` (CLI flag `--specs-root`, env, then default `"docs/specs"`).
  * If settings file missing, return default settings (with `SpecsRoot=root`).
* `SaveSettings(root string, settings *Settings) error`.

### 2.3 Spec Discovery

A “spec folder” is:

* Directory name `spec-*` under the specs root.
* Contains `SPEC.md` and `metadata.json`.

Go struct:

```go
type SpecFolder struct {
    ID          string         // from metadata.id or directory basename
    Name        string         // from metadata.name or first # heading
    Path        string         // absolute or relative path
    Metadata    *SpecMetadata
    Checklist   string         // path to acceptance-checklist.md
    CanRun      bool           // derived from deps
    UnmetDeps   []string       // IDs of deps not done
    LastVerifierStatus string  // last STATUS from implementation-report.md (optional)
    LastVerifierAt      *time.Time
}
```

Helpers in `internal/specs`:

* `DiscoverSpecs(root string) ([]*SpecFolder, error)`

  * Walk `root`, match `/spec-*` directories.
  * Try to read `metadata.json`, `acceptance-checklist.md`, `implementation-report.md`.
* `ComputeDependencyState(specs []*SpecFolder)`

  * For each spec, fill `CanRun` and `UnmetDeps` based on metadata.status of dependencies.

---

## 3. TUI Design (Bubble Tea)

### 3.1 Common Style and Components

In `internal/tui/common`:

* `StatusBadge(status SpecStatus) string`

  * Returns a styled Lipgloss text: `[TODO]`, `[IN PROGRESS]`, `[DONE]`, `[BLOCKED]`.
* Layout helpers:

  * `RenderTwoColumns(left, right string, width int) string`
* Key help:

  * Standard bottom bar showing keys for each view.

---

## 4. Command: `scaffold`

### 4.1 CLI Wrapper (Cobra)

`cmd/helm/main.go`:

* `scaffoldCmd := &cobra.Command{Use: "scaffold", Short: "Initialize specs structure", RunE: runScaffoldTUI}`

### 4.2 Scaffold TUI Flow

Bubble Tea model: `internal/tui/scaffold/model.go`

Model fields:

```go
type scaffoldStep int

const (
    stepIntro scaffoldStep = iota
    stepMode
    stepAcceptanceCommands
    stepOptionalPaths
    stepConfirm
    stepRunning
    stepDone
)

type Model struct {
    step scaffoldStep

    SpecsRoot  string   // default "docs/specs"
    Mode       Mode     // parallel/strict
    Commands   []string // acceptance commands
    GenGraph   bool     // whether to generate sample dependency graph (optional)
    Err        error
    Done       bool

    // Bubbles: text input for commands, list for yes/no, etc.
}
```

Steps:

1. **Intro screen**:

   * Explain that this will create `docs/specs` structure and templates.
   * Key: `enter` to continue, `ctrl+c` to quit.

2. **Mode selection**:

   * Prompt: “Run tasks in parallel?” Yes/No.
   * Map Yes = `ModeParallel`, No = `ModeStrict`.
   * Show short description for each mode.

3. **Acceptance commands input**:

   * Multi-line input or iterative adding:

     * Show a text input for a command like `pnpm typecheck` and allow hitting `enter` to add; blank input and pressing `enter` moves to next step.
     * Show list of added commands below.

4. **Optional paths & graph**:

   * Specs root path override input (default `docs/specs`).
   * Checkbox: “Generate sample dependency graph?” (for example, create `spec-01-a` and `spec-02-b` with dependsOn relationship in example).

5. **Confirm screen**:

   * Summary:

     * Mode: Parallel/Strict
     * Specs root: …
     * Commands: list
   * Keys: `enter` = “Apply & write files”, `esc` = go back.

6. **Running**:

   * Non-interactive progress messages:

     * “Creating directories…”
     * “Writing templates…”
     * “Writing example spec…”
   * Any error sets `m.Err` and shows an error message with `q` to quit.

7. **Done**:

   * Final screen summarizing created files, telling user next commands (`helm spec`, `helm run`).

### 4.3 Files to Create

All **paths are under `SpecsRoot`** (e.g., `docs/specs`):

1. `README.md`

   * Content: generic workflow description:

     * Overview of specs, `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, `implementation-report.md`.
     * Explanation of `implement-spec.mjs` runner.
     * No project-specific details.

2. `implement.prompt-template.md`

   * Template containing macros (brace syntax recommended):

     * `{{SPEC_ID}}`
     * `{{SPEC_NAME}}`
     * `{{SPEC_BODY}}` (contents of SPEC.md)
     * `{{ACCEPTANCE_COMMANDS}}` (bulleted list or plain)
     * `{{MODE}}` (`parallel` or `strict`)
     * `{{PREVIOUS_REMAINING_TASKS}}` (JSON object, may be empty)

   * Two variants internally; the file should embed both variants as text conditioned by mode (the Go CLI chooses mode at template fill time OR embed one template but with conditional text macros). For simplicity in v1:

     * Generate **one template fully specialized** to the chosen mode at scaffold time:

       * If `ModeParallel`:

         * Include bullets:

           * Warn about concurrent agents.
           * Forbid reverting unrelated changes.
           * Avoid repo-wide fixups; only fix what you touch.
           * It is acceptable to leave failing checks unrelated to changed files (report them).
       * If `ModeStrict`:

         * Require all acceptance commands to pass.
         * Require clean working tree at the end.
         * Demand repo-wide formatting/linting/typecheck.

   * Implementation deliverables section:

     * Summary
     * Traceability matrix (SPEC items → code changes)
     * Runbook (how to re-run checks etc.)
     * Manual smoke test description
     * Open questions/risks

3. `review.prompt-template.md`

   * Requirements:

     * **First line** of verifier output MUST be: `STATUS: ok` or `STATUS: missing`.
     * **Second line** MUST be JSON string `{"remainingTasks":[...]}` when status is `missing`.
     * Verifier is read-only.
     * Must verify:

       * All acceptance commands relevant to this spec have been run and passed (or flagged).
       * Implementation deliverables present.
       * E2E considerations and architecture rules are respected.
     * Mode-specific expectations:

       * Parallel: focus on spec coverage, not requiring global green.
       * Strict: require all acceptance commands to pass successfully; fail if not.

4. `implement-spec.mjs`

   * Detailed spec in section **7** below.

5. `spec-splitting-guide.md`

   * Human-readable instructions (non-project-specific) describing how to split:

     * Group by acceptance criteria.
     * Keep each spec at most N ACs (default ~5).
     * Respect explicit dependencies mentioned in large spec.
     * Each spec should be self-contained and independent given its deps.
   * Mention that this is used by the `spec` command/LLM.

6. Example spec folder: `spec-00-example/`:

   * `SPEC.md`

     * Example spec with ACs and simple implementation tasks.

   * `acceptance-checklist.md`

     * Pre-filled with acceptance commands collected from user.

   * `metadata.json`

     * Initial object:

       ```json
       {
         "id": "spec-00-example",
         "name": "Example Feature",
         "status": "todo",
         "dependsOn": [],
         "notes": "",
         "acceptanceCommands": ["<from settings>"]
       }
       ```

     * `lastRun` omitted initially.

   * `implementation-report.md`

     * Single line placeholder: “Implementation report will be generated by implement-spec.mjs.”

7. **.cli-settings.json**

   * Write with values from the scaffold flow.

---

## 5. Command: `run` (Spec Runner TUI)

### 5.1 CLI Wrapper

`runCmd := &cobra.Command{Use: "run", Short: "Run implementation/verifier loop for a spec", RunE: runRunTUI}`

Flags (optional for v1):

* `--spec-id <id>` run directly without TUI selection (if found).
* `--max-attempts <n>` override settings.

### 5.2 TUI Model

`internal/tui/run/model.go`

Model state:

```go
type phase int

const (
    phaseList phase = iota
    phaseRunning
    phaseResult
)

type Model struct {
    phase       phase
    Specs       []*SpecFolder
    List        list.Model         // from bubbles/list
    Selected    *SpecFolder
    LogLines    []string           // for streaming implement-spec.mjs stdout/stderr
    Done        bool
    Err         error
    MaxAttempts int
}
```

Flow:

1. **On init**:

   * Load settings (`SpecsRoot`, default max attempts).
   * Discover specs.
   * Compute dependency states (CanRun, UnmetDeps).
   * Build Bubble list of items:

     * Title: `spec.ID` + `spec.Name`
     * Description: status, last run, dependencies summary.

2. **List view (`phaseList`)**:

   * Each list item shows:

     * Status badge `[DONE]`, `[TODO]`, etc.
     * Spec ID and Name.
     * Dependencies: e.g. `Deps: spec-01-foo(DONE), spec-02-bar(TODO)`.

   * Unmet deps highlighted (Lipgloss color/style).

   * Keys:

     * Up/Down to navigate.
     * `enter` to select.
     * `q` to quit.
     * `f` to cycle filters (`All`, `Runnable only` (CanRun), `Blocked`, etc.).

   * If user selects a spec with unmet deps:

     * Show a modal panel:

       * “This spec has unmet dependencies: spec-02-bar (TODO). Run anyway? [y/N]”
     * On `y` proceed, on `n` return to list.

3. **Running view (`phaseRunning`)**:

   * Once a spec is selected:

     * Set `phase = phaseRunning`.
     * Spawn `implement-spec.mjs` via `os/exec.Command`:

       * Command: `node implement-spec.mjs <spec-dir>` (configurable path).
       * Set env:

         * `MAX_ATTEMPTS` (either from settings or flag).
         * `CODEX_MODEL_IMPL` / `CODEX_MODEL_VER` (optional).
       * Combined output (stdout + stderr) read via pipe.
     * As lines are read, send Bubble Tea `Msg` (e.g. `logLineMsg(line string)`) and append to `LogLines`.
     * Display tail of `LogLines` in a scrolling window.
     * When process exits:

       * On non-zero exit, set `Err`, move to `phaseResult`.
       * Always reload `metadata.json` for this spec to get updated status & lastRun.

4. **Result view (`phaseResult`)**:

   * Show:

     * New status from `metadata.json`.
     * `lastRun` timestamp.
     * Summary of remaining tasks if status is `in-progress`:

       * Parse from `implementation-report.md` if possible.
   * Keys:

     * `r` to go back to spec list (refresh).
     * `q` to quit.

---

## 6. Command: `spec` (Splitting a Large Spec)

### 6.1 CLI Wrapper

`specCmd := &cobra.Command{Use: "spec", Short: "Split a large spec into multiple spec folders", RunE: runSpecSplitTUI}`

Optional flags:

* `--file <path>`: read large spec from file, skip paste step, still show summary TUI.

### 6.2 TUI Model

`internal/tui/specsplit/model.go`

Model:

```go
type splitPhase int

const (
    splitPhaseIntro splitPhase = iota
    splitPhaseInput
    splitPhaseReview
    splitPhaseRunning
    splitPhaseDone
)

type Model struct {
    phase      splitPhase
    RawSpec    string             // full pasted spec text (or file content)
    Preview    string             // first N lines for display
    NumSpecs   int                // predicted/returned by LLM
    Results    []GeneratedSpec    // from codex call
    Err        error
    Done       bool
}

type GeneratedSpec struct {
    ID        string   // directory name e.g. "spec-01-auth"
    Name      string   // spec name from LLM
    DependsOn []string // spec IDs
}
```

Flow:

1. **Intro**:

   * Explain what this command does (LLM-based splitting).
   * Keys: `enter` to continue.

2. **Input**:

   * If `--file` not provided:

     * Present a big text area for user to paste spec (e.g., using a `textarea`-style bubble).
     * `ctrl+d` or `esc`+confirmation to finish input.
   * If `--file` provided:

     * Load file content into `RawSpec` and show a preview.

3. **Review**:

   * Show preview (first ~40 lines).
   * Ask to confirm: `[enter] Split into specs` or `[esc] Cancel`.

4. **Running**:

   * Build prompt for codex:

     * Include:

       * `spec-splitting-guide.md` contents.
       * The raw spec text.
       * `settings.AcceptanceCommands`.
     * Request JSON output of the form:

       ```json
       {
         "specs": [
           {
             "idSuffix": "auth",
             "index": 1,
             "name": "Authentication Flow",
             "dependsOn": [],
             "acceptanceCriteria": ["...", "..."]
           }
         ]
       }
       ```

   * Execute:

     ```go
     cmd := exec.Command("codex", "exec", "--sandbox", "read-only", "--model", settings.CodexModelSplit)
     ```

     * Send prompt via stdin, read stdout.

   * Parse JSON result into `[]GeneratedSpec`, generating full IDs as `spec-%02d-%s`.

   * For each generated spec:

     * Create directory `${SpecsRoot}/${ID}`.
     * Write:

       * `SPEC.md`: body extracted for this spec (either full text from codex or by splitting original spec).

         * Simpler v1: have codex output the full `SPEC.md` content for each spec.
       * `acceptance-checklist.md`: include `settings.AcceptanceCommands` and spec-specific ACs.
       * `metadata.json`:

         * `id`: spec ID.
         * `name`: spec name.
         * `status`: `"todo"`.
         * `dependsOn`: from `dependsOn`.
         * `acceptanceCommands`: `settings.AcceptanceCommands`.

   * Add cross-links:

     * In each `SPEC.md`, at bottom append section:

       ```md
       ## Depends on

       - spec-01-core: Core Platform Setup
       ```

5. **Done**:

   * Show a table of created spec IDs, names, dependencies.
   * Offer `q` to quit.

---

## 7. Command: `status` (Dependency Graph & Readiness TUI)

### 7.1 CLI Wrapper

`statusCmd := &cobra.Command{Use: "status", Short: "Show status and dependency graph for specs", RunE: runStatusTUI}`

### 7.2 TUI Model

`internal/tui/status/model.go`

Model:

```go
type focusMode int

const (
    focusAll focusMode = iota
    focusSubtree
    focusRunnable
)

type Model struct {
    Specs     []*SpecFolder
    Focus     focusMode
    Selected  *SpecFolder // for subtree focus
    Table     table.Model // bubbles/table if used
    Err       error
}
```

Display:

* **Top area**: summary counts:

  * `TODO: X | IN PROGRESS: Y | DONE: Z | BLOCKED: W`

* **Middle area**: dependency graph for current focus:

  * Render each spec line with indentation according to dependency depth.
  * Example:

    ```text
    spec-00-example [DONE]
    ├─ spec-01-core [DONE]
    │  └─ spec-02-ui [IN PROGRESS]
    └─ spec-03-docs [TODO]
    ```

* **Bottom area**: table view:

  * Columns: `ID`, `Name`, `Status`, `Deps`, `Last Run`.

* Keys:

  * `tab` switch between **graph** and **table** focus.
  * `f` cycle focus modes: `All` → `Runnable only` (CanRun) → `Subtree of selected`.
  * `enter` on a spec row sets `Selected` (used for subtree).
  * `q` quit.

Dependency graph construction:

* Build adjacency from `DependsOn` relationships.
* For each spec, compute depth via DFS starting from roots (specs with no one depending on them, or `dependsOn` empty). For cycle detection, show a special marker if cycle appears.

Runnable highlight:

* A spec is runnable if:

  * `Status` is not `done`.
  * All `dependsOn` are `done`.
* In runnable view, show only `CanRun` specs.

---

## 8. Implement-Spec Runner Script (`implement-spec.mjs`)

This is Node-based, but Go CLI treats it as a black box runner. Still, we define how it must behave.

Location: `${SpecsRoot}/implement-spec.mjs`

### 8.1 Inputs

* Called as:

  ```sh
  node implement-spec.mjs <spec-dir>
  ```

  where `<spec-dir>` is:

  * Absolute path OR
  * Relative path resolved as:

    * If path exists as given: use it.
    * Else, prefix with `docs/specs/`.

* Environment variables:

  * `MAX_ATTEMPTS` (string int, default `2`).
  * `CODEX_MODEL_IMPL` and `CODEX_MODEL_VER` (optional; fallback to sensible defaults).

### 8.2 Behavior

1. **Resolve spec path**.

2. **Load spec artifacts**:

   * `SPEC.md` → string content.
   * `implement.prompt-template.md` and `review.prompt-template.md` from `${SpecsRoot}`.
   * `metadata.json` for this spec:

     * Update `status` to `"in-progress"` before starting loop.

3. **Derive spec ID/name**:

   * Use `metadata.id` and `metadata.name` if present.
   * If `name` empty:

     * Parse first `#` heading from `SPEC.md` as spec name.

4. **Main loop** (attempts):

   ```js
   let remainingTasks = [];
   for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt++) {
       // Build worker prompt
       // Run codex worker
       // Build verifier prompt with worker output
       // Run codex verifier
       // Parse STATUS line
       // If ok => status done, break
       // else => update remainingTasks and continue
   }
   ```

5. **Worker phase**:

   * Fill `implement.prompt-template.md` with context:

     * `{{SPEC_ID}}`, `{{SPEC_NAME}}`, `{{SPEC_BODY}}`, `{{ACCEPTANCE_COMMANDS}}`, `{{PREVIOUS_REMAINING_TASKS}}`, `{{MODE}}`.

   * Call:

     ```sh
     codex exec --dangerously-bypass-approvals-and-sandbox \
       --model "$CODEX_MODEL_IMPL" --stdin
     ```

   * Worker has full file access.

   * Captured full stdout as `workerOutput`.

   * Stream worker logs directly to stdout as they arrive.

6. **Verifier phase**:

   * Build verifier prompt:

     * Include:

       * `SPEC.md`
       * Implementation summary (or full `workerOutput`).
       * `acceptance-checklist.md`.
       * Explanation of mode expectations (parallel/strict).

   * Call:

     ```sh
     codex exec --sandbox read-only --model "$CODEX_MODEL_VER" --stdin
     ```

   * Stream verifier output.

   * Capture first line and second line:

     * `STATUS: ok` or `STATUS: missing`
     * JSON line with `remainingTasks` if missing.

7. **Status handling**:

   * On `STATUS: ok`:

     * Set `metadata.status = "done"`.
     * `metadata.lastRun = new Date().toISOString()`.
     * Append high-level summary to `metadata.notes` (final worker summary).
     * Break loop and exit with status 0.

   * On `STATUS: missing`:

     * Set `metadata.status = "in-progress"`.
     * Append remaining tasks summary string to `metadata.notes`.
     * If attempts exhausted:

       * Exit with non-zero status.

8. **Blocked state**:

   * `implement-spec.mjs` itself doesn’t set `blocked`; CLI uses dependency graph to mark blocked based on unmet dependencies. (Optional: script could set `blocked` if verifier finds fundamental dependency issues, but v1 not required.)

9. **Implementation report**:

   * Create/overwrite `implementation-report.md` in spec folder.
   * Include:

     * Spec ID/Name.
     * Mode.
     * Number of attempts.
     * Final `STATUS`.
     * Remaining tasks (if any).
     * Full final worker output (or truncated with pointer to logs).

10. **Metadata persistence**:

    * Always write back metadata after each verifier run.

---

## 9. Metadata and Status Lifecycle

### 9.1 Status Transitions

* On `scaffold` example creation:

  * `spec-00-example` starts as `todo`.
* On first `run`/`implement-spec.mjs` invocation:

  * Immediately set `in-progress`.
* On `STATUS: ok` verifier:

  * Set `done`.
* On `STATUS: missing`:

  * Remain `in-progress`.
* A spec is considered **blocked** for UI purposes if:

  * Its metadata.status is `todo` or `in-progress`, **and**
  * One or more `dependsOn` specs != `done`.
  * `status` field remains `todo`/`in-progress`; blocked is a **derived view**, not persisted.
  * (Optional extension: you may allow manual setting of `blocked` in `metadata.json`.)

### 9.2 Acceptance Commands

* Captured in `settings.AcceptanceCommands`.

* For each new spec folder:

  * `metadata.acceptanceCommands` pre-filled with settings list.
  * `acceptance-checklist.md` includes section:

    ```md
    ## Required Commands

    - pnpm typecheck
    - pnpm test
    ```

* Implementation/verifier templates explicitly instruct:

  * Worker: run these commands where relevant, and record results in report.
  * Verifier: confirm that these commands were considered.

---

## 10. Step-by-Step Implementation Order

If you want to actually build this, here’s a sensible order:

1. **Bootstrap CLI skeleton**

   * Initialize Go module.
   * Add Cobra root and subcommands: `scaffold`, `run`, `spec`, `status`.
   * Wire each subcommand to a stub `RunE`.

2. **Implement config + FS helpers**

   * `internal/config` with `LoadSettings`/`SaveSettings`.
   * `internal/fs` for `ResolveSpecsRoot()` and `EnsureDir(path string)`.

3. **Implement metadata package**

   * `SpecMetadata` struct, `SpecStatus` enum, load/save with JSON and pretty formatting.

4. **Implement spec discovery**

   * `DiscoverSpecs` and `ComputeDependencyState`.
   * Test on a manually created spec folder.

5. **Build TUI `scaffold` flow**

   * Create Bubble Tea model for `scaffold`.
   * Implement form steps.
   * On confirmation, create FS structure and template files.
   * Generate `.cli-settings.json` and example spec.

6. **Implement `implement-spec.mjs` template**

   * At least create a stub script written by scaffold:

     * For now, print out what it *would* do without calling codex.
   * Later, flesh out codex calls as per the spec.

7. **Build TUI `run` flow**

   * Use discovered specs.
   * Show Bubble Tea list.
   * On selection, spawn `implement-spec.mjs` and stream logs.
   * After exit, reload metadata and show status.

8. **Build TUI `status` flow**

   * Reuse spec discovery and dependency computation.
   * Implement text-based dependency graph + table.
   * Implement focus/filter keybinds.

9. **Build TUI `spec` (splitting) flow**

   * Simple text input / preview.
   * For initial implementation:

     * Stub codex call with local splitting (e.g., naive stub that creates one spec).
   * Then replace stub with actual `codex exec` integration using `spec-splitting-guide.md`.

10. **Polish and harden**

* Persistent error handling and friendly messages.
* Config overrides via env vars and flags.
* Optionally add non-interactive modes for CI.