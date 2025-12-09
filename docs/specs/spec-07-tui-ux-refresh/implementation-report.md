# Implementation Report for spec-07-tui-ux-refresh — Unified TUI component system

- Mode: strict
- Max attempts: 2
- Attempts used: 1
- Final verifier status: ok

## Remaining tasks

{
  "remainingTasks": []
}

## Final worker output

SUMMARY
- Built a Glow-derived palette and a reusable Bubble Tea kit (`internal/tui/components`) containing TitleBar/PageShell, HelpBar, Flash/Modal, spinner & resume chips, text inputs/textarea skins, menu/spec list renderers, summary tables/bars, viewport cards, and bullet/FormField helpers so every screen shares the same primitives.
- Rewrote Home, Run (list/running/result), Status, Spec Split (intro/input/running/done), Scaffold (all wizard steps), and Settings views to render exclusively through the shared components while keeping the underlying update/keybinding logic intact.
- Run mode now streams attempt spinner lines, resume chips, flash banners, modals for unmet deps/kill confirmations, and log viewport cards; Status shows both summary bar/table plus styled table/graph panes; Scaffold/Spec Split/Settings use shared form fields, bullet lists, and help bars.
- Ported Ultraviolet primitives: shared layout rectangles (`ContentArea`, `SplitVertical/Horizontal`, `Top*/Bottom*Rect`), border variants (normal/rounded/thick/double/block/half-block/hidden/markdown/ascii), ANSI-aware styled string wrapper (wrap/truncate with wcwidth/grapheme tail), and centralized key normalization before Bubble Tea handling. Verified against `references/ultraviolet/{layout.go,border.go,styled.go,key_table.go}`.
- Added a direct `github.com/mattn/go-runewidth` dependency for width-aware help/tables and wired the run model to tick a Glow-styled spinner during executions.
- `make all` (deps, tidy, fmt, vet, lint, test, build, multi-platform release) completes successfully.

CHANGELOG
- `go.mod` – promoted `github.com/mattn/go-runewidth` to a direct dependency for component width calculations.
- `internal/tui/theme/theme.go` – replaced ad-hoc colors with Glow palette tokens, shared spacing constants, and new badge styles.
- `internal/tui/components/util.go` – helper for consistent viewport widths respecting Glow padding.
- `internal/tui/components/layout.go` – TitleBar, PageShell, and Modal primitives with Glow padding/colors.
- `internal/tui/components/help.go` – width-aware HelpBar rendering keys/labels with Glow styles.
- `internal/tui/components/feedback.go` – Flash banners, spinner factory/line renderer, and resume chip pill.
- `internal/tui/components/inputs.go` – styled text input and textarea factories applying Glow prompts/cursors/borders.
- `internal/tui/components/lists.go` – MenuList cursoring, SpecListItem renderer, and accent BulletList.
- `internal/tui/components/form.go` – FormFieldView for labeled value rows with focus/error messaging.
- `internal/tui/components/tables.go` – SummaryBar, SummaryTable, and Bubble-table styles aligned to the palette.
- `internal/tui/components/viewport.go` – ViewportCard border + status bar wrapper for logs/graphs with selectable border variants and styled-string wrapping.
- `internal/tui/components/layout2.go` – Ultraviolet rectangle helpers (`Split*Rect`, centering, edge rects) plus `ContentArea`/`ViewArea` for padding-aware sizing.
- `internal/tui/components/styled.go` – ANSI-aware fit/wrap/truncate helper using wcwidth/grapheme width and styled string width utility.
- `internal/tui/components/keys.go` – Shared key normalization (Ctrl+I vs Tab, Shift+Tab, Backspace/Delete, keypad modes) applied before pane logic.
- `internal/tui/home/home.go` – tracked window size and re-rendered the home menu via PageShell + MenuList + HelpBar.
- `internal/tui/run/model.go` – added Glow spinner state/ticks, preserved behavior, ensured resize/start hooks feed new components, and rewrapped logs via styled-string helper.
- `internal/tui/run/view.go` – replaced bespoke strings with SpecListItem, Flash, ResumeChip, SpinnerLine, ViewportCard, Modal, and HelpBar for every phase.
- `internal/tui/scaffold/model.go` – used componentized text inputs/spinner and passed terminal width to view helpers.
- `internal/tui/scaffold/views.go` – rewrote each wizard step with PageShell, MenuList, FormField, BulletList, SpinnerLine, and shared help bars.
- `internal/tui/settings/model.go` – adopted component text inputs/form fields and wrapped the page in PageShell.
- `internal/tui/specsplit/model.go` – swapped in Glow textarea/spinner, propagated width to viewport cards, and wrapped streamed logs using the styled-string helper.
- `internal/tui/specsplit/view.go` – reworked intro/input/running/done phases with Flash, ResumeChip, ViewportCard, SummaryTable, BulletList, and HelpBar.
- `internal/tui/status/model.go` – applied component table styles, padding-aware sizing helpers, and styled-string wrapping for graph viewport content.
- `internal/tui/status/view.go` – wrapped the view in PageShell, added summary bar+table, viewport card for graph mode, and shared help entries.
- `references/glow` (submodule) – gofumpt/goimports (via `make fmt`) touched tracked files; no intentional logic changes.
- `references/ultraviolet` (submodule) – added as upstream source-of-truth for layout/border/styled-string/key helpers referenced above.

