# spec-07-tui-ux-refresh — Modern, cohesive TUI experience

## Summary

Refresh the Helm TUI with a professional, minimalist visual system using existing Bubble Tea ecosystem libraries (no hand-rolled styling). Standardize theme, layout, components, and help cues across all panes (home, Run, Status, Scaffold, Spec Split, Settings) so the interface looks intentional and is easy to scan.

## Goals

- Adopt a curated theme (Catppuccin or similar) and apply it consistently via reusable components.
- Use off-the-shelf libraries (lipgloss-contrib, catppuccin/lipgloss, charmbracelet/huh, glamour, bubbles/table/list/progress/spinner) instead of custom layout/styling.
- Unify layout primitives (page, cards/panels, key-hint bar) and spacing across views with responsive behavior for narrow terminals.
- Improve readability: clear headers/subtitles, consistent badges, borders, and key hints; markdown-rendered copy for intros/help.
- Make forms (Scaffold, Settings) polished and validated via huh forms; minimize bespoke textinput wiring.

## Non-Goals

- Replacing Bubble Tea with a GUI/web framework.
- Changing business logic of run/status/scaffold/specsplit flows beyond UI/UX.
- Adding mouse support.

## Detailed Requirements

1) **Theme & Tokens**
- Use `catppuccin/lipgloss` (Mocha palette by default). Provide a light variant toggle in config and a low-color (256) fallback.
- Define single-source tokens: colors (bg/surface/overlay/muted/primary/success/warn/error), radii, border styles, spacing scale, and typography weights.
- Badge styles derive from tokens; no ad-hoc color codes in views.

2) **Layout Primitives (lipgloss-contrib)**
- Introduce flex-based `Page`, `Row`, `Column`, `Gutter` helpers for consistent width (clamp ~92–110 cols) and padding.
- Components:
  - `Panel`/`Card` with title, optional subtitle, body.
  - `KeyHints` bar rendered uniformly at bottom of each view.
  - `Alert` banner (info/warn/error/success) with icon fallback (`!/?/✓`).
  - `StatusBadge` using theme tokens.
- All views must compose these primitives; no manual string builders for layout.

3) **Reusable Libraries**
- **Forms:** Replace scaffold + settings flows with `charmbracelet/huh` forms (inputs, selects, multiselect, confirm). Validation messages come from huh; focus styling uses theme tokens.
- **Markdown copy:** Render intros/help/summaries with `glamour` using the Catppuccin style for headings, lists, code blocks.
- **Lists/Tables:** Keep `bubbles/list` and `bubbles/table` but restyle via theme tokens; add alternating row shading and focused border.
- **Progress/Spinner:** Use `bubbles/progress` with a gradient (primary→success) and themed spinner for long operations.

4) **View-Specific Requirements**
- **Home:** Card grid or vertical stack with icon, title, subtitle; consistent key bar. Focused card shows primary border.
- **Run:** Two-column layout (status card + log viewport). Key bar shows run controls. Unmet-deps warning uses Alert. Resume command rendered as pill.
- **Status:** Legend panel, focus panel, and graph/table panel in a consistent page. Table rows use themed badges; graph connectors use muted/primary colors with ASCII fallback.
- **Scaffold:** Multi-step huh form (mode, acceptance commands list editor, specs root) with progress indicator and summary/confirm panel.
- **Spec Split:** Markdown intro; text area styled with theme border; running view shows progress + log viewport; done view renders summary table with themed table component.
- **Settings:** Single huh form with grouped sections (Paths, Defaults, Models). Inline validation on save.

5) **Responsiveness & Accessibility**
- Support 80-column terminals by collapsing two-column layouts to single-column stack; never truncate key hints.
- Ensure contrast ratio (primary/muted on surface) meets WCAG-ish thresholds; provide ASCII fallback for icons/borders when Nerd Fonts unavailable.

## Acceptance Criteria

- `make all` passes.
- Home, Run, Status, Scaffold, Spec Split, and Settings panes render using the new theme tokens and shared components—no view contains raw color codes or ad-hoc borders.
- Scaffold and Settings flows are implemented with `charmbracelet/huh` forms, including validation and focus styling.
- Markdown rendering (glamour) is used for multi-paragraph copy (intros/help) with the theme style.
- Key hint bar appears and is consistent in every pane; warnings/errors use the Alert component.
- Narrow-width test: at 80 cols, layouts collapse to readable single-column stacks without clipping key hints or badges.
- Palette switch (dark default, light fallback) is selectable via config or env and applies across all components.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Repo config, metadata, spec discovery
- spec-04-run-command — TUI shell and Run pane
- spec-05-spec-splitting-command — Spec splitting flow
- spec-06-status-command — Status view foundation
