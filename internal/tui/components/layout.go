package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/tui/theme"
)

// TitleConfig controls how a TitleBar renders.
type TitleConfig struct {
	Title    string
	Subtitle string
}

// TitleBar renders a bold title aligned with an optional subtitle label.
func TitleBar(cfg TitleConfig) string {
	title := strings.TrimSpace(cfg.Title)
	if title == "" {
		return ""
	}
	rendered := theme.TitleStyle.Render(title)
	if strings.TrimSpace(cfg.Subtitle) == "" {
		return rendered
	}
	label := theme.SubtitleStyle.Render(" " + cfg.Subtitle)
	divider := lipgloss.NewStyle().Foreground(theme.Colors.Muted).Render(" — ")
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered, divider, label)
}

// PageShellOptions configures the shared page wrapper.
type PageShellOptions struct {
	Width       int
	Title       TitleConfig
	Body        string
	HelpEntries []HelpEntry
}

// PageShell wraps a title, body, and help bar with shared padding and spacing.
func PageShell(opts PageShellOptions) string {
	width := viewWidth(opts.Width)
	sections := make([]string, 0, 3)
	if title := strings.TrimSpace(TitleBar(opts.Title)); title != "" {
		sections = append(sections, title)
	}
	body := strings.TrimRight(opts.Body, "\n")
	if strings.TrimSpace(body) != "" {
		sections = append(sections, body)
	}
	if len(opts.HelpEntries) > 0 {
		help := HelpBar(contentWidth(width), opts.HelpEntries...)
		if help != "" {
			sections = append(sections, help)
		}
	}
	content := strings.Join(sections, "\n\n")
	container := lipgloss.NewStyle().
		Width(width).
		PaddingLeft(theme.ViewHorizontalPadding).
		PaddingRight(theme.ViewHorizontalPadding).
		PaddingTop(theme.ViewTopPadding).
		PaddingBottom(theme.ViewBottomPadding)
	return container.Render(content)
}

// ModalConfig defines the body of a confirmation modal.
type ModalConfig struct {
	Width int
	Title string
	Body  []string
}

// Modal renders a warning/danger modal using Glow-inspired colors.
func Modal(cfg ModalConfig) string {
	if cfg.Title == "" && len(cfg.Body) == 0 {
		return ""
	}
	width := contentWidth(cfg.Width)
	header := lipgloss.NewStyle().
		Background(theme.Colors.Warning).
		Foreground(theme.Colors.Surface).
		Bold(true).
		Padding(0, 1).
		Render(strings.TrimSpace(cfg.Title))

	body := strings.Join(cfg.Body, "\n")
	bodyStyle := lipgloss.NewStyle().
		Width(width-4).
		Padding(1, 2).
		Background(theme.Colors.Surface).
		Foreground(theme.Colors.Border)

	panel := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Colors.Warning).
		Align(lipgloss.Left)

	return panel.Render(lipgloss.JoinVertical(lipgloss.Left, header, bodyStyle.Render(body)))
}
