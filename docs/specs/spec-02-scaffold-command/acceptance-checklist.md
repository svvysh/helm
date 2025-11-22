# Acceptance Checklist — spec-02-scaffold-command

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] In a temp repo without `helm.config.json`, running `go run ./cmd/helm` or `go run ./cmd/helm scaffold` shows only the initialization prompt with the default specs root `specs/` and allows editing it.
- [ ] Confirming scaffold creates the specs root with README, prompt templates, runner script, spec-splitting guide, and the `spec-00-example` folder; existing files are left untouched and called out in the summary.
- [ ] After scaffold, `helm.config.json` exists with `initialized=true` and the chosen `specsRoot`.
- [ ] Re-running `helm` after initialization skips the scaffold gate and routes to the home menu (run/breakdown/status options).
- [ ] Cancelling the scaffold prompt exits cleanly with guidance to rerun Helm later.
