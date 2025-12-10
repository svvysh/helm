# Helm TUI Overview

A language-agnostic, comprehensive specification of Helm’s terminal user interface: the data it consumes, the entrypoints and flows, every screen and widget, navigation rules, and the behaviors required to recreate the product.

## Purpose & scope
- Helm is a cross-project spec runner. It discovers `spec-*` folders, orchestrates Codex worker/verifier runs, scaffolds new workspaces, splits large specs, and shows portfolio status.
- The multi-pane TUI opens when running bare `helm`; subcommands (`helm run`, `helm spec`, `helm status`, `helm scaffold`, `helm settings`) launch the corresponding single-flow TUIs directly.
- The full behavior described here must be implemented inside this codebase—there is no external pre-existing Helm logic to rely on. Implement discovery, runner, split, scaffold, settings, acceptance command resolution, and dependency computation as described. No mouse-only actions—keyboard-first with optional mouse scroll in viewports.

## Global data & configuration
- **Specs root:** Default `specs/`; resolved via `helm.config.json` (repo-scoped) plus user settings fallback. Resolved with `config.ResolveSpecsRoot(root, settings.SpecsRoot)`.
- **Repo config (`helm.config.json`):** `{ specsRoot, initialized }`, created after scaffold. Non-scaffold commands require this file and `initialized=true`.
- **Metadata:** Each `spec-*` dir must contain `SPEC.md`, `metadata.json`, optional `acceptance-checklist.md`. `metadata.json` fields: `id`, `name`, `status` (`todo|in-progress|done|blocked|failed`), `dependsOn[]`, `lastRun`, `notes`, `acceptanceCommands[]`.
- **Dependency state:** `specs.ComputeDependencyState` derives `CanRun` and `UnmetDeps` (deps not `done`). Used across Run and Status panes.
- **User settings (`~/.helm/settings.json` or `$HELM_CONFIG_DIR/settings.json`):**
  - `specsRoot`, `mode` (`strict|parallel`), `defaultMaxAttempts` (int)
  - Codex choices (model + reasoning) for scaffold/run worker/run verifier/split
  - `acceptanceCommands[]`
- **Acceptance commands resolution:** repo config > scaffold defaults.

## Shared UI system
- **Layout:** All screens use `PageShell` → title, body, help bar with global padding.
- **Theme:** Large and consistent palette (`primary`, `accent`, `muted`, `warning`, `success`, `surface`, `border`, `highlight`).
- **Components:** badges, menu list, help bar, flash banners (info/success/warning/danger), spinner line, resume chip, modal, form field, bullet list, summary bar/table, viewport card (border + optional footer), styled text wrapping/truncation, layout helpers (content area, split rects, center/top/bottom placements).
- **Input styling:** Text input/textarea share accent prompt + cursor and bordered states (focused vs blurred).
- **Key normalization:** Converts terminal variants (tab/shift+tab, ctrl+h, backspace2, delete, ctrl+m/enter) before state handling.
- **Responsive rules:** Every view sizes itself depending on available height and width. Lists/tables/viewports resize to fill remaining space and clamp overflow (with a min height). Content is padded to minimum height to avoid repaint artifacts.
- **Scrolling:** Models with mouse-wheel enabled; `ViewportCard` supplies consistent inner widths and frame height calculations.
- **Resume capture:** Any log line matching `session id: <uuid>` triggers a “Resume codex resume <id>” chip and a flash hint. `c` copies the command to clipboard (fallback: prints command in flash if clipboard unavailable).
- **Kill/quit confirmations:** Long-running phases (Run/Split) require double-press of `esc` (stop current process) or `q` (stop + quit Helm) within 2s; modal displayed while armed. `n` cancels when modal shown.

## Entry & routing flow
- Bare `helm`:
  1. Load user settings; resolve specs root (fallback logic above).
  2. Ensure repo config exists/initialized; if missing, auto-launch scaffold TUI; on completion, save `helm.config.json` and continue.
  3. Show Home menu (Run / Breakdown / Status / Quit). Selection loops until Quit.
