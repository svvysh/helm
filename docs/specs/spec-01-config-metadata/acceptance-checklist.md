# Acceptance Checklist — spec-01-config-metadata

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] Creating a `docs/specs` directory with a couple of `spec-*` folders and `metadata.json` files leads to those specs being discovered by `DiscoverSpecs`.
- [ ] `ComputeDependencyState` marks a spec as runnable only when all its dependencies have status `"done"`.
- [ ] `LoadSettings` returns sensible defaults when `.cli-settings.json` does not exist.
- [ ] `LoadMetadata` and `SaveMetadata` correctly round-trip `SpecMetadata` fields including `dependsOn` and `acceptanceCommands`.
- [ ] The root CLI command can import and compile against the new `internal/config`, `internal/specs`, and `internal/metadata` packages.
