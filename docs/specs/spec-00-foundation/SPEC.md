# spec-00-foundation — Go module, CLI skeleton, and TUI entrypoints

## Summary

Establish the foundational Go module and Cobra-based CLI for Helm. The CLI must expose the four public entrypoints (`scaffold`, `run`, `spec`, `status`). The root command `helm` will launch the TUI; subcommands run their flows directly. For now the subcommands may be stubs, but their shapes must match this split.

## Goals

- Initialize a Go module for the project.
- Add a `helm` CLI with Cobra.
- Provide subcommands matching the product spec:
  - `scaffold`
  - `run`
  - `spec` (split large specs)
  - `status`
- Wire the root command so running `helm` (no subcommand) clearly indicates that it opens the TUI, while subcommands run directly.
- Ensure the project builds and tests cleanly (`go test ./...` and `go vet ./...`).
- Add a linting target (golangci-lint) and include it in the default `make all` flow.

## Non-Goals

- No Bubble Tea TUI yet; commands can print stub messages.
- No spec discovery, metadata handling, or runner integration.
- No Codex integration.

## Detailed Requirements

1. **Go Module Setup**
   - Initialize a Go module (e.g., `module github.com/your-org/helm`).
   - Add standard `.gitignore` entries for Go projects if missing.

2. **CLI Entrypoint**
   - Create `cmd/helm/main.go` as the entrypoint.
   - Use Cobra to define a root command named `helm`.
   - The root command should:
     - Provide a short and long description mentioning the forthcoming TUI-first experience.
     - Print helpful usage information when invoked with `--help`.

3. **Subcommands (Stubs)**
- Implement stub Cobra commands:
     - `helm scaffold`
     - `helm run`
     - `helm spec`
     - `helm status`
   - Each subcommand may simply print a one-line message (e.g., "run will open the TUI in a future spec") and exit 0.
   - Wire subcommands so `helm --help` and `helm <subcommand> --help` show correct usage.

4. **Project Layout Scaffold**
   - Create stub internal package layout in `internal/`:
     - `internal/config`
     - `internal/specs`
     - `internal/metadata`
     - `internal/tui`
   - These packages will be filled in by later specs.

5. **Tooling & Basic Testing**
   - Add at least one trivial unit test to ensure the test harness is wired correctly.
   - Ensure `go test ./...` and `go vet ./...` succeed.
   - Provide a Makefile with targets for `run`, `build`, `deps`, `test`, `vet`, `lint`, `fmt`, `all`, `release`, and `clean` as in the prior version of this spec.

## Acceptance Criteria

- Running `go test ./...` succeeds.
- Running `go vet ./...` succeeds.
- `go run ./cmd/helm --help` shows:
  - A root command named `helm` with text referencing the TUI-first interface.
  - Subcommands: `scaffold`, `run`, `spec`, and `status`.
- `go run ./cmd/helm scaffold` prints a human-readable stub message and exits with status 0.
- `go run ./cmd/helm run` prints a human-readable stub message and exits with status 0.
- `go run ./cmd/helm spec` prints a human-readable stub message and exits with status 0.
- `go run ./cmd/helm status` prints a human-readable stub message and exits with status 0.

## Implementation Notes

- Prefer using `spf13/cobra` for the CLI.
- Keep the subcommands minimal; later specs will replace the stub implementations with real behavior.
- Mention in descriptions that the default behavior is to open the TUI; this keeps expectations aligned with later specs.
- Testing convention: later specs will direct tests to a temp specs root to keep the tracked `docs/specs/` tree clean.

## Depends on

- (None — this is the first spec.)