- Subcommands:
  - `helm scaffold`: run scaffold wizard.
  - `helm settings`: open settings form, save to global config.
  - `helm run`: jump straight to Run pane.
  - `helm spec [-f file] [--plan-file path]`: jump to Split flow, optional preload text from file, optional dev plan file.
  - `helm status`: jump to Status pane.

## Screen specifications
### Home
- Menu items: Run specs / Breakdown specs / Status overview / Quit.
- Navigation: `↑/↓` move, `enter`/`space` select, `q`/`esc`/`ctrl+c` cancel (returns ErrCanceled to caller). Alt-screen enabled.
- Layout: MenuList with pointer; help bar (`↑/↓ move`, `enter select`, `q quit`).

### Run (helm run)
Phases: list → running → result. Alt-screen + mouse scroll enabled.

1) **List phase**
- Source: discovered spec folders (sorted). Rows show badge (status + unmet deps awareness), `ID — Name`, summary line (“Unmet deps: …” | “No dependencies” | “Dependencies satisfied”), last-run line (“Last run: <ts> (status <STATUS>)” or “never”).
- Controls: `↑/↓/pgup/pgdown/j/k` navigate; `f` toggles filter runnable-only (keeps selection if possible); `enter` starts run; `q/ctrl+c` quit all; `esc` closes unmet-deps modal or quits; unmet deps modal accepts `y` run anyway, `n/esc` cancel.
- Run eligibility: only specs not `done` or `in-progress` and `CanRun=true` unless user confirms unmet deps. `confirmUnmet` modal lists missing deps.
- Layout chrome: title “helm run”, optional unmet-deps modal, optional flash, help bar with filter label.

2) **Running phase**
- Header: spinner line with stage/attempt text (from parsed log markers `Attempt X of Y` or `Stage: implementing|verifying`; falls back to “Streaming Codex logs…”). Resume chip when session id captured. Optional flash.
- Log area: bordered viewport (preformatted, no wrap) with status footer “Scroll with ↑/↓, PgUp/PgDn or mouse”. Mouse wheel active; preserves scroll position unless at bottom.
- Kill confirm modal appears after first `esc`/`q`; second press within 2s stops runner (placeholder kill) and quits if `q`.
- Help: `↑/↓ PgUp/PgDn` scroll, `mouse` scroll, `c` copy resume, `esc×2` stop run, `q×2` quit.

3) **Result phase**
- Content: status line (“Spec status: <STATUS>”), flash for runner exit (success/danger), bullet list of remaining tasks (parsed from `implementation-report.md` “Remaining Tasks” JSON), resume chip, optional info flash.
- Log viewport with footer “Scroll… — enter to return”.
- Controls: `enter`/`r` back to list, `c` copy resume, `q` quits all, `esc/ctrl+c` exit to caller.
- On finish, specs are rediscovered to refresh badges/deps; preserves selection when possible.

### Breakdown / Spec Split (helm spec)
Phases: intro → input → running → done. Alt-screen + mouse scroll enabled.

1) **Intro**
- Text: explains purpose (paste large spec; streams Codex progress). Keys: `enter` begin, `q/esc` quit.

2) **Input**
- Instruction line; if `--plan-file` set, shows dev-note. Error flash shown inline.
- Draft preview card: first ~10 lines shown in a `ViewportCard` with footer summarizing line count or “(empty)”.
- Controls: `e` opens `$EDITOR` on temp file preloaded with draft (title “Helm Spec Split”, cursor at end). Plain `enter` starts split only if draft non-empty; `enter` with empty draft or Alt/Shift/Ctrl+Enter opens editor instead. `q` quits, `esc` returns to intro.

3) **Running**
- Spinner line (“Splitting via Codex…” or “Reading plan from <path>”). Resume chip + flash support. Log viewport (preformatted) with footer about scrolling. Kill-confirm modal on `esc`/`q` double-press. Help mirrors Run phase.
- Backend: starts `specsplit.Split` with guide path (`specs-breakdown-guide.md`), acceptance commands, Codex model from settings, optional plan path; streams stdout/stderr tagged lines.

