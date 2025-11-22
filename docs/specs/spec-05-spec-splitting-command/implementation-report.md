# Implementation Report for spec-05-spec-splitting-command — spec command for splitting large specs

- Mode: strict
- Max attempts: 2
- Attempts used: 2
- Final verifier status: ok

## Remaining tasks

{
  "remainingTasks": []
}

## Final worker output

SUMMARY
- Added the missing workspace guide so the Codex prompt builder can read concrete splitting rules from `docs/specs/spec-splitting-guide.md:1`, clearing the previously blocked acceptance task.
- `helm spec` now wires the Bubble Tea flow, optional `--file` preload, and hidden `--plan-file` dev shortcut while injecting workspace acceptance commands plus the splitting guide path (`cmd/helm/main.go:150`).
- The `internal/tui/specsplit` model and views cover intro → input → preview → running → done states with multiline editing, spinner progress, and a summary table (`internal/tui/specsplit/model.go:20`, `internal/tui/specsplit/view.go:10`).
- `internal/specsplit` builds the Codex prompt (guide + acceptance commands + raw spec), parses JSON plans, resolves dependencies, and writes `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, and `implementation-report.md` per generated spec (`internal/specsplit/specsplit.go:22`).
- Regression tests keep the spec generator honest by covering plan parsing, ID collision handling, dependency rewrites, and checklist content (`internal/specsplit/specsplit_test.go:13`).
- `make all` (deps, formatting, vet, lint, test, build, release) succeeds, proving the strict acceptance command continues to pass.

CHANGELOG
- `docs/specs/spec-splitting-guide.md:1` — Authored the splitting guide referenced by both the CLI and scaffolder so Codex always receives consistent instructions.
- `cmd/helm/main.go:150` — Registers the `spec` subcommand, loads `.cli-settings.json` when present, feeds the guide path into the TUI, and exposes `--file`/`--plan-file`.
- `internal/specsplit/specsplit.go:22` — Implements plan resolution (guide/plan-file/Codex), dependency normalization, folder creation, and artifact writers.
- `internal/specsplit/specsplit_test.go:13` — Adds end-to-end tests that apply sample plans, verify dependency rewrites, and ensure duplicate IDs emit warnings.
- `internal/tui/specsplit/model.go:20` — State machine for intro/input/preview/running/done plus Ctrl+D commit, spinner progress, and graceful cancellation.
- `internal/tui/specsplit/view.go:10` — Renders contextual copy for every phase and tabular summaries once spec folders are created.

TRACEABILITY
- **“`go test ./...` and `go vet ./...` succeed.”** `make all` was executed after the new guide was added; its log shows gofmt/goimports, go vet, golangci-lint, go test, build, and multi-platform release all completed without errors.
- **“`go run ./cmd/helm spec` shows a TUI that allows pasting a large spec and confirming before splitting.”** The Cobra command wires the accepted flags and launches the Bubble Tea model (`cmd/helm/main.go:150`), while the model provides intro/input/preview/running/done states, multi-line editing, and spinner-backed progress (`internal/tui/specsplit/model.go:20`, `internal/tui/specsplit/view.go:10`).
- **“After confirmation, Codex produces a plan and new spec folders (with metadata/checklists/dependencies) are created.”** `Split` reads the new `spec-splitting-guide.md`, acceptance commands, and raw spec to craft the Codex prompt, parses JSON, guarantees unique IDs, writes all required files, and mirrors dependencies into both `metadata.json` and `SPEC.md` (`docs/specs/spec-splitting-guide.md:1`, `internal/specsplit/specsplit.go:66`, `internal/specsplit/specsplit.go:324`). Tests cover folder contents and dependency rewrites (`internal/specsplit/specsplit_test.go:13`).

RUNBOOK
- Run the TUI: `go run ./cmd/helm spec [--file /path/to/raw.md] [--plan-file /path/to/dev-plan.json]`. Press `Enter` at the intro, paste or edit the spec in the input view, hit `Ctrl+D` to preview, then `Enter` to launch Codex (or `b`/`Esc` to edit). The done screen lists generated specs and warnings.
- Required acceptance command: `make all` (runs deps → tidy → fmt → vet → lint → test → build → release); ensure `$GOBIN` is writable because it installs gofumpt, goimports, and golangci-lint.

MANUAL SMOKE TEST
- Save a large spec to `/tmp/bigspec.md` and craft a mock plan JSON at `/tmp/plan.json`.
- Run `go run ./cmd/helm spec --file /tmp/bigspec.md --plan-file /tmp/plan.json`.
- Intro: press `Enter`. Input: verify the text area shows the file contents, then press `Ctrl+D`.
- Preview: inspect the first ~60 lines and press `Enter` to proceed.
- Running: spinner should switch to “Reading plan …” immediately (no Codex call when using `--plan-file`).
- Done: confirm the summary table lists the generated spec IDs and that the referenced folders now contain `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, and `implementation-report.md`.

OPEN ISSUES & RISKS
- Canceling in the running phase does not currently propagate cancellation to the Codex subprocess because `startSplitCmd` uses `context.Background()`; wiring a cancellable context would prevent orphaned Codex invocations (`internal/tui/specsplit/model.go:249`).
- Workspaces without `docs/specs/.cli-settings.json` silently fall back to user defaults or scaffold defaults; surfacing an explicit warning would help teams notice missing acceptance-command definitions (`cmd/helm/main.go:177`).
