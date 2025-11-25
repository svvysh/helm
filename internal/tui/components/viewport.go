package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/tui/theme"
)

// ViewportCardOptions configures the bordered viewport wrapper.
type ViewportCardOptions struct {
	Width   int
	Content string
	Status  string
}

// ViewportCard renders log/graph viewports with a shared frame and footer.
func ViewportCard(opts ViewportCardOptions) string {
	width := contentWidth(opts.Width)
	body := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.Colors.Border).
		Padding(0, 1).
		Render(strings.TrimRight(opts.Content, "\n"))
	if strings.TrimSpace(opts.Status) == "" {
		return body
	}
	footer := lipgloss.NewStyle().
		Width(width).
		Background(theme.Colors.Surface).
		Foreground(theme.Colors.Muted).
		Padding(0, 1).
		Render(opts.Status)
	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}
