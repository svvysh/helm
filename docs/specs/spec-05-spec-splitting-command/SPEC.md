# spec-05-spec-splitting-command — `spec` command for splitting large specs

## Summary

Implement the `helm spec` command and TUI flow that accepts a large, pasted product spec (or a file path), uses the `spec-splitting-guide.md` to instruct Codex, and generates multiple `spec-XX-*` folders with corresponding `SPEC.md`, `acceptance-checklist.md`, and `metadata.json` files.

## Goals

- Provide an interactive `helm spec` TUI to accept a large spec input.
- Use Codex to propose a machine-readable plan for splitting the spec into multiple smaller specs.
- Generate spec folders under `docs/specs` according to the plan.
- Seed each spec folder with acceptance checklists and metadata (status `todo`, dependencies set).

## Non-Goals

- No support for editing existing specs; this command is for creating new ones.
- No “undo” or deletion of generated specs (for now).

## Detailed Requirements

1. **TUI Model for `spec`**
   - Implement a Bubble Tea model in `internal/tui/specsplit` with phases:
     1. Intro:
        - Explain what the command does.
     2. Input:
        - Allow user to either:
          - Paste a large spec into a multi-line text area, or
          - Provide a file path (e.g., via flag) that is read at startup.
     3. Preview:
        - Show the beginning (e.g., first 40–60 lines) of the spec.
        - Ask the user to confirm before splitting.
     4. Running:
        - Show a progress view while the Codex request is in flight.
     5. Done:
        - Show a summary table of created specs: ID, name, dependencies.

2. **Codex Split Plan**
   - Build a prompt using:
     - The contents of `spec-splitting-guide.md`.
     - The raw pasted spec.
     - The default acceptance commands from `.cli-settings.json`.
   - Ask Codex to respond with a JSON object conforming to:

     ```json
     {
       "specs": [
         {
           "index": 0,
           "idSuffix": "foundation",
           "name": "Go module and CLI skeleton",
           "dependsOn": [],
           "acceptanceCriteria": [
             "CLI binary exposes scaffold/run/spec/status subcommands",
             "go test ./... passes"
           ]
         }
       ]
     }
     ```

3. **Spec Folder Generation**
   - For each entry in the JSON plan:
     - Compute a spec ID: `spec-%02d-%s` where `%02d` is `index` and `%s` is `idSuffix` (normalized to a safe directory name).
     - Create a directory under `docs/specs` with that ID.
     - Create:
       - `SPEC.md`:
         - Include the spec name, a summary, and the provided acceptance criteria.
         - Reference any dependencies in a `## Depends on` section.
       - `acceptance-checklist.md`:
         - Include required acceptance commands from `.cli-settings.json`.
         - Expand spec-specific acceptance criteria into checkboxes.
       - `metadata.json`:
         - `id`: the folder ID.
         - `name`: the spec name.
         - `status`: `"todo"`.
         - `dependsOn`: the list of spec IDs/equivalents from the JSON plan.
         - `acceptanceCommands`: from `.cli-settings.json`.
       - `implementation-report.md`:
         - A simple placeholder text.

4. **Cross-Links**
   - In each generated `SPEC.md`, add a `## Depends on` section that lists dependencies by ID and (if available) name.

5. **Safety & Idempotency**
   - If a target spec folder already exists:
     - Prompt the user before overwriting, or
     - Use a different ID (e.g., increment the index) and warn the user.
   - Avoid silently overwriting existing human-authored specs.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Running `go run ./cmd/helm spec`:
  - Shows a TUI that allows pasting a large spec.
  - After confirmation, triggers a Codex request for a split plan.
- When Codex returns a valid JSON split plan:
  - New spec folders are created under `docs/specs`.
  - Each folder contains `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, and `implementation-report.md`.
  - Dependencies between specs are encoded in both `metadata.json.dependsOn` and the `## Depends on` section of `SPEC.md`.

## Implementation Notes

- Codex calls for splitting should use a read-only sandbox (`--sandbox read-only`).
- For testing without real Codex access, consider a dev flag to read a split plan from a local JSON file.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Settings, metadata, and spec discovery
- spec-02-scaffold-command — scaffold command and initial specs layout
