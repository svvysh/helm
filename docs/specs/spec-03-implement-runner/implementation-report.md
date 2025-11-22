# Implementation Report for spec-03-implement-runner — implement-spec runner with worker/verifier loop

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
- Added a production-ready runner package that mirrors the Node reference: it renders the implement/review templates, shells out to `codex exec`, parses verifier output, and persists metadata/report updates per attempt.  
- Wired `helm run` to the new runner, including argument validation, `MAX_ATTEMPTS`/model environment overrides, and an optional `--mode` flag so the CLI drives any spec folder from Go instead of the deprecated script.  
- Captured success, failure, and multi-attempt flows in unit tests with a fake Codex executor so metadata status, notes, remaining tasks, and report contents stay correct.  
- Executed the strict acceptance suite via `make all`, which covered gofmt/goimports, golangci-lint, go vet, go test, and multi-platform builds.

CHANGELOG
- `cmd/helm/main.go:96` – replaced the stub `helm run` command with argument validation, env/flag handling, runner instantiation, and shared settings retrieval (`settingsFromContext`).  
- `internal/runner/runner.go:20` – added the runner implementation: command execution abstraction, spec resolution, template filling, worker/verifier loop, metadata/report writers, and helper utilities (status parsing, acceptance command formatting, note summarization).  
- `internal/runner/runner_test.go:16` – introduced fake Codex-backed tests that assert OK/missing flows, metadata/report side effects, fallback acceptance commands, and prompt propagation of remaining tasks.

TRACEABILITY
- Go CLI entrypoint now runs specs end-to-end: `newRunCmd` pulls settings, honors `MAX_ATTEMPTS`, `CODEX_MODEL_IMPL`, `CODEX_MODEL_VER`, and the new `--mode` override before calling the runner (`cmd/helm/main.go:96`).  
- The runner resolves spec folders, loads templates/metadata, renders prompts, and performs the worker→verifier loop with streaming Codex calls and strict `STATUS` parsing (`internal/runner/runner.go:74`, `internal/runner/runner.go:222`). This satisfies the requirement to mirror `implement-spec.mjs` in Go.  
- Metadata/status management and report generation match the spec: success/missing notes, ISO `lastRun`, remaining-task summaries, and implementation-report sections are handled in `updateMetadata`/`writeReport` (`internal/runner/runner.go:371`, `internal/runner/runner.go:425`).  
- Unit tests simulate dummy specs plus mock Codex behavior to confirm OK vs. missing verifier output updates `metadata.json`, `implementation-report.md`, and prompt inputs as expected, covering the checklist scenarios (`internal/runner/runner_test.go:16`, `internal/runner/runner_test.go:79`, `internal/runner/runner_test.go:132`).  
- Acceptance commands `make all` (which runs gofmt/goimports, golangci-lint, go vet, go test, go build, and release builds) completed successfully after the changes, demonstrating the strict-mode requirement (`2025-11-22` run captured in this attempt).

RUNBOOK
- Prereqs: Go 1.25+, the Codex CLI in `$PATH`, and a configured `docs/specs/.cli-settings.json`.  
- Running a spec: `go run ./cmd/helm run spec-03-implement-runner` (spec argument may be absolute, relative, or a bare `spec-XX-name`). Optional overrides: `MAX_ATTEMPTS`, `CODEX_MODEL_IMPL`, `CODEX_MODEL_VER`, and `--mode {strict|parallel}`. Output from both worker and verifier streams directly to your terminal.  
- Inspect results: review the spec’s `metadata.json` for updated status/notes and `implementation-report.md` for the summarized attempt output.  
- Acceptance suite: run `make all` from the repo root; it tidies modules, formats, vets, lints, tests, builds, and produces the platform release artifacts.

MANUAL SMOKE TEST
1. Create a lightweight Codex stub earlier in your `$PATH`, e.g. a `codex` shell script that echoes stdin for worker calls and prints `STATUS: ok` + `{"remainingTasks":[]}` for `--sandbox read-only`.  
2. From the repo root, run `go run ./cmd/helm run docs/specs/spec-00-foundation`. Confirm the worker prompt text streams to stdout and the command exits 0.  
3. Open `docs/specs/spec-00-foundation/metadata.json` and verify `status` flipped to `"done"`, `lastRun` is recent, and `notes` gained the summary line.  
4. Inspect `docs/specs/spec-00-foundation/implementation-report.md` to ensure it lists the attempt count, final status, remaining tasks JSON, and the echoed worker output. Remove or restore the real Codex binary afterward.

OPEN ISSUES & RISKS
- The runner depends on an external `codex` binary at runtime; without it, `helm run` fails immediately. Consider bundling a mock or adding clearer preflight checks.  
- `implementation-report.md` stores the full worker transcript each attempt; large outputs could bloat the file, so truncation or tailing logic might eventually be desirable.  
- Only `MAX_ATTEMPTS` can be overridden via env; exposing a dedicated CLI flag (and perhaps a dry-run mode) could make experimentation safer without exporting environment variables.
