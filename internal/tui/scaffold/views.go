package scaffold

import (
	"fmt"
	"strings"

	"github.com/polarzero/helm/internal/config"
	innerscaffold "github.com/polarzero/helm/internal/scaffold"
	"github.com/polarzero/helm/internal/tui/components"
)

func introView(width int) string {
	body := strings.Join([]string{
		"This flow creates specs/ with prompt templates, settings, the runner script,",
		"and a sample spec so you can start executing specs immediately.",
		"",
		"Press Enter to get started, Esc to go back/quit, q to quit.",
	}, "\n")
	help := []components.HelpEntry{
		{Key: "enter", Label: "continue"},
		{Key: "esc/q", Label: "quit"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       width,
		Title:       components.TitleConfig{Title: "helm scaffold"},
		Body:        body,
		HelpEntries: help,
	})
}

func modeView(width int, modes []config.Mode, index int) string {
	items := make([]components.MenuItem, len(modes))
	for i, mode := range modes {
		desc := "Strict runs acceptance commands sequentially."
		if mode == config.ModeParallel {
			desc = "Parallel may fan out acceptance commands when safe."
		}
		items[i] = components.MenuItem{Title: string(mode), Description: desc}
	}
	list := components.MenuList(width, items, index)
	help := []components.HelpEntry{
		{Key: "↑/↓/tab", Label: "move"},
		{Key: "enter", Label: "select"},
		{Key: "esc", Label: "back"},
	}
	return components.PageShell(components.PageShellOptions{
		Width: width,
		Title: components.TitleConfig{Title: "Select workflow mode"},
		Body: strings.Join([]string{
			"Strict runs acceptance commands sequentially; Parallel may fan out where safe.",
			"",
			list,
		}, "\n"),
		HelpEntries: help,
	})
}

func commandsView(width int, existing []string, input string) string {
	summary := "(none yet)"
	if len(existing) > 0 {
		summary = components.BulletList(existing)
	}
	field := components.FormFieldView(components.FormField{
		Label:       "Enter acceptance command (blank + Enter to continue, ctrl+w remove last)",
		Value:       input,
		Description: "Commands run sequentially during strict mode.",
		Focused:     true,
	})
	help := []components.HelpEntry{
		{Key: "enter", Label: "add"},
		{Key: "ctrl+w", Label: "remove last"},
		{Key: "esc", Label: "back"},
	}
	body := strings.Join([]string{
		"Existing commands:",
		summary,
		"",
		field,
	}, "\n")
	return components.PageShell(components.PageShellOptions{
		Width:       width,
		Title:       components.TitleConfig{Title: "Acceptance commands"},
		Body:        body,
		HelpEntries: help,
	})
}

func optionsView(width int, specsInput string, focus int, errMsg string) string {
	field := components.FormFieldView(components.FormField{
		Label:       "Specs root",
		Value:       specsInput,
		Description: "Type to edit, Enter to continue",
		Focused:     focus == 0,
		Error:       errMsg,
	})
	help := []components.HelpEntry{
		{Key: "enter", Label: "continue"},
		{Key: "esc", Label: "back"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       width,
		Title:       components.TitleConfig{Title: "Optional settings"},
		Body:        field,
		HelpEntries: help,
	})
}

func confirmView(width int, answers innerscaffold.Answers) string {
	lines := []string{
		fmt.Sprintf("Mode: %s", answers.Mode),
		fmt.Sprintf("Specs root: %s", answers.SpecsRoot),
	}
	if len(answers.AcceptanceCommands) == 0 {
		lines = append(lines, "Acceptance commands: (none)")
	} else {
		lines = append(lines, "Acceptance commands:", components.BulletList(answers.AcceptanceCommands))
	}
	help := []components.HelpEntry{
		{Key: "enter", Label: "scaffold"},
		{Key: "esc", Label: "back"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       width,
		Title:       components.TitleConfig{Title: "Confirm scaffold"},
		Body:        strings.Join(lines, "\n\n"),
		HelpEntries: help,
	})
}

func runningView(width int, spin string) string {
	body := components.SpinnerLine(spin, "Creating workspace... (Esc/q to cancel)")
	help := []components.HelpEntry{
		{Key: "esc/q", Label: "cancel"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       width,
		Title:       components.TitleConfig{Title: "Scaffolding"},
		Body:        body,
		HelpEntries: help,
	})
}

func completeView(width int, result *innerscaffold.Result, err error) string {
	var body []string
	if err != nil {
		body = append(body, components.Flash(components.FlashDanger, fmt.Sprintf("Scaffold failed: %v", err)))
	} else if result != nil {
		body = append(body, fmt.Sprintf("Specs root: %s", result.SpecsRoot))
		if len(result.Created) > 0 {
			body = append(body, "Created:", components.BulletList(result.Created))
		}
		if len(result.Skipped) > 0 {
			body = append(body, "Skipped (already existed):", components.BulletList(result.Skipped))
		}
	} else {
		body = append(body, "Scaffold finished.")
	}
	help := []components.HelpEntry{
		{Key: "enter", Label: "exit"},
		{Key: "q", Label: "quit"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       width,
		Title:       components.TitleConfig{Title: "Scaffold complete"},
		Body:        strings.Join(body, "\n\n"),
		HelpEntries: help,
	})
}
