# spec-07-tui-ux-refresh — Unified TUI component system

## Summary
Rebuild the Helm TUI visuals around a small, reusable Bubble Tea component kit (colors, layout, controls) cloned from `references/glow` and Charm’s Ultraviolet primitives. The Ultraviolet repo is vendored as a submodule at `references/ultraviolet` and is the authoritative source for layout helpers, borders, styled strings, and key normalization. Every screen—home, Run, Status, Breakdown/spec split, Scaffold, and Settings—must render with the same primitives so no view has bespoke lipgloss styling. Logic, hotkeys, and data stay the same; only presentation and component reuse change.

## Goals
- Produce a complete inventory of everything we render across all Helm TUIs and map each element to a shared component.
- Build a component kit under `internal/tui/components` (plus palette in `internal/tui/theme`) by cloning styles/behaviors from `references/glow`, augmented with Ultraviolet layout/border/text/key primitives kept in sync with `references/ultraviolet`.
- Re-skin every TUI view to use those components, preserving all existing behavior and shortcuts.
- Ensure future screens can be assembled only from the shared components (no one-off lipgloss styles).

## Non-Goals
- Adding new commands, flows, or data. No new panes or extra fields.
- Changing keybindings, runner logic, spec discovery, or acceptance flows.
- Introducing mouse support or animations beyond what Bubble Tea already provides.

## Phase 1 — Inventory of current UI surfaces & required components
Every renderable element must be covered by a shared component.

- **Home menu**: page title, vertical menu list (Run/Breakdown/Status/Quit), pointer/cursor indicator, bottom hint bar.
- **Run pane — list phase**: spec list rows showing status badge, ID, name, unmet-deps/dep summary line, last-run line; filter toggle label; confirmation banner for unmet deps; hint bar.
- **Run pane — running phase**: title with spec ID/name; attempt line; resume chip with copy hint; flash message line; log viewport with scroll; kill confirmation banner; hint bar.
- **Run pane — result phase**: title; spec status + exit summary; remaining tasks bullet list; resume chip; flash line; log viewport; hint bar.
- **Status pane**: title with current view label; summary bar (TODO/IN PROGRESS/DONE/BLOCKED/FAILED counts); focus line + optional info message; table view (ID/Name/Status/Deps/Last Run); graph viewport; hint bar (tab/f/focus etc.).
- **Breakdown/spec split pane**:
  - Intro screen text block.
  - Input screen: multiline textarea, optional plan path note, inline error message, hint bar.
  - Running screen: spinner + status line, resume chip + copy hint, flash line, log viewport, hint bar.
  - Done screen (success): title, summary table (Spec ID/Name/Depends On), warnings list, resume chip, log tail, hint bar.
  - Done screen (error): error message, resume chip, log tail, hint bar.
- **Scaffold wizard**:
  - Intro copy block.
  - Mode picker list (strict/parallel) with selector cursor and hint bar.
  - Acceptance commands step: existing commands list, single-line input with prompt/cursor styling, ability to drop last command, hint bar.
  - Options step: specs root text input with inline error, focus highlight, hint bar.
  - Confirm step: summary list of chosen options, hint bar.
  - Running step: spinner + status line, hint bar.
  - Complete step: lists of created/skipped files, hint bar.
- **Settings form**: stacked rows for specs root input, mode toggle, default max attempts input, acceptance commands input, model/reasoning pairs (scaffold/run worker/run verifier/split), save row; focused row highlighting; hint line about navigation.
- **Cross-cutting elements**: status badges, title text, warning/errors, flash/ephemeral messages, modal confirmation panels, key-help bar, resume/copy chip, spinner line, table/graph frames, viewport scroll styling, consistent spacing/margins.

## Phase 2 — Component kit (clone from references/glow)
Implement under `internal/tui/components` (palette in `internal/tui/theme`). Use Glow as the visual source; keep names small and descriptive.

### Foundation tokens
- **Palette**: import/adapt colors from `references/glow/ui/styles.go` (`fuchsia`, `green`, `gray`, `yellowGreen`, status bar colors). Map to semantic tokens: primary, accent, muted, warning, success, surface, border.
- **Typography & spacing**: base mono font; Title = bold; Hint = muted foreground; padding/margins pulled from Glow stash constants (`stashViewHorizontalPadding`, `stashViewTopPadding`).

