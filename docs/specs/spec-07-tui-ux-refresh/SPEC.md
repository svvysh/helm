# spec-07-tui-ux-refresh — Glow-patterned Helm TUI revamp

## Summary
Rebuild every Helm TUI screen to follow the exact layout, sizing, help, and rendering patterns used in Charmbracelet's Glow (submodule at `references/glow`, commit `ba37804fd57d82fdea1e2a4275884a76f27c1d8f`). The goal is to eliminate bespoke spacing/styling and reuse proven Bubble Tea conventions for background repainting, key help, pagination, and markdown/pager flows. The spec is prescriptive: use the cited Glow files as the canonical reference; deviations require justification in code review.

## Canonical references (read before coding)
- Program/bootstrap: `references/glow/ui/ui.go`
- Layout + stash list (list+pager split, pagination, help): `references/glow/ui/stash.go`, `stashhelp.go`, `keys.go`
- Pager (viewport, status bar, help toggle, high-perf rendering): `references/glow/ui/pager.go`
- Styles + adaptive colors: `references/glow/ui/styles.go`, top-level `references/glow/style.go`
- Markdown rendering: `references/glow/ui/markdown.go`, `references/glow/ui/pager.go` glamour usage

## Goals
- Replace Helm’s ad-hoc layout/string building with Glow-derived primitives: alt-screen program setup, list+pager split layouts, mini/full help bars, status messages, high-performance viewports, and pagination.
- Keep our Catppuccin palette, but map it onto Glow’s structural styling (borders, padding, adaptive color fallback) so screens look cohesive and repaint correctly on resize.
- Standardize key maps, help rendering, status/pill badges, and background fills across Home, Run, Status, Scaffold, Spec Split, Settings.
- Remove guesswork: for each screen, specify which Glow pattern to copy and how to adapt it to Helm data.

## Non-goals
- Changing business logic of run/status/scaffold/specsplit flows.
- Adding mouse interactions beyond what Glow already wires (only if we opt-in via config; see UI bootstrap).
- Replacing Bubble Tea stack.

## Global design system (derive from Glow, re-skin with Catppuccin)
1) **Program shell** (mirror `ui.NewProgram`):
   - Start TUI with `tea.WithAltScreen()` and optional `tea.WithMouseCellMotion()` based on a Helm setting (`settings.TUI.EnableMouse`).
   - Keep a `commonModel` equivalent holding width/height/settings for all sub-models.

2) **Theme/tokens** (keep Catppuccin, mirror Glow style structure):
   - Colors: retain `theme.Colors` but add `AdaptiveColor` fallback semantics like `styles.go` (light/dark fields) for when 256-color fallback is active.
   - Spacing: adopt Glow’s explicit padding constants for list/pager (e.g., `stashViewTopPadding=5`, `stashViewBottomPadding=3`, `stashViewHorizontalPadding=6`). Map them to our spacing scale (XS/SM/MD/LG/XL) and document the mapping.
   - Typography: keep current Typography but add helpers matching Glow (muted, subtle, accent, pill). Provide a one-liner helper to render mono pills similar to Glow’s status bar messages.
   - Borders: provide ASCII and rounded variants; default mirrors Glow (rounded for panels, thick for key/help bars).

3) **Layout primitives** (replace/extend existing components with Glow patterns):
   - `Page`: still clamps width, but must pad every line to target width and recenter like Glow’s stash view (background repaint safe). Reuse `lipgloss.PlaceHorizontal` as in `layout.go`, but size math mirrors Glow’s available-height calculation.
   - `Row/Column/Gutter`: support responsive stack; add Glow-like fixed row height for list items (3 lines including gap) to avoid jitter.
   - `Panel/Card`: retain, but styles must derive from Glow’s `styles.go` color placements (muted headers, accent borders on focus).
   - `HelpBar`: new component modeled on `stashhelp.go` with mini vs full help, column rendering, and width-aware truncation. Toggle with `?`; ESC/enter behavior same as Glow.
   - `StatusBar`: new component modeled on `pager.go` status bar—left note, center scroll pos, right help hint; message flash with timeout uses pagerStatusMessage pattern.
   - `Paginator`: use `bubbles/paginator` with dot style and active/inactive colors from Glow; expose helper to set `PerPage` based on available height.

4) **Input/display controls**:
   - **Viewport**: enable `HighPerformanceRendering` flag like Glow; after scroll commands call `viewport.Sync` when flag is on.
   - **Markdown**: use `glamour` renderer wired exactly like `renderWithGlamour` in `pager.go`, but feed Catppuccin-based style sheet (derive once in theme).
   - **Lists/Tables**: tables keep alternating row shading; lists should follow Glow’s stash item layout: title line + meta line + spacer; use `reflow/truncate` to avoid wrapping.
   - **Forms**: continue with `charmbracelet/huh`, but align focus/blur/error styles to Glow’s textinput and filter bar (prompt style, cursor style).

5) **Keys and help**:
   - Define a single key map file per view (similar to `ui/keys.go`) and reuse labels across help/status bar.
   - Mini help (one line) by default; full help on `?`; full help uses columns with padded keys/values as in `stashhelp.go` (renderHelp/miniHelpView/fullHelpView patterns).

