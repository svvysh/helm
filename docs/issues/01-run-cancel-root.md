# Issue: Run cancelability & jump-to-run root bug

## Problem
- `helm` jump-to-run after spec split calls `runtui.Run` with `Root: specsRoot`, causing resolution to look in `<specsRoot>/specs` and fail in repos using default `specs/` at repo root (`cmd/helm/main.go:339-371`).
- Run pane “Stop this run?” prompt is non-functional because `runner.Run` uses `context.Background()` and TUI `killProcess` only clears a flag (`internal/tui/run/model.go:398-435`, `internal/tui/run/runner.go:18-79`). Users cannot abort long Codex sessions.

## Desired fix (conceptual)
- Pass the actual repo root to jump-to-run; keep `SpecsRoot` as-is.
- Thread a cancelable context from the run TUI into `runner.Run`; ensure log streaming goroutines respect cancellation and Codex processes are killed. The kill confirmation should send the cancel, propagate an error/exit code, and surface the interruption in the result view.
- Update unit tests (or add new) covering jump-to-run path and cancel flow.
- Reflect any logic or UX changes in relevant specs under `docs/specs/` so fresh runs exercise the corrected behavior.

## Acceptance criteria
- Jump-to-run starts in the correct repo root and successfully resolves existing specs without needing a nested `specs` folder.
- Pressing `q` → `y` (or chosen kill binding) stops the run, terminates Codex sub-processes, and shows a canceled status in the result pane.
- New/updated tests cover both behaviors and pass.
- Specs and docs updated to describe cancel behavior and correct jump-to-run flow.

## Prompt (copy/paste to LLM)
```
You are a senior Go engineer. Fix the Helm CLI/TUI issues:
1) In cmd/helm/main.go, when spec-split chooses “jump to run”, pass the actual repo root (cwd) to runtui.Run; keep SpecsRoot unchanged so specs resolve correctly. Add/adjust tests for this path.
2) Make the run TUI cancelable. Thread a cancelable context from the TUI into runner.Run; ensure Codex child processes are killed on cancel, log streams close cleanly, and the result view shows a canceled/aborted state. The existing “Stop this run?” confirmation should trigger cancel. Add tests covering cancel.
Update code, tests, and any relevant specs under docs/specs/ so a fresh runner includes these fixes. Do not regress other behaviors.
```
