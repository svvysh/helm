package components

import (
	"strings"

	"github.com/polarzero/helm/internal/tui/theme"
)

// FormField describes a labeled value block within settings/scaffold forms.
type FormField struct {
	Label       string
	Value       string
	Focused     bool
	Description string
	Error       string
}

// FormFieldView renders the label/value/error stack for a single field.
func FormFieldView(field FormField) string {
	if strings.TrimSpace(field.Label) == "" {
		return strings.TrimSpace(field.Value)
	}
	cursor := "  "
	labelStyle := theme.HintStyle
	if field.Focused {
		cursor = "▶ "
		labelStyle = theme.SelectedStyle
	}
	lines := []string{labelStyle.Render(cursor + field.Label)}
	if strings.TrimSpace(field.Value) != "" {
		lines = append(lines, theme.BodyStyle.Render(field.Value))
	}
	if field.Description != "" {
		lines = append(lines, theme.HintStyle.Render(field.Description))
	}
	if field.Error != "" {
		lines = append(lines, theme.WarningStyle.Render(field.Error))
	}
	return strings.Join(lines, "\n")
}
