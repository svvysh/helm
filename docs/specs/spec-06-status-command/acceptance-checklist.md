# Acceptance Checklist — spec-06-status-command

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] `go run ./cmd/helm status` runs a standalone status view (non-shell). Within the shell opened by bare `helm`, selecting Status mounts the pane and `q` returns to home without exiting the process.
- [ ] Summary counts for TODO / IN PROGRESS / DONE / BLOCKED are shown at the top.
- [ ] Dependency graph renders with ASCII connectors and reflects dependencies from metadata.
- [ ] Table view shows ID, Name, Status, Deps, Last Run; rows are scrollable.
- [ ] `tab` toggles graph/table; `f` cycles all / runnable / subtree modes; `enter` on a row sets subtree focus.
- [ ] Specs with unmet dependencies are visually indicated as BLOCKED even if their metadata status is todo/in-progress.
