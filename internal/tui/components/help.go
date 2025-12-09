package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/polarzero/helm/internal/tui/theme"
)

// HelpEntry captures a key/label pair rendered in the shared help bar.
type HelpEntry struct {
	Key   string
	Label string
}

var (
	helpKeyStyle   = lipgloss.NewStyle().Foreground(theme.Colors.Highlight).Bold(true)
	helpValueStyle = lipgloss.NewStyle().Foreground(theme.Colors.Muted)
	helpDivider    = lipgloss.NewStyle().Foreground(theme.Colors.Muted).Render(" • ")
)

// HelpBar renders a single-line key legend, truncating to the provided width.
func HelpBar(width int, entries ...HelpEntry) string {
	if len(entries) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(entries))
	for _, entry := range entries {
		key := strings.TrimSpace(entry.Key)
		label := strings.TrimSpace(entry.Label)
		if key == "" && label == "" {
			continue
		}
		part := helpKeyStyle.Render(key)
		if label != "" {
			part = lipgloss.JoinHorizontal(lipgloss.Left, part, helpValueStyle.Render(" "+label))
		}
		pairs = append(pairs, part)
	}
	if len(pairs) == 0 {
		return ""
	}
	line := strings.Join(pairs, helpDivider)
	maxWidth := width
	if maxWidth <= 0 {
		maxWidth = ViewWidth(width) - theme.ViewHorizontalPadding*2
	}
	if runewidth.StringWidth(line) > maxWidth {
		line = runewidth.Truncate(line, maxWidth, "…")
	}
	return line
}
