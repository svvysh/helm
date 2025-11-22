package config

import (
	"os"
	"testing"
)

func TestLoadSettingsDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HELM_CONFIG_DIR", dir)

	got, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if got.SpecsRoot != "specs" {
		t.Fatalf("SpecsRoot = %q, want specs", got.SpecsRoot)
	}
	if got.Mode != ModeStrict {
		t.Fatalf("Mode = %q, want %q", got.Mode, ModeStrict)
	}
	if got.DefaultMaxAttempts != 2 {
		t.Fatalf("DefaultMaxAttempts = %d, want 2", got.DefaultMaxAttempts)
	}
	if got.AcceptanceCommands == nil || len(got.AcceptanceCommands) != 0 {
		t.Fatalf("AcceptanceCommands default mismatch: %v", got.AcceptanceCommands)
	}
}

func TestSaveSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HELM_CONFIG_DIR", dir)
	want := &Settings{
		SpecsRoot:          "alt/specs",
		Mode:               ModeParallel,
		DefaultMaxAttempts: 5,
		CodexScaffold:      CodexChoice{Model: "gpt-5.1", Reasoning: "high"},
		CodexRunImpl:       CodexChoice{Model: "gpt-5.1-codex", Reasoning: "medium"},
		CodexRunVer:        CodexChoice{Model: "gpt-5.1-codex", Reasoning: "high"},
		CodexSplit:         CodexChoice{Model: "gpt-5.1", Reasoning: "medium"},
		AcceptanceCommands: []string{"go test ./..."},
	}

	if err := SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	got, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if got.SpecsRoot != want.SpecsRoot {
		t.Fatalf("SpecsRoot mismatch: got=%q want=%q", got.SpecsRoot, want.SpecsRoot)
	}

	if got.Mode != want.Mode || got.DefaultMaxAttempts != want.DefaultMaxAttempts {
		t.Fatalf("LoadSettings mismatch got=%+v want=%+v", got, want)
	}
	if got.CodexScaffold != want.CodexScaffold || got.CodexRunImpl != want.CodexRunImpl || got.CodexRunVer != want.CodexRunVer || got.CodexSplit != want.CodexSplit {
		t.Fatalf("Codex choices mismatch got=%+v want=%+v", got, want)
	}
	if len(got.AcceptanceCommands) != len(want.AcceptanceCommands) || got.AcceptanceCommands[0] != want.AcceptanceCommands[0] {
		t.Fatalf("AcceptanceCommands mismatch: got=%v want=%v", got.AcceptanceCommands, want.AcceptanceCommands)
	}
}
