package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/tui/theme"
)

func viewWidth(width int) int {
	if width <= 0 {
		return 80
	}
	return width
}

func contentWidth(width int) int {
	w := viewWidth(width) - theme.ViewHorizontalPadding*2
	if w < 24 {
		return 24
	}
	return w
}

// PadToHeight ensures the rendered view fills at least the given height, so
// Bubble Tea doesn't leave artifacts from previous, taller frames.
func PadToHeight(view string, minHeight int) string {
	if minHeight <= 0 {
		return view
	}
	height := lipgloss.Height(view)
	if height >= minHeight {
		return view
	}
	padding := strings.Repeat("\n", minHeight-height)
	return view + padding
}
