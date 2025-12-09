package components

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// FitStyledContent wraps or truncates ANSI-styled text to the target width
// using wcwidth/grapheme aware helpers. When wrap is false, tail is appended
// to indicate truncation.
func FitStyledContent(text string, width int, wrap bool, tail string) string {
	if width <= 0 {
		return strings.TrimRight(text, "\n")
	}
	if tail == "" {
		tail = "…"
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if wrap {
			wrapped := ansi.Hardwrap(line, width, false)
			out = append(out, strings.Split(wrapped, "\n")...)
			continue
		}
		out = append(out, ansi.Truncate(line, width, tail))
	}
	return strings.Join(out, "\n")
}

// StyledWidth returns the visible cell width of a string that may contain ANSI
// sequences.
func StyledWidth(text string) int {
	return ansi.StringWidth(text)
}
