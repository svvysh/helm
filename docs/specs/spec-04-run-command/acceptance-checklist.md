# Acceptance Checklist — spec-04-run-command

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] Running `go run ./cmd/helm` in an initialized temp repo opens the home menu with Run/Breakdown/Status options.
- [ ] In an uninitialized repo the TUI shows only the scaffold gate; after completing scaffold it returns to the home menu without restarting the process.
- [ ] The Run pane lists specs with ID, name, and status badges (TODO / IN PROGRESS / DONE / BLOCKED) plus unmet dependency hints.
- [ ] Pressing `f` toggles All vs Runnable filters; pressing `n` immediately runs the first runnable spec.
- [ ] Selecting a spec with unmet dependencies triggers a confirmation dialog before running.
- [ ] During execution, runner logs stream in a viewport; cancelling prompts for confirmation. On completion, the Result view shows DONE vs IN PROGRESS and any remaining tasks.
- [ ] Returning to the list refreshes metadata so the updated status is visible.
- [ ] `helm run/spec/status` execute their flows directly and do not open the TUI shell; only bare `helm` opens the shell.
