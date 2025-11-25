package components

import "github.com/polarzero/helm/internal/tui/theme"

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
