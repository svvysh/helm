# spec-03-implement-runner — Go worker/verifier loop and metadata updates

## Summary

Build a Go-based implementation runner (the `helm run` command) that performs the same worker → verifier loop as `docs/specs/implement-spec.mjs`. The Node script remains an internal reference for TUI authoring, but the shipped tooling in this repo must use the Go runner to execute acceptance commands and update spec metadata.

## Goals

- Implement a Go CLI entrypoint that runs the worker/verifier loop for a single spec directory, mirroring the behavior of `docs/specs/implement-spec.mjs`.
- Consume `acceptanceCommands` and other metadata from `metadata.json` to build prompts and drive execution.
- Integrate the existing templates (`implement.prompt-template.md`, `review.prompt-template.md`) exactly as the Node script does.
- Stream worker/verifier output to stdout/stderr and persist updates to `metadata.json` and `implementation-report.md`.
- Keep the runner generic so it works for any spec folder under the configured specs root.

## Non-Goals

- Do not ship or invoke the Node script at runtime; it is reference material only.
- No TUI changes yet; this is a CLI implementation.
- No parallel execution of multiple specs; one spec per process.

## Detailed Requirements

1. **Inputs & Environment**
   - Invoke via Go CLI, e.g.:

     ```sh
     helm run <spec-dir>
     ```

   - `<spec-dir>` accepts the same forms as the Node runner: a relative path (`docs/specs/spec-00-foundation`) or a bare spec name (`spec-00-foundation`, resolved under `docs/specs` by default).
   - Environment variables:
     - `MAX_ATTEMPTS` (optional, default: 2).
     - `CODEX_MODEL_IMPL` (optional) to override the worker model.
     - `CODEX_MODEL_VER` (optional) to override the verifier model.
   - Accept optional flags that mirror the Node behavior (e.g., `--mode`, `--dry-run` if you add one) but keep defaults identical.

2. **Spec Resolution**
   - Resolve the absolute path to the spec directory.
   - Load:
     - `SPEC.md` (spec body),
     - `metadata.json` (ID, name, acceptance commands, prior state),
     - `implement.prompt-template.md`,
     - `review.prompt-template.md`.
   - Derive:
     - `specID` from `metadata.id` or folder name if missing.
     - `specName` from `metadata.name` or the first `#` heading in `SPEC.md`.

3. **Main Loop (Go pseudocode)**

   ```go
   remaining := anyPreviousTasks
   for attempt := 1; attempt <= maxAttempts; attempt++ {
       workerOut, err := runWorker(ctx, remaining)
       if err != nil { return err }

       status, remaining, err := runVerifier(ctx, workerOut)
       if err != nil { return err }

       updateMetadata(status, remaining, workerOut)
       writeReport(status, remaining, attempt, maxAttempts, workerOut)

       if status == "ok" { return nil }
   }
   return fmt.Errorf("exhausted attempts without STATUS: ok")
   ```

4. **Worker Phase**
   - Fill `implement.prompt-template.md` with:
     - `{{SPEC_ID}}`, `{{SPEC_NAME}}`, `{{SPEC_BODY}}`.
     - `{{ACCEPTANCE_COMMANDS}}` rendered from `metadata.acceptanceCommands` as a bullet list.
     - `{{PREVIOUS_REMAINING_TASKS}}` as JSON (empty array/object when none).
     - `{{MODE}}` (default `"strict"` unless overridden).
   - Invoke the Codex CLI:

     ```sh
     codex exec --dangerously-bypass-approvals-and-sandbox --model "$CODEX_MODEL_IMPL" --stdin
     ```

     falling back to `codexModelRunImpl` from settings when the env var is absent.
   - Stream stdout/stderr live; keep a full copy of stdout for verifier input and reporting.

5. **Verifier Phase**
   - Fill `review.prompt-template.md` with the spec body, acceptance commands, the worker output, and mode.
   - Invoke:

     ```sh
     codex exec --sandbox read-only --model "$CODEX_MODEL_VER" --stdin
     ```

     with a fallback to `codexModelRunVer` when the env var is missing.
   - Parse verifier stdout strictly:
     - Line 1: `STATUS: ok` or `STATUS: missing`.
     - Line 2: JSON such as `{ "remainingTasks": [ ... ] }`.
   - Any deviation → exit non-zero.

6. **Metadata Updates**
   - On each verifier run:
     - `STATUS: ok` → `metadata.status = "done"`, update `metadata.lastRun` (ISO8601), append a short success note (e.g., last worker summary line).
     - `STATUS: missing` → `metadata.status = "in-progress"`, append a succinct summary of `remainingTasks` to `metadata.notes`.
   - Persist `metadata.json` after every attempt.

7. **Implementation Report**
   - Write or overwrite `implementation-report.md` under the spec directory containing:
     - Spec ID and name.
     - Mode and `maxAttempts` (from env/flags).
     - Attempts performed.
     - Final verifier `STATUS`.
     - Final `remainingTasks` JSON.
     - A tail or full copy of the final worker output sufficient for debugging.

8. **Exit Codes**
   - `STATUS: ok` within `MAX_ATTEMPTS` → exit 0.
   - Exhausted attempts without `STATUS: ok` → exit > 0.
   - System errors (missing files, invalid JSON, Codex CLI missing, bad verifier output) → print a clear error to stderr and exit > 0.

## Reference Go Shape (illustrative, not prescriptive)

```go
type Runner struct {
    SpecsRoot     string
    MaxAttempts   int
    Mode          string
    WorkerModel   string
    VerifierModel string
}

func (r Runner) Run(ctx context.Context, specPath string) error {
    spec, err := loadSpec(specPath)
    if err != nil { return err }

    remaining := spec.PreviousRemainingTasks()
    for attempt := 1; attempt <= r.MaxAttempts; attempt++ {
        workerOut, err := r.runWorker(ctx, spec, remaining)
        if err != nil { return err }

        status, remaining, err := r.runVerifier(ctx, spec, workerOut)
        if err != nil { return err }

        if err := r.updateMetadata(spec, status, remaining, workerOut); err != nil { return err }
        if err := r.writeReport(spec, attempt, status, remaining, workerOut); err != nil { return err }

        if status == "ok" {
            return nil
        }
    }
    return fmt.Errorf("exhausted %d attempts without STATUS: ok", r.MaxAttempts)
}
```

Mirror the control flow and prompt construction of `docs/specs/implement-spec.mjs`; only the language/runtime changes.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed (including any new tests around the runner and metadata helpers).
- With a dummy spec and a mockable/fake `codex` CLI:
  - `helm run docs/specs/spec-00-example` completes without crashing.
  - Verifier output `STATUS: ok` + empty `remainingTasks` sets `metadata.status` to `"done"`, updates `lastRun`, and writes `implementation-report.md` with the expected fields.
  - Verifier output `STATUS: missing` updates `metadata.status` to `"in-progress"`, appends a note summarizing `remainingTasks`, and captures the failure in `implementation-report.md`.
- Behavior matches `implement-spec.mjs`: same prompt content, parsing rules, exit codes, and metadata/report side effects.

## Implementation Notes

- Use `os/exec` with `exec.CommandContext` to call `codex exec`, streaming stdout/stderr while also capturing stdout for the verifier and reports (e.g., via `io.TeeReader`).
- Keep the runner self-contained in Go; do not shell out to the Node script.
- Tests should point to a temp specs root (e.g., `t.TempDir()/specs-test`) with its own `.cli-settings.json` so `docs/specs/` stays untouched.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Settings, metadata, and spec discovery
- spec-02-scaffold-command — scaffold command and initial specs layout
