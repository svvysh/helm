# Helm Specs Workspace

This `specs/` directory contains your runnable specs plus the prompt templates Helm uses at runtime.

## What was scaffolded

- `implement.prompt-template.md` — worker prompt (edit to change tone or required outputs).
- `review.prompt-template.md` — verifier prompt (enforces STATUS: ok|missing + remainingTasks JSON).
- `specs-breakdown-guide.md` — guidance the split command feeds to Codex when breaking down large specs.
- `spec-00-example/` — a minimal example spec showing the required files:
  - `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, `implementation-report.md`.

## How to use

1. Add or edit `spec-XX-*` folders under this directory.
2. Run `helm` to open the TUI, or use direct commands:
   - `helm run` — pick and execute a spec (uses the Go runner).
   - `helm spec` — paste a big spec to split it using `specs-breakdown-guide.md`.
   - `helm status` — view overall state (when implemented).
3. Customize the prompt templates here as needed; Helm always reads them from disk.

## File conventions

- Each spec lives in `spec-XX-name/` with the four files above.
- `metadata.json` tracks status (`todo`, `in-progress`, `done`) and dependencies.
- `acceptance-checklist.md` plus your acceptance commands guide the verifier.

Customize freely—these files are meant to be edited per-repo.