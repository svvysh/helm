package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/polarzero/helm/internal/tui/theme"
)

// SummaryBar renders status counts as shared badges.
func SummaryBar(counts map[theme.StatusCategory]int) string {
	order := []theme.StatusCategory{
		theme.StatusTodo,
		theme.StatusInProgress,
		theme.StatusDone,
		theme.StatusBlocked,
		theme.StatusFailed,
	}
	parts := make([]string, 0, len(order))
	for _, cat := range order {
		label := statusLabel(cat)
		style := statusStyle(cat)
		parts = append(parts, style.Render(fmt.Sprintf("%s %d", label, counts[cat])))
	}
	return strings.Join(parts, "  ")
}

func statusLabel(cat theme.StatusCategory) string {
	switch cat {
	case theme.StatusDone:
		return "DONE"
	case theme.StatusInProgress:
		return "IN PROGRESS"
	case theme.StatusBlocked:
		return "BLOCKED"
	case theme.StatusFailed:
		return "FAILED"
	default:
		return "TODO"
	}
}

func statusStyle(cat theme.StatusCategory) lipgloss.Style {
	switch cat {
	case theme.StatusDone:
		return theme.BadgeDone
	case theme.StatusInProgress:
		return theme.BadgeProgress
	case theme.StatusBlocked:
		return theme.BadgeBlocked
	case theme.StatusFailed:
		return theme.BadgeFailed
	default:
		return theme.BadgeTodo
	}
}

// SummaryTableData describes rows for the monospace summary table.
type SummaryTableData struct {
	Headers []string
	Rows    [][]string
}

// SummaryTable renders a width-aware monospace table with header underline.
func SummaryTable(data SummaryTableData) string {
	if len(data.Headers) == 0 {
		return ""
	}
	widths := make([]int, len(data.Headers))
	for i, header := range data.Headers {
		widths[i] = runewidth.StringWidth(header)
	}
	for _, row := range data.Rows {
		for col, cell := range row {
			if col >= len(widths) {
				continue
			}
			if w := runewidth.StringWidth(cell); w > widths[col] {
				widths[col] = w
			}
		}
	}
	var b strings.Builder
	// Header
	for i, header := range data.Headers {
		fmt.Fprintf(&b, "%-*s", widths[i]+2, header)
	}
	b.WriteString("\n")
	// Divider
	total := 0
	for _, w := range widths {
		total += w + 2
	}
	b.WriteString(strings.Repeat("-", max(total, 0)))
	b.WriteString("\n")
	// Rows
	for _, row := range data.Rows {
		for col, cell := range row {
			if col >= len(widths) {
				continue
			}
			fmt.Fprintf(&b, "%-*s", widths[col]+2, cell)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// TableStyles customizes Bubble table colors to match the shared palette.
func TableStyles() table.Styles {
	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(theme.Colors.Border).
		Bold(true).
		Foreground(theme.Colors.Primary).
		Padding(0, 1)
	styles.Cell = styles.Cell.
		Foreground(theme.Colors.Primary).
		Padding(0, 1)
	styles.Selected = styles.Selected.
		Foreground(theme.Colors.Surface).
		Background(theme.Colors.Accent).
		Bold(true)
	return styles
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
