# Cross-Project Spec Runner CLI — Specs Workspace

This `docs/specs/` directory contains the specification bundles and templates used to build and evolve the Cross-Project Spec Runner CLI.

The refined product model is **TUI-first**:

1. Run `helm` (or any subcommand) inside a repo to open the TUI.
2. On first open, it prompts for the specs folder (default `specs/`) and writes `helm.config.json` in the repo root with `specsRoot` and an `initialized` flag.
3. If the repo has never been scaffolded, the TUI shows a single call-to-action to scaffold the chosen specs root; after scaffolding it records the initialization flag and explains how to re-run scaffold (delete `helm.config.json`).
4. Once initialized, the home menu offers three panes:
   - **Breakdown specs** — feed a large spec (or file) and split into `spec-*` folders.
   - **Run specs** — pick a spec, stream worker/verifier attempts, and update `metadata.json` + `implementation-report.md`.
   - **Status** — browse overall readiness and dependencies.
5. CLI entrypoints (`helm scaffold`, `helm run`, `helm spec`, `helm status`) run directly in CLI/mini-flows; the **only** way to open the multi-pane TUI is by invoking bare `helm`.

**Testing convention:** when writing automated tests or dry-running scaffold/runner flows, point `SpecsRoot` to a temp directory (e.g., `t.TempDir()/specs-test`) so the tracked `docs/specs/` tree stays untouched.

## Folder layout

- `implement.prompt-template.md` — worker prompt template (strict/parallel handled by settings).
- `review.prompt-template.md` — verifier prompt template.
- `.cli-settings.json` — legacy defaults (kept for reference; new flow persists per-repo config in `helm.config.json`).

Example spec folders:

- `spec-00-foundation/` — CLI skeleton and TUI entrypoints.
- `spec-01-config-metadata/` — repo config, metadata model, and first-run detection.
- `spec-02-scaffold-command/` — scaffold flow inside the TUI and file generation.
- `spec-03-implement-runner/` — worker/verifier loop in Go.
- `spec-04-run-command/` — TUI home and Run pane integration.
- `spec-05-spec-splitting-command/` — Breakdown/`split` pane.
- `spec-06-status-command/` — Status pane and dependency graph.
- `spec-07-tui-ux-refresh/` — Modern cohesive TUI theme and component system using Bubble Tea ecosystem libraries.

## Using this workspace

1. Pick a spec folder whose `metadata.json.status` is "todo" and whose dependencies are "done".
2. Use the Go runner (`helm run --exec <spec-id>` in headless mode or via the TUI) to execute a spec.
3. Follow the remaining tasks reported by the verifier until `STATUS: ok`.

The Go TUI (`helm`) automates this flow across the home, Run, Breakdown, and Status panes.
