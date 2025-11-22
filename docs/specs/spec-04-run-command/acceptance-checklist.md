# Acceptance Checklist — spec-04-run-command

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] Running `go run ./cmd/helm run` presents a TUI list of specs rather than plain text output.
- [ ] Each spec row shows ID, name, and a status badge (TODO / IN PROGRESS / DONE / BLOCKED).
- [ ] Pressing `f` filters the list to show only runnable specs.
- [ ] Selecting a spec with unmet dependencies presents a confirmation dialog before running.
- [ ] Selecting a runnable spec starts `implement-spec.mjs` as a subprocess and streams its logs into the TUI.
- [ ] When the subprocess exits, the TUI shows a Result view summarizing the outcome.
- [ ] After returning to the list, the spec’s status is updated according to the new `metadata.json`.
