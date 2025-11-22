# Acceptance Checklist — spec-07-tui-ux-refresh

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] Theme is sourced from `catppuccin/lipgloss` (dark by default) with a light/fallback option; no hard-coded color literals remain in views.
- [ ] Layout uses shared Page/Panel/Alert/KeyHints components (lipgloss-contrib) across Home, Run, Status, Scaffold, Spec Split, and Settings.
- [ ] Home view shows uniform cards with icon/title/subtitle; focused card uses primary border; key bar present.
- [ ] Run view uses two-column layout (status card + log viewport), Alert for unmet deps/kill confirm, progress/spinner themed; key bar present.
- [ ] Status view shows legend, focus panel, graph/table panel with themed badges; graph/table share the same key bar styling; ASCII fallback looks acceptable.
- [ ] Scaffold and Settings flows are implemented with `charmbracelet/huh` forms, including validation feedback and focused field styling.
- [ ] Spec Split intro/help rendered via glamour markdown; text area and log viewport share panel styling; key bar present.
- [ ] Key hint bar is visible and consistent in every pane.
- [ ] At 80-column terminal width, layouts collapse to single-column without clipping key hints or badges.
- [ ] Palette switch (dark/light) can be toggled via config/env and applies across all components without visual regressions.
