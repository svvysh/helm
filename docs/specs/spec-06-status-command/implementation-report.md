# Implementation Report for spec-06-status-command — Status pane and dependency graph

- Mode: strict
- Max attempts: 2
- Attempts used: 2
- Final verifier status: ok

## Remaining tasks

{
  "remainingTasks": []
}

## Final worker output

SUMMARY
- Implemented reverse dependency tracking (`internal/tui/status/model.go:149-228`) so each spec entry now knows which other specs depend on it, enabling accurate downstream analysis.
- Reworked subtree focus to walk the dependents graph (`internal/tui/status/model.go:487-507`), satisfying the outstanding TODO so `f` and `enter` now surface the selected spec plus everything blocked by it.
- Ensured the visible table/graph refresh logic picks up the new focus sets without disturbing summary counts, key bindings, or reload behavior (`internal/tui/status/model.go:352-534`).
- Verified repo health by running the strict acceptance pipeline `make all`, covering deps, fmt, vet, lint, tests, build, and release artifacts.

CHANGELOG
- `internal/tui/status/model.go`: Added a `Dependents` field, built reverse links, deduped/sorted them, updated focus filtering to use dependents, refreshed subtree selection logic, and wired the reload/newModel paths to rebuild this state so downstream focus views now pull the correct specs.

TRACEABILITY
- “`go test ./...` and `go vet ./...` succeed”: `make all` runs both commands (plus lint/build) and completed without errors in this workspace.
- “`go run ./cmd/helm status` … opens the TUI shell with the Status pane selected”: `cmd/helm/main.go:272-289` keeps the `status` subcommand wired to `statusui.Run`, so invoking it opens the status pane directly; within the home loop, selecting Status still calls the same entrypoint (`cmd/helm/main.go:144-166`), so pressing `q` exits the pane and returns to the menu.
- “Status pane shows summary counts, dependency graph, and table view; `tab` toggles views”: Rendering already draws counts and graph/table bodies (`internal/tui/status/view.go:12-96`), and the key handler keeps `tab` toggling between view modes (`internal/tui/status/model.go:303-338,433-439`).
- “`f` cycles focus modes (all/runnable/subtree); `enter` on a row sets subtree focus”: Focus cycling and selection live at `internal/tui/status/model.go:441-474`, and the new dependents-based subtree collection (`internal/tui/status/model.go:487-507`) now returns the selected spec plus every spec that depends on it, matching the acceptance TODO.
- “BLOCKED specs are indicated even if metadata says todo/in-progress”: Entry construction still sets `blocked := len(UnmetDeps)>0 && status!=done` before asking `theme.StatusBadge`, so blocked rows render with the BLOCKED badge (`internal/tui/status/model.go:158-173`), satisfying the visual requirement.
- “Pressing `q` from the pane returns to the home menu without exiting the entire app”: The status model exits on `q` (`internal/tui/status/model.go:303-318`), and the home loop simply relaunches when the pane finishes (`cmd/helm/main.go:144-166`), so users land back on the menu rather than quitting.

RUNBOOK
1. Status CLI: from the repo root, run `go run ./cmd/helm status` (requires Go 1.21+ and the repo initialized so specs exist). Use the on-screen hints to navigate (`tab`, `f`, `enter`, `q`).
2. Acceptance suite: execute `make all` to run deps, tidy, formatting, vet, lint, tests, build, and cross-platform release artifacts. Go, gofumpt, goimports, and golangci-lint must be available (the Makefile installs tooling automatically into `$(go env GOBIN)`).

MANUAL SMOKE TEST
- `go run ./cmd/helm status`; confirm the TUI opens showing summary badges plus the table view.
- Press `tab`; expect the ASCII dependency graph for the current focus to appear, `tab` again returns to the table.
- Press `f` repeatedly; ensure focus cycles All → Runnable → Subtree (if no subtree target yet it prompts for selection), and the summary line updates with counts for the filtered set.
- In table view, move to a spec that other specs depend on, press `enter`; the table and graph should shrink to that spec plus any dependents, confirming downstream focus uses the new reverse links.
- Press `q`; you should land back at the home menu (if launched via bare `helm`) or the CLI prompt (if run via subcommand).

OPEN ISSUES & RISKS
- The spec text mentions “`helm status` runs a standalone status view (non-shell)” while the acceptance checklist says it should open “the TUI shell with the Status pane selected.” The current implementation launches the standalone status pane directly; I assumed that satisfies both wordings, but if a full multi-pane shell is expected this discrepancy should be clarified in a follow-up.
