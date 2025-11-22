# spec-04-run-command — TUI shell and Run pane

## Summary

Build the Bubble Tea TUI that is the primary interface for Helm. When the repo is initialized, the TUI home (opened only by bare `helm`) exposes three panes: **Run specs**, **Breakdown specs**, and **Status**. This spec covers the TUI shell plus the Run pane (spec execution). The Breakdown and Status panes are defined in spec-05 and spec-06 but must be reachable from the shell.

## Goals

- Launch the TUI only from `helm` (root).
- If `NeedsInitialization` is true, route to the scaffold gate from spec-02 and show nothing else.
- Provide a home navigation bar (e.g., tabs or buttons) for Run / Breakdown / Status panes.
- Implement the Run pane: discover specs, show statuses, allow filtering, and run a selected spec via the Go runner (spec-03) with no Node runner dependency.
- Add a shortcut to run the “next available” spec (first runnable in dependency order).

## Non-Goals

- Implementing the Breakdown or Status pane details (handled in their specs), beyond wiring navigation and mounting placeholders.
- Editing metadata by hand.
- Spec splitting or visualization logic beyond what is required to host those panes.

## Detailed Requirements

1. **TUI Shell & Navigation**
   - On startup, load `helm.config.json` via spec-01 helpers.
   - If `NeedsInitialization` returns true, immediately display the scaffold gate (spec-02). After scaffold completes successfully, reload config and show the home menu.
   - Home shows three options:
     - `Run specs` (implemented here)
     - `Breakdown specs` (mounts the spec-05 pane when available; otherwise show a placeholder)
     - `Status` (mounts the spec-06 pane when available; otherwise show a placeholder)
   - Keyboard: left/right (or numbers 1/2/3) switch panes; `q` quits.
   - Bare `helm` opens the shell (defaulting to the last used pane or Run). Subcommands do **not** open the shell.

2. **Spec Discovery Integration**
   - Use `internal/specs.DiscoverSpecs` with `RepoConfig.SpecsRoot` to populate the Run pane list on load and after each execution.
   - Compute dependency readiness so runnable specs are known.

3. **Run Pane UI**
   - List view of specs with badges and unmet dependency hints (carry over from the previous version of this spec):
     - ID + name.
     - Status badge: TODO / IN PROGRESS / DONE / BLOCKED (BLOCKED = unmet deps while not done).
     - Quick summary of unmet dependencies if any.
   - Controls:
     - Up/Down to move selection.
     - `f` toggles filter (All vs Runnable only).
     - `enter` on a spec opens a confirmation dialog if deps unmet; otherwise starts execution.
     - `n` runs the “next available” spec (first runnable according to the current ordering/filter), skipping the dialog.
     - `q` from the Run pane returns to home; `q` from home exits.

4. **Execution Flow**
   - When confirmed, invoke the Go runner from spec-03 (not the Node script) in a subprocess or goroutine so the TUI stays responsive.
   - Stream stdout/stderr into a scrollable viewport during execution.
   - Provide cancel handling: `q` during execution asks for confirmation before terminating the process.
   - After runner exit, reload metadata for that spec and show a Result view summarizing DONE vs IN PROGRESS and any remaining tasks parsed from the latest report.
   - Returning from Result refreshes the list with updated statuses.

5. **Error Handling**
   - If the runner exits non-zero, show the error and keep logs visible; allow the user to return to the list.
   - If the runner binary is missing or cannot start, surface a clear error and offer to return to the list/home.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Running `go run ./cmd/helm` in an initialized temp repo opens the TUI home with Run/Breakdown/Status options.
- When `NeedsInitialization` is true, only the scaffold gate is shown; after scaffold, the home menu appears.
- In the Run pane:
  - Specs display ID, name, status badge, and unmet deps summary.
  - `f` filters to runnable specs.
  - `n` starts the first runnable spec without extra prompts.
  - Selecting a blocked spec shows a confirmation dialog before execution.
  - Runner logs stream while executing, and Result view reflects updated metadata (DONE vs IN PROGRESS).
- `helm run` focuses the Run pane; `helm split` focuses Breakdown; `helm status` focuses Status; all still open the TUI shell.

## Implementation Notes

- Keep shared styling (badges, headers) reusable so spec-05 and spec-06 panes can plug into the shell.
- Limit in-memory log buffer (e.g., last N lines) to keep the TUI responsive.
- Home navigation should be easy to extend (e.g., using a simple enum and switch) without coupling pane implementations.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Repo config, metadata, and spec discovery
- spec-02-scaffold-command — Scaffold flow inside the TUI
- spec-03-implement-runner — Go runner for execution
