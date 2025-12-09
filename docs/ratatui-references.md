# Ratatui Reference Deck (Dec 9, 2025)

Purpose: Collect the strongest Ratatui-based TUIs and widget crates we can mine so we build Helm’s Ratatui UI with minimal new UI code. Mapped against the Helm spec in `docs/helm-overview.md`.

## High-value reference apps (cloned as submodules in `references/`)
- **crates-tui** — Ratatui crates.io explorer with async fetch, tabbed sections, clipboard copy, and Base16 theming; solid modern app skeleton. citeturn0search0
- **gitui** — fast git client with context help, modal dialogs, stacked list/detail panes, and responsive splits. citeturn1search5
- **bottom** — cross-platform process/system monitor; showcases resizable grids, scrollable charts/logs, and status footers. citeturn4search0
- **spotify-tui** — menu-driven Spotify client; demonstrates navigation list, settings flow, and clipboard auth URL copy (upstream of Spotatui). citeturn3search0
- **xplr** — hackable file manager with tree/list navigation and hint area; good inspiration for dependency graph + detail panel. citeturn1search0
- **kmon** — kernel module manager/monitor with dashboards, modals, and log panes built on Ratatui. citeturn5search1
- **rainfrog** — database TUI with query textarea, schema tree, status flashes, and mouse-aware scroll. citeturn2search1
- **oxker** — Docker container TUI with multi-pane tables, log viewport, filtering, and configurable keymap. citeturn2search0

_All eight are already added as git submodules under `references/` for quick code spelunking._

## Widget / library grab-bag (ready-made pieces)
- **tui_widgets** crate: bundle of third-party widgets (popups, cards, prompts, scroll view) ready to drop into layouts. citeturn0search5
- **Editor widget (`edtui`)**: vim-like editor with wrap/search/mouse/highlighting—fallback inline Split editor. citeturn0search11
- **Ratatui templates**: `cargo generate ratatui/templates` bootstraps event loop, routing, theming scaffolds. citeturn0search3

## Helm component → reference mapping
- **PageShell (title/body/help padding)** — layout & routing patterns from `crates-tui` and `gitui`; both keep help hints visible via Ratatui layout primitives. citeturn0search0turn1search5
- **Menu list (Home / mode pickers)** — `gitui` side menu + `spotify-tui` navigation list cover pointer, descriptions, and key hints. citeturn1search5turn3search0
- **Badges & summary bar** — `kmon` and `bottom` show colored pill badges and count bars we can mirror for spec statuses. citeturn5search1turn4search0
- **Flash banners (info/success/warning/danger)** — `crates-tui` logging view demonstrates padded severity flashes. citeturn0search0
- **Spinner line + status text** — `bottom` footers and `kmon` activity monitor map neatly to our Run/Split spinner line. citeturn4search0turn5search1
- **ViewportCard (scrollable log + footer)** — `bottom` log widgets and `oxker` log panel provide scroll + mouse support with fixed footers. citeturn4search0turn2search0
- **Dependency graph (Status)** — use `xplr` tree/list rendering for ASCII graph + selection highlighting. citeturn1search0
- **Forms & toggles (Settings/Scaffold/Options)** — `spotify-tui` settings layout plus `tui_widgets` prompts/inputs for focus/validation. citeturn3search0turn0search5
- **Modal / confirm dialogs (kill/quit, unmet deps)** — `gitui` confirm dialogs and `kmon` modals show centered rounded borders and stacked actions. citeturn1search5turn5search1
- **Help bar (key legend)** — `crates-tui` and `gitui` render per-mode key legends; reuse their truncation/alignment logic. citeturn0search0turn1search5
- **Resume chip / clipboard copy** — `crates-tui` copies `cargo add` commands; `spotify-tui` copies auth URLs—lift clipboard fallback for resume chips. citeturn0search0turn3search0
- **Responsive layout** — `bottom` resizable grid and `gitui` split panes adapt fluidly to width/height changes. citeturn4search0turn1search5

## Suggested reuse plan (minimal bespoke UI)
1) **Base skeleton**: start from `crates-tui` template structure (async runtime + app state machine); swap data layer with Helm’s runners. 
2) **Home/menus**: lift menu list widget from `gitui`, adapt pointer/description + key hints. 
3) **Run & Split viewports**: reuse `bottom` log panel + footer patterns; adopt kill-confirm modal from `gitui`. 
4) **Status graph**: embed `tui-tree-widget-table` with `xplr`-style selection highlighting; add summary bar badges styled like `kmon`. 
5) **Forms**: compose `tui-input` + `tui-textarea` with `spotify-tui` settings layout for scaffold/settings flows. 
6) **Theme**: start with `crates-tui` Base16 palettes; map Helm colors (`primary/accent/muted/etc.`) to its theme config and share across widgets. 
7) **Clipboard/resume**: reuse the clipboard pattern from `spotify-tui` (copy URL with graceful fallback) for resume chips and acceptance commands. 

## Notes & gaps to fill
- Spotatui is the actively maintained fork of spotify-tui; consider swapping the submodule if we need current Spotify auth patterns. citeturn0search3
- For inline editing without external `$EDITOR`, evaluate `edtui` for the Split draft step. citeturn0search11
