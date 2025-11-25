# Acceptance Checklist — spec-00-aaa-command

## Automated commands

- [ ] `pnpm all`

## Spec criteria

- [ ] A Helm CLI subcommand `aaa` is available and discoverable via `helm --help` or the equivalent Helm CLI help output.
- [ ] Running `helm aaa` (or the project’s equivalent CLI entry) exits with code 0 and prints a user-visible message containing the string `aaa`.
- [ ] Running `pnpm all` completes successfully without errors.

