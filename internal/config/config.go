package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	settingsFileName = ".cli-settings.json"
	defaultSpecsRoot = "docs/specs"
)

// Mode describes how acceptance commands should be evaluated.
type Mode string

const (
	ModeParallel Mode = "parallel"
	ModeStrict   Mode = "strict"
)

// Settings represents the persisted CLI configuration.
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

// LoadSettings reads CLI settings from docs/specs/.cli-settings.json.
func LoadSettings(root string) (*Settings, error) {
	path := settingsFilePath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			defaults := defaultSettings()
			return &defaults, nil
		}

		return nil, fmt.Errorf("read settings %s: %w", path, err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse settings %s: %w", path, err)
	}

	applyDefaults(&settings)
	return &settings, nil
}

// SaveSettings persists CLI settings to specsRoot/.cli-settings.json relative to root.
func SaveSettings(root string, settings *Settings) error {
	if settings == nil {
		return errors.New("settings is nil")
	}

	applyDefaults(settings)
	path := settingsFilePath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ensure settings dir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write settings %s: %w", path, err)
	}

	return nil
}

func settingsFilePath(root string) string {
	if root == "" {
		root = "."
	}
	return filepath.Join(root, defaultSpecsRoot, settingsFileName)
}

func defaultSettings() Settings {
	return Settings{
		SpecsRoot:          defaultSpecsRoot,
		Mode:               ModeStrict,
		DefaultMaxAttempts: 2,
		AcceptanceCommands: []string{},
	}
}

func applyDefaults(settings *Settings) {
	if settings.SpecsRoot == "" {
		settings.SpecsRoot = defaultSpecsRoot
	}
	if settings.Mode == "" {
		settings.Mode = ModeStrict
	}
	if settings.DefaultMaxAttempts == 0 {
		settings.DefaultMaxAttempts = 2
	}
	if settings.AcceptanceCommands == nil {
		settings.AcceptanceCommands = []string{}
	}
}
