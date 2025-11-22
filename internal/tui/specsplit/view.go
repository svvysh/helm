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
		"This command collects a large spec (paste or file) and streams Codex progress while it generates smaller Helm specs.",
		"",
		"Keys: Enter begin, q/Esc quit.",
	}, "\n") + "\n"
}

func inputView(m *model) string {
	lines := []string{
		"Paste the full spec below. Enter = split, Ctrl+O = load file (path in box), Ctrl+L = clear, q/Esc = quit.",
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

func runningView(m *model) string {
	status := "Splitting via Codex..."
	if m.opts.PlanPath != "" {
		status = fmt.Sprintf("Reading plan from %s...", m.opts.PlanPath)
	}
	lines := []string{
		fmt.Sprintf("%s %s", m.spinner.View(), status),
	}
	if m.resumeCmd != "" {
		lines = append(lines, fmt.Sprintf("Resume: %s  [c] copy", m.resumeCmd))
	}
	if m.flash != "" {
		lines = append(lines, m.flash)
	}
	lines = append(lines, "", m.vp.View())
	return strings.Join(lines, "\n")
}

func doneView(m *model) string {
	if m.err != nil {
		lines := []string{
			"Split failed:",
			m.err.Error(),
		}
		if m.resumeCmd != "" {
			lines = append(lines, fmt.Sprintf("Resume: %s  [c] copy", m.resumeCmd))
		}
		if m.flash != "" {
			lines = append(lines, m.flash)
		}
		if len(m.logs) > 0 {
			lines = append(lines, "", "Recent logs:")
			lines = append(lines, renderLogTail(m.logs, 15))
		}
		lines = append(lines, "", "Press Enter/q/Esc to exit, n to split another, r to jump to Run.")
		return strings.Join(lines, "\n") + "\n"
	}
	if m.result == nil {
		return "No specs were created. Press Enter/q/Esc to exit.\n"
	}

	lines := []string{
		"Spec split complete!",
		"",
		renderSummaryTable(m.result.Specs),
	}
	if m.resumeCmd != "" {
		lines = append(lines, fmt.Sprintf("Resume: %s  [c] copy", m.resumeCmd))
	}
	if m.flash != "" {
		lines = append(lines, m.flash)
	}
	if len(m.result.Warnings) > 0 {
		lines = append(lines, "", "Warnings:")
		for _, warn := range m.result.Warnings {
			lines = append(lines, fmt.Sprintf("- %s", warn))
		}
	}
	if len(m.logs) > 0 {
		lines = append(lines, "", "Recent logs:", renderLogTail(m.logs, 15))
	}
	lines = append(lines, "", "Keys: Enter/q/Esc exit, r jump to Run, n split another.")
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

func renderLogTail(lines []string, max int) string {
	if len(lines) == 0 {
		return "(no logs)"
	}
	if len(lines) > max {
		lines = lines[len(lines)-max:]
	}
	return strings.Join(lines, "\n")
}
