# Helm TUI Build Specification
Date: Dec 9, 2025

This document translates `docs/helm-overview.md` and `docs/ratatui-references.md` into an actionable build plan for the Helm TUI. It defines architecture, design system, reusable components, workflows, CI/CD, and Makefile targets so we can implement the terminal UI with minimal bespoke widget code.

## 1) Goals & Scope
- Ship the complete Helm terminal UI (Home, Run, Breakdown/Spec Split, Status, Scaffold, Settings) as described in `helm-overview.md`.
- Maximize reuse of proven Ratatui components/patterns from the reference deck to avoid hand-rolling widgets—copy/port them into this repo rather than leaving stubs.
- Provide deterministic developer ergonomics: Makefile commands, GitHub Actions (setup composite action + workflows for CI and Release).
- Keyboard-first; mouse only for viewport scroll. Alt-screen everywhere except where explicitly display-only.
- Implement **all Helm behavior in this repository**: discovery, dependency computation, runner/split streaming, scaffold workspace creation, settings persistence, acceptance command resolution. There is no external “existing logic” to depend on.

## 2) Non-Goals
- Do not redesign flows or shortcut validation rules defined in `helm-overview.md`.
- Do not invent new widgets when an existing reference provides a pattern; wrap/adapt instead.
- No new networked services; rely on existing filesystem + Codex integrations.

## 3) Technical Foundation
- **Language/runtime**: Rust (stable, toolchain pinned via `rust-toolchain.toml` if present; else minimum stable from CI setup below).
- **UI stack**: Ratatui + Crossterm; reuse scaffolding from `references/crates-tui` project skeleton (async runtime + state machine + Base16 theming).
- **Async**: Tokio (match crates-tui base), with channel/event loop from the template.
- **Clipboard**: use the same cross-platform helper as `spotify-tui`/`crates-tui` for resume-chip copy (with fallback flash when unavailable).
- **File discovery & logic**: consume existing Helm crates/modules: config resolution, specs discovery, runner, specsplit, scaffold, settings persistence.

## 4) Design System (reuse-first)
### Palette
- Start from `crates-tui` Base16 theme map; bind Helm tokens:
  - `primary` → base06; `accent` → base0D; `muted` → base03; `warning` → base0A; `success` → base0B; `surface` → base01; `border` → base04; `highlight` → base09.
- Single source of truth in `theme.rs`; exposed as styled primitives used by all widgets.

### Typography & Spacing
- Adopt `gitui`/`crates-tui` text styling for headings, pill badges, and help legends.
- Standard gutters/padding from PageShell (top/bottom margin + inner padding) to prevent repaint artifacts at narrow widths.

### Components (all reused/adapted)
- **PageShell (layout + help bar)**: lift layout primitives from `crates-tui` + `gitui` (persistent help legend row).
- **Menus**: side/stacked menu list from `gitui` with pointer + muted descriptions.
- **Badges & Summary Bar**: pill badges and count bar from `kmon`/`bottom` patterns.
- **Flash banners**: severity-padded banners from `crates-tui` logging view.
- **Spinner line**: footer-style spinner from `bottom`/`kmon` activity header.
- **ViewportCard (logs)**: scrollable log panel with footer and mouse wheel from `bottom` and `oxker`.
- **Modal dialogs**: centered rounded modal from `gitui`/`kmon`; supports two-step kill/unmet-deps confirmations.
- **Forms**: text inputs/toggles from `spotify-tui` settings layout + `tui_widgets` prompts for focus/validation states.
- **Dependency graph**: tree rendering borrowed from `xplr`-style list/tree widget (ASCII graph with selection highlight).
- **Clipboard/resume chip**: clipboard copy helper from `spotify-tui` auth flow with flash fallback.
- **Editor fallback**: optional inline editor widget via `edtui` for Split input when `$EDITOR` unavailable.

### Responsiveness
- Use `bottom`/`gitui` responsive splits: compute rects per frame; enforce min width 24 cols; clamp viewports with min height.
- List/table/viewports grow to remaining space; pad to avoid flicker.

## 5) Input & Keymap Rules
- Normalize keys (tab/shift+tab, ctrl+h/backspace, delete, ctrl+m/enter) before state handling.
- Global quit/cancel: `q` or `ctrl+c`; `esc` usually backs up; long-running phases require double-press within 2s (`esc` for stop, `q` for stop+quit) with modal feedback and `n` to cancel.
- Mouse: enable wheel scrolling inside viewports only.

## 6) Data & Config Wiring
- Repo config `helm.config.json` (specsRoot, initialized) must exist for non-scaffold flows; scaffold writes it.
- User settings `~/.helm/settings.json` (or `$HELM_CONFIG_DIR/settings.json`): specsRoot, mode, defaultMaxAttempts, acceptanceCommands, Codex model+reasoning per flow.
- Spec metadata per `metadata.json`; dependency state via `specs.ComputeDependencyState` (fields `CanRun`, `UnmetDeps`).
- Acceptance commands resolution order: repo config → scaffold defaults → user settings.
- All of the above must be implemented in this codebase; there is no upstream Helm library providing these functions.

