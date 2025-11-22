# Issue: Spec discovery resilience & docs gaps

## Problem
- `DiscoverSpecs` aborts on the first spec folder missing `SPEC.md` or `metadata.json`, preventing Run/Status TUIs from listing remaining valid specs (`internal/specs/specs.go:24-72`).
- README and user docs are minimal; missing install/run instructions, TUI keymaps, and guidance for non-default specs roots—hurting onboarding and DX.

## Desired fix (conceptual)
- Make spec discovery resilient: skip invalid folders with a warning/flag instead of returning an error, so TUIs still render other specs. Surface a summary in Status/Run (e.g., info message) when skips occur.
- Expand docs: README plus a doc page covering installation, basic commands, TUI keybindings (home, run, split, status), and how to set/override specs root. Ensure the specs under `docs/specs/` incorporate these expectations so fresh incremental runs stay aligned.

## Acceptance criteria
- Run/Status TUIs continue to function when some spec folders are malformed, and users see an indication of skipped folders.
- README/docs updated with quickstart, keymaps, and specs-root guidance.
- Specs updated to encode the new resilience and documentation expectations.

## Prompt (copy/paste to LLM)
```
You are a senior Go CLI/TUI engineer. Improve robustness and docs:
1) Update DiscoverSpecs (and callers) to skip invalid spec folders instead of failing; capture skipped folders and surface an info/warning in Status/Run views.
2) Add documentation: README and a doc page describing install/run, TUI keybindings (home, run, split, status), and configuring specs root. Keep it concise but complete.
3) Update relevant specs under docs/specs/ so a fresh repo run reflects the resilient discovery and documented UX.
Add/adjust tests for discovery skipping behavior.
```
