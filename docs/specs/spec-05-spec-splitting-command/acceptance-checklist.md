# Acceptance Checklist — spec-05-spec-splitting-command

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] `go run ./cmd/helm spec` runs the Breakdown flow directly. The Breakdown pane is also reachable from the shell opened by bare `helm`; `q`/`Esc` returns to home.
- [ ] The Breakdown flow accepts pasted text or a file path (Ctrl+O), then immediately streams Codex stdout/stderr during split (no spec text preview). Errors are surfaced in the done view along with recent logs.
- [ ] A valid Codex JSON plan results in new `spec-*` folders under the configured `specsRoot`, each containing `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, and `implementation-report.md`.
- [ ] Dependencies from the plan appear in both `metadata.json.dependsOn` and the `## Depends on` section of `SPEC.md`.
- [ ] Existing spec folders are not overwritten without explicit confirmation; collisions are surfaced in the completion summary.
- [ ] The completion view offers a way to jump to the Run pane; otherwise returning lands on the home menu.
