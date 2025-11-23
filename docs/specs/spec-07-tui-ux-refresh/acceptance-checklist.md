# Acceptance Checklist — spec-07-tui-ux-refresh

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] `go run ./cmd/helm` opens the home menu wrapped in the new component shell (header + key help bar); quitting with `q` returns to the terminal without panic.
- [ ] Home menu items reuse the shared `MenuList` component (same cursor, spacing, and typography as spec list rows) and the hint bar matches other panes.
- [ ] Run pane list phase renders every spec row with the shared status badge + two-line meta layout; filter toggle/hint is shown via the common key-help bar; unmet-deps confirmation uses the standard modal component.
- [ ] Run pane running/result phases use the shared log viewer (status bar + scrollable viewport), resume chip, and flash/confirmation components; copy-to-clipboard messaging is consistent with other panes.
- [ ] Status pane uses the shared summary bar, table theme, graph viewport frame, and key-help bar; toggling views/focus does not change fonts/colors.
- [ ] Breakdown (`helm spec`) intro/input/running/done views use the shared shell, textarea/input styling, spinner line, summary table, resume chip, and warning list components.
- [ ] Scaffold wizard steps (intro, mode picker, commands, options, confirm, running, complete) all use the shared shell; inputs and spinners match the shared form and progress components.
- [ ] Settings form rows use the shared form field component (focused indicator + aligned value text); saving/canceling behaves as before.
- [ ] All reused components live under `internal/tui/components` (or theme) and each pane imports them instead of bespoke lipgloss/bubble styles.