4) **Done**
- **Success:** summary table (ID/Name/Depends on), resume chip if session id captured, info flash, warnings bullet list, recent-log tail card. Help: `enter/q/esc` exit, `r` jump to Run TUI, `n` start another split (back to input).
- **Failure:** danger flash with error, optional resume chip/flash/log tail. Help: `enter/q/esc` exit, `n` retry, `r` jump to Run.
- If no result produced: message “No specs were created.” with exit help.

### Status Overview (helm status)
- Purpose: browse all specs, dependency graph, and counts; filter views.
- Data: `specs.DiscoverSpecs` + dependency computation. Entries carry ID, Name, DependsOn, Dependents (computed), badge, status category, block reason, last-run display.
- Focus modes: `All` (default), `Runnable` (CanRun), `Subtree` (current selection + its dependents). Cycle with `f` (keeps selection/preserves target). `enter` switches to Subtree rooted at selected row.
- Summary: badge bar + 2-column summary table of counts. Focus/selection/info lines shown in hint style.
- Main content: `ViewportCard` containing ASCII dependency graph; selection highlighted with “▶”. Footer shows spec count. Graph updates when selection or focus changes.
- Detail panel: summary table (Field/Value) for selected spec (ID, Name, Status badge, Last run, Depends on, Dependents) plus warning flash when unmet deps.
- Controls: `↑/↓` move, `enter` focus subtree, `f` cycle focus modes, `r` reload filesystem + recompute state, `q` quit all, `esc/ctrl+c` exit to caller. Alt-screen enabled.
- Layout: table rows are not shown in view output (table model kept for selection); graph acts as primary visual.

### Scaffold Wizard (helm scaffold)
Steps: intro → mode picker → acceptance commands → options → confirm → running → complete. Alt-screen enabled.

- **Intro:** overview text; `enter` continue; `esc/q` quit.
- **Mode picker:** menu of `strict` vs `parallel` with descriptions. `↑/↓/tab` move, `enter` select, `esc` back.
- **Acceptance commands:** bullet list of existing commands; single-line input (prompt `↪`) for a new command. `enter` appends non-empty; blank `enter` moves forward; `ctrl+w` drops last command; `esc` back. Description text clarifies sequential behavior.
- **Options:** field for Specs root (prompt `↪`, placeholder default). Focus index 0 keeps input focused; validation requires non-empty root. `enter` validates and advances; `esc` back.
- **Confirm:** summary of mode, specs root, acceptance commands (bullet list or “(none)”). `enter` runs scaffold (starts spinner); `esc` back.
- **Running:** spinner line “Creating workspace... (Esc/q to cancel)”; `esc/q` cancels (sets canceled + quits program).
- **Complete:** shows specs root, created files (bullet list), skipped files. Danger flash on errors. `enter` exits; `q` quits.
- Outputs: returns `innerscaffold.Result{SpecsRoot, Created[], Skipped[]}`; caller persists `helm.config.json` when run from root command.

### Settings (helm settings)
- Form fields stacked vertically (each uses shared FormField rendering):
  1) Specs root (text input)
  2) Mode (left/right toggles strict|parallel)
  3) Default max attempts (text input, must parse int >0)
  4) Acceptance commands (comma-separated text input)
  5) Codex model & reasoning for: scaffold, run worker, run verifier, split (each two rows)
  6) Save row (enter commits)
- Navigation: `↑/↓` move focus; `←/→` cycle options for toggle/model/reasoning fields; text inputs active when focused. `enter` on Save persists settings (with validation), writes to user config, and exits. `esc/ctrl+c` cancels without saving.
- Validation feedback: invalid max attempts shown via placeholder error text; acceptance commands split on commas and trimmed.

