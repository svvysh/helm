# Issue: Spec Split UX & guide mismatch

## Problem
- In the spec split TUI, Enter submits immediately, preventing multi-line editing; users cannot type a spec unless they paste it (`internal/tui/specsplit/model.go:208-226`).
- Default guide fallback looks for `spec-splitting-guide.md`, but scaffold generates `specs-breakdown-guide.md`, so `specsplit.Split` fails when no guide path is given (`internal/specsplit/specsplit.go:124-131`).
- Plan prompt example closes the JSON code block with a stray `)` producing invalid guidance (`internal/specsplit/specsplit.go:543-562`).

## Desired fix (conceptual)
- Allow Enter to insert newlines; use Ctrl+Enter (or similar) to start splitting, and keep a clear submit hint in the footer.
- Align default guide filename with the scaffolded `specs-breakdown-guide.md`.
- Fix the plan prompt so the JSON snippet is valid markdown/JSON (remove the stray `)`).
- Update specs in `docs/specs/` to reflect the corrected guide name, prompt contract, and keybindings for the split flow.

## Acceptance criteria
- Users can type multi-line specs; Enter inserts a newline, and a dedicated shortcut submits.
- Running split without explicit guide path succeeds in a freshly scaffolded repo.
- Plan prompt shows valid JSON contract.
- Specs/docs updated to mirror the new behavior and guide name.

## Prompt (copy/paste to LLM)
```
You are a senior Go TUI engineer. Fix Helm spec-split UX and guide defaults:
1) In the spec split TUI, Enter should insert a newline. Introduce a clear submit shortcut (e.g., Ctrl+Enter) and update footer hints. Ensure tests cover this behavior.
2) Align default guide fallback with the scaffolded filename `specs-breakdown-guide.md` so Split works out of the box.
3) Correct the plan prompt JSON snippet (remove stray characters) so it is valid markdown/JSON.
Update code, tests, and the relevant specs under docs/specs/ to document the new keybinding and guide name. Ensure fresh runs follow the corrected contract.
```
