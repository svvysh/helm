package status

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/tui/theme"
)

func renderView(m *model) string {
	var b strings.Builder

	title := theme.TitleStyle.Render(fmt.Sprintf("Status overview — %s view", viewLabel(m.viewMode)))
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(renderSummaryLine(m.summary))
	b.WriteString("\n")
	b.WriteString(theme.HintStyle.Render(focusLine(m)))
	if m.infoMessage != "" {
		b.WriteString("\n")
		b.WriteString(theme.HintStyle.Render(m.infoMessage))
	}
	b.WriteString("\n\n")

	if m.viewMode == viewTable {
		b.WriteString(m.table.View())
	} else {
		b.WriteString(m.graphViewport.View())
	}

	b.WriteString("\n\n")
	b.WriteString(theme.HintStyle.Render("[tab] toggle graph/table  [f] cycle focus  [enter] set subtree  [r] reload  [q] back"))
	return b.String()
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

func renderSummaryLine(sum summaryCounts) string {
	var parts []string
	parts = append(parts, badge(theme.StatusTodo, sum.counts[theme.StatusTodo]))
	parts = append(parts, badge(theme.StatusInProgress, sum.counts[theme.StatusInProgress]))
	parts = append(parts, badge(theme.StatusDone, sum.counts[theme.StatusDone]))
	parts = append(parts, badge(theme.StatusBlocked, sum.counts[theme.StatusBlocked]))
	parts = append(parts, badge(theme.StatusFailed, sum.counts[theme.StatusFailed]))
	return strings.Join(parts, "  ")
}

func badge(category theme.StatusCategory, count int) string {
	var style lipgloss.Style
	var label string

	switch category {
	case theme.StatusDone:
		style, label = theme.BadgeDone, "DONE"
	case theme.StatusInProgress:
		style, label = theme.BadgeProgress, "IN PROGRESS"
	case theme.StatusBlocked:
		style, label = theme.BadgeBlocked, "BLOCKED"
	case theme.StatusFailed:
		style, label = theme.BadgeFailed, "FAILED"
	default:
		style, label = theme.BadgeTodo, "TODO"
	}
	return style.Render(fmt.Sprintf("%s %d", label, count))
}
