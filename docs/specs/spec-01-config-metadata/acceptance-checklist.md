# Acceptance Checklist — spec-01-config-metadata

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] With no `helm.config.json`, `LoadRepoConfig` (via a temp repo root) returns defaults: `specsRoot="specs"`, `initialized=false`, and default Codex choices.
- [ ] `ValidateRepoConfig` rejects an invalid model/reasoning pair and succeeds on allowed combinations.
- [ ] `NeedsInitialization` reports missing config, missing specs root, and `Initialized=false` as separate reasons without mutating files.
- [ ] Creating a temp specs root with a couple of `spec-*` folders containing `metadata.json` leads to those specs being discovered by `DiscoverSpecs` and dependency readiness computed by `ComputeDependencyState`.
- [ ] The Cobra root command loads repo config at startup and exits with a friendly message when the repo is not initialized (no `helm.config.json` or missing specs root).
