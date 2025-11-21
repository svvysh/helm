# Acceptance Checklist — spec-06-status-command

## Automated commands

- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.

## Manual checks

- [ ] Running `go run ./cmd/helm status` presents a TUI instead of plain text.
- [ ] Summary counts at the top correctly reflect the number of TODO, IN PROGRESS, DONE, and BLOCKED specs based on `metadata.json` and dependencies.
- [ ] The dependency graph view shows specs in a tree structure with ASCII connectors.
- [ ] The table view shows ID, name, status, dependencies, and last run time.
- [ ] Pressing `tab` toggles between the graph view and the table view.
- [ ] Pressing `f` cycles through focus modes (all / runnable / subtree).
- [ ] Selecting a spec in the table and pressing `enter` updates the subtree focus accordingly.
- [ ] Specs that are not done and have unmet dependencies are clearly marked as BLOCKED in the UI.