## Logs & resumability
- Both Run and Split phases stream stdout/stderr lines tagged in-view (prefix `stdout:` / `stderr:` for Run; combined for Split). Stored up to 2000 entries; oldest dropped beyond limit. Viewport preserves scroll when not at bottom.
- Session ID detection via regex `^session id:\s*([a-f0-9-]{36})$` (case-insensitive) on any line.
- Resume commands follow format `codex resume <uuid>`.

## Error handling & quitting rules
- Global quit keys: `ctrl+c` or `q` in most screens (Run list uses ErrQuitAll; Status uses ErrQuitAll; Split uses ErrQuitAll; Scaffold/Settings return canceled flags). `esc` generally backs up one step; in list phases it quits.
- Long-running cancellation: double `esc`/`q` as noted; timers auto-clear confirmation after 2s.
- Validation errors surface as inline flash or placeholder text; flows block advancement until resolved (e.g., empty split draft, empty specs root, invalid max attempts).

## Rendering details by component
- **Badges:** TODO (border background), IN PROGRESS (accent), DONE (success), BLOCKED (muted), FAILED (warning); bold pill padding.
- **Menus:** pointer indicates selection; descriptions muted and indented.
- **Help bar:** key in highlight color, label muted, separated; truncated to fit content width with ellipsis.
- **Flash banners:** single-line, padded, bold; severity colors (success=green, warning=yellow-green, danger=red, info=accent on surface).
- **Spinner line:** dot spinner in accent followed by body text.
- **Modal:** rounded border by default; warning header background; body padded on surface background; width matches content width with padding.
- **Viewport card:** bordered block with padding; optional status footer; wraps or truncates ANSI-aware text according to `NoWrap`/`Preformatted` flags; inner width helpers exposed for viewport sizing.
- **Summary table:** monospace columns auto-sized to widest content; header underline divider; used for split results and status details.
- **Summary bar:** row of badges with counts in status order (TODO, IN PROGRESS, DONE, BLOCKED, FAILED).
- **Form fields:** label line prefixed with cursor when focused; description muted; errors in warning style.

## Dependencies & external tools
- Alt-screen programs (where applicable).
- Clipboard support (fall back to showing command if unavailable).

## Non-UI behavioral notes
- Run phase runner wraps `runner.Run` with options: repo root, specs root, mode, max attempts, worker/verifier model choices, acceptance commands; it is not cancelable mid-run (kill confirmation only exits UI).
- Split phase uses `specsplit.Split`, optionally reads a provided JSON plan file instead of Codex.
- Specs discovery requires `spec-*` folder names; missing `SPEC.md` or `metadata.json` fails the flow.
- Acceptance commands and max attempts can be overridden via `MAX_ATTEMPTS` env when running specs.

## Accessibility & resiliency expectations
- All screens must render legibly at narrow widths: minimum content width enforced at 24 cols; viewport heights clamped to avoid pushing help bars off-screen.
- No color-only cues: status badges include text labels; warnings/dangers accompanied by text.
- Layout recalculates on every window size change.

## End-to-end user journeys (reference)
1) **New repo initialization:** Run `helm` → auto-scaffold (choose mode, commands, specs root) → `helm.config.json` saved → Home menu → proceed to Run/Breakdown/Status.
2) **Executing a spec:** From Home select Run → choose spec (optionally filter runnable) → confirm unmet deps if shown → watch spinner/logs; copy resume; on completion review status/remaining tasks → enter to return.
3) **Splitting a large spec:** From Home select Breakdown or run `helm spec` → open editor, paste spec → enter to start → monitor logs/resume → view generated spec table and warnings → optionally jump directly to Run (`r`).
4) **Inspecting readiness:** From Home select Status → view counts and dependency graph → focus runnable-only or a subtree → reload after external changes (`r`).
5) **Adjusting settings:** Run `helm settings` → edit specs root/mode/attempts/commands/model choices → enter on Save; `esc` to cancel without persistence.
6) **Scaffolding later:** Run `helm scaffold` to regenerate files (e.g., new repo) or reset by deleting `helm.config.json` then re-run `helm`.
