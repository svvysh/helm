package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/tui/theme"
)

// MenuItem describes a two-line vertical selection entry.
type MenuItem struct {
	Title       string
	Description string
}

// MenuList renders a vertically stacked list with shared cursor styling.
func MenuList(width int, items []MenuItem, selected int) string {
	if len(items) == 0 {
		return ""
	}
	bodyWidth := contentWidth(width)
	lines := make([]string, 0, len(items)*2)
	for i, item := range items {
		cursor := "  "
		lineStyle := theme.BodyStyle
		if i == selected {
			cursor = "▶ "
			lineStyle = theme.SelectedStyle
		}
		title := fmt.Sprintf("%s%s", cursor, strings.TrimSpace(item.Title))
		lines = append(lines, lineStyle.Render(title))
		if desc := strings.TrimSpace(item.Description); desc != "" {
			lines = append(lines, theme.HintStyle.Width(bodyWidth).Render("   "+desc))
		}
	}
	return strings.Join(lines, "\n")
}

// SpecListItemViewModel describes the data required to render a spec row.
type SpecListItemViewModel struct {
	BadgeText  string
	BadgeStyle lipgloss.Style
	ID         string
	Name       string
	Summary    string
	LastRun    string
	Selected   bool
	Warning    bool
}

// SpecListItem renders a two-line spec list row.
func SpecListItem(vm SpecListItemViewModel) string {
	cursor := "  "
	headerStyle := theme.BodyStyle
	summaryStyle := theme.HintStyle
	lastStyle := theme.HintStyle
	if vm.Selected {
		cursor = "▶ "
		// Match status selector: highlight with bold text but keep background unchanged.
		headerStyle = theme.BodyStyle.Bold(true)
	}
	if vm.Warning {
		summaryStyle = theme.WarningStyle
	}

	header := headerStyle.Render(fmt.Sprintf("%s%s — %s", cursor+vm.BadgeStyle.Render(vm.BadgeText), vm.ID, vm.Name))
	summary := summaryStyle.Render(fmt.Sprintf("%s%s", strings.Repeat(" ", len(cursor)), vm.Summary))
	last := lastStyle.Render(fmt.Sprintf("%s%s", strings.Repeat(" ", len(cursor)), vm.LastRun))
	return strings.Join([]string{header, summary, last}, "\n")
}

// BulletList renders lines prefixed with an accent bullet.
func BulletList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	bullet := theme.SubtitleStyle.Render("•")
	lines := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s", bullet, theme.BodyStyle.Render(item)))
	}
	return strings.Join(lines, "\n")
}
