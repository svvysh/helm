# Acceptance Checklist — spec-03-implement-runner

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] Running `node docs/specs/implement-spec.mjs docs/specs/spec-00-example` with a fake or mock Codex CLI does not crash and produces readable logs.
- [ ] When the verifier prints `STATUS: ok` and `{"remainingTasks":[]}`, the corresponding `metadata.json` is updated with `"status": "done"` and a recent `lastRun` timestamp.
- [ ] When the verifier prints `STATUS: missing` and a non-empty `remainingTasks`, the corresponding `metadata.json` is updated with `"status": "in-progress"` and `notes` mentions the remaining tasks.
- [ ] `implementation-report.md` contains the number of attempts, final status, remaining tasks, and a summary of the worker output.
- [ ] The script exits with status code 0 when `STATUS: ok` is reached, and non-zero when all attempts are exhausted without `STATUS: ok`.
