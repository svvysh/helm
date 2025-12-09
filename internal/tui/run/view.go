package run

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/polarzero/helm/internal/specs"
	"github.com/polarzero/helm/internal/tui/components"
	"github.com/polarzero/helm/internal/tui/theme"
)

type specItem struct {
	folder *specs.SpecFolder
}

func (s specItem) Title() string {
	if s.folder == nil || s.folder.Metadata == nil {
		return ""
	}
	return fmt.Sprintf("%s — %s", s.folder.Metadata.ID, s.folder.Metadata.Name)
}

func (s specItem) Description() string {
	if s.folder == nil {
		return ""
	}
	return dependencySummary(s.folder)
}

func (s specItem) FilterValue() string {
	if s.folder == nil || s.folder.Metadata == nil {
		return ""
	}
	return s.folder.Metadata.ID + " " + s.folder.Metadata.Name
}

func specItemsFromFolders(folders []*specs.SpecFolder) []list.Item {
	return filterItems(folders, false)
}

func filterItems(folders []*specs.SpecFolder, runnableOnly bool) []list.Item {
	if len(folders) == 0 {
		return []list.Item{}
	}
	items := make([]list.Item, 0, len(folders))
	for _, folder := range folders {
		if folder == nil {
			continue
		}
		if runnableOnly && !folder.CanRun {
			continue
		}
		items = append(items, specItem{folder: folder})
	}
	if len(items) == 0 {
		return []list.Item{}
	}
	return items
}

func newDelegate() list.ItemDelegate {
	return specDelegate{}
}

type specDelegate struct{}

func (specDelegate) Height() int { return 3 }

func (specDelegate) Spacing() int { return 1 }

