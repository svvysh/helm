# spec-05-spec-splitting-command — Breakdown/`spec` pane

## Summary

Implement the Breakdown pane of the TUI (accessible from the home navigation) and the `helm spec` entrypoint that runs the same flow directly without opening the shell. The pane accepts a large spec (or file), asks Codex for a split plan, and generates `spec-*` folders under the configured specs root.

## Goals

- Provide an interactive Bubble Tea flow for pasting a large spec or pointing to a file.
- Use the `spec-splitting-guide.md` plus repo config acceptance commands to ask Codex for a JSON split plan.
- Generate spec folders under `RepoConfig.SpecsRoot` with metadata, acceptance checklists, and placeholder reports.
- Integrate with the TUI shell so navigation returns to Run/Status panes when done.

## Non-Goals

- Editing existing specs or deleting generated specs.
- Running specs; execution remains in the Run pane.

## Detailed Requirements

1. **Entry & Navigation**
- `helm spec` runs the Breakdown flow directly (without the multi-pane shell). From the home navigation (opened via bare `helm`), selecting Breakdown mounts this pane; `q` returns to home.

2. **Input Flow**
   - Steps inside the pane:
     1. Intro text explaining what Breakdown does.
     2. Input step allowing either:
        - Multi-line paste of a spec, or
        - Providing a file path (via flag or prompt) read at startup.
     3. Preview of the first ~50 lines with a confirmation prompt.
     4. Progress view while the Codex request runs.
     5. Completion view summarizing created specs (ID, name, deps) and offering a button to jump to the Run pane.
   - Keyboard: Up/Down or tab between controls, enter to confirm, esc/ctrl+c to cancel back to home.

3. **Codex Split Plan**
   - Build the prompt using:
     - `spec-splitting-guide.md` from the specs root.
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
   - Show a summary table of created specs (ID, name, deps) and allow pressing a key/button to jump directly to the Run pane (keeping the shell running) or return home.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Running `go run ./cmd/helm split` in an initialized temp repo opens the Breakdown pane.
- Past­ing a sample spec or pointing to a file triggers a Codex request and generates spec folders under the configured `specsRoot` when the plan is valid.
- Generated folders contain `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, and `implementation-report.md` with correct IDs, names, and dependencies.
- Existing spec folders are not overwritten without confirmation; collisions are reported in the completion view.
- Returning from the completion view lands on home (or Run if the “jump to Run” action was chosen).

## Implementation Notes

- Codex calls for splitting should use a read-only sandbox (`--sandbox read-only`).
- Provide a dev flag to load a split plan from a local JSON file for tests.
- Generate into a temp specs root during automated tests so the tracked `docs/specs/` tree is untouched.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Repo config, metadata, and spec discovery
- spec-02-scaffold-command — Scaffold flow inside the TUI
- spec-04-run-command — TUI shell navigation
