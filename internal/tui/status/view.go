package status

import (
	"fmt"
	"strings"

	"github.com/polarzero/helm/internal/tui/components"
	"github.com/polarzero/helm/internal/tui/theme"
)

func renderView(m *model) string {
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
	if m.infoMessage != "" {
		body = append(body, theme.HintStyle.Render(m.infoMessage))
	}

	if m.viewMode == viewTable {
		body = append(body, m.table.View())
	} else {
		body = append(body, components.ViewportCard(components.ViewportCardOptions{
			Width:   m.width,
			Content: m.graphViewport.View(),
			Status:  fmt.Sprintf("%d specs shown", len(m.visible)),
		}))
	}

	help := []components.HelpEntry{
		{Key: "tab", Label: "toggle view"},
		{Key: "f", Label: "cycle focus"},
		{Key: "enter", Label: "set subtree"},
		{Key: "r", Label: "reload"},
		{Key: "q", Label: "back"},
	}

	return components.PageShell(components.PageShellOptions{
		Width:       m.width,
		Title:       components.TitleConfig{Title: fmt.Sprintf("Status overview — %s", viewLabel(m.viewMode))},
		Body:        strings.Join(body, "\n\n"),
		HelpEntries: help,
	})
}

func viewLabel(mode viewMode) string {
	switch mode {
	case viewGraph:
		return "Graph"
	default:
		return "Table"
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