### Primitives
- **Badge**: pill styles for TODO / IN PROGRESS / DONE / BLOCKED / FAILED using Glow colors; keep existing status logic but style from palette.
- **TitleBar**: left-aligned bold title + optional view label; adopt Glow’s `logoStyle` padding/foreground.
- **HelpBar**: reusable key legend, cloned from `references/glow/ui/stashhelp.go` mini/ full help rendering; accepts pairs of key/label strings and fits to width.
- **Flash**: single-line info/warning banner (success, warning, danger) using Glow’s `statusMessage` styling from `stash.go`/`styles.go`.
- **SpinnerLine**: inline spinner + text using Glow spinner style (`stashSpinnerStyle`) and dot spinner.
- **TextInput**: single-line input with prompt/cursor colors from `stashInputPromptStyle` and `stashInputCursorStyle`.
- **Textarea**: multiline input styled to match TextInput (border/padding, same prompt colors) for spec split.
- **Modal**: centered/wide warning panel for confirmations (unmet deps, kill run) using `errorTitleStyle` background/foreground from `styles.go`.
- **ResumeChip**: pill with command text and copy hint, borrowing `statusBarHelpStyle`/`statusBarMessageStyle` from `ui/pager.go`.
- **ViewportCard**: bordered viewport with consistent padding and optional status bar at bottom (clone pager status bar structure from `ui/pager.go`). Used for logs, graph, and long content.
- **SummaryTable**: monospace table renderer (width-aware) reused by Status graph/table and spec split results; header underline like Glow’s pager status bar.
- **Layout helpers** (sourced from `references/ultraviolet/layout.go`): shared rectangle helpers (`SplitVertical/Horizontal`, `CenterRect`, `Top*/Bottom*Rect`) plus `ContentArea`/`ContentWidth` to replace ad-hoc width/height math in every view.
- **Border variants** (from `references/ultraviolet/border.go`): reusable lipgloss borders (`normal`, `rounded`, `thick`, `double`, `block`, `outer-half`, `inner-half`, `hidden`, `markdown`, `ascii`) selectable per card/modal/textarea—no per-view border definitions.
- **Styled text** (from `references/ultraviolet/styled.go`): ANSI-aware styled string helper (wrap vs truncate with tail) using wcwidth/grapheme width (`StyledString`/`printString`/`ReadStyle`/`ReadLink`) for logs, viewports, and help truncation.
- **Key normalization** (from `references/ultraviolet/key_table.go`): central mapping that reconciles Ctrl+I vs Tab, Backspace/Delete, Shift+Tab, keypad/app-mode sequences before routing to Bubble Tea models.
- **Terminal lifecycle**: raw → start → alt-screen → shutdown sequence from `references/ultraviolet/TUTORIAL.md`, ready for inline/alt-screen toggles.
- **Terminal lifecycle**: raw → start → alt-screen → shutdown sequence from `references/ultraviolet/TUTORIAL.md`, ready for inline/alt-screen toggles.

### Composites
- **PageShell**: wraps every view with consistent top/bottom padding plus TitleBar + body + HelpBar.
- **MenuList**: vertical list item renderer based on `stashItemView` highlighting rules (selected vs idle vs filtering). Used for Home and mode picker.
- **SpecListItem**: two-line row (badge + ID/Name, dependency/last-run summary) using MenuList selection style; accepts flags for unmet deps and runnable state.
- **SummaryBar**: row of badges with counts for each status (used in Status pane header).
- **TableView**: bubble Table style override that matches palette (header border, selected row colors from Glow tab styles).
- **GraphView**: `ViewportCard` showing dependency tree lines with same padding as TableView.
- **FormField**: label/value row with focus indicator and shared TextInput; used by Settings and Scaffold options.
- **BulletList**: simple list with accent bullets matching palette (for remaining tasks, warnings, created/skipped files).

## Phase 3 — Page rewrites (reuse only the new components)
Keep data flow and hotkeys unchanged; swap rendering to the kit above.

