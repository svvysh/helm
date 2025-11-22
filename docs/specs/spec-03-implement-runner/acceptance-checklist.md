# Acceptance Checklist — spec-03-implement-runner

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] With a fake Codex binary and a temp `specs` root, running `helm run --exec spec-00-example` executes the Go runner without opening the TUI and streams worker/verifier output.
- [ ] When the verifier prints `STATUS: ok` and `{"remainingTasks":[]}`, `metadata.json` updates to `"status":"done"` with a fresh `lastRun`, and `implementation-report.md` records the attempt count and final status.
- [ ] When the verifier prints `STATUS: missing` with tasks, `metadata.json` updates to `"status":"in-progress"` with notes summarizing remaining tasks, and the report includes them.
- [ ] Invoking `helm run` **without** `--exec` delegates to the TUI entrypoint instead of running headless.
- [ ] Non-zero exit when attempts are exhausted without `STATUS: ok`, or when verifier output is malformed.
