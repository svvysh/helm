package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/polarzero/helm/internal/config"
	"github.com/polarzero/helm/internal/runner"
	scaffoldtui "github.com/polarzero/helm/internal/tui/scaffold"
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
		Long:         "Helm orchestrates cross-project specs via a cohesive CLI interface.",
		SilenceUsage: true,
	}

	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if isScaffoldCommand(cmd) {
			return nil
		}
		settings, err := config.LoadSettings(".")
		if err != nil {
			return fmt.Errorf("load settings: %w", err)
		}

		specsRoot := filepath.Join(".", settings.SpecsRoot)
		if _, err := os.Stat(specsRoot); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("specs root %s does not exist; initialize docs/specs before running Helm", specsRoot)
			}
			return fmt.Errorf("stat specs root: %w", err)
		}

		ctx := context.WithValue(cmd.Context(), settingsContextKey, settings)
		cmd.SetContext(ctx)
		return nil
	}

	cmd.AddCommand(
		newScaffoldCmd(),
		newRunCmd(),
		newSpecCmd(),
		newStatusCmd(),
	)

	return cmd
}

func newScaffoldCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scaffold",
		Short: "Scaffold assets for a new spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := config.LoadSettings(".")
			if err != nil {
				return fmt.Errorf("load settings: %w", err)
			}
			result, err := scaffoldtui.Run(scaffoldtui.Options{
				Root:     ".",
				Defaults: settings,
			})
			if err != nil {
				if errors.Is(err, scaffoldtui.ErrCanceled) {
					fmt.Fprintln(cmd.OutOrStdout(), "Scaffold canceled.")
					return nil
				}
				return err
			}
			if result != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Workspace ready at %s\n", result.SpecsRoot)
			}
			return nil
		},
	}
}

func newRunCmd() *cobra.Command {
	var modeOverride string

	cmd := &cobra.Command{
		Use:   "run <spec-dir>",
		Short: "Run the spec workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := settingsFromContext(cmd.Context())
			if err != nil {
				return err
			}

			root, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determine working directory: %w", err)
			}

			maxAttempts := settings.DefaultMaxAttempts
			if env := os.Getenv("MAX_ATTEMPTS"); env != "" {
				value, convErr := strconv.Atoi(env)
				if convErr != nil || value <= 0 {
					return fmt.Errorf("invalid MAX_ATTEMPTS value %q", env)
				}
				maxAttempts = value
			}

			mode := settings.Mode
			if modeOverride != "" {
				switch config.Mode(modeOverride) {
				case config.ModeStrict, config.ModeParallel:
					mode = config.Mode(modeOverride)
				default:
					return fmt.Errorf("invalid mode %q: must be strict or parallel", modeOverride)
				}
			}

			workerModel := os.Getenv("CODEX_MODEL_IMPL")
			if workerModel == "" {
				workerModel = settings.CodexModelRunImpl
			}
			verifierModel := os.Getenv("CODEX_MODEL_VER")
			if verifierModel == "" {
				verifierModel = settings.CodexModelRunVer
			}

			specsRoot := config.ResolveSpecsRoot(root, settings.SpecsRoot)

			r := &runner.Runner{
				Root:                      root,
				SpecsRoot:                 specsRoot,
				Mode:                      mode,
				MaxAttempts:               maxAttempts,
				WorkerModel:               workerModel,
				VerifierModel:             verifierModel,
				DefaultAcceptanceCommands: settings.AcceptanceCommands,
				Stdout:                    cmd.OutOrStdout(),
				Stderr:                    cmd.ErrOrStderr(),
			}

			return r.Run(cmd.Context(), args[0])
		},
	}

	cmd.Flags().StringVar(&modeOverride, "mode", "", "Override run mode (strict or parallel)")
	return cmd
}

func newSpecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "spec",
		Short: "Inspect specs",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "spec not implemented yet")
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the status of specs",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "status not implemented yet")
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
