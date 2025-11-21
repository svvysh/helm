# spec-03-implement-runner — Codex worker/verifier loop and metadata updates

## Summary

Complete the implementation of `implement-spec.mjs`, the Node-based runner that orchestrates a worker/verifier loop via the Codex CLI. The runner must read templates, build prompts, run worker and verifier phases, and update per-spec metadata and reports.

## Goals

- Implement the full worker → verifier loop for a single spec directory.
- Integrate with `implement.prompt-template.md` and `review.prompt-template.md`.
- Call the Codex CLI with appropriate flags for worker and verifier.
- Stream logs to stdout/stderr for human and TUI consumption.
- Update `metadata.json` and `implementation-report.md` according to the product spec.

## Non-Goals

- No changes to the Go TUI yet.
- No sophisticated error recovery beyond clear failures and exit codes.
- No parallel execution of multiple specs; one spec per process.

## Detailed Requirements

1. **Inputs & Environment**
   - The runner is invoked as:

     ```sh
     node docs/specs/implement-spec.mjs <spec-dir>
     ```

   - `<spec-dir>` may be:
     - A relative path like `docs/specs/spec-00-foundation`, or
     - A bare spec name like `spec-00-foundation` (which should be resolved under `docs/specs` by default).
   - Environment variables:
     - `MAX_ATTEMPTS` (optional, default: 2).
     - `CODEX_MODEL_IMPL` (optional).
     - `CODEX_MODEL_VER` (optional).

2. **Spec Resolution**
   - Resolve the absolute path to the spec directory.
   - Load:
     - `SPEC.md` (spec body),
     - `metadata.json` (to get ID, name, acceptance commands),
     - `implement.prompt-template.md`,
     - `review.prompt-template.md`.
   - Derive:
     - `specID` from `metadata.id` or the folder name if missing.
     - `specName` from `metadata.name` or from the first `#` heading in `SPEC.md`.

3. **Main Loop**
   - Pseudocode:

     ```js
     let remainingTasks = [];
     for (let attempt = 1; attempt <= maxAttempts; attempt++) {
       // Build and send worker prompt
       // Build and send verifier prompt
       // Parse STATUS and remainingTasks
       // Update metadata and report incrementally
       // If STATUS: ok, break and exit 0
     }
     // If we exhausted attempts without STATUS: ok, exit non-zero
     ```

4. **Worker Phase**
   - Fill `implement.prompt-template.md` with:
     - `{{SPEC_ID}}`, `{{SPEC_NAME}}`, `{{SPEC_BODY}}`,
     - `{{ACCEPTANCE_COMMANDS}}` (bullet list),
     - `{{PREVIOUS_REMAINING_TASKS}}` (JSON array or object),
     - `{{MODE}}` (string from settings or default `"strict"`).
   - Call:

     ```sh
     codex exec --dangerously-bypass-approvals-and-sandbox --model "$CODEX_MODEL_IMPL" --stdin
     ```

     - If `CODEX_MODEL_IMPL` is not set, fall back to `codexModelRunImpl` from `.cli-settings.json` or a hard-coded default.
   - Stream worker stdout and stderr to the console.
   - Capture the full worker output as a string.

5. **Verifier Phase**
   - Fill `review.prompt-template.md` with:
     - Spec body and checklist.
     - Acceptance commands.
     - Implementation report from the worker (`workerOutput`).
     - Mode.
   - Call:

     ```sh
     codex exec --sandbox read-only --model "$CODEX_MODEL_VER" --stdin
     ```

     - If `CODEX_MODEL_VER` is not set, fall back to `codexModelRunVer` from settings.
   - Stream verifier stdout and stderr to the console.
   - Parse the first two lines of verifier stdout:
     - Line 1: `STATUS: ok` or `STATUS: missing`.
     - Line 2: JSON like `{"remainingTasks":[... ]}`.
   - Treat any deviation from this format as a failure and exit non-zero.

6. **Metadata Updates**
   - On each verifier run:
     - If `STATUS: ok`:
       - Set `metadata.status = "done"`.
       - Set `metadata.lastRun` to the current ISO8601 timestamp.
       - Append a short success summary to `metadata.notes` (e.g., last worker summary line).
     - If `STATUS: missing`:
       - Set `metadata.status = "in-progress"`.
       - Append a succinct summary of `remainingTasks` to `metadata.notes`.
   - Persist changes to `metadata.json` after each attempt.

7. **Implementation Report**
   - Write or overwrite `implementation-report.md` under the spec directory, containing:
     - Spec ID and name.
     - Mode and max attempts.
     - Number of attempts actually performed.
     - Final verifier `STATUS`.
     - Final `remainingTasks` JSON.
     - Tail or full text of the final worker output (enough to understand what was done).

8. **Exit Codes**
   - If `STATUS: ok` is reached within `MAX_ATTEMPTS`:
     - Exit with code 0.
   - If all attempts are exhausted without `STATUS: ok`:
     - Exit with code > 0.
   - If any system-level error occurs (missing files, invalid JSON, missing Codex CLI, etc.):
     - Emit a clear error message to stderr.
     - Exit with code > 0.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` still succeed (including any new tests you add for metadata helper usage, if applicable).
- Using a simple dummy spec and a mockable or fake `codex` CLI:
  - The runner can be invoked as `node docs/specs/implement-spec.mjs docs/specs/spec-00-example` without crashing.
  - If the verifier prints `STATUS: ok` and an empty `remainingTasks` list, the runner:
    - Sets `metadata.status` to `"done"`,
    - Updates `metadata.lastRun`,
    - Writes an `implementation-report.md` with the expected fields.
  - If the verifier prints `STATUS: missing` with a few remaining tasks:
    - `metadata.status` becomes `"in-progress"`,
    - `metadata.notes` includes a summary of remaining tasks,
    - `implementation-report.md` captures the failure.

## Implementation Notes

- Prefer async/await with `fs/promises` for filesystem IO.
- Keep the script self-contained; avoid external dependencies beyond Node and the Codex CLI.
- You may add a `--dry-run` flag for development/testing if helpful, but it is not required by this spec.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
- spec-01-config-metadata — Settings, metadata, and spec discovery
- spec-02-scaffold-command — scaffold command and initial specs layout
