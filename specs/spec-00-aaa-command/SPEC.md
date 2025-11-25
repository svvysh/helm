# Add basic aaa CLI command

## Summary

Introduce a minimal Helm CLI command named `aaa` that prints a static message, validating the wiring of the CLI and test harness.

## Acceptance Criteria

- A Helm CLI subcommand `aaa` is available and discoverable via `helm --help` or the equivalent Helm CLI help output.
- Running `helm aaa` (or the project’s equivalent CLI entry) exits with code 0 and prints a user-visible message containing the string `aaa`.
- Running `pnpm all` completes successfully without errors.

## Depends on

- _None_

