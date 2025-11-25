package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/tui/theme"
)

// FlashKind enumerates flash banner severity levels.
type FlashKind int

const (
	FlashNone FlashKind = iota
	FlashInfo
	FlashSuccess
	FlashWarning
	FlashDanger
)

// Flash renders a single-line banner highlighting ephemeral state.
func Flash(kind FlashKind, message string) string {
	msg := strings.TrimSpace(message)
	if msg == "" || kind == FlashNone {
		return ""
	}
	style := lipgloss.NewStyle().Padding(0, 1).Bold(true)
	switch kind {
	case FlashSuccess:
		style = style.Background(theme.Colors.Success).Foreground(theme.Colors.Surface)
	case FlashWarning:
		style = style.Background(theme.Colors.Highlight).Foreground(theme.Colors.Border)
	case FlashDanger:
		style = style.Background(theme.Colors.Warning).Foreground(theme.Colors.Surface)
	default:
		style = style.Background(theme.Colors.Surface).Foreground(theme.Colors.Accent)
	}
	return style.Render(msg)
}

// NewSpinner returns a Glow-styled dot spinner for inline status lines.
func NewSpinner() spinner.Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Colors.Accent)
	return sp
}

// SpinnerLine renders an inline spinner with descriptive text.
func SpinnerLine(spinnerView, text string) string {
	if strings.TrimSpace(text) == "" {
		return spinnerView
	}
	return lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Foreground(theme.Colors.Accent).Render(spinnerView),
		theme.BodyStyle.Render(" "+text),
	)
}

// ResumeChip renders a reusable command pill with a copy hint.
func ResumeChip(cmd string) string {
	if strings.TrimSpace(cmd) == "" {
		return ""
	}
	chip := lipgloss.NewStyle().
		Background(theme.Colors.Surface).
		Foreground(theme.Colors.Border).
		Padding(0, 1).
		Bold(true).
		Render(cmd)
	hint := theme.HintStyle.Render(" [c] copy")
	return fmt.Sprintf("Resume %s%s", chip, hint)
}