TRACEABILITY
- **Shared component usage:** Every TUI view now imports `internal/tui/components` for rendering (home/run/specsplit/scaffold/status/settings files above) eliminating one-off lipgloss styles; spec list rows use `SpecListItem`, menus use `MenuList`, forms use `FormFieldView`, logs/graphs use `ViewportCard`, and hints run through `HelpBar`.
- **Glow palette/layout:** `internal/tui/theme/theme.go` encodes the Glow colors/padding and badge styles, while all components/panes consume those tokens, ensuring consistent colors, spacing, and selection highlights across screens.
- **Screen-specific primitives:** Run running/result phases gained SpinnerLine, ResumeChip, Modal, and log ViewportCards (`internal/tui/run/view.go`); Status combines `SummaryBar` + `SummaryTable` with component-styled tables/graphs (`internal/tui/status/view.go/model.go`); Spec Split and Scaffold each use Textarea/TextInput/FormField/BulletList flows (`internal/tui/specsplit/*.go`, `internal/tui/scaffold/*.go`); Settings relies on the same form primitives.
- **Behavior/hotkeys preserved:** Update loops in run/specsplit/scaffold/settings/status were left intact—only view code changed—and spinner ticks were added without altering existing key handling (model files above).
- **Acceptance command:** `make all` was executed successfully (see command log) covering deps, tidy, gofumpt/goimports, vet, golangci-lint, tests, build, and multi-platform release.

RUNBOOK
- **CLI flows**
  - Initialize or update scaffolding: `go run ./cmd/helm scaffold` (walk through the new componentized wizard).
  - Launch home menu: `go run ./cmd/helm` (opens the PageShell-based menu that routes to Run/Spec/Status/Quit).
  - Run specs: `go run ./cmd/helm run` (optional `--specs-root <path>` if not set in config); navigate list, hit `enter` to run, observe spinner/log cards, `q` to stop, `c` to copy resume.
  - Split specs directly: `go run ./cmd/helm spec [--file large_spec.txt]` (same UI as the breakdown command from home).
  - Inspect status: `go run ./cmd/helm status` (switch between table/graph with `tab`, adjust focus with `f`).
  - Edit settings: `go run ./cmd/helm settings` (navigate with arrows, `enter` on Save).
- **Acceptance command**
  - Ensure Go ≥1.25 and network access (Makefile auto-installs `gofumpt`, `goimports`, `golangci-lint` into `$GOBIN`). Run `make all` from the repo root to execute deps/tidy/fmt/vet/lint/test/build/release.

MANUAL SMOKE TEST
- `go run ./cmd/helm run` → verify the spec list uses the new badge/selection styling, toggle the runnable filter (`f`), start a spec, watch the SpinnerLine + log ViewportCard, copy the resume chip (`c`), and trigger the kill modal (`q` then `y/n`).
- `go run ./cmd/helm spec --file docs/specs/sample.md` → step through intro, paste/input view (Textarea styling), start splitting (spinner + resume chip), confirm flash messages/log viewport, and review the SummaryTable + warnings on completion.
- `go run ./cmd/helm scaffold` → check the intro PageShell, mode picker MenuList, acceptance command bullet list + input, specs root FormField with inline error, confirmation summary, running spinner, and completion bullet lists.
- `go run ./cmd/helm status` → observe the summary bar + summary table, focus line/help hints, styled Bubble table, switch to graph view and ensure the dependency tree renders inside a viewport card with proper help text.
- `go run ./cmd/helm settings` → navigate through the form fields, confirm focused rows highlight, inline instructions remain readable, and hitting `enter` on “Save” still persists validated settings.

OPEN ISSUES & RISKS
- Running `make fmt` (required by `make all`) reformats the `references/glow` submodule; if you need to avoid dirtying that reference, consider excluding it from fmt tooling or resetting the submodule manually afterward.
- `PageShell` honors Glow’s sizable top/bottom padding; on very short terminals this reduces viewport height, so further tuning or dynamic padding may be desirable.
- Help bars currently truncate to a single line with an ellipsis; extremely narrow terminals may still wrap, so future enhancements could include multi-line help support.