## 7) Screen-by-Screen Build Checklist
For each screen, follow the behavior in `helm-overview.md`; the items below translate to implementation tasks with reference mappings.

### Home (bare `helm`)
1. Resolve config/settings; if missing/!initialized, auto-launch Scaffold wizard, then return.
2. Render menu list (Run / Breakdown / Status / Quit) using `gitui` menu widget; help bar shows `↑/↓ move`, `enter select`, `q quit`.
3. Alt-screen enabled; selection loops until Quit.

### Run (`helm run`)
**List phase**
- Data: discovered specs, with status badge and unmet deps line.
- Controls: `↑/↓/pgup/pgdn/j/k`, `f` toggle runnable-only filter, `enter` run, `q/ctrl+c` quit, `esc` close modal or quit, unmet-deps modal `y/n`.
- Layout: title, optional unmet modal, flash, MenuList; help legend with filter state.

**Running phase**
- Header spinner line with stage/attempt text parsed from log markers; resume chip when session id seen.
- Log viewport (no wrap, preformatted) with scroll+mouse; footer instructions.
- Double-press `esc`/`q` kill confirmation modal from `gitui` pattern; `c` copies resume.

**Result phase**
- Status line + flash (success/danger), remaining tasks bullet list, resume chip.
- Log viewport with footer; `enter/r` return to list, `c` copy resume, `q` quit, `esc/ctrl+c` exit.
- Rediscover specs on exit to refresh badges/deps, preserving selection where possible.

### Breakdown / Spec Split (`helm spec [-f file] [--plan-file path]`)
**Intro**: text card; `enter` continue; `q/esc` quit.

**Input**
- Instruction line; dev-note when `--plan-file` set; error flash inline.
- Draft preview card (first ~10 lines) using ViewportCard; footer shows line count or `(empty)`.
- Controls: `e` open `$EDITOR` (temp file preloaded, cursor at end); plain `enter` starts split only if draft non-empty, otherwise opens editor; `q` quits; `esc` back.
- Optional inline `edtui` fallback when no `$EDITOR` available.

**Running**
- Spinner line “Splitting via Codex…” or plan path; resume chip/flash; log viewport + footer; double-press kill modal.
- Backend: call `specsplit.Split` with guide path and settings.

**Done**
- Success: summary table (ID/Name/Depends on), warnings bullet list, log tail card; help (`enter/q/esc` exit, `r` jump to Run, `n` new split).
- Failure: danger flash, optional resume, log tail; same controls with retry.
- If no specs created: message “No specs were created.”

### Status Overview (`helm status`)
- Data: specs with deps/dependents computed.
- Focus modes: All / Runnable / Subtree (cycle `f`, `enter` sets subtree); selection highlight in ASCII graph using `xplr` tree style.
- Layout: summary badge bar + two-column counts; graph viewport card with footer showing spec count; detail panel table for selected spec; unmet deps flash when relevant.
- Controls: `↑/↓` move, `enter` focus subtree, `f` cycle focus, `r` reload, `q` quit all, `esc/ctrl+c` exit.

### Scaffold Wizard (`helm scaffold`)
- Steps: Intro → Mode picker → Acceptance commands → Options (specs root) → Confirm → Running → Complete.
- Use menu + form components from `spotify-tui`/`tui_widgets`; acceptance commands input supports `ctrl+w` drop-last.
- Running step uses spinner line with cancel; Complete shows created/skipped files list; danger flash on errors.

### Settings (`helm settings`)
- Form stack: specs root, mode toggle, default max attempts (int >0), acceptance commands (comma-separated), Codex model+reasoning per flow (scaffold/run worker/run verifier/split), Save row.
- Navigation `↑/↓`, `←/→` for toggles, text inputs editable when focused; validation errors inline; `enter` on Save persists and exits; `esc/ctrl+c` cancels.

## 8) Logs, Resume, Error Handling
- Stream stdout/stderr with tags; store up to 2000 entries, drop oldest beyond limit; preserve scroll when not at bottom.
- Session ID regex `^session id:\\s*([a-f0-9-]{36})$`; display resume chip and flash; `c` copies `codex resume <uuid>`.
- Long-running cancel: double `esc`/`q` within 2s; modal shows armed state; `n` cancels.
- Validation errors surface as flash or placeholder text; flows block until resolved.

## 9) Accessibility & Responsiveness
- Minimum width 24 cols; clamp heights so help bars remain visible.
- No color-only cues: badges include text; warnings have text labels.
- Recompute layout on every terminal resize.

