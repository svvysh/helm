package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/polarzero/helm/internal/config"
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
			fmt.Fprintln(cmd.OutOrStdout(), "scaffold not implemented yet")
			return nil
		},
	}
}

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the spec workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "run not implemented yet")
			return nil
		},
	}
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

func settingsFromContext(ctx context.Context) *config.Settings {
	if ctx == nil {
		return nil
	}
	if val, ok := ctx.Value(settingsContextKey).(*config.Settings); ok {
		return val
	}
	return nil
}
