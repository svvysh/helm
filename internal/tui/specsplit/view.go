package specsplit

import (
	"fmt"
	"strings"

	splitting "github.com/polarzero/helm/internal/specsplit"
)

func introView() string {
	return strings.Join([]string{
		"helm spec — split large specs",
		"",
		"This command collects a large spec (paste or file) and asks Codex to split it into smaller Helm specs.",
		"You'll preview the first chunk, confirm, and then Helm will generate new spec folders under specs/.",
		"",
		"Press Enter to begin or Ctrl+C to exit.",
	}, "\n") + "\n"
}

func inputView(m *model) string {
	lines := []string{
		"Paste the full spec below. Press Ctrl+D when you're ready to preview, Ctrl+C to cancel.",
	}
	if m.opts.PlanPath != "" {
		lines = append(lines, fmt.Sprintf("Dev mode: plan will be loaded from %s", m.opts.PlanPath))
	}
	if m.inputErr != "" {
		lines = append(lines, "", fmt.Sprintf("Error: %s", m.inputErr))
	}
	lines = append(lines, "", m.ta.View())
	return strings.Join(lines, "\n") + "\n"
}

func previewView(m *model) string {
	header := []string{
		"Previewing spec (first ~60 lines).",
		"Enter = split, b = edit, Esc = edit.",
	}
	if m.opts.PlanPath != "" {
		header = append(header, fmt.Sprintf("Plan source: %s", m.opts.PlanPath))
	} else if m.opts.CodexChoice.Model != "" {
		detail := fmt.Sprintf("Codex model: %s", m.opts.CodexChoice.Model)
		if m.opts.CodexChoice.Reasoning != "" {
			detail = fmt.Sprintf("%s (reasoning %s)", detail, m.opts.CodexChoice.Reasoning)
		}
		header = append(header, detail)
	}
	header = append(header, "")
	preview := strings.Join(m.preview, "\n")
	return strings.Join(append(header, preview), "\n") + "\n"
}

func runningView(m *model) string {
	status := "Requesting Codex split plan..."
	if m.opts.PlanPath != "" {
		status = fmt.Sprintf("Reading plan from %s...", m.opts.PlanPath)
	}
	if m.opts.CodexChoice.Model != "" && m.opts.PlanPath == "" {
		status = fmt.Sprintf("Codex %s split in progress...", m.opts.CodexChoice.Model)
	}
	return fmt.Sprintf("%s %s\n\nPress Ctrl+C to abort.", m.spinner.View(), status)
}

func doneView(m *model) string {
	if m.err != nil {
		return strings.Join([]string{
			"Split failed:",
			m.err.Error(),
			"",
			"Press Enter to exit.",
		}, "\n") + "\n"
	}
	if m.result == nil {
		return "No specs were created. Press Enter to exit.\n"
	}

	lines := []string{
		"Spec split complete!",
		"",
		renderSummaryTable(m.result.Specs),
	}
	if len(m.result.Warnings) > 0 {
		lines = append(lines, "", "Warnings:")
		for _, warn := range m.result.Warnings {
			lines = append(lines, fmt.Sprintf("- %s", warn))
		}
	}
	lines = append(lines, "", "Press Enter or q to exit.")
	return strings.Join(lines, "\n") + "\n"
}

func renderSummaryTable(specs []splitting.GeneratedSpec) string {
	if len(specs) == 0 {
		return "(no specs created)"
	}
	idWidth := len("Spec ID")
	nameWidth := len("Name")
	for _, spec := range specs {
		if l := len(spec.ID); l > idWidth {
			idWidth = l
		}
		if l := len(spec.Name); l > nameWidth {
			nameWidth = l
		}
	}
	header := fmt.Sprintf("%-*s  %-*s  %s", idWidth, "Spec ID", nameWidth, "Name", "Depends on")
	divider := strings.Repeat("-", len(header))
	rows := []string{header, divider}
	for _, spec := range specs {
		deps := strings.Join(spec.DependsOn, ", ")
		rows = append(rows, fmt.Sprintf("%-*s  %-*s  %s", idWidth, spec.ID, nameWidth, spec.Name, deps))
	}
	return strings.Join(rows, "\n")
}
