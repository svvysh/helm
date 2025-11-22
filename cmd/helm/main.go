package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/polarzero/helm/internal/config"
	runtui "github.com/polarzero/helm/internal/tui/run"
	scaffoldtui "github.com/polarzero/helm/internal/tui/scaffold"
	settingsui "github.com/polarzero/helm/internal/tui/settings"
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
		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("load settings: %w", err)
		}

		specsRoot := config.ResolveSpecsRoot(".", settings.SpecsRoot)
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
		newSettingsCmd(),
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
			settings, err := config.LoadSettings()
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
		Short: "Run specs via an interactive TUI",
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
			return runtui.Run(runtui.Options{
				Root:      root,
				SpecsRoot: specsRoot,
				Settings:  settings,
			})
		},
	}
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