- **Home**: `PageShell(TitleBar + MenuList(items=Run/Breakdown/Status/Quit) + HelpBar("↑/↓", "move", "enter", "select", "q", "quit"))`.
- **Run list phase**: `PageShell` with `TitleBar("helm run")`, `SpecList` built from SpecListItem, filter label rendered via HelpBar, unmet-deps confirmation via Modal, flash via Flash. No bespoke lipgloss strings.
- **Run running phase**: `TitleBar("Running <id> — <name>")`, `SpinnerLine` for attempts, optional `ResumeChip`, `Flash`, `ViewportCard` for logs, `Modal` for kill confirm, `HelpBar` for scroll/quit/copy hints.
- **Run result phase**: `TitleBar("Run result — <id>")`, badge + exit summary line, `BulletList` for remaining tasks, `ResumeChip`, `Flash`, `ViewportCard` for logs, `HelpBar` for navigation.
- **Status pane**: `TitleBar("Status overview — <Table|Graph>")`, `SummaryBar`, focus/info line styled as Hint, switchable `TableView`/`GraphView` inside a `ViewportCard`, `HelpBar` with tab/f/enter/r/q bindings.
- **Breakdown/spec split**:
  - Intro uses `PageShell` with text body.
  - **Input** uses an **external editor flow cloned from Glow** (`references/glow/ui/editor.go`):
    - Key `e` (and prompt when pressing Enter with empty content) opens `$EDITOR` on a temp file preloaded with the current draft (or blank). Use a Tea cmd wrapper (`openEditor`) to launch `github.com/charmbracelet/x/editor` with a friendly title, return `editorFinishedMsg`, and read the edited file back into the draft.
    - The in-TUI view shows only a compact preview box (first ~10 lines) in a `ViewportCard`, plus errors via Flash and a HelpBar with `e edit`, `enter split`, `q quit`, `esc back`.
    - Enter starts the split only if `draft` is non-empty; Shift/Alt/Ctrl+Enter insert newlines by delegating to the editor (not the TUI widget).
  - **Running** uses `SpinnerLine`, `ResumeChip`, `ViewportCard` logs, `Flash`, and **Modal-based double-press esc/q confirmation** identical to Run: first press sets a 2s window; second press of the same key stops Codex (`esc`) or quits Helm (`q`). Help shows `esc×2 stop split`, `q×2 quit`.
  - **Done (success/error)** uses `TitleBar`, `SummaryTable`, `BulletList` for warnings, `ResumeChip`, `ViewportCard` log tail, `HelpBar`. Error case uses danger Flash. No textarea remnants remain.
- **Scaffold wizard**: Each step wrapped in `PageShell`. Intro/Confirm/Complete use text + BulletList. Mode picker uses `MenuList`. Commands step uses TextInput plus BulletList of existing commands and shared HelpBar. Options step uses FormField for specs root + inline error via Flash. Running step uses SpinnerLine. Complete step uses BulletList for created/skipped items.
- **Settings**: `PageShell` with stacked `FormField` rows (Specs root input, Mode toggle, Default attempts input, Acceptance commands input, model/reasoning pairs, Save). HelpBar explains navigation; Flash for validation errors.

## Acceptance Criteria (implementation-level)
- All Bubble Tea views import styling/helpers from `internal/tui/components` and `internal/tui/theme`; no duplicate lipgloss styles inside individual panes.
- Colors, padding, and selection highlights match the Glow-derived palette across every screen.
- Existing keybindings and behaviors continue to work (list navigation, filters, focus modes, resume copy, quit semantics).
- `make all` succeeds.
- Border styles are chosen from the shared variants; there are no bespoke lipgloss border structs in view code.
- All width/height calculations rely on `ContentArea`/`ContentWidth` + `Split*Rect` helpers; bespoke `width - padding` math is removed from panes.
- Viewport/log/help text uses the styled string helper (wcwidth/grapheme aware) for wrap/truncate; no manual rune/len trimming.
- Key handling goes through the shared normalization helper before view-specific logic.
- Ultraviolet submodule is present at `references/ultraviolet` and pinned; helpers in `internal/tui/components` match the corresponding upstream files noted above.

### Responsive sizing rules (required)
- Every pane derives its layout from `tea.WindowSizeMsg`; no hard-coded heights or width magic numbers remain.
- Available content height is computed by measuring the rendered chrome (title, flashes/modals, spinner/resume lines, status bars, help bar) with lipgloss, then subtracting chrome plus `theme.ViewTopPadding`/`theme.ViewBottomPadding` from the window height.
- Lists, tables, and viewports resize to the available height (with sensible minimums) while keeping footers/help on-screen at any terminal size.
- `PadToHeight` may be used only as a safeguard against repaint artifacts, never as the primary sizing mechanism.
- All existing TUI components have been refactored to follow these responsive rules.

## Depends on
- spec-00-foundation — module + CLI skeleton
- spec-01-config-metadata — settings/metadata/discovery
- spec-02-scaffold-command — scaffold flow
- spec-03-implement-runner — runner wiring
- spec-04-run-command — TUI shell + Run pane
- spec-05-spec-splitting-command — Breakdown/spec split pane
- spec-06-status-command — Status pane
