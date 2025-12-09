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

// ViewWidth clamps a provided width to a sensible default when the terminal
// reports zero during initialization.
func ViewWidth(width int) int {
	return viewWidth(width)
}

// ContentWidth returns the usable width once horizontal padding is removed and
// enforces a small minimum so cards don’t collapse.
func ContentWidth(width int) int {
	w := ViewWidth(width) - theme.ViewHorizontalPadding*2
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

// ClampHeight trims the rendered view to at most the given height. This
// prevents oversized layouts from pushing important chrome (like spinners) off
// screen on small terminals.
func ClampHeight(view string, maxHeight int) string {
	if maxHeight <= 0 {
		return view
	}
	lines := strings.Split(view, "\n")
	if len(lines) <= maxHeight {
		return view
	}
	return strings.Join(lines[:maxHeight], "\n")
}
