package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/polarzero/helm/internal/config"
	innerscaffold "github.com/polarzero/helm/internal/scaffold"
	"github.com/polarzero/helm/internal/tui/home"
	runtui "github.com/polarzero/helm/internal/tui/run"
	scaffoldtui "github.com/polarzero/helm/internal/tui/scaffold"
	settingsui "github.com/polarzero/helm/internal/tui/settings"
	specsplittui "github.com/polarzero/helm/internal/tui/specsplit"
	statusui "github.com/polarzero/helm/internal/tui/status"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

type contextKey string

const settingsContextKey contextKey = "settings"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "helm",
		Short:        "Cross-project spec runner",
		Long:         "Helm orchestrates cross-project specs via a cohesive CLI interface. Run without a subcommand to open the multi-pane TUI.",
		SilenceUsage: true,
	}

	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if isScaffoldCommand(cmd) {
			return nil
		}

		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("load settings: %w", err)
		}
		root, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("determine working directory: %w", err)
		}
		if err := applySpecsRootFallback(root, settings); err != nil {
			return err
		}

		// Enforce repo initialization for subcommands.
		if !isRootCommand(cmd) {
			rc, err := loadRepoConfig(root)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("helm.config.json not found; run `helm scaffold` or `helm` to initialize this repo")
				}
				return err
			}
			if !rc.Initialized {
				return fmt.Errorf("repo not initialized; run `helm scaffold` or `helm` first")
			}
			specsRoot := config.ResolveSpecsRoot(root, rc.SpecsRoot)
			if _, err := os.Stat(specsRoot); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("specs root %s does not exist; run `helm scaffold` to create it", specsRoot)
				}
				return fmt.Errorf("stat specs root: %w", err)
			}
			// keep settings in context for downstream TUIs
			ctx := context.WithValue(cmd.Context(), settingsContextKey, settings)
			cmd.SetContext(ctx)
			return nil
		}

		ctx := context.WithValue(cmd.Context(), settingsContextKey, settings)
		cmd.SetContext(ctx)
		return nil
	}

	cmd.AddCommand(
		newScaffoldCmd(),
		newSettingsCmd(),
		newRunCmd(),
		newSpecCmd(),
		newStatusCmd(),
	)

	// Root (bare `helm`) opens the home TUI.
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		settings, err := settingsFromContext(cmd.Context())
		if err != nil {
			return err
		}
		root, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("determine working directory: %w", err)
		}
		if err := applySpecsRootFallback(root, settings); err != nil {
			return err
		}

		rc, err := loadRepoConfig(root)
		if err != nil || !rc.Initialized {
			fmt.Fprintln(cmd.OutOrStdout(), "Helm has not been initialized in this repo. Launching scaffold...")
			createdRoot, err := runScaffoldFlow(cmd, settings)
			if err != nil {
				return err
			}
			if createdRoot == "" {
				return nil // user cancelled; do not continue
			}
			rc = &repoConfig{SpecsRoot: createdRoot, Initialized: true}
			if err := saveRepoConfig(root, rc); err != nil {
				return fmt.Errorf("save repo config: %w", err)
			}
		}

		specsRoot := config.ResolveSpecsRoot(root, rc.SpecsRoot)
		acceptance := resolveAcceptance(specsRoot, settings)

		for {
			result, err := home.Run(home.Options{
				Root:               root,
				SpecsRoot:          specsRoot,
				Settings:           settings,
				AcceptanceCommands: acceptance,
			})
			if err != nil {
				if errors.Is(err, home.ErrCanceled) {
					return nil
				}
				return err
			}

			switch result.Selection {
			case home.SelectRun:
				if err := runtui.Run(runtui.Options{
					Root:      root,
					SpecsRoot: specsRoot,
					Settings:  settings,
				}); err != nil {
					if errors.Is(err, specsplittui.ErrQuitAll) || errors.Is(err, runtui.ErrQuitAll) {
						return nil
					}
					return err
				}
			case home.SelectBreakdown:
				if err := runSpecSplit(cmd, settings, specsRoot, "", ""); err != nil {
					if errors.Is(err, specsplittui.ErrQuitAll) {
						return nil
					}
					return err
				}
			case home.SelectStatus:
				if err := statusui.Run(statusui.Options{SpecsRoot: specsRoot}); err != nil {
					if errors.Is(err, statusui.ErrQuitAll) {
						return nil
					}
					return err
				}
			case home.SelectQuit:
				return nil
			default:
				return nil
			}
		}
	}

	return cmd
}

func newScaffoldCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scaffold",
		Short: "Scaffold assets for a new spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := config.LoadSettings()
			if err != nil {
				return fmt.Errorf("load settings: %w", err)
			}
			_, err = runScaffoldFlow(cmd, settings)
			return err
		},
	}
}

func newSettingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "settings",
		Short: "Edit global Helm settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			current, _ := config.LoadSettings()
			updated, err := settingsui.Run(settingsui.Options{Initial: current})
			if err != nil {
				if errors.Is(err, settingsui.ErrCanceled) {
					fmt.Fprintln(cmd.OutOrStdout(), "Settings unchanged.")
					return nil
				}
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Settings saved.")
			if updated != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Specs root: %s\n", updated.SpecsRoot)
			}
			return nil
		},
	}
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run specs via an interactive TUI (direct flow)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := settingsFromContext(cmd.Context())
			if err != nil {
				return err
			}

			root, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determine working directory: %w", err)
			}

			specsRoot := config.ResolveSpecsRoot(root, settings.SpecsRoot)
			if err := runtui.Run(runtui.Options{
				Root:      root,
				SpecsRoot: specsRoot,
				Settings:  settings,
			}); err != nil {
				if errors.Is(err, runtui.ErrQuitAll) {
					return nil
				}
				return err
			}
			return nil
		},
	}
	return cmd
}

func newSpecCmd() *cobra.Command {
	var filePath string
	var planPath string
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Split large specs into incremental ones (direct flow)",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := settingsFromContext(cmd.Context())
			if err != nil {
				return err
			}
			root, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determine working directory: %w", err)
			}
			rc, err := loadRepoConfig(root)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("helm.config.json not found; run `helm scaffold` or `helm` first")
				}
				return err
			}
			if err := applySpecsRootFallback(root, settings); err != nil {
				return err
			}
			specsRoot := config.ResolveSpecsRoot(root, rc.SpecsRoot)
			if err := runSpecSplit(cmd, settings, specsRoot, filePath, planPath); err != nil {
				if errors.Is(err, specsplittui.ErrQuitAll) {
					return nil
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "optional path to a spec text file to preload")
	cmd.Flags().StringVar(&planPath, "plan-file", "", "dev: provide a JSON plan instead of calling Codex")
	_ = cmd.Flags().MarkHidden("plan-file")
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the status of specs (direct flow)",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := settingsFromContext(cmd.Context())
			if err != nil {
				return err
			}
			root, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determine working directory: %w", err)
			}
			specsRoot := config.ResolveSpecsRoot(root, settings.SpecsRoot)
			if err := statusui.Run(statusui.Options{SpecsRoot: specsRoot}); err != nil {
				if errors.Is(err, statusui.ErrQuitAll) {
					return nil
				}
				return err
			}
			return nil
		},
	}
}

func isScaffoldCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "scaffold" {
			return true
		}
	}
	return false
}

func settingsFromContext(ctx context.Context) (*config.Settings, error) {
	if ctx == nil {
		return nil, errors.New("context missing CLI settings")
	}
	val := ctx.Value(settingsContextKey)
	settings, ok := val.(*config.Settings)
	if !ok || settings == nil {
		return nil, errors.New("CLI settings not initialized; run from a Helm workspace")
	}
	return settings, nil
}

