package scaffold

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/config"
	innerscaffold "github.com/polarzero/helm/internal/scaffold"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	inputLabel    = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
)

func introView() string {
	lines := []string{
		titleStyle.Render("helm scaffold"),
		"",
		"This flow will create specs/ with prompt templates, settings, the runner script,",
		"and a sample spec so you can start executing specs immediately.",
		"",
		"Press Enter to get started or Esc to cancel.",
	}
	return strings.Join(lines, "\n")
}

func modeView(modes []config.Mode, index int) string {
	lines := []string{titleStyle.Render("Select workflow mode"), ""}
	for i, mode := range modes {
		text := string(mode)
		label := fmt.Sprintf("  %s", text)
		if i == index {
			label = selectedStyle.Render("▶ " + text)
		}
		lines = append(lines, label)
	}
	lines = append(lines, "", "Use ↑/↓ or tab to move, Enter to confirm.")
	return strings.Join(lines, "\n")
}

func commandsView(existing []string, input string) string {
	lines := []string{titleStyle.Render("Acceptance commands"), ""}
	if len(existing) == 0 {
		lines = append(lines, "(none yet)")
	} else {
		for _, cmd := range existing {
			lines = append(lines, fmt.Sprintf("- %s", cmd))
		}
	}
	lines = append(lines, "", inputLabel.Render("Enter command (blank + Enter to continue, ctrl+w to remove last):"), input)
	return strings.Join(lines, "\n")
}

func optionsView(specsInput string, focus int, makeGraph bool, errMsg string) string {
	checkbox := "[ ]"
	if makeGraph {
		checkbox = "[x]"
	}
	rootLabel := "Specs root"
	graphLabel := "Generate sample dependency graph"
	if focus == 0 {
		rootLabel = selectedStyle.Render(rootLabel)
	} else {
		graphLabel = selectedStyle.Render(graphLabel)
	}
	lines := []string{titleStyle.Render("Optional settings"), "", fmt.Sprintf("%s:", rootLabel), specsInput, "", fmt.Sprintf("%s %s", checkbox, graphLabel)}
	if errMsg != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(errMsg))
	}
	lines = append(lines, "", "Use tab to switch fields, space to toggle, Enter to continue.")
	return strings.Join(lines, "\n")
}

func confirmView(answers innerscaffold.Answers) string {
	lines := []string{titleStyle.Render("Confirm scaffold"), ""}
	lines = append(lines, fmt.Sprintf("Mode: %s", answers.Mode))
	lines = append(lines, fmt.Sprintf("Specs root: %s", answers.SpecsRoot))
	if len(answers.AcceptanceCommands) == 0 {
		lines = append(lines, "Acceptance commands: (none)")
	} else {
		lines = append(lines, "Acceptance commands:")
		for _, cmd := range answers.AcceptanceCommands {
			lines = append(lines, fmt.Sprintf("- %s", cmd))
		}
	}
	if answers.GenerateSampleGraph {
		lines = append(lines, "Sample dependency graph: yes")
	} else {
		lines = append(lines, "Sample dependency graph: no")
	}
	lines = append(lines, "", "Press Enter to scaffold or Esc to cancel.")
	return strings.Join(lines, "\n")
}

func runningView(spin string) string {
	return fmt.Sprintf("%s Creating workspace...", spin)
}

func completeView(result *innerscaffold.Result, err error) string {
	if err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Scaffold failed: %v", err))
	}
	if result == nil {
		return "Scaffold finished."
	}
	lines := []string{
		titleStyle.Render("Scaffold complete"), "",
		fmt.Sprintf("Specs root: %s", result.SpecsRoot), "",
	}
	if len(result.Created) > 0 {
		lines = append(lines, "Created:")
		for _, path := range result.Created {
			lines = append(lines, fmt.Sprintf("- %s", path))
		}
		lines = append(lines, "")
	}
	if len(result.Skipped) > 0 {
		lines = append(lines, "Skipped (already existed):")
		for _, path := range result.Skipped {
			lines = append(lines, fmt.Sprintf("- %s", path))
		}
		lines = append(lines, "")
	}
	lines = append(lines, "Press Enter to exit.")
	return strings.Join(lines, "\n")
}
