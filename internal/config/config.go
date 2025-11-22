package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	settingsFileName = ".cli-settings.json"
	defaultSpecsRoot = "docs/specs"
)

// DefaultSpecsRoot returns the canonical specs root path used when no override is provided.
func DefaultSpecsRoot() string {
	return defaultSpecsRoot
}

// ResolveSpecsRoot returns the absolute path to the specs root relative to root when the
// provided specsRoot is relative. Absolute specsRoot values are returned untouched.
func ResolveSpecsRoot(root, specsRoot string) string {
	return resolveSpecsRoot(root, specsRoot)
}

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
	path, err := locateSettingsFile(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			defaults := defaultSettings()
			return &defaults, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
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
	path := settingsFilePath(root, settings.SpecsRoot)
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

func settingsFilePath(root, specsRoot string) string {
	return filepath.Join(resolveSpecsRoot(root, specsRoot), settingsFileName)
}

func resolveSpecsRoot(root, specsRoot string) string {
	if specsRoot == "" {
		specsRoot = defaultSpecsRoot
	}
	if filepath.IsAbs(specsRoot) {
		return specsRoot
	}
	if root == "" {
		root = "."
	}
	return filepath.Join(root, specsRoot)
}

func locateSettingsFile(root string) (string, error) {
	defaultPath := settingsFilePath(root, defaultSpecsRoot)
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat settings %s: %w", defaultPath, err)
	}

	stopErr := errors.New("stop settings walk")
	var found string
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == settingsFileName {
			found = path
			return stopErr
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, stopErr) {
		return "", fmt.Errorf("locate settings: %w", walkErr)
	}
	if found == "" {
		return "", os.ErrNotExist
	}
	return found, nil
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
