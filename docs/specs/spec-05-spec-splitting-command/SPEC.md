# spec-05-spec-splitting-command — Breakdown/`spec` pane

## Summary

Implement the Breakdown pane of the TUI (accessible from the home navigation) and the `helm spec` entrypoint that runs the same flow directly without opening the shell. The pane accepts a large spec (paste or file), streams Codex progress, and generates `spec-*` folders under the configured specs root using the on-disk `specs-breakdown-guide.md`.

## Goals

- Provide an interactive Bubble Tea flow with a single screen that switches phases (input → preview → progress → done) and keeps keyboard hints visible.
- Use the `specs-breakdown-guide.md` plus repo config acceptance commands to ask Codex for a JSON split plan.
- Generate spec folders under `RepoConfig.SpecsRoot` with metadata, acceptance checklists, and placeholder reports.
- Integrate with the TUI shell so navigation returns to Run/Status panes when done.

## Non-Goals

- Editing existing specs or deleting generated specs.
- Running specs; execution remains in the Run pane.

## Detailed Requirements

1. **Entry & Navigation**
   - `helm spec` runs the Breakdown flow directly (without the multi-pane shell). From the home navigation (opened via bare `helm`), selecting Breakdown mounts this pane; `q` or `esc` returns to home.

2. **Input Flow**
   - Single-screen UI with phases:
     1) Input: multiline editor with placeholder; `Ctrl+O` loads a file path typed into the box; `Ctrl+L` clears; `Enter` starts splitting; `esc/q` cancels.
     2) Progress: spinner + **live Codex stdout/stderr only** (no spec-text preview); `esc/q` cancels request if still in flight.
     3) Done: table of generated specs (ID, name, deps), warnings, and recent logs (tail ~15 lines); actions `r` → jump to Run pane, `n` → new split, `enter/q/esc` → exit.
   - Keyboard hints persist: `enter` primary action, `esc/q` back/quit, `Ctrl+O` load file, `Ctrl+L` clear, `r` run, `n` new split.

3. **Codex Split Plan**
   - Build the prompt using:
     - `specs-breakdown-guide.md` from the specs root.
     - The raw pasted/file content.
     - Acceptance commands from `RepoConfig.AcceptanceCommands`.
   - Ask Codex to return JSON of the form:

     ```json
     { "specs": [ { "index": 0, "idSuffix": "foundation", "name": "Go module and CLI skeleton", "dependsOn": [], "acceptanceCriteria": ["..."] } ] }
     ```

4. **Spec Folder Generation**
   - For each plan entry, create `spec-%02d-%s` under `SpecsRoot` (respecting the repo-configured root, default `specs/`).
   - Write:
     - `SPEC.md` with summary and `## Depends on` section.
     - `acceptance-checklist.md` combining acceptance commands and criteria.
     - `metadata.json` with `status="todo"`, `dependsOn` from the plan, and acceptance commands from config.
     - `implementation-report.md` placeholder.
   - Do not overwrite existing spec folders without explicit confirmation; skip and report any collisions.

5. **Completion State**
   - Show a summary table of created specs (ID, name, deps) plus warnings for skipped/overwritten items. Offer `r` to jump to Run (keeping the shell running), `n` to start another split, or `esc/q` to return home.
   - Capture the first `session id:` line emitted by Codex stdout during the split request, show a persistent hint “Resume with: `codex resume <id>`”, and add a keybinding (e.g., `c`) to copy the command to the clipboard in both the running and done phases. Only the first match is used; subsequent matches are ignored to avoid false positives.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Running `go run ./cmd/helm spec` in an initialized temp repo opens the Breakdown pane.
- Past­ing a sample spec or pointing to a file triggers a Codex request and generates spec folders under the configured `specsRoot` when the plan is valid.
- Generated folders contain `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, and `implementation-report.md` with correct IDs, names, and dependencies.
- Existing spec folders are not overwritten without confirmation; collisions are reported in the completion view.
- The running view streams Codex stdout/stderr; errors are shown in the done view along with recent logs.
- Returning from the completion view lands on home (or Run if the “jump to Run” action was chosen).

## Implementation Notes

- Codex calls for splitting should use a read-only sandbox (`--sandbox read-only`).
- Provide a dev flag to load a split plan from a local JSON file for tests.
- Generate into a temp specs root during automated tests so the tracked `docs/specs/` tree is untouched.
- Clipboard copy should degrade gracefully: if the clipboard cannot be written, at least render the resume command inline so it is visible in the terminal scrollback.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Repo config, metadata, and spec discovery
- spec-02-scaffold-command — Scaffold flow inside the TUI
- spec-04-run-command — TUI shell navigation
