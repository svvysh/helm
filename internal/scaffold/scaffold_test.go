package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/polarzero/helm/internal/config"
)

func TestRunCreatesWorkspace(t *testing.T) {
	root := t.TempDir()
	os.Setenv("HELM_CONFIG_DIR", filepath.Join(root, ".config"))
	answers := Answers{
		Mode:                config.ModeStrict,
		AcceptanceCommands:  []string{"go test ./..."},
		SpecsRoot:           "tmp/test-specs",
		GenerateSampleGraph: true,
	}
	result, err := Run(root, answers)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	settingsPath := filepath.Join(root, ".config", "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Fatalf("missing settings file: %v", err)
	}
	var settings config.Settings
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	if settings.Mode != config.ModeStrict {
		t.Fatalf("settings.Mode = %s", settings.Mode)
	}
	metaPath := filepath.Join(root, "tmp", "test-specs", "spec-00-example", "metadata.json")
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("missing example metadata: %v", err)
	}
	graphPath := filepath.Join(root, "tmp", "test-specs", "sample-dependency-graph.json")
	if _, err := os.Stat(graphPath); err != nil {
		t.Fatalf("missing dependency graph: %v", err)
	}
}

func TestRunSkipsExistingFiles(t *testing.T) {
	root := t.TempDir()
	answers := Answers{SpecsRoot: "tmp/test-specs"}
	if _, err := Run(root, answers); err != nil {
		t.Fatalf("first run error: %v", err)
	}
	res, err := Run(root, answers)
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}
	if len(res.Skipped) == 0 {
		t.Fatalf("expected skipped files on rerun")
	}
}
