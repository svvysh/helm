# Acceptance Checklist — spec-02-scaffold-command

## Automated commands

- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.

## Manual checks

- [ ] Running `go run ./cmd/helm scaffold` starts a Bubble Tea TUI instead of a bare CLI prompt.
- [ ] The TUI walks through: intro → mode selection → acceptance commands input → optional settings → confirmation → progress → completion.
- [ ] After completing the flow, a `docs/specs` directory exists at the chosen root.
- [ ] `.cli-settings.json` reflects the chosen mode and acceptance commands.
- [ ] `implement.prompt-template.md` and `review.prompt-template.md` exist and contain the documented placeholders.
- [ ] `implement-spec.mjs` exists and is a valid Node script file (at minimum, running `node docs/specs/implement-spec.mjs docs/specs/spec-00-example` does not crash immediately).
- [ ] `spec-00-example` exists and contains `SPEC.md`, `acceptance-checklist.md`, `metadata.json`, and `implementation-report.md`.
