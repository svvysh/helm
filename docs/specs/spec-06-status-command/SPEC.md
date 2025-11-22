# spec-06-status-command — Status pane and dependency graph

## Summary

Implement the Status pane within the TUI shell (spec-04) and the `helm status` entrypoint that runs a direct view without opening the shell. The pane provides a read-only overview of all specs, their statuses, and dependencies when inside the shell.

## Goals

- Show summary counts and a simple dependency graph.
- Provide a tabular view with filtering and subtree focus.
- Integrate with the TUI navigation so users can jump between Run, Breakdown, and Status without leaving the app.

## Non-Goals

- Executing or modifying specs (handled by Run pane).
- Editing metadata.

## Detailed Requirements

1. **Entry & Navigation**
- Selecting Status from the home navigation mounts this pane; `helm status` runs a standalone status view (non-shell).
   - `tab` (or another key) toggles between graph and table view; `q` returns to the home menu.

2. **Summary Counts**
   - Display counts for TODO, IN PROGRESS, DONE, FAILED, BLOCKED (BLOCKED = unmet deps while not done; FAILED = exhausted max attempts without STATUS: ok) at the top of the pane.

3. **Dependency Graph View**
   - Render an ASCII tree of specs and their dependencies (same formatting as the previous spec version).
   - Root nodes are specs not listed as dependencies of others.

4. **Table View**
   - Columns: ID, Name, Status, Deps (comma-separated IDs), Last Run (or `-`).
   - Allow scrolling through rows.

5. **Focus Modes**
   - Support at least three modes:
     - All specs.
     - Runnable specs only (no unmet deps and not done).
     - Subtree of a selected spec (spec and everything that depends on it).
   - Key bindings:
     - `f` cycles focus modes.
     - `enter` on a table row sets subtree mode to that spec.

6. **Data Source**
   - Use `internal/specs` discovery and dependency computation from spec-01. Avoid duplicating logic already used by the Run pane.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Running `go run ./cmd/helm status` in an initialized temp repo opens the TUI shell with the Status pane selected.
- Status pane shows summary counts, a dependency graph, and a table view; `tab` toggles views.
- `f` cycles focus modes (all / runnable / subtree); selecting a row and pressing `enter` sets subtree focus.
- BLOCKED specs (unmet deps) are visually indicated even if `metadata.status` is `todo` or `in-progress`.
- Pressing `q` from the pane returns to the home menu without exiting the entire app.

## Implementation Notes

- Reuse styling helpers from the Run pane for badges and headings.
- Keep rendering efficient for dozens of specs.
- Specs with status FAILED remain rerunnable (when deps are met); IN PROGRESS and DONE should not be runnable.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Repo config, metadata, and spec discovery
- spec-04-run-command — TUI shell and Run pane
