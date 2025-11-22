# spec-03-implement-runner — Go worker/verifier loop and metadata updates

## Summary

Build the Go-based runner that performs the worker → verifier loop for a single spec directory. The runner will be invoked from the Run pane of the TUI (spec-04). A thin headless flag on `helm run` may reuse the same code, but the default `helm run` behavior is to open the TUI.

## Goals

- Implement a reusable Go runner that mirrors `docs/specs/implement-spec.mjs`.
- Consume `acceptanceCommands` and metadata to build prompts and drive execution.
- Stream worker/verifier output and persist updates to `metadata.json` and `implementation-report.md`.
- Expose a headless entrypoint (e.g., `helm run --exec <spec>`) for automation without altering the TUI-first default.

## Non-Goals

- No Bubble Tea code here (handled in spec-04).
- No multi-spec orchestration; one spec per process.
- No global settings; use repo-local `helm.config.json` defaults.

## Detailed Requirements

1. **Inputs & Environment**
   - Export a `Runner` type (or similar) that can be called from the TUI with a fully resolved spec path and repo config.
   - Provide a headless CLI path such as:

     ```sh
     helm run --exec <spec-dir-or-id>
     ```

     When `--exec` (or equivalent flag) is absent, the command should delegate to the TUI (spec-04) and not run headless.
   - Environment variables:
     - `MAX_ATTEMPTS` (optional, default: `RepoConfig.DefaultMaxAttempts` or 2).
     - `CODEX_MODEL_IMPL` / `CODEX_MODEL_VER` override the repo config model choices.

2. **Spec Resolution**
   - Resolve the absolute path to the spec directory using `RepoConfig.SpecsRoot` when given a bare ID.
   - Load `SPEC.md`, `metadata.json`, `implement.prompt-template.md`, and `review.prompt-template.md` from the specs root.
   - Derive `specID` and `specName` from metadata or headings as before.

3. **Worker/Verifier Loop**
   - Preserve the control flow and parsing rules from the prior version of this spec (STATUS lines, remainingTasks JSON, attempt loop capped by `maxAttempts`).
   - Update metadata and write `implementation-report.md` after each verifier run.
   - Exit codes: 0 on `STATUS: ok` within attempts; non-zero otherwise or on parse/system errors.

4. **Metadata Updates**
   - `STATUS: ok` → `status="done"`, update `lastRun`, append a succinct success note.
   - `STATUS: missing` → `status="in-progress"`, append a note summarizing remaining tasks.
   - Persist `metadata.json` after every verifier run.

5. **Headless CLI Flag**
   - Implement the `--exec` (or similarly named) flag on `helm run` to call the runner directly for automation/CI.
   - When the flag is absent, return control to the TUI entry defined in spec-04 (i.e., do not run the loop here).

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed (including runner tests).
- With a dummy spec in a temp `specs` root and a fake `codex` binary:
  - `helm run --exec spec-00-example` completes without crashing.
  - `STATUS: ok` sets metadata to `done`, updates `lastRun`, and writes a report.
  - `STATUS: missing` sets metadata to `in-progress` and captures remaining tasks in both notes and the report.
- Running `helm run` **without** `--exec` delegates to the TUI (spec-04) instead of running headless.

## Implementation Notes

- Use `os/exec` with streaming stdout/stderr and teeing stdout for verifier input.
- Keep the runner self-contained in Go; the Node script remains reference material only.
- Tests should write into a temp specs root to keep the tracked `docs/specs/` tree clean.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Repo config, metadata, and spec discovery
- spec-02-scaffold-command — Scaffold flow inside the TUI