6) **Status messages and flashes**:
   - Implement timed status flashes (success/error) using Glow’s `statusMessage` struct + timer reset logic; use same timeout (3s).
   - For long operations, keep spinner/progress but pipe flash messages through status bar instead of inline text where possible.

7) **Resizing rules**:
   - On `tea.WindowSizeMsg`, recompute available width/height per Glow: subtract help height, status bar height, and paddings before setting list/pager widths/heights.
   - Ensure backgrounds repaint: pad each rendered line to target width and, when centered, wrap with `lipgloss.PlaceHorizontal` like `Page.padHeight` and Glow’s stash view.

## View-by-view prescriptions (map Helm screens to Glow patterns)
1) **Home (launcher)**
   - Layout: follow Glow stash layout: header/logo area (use Helm logo/text), list of actions (Run, Spec Split, Status, Settings) rendered as fixed-height list rows with cursor highlight, and pagination if actions ever exceed view height (reuse paginator even if single page for consistent spacing).
   - Help: mini help shows nav + enter/quit; full help mirrors stash sections (navigation, actions, app). Toggle with `?`.
   - Status flash: reuse status bar for “repo initialized/needs scaffold” messages instead of inline alerts.

2) **Run**
   - Two-pane layout mirrors Glow’s stash (left list/right pager): left pane is run summary + controls, right pane is log viewport using pager patterns (status bar at bottom of pane, help toggle, line numbers optional). Use high-perf viewport.
   - Key map: `run` view should use same mini/full help renderer; keys include start/cancel/resume/copy.
   - Resume command pill uses Glow status message style (mint green on dark green) rendered via `StatusBar` message mode.

3) **Status**
   - Left column: list of specs with status badges; treat like stash list items (title + meta line with counts). Paginator if more than fits.
   - Right column: detail pager showing selected spec summary/log tail; uses Glow pager rendering (status bar, help, glamour for markdown). Status flashes (e.g., “reloaded statuses”) go through status bar.
   - Graph view: draw connectors with muted/primary colors, ASCII fallback; wrap inside panel with background fill matching Glow help bar background.

4) **Spec Split**
   - Intro/help: render with glamour markdown using Glow style.
   - Input screen: textarea adopts Glow textinput styles (prompt/caret colors) and respects fixed heights; errors shown as status message flash + alert component.
   - Running: left pane progress/spinner + status flash; right pane log viewport using pager model; mini help shows cancel/copy.
   - Done: results table restyled using the same table style as Glow’s pager status bar colors for header; resume pill via status bar.

5) **Scaffold**
   - Multi-step form built with huh; outer layout mirrors Glow pager-with-status: form body sits above status bar, help bar shows keys (`tab` next, `shift+tab` prev, `enter` submit, `esc` cancel).
   - Validation errors should surface both inline (huh) and via timed status message (error flavor).

6) **Settings**
   - Single-page huh form grouped into sections; background and padding match Glow pager view.
   - Use the same help system; status flash on save or cancel.

## Detailed implementation checklist
- [ ] Replace current key hint bar with Glow-style `HelpBar` (mini/full) component and wire to `?` toggle on every screen.
- [ ] Add `StatusBar` component with message timeout; use in Run (logs), Spec Split (running/done), Scaffold (form), Pager-like screens.
- [ ] Introduce shared paginator helper mirroring `newStashPaginator` (dot style, active/inactive colors); use in any scrolling list/table where rows can exceed viewport.
- [ ] Refactor list item rendering to fixed-height rows with title/meta/spacer like `stashItemView`; ensure truncation via `reflow/truncate` to avoid wrapping artifacts.
- [ ] Implement glamour renderer with Catppuccin theme; reuse for markdown intros, result tables (where applicable), and status detail pages.
- [ ] Set `viewport.HighPerformanceRendering` flag based on settings (default on); after any scroll command call `viewport.Sync` when enabled.
- [ ] Ensure every view pads to window width and fills background color (no unpainted columns after resize).
- [ ] Add ASCII fallback path for borders/icons identical to current theme, but validated in all new components.

## Acceptance criteria
- Screens (Home, Run, Status, Scaffold, Spec Split, Settings) visibly follow Glow patterns: mini/full help toggle, status bar with timed messages, fixed-height list rows, paginator dots, glamour-rendered markdown, high-performance viewport behavior.
- No view uses raw color literals or bespoke spacing; all styles come from theme tokens mapped to Glow-style adaptive colors.
- At 80×24 terminal: layouts collapse cleanly (list stacks above pager), help does not wrap, status bar and background fill the full width without artifacts.
- Status flashes time out after 3 seconds; toggling help recalculates viewport height without clipping content.
- `go test ./...` and `make all` pass; linting does not report dead code from removed bespoke components.
- Implementation cites Glow reference in code comments where patterns are mirrored (one-line comment linking file/function name), ensuring future maintainers know the source pattern.

## Depends on
- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Repo config, metadata, spec discovery
- spec-04-run-command — TUI shell and Run pane
- spec-05-spec-splitting-command — Spec splitting flow
- spec-06-status-command — Status view foundation
