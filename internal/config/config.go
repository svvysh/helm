package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	defaultSpecsRoot    = "specs"
	defaultSettingsDir  = ".helm"
	defaultSettingsFile = "settings.json"
)

// DefaultSpecsRoot returns the canonical specs root when no override is provided.
func DefaultSpecsRoot() string {
	return defaultSpecsRoot
}

// ResolveSpecsRoot returns the absolute path to the specs root relative to root when the
// provided specsRoot is relative. Absolute specsRoot values are returned untouched.
func ResolveSpecsRoot(root, specsRoot string) string {
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

// Mode describes how acceptance commands should be evaluated.
type Mode string

const (
	ModeParallel Mode = "parallel"
	ModeStrict   Mode = "strict"
)

// CodexChoice couples a model with reasoning effort.
type CodexChoice struct {
	Model     string `json:"model"`
	Reasoning string `json:"reasoning"`
}

// Settings represents the persisted CLI configuration (global, per-user).
type Settings struct {
	SpecsRoot          string      `json:"specsRoot"`
	Mode               Mode        `json:"mode"`
	DefaultMaxAttempts int         `json:"defaultMaxAttempts"`
	CodexScaffold      CodexChoice `json:"codexScaffold"`
	CodexRunImpl       CodexChoice `json:"codexRunImpl"`
	CodexRunVer        CodexChoice `json:"codexRunVer"`
	CodexSplit         CodexChoice `json:"codexSplit"`
	AcceptanceCommands []string    `json:"acceptanceCommands"`
}

// LoadSettings reads global CLI settings from the user config directory.
func LoadSettings() (*Settings, error) {
	path, err := settingsFilePath()
	if err != nil {
		return nil, err
	}
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

	ApplyDefaults(&settings)
	if err := Validate(&settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// SaveSettings persists CLI settings to the user config directory.
func SaveSettings(settings *Settings) error {
	if settings == nil {
		return errors.New("settings is nil")
	}
	ApplyDefaults(settings)
	if err := Validate(settings); err != nil {
		return err
	}

	path, err := settingsFilePath()
	if err != nil {
		return err
	}
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

func settingsFilePath() (string, error) {
	if override := os.Getenv("HELM_CONFIG_DIR"); override != "" {
		return filepath.Join(override, defaultSettingsFile), nil
	}

	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", fmt.Errorf("resolve user config dir: %v, %v", err, herr)
		}
		dir = filepath.Join(home, defaultSettingsDir)
	} else {
		if runtime.GOOS == "windows" {
			dir = filepath.Join(dir, "Helm")
		} else {
			dir = filepath.Join(dir, defaultSettingsDir)
		}
	}
	return filepath.Join(dir, defaultSettingsFile), nil
}

func defaultSettings() Settings {
	return Settings{
		SpecsRoot:          defaultSpecsRoot,
		Mode:               ModeStrict,
		DefaultMaxAttempts: 2,
		CodexScaffold:      CodexChoice{Model: "gpt-5.1", Reasoning: "medium"},
		CodexRunImpl:       CodexChoice{Model: "gpt-5.1-codex", Reasoning: "medium"},
		CodexRunVer:        CodexChoice{Model: "gpt-5.1-codex", Reasoning: "medium"},
		CodexSplit:         CodexChoice{Model: "gpt-5.1", Reasoning: "medium"},
		AcceptanceCommands: []string{},
	}
}

// ApplyDefaults fills empty fields with defaults; exported for tests and TUIs.
func ApplyDefaults(settings *Settings) {
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
	if settings.CodexScaffold.Model == "" {
		settings.CodexScaffold = CodexChoice{Model: "gpt-5.1", Reasoning: "medium"}
	}
	if settings.CodexRunImpl.Model == "" {
		settings.CodexRunImpl = CodexChoice{Model: "gpt-5.1-codex", Reasoning: "medium"}
	}
	if settings.CodexRunVer.Model == "" {
		settings.CodexRunVer = CodexChoice{Model: "gpt-5.1-codex", Reasoning: "medium"}
	}
	if settings.CodexSplit.Model == "" {
		settings.CodexSplit = CodexChoice{Model: "gpt-5.1", Reasoning: "medium"}
	}
}

var allowedChoices = map[string][]string{
	"gpt-5.1":            {"low", "medium", "high"},
	"gpt-5.1-codex":      {"low", "medium", "high"},
	"gpt-5.1-codex-mini": {"medium", "high"},
	"git-5.1-codex-max":  {"low", "medium", "high", "xhigh"},
}

// Validate ensures model/reasoning pairs are allowed.
func Validate(s *Settings) error {
	check := func(label string, c CodexChoice) error {
		allowed, ok := allowedChoices[c.Model]
		if !ok {
			return fmt.Errorf("%s model %q is not supported", label, c.Model)
		}
		for _, r := range allowed {
			if r == c.Reasoning {
				return nil
			}
		}
		return fmt.Errorf("%s reasoning %q not allowed for model %q", label, c.Reasoning, c.Model)
	}
	if err := check("codexScaffold", s.CodexScaffold); err != nil {
		return err
	}
	if err := check("codexRunImpl", s.CodexRunImpl); err != nil {
		return err
	}
	if err := check("codexRunVer", s.CodexRunVer); err != nil {
		return err
	}
	if err := check("codexSplit", s.CodexSplit); err != nil {
		return err
	}
	return nil
}
