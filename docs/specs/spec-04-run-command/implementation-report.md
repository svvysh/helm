# Implementation Report for spec-04-run-command — run command TUI and spec execution

- Mode: strict
- Max attempts: 2
- Attempts used: 1
- Final verifier status: ok

## Remaining tasks

{
  "remainingTasks": []
}

## Final worker output

SUMMARY
- Built the new Bubble Tea workflow for `helm run`, adding list/running/result phases with filtering, dependency warnings, and metadata refresh in `internal/tui/run/model.go` plus the styled views in `internal/tui/run/view.go`.
- Streamed logs from `node docs/specs/implement-spec.mjs`, captured attempt progress, and propagated MAX_ATTEMPTS/CODEX env vars through the subprocess helpers in `internal/tui/run/runner.go`.
- Reloaded spec metadata/implementation reports after each run so DONE/IN PROGRESS states and remaining tasks are reflected back in the list view, matching spec expectations.
- Added focused tests in `internal/tui/run/runner_test.go` for attempt detection, remaining-task parsing, and environment propagation to guard the new helpers.
- Rewired `cmd/helm/main.go` so `helm run` launches the TUI, and updated `go.mod`/`go.sum` for new list dependencies; repo verification completed via `make all`.

CHANGELOG
- `cmd/helm/main.go`: replaced the old single-spec runner CLI with the interactive TUI launcher, removing direct `runner.Runner` usage.
- `internal/tui/run/model.go`: introduced the primary Bubble Tea model (phase transitions, filtering, confirmation prompts, metadata refresh, viewport handling, log retention, and run lifecycle commands).
- `internal/tui/run/view.go`: defined list-item delegate rendering with status badges, dependency summaries, last-run info, and the running/result panes with instructional copy.
- `internal/tui/run/runner.go`: added subprocess management for `implement-spec.mjs`, streaming stdout/stderr into Bubble Tea messages, watching exit codes, and building child env vars.
- `internal/tui/run/runner_test.go`: covered `parseAttemptLine`, `parseRemainingTasks`, and `buildRunnerEnv` behavior.
- `go.mod`, `go.sum`: tidied modules and recorded additional transitive deps (`github.com/muesli/reflow`, `github.com/sahilm/fuzzy`) needed by the list component.

TRACEABILITY
- “`go test ./...` and `go vet ./...` succeed” → satisfied as part of `make all`, which ran `go mod download`, `go vet ./...`, `golangci-lint run ./...`, `go test ./...`, `go build`, and the release matrix without errors.
- “Running `go run ./cmd/helm run` shows a TUI list with IDs/names/status badges, allows filtering to runnable specs, and prompts before running blocked specs” → list phase implemented in `internal/tui/run/model.go` + `internal/tui/run/view.go` using `internal/specs` discovery, badge styling, and `f` toggles; unmet dependencies trigger the `[y/N]` dialog.
- “Selecting a runnable spec starts the runner, streaming logs, and on completion updates DONE status” → `startRun` + `startRunnerCmd` in `internal/tui/run/model.go`/`runner.go` launch `implement-spec.mjs`, feed logs into the viewport, and `refreshSpecs` reloads metadata so the list reflects DONE states after verifier `STATUS: ok`.
- “STATUS: missing keeps status IN PROGRESS and shows remaining tasks” → after each run, `parseRemainingTasks` (`internal/tui/run/runner.go`) extracts the JSON block from `implementation-report.md`; result view (`internal/tui/run/view.go`) prints the task list while metadata reload keeps the badge at IN PROGRESS.

RUNBOOK
- Run the TUI: `go run ./cmd/helm run`. Requirements: Node (for `implement-spec.mjs`), the Codex CLI on PATH, and `docs/specs` populated. Keys: `↑/↓` to move, `f` to toggle runnable filter, `enter` to run selected spec, `q` to quit, `y/N` to confirm dependency overrides or early termination.
- During a run, `q` prompts before killing the subprocess; logs stream live in the viewport and show attempt counts parsed from the script output.
- Acceptance commands: `make all` (downloads modules, runs gofumpt/goimports, vet, golangci-lint, go test, go build, and cross-platform release builds). Ensure `$GOBIN` is writable so the formatter/linter helpers can be installed automatically.

MANUAL SMOKE TEST
1. `go run ./cmd/helm run` → confirm the list shows each `spec-*` with a colored badge, unmet dependency summary, and last-run note.
2. Press `f` to toggle “Runnable only” and verify blocked/done specs disappear until toggled back.
3. Highlight a blocked spec, press `enter`, and ensure the unmet-dependency confirmation appears; press `n` to cancel.
4. Select a runnable spec, hit `enter`, and watch the running view stream `implement-spec.mjs` logs plus attempt counts. Hit `q` and confirm you must press `y` before the process is killed.
5. Let a run finish; the result view should show spec status (DONE or IN PROGRESS). For IN PROGRESS runs, confirm remaining tasks from `implementation-report.md` are listed. Press `r` or `enter` to return to the refreshed list.

OPEN ISSUES & RISKS
- Attempt tracking and remaining-task parsing rely on the current `implement-spec.mjs` log/report format; if that script changes its banners or report layout, the progress indicator or “remaining tasks” section may stop updating until the regex/JSON parser is adjusted.
- The log viewport keeps the last 2,000 lines in memory; extremely verbose runs will truncate older output and could still consume noticeable memory.
- Killing the subprocess via `q`/`y` forcibly terminates Node; partial side effects (e.g., half-written metadata) are surfaced as-is and may require manual cleanup.
- The result view surfaces metadata reload errors but still leaves the spec list in its previous state; if specs outside the run were modified concurrently, users may need to restart `helm run` to refresh the full list.
