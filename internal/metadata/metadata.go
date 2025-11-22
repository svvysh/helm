package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SpecStatus enumerates the lifecycle state for a spec.
type SpecStatus string

const (
	StatusTodo       SpecStatus = "todo"
	StatusInProgress SpecStatus = "in-progress"
	StatusDone       SpecStatus = "done"
	StatusBlocked    SpecStatus = "blocked"
)

// SpecMetadata represents the metadata.json schema for a spec folder.
type SpecMetadata struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Status             SpecStatus `json:"status"`
	DependsOn          []string   `json:"dependsOn"`
	LastRun            *time.Time `json:"lastRun,omitempty"`
	Notes              string     `json:"notes,omitempty"`
	AcceptanceCommands []string   `json:"acceptanceCommands"`
}

// LoadMetadata reads metadata.json from the provided path.
func LoadMetadata(path string) (*SpecMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata %s: %w", path, err)
	}

	var meta SpecMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse metadata %s: %w", path, err)
	}

	return &meta, nil
}

// SaveMetadata writes metadata.json to the provided path.
func SaveMetadata(path string, meta *SpecMetadata) error {
	if meta == nil {
		return errors.New("metadata is nil")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ensure metadata dir %s: %w", path, err)
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write metadata %s: %w", path, err)
	}

	return nil
}
