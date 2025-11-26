package status

import (
	"fmt"
	"strings"

	"github.com/polarzero/helm/internal/tui/components"
	"github.com/polarzero/helm/internal/tui/theme"
)

func renderView(m *model) string {
	opts := m.pageShellOptions(m.graphViewport.View())
	return components.PageShell(opts)
}

func (m *model) pageShellOptions(graphContent string) components.PageShellOptions {
	body := []string{
		components.SummaryBar(m.summary.counts),
		components.SummaryTable(components.SummaryTableData{
			Headers: []string{"Status", "Count"},
			Rows: [][]string{
				{"TODO", fmt.Sprintf("%d", m.summary.counts[theme.StatusTodo])},
				{"IN PROGRESS", fmt.Sprintf("%d", m.summary.counts[theme.StatusInProgress])},
				{"DONE", fmt.Sprintf("%d", m.summary.counts[theme.StatusDone])},
				{"BLOCKED", fmt.Sprintf("%d", m.summary.counts[theme.StatusBlocked])},
				{"FAILED", fmt.Sprintf("%d", m.summary.counts[theme.StatusFailed])},
			},
		}),
		theme.HintStyle.Render(focusLine(m)),
	}

	if sel := m.currentSelectionID(); sel != "" {
		body = append(body, theme.HintStyle.Render(fmt.Sprintf("Selection: %s (↑/↓ to move, enter to focus subtree)", sel)))
	}
	if m.infoMessage != "" {
		body = append(body, theme.HintStyle.Render(m.infoMessage))
	}

	body = append(body, components.ViewportCard(components.ViewportCardOptions{
		Width:   m.width,
		Content: graphContent,
		Status:  fmt.Sprintf("%d specs shown — PgUp/PgDn to scroll", len(m.visible)),
	}))

	if sel := m.currentSelectionID(); sel != "" {
		if entry, ok := m.entryByID[sel]; ok && entry != nil {
			body = append(body, selectedDetails(entry))
		}
	}

	help := []components.HelpEntry{
		{Key: "↑/↓", Label: "move selection"},
		{Key: "enter", Label: "focus subtree"},
		{Key: "f", Label: "cycle focus"},
		{Key: "r", Label: "reload"},
		{Key: "q", Label: "quit"},
		{Key: "esc", Label: "back"},
	}

	return components.PageShellOptions{
		Width:       m.width,
		Title:       components.TitleConfig{Title: "Status overview — Graph"},
		Body:        strings.Join(body, "\n\n"),
		HelpEntries: help,
	}
}

func focusLine(m *model) string {
	switch m.focusMode {
	case focusRunnable:
		return fmt.Sprintf("Focus: Runnable specs (%d shown)", len(m.visible))
	case focusSubtree:
		target := m.focusTarget
		if target == "" {
			return fmt.Sprintf("Focus: Subtree (no spec selected) — %d shown", len(m.visible))
		}
		name := ""
		if entry, ok := m.entryByID[target]; ok {
			name = entry.Name
		}
		if name != "" {
			return fmt.Sprintf("Focus: Subtree of %s — %s (%d shown)", target, name, len(m.visible))
		}
		return fmt.Sprintf("Focus: Subtree of %s (%d shown)", target, len(m.visible))
	default:
		return fmt.Sprintf("Focus: All specs (%d total)", len(m.entries))
	}
}

func selectedDetails(e *entry) string {
	if e == nil {
		return ""
	}
	deps := "-"
	if len(e.DependsOn) > 0 {
		deps = strings.Join(e.DependsOn, ", ")
	}
	dependents := "-"
	if len(e.Dependents) > 0 {
		dependents = strings.Join(e.Dependents, ", ")
	}
	rows := [][]string{
		{"ID", e.ID},
		{"Name", e.Name},
		{"Status", e.BadgeStyle.Render(e.BadgeText)},
		{"Last run", e.LastRunDisplay},
		{"Depends on", deps},
		{"Dependents", dependents},
	}
	detail := components.SummaryTable(components.SummaryTableData{
		Headers: []string{"Field", "Value"},
		Rows:    rows,
	})
	var extra []string
	if e.HasUnmetDeps && e.BlockReason != "" {
		extra = append(extra, components.Flash(components.FlashWarning, fmt.Sprintf("Unmet deps: %s", e.BlockReason)))
	}
	if len(extra) == 0 {
		return detail
	}
	return strings.Join(append([]string{detail}, extra...), "\n\n")
}
