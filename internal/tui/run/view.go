package run

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/metadata"
	"github.com/polarzero/helm/internal/specs"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	hintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	badgeBase     = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	badgeTodo     = badgeBase.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("94"))
	badgeProgress = badgeBase.Foreground(lipgloss.Color("16")).Background(lipgloss.Color("148"))
	badgeDone     = badgeBase.Foreground(lipgloss.Color("16")).Background(lipgloss.Color("120"))
	badgeBlocked  = badgeBase.Foreground(lipgloss.Color("231")).Background(lipgloss.Color("124"))
	badgeFailed   = badgeBase.Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196"))
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
	badgeText, badgeStyle := statusBadge(folder)
	header := fmt.Sprintf("%s %s — %s", badgeStyle.Render(badgeText), folder.Metadata.ID, folder.Metadata.Name)
	if selected {
		header = selectedStyle.Render(header)
	}
	dep := dependencySummary(folder)
	last := lastRunSummary(folder)
	lines := []string{header, hintStyle.Render(dep), hintStyle.Render(last)}
	return strings.Join(lines, "\n")
}

func statusBadge(folder *specs.SpecFolder) (string, lipgloss.Style) {
	if folder == nil || folder.Metadata == nil {
		return "TODO", badgeTodo
	}
	status := folder.Metadata.Status
	switch status {
	case metadata.StatusDone:
		return "DONE", badgeDone
	case metadata.StatusInProgress:
		if len(folder.UnmetDeps) > 0 {
			return "BLOCKED", badgeBlocked
		}
		return "IN PROGRESS", badgeProgress
	case metadata.StatusBlocked:
		return "BLOCKED", badgeBlocked
	case metadata.StatusFailed:
		return "FAILED", badgeFailed
	case metadata.StatusTodo:
		if len(folder.UnmetDeps) > 0 {
			return "BLOCKED", badgeBlocked
		}
		return "TODO", badgeTodo
	default:
		if len(folder.UnmetDeps) > 0 {
			return "BLOCKED", badgeBlocked
		}
		if folder.CanRun {
			return "TODO", badgeTodo
		}
		return "TODO", badgeTodo
	}
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
	var b strings.Builder
	b.WriteString(m.list.View())
	b.WriteString("\n\n")
	filterLabel := "All specs"
	if m.filterRunnable {
		filterLabel = "Runnable only"
	}
	b.WriteString(hintStyle.Render(fmt.Sprintf("[↑/↓] move  [enter] run  [f] filter: %s  [q] quit", filterLabel)))
	if m.confirmUnmet {
		deps := "none"
		if item := m.currentItem(); item != nil {
			if len(item.folder.UnmetDeps) > 0 {
				deps = strings.Join(item.folder.UnmetDeps, ", ")
			}
		}
		b.WriteString("\n\n")
		b.WriteString(warningStyle.Render(fmt.Sprintf("This spec has unmet dependencies: %s. Run anyway? [y/N]", deps)))
	}
	return b.String()
}

func (m *model) runningView() string {
	if m.running == nil || m.running.spec == nil || m.running.spec.Metadata == nil {
		return "Starting run..."
	}
	spec := m.running.spec.Metadata
	title := titleStyle.Render(fmt.Sprintf("Running %s — %s", spec.ID, spec.Name))
	attemptLine := "Waiting for attempts to start"
	if m.running.attempt > 0 && m.running.totalAttempts > 0 {
		attemptLine = fmt.Sprintf("Attempt %d of %d", m.running.attempt, m.running.totalAttempts)
	}
	lines := []string{
		title,
		hintStyle.Render(attemptLine + "  •  Press q to stop, PgUp/PgDn to scroll"),
		"",
		m.viewport.View(),
	}
	if m.confirmKill {
		lines = append(lines, "", warningStyle.Render("Stop this run and terminate implement-spec? [y/N]"))
	}
	return strings.Join(lines, "\n")
}

func (m *model) resultView() string {
	if m.result == nil {
		return "Run complete"
	}
	title := titleStyle.Render(fmt.Sprintf("Run result — %s", m.result.specID))
	lines := []string{title}
	if m.result.err != nil {
		lines = append(lines, warningStyle.Render(fmt.Sprintf("Error: %v", m.result.err)))
	} else {
		statusLabel := strings.ToUpper(string(m.result.status))
		if statusLabel == "" {
			statusLabel = "UNKNOWN"
		}
		lines = append(lines, fmt.Sprintf("Spec status: %s", statusLabel))
		if m.result.exitErr != nil {
			lines = append(lines, warningStyle.Render(fmt.Sprintf("implement-spec exited with code %d: %v", m.result.exitCode, m.result.exitErr)))
		} else {
			lines = append(lines, hintStyle.Render("implement-spec exited successfully."))
		}
	}
	if len(m.result.remaining) > 0 {
		lines = append(lines, "", "Remaining tasks:")
		for _, task := range m.result.remaining {
			lines = append(lines, fmt.Sprintf("- %s", task))
		}
	}
	lines = append(lines, "", hintStyle.Render("Press enter/r to return to list, q to quit."), "", m.viewport.View())
	return strings.Join(lines, "\n")
}
