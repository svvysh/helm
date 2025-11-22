package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettingsDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()

	got, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if got.SpecsRoot != "docs/specs" {
		t.Fatalf("SpecsRoot = %q, want docs/specs", got.SpecsRoot)
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
	want := &Settings{
		SpecsRoot:          "alt/specs",
		Mode:               ModeParallel,
		DefaultMaxAttempts: 5,
		CodexModelScaffold: "gpt-4",
		AcceptanceCommands: []string{"go test ./..."},
	}

	if err := SaveSettings(dir, want); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	path := filepath.Join(dir, "docs", "specs", ".cli-settings.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected settings file: %v", err)
	}

	got, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if got.SpecsRoot != want.SpecsRoot {
		t.Fatalf("SpecsRoot mismatch: got=%q want=%q", got.SpecsRoot, want.SpecsRoot)
	}

	if got.Mode != want.Mode || got.DefaultMaxAttempts != want.DefaultMaxAttempts || got.CodexModelScaffold != want.CodexModelScaffold {
		t.Fatalf("LoadSettings mismatch got=%+v want=%+v", got, want)
	}

	if len(got.AcceptanceCommands) != len(want.AcceptanceCommands) || got.AcceptanceCommands[0] != want.AcceptanceCommands[0] {
		t.Fatalf("AcceptanceCommands mismatch: got=%v want=%v", got.AcceptanceCommands, want.AcceptanceCommands)
	}
}
