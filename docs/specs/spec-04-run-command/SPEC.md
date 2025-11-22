# spec-04-run-command — `run` TUI and spec execution

## Summary

Implement the `helm run` command as a Bubble Tea TUI that discovers specs, displays their status and dependencies, and lets the user run `implement-spec.mjs` for a selected spec. This command is the primary interface for executing specs via the worker/verifier loop.

## Goals

- Discover available spec folders from `docs/specs`.
- Render a TUI list of specs with status badges, dependency info, and last verifier result.
- Allow filtering by runnable specs (those whose dependencies are satisfied).
- Integrate with `implement-spec.mjs` by:
  - Running it as a subprocess.
  - Streaming its logs into the TUI.
  - Refreshing metadata on completion.

## Non-Goals

- No editing of specs or metadata from the TUI.
- No spec splitting or `status`-style dependency graph view; that is handled in other specs.

## Detailed Requirements

1. **Spec Discovery Integration**
   - Use the `internal/specs` package from `spec-01-config-metadata` to:
     - Load all `SpecFolder` instances.
     - Compute dependency state (`CanRun`, `UnmetDeps`).
   - Call this from the `run` command on startup.

2. **TUI Model for `run`**
   - Implement a Bubble Tea model in `internal/tui/run` with at least three phases:
     1. `List` phase:
        - Shows a list of specs (e.g., using `bubbles/list`).
        - Each item shows:
          - Spec ID and name.
          - Status badge: TODO / IN PROGRESS / DONE / BLOCKED (derived from `CanRun` and dependencies).
          - Quick summary of unmet dependencies if any.
        - Key bindings:
          - Up/Down: move selection.
          - `f`: toggle filters (`All` vs `Runnable only`).
          - Enter: select a spec to run.
          - `q`: quit.
        - If the selected spec has unmet dependencies:
          - Show a confirmation dialog: "This spec has unmet dependencies: … Run anyway? [y/N]".
     2. `Running` phase:
        - After user confirms, spawn `implement-spec.mjs <spec-dir>` as a subprocess.
        - Stream stdout/stderr lines into a scrollable view (e.g., `bubbles/viewport`).
        - Show an indicator of attempt progress if it can be inferred from logs.
        - Key bindings:
          - `q` should ask the user to confirm before killing the process.
     3. `Result` phase:
        - After the subprocess exits:
          - Reload metadata for that spec.
          - Show final status: DONE vs IN PROGRESS.
          - If status is IN PROGRESS, show the remaining tasks summary (if present in `implementation-report.md`).
        - Key bindings:
          - `enter` or `r`: return to the spec list (with refreshed data).
          - `q`: quit.

3. **Subprocess Management**
   - Use Go’s `os/exec` package to run `node docs/specs/implement-spec.mjs <spec-dir>`.
   - Propagate environment variables:
     - `MAX_ATTEMPTS` from settings or the process environment.
     - `CODEX_MODEL_IMPL` / `CODEX_MODEL_VER` if set in settings.
   - Capture stdout and stderr:
     - Read line-by-line and send messages into the Bubble Tea model.
     - Display them in chronological order.

4. **Status Updates**
   - After each run:
     - Reload `metadata.json` for the selected spec.
     - Reflect changes in the list view:
       - Specs with `status = "done"` are shown as DONE.
       - Specs with `status = "in-progress"` are shown with the IN PROGRESS badge.
     - A spec is visually BLOCKED if:
       - Its `status` is `"todo"` or `"in-progress"`, and
       - It has one or more `UnmetDeps`.

5. **Error Handling**
   - If `implement-spec.mjs` exits with a non-zero code:
     - Show an error message in the Result phase.
     - Keep the logs visible for debugging.
   - If the runner cannot be found or cannot be executed:
     - Surface a clear error to the user.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Running `go run ./cmd/helm run` with several specs in `docs/specs`:
  - Shows a TUI list of specs with IDs, names, and status badges.
  - Allows filtering to show only runnable specs.
  - Allows selecting a spec whose dependencies are done and starts the runner.
- When a spec run completes successfully (verifier `STATUS: ok`):
  - The spec’s status is updated to DONE in the list view after returning from the Result phase.
- When a spec run ends with `STATUS: missing`:
  - The spec’s status is shown as IN PROGRESS.
  - The Result phase shows the remaining tasks summary from `implementation-report.md`.

## Implementation Notes

- Keep the TUI responsive: stream logs incrementally rather than waiting for the process to finish.
- Consider limiting the number of log lines kept in memory (e.g., keep the last N lines).
- Use consistent styling (Lipgloss) for status badges and headings.
- Testing convention: exercise the run command against specs rooted in a temp directory (e.g., `t.TempDir()/specs-test`) and ensure settings/spec paths point there so repo-tracked `docs/specs/` files are not mutated.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Settings, metadata, and spec discovery
- spec-02-scaffold-command — scaffold command and initial specs layout
- spec-03-implement-runner — implement-spec runner with worker/verifier loop
