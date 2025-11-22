# spec-01-config-metadata — Settings, metadata, and spec discovery

## Summary

Introduce the core domain types and filesystem conventions for the spec runner. This spec defines how `metadata.json` and global app settings are represented in Go, adds a settings TUI for editing them, and adds basic logic to discover `spec-*` folders under `docs/specs`.

## Goals

- Define the `SpecMetadata` struct matching the metadata schema.
- Define a global `Settings` struct (stored outside individual repos) with model + reasoning options.
- Implement load/save helpers for metadata and settings.
- Provide a settings TUI that edits these global settings with pickers (no free-form model input).
- Implement basic spec discovery that finds spec folders and attaches metadata.
- Lay the groundwork for future TUI commands (`run`, `status`, etc.) that will use this information.

## Non-Goals

- No Bubble Tea TUI yet.
- No actual CLI command behavior beyond wiring stubs to call the new helpers.
- No Codex integration or runner orchestration.

## Detailed Requirements

1. **Metadata Model**
   - Create an `internal/metadata` package with:
     - A `SpecStatus` type with values:
       - `"todo"`, `"in-progress"`, `"done"`, `"blocked"`.
     - A `SpecMetadata` struct:

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

   - Provide `LoadMetadata(path string) (*SpecMetadata, error)` and `SaveMetadata(path string, *SpecMetadata) error`.

2. **Settings Model (Global, with model + reasoning options)**
   - Create an `internal/config` package with:

     ```go
     type Mode string

     const (
         ModeParallel Mode = "parallel"
         ModeStrict   Mode = "strict"
     )

     type CodexChoice struct {
         Model     string `json:"model"`
         Reasoning string `json:"reasoning"`
     }

     type Settings struct {
         SpecsRoot          string      `json:"specsRoot"`
         Mode               Mode        `json:"mode"`
         DefaultMaxAttempts int         `json:"defaultMaxAttempts"`
         CodexScaffold      CodexChoice `json:"codexScaffold"`
         CodexRunImpl       CodexChoice `json:"codexRunImpl"`
         CodexRunVer        CodexChoice `json:"codexRunVer"`
         CodexSplit         CodexChoice `json:"codexSplit"`
         AcceptanceCommands []string    `json:"acceptanceCommands"`
     }
     ```

   - Allowed model values (fixed list, no free-form input):
     - `gpt-5.1`
     - `gpt-5.1-codex`
     - `gpt-5.1-codex-mini`
     - `git-5.1-codex-max`
   - Allowed reasoning values (per model):
     - `gpt-5.1`: `low`, `medium`, `high`
     - `gpt-5.1-codex`: `low`, `medium`, `high`
     - `gpt-5.1-codex-mini`: `medium`, `high`
     - `git-5.1-codex-max`: `low`, `medium`, `high`, `xhigh`
   - The stored `model` and `reasoning` values must be usable directly as `codex exec --model <model> --reasoning <value>` in later specs.
   - Implement helpers:
     - `LoadSettings() (*Settings, error)`
       - Reads from a **global** settings file, e.g. `$HOME/.helm/settings.json` (or platform-appropriate config dir).
       - If missing, returns defaults:
         - `SpecsRoot = "docs/specs"`,
         - `Mode = ModeStrict`,
         - `DefaultMaxAttempts = 2`,
         - `AcceptanceCommands` empty,
         - Default models: `CodexRunImpl` and `CodexRunVer` → `gpt-5.1-codex` with `reasoning="medium"`; `CodexScaffold` and `CodexSplit` → `gpt-5.1` with `reasoning="medium"`.
     - `SaveSettings(settings *Settings) error` writes to the same global location.
     - `Validate(settings *Settings) error` ensures model/reasoning pairs are from the allowed combinations above.
   - The `root` parameter is no longer required for settings since storage is global; repo-relative paths only affect spec discovery.

3. **Settings TUI (new)**
   - Add a Bubble Tea model in `internal/tui/settings` plus a Cobra command `helm settings` that launches it.
   - The TUI must let users edit:
     - `specsRoot` (text input, default `docs/specs`).
     - `mode` (toggle strict/parallel).
     - `defaultMaxAttempts` (numeric input with sane bounds).
     - `acceptanceCommands` (multi-line or per-line editor, reuse scaffold flow UX).
     - Codex slots (`Scaffold`, `RunImpl`, `RunVer`, `Split`): each uses **option pickers** (no free text) for model, and a dependent picker for reasoning effort constrained by the chosen model’s allowed values.
   - On save:
     - Validate combinations; show an inline error if invalid.
     - Persist via `SaveSettings` to the global settings file.
   - Keyboard expectations (minimum): up/down or tab to move fields, enter to pick/save, esc/ctrl+c to cancel.

4. **Spec Folder Representation**
   - Create an `internal/specs` package with:

     ```go
     type SpecFolder struct {
         ID        string
         Name      string
         Path      string
         Metadata  *metadata.SpecMetadata
         Checklist string   // path to acceptance-checklist.md
         CanRun    bool     // derived by dependency analysis
         UnmetDeps []string // IDs of deps not yet done
     }
     ```

   - Implement `DiscoverSpecs(root string) ([]*SpecFolder, error)` that:
     - Scans `root` (`docs/specs` by default).
     - Finds subdirectories whose names start with `spec-`.
     - Requires each spec folder to contain `SPEC.md` and `metadata.json`.
     - Reads `metadata.json` and populates `SpecFolder.Metadata`.
     - Points `Checklist` to `acceptance-checklist.md` if present (or leaves it empty).

5. **Dependency State Calculation**
   - Implement a helper, e.g., `ComputeDependencyState(specs []*SpecFolder)` that:
     - For each spec:
       - Sets `CanRun = true` if:
         - Its `Status` is not `"done"`, and
         - All `dependsOn` entries correspond to specs whose status is `"done"`.
       - Fills `UnmetDeps` with IDs that are not `"done"`.
   - The `blocked` status will be a *derived view*:
     - Do not mutate `metadata.status` to `"blocked"` automatically.
     - Later TUI code will decide when to show a spec as visually “blocked”.

6. **Root Command Wiring (Minimal)**
   - Update the root CLI command (from `spec-00-foundation`) so that:
     - It loads **global** settings on startup (failing fast with a helpful message if unreadable).
     - It prints a helpful error if `SpecsRoot` does not exist yet.
     - It registers `helm settings` to launch the new settings TUI.
   - Do not add other TUI behavior yet; just ensure the infrastructure compiles and can be imported by later specs.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Unit tests cover:
  - `DiscoverSpecs` and `ComputeDependencyState` as before.
  - `LoadSettings`/`SaveSettings` round-trip via a temp global settings path (no repo file required).
  - Validation rejects disallowed model/reasoning pairs.
- Manual or integration test notes: launching `helm settings` allows selecting models from the fixed list (no free text) and choosing reasoning options constrained by the model, then saves to the global settings file.

## Implementation Notes

- Keep the `internal` packages self-contained and free of CLI-specific concerns.
- Favor small helper functions that will be easy to unit test.
- The settings TUI should be minimal but must rely on the same `Settings` struct and validation helper; when persisted, the chosen reasoning value will later be passed to `codex exec` via `--reasoning` alongside `--model`.
- Testing convention: when exercising settings/spec discovery in tests, point `SpecsRoot` at a temp path (e.g., `t.TempDir()/specs-test`) to avoid mutating the tracked `docs/specs/` workspace.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
