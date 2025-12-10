# Prompt for Implementing the Helm TUI

You are tasked with implementing the complete Helm Terminal UI in Rust using Ratatui, fully satisfying `docs/helm-overview.md` and `docs/helm-implementation-specs.md`. Follow these instructions precisely.

## What to read first
1. `docs/helm-overview.md` — functional behavior, flows, keys, and data requirements.
2. `docs/helm-implementation-specs.md` — architecture, design system, file plan, Makefile/CI requirements.
3. `docs/ratatui-references.md` — lists reference TUIs and widgets to reuse; submodules live in `references/`.

## Core directives
- Implement **all screens and flows end-to-end**: Home, Run (list/running/result), Breakdown/Spec Split (intro/input/running/done), Status Overview, Scaffold Wizard, Settings.
- **Copy/port widgets/layouts/patterns** from reference apps in `references/` as mapped in `docs/ratatui-references.md` (MenuList, HelpBar, Modal, ViewportCard/logs, badges, summary bar, responsive splits, clipboard helper, etc.). Bring the code into this repo; do not leave stubs.
- There is **no pre-existing Helm logic to call**. Implement the discovery/runner/split/scaffold/settings/data handling described in `docs/helm-overview.md` entirely within this codebase (you may structure it as services/modules, but they must be real, not mocks). The TUI should be backed by these implementations, not placeholders.
- Keyboard-first; mouse only for scrolling viewports. Alt-screen where specified.
- Maintain accessibility/responsiveness rules (min width 24 cols, help bar always visible, no color-only cues).
- Implement key normalization, kill/unmet-deps double-confirm, resume chip/clipboard flow.

## Project structure to produce
- `src/theme.rs` — palette + text styles (Base16 mapping to Helm tokens).
- `src/components/` — reusable wrappers (MenuList, Badge, SummaryBar, Flash, SpinnerLine, HelpBar, Modal, ViewportCard, FormField, ResumeChip, GraphView, etc.) adapted from references.
- `src/layout.rs` — rect helpers for responsive splits.
- `src/keys.rs` — key normalization enums/mapping.
- `src/app.rs` — state machine, router, event loop (lift from `references/crates-tui` skeleton).
- `src/screens/{home.rs,run.rs,split.rs,status.rs,scaffold.rs,settings.rs}` — screen implementations per spec.
- `src/services/` — adapters to Helm logic (spec discovery, dependency state, runner/split streaming with channels, settings IO, clipboard helper, editor launcher, session-id detection).
- Update `Cargo.toml` dependencies for Ratatui, Crossterm, Tokio, clipboard crate, etc., following versions used by reference repos where possible.

## Implementation steps (execute in order)
1. Sync and inspect reference code in `references/`; copy needed widgets/layout patterns rather than rewriting.
2. Set up theme/design tokens; ensure all components consume them.
3. Build foundational components (PageShell, HelpBar, Flash, Modal, ViewportCard) then menus/badges/summary bar.
4. Wire event loop/router (alt-screen, tick handling) using crates-tui skeleton.
5. Implement screens in this order: Home → Run (list/running/result) with log streaming & kill-confirm & resume copy → Split flow with editor + optional inline `edtui` fallback → Status graph view → Scaffold wizard → Settings form.
6. Hook data/config: repo config + user settings resolution, spec discovery, dependency computation, acceptance command resolution.
7. Enforce accessibility/responsiveness and validation behaviors from `docs/helm-overview.md`.
8. Add Make targets if new ones are needed; keep existing `Makefile` commands working.
9. Ensure CI passes (`cargo fmt`, `cargo clippy -D warnings`, `cargo test --all --locked`).

## CI/CD and tooling
- Keep `Makefile` as the dev entrypoint (`make all`, `make run ARGS="..."`, `make release`).
- Workflows in `.github/workflows/ci.yml` and `.github/workflows/release.yml` must remain green.
- Use composite action `.github/actions/setup-rust/action.yml` for Rust setup.

## Behavioral must-haves (non-negotiable)
- Keybinds, flows, and validation exactly match `docs/helm-overview.md`.
- Resume chip appears when session id regex matches; `c` copies `codex resume <uuid>` with graceful fallback.
- Double-press `esc`/`q` kill/quit confirmation with 2s timer and cancel on `n`.
- Lists/tables/logs maintain scroll position unless at bottom; drop-oldest over 2000 entries.
- Status view shows ASCII dependency graph with focus modes (All/Runnable/Subtree) and detail panel.

## Testing
- Run `make all` locally; ensure formatting, lint, and tests pass.
- Add/adjust tests as needed to cover key behaviors and state machines.

## Output expectations
- Clean, idiomatic Rust; minimal bespoke UI code thanks to reused components.
- File references use workspace-relative paths (e.g., `src/screens/run.rs`).
- Commit messages are **not** required by this prompt; just produce working code.

Deliver the complete implementation respecting these instructions. Do not omit any screen, control, or validation described in the source docs.***
