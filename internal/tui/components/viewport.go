package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/tui/theme"
)

// ViewportCardOptions configures the bordered viewport wrapper.
type ViewportCardOptions struct {
	Width        int
	Content      string
	Status       string
	Border       theme.BorderVariant
	NoWrap       bool   // when true, truncate instead of wrapping
	Tail         string // optional truncation tail, defaults to …
	Preformatted bool   // when true, render content as-is without re-wrapping
}

// ViewportCard renders log/graph viewports with a shared frame and footer.
func ViewportCard(opts ViewportCardOptions) string {
	width := ContentWidth(opts.Width)
	border := theme.BorderFor(theme.DefaultCardBorder)
	if opts.Border != "" {
		border = theme.BorderFor(opts.Border)
	}

	tail := opts.Tail
	if tail == "" {
		tail = "…"
	}

	bodyStyle := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Border(border).
		BorderForeground(theme.Colors.Border).
		Padding(0, 1)

	frameW, _ := bodyStyle.GetFrameSize()
	// Leave a one-cell safety margin inside the frame to avoid overflow that
	// can push the closing border to the next line in some terminals.
	bodyWidth := maxInt(1, width-frameW-1)

	content := strings.TrimRight(opts.Content, "\n")
	if !opts.Preformatted {
		content = FitStyledContent(content, bodyWidth, !opts.NoWrap, tail)
	}
	body := bodyStyle.Render(content)
	if strings.TrimSpace(opts.Status) == "" {
		return body
	}
	footerStyle := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Background(theme.Colors.Surface).
		Foreground(theme.Colors.Muted).
		Padding(0, 1)
	footerFrameW, _ := footerStyle.GetFrameSize()
	footerContentWidth := maxInt(1, width-footerFrameW-1)
	footerContent := FitStyledContent(opts.Status, footerContentWidth, false, "…")
	footer := footerStyle.Render(footerContent)
	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ViewportInnerWidth returns the usable inner width for content rendered inside
// a ViewportCard with the given total width and border variant. Callers should
// use this to size bubble viewport models so their View output fits without
// additional wrapping.
func ViewportInnerWidth(totalWidth int, border theme.BorderVariant) int {
	width := ContentWidth(totalWidth)
	borderStyle := theme.BorderFor(theme.DefaultCardBorder)
	if border != "" {
		borderStyle = theme.BorderFor(border)
	}
	bodyStyle := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Border(borderStyle).
		BorderForeground(theme.Colors.Border).
		Padding(0, 1)
	frameW, _ := bodyStyle.GetFrameSize()
	return maxInt(1, width-frameW-1)
}

// ViewportChromeHeight returns the vertical space consumed by a ViewportCard's
// frame (border + padding) and optional status footer, excluding any content
// lines. This lets callers compute the available height for the inner viewport
// content before rendering.
func ViewportChromeHeight(totalWidth int, border theme.BorderVariant, hasStatus bool) int {
	width := ContentWidth(totalWidth)
	borderStyle := theme.BorderFor(theme.DefaultCardBorder)
	if border != "" {
		borderStyle = theme.BorderFor(border)
	}
	bodyStyle := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Border(borderStyle).
		BorderForeground(theme.Colors.Border).
		Padding(0, 1)
	_, bodyFrameH := bodyStyle.GetFrameSize()

	footerH := 0
	if hasStatus {
		footerStyle := lipgloss.NewStyle().
			Width(width).
			MaxWidth(width).
			Background(theme.Colors.Surface).
			Foreground(theme.Colors.Muted).
			Padding(0, 1)
		_, footerFrameH := footerStyle.GetFrameSize()
		footerH = footerFrameH + 1 // one status line
	}

	return bodyFrameH + footerH
}
