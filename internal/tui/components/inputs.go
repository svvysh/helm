package components

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/tui/theme"
)

// NewTextInput returns a text input configured with the shared palette.
func NewTextInput() textinput.Model {
	input := textinput.New()
	ApplyTextInputStyle(&input)
	return input
}

// ApplyTextInputStyle mutates a text input with Glow-derived colors.
func ApplyTextInputStyle(input *textinput.Model) {
	if input == nil {
		return
	}
	promptStyle := lipgloss.NewStyle().Foreground(theme.Colors.Highlight)
	cursorStyle := lipgloss.NewStyle().Foreground(theme.Colors.Accent)
	input.PromptStyle = promptStyle
	input.Cursor.Style = cursorStyle
	input.TextStyle = theme.BodyStyle
	input.PlaceholderStyle = theme.HintStyle
}

// NewTextarea returns a textarea model with shared borders and colors.
func NewTextarea() textarea.Model {
	ta := textarea.New()
	ApplyTextareaStyle(&ta)
	return ta
}

// ApplyTextareaStyle mutates a textarea so borders/cursor match the theme.
func ApplyTextareaStyle(ta *textarea.Model) {
	if ta == nil {
		return
	}
	borderFocused := lipgloss.NewStyle().BorderForeground(theme.Colors.Accent).BorderStyle(lipgloss.RoundedBorder())
	borderBlurred := lipgloss.NewStyle().BorderForeground(theme.Colors.Border).BorderStyle(lipgloss.NormalBorder())

	ta.FocusedStyle.Base = borderFocused
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(theme.Colors.Surface)
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(theme.Colors.Highlight)
	ta.FocusedStyle.Text = theme.BodyStyle

	ta.BlurredStyle.Base = borderBlurred
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle().Background(theme.Colors.Surface)
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(theme.Colors.Muted)
	ta.BlurredStyle.Text = theme.BodyStyle
}
