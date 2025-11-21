# spec-01-config-metadata — Settings, metadata, and spec discovery

## Summary

Introduce the core domain types and filesystem conventions for the spec runner. This spec defines how `metadata.json` and `.cli-settings.json` are represented in Go, and adds basic logic to discover `spec-*` folders under `docs/specs`.

## Goals

- Define the `SpecMetadata` struct matching the metadata schema.
- Define a `Settings` struct for `.cli-settings.json`.
- Implement load/save helpers for metadata and settings.
- Implement basic spec discovery that finds spec folders and attaches metadata.
- Lay the groundwork for future TUI commands (`run`, `status`, etc.) that will use this information.

## Non-Goals

- No Bubble Tea TUI yet.
- No actual CLI command behavior beyond wiring stubs to call the new helpers.
- No Codex integration or runner orchestration.

## Detailed Requirements

1. **Metadata Model**
   - Create an `internal/metadata` package with:
     - A `SpecStatus` type with values:
       - `"todo"`, `"in-progress"`, `"done"`, `"blocked"`.
     - A `SpecMetadata` struct:

       ```go
       type SpecStatus string

       const (
           StatusTodo       SpecStatus = "todo"
           StatusInProgress SpecStatus = "in-progress"
           StatusDone       SpecStatus = "done"
           StatusBlocked    SpecStatus = "blocked"
       )

       type SpecMetadata struct {
           ID                 string      `json:"id"`
           Name               string      `json:"name"`
           Status             SpecStatus  `json:"status"`
           DependsOn          []string    `json:"dependsOn"`
           LastRun            *time.Time  `json:"lastRun,omitempty"`
           Notes              string      `json:"notes,omitempty"`
           AcceptanceCommands []string    `json:"acceptanceCommands"`
       }
       ```

   - Provide `LoadMetadata(path string) (*SpecMetadata, error)` and `SaveMetadata(path string, *SpecMetadata) error`.

2. **Settings Model**
   - Create an `internal/config` package with:

     ```go
     type Mode string

     const (
         ModeParallel Mode = "parallel"
         ModeStrict   Mode = "strict"
     )

     type Settings struct {
         SpecsRoot          string   `json:"specsRoot"`
         Mode               Mode     `json:"mode"`
         DefaultMaxAttempts int      `json:"defaultMaxAttempts"`
         CodexModelScaffold string   `json:"codexModelScaffold"`
         CodexModelRunImpl  string   `json:"codexModelRunImpl"`
         CodexModelRunVer   string   `json:"codexModelRunVer"`
         CodexModelSplit    string   `json:"codexModelSplit"`
         AcceptanceCommands []string `json:"acceptanceCommands"`
     }
     ```

   - Implement:
     - `LoadSettings(root string) (*Settings, error)`
       - Looks for `docs/specs/.cli-settings.json` by default.
       - If the file does not exist, returns sensible defaults:
         - `SpecsRoot = "docs/specs"`,
         - `Mode = ModeStrict`,
         - `DefaultMaxAttempts = 2`,
         - `AcceptanceCommands` empty.
     - `SaveSettings(root string, settings *Settings) error`.

3. **Spec Folder Representation**
   - Create an `internal/specs` package with:

     ```go
     type SpecFolder struct {
         ID        string
         Name      string
         Path      string
         Metadata  *metadata.SpecMetadata
         Checklist string   // path to acceptance-checklist.md
         CanRun    bool     // derived by dependency analysis
         UnmetDeps []string // IDs of deps not yet done
     }
     ```

   - Implement `DiscoverSpecs(root string) ([]*SpecFolder, error)` that:
     - Scans `root` (`docs/specs` by default).
     - Finds subdirectories whose names start with `spec-`.
     - Requires each spec folder to contain `SPEC.md` and `metadata.json`.
     - Reads `metadata.json` and populates `SpecFolder.Metadata`.
     - Points `Checklist` to `acceptance-checklist.md` if present (or leaves it empty).

4. **Dependency State Calculation**
   - Implement a helper, e.g., `ComputeDependencyState(specs []*SpecFolder)` that:
     - For each spec:
       - Sets `CanRun = true` if:
         - Its `Status` is not `"done"`, and
         - All `dependsOn` entries correspond to specs whose status is `"done"`.
       - Fills `UnmetDeps` with IDs that are not `"done"`.
   - The `blocked` status will be a *derived view*:
     - Do not mutate `metadata.status` to `"blocked"` automatically.
     - Later TUI code will decide when to show a spec as visually “blocked”.

5. **Root Command Wiring (Minimal)**
   - Update the root CLI command (from `spec-00-foundation`) so that:
     - It attempts to load settings on startup.
     - It prints a helpful error if `docs/specs` does not exist yet.
   - Do not add TUI behavior; just ensure the infrastructure compiles and can be imported by later specs.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` succeed.
- A unit test exists that:
  - Creates a temporary directory with a couple of fake `spec-*` folders and `metadata.json`.
  - Confirms that `DiscoverSpecs` returns the expected spec IDs.
  - Confirms that `ComputeDependencyState` correctly sets `CanRun` and `UnmetDeps`.
- Calling `LoadSettings` on a repo **without** `.cli-settings.json` returns defaults instead of failing.
- Calling `LoadMetadata` and `SaveMetadata` round-trips a `SpecMetadata` struct (including `dependsOn` and `acceptanceCommands`).

## Implementation Notes

- Keep the `internal` packages self-contained and free of CLI-specific concerns.
- Favor small helper functions that will be easy to unit test.
- Avoid embedding any TUI-specific logic here; that will come in later specs.

## Depends on

- spec-00-foundation — Go module and CLI skeleton
