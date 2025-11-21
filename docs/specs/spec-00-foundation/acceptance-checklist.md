# Acceptance Checklist — spec-00-foundation

## Automated commands

- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.

## Manual checks

- [ ] A Go module is initialized (a `go.mod` file exists at the repository root).
- [ ] `go run ./cmd/helm --help` prints usage and lists the `scaffold`, `run`, `spec`, and `status` subcommands.
- [ ] `go run ./cmd/helm scaffold` prints a stub "not implemented yet" style message and exits successfully.
- [ ] `go run ./cmd/helm run` prints a stub message and exits successfully.
- [ ] `go run ./cmd/helm spec` prints a stub message and exits successfully.
- [ ] `go run ./cmd/helm status` prints a stub message and exits successfully.
- [ ] At least one trivial unit test exists and is executed by `go test ./...`.
