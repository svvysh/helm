# Acceptance Checklist — spec-05-spec-splitting-command

## Automated commands

- [ ] `make all` passes.

## Manual checks

- [ ] Running `go run ./cmd/helm spec` presents an intro and then a text area where a large spec can be pasted.
- [ ] After pasting and confirming, a progress view is shown while waiting for Codex.
- [ ] When a valid JSON split plan is returned, multiple `spec-XX-*` folders are created under `docs/specs`.
- [ ] Each generated spec folder contains `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, and `implementation-report.md`.
- [ ] Dependencies from the JSON plan are represented in both `metadata.json.dependsOn` and the `## Depends on` section of `SPEC.md`.
- [ ] Existing spec folders are not silently overwritten without a prompt or warning.
