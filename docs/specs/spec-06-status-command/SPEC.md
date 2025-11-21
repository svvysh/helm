# spec-06-status-command — `status` TUI and dependency graph

## Summary

Implement the `helm status` command as a Bubble Tea TUI that summarizes the readiness of all specs, including a dependency graph and a table view. This command is read-only and designed to answer “What specs are ready to run, in progress, or blocked?”.

## Goals

- Provide a TUI overview of all specs, their statuses, and dependencies.
- Render a simple textual dependency graph.
- Show which specs are runnable (dependencies satisfied).
- Allow filtering by status or dependency subtree.

## Non-Goals

- No execution or modification of specs (that is handled by `run`).
- No inline editing of metadata.

## Detailed Requirements

1. **Summary Counts**
   - At the top of the TUI, show counts:
     - TODO
     - IN PROGRESS
     - DONE
     - BLOCKED (derived)
   - BLOCKED is defined as:
     - A spec whose `status` is `"todo"` or `"in-progress"` AND has at least one unmet dependency.

2. **Dependency Graph View**
   - Render a simple tree-like view of specs and their dependencies, for example:

     ```text
     spec-00-foundation [DONE]
     ├─ spec-01-config-metadata [DONE]
     │  ├─ spec-04-run-command [TODO]
     │  └─ spec-06-status-command [TODO]
     └─ spec-02-scaffold-command [IN PROGRESS]
     ```

   - Use indentation and ASCII connectors (`├─`, `└─`) to convey parent-child relationships.
   - Root nodes are specs that are not listed as dependencies of any other spec.

3. **Table View**
   - Provide a tabular view (e.g., using `bubbles/table`) with columns:
     - ID
     - Name
     - Status
     - Deps (comma-separated IDs)
     - Last Run (or `-` if none)
   - Allow the user to scroll through the table.

4. **Focus Modes**
   - Support at least three focus modes:
     - All specs.
     - Runnable specs only (those with no unmet dependencies and not done).
     - Subtree of a selected spec (that spec and all specs that depend on it).
   - Key bindings:
     - `tab`: toggle between graph view and table view.
     - `f`: cycle focus modes.
     - `enter` on a spec row in the table: select it as the root for subtree mode.
     - `q`: quit.

5. **Data Source**
   - Use the same `internal/specs` discovery and dependency computation as the `run` command.
   - Do not duplicate logic; factor shared code where appropriate (e.g., a helper to compute dependency graph roots and adjacency).

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- With several specs defined in `docs/specs`:
  - Running `go run ./cmd/helm status` presents a TUI with:
    - Summary counts by status.
    - A dependency graph view.
    - A table view.
  - Pressing `tab` toggles between graph and table.
  - Pressing `f` cycles through focus modes (all / runnable / subtree).
  - Selecting a row in the table and pressing `enter` changes subtree mode to focus on that spec.
- Specs with unmet dependencies are visually indicated as BLOCKED in the UI (even if their `metadata.status` is `"todo"` or `"in-progress"`).

## Implementation Notes

- Reuse styling components (status badges, headings) from the `run` TUI where possible.
- Keep rendering efficient enough to handle dozens of specs without flicker.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Settings, metadata, and spec discovery
- spec-04-run-command — run command TUI and spec execution