- App state machine: enums for Screen + SubPhase (e.g., RunList/RunStreaming/RunResult, SplitIntro/SplitInput/SplitRunning/SplitDone).
- Use shared `PageShell` wrapper that accepts title, body, help items, optional flash/modal overlay.
- Event loop from `crates-tui` template with tick rate; spawn async tasks for runner/split to stream logs over channels.
- Shared models: `SpecRow` (metadata + computed dep state), `ResumeState`, `Flash`, `ModalState`, `ViewportState`.
- Router: bare `helm` enters Home; subcommands jump directly to their screen; all respect global quit semantics.
- All data/logic modules (config, discovery, dependency state, runner, split, scaffold, settings) must be implemented here; no mocks or placeholders in the final product.

## 11) Makefile (to be created)
Create `Makefile` with phony targets:
- `setup`: install toolchain & components (rustup toolchain install + `rustfmt`/`clippy`; ensure `just`/`cargo-binstall` optional), fetch submodules `git submodule update --init --recursive`.
- `fmt`: run `cargo fmt --all`.
- `lint`: run `cargo clippy --all-targets --all-features -D warnings`.
- `test`: run `cargo test --all`.
- `run`: `cargo run --bin helm --` (pass `ARGS=\"...\"` override via env).
- `clean`: `cargo clean`.
- `all`: `setup fmt lint test`.
- `release`: `cargo build --release` (optionally `--locked`); emits to `target/release/helm`.
- Support `RUSTFLAGS`/`CARGO_TARGET_DIR` passthrough; mark all as `.PHONY`.

## 12) GitHub Actions
### Composite action: `.github/actions/setup-rust/action.yml`
- Inputs: `toolchain` (default `stable`), `components` (`rustfmt, clippy`), `cache` (bool, default true).
- Steps: checkout (for actions using it), install rustup toolchain, add components, enable sccache or actions/cache for cargo registry + target, print versions.

### Workflow: CI (`.github/workflows/ci.yml`)
- Triggers: `pull_request`, `push` to `main`.
- Jobs (Ubuntu latest):
  1) `lint-test`:
     - uses setup action (toolchain `stable`, cache on).
     - `cargo fmt --all -- --check`.
     - `cargo clippy --all-targets --all-features -D warnings`.
     - `cargo test --all --locked`.
- Concurrency: group by ref to avoid duplicate runs.

### Workflow: Release (`.github/workflows/release.yml`)
- Trigger: `push` tag matching `v*`.
- Jobs:
  1) `build`: matrix over OS (ubuntu-latest, macos-latest); uses setup action; `cargo build --release --locked`; upload artifacts (`helm` binary zipped with README/LICENSE`).
  2) `publish`: needs `build`; uses `actions/create-release` to draft GitHub release with changelog notes; attach artifacts from previous job.
- Secrets: `GITHUB_TOKEN` provided; optional signing key if later added.

## 13) File/Module Plan
- `src/theme.rs`: palette & text styles (pulled from crates-tui Base16 map).
- `src/components/`: wrappers for reused widgets (MenuList, Flash, SpinnerLine, ViewportCard, Modal, SummaryBar, Badge, HelpBar, FormField, ResumeChip, GraphView).
- `src/app.rs`: state machine, router, event loop wiring.
- `src/screens/`: `home.rs`, `run.rs` (phases), `split.rs`, `status.rs`, `scaffold.rs`, `settings.rs`.
- `src/services/`: adapters to Helm logic (spec discovery, runner, split, settings IO, clipboard helper, editor launcher, resume detection, key normalization).
- `src/layout.rs`: rect helpers for responsive splits.
- `src/keys.rs`: normalized key enums and mapping tables.

## 14) Implementation Order (suggested)
1. Import reference code: sync submodules; copy/port layout + event loop skeleton from `references/crates-tui` and menu/modal widgets from `gitui`/`kmon`/`bottom` as needed.
2. Establish theme + design tokens; wire into all components.
3. Build PageShell, HelpBar, Flash, Modal, ViewportCard primitives.
4. Implement Home navigation.
5. Implement Run phases (list → running → result) with log streaming + kill confirmation + resume chip.
6. Implement Split flow with editor integration and optional inline editor fallback.
7. Implement Status graph view and detail panel.
8. Implement Scaffold wizard forms; then Settings form.
9. Add config resolution + dependency computation glue; add acceptance command resolution.
10. Polish accessibility/responsiveness; validate min widths and resize handling.
11. Wire Makefile targets; add CI + Release workflows; verify locally via `make all` and `act` if available.

## 15) Acceptance Criteria
- All behaviors in `helm-overview.md` are present with correct keybindings, flows, and validation.
- UI uses reused widgets/patterns listed above; minimal bespoke styling.
- Makefile and GitHub Actions exist and run successfully (`make all` passes on fresh clone; CI green on PR).
- Resume chip works with clipboard fallback; kill/unmet-deps confirmations require double-press.
- Responsive layouts remain legible at 24-col width; help bar always visible.
- No regressions to existing runner/split/scaffold logic; UI remains display/stateful only.

---
This spec is the source of truth for implementing Helm’s TUI. Keep it in sync with `docs/helm-overview.md` if flows change.
