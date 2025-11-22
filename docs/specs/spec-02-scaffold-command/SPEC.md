# spec-02-scaffold-command — `scaffold` command and initial docs/specs layout

## Summary

Implement the `helm scaffold` command and associated Bubble Tea flow that creates the `docs/specs/` workspace, templates, and an example spec. This command is the entrypoint for bootstrapping the spec runner in any project.

In this repository, it should be capable of recreating a structure similar to the current `docs/specs/` contents (with templates, spec-splitting guide, and example spec).

## Goals

- Implement an interactive TUI for `helm scaffold` using Bubble Tea.
- Ask the user whether the workflow should be **parallel** or **strict**.
- Ask the user for default acceptance commands (e.g., `go test ./...`).
- Create the `docs/specs/` directory and the following files:
  - `README.md`
  - `.cli-settings.json`
  - `implement.prompt-template.md`
  - `review.prompt-template.md`
  - `implement-spec.mjs`
  - `spec-splitting-guide.md`
  - `spec-00-example/` example spec folder.
- Ensure the command is idempotent and safe to run on an existing project.

## Non-Goals

- No integration with Codex beyond writing the runner script file (`implement-spec.mjs`).
- No spec splitting or `run`/`status` TUI behavior.
- No opinionated project-specific content in templates (keep them generic).

## Detailed Requirements

1. **TUI Model for `scaffold`**
   - Implement a Bubble Tea model for the `scaffold` flow with the following steps:
     1. Intro screen explaining what will be created.
     2. Mode selection: “Run tasks in parallel?” (yes = parallel, no = strict).
     3. Acceptance commands input:
        - Allow the user to enter one command per line.
        - Empty input + enter finishes the list.
     4. Optional settings:
        - Specs root path (default: `docs/specs`).
        - (Optional) Whether to generate a sample dependency graph among example specs.
     5. Confirmation screen summarizing:
        - Selected mode.
        - Specs root.
        - Acceptance commands.
     6. Running/progress state.
     7. Completion screen.

   - Provide keyboard shortcuts:
     - Up/Down or tab to move between choices.
     - Enter to confirm steps.
     - Escape or ctrl+c to cancel.

2. **Filesystem Creation**
   - Given the collected answers, create `SpecsRoot` (e.g., `docs/specs`), ensuring:
     - Intermediate directories are created as needed.
     - Existing files are not blindly overwritten:
       - If a file already exists, either:
         - Leave it untouched and note it in the final summary, or
         - Offer a `--force` flag in the future (but not required in this spec).

3. **Settings File**
   - Write `.cli-settings.json` at the specs root with keys:
     - `specsRoot`
     - `mode`
     - `defaultMaxAttempts` (default 2)
     - `codexModelScaffold`, `codexModelRunImpl`, `codexModelRunVer`, `codexModelSplit` (use reasonable defaults or placeholders).
     - `acceptanceCommands` (from user input).
   - Use the `Settings` struct from `spec-01-config-metadata` and `SaveSettings` to serialize.

4. **Prompt Templates**
   - Write `implement.prompt-template.md` and `review.prompt-template.md` based on the product spec:

     - `implement.prompt-template.md`:
       - Use the strict or parallel variants depending on selected mode.
       - Include placeholders:
         - `{{SPEC_ID}}`, `{{SPEC_NAME}}`, `{{SPEC_BODY}}`,
         - `{{ACCEPTANCE_COMMANDS}}`,
         - `{{PREVIOUS_REMAINING_TASKS}}`,
         - `{{MODE}}`.
       - Enumerate required deliverables: summary, changelog, traceability, runbook, manual smoke, open issues/risks.

     - `review.prompt-template.md`:
       - Enforce the `STATUS: ok|missing` first line and JSON `remainingTasks` second line format.
       - Include placeholders:
         - `{{SPEC_ID}}`, `{{SPEC_NAME}}`, `{{SPEC_BODY}}`,
         - `{{ACCEPTANCE_CHECKLIST}}`,
         - `{{ACCEPTANCE_COMMANDS}}`,
         - `{{IMPLEMENTATION_REPORT}}`,
         - `{{MODE}}`.

5. **Runner Script**
   - Create `implement-spec.mjs` as a Node script that:
     - Accepts a spec directory path as its argument.
     - Resolves the specs root and reads:
       - `implement.prompt-template.md`,
       - `review.prompt-template.md`,
       - `metadata.json` for the selected spec.
     - Builds worker and verifier prompts with string substitution.
     - Calls the Codex CLI:
       - Worker: `codex exec --dangerously-bypass-approvals-and-sandbox ...`
       - Verifier: `codex exec --sandbox read-only ...`
     - Loops worker → verifier up to `MAX_ATTEMPTS` (from env or settings).
     - Streams both worker and verifier stdout to the console.
     - Writes `implementation-report.md`.
     - Updates `metadata.json.status` and `metadata.json.lastRun` based on verifier status:
       - `STATUS: ok` → `done`
       - `STATUS: missing` → `in-progress` with notes appended.

   - The actual behavior of the script will be fully exercised in later specs; in this spec, you just need to implement the file and ensure it compiles and runs in a simple dry-run scenario.

6. **Example Spec Folder**
   - Create `spec-00-example/` with:
     - `SPEC.md` — a simple example feature spec.
     - `acceptance-checklist.md` — references the default acceptance commands.
     - `metadata.json` — status set to `"todo"`, no dependencies.
     - `implementation-report.md` — placeholder text.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Running `go run ./cmd/helm scaffold`:
  - Presents an interactive TUI flow with the steps listed above.
  - After completing the flow, creates (or confirms the existence of) the `docs/specs` directory and required files.
- After scaffold:
  - `.cli-settings.json` exists and matches the selected mode and acceptance commands.
  - `implement.prompt-template.md` and `review.prompt-template.md` are present and include all required placeholders.
  - `implement-spec.mjs` exists and is executable by Node (at least prints some output or a help message when run against an example spec).
  - `spec-00-example` exists with the expected files.

## Implementation Notes

- Keep the `scaffold` TUI as a separate Bubble Tea model in `internal/tui/scaffold`.
- The Cobra `scaffold` command should just instantiate the model and run it.
- For now, you can keep the runner script simple; later specs may refine its behavior and error handling.
- Testing convention: run scaffold in tests against a temp specs root (e.g., `t.TempDir()/specs-test`) and assert outputs there so we never write into the repository’s `docs/specs/` directory.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Settings, metadata, and spec discovery