func workspaceAcceptanceCommands(specsRoot string) []string {
	if specsRoot == "" {
		return nil
	}
	path := filepath.Join(specsRoot, ".cli-settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg config.Settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return cloneStrings(cfg.AcceptanceCommands)
}

func resolveAcceptance(specsRoot string, settings *config.Settings) []string {
	acc := workspaceAcceptanceCommands(specsRoot)
	if len(acc) == 0 {
		acc = cloneStrings(settings.AcceptanceCommands)
	}
	if len(acc) == 0 {
		acc = innerscaffold.DefaultAcceptanceCommands()
	}
	return acc
}

func runSpecSplit(cmd *cobra.Command, settings *config.Settings, specsRoot, filePath, planPath string) error {
	var initial string
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read spec file %s: %w", filePath, err)
		}
		initial = string(data)
	}

	acceptance := resolveAcceptance(specsRoot, settings)

	outcome, err := specsplittui.Run(specsplittui.Options{
		SpecsRoot:          specsRoot,
		GuidePath:          filepath.Join(specsRoot, "specs-breakdown-guide.md"),
		AcceptanceCommands: acceptance,
		CodexChoice:        settings.CodexSplit,
		InitialInput:       initial,
		PlanPath:           planPath,
	})
	if err != nil {
		if errors.Is(err, specsplittui.ErrQuitAll) {
			return specsplittui.ErrQuitAll
		}
		if errors.Is(err, specsplittui.ErrCanceled) {
			fmt.Fprintln(cmd.OutOrStdout(), "Spec split canceled.")
			return nil
		}
		return err
	}
	if outcome != nil && outcome.Result != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Created %d spec(s).\n", len(outcome.Result.Specs))
	}
	if outcome != nil && outcome.JumpToRun {
		return runtui.Run(runtui.Options{Root: specsRoot, SpecsRoot: specsRoot, Settings: settings})
	}
	return nil
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func isRootCommand(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Parent() == nil
}

// applySpecsRootFallback adjusts settings.SpecsRoot to point to an existing root when possible.
// Preference order: configured path (if exists), default "specs" (if exists), fallback "docs/specs" (if exists).
func applySpecsRootFallback(root string, settings *config.Settings) error {
	if settings == nil {
		return errors.New("nil settings")
	}

	// keep configured if it exists
	if p := config.ResolveSpecsRoot(root, settings.SpecsRoot); exists(p) {
		return nil
	}

	// default path exists?
	if settings.SpecsRoot == "" || settings.SpecsRoot == config.DefaultSpecsRoot() {
		fallback := config.ResolveSpecsRoot(root, filepath.Join("docs", "specs"))
		if exists(fallback) {
			settings.SpecsRoot = filepath.Join("docs", "specs")
			return nil
		}
	}

	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// runScaffoldFlow runs the scaffold TUI with provided defaults.
func runScaffoldFlow(cmd *cobra.Command, defaults *config.Settings) (string, error) {
	result, err := scaffoldtui.Run(scaffoldtui.Options{
		Root:     ".",
		Defaults: defaults,
	})
	if err != nil {
		if errors.Is(err, scaffoldtui.ErrCanceled) {
			fmt.Fprintln(cmd.OutOrStdout(), "Scaffold canceled.")
			return "", nil
		}
		return "", err
	}
	if result != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Workspace ready at %s\n", result.SpecsRoot)
		root, _ := os.Getwd()
		rc := &repoConfig{
			SpecsRoot:   repoRelative(root, result.SpecsRoot),
			Initialized: true,
		}
		_ = saveRepoConfig(root, rc)
	}
	if result == nil {
		return "", nil
	}
	return repoRelative(".", result.SpecsRoot), nil
}

// repoConfig persists minimal repo-scoped initialization state.
type repoConfig struct {
	SpecsRoot   string `json:"specsRoot"`
	Initialized bool   `json:"initialized"`
}

func repoConfigPath(root string) string {
	if root == "" {
		root = "."
	}
	return filepath.Join(root, "helm.config.json")
}

func loadRepoConfig(root string) (*repoConfig, error) {
	data, err := os.ReadFile(repoConfigPath(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	var rc repoConfig
	if err := json.Unmarshal(data, &rc); err != nil {
		return nil, err
	}
	return &rc, nil
}

func saveRepoConfig(root string, rc *repoConfig) error {
	if rc == nil {
		return errors.New("nil repo config")
	}
	data, err := json.MarshalIndent(rc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(repoConfigPath(root), data, 0o644)
}

// repoRelative stores a path relative to repo when possible; otherwise returns abs.
func repoRelative(root, abs string) string {
	if abs == "" {
		return ""
	}
	if root == "" {
		root = "."
	}
	rel, err := filepath.Rel(root, abs)
	if err == nil && !strings.HasPrefix(rel, "..") && rel != "" {
		return rel
	}
	return abs
}
