# spec-01-config-metadata — Repo config, metadata, and spec discovery

## Summary

Define the repo-scoped configuration (`helm.config.json`), metadata structures, and spec discovery helpers that the TUI will rely on for its first-run experience. The config remembers the specs root and whether scaffold has already been performed so the TUI can gate features until initialization is complete.

## Goals

- Represent spec metadata (`metadata.json`) and repo config (`helm.config.json`) in Go.
- Persist repo config in the working repo (not globally) with fields for specs root and initialization state.
- Provide helpers to detect “first-run” vs “initialized” and to discover spec folders under the configured specs root.
- Keep strict validation of model/effort options so later Codex calls can reuse them.

## Non-Goals

- No Bubble Tea UI flows yet (the first-run prompt will be handled in later specs).
- No Codex calls or runner orchestration.
- No editing of metadata from a UI; just load/save.

## Detailed Requirements

1. **Metadata Model**
   - Keep the `SpecStatus` type and `SpecMetadata` struct (id, name, status, dependsOn, lastRun, notes, acceptanceCommands) as already defined in the previous version of this spec. Validation remains unchanged.

2. **Repo Config File (`helm.config.json`)**
   - Stored at the repository root where `helm` is invoked (same directory as `.git` in most cases).
   - Define a `Mode` enum (`strict`, `parallel`) reused from the prior spec.
   - Define:

     ```go
     type RepoConfig struct {
         SpecsRoot          string        `json:"specsRoot"`
         Initialized        bool          `json:"initialized"`
         Mode               Mode          `json:"mode"`
         DefaultMaxAttempts int           `json:"defaultMaxAttempts"`
         AcceptanceCommands []string      `json:"acceptanceCommands"`
         CodexScaffold      CodexChoice   `json:"codexScaffold"`
         CodexRunImpl       CodexChoice   `json:"codexRunImpl"`
         CodexRunVer        CodexChoice   `json:"codexRunVer"`
         CodexSplit         CodexChoice   `json:"codexSplit"`
     }

     type CodexChoice struct {
         Model     string `json:"model"`
         Reasoning string `json:"reasoning"`
     }
     ```

   - Defaults when the file is absent:
     - `SpecsRoot = "specs"`
     - `Initialized = false`
     - `Mode = ModeStrict`
     - `DefaultMaxAttempts = 2`
     - `AcceptanceCommands = []`
     - Codex defaults: scaffold/split → `gpt-5.1` with `reasoning="medium"`; run impl/ver → `gpt-5.1-codex` with `reasoning="medium"`.
   - Allowed model/reasoning pairs remain the fixed list from the earlier spec; `ValidateRepoConfig` must reject invalid combinations.
   - Provide helpers:
     - `LoadRepoConfig(root string) (*RepoConfig, error)` reading `<root>/helm.config.json`.
     - `SaveRepoConfig(root string, cfg *RepoConfig) error` writing the same path.
     - `DefaultRepoConfig() *RepoConfig` and `ValidateRepoConfig(cfg *RepoConfig) error`.
     - If `.cli-settings.json` exists (from older scaffold flows) and `helm.config.json` is missing, bootstrap defaults from it but persist only to `helm.config.json`.

3. **First-Run Detection**
   - Add a helper `NeedsInitialization(cfg *RepoConfig, fsExists func(string) bool) (bool, string)` that returns whether the TUI should show the initialization gate, plus a human-friendly reason, e.g.,
     - `helm.config.json` missing
     - `specsRoot` directory missing
     - `Initialized` is false
   - The helper must not mutate files; it only inspects configuration and filesystem existence.

4. **Spec Folder Representation and Discovery**
   - Keep the `SpecFolder` struct (`ID`, `Name`, `Path`, `Metadata`, `Checklist`, `CanRun`, `UnmetDeps`).
   - `DiscoverSpecs(root string)` now uses the repo-configured `SpecsRoot` (default `specs/`).
   - Require each spec folder to contain `SPEC.md` and `metadata.json`; `acceptance-checklist.md` remains optional.
   - Provide `ComputeDependencyState` exactly as before to fill `CanRun` and `UnmetDeps` without mutating `metadata.status`.

5. **Root Command Wiring (Minimal)**
   - Update the Cobra root initialization so every command loads `helm.config.json` first and surfaces a clear error if it cannot be read/validated.
   - If the config says `Initialized=false` or the `SpecsRoot` does not exist, the command should exit with a friendly message pointing the user to the TUI initialization flow (implemented in later specs).
   - Do not build any Bubble Tea screens here; just ensure all commands have access to `RepoConfig` and the resolved specs root.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Unit tests cover:
  - `LoadRepoConfig`/`SaveRepoConfig` round-trip via a temp repo root.
  - Validation of model/reasoning combinations and rejection of invalid pairs.
  - `NeedsInitialization` returning expected reasons for missing config, missing specs root, and `Initialized=false`.
  - `DiscoverSpecs` + `ComputeDependencyState` using a temp specs root (no writes into the real `docs/specs/`).
- Running `go run ./cmd/helm --help` still shows stub subcommands, but they fail fast with a readable message when `helm.config.json` is missing or invalid.

## Implementation Notes

- Keep repo config strictly repo-scoped; no `$HOME` state. This allows different repos to have different `specsRoot` values while the CLI behavior stays consistent.
- The default specs root changed from `docs/specs` to `specs` to match the new TUI prompt; the helper must honor custom values set during initialization.
- Tests should pass a temp directory as the repo root and clean up any written `helm.config.json` files.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