func (specDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (specDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	spec, ok := listItem.(specItem)
	if !ok {
		return
	}
	view := renderSpecItem(spec.folder, index == m.Index())
	fmt.Fprint(w, view)
}

func renderSpecItem(folder *specs.SpecFolder, selected bool) string {
	if folder == nil || folder.Metadata == nil {
		return ""
	}
	badgeText, badgeStyle, _ := theme.StatusBadge(folder.Metadata, len(folder.UnmetDeps) > 0)
	vm := components.SpecListItemViewModel{
		BadgeText:  badgeText,
		BadgeStyle: badgeStyle,
		ID:         folder.Metadata.ID,
		Name:       folder.Metadata.Name,
		Summary:    dependencySummary(folder),
		LastRun:    lastRunSummary(folder),
		Selected:   selected,
		Warning:    len(folder.UnmetDeps) > 0,
	}
	return components.SpecListItem(vm)
}

func dependencySummary(folder *specs.SpecFolder) string {
	if folder == nil || folder.Metadata == nil {
		return ""
	}
	if len(folder.UnmetDeps) > 0 {
		return fmt.Sprintf("Unmet deps: %s", strings.Join(folder.UnmetDeps, ", "))
	}
	if len(folder.Metadata.DependsOn) == 0 {
		return "No dependencies"
	}
	return "Dependencies satisfied"
}

func lastRunSummary(folder *specs.SpecFolder) string {
	if folder == nil || folder.Metadata == nil {
		return "Last run: unknown"
	}
	meta := folder.Metadata
	if meta.LastRun == nil {
		return fmt.Sprintf("Last run: never (status %s)", strings.ToUpper(string(meta.Status)))
	}
	t := meta.LastRun.In(time.Now().Location())
	return fmt.Sprintf("Last run: %s (status %s)", t.Format("2006-01-02 15:04"), strings.ToUpper(string(meta.Status)))
}

func (m *model) listView() string {
	body := []string{m.list.View()}
	filterLabel := "all specs"
	if m.filterRunnable {
		filterLabel = "runnable only"
	}
	if m.confirmUnmet {
		deps := "None"
		if item := m.currentItem(); item != nil {
			if len(item.folder.UnmetDeps) > 0 {
				deps = strings.Join(item.folder.UnmetDeps, ", ")
			}
		}
		modal := components.Modal(components.ModalConfig{
			Width: m.width,
			Title: "Run with unmet dependencies?",
			Body: []string{
				fmt.Sprintf("Unmet deps: %s", deps),
				"Press y to run anyway or n/esc to cancel.",
			},
		})
		body = append(body, modal)
	}
	if m.flash != "" {
		body = append(body, components.Flash(components.FlashInfo, m.flash))
	}
	help := []components.HelpEntry{
		{Key: "↑/↓", Label: "move"},
		{Key: "enter", Label: "run"},
		{Key: "f", Label: fmt.Sprintf("filter (%s)", filterLabel)},
		{Key: "q", Label: "quit"},
	}
	if m.confirmUnmet {
		help = append(help,
			components.HelpEntry{Key: "y", Label: "confirm"},
			components.HelpEntry{Key: "n/esc", Label: "cancel"},
		)
	}
	return components.PageShell(components.PageShellOptions{
		Width:       m.width,
		Title:       components.TitleConfig{Title: "helm run"},
		Body:        strings.Join(body, "\n\n"),
		HelpEntries: help,
	})
}

func (m *model) runningView() string {
	if m.running == nil || m.running.spec == nil || m.running.spec.Metadata == nil {
		return "Starting run..."
	}
	spec := m.running.spec.Metadata
	attemptLine := "Waiting for attempts to start"
	stage := strings.TrimSpace(m.running.stage)
	if stage != "" {
		stage = cases.Title(language.Und).String(stage)
	}

	if m.running.attempt > 0 && m.running.totalAttempts > 0 {
		if stage != "" {
			attemptLine = fmt.Sprintf("%s — attempt %d of %d", stage, m.running.attempt, m.running.totalAttempts)
		} else {
			attemptLine = fmt.Sprintf("Attempt %d of %d", m.running.attempt, m.running.totalAttempts)
		}
	} else if stage != "" {
		attemptLine = stage
	} else if m.running.started {
		attemptLine = "Streaming Codex logs..."
	}
	sections := []string{
		components.SpinnerLine(m.spinner.View(), attemptLine),
	}
	if chip := components.ResumeChip(m.running.resumeCmd); chip != "" {
		sections = append(sections, chip)
	}
	if m.flash != "" {
		sections = append(sections, components.Flash(components.FlashInfo, m.flash))
	}
	viewport := components.ViewportCard(components.ViewportCardOptions{
		Width:        m.width,
		Content:      m.viewport.View(),
		Status:       "Scroll with ↑/↓, PgUp/PgDn or mouse",
		Preformatted: true,
	})
	sections = append(sections, viewport)
	if m.confirmKill {
		sections = append(sections, components.Modal(components.ModalConfig{
			Width: m.width,
			Title: "Stop the current run?",
			Body: []string{
				"implement-spec will be terminated.",
				"Press ESC again within 2s to stop; press Q twice to quit Helm.",
			},
		}))
	}
	help := []components.HelpEntry{
		{Key: "↑/↓ PgUp/PgDn", Label: "scroll"},
		{Key: "mouse", Label: "scroll"},
		{Key: "c", Label: "copy resume"},
		{Key: "esc×2", Label: "stop run"},
		{Key: "q×2", Label: "quit"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       m.width,
		Title:       components.TitleConfig{Title: fmt.Sprintf("Running %s — %s", spec.ID, spec.Name)},
		Body:        strings.Join(sections, "\n\n"),
		HelpEntries: help,
	})
}

func (m *model) resultView() string {
	if m.result == nil {
		return "Run complete"
	}
	lines := []string{}
	if m.result.err != nil {
		lines = append(lines, components.Flash(components.FlashDanger, fmt.Sprintf("Error: %v", m.result.err)))
	} else {
		statusLabel := strings.ToUpper(string(m.result.status))
		if statusLabel == "" {
			statusLabel = "UNKNOWN"
		}
		lines = append(lines, fmt.Sprintf("Spec status: %s", statusLabel))
		if m.result.exitErr != nil {
			lines = append(lines, components.Flash(components.FlashDanger, fmt.Sprintf("implement-spec exited with code %d: %v", m.result.exitCode, m.result.exitErr)))
		} else {
			lines = append(lines, components.Flash(components.FlashSuccess, "implement-spec exited successfully."))
		}
	}
	if len(m.result.remaining) > 0 {
		lines = append(lines, components.BulletList(m.result.remaining))
	}
	if chip := components.ResumeChip(m.result.resumeCmd); chip != "" {
		lines = append(lines, chip)
	}
	if m.flash != "" {
		lines = append(lines, components.Flash(components.FlashInfo, m.flash))
	}
	viewport := components.ViewportCard(components.ViewportCardOptions{
		Width:        m.width,
		Content:      m.viewport.View(),
		Status:       "Scroll with ↑/↓, PgUp/PgDn or mouse — enter to return",
		Preformatted: true,
	})
	lines = append(lines, viewport)
	help := []components.HelpEntry{
		{Key: "enter/r", Label: "back to list"},
		{Key: "c", Label: "copy resume"},
		{Key: "q", Label: "quit"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       m.width,
		Title:       components.TitleConfig{Title: fmt.Sprintf("Run result — %s", m.result.specID)},
		Body:        strings.Join(lines, "\n\n"),
		HelpEntries: help,
	})
}
