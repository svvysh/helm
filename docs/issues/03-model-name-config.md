# Issue: Codex model option typo and duplication

## Problem
- Allowed model list uses `git-5.1-codex-max` (typo) in both config validation and settings TUI (`internal/config/config.go:181-186`, `internal/tui/settings/model.go:71-78`). This option will be rejected by Codex and confuses users. Model names are duplicated across packages, risking drift.

## Desired fix (conceptual)
- Correct the model name to `gpt-5.1-codex-max` everywhere (validation, defaults, settings TUI options).
- Centralize allowed model names/reasoning levels in a single source (e.g., config package) and have TUIs consume it to avoid duplication.
- Update specs under `docs/specs/` to reflect the corrected model option and any defaults shown to users.

## Acceptance criteria
- Settings validation passes for the corrected model; typo removed everywhere.
- TUI model cycling pulls options from a single shared list; no duplication.
- Specs/docs updated to show the corrected model names.

## Prompt (copy/paste to LLM)
```
You are a senior Go engineer. Fix Helm Codex model naming:
1) Rename the erroneous model option `git-5.1-codex-max` to `gpt-5.1-codex-max` across validation, defaults, and UI.
2) Centralize allowed model names/reasoning pairs in the config package; make the settings TUI read from that shared source instead of hard-coded lists.
3) Update specs in docs/specs/ so documentation reflects the corrected model list.
Add/adjust tests if needed to cover validation and TUI option sourcing.
```
