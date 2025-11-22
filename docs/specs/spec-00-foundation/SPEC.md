# spec-00-foundation — Go module and CLI skeleton

## Summary

Establish the foundational Go module and CLI entrypoint for the Cross-Project Spec Runner CLI (`helm`). This spec creates the basic project layout, wiring a Cobra-based CLI with stub subcommands that will later be implemented by other specs.

## Goals

- Initialize a Go module for the project.
- Add a `helm` CLI with Cobra.
- Provide subcommands matching the product spec:
  - `scaffold`
  - `run`
  - `spec`
  - `status`
- Ensure the project builds and tests cleanly (`go test ./...` and `go vet ./...`).
- Add a linting target (golangci-lint) and include it in the default `make all` flow.

## Non-Goals

- No Bubble Tea TUI logic yet.
- No actual spec discovery, metadata handling, or runner integration.
- No interaction with Codex or `implement-spec.mjs`.

## Detailed Requirements

1. **Go Module Setup**
   - Initialize a Go module (e.g., `module github.com/your-org/helm`).
   - Ensure module path is reasonable and easy to change later.
   - Add standard `.gitignore` entries for Go projects if they do not already exist.

2. **CLI Entrypoint**
   - Create `cmd/helm/main.go` as the entrypoint.
   - Use Cobra to define a root command named `helm`.
   - The root command should:
     - Provide a short and long description.
     - Print helpful usage information when invoked with `--help`.

3. **Subcommands (Stubs)**
   - Implement stub Cobra commands:
     - `helm scaffold`
     - `helm run`
     - `helm spec`
     - `helm status`
   - For this spec, each subcommand may simply:
     - Print a one-line message (e.g., "scaffold not implemented yet").
     - Exit with status code 0.
   - Wire subcommands so that `helm --help` and `helm <subcommand> --help` show correct usage.

4. **Project Layout Scaffold**
   - Create stub internal package layout in `internal/`:
     - `internal/config` (empty or with a minimal placeholder).
     - `internal/specs` (empty or with a minimal placeholder).
     - `internal/metadata` (empty or with a minimal placeholder).
     - `internal/tui` (empty or with a minimal placeholder).
   - These packages will be filled in by later specs.

5. **Tooling & Basic Testing**
   - Add at least one trivial unit test (e.g., verifying a constant or a simple function) to ensure the test harness is wired correctly.
   - Ensure `go test ./...` and `go vet ./...` succeed.

## Acceptance Criteria

- Running `go test ./...` succeeds.
- Running `go vet ./...` succeeds.
- `go run ./cmd/helm --help` shows:
  - A root command named `helm`.
  - Subcommands: `scaffold`, `run`, `spec`, and `status`.
- `go run ./cmd/helm scaffold` prints a human-readable stub message and exits with status 0.
- `go run ./cmd/helm run` prints a human-readable stub message and exits with status 0.
- `go run ./cmd/helm spec` prints a human-readable stub message and exits with status 0.
- `go run ./cmd/helm status` prints a human-readable stub message and exits with status 0.

## Implementation Notes

- Prefer using `spf13/cobra` for the CLI.
- Keep the subcommands minimal; later specs will replace the stub implementations with real behavior.
- Make it easy for future specs to add flags and configuration to each subcommand.
- Provide a tiny Makefile to streamline local use:
  - `make run CMD=run` executes `go run ./cmd/helm run`.
  - `make build` produces a `bin/helm` binary.
  - `make deps`, `make test`, and `make vet` wrap the corresponding Go tooling.
  - `make lint` runs `golangci-lint` (and is included in `make all`).
  - `make fmt` runs `gofumpt` + `goimports` for strict formatting and import fixes (installs them automatically if missing).
  - `make all` runs deps → tidy → fmt → vet → test → build for a quick local verification pass.
  - `make release` cross-compiles into `dist/helm_<GOOS>_<GOARCH>[.exe]` for common platforms (macOS, Linux, Windows on amd64/arm64).
  - `make clean` removes `bin/` and `dist/` artifacts.
  - CI should invoke setup via the shared `.github/actions/setup-go` composite and run vet/test/build as separate steps (not `make all`) for clearer debugging. Pushes to `main` also build and upload cross-platform artifacts from `dist/` as workflow artifacts.
  - Releases should run in a dedicated workflow (triggered by tags `v*` or manual dispatch) that reuses the shared setup action, runs fmt/vet/test/build, runs `make release`, and publishes the artifacts to a GitHub Release.
- Testing convention: when later specs scaffold or discover specs, direct tests to a temp specs root (e.g., `t.TempDir()/specs-test`) to keep the tracked `docs/specs/` tree clean.

## Depends on

- (None — this is the first spec.)
