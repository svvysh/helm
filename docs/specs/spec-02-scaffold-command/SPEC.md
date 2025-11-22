# spec-02-scaffold-command — Scaffold flow inside the TUI

## Summary

Implement the initialization flow that runs when a repo is not yet set up for Helm. The flow lives inside the TUI and is triggered on first open (or via `helm scaffold`). It asks for confirmation to scaffold the configured specs root, writes starter templates, and marks the repo as initialized in `helm.config.json`.

## Goals

- Detect an uninitialized repo using `helm.config.json` and the presence of the specs root directory.
- Present a minimal first-run TUI gate that only offers to scaffold the specs root (default `specs/`).
- Create the specs workspace (templates, runner script, example spec) without clobbering existing files.
- Persist `Initialized=true` and the chosen specs root back to `helm.config.json`, with guidance on how to re-run scaffold (delete the config file).

## Non-Goals

- No multi-pane home menu here; this flow runs **before** the home panes unlock.
- No spec splitting, run, or status behavior.
- No Codex calls beyond writing prompt templates.

## Detailed Requirements

1. **Entry Conditions**
   - The TUI starts by loading `helm.config.json` (from spec-01).
   - If `NeedsInitialization` is true, the only visible screen is an initialization prompt. Otherwise control passes to the home menu (spec-04).

2. **Initialization Prompt**
   - Show text similar to: "Helm has not been initialized in this repo. Scaffold `<specsRoot>` now?"
   - Default `specsRoot` comes from config (default `specs/` if unset). Allow the user to edit it before confirming.
   - Buttons: `Yes, scaffold` and `Quit`. If the user cancels, exit with a helpful message.

3. **Scaffold Steps (upon confirmation)**
   - Create the specs root directory (and parents) if missing.
   - Write or keep the following files relative to `SpecsRoot`:
     - `README.md`
     - `.cli-settings.json` (legacy defaults derived from repo config values)
     - `implement.prompt-template.md`
     - `review.prompt-template.md`
     - `implement-spec.mjs`
     - `spec-splitting-guide.md`
     - `spec-00-example/` folder with `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, `implementation-report.md`.
   - Do **not** overwrite files that already exist; instead surface a summary of skipped files. Overwrite policy can be revisited later but is not required now.

4. **Config Persistence**
   - After successful scaffold:
     - Set `RepoConfig.SpecsRoot` to the confirmed path.
     - Set `RepoConfig.Initialized = true`.
     - Keep any user-provided `AcceptanceCommands`, `Mode`, and Codex choices in the config.
     - Save to `<repo>/helm.config.json`.
   - Show a short note: "To scaffold again and overwrite, delete helm.config.json and rerun Helm."

5. **CLI Entrypoint Behavior**
   - `helm scaffold` launches the same TUI gate. If the repo is already initialized, show a short message pointing to the main TUI instead of running scaffold again.
   - `helm` (no subcommand) and other subcommands should automatically route to this gate when initialization is needed.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- Running `go run ./cmd/helm` in a temp repo with no `helm.config.json` presents only the initialization prompt and lets the user edit the specs root before confirming.
- After choosing `Yes`, the specs root is created with all listed files, none of the pre-existing files are overwritten, and `helm.config.json` is written with `initialized=true` and the chosen `specsRoot`.
- If the user cancels, the process exits cleanly with a hint to rerun Helm when ready.
- Re-running `helm` after initialization skips the scaffold gate and proceeds to the home menu (spec-04).

## Implementation Notes

- Reuse Bubble Tea components for confirmation dialogs; keep the flow short (no extra steps beyond path confirmation and summary).
- Respect the repo-local config only; do not write to `$HOME`.
- Tests should scaffold into a temp directory so tracked `docs/specs/` remains untouched.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Repo config, metadata, and spec discovery
