package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAndSaveMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.json")
	ts := time.Now().UTC().Truncate(time.Second)
	original := &SpecMetadata{
		ID:        "spec-01-config-metadata",
		Name:      "Settings, metadata, and spec discovery",
		Status:    StatusInProgress,
		DependsOn: []string{"spec-00-foundation"},
		LastRun:   &ts,
		Notes:     "test note",
		AcceptanceCommands: []string{
			"go test ./...",
			"go vet ./...",
		},
	}

	if err := SaveMetadata(path, original); err != nil {
		t.Fatalf("SaveMetadata() error = %v", err)
	}

	got, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v", err)
	}

	if got.ID != original.ID || got.Name != original.Name || got.Status != original.Status {
		t.Fatalf("LoadMetadata() mismatch got=%+v want=%+v", got, original)
	}

	if len(got.DependsOn) != len(original.DependsOn) || got.DependsOn[0] != original.DependsOn[0] {
		t.Fatalf("dependsOn mismatch: got=%v want=%v", got.DependsOn, original.DependsOn)
	}

	if len(got.AcceptanceCommands) != len(original.AcceptanceCommands) {
		t.Fatalf("acceptanceCommands mismatch: got=%v want=%v", got.AcceptanceCommands, original.AcceptanceCommands)
	}

	if got.LastRun == nil || !got.LastRun.Equal(*original.LastRun) {
		t.Fatalf("LastRun mismatch: got=%v want=%v", got.LastRun, original.LastRun)
	}

	if got.Notes != original.Notes {
		t.Fatalf("Notes mismatch: got=%q want=%q", got.Notes, original.Notes)
	}

	// Ensure file actually created
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected metadata file to exist: %v", err)
	}
}
