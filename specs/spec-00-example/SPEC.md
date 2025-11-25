# spec-00-example — Example Feature Spec

## Summary

Demonstrates how specs, metadata, and the implementation runner work together. This example does not change production code; it simply proves the workflow end to end.

## Goals

- Provide a concrete `spec-XX-*` folder that new contributors can inspect.
- Exercise the implementation runner and verifier loop.
- Document how acceptance commands and metadata fit together.

## Non-Goals

- No new CLI commands are shipped as part of this example.
- Does not replace real specs for future roadmap items.

## Detailed Requirements

1. Keep this folder checked into source control so `helm scaffold` can recreate it elsewhere.
2. Show how to structure `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, and `implementation-report.md`.
3. Reference the default acceptance commands captured in the global settings file (`~/.helm/settings.json`).
4. Explain what the verifier should look for when reviewing future specs.

## Acceptance Criteria

- The acceptance checklist references each required command with short descriptions.
- `metadata.json.status` starts as `"todo"` with no dependencies.
- `implementation-report.md` documents when the runner last updated the spec (initially a placeholder).

Use this example as a template for creating your own specs.