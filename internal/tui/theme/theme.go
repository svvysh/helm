package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/metadata"
)

// Shared lipgloss styles so TUIs remain visually consistent.
var (
	TitleStyle    = lipgloss.NewStyle().Bold(true)
	HintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	WarningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	SelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)

	badgeBase     = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	BadgeTodo     = badgeBase.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("94"))
	BadgeProgress = badgeBase.Foreground(lipgloss.Color("16")).Background(lipgloss.Color("148"))
	BadgeDone     = badgeBase.Foreground(lipgloss.Color("16")).Background(lipgloss.Color("120"))
	BadgeBlocked  = badgeBase.Foreground(lipgloss.Color("231")).Background(lipgloss.Color("124"))
	BadgeFailed   = badgeBase.Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196"))
)

// StatusCategory groups effective statuses for summary counts.
type StatusCategory string

const (
	StatusTodo       StatusCategory = "TODO"
	StatusInProgress StatusCategory = "IN PROGRESS"
	StatusDone       StatusCategory = "DONE"
	StatusBlocked    StatusCategory = "BLOCKED"
	StatusFailed     StatusCategory = "FAILED"
)

// StatusBadge renders a badge label/style pair plus the effective category given metadata.
func StatusBadge(meta *metadata.SpecMetadata, blocked bool) (string, lipgloss.Style, StatusCategory) {
	if blocked && (meta == nil || meta.Status != metadata.StatusDone) {
		return "BLOCKED", BadgeBlocked, StatusBlocked
	}
	if meta == nil {
		return "TODO", BadgeTodo, StatusTodo
	}

	switch meta.Status {
	case metadata.StatusDone:
		return "DONE", BadgeDone, StatusDone
	case metadata.StatusInProgress:
		if blocked {
			return "BLOCKED", BadgeBlocked, StatusBlocked
		}
		return "IN PROGRESS", BadgeProgress, StatusInProgress
	case metadata.StatusFailed:
		return "FAILED", BadgeFailed, StatusFailed
	case metadata.StatusBlocked:
		return "BLOCKED", BadgeBlocked, StatusBlocked
	case metadata.StatusTodo:
		if blocked {
			return "BLOCKED", BadgeBlocked, StatusBlocked
		}
		return "TODO", BadgeTodo, StatusTodo
	default:
		label := strings.ToUpper(string(meta.Status))
		return label, BadgeTodo, StatusTodo
	}
}
