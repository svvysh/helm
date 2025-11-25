package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/metadata"
)

// Palette exposes the semantic color tokens derived from the Glow reference theme.
type Palette struct {
	Primary   lipgloss.TerminalColor
	Accent    lipgloss.TerminalColor
	Muted     lipgloss.TerminalColor
	Warning   lipgloss.TerminalColor
	Success   lipgloss.TerminalColor
	Surface   lipgloss.TerminalColor
	Border    lipgloss.TerminalColor
	Highlight lipgloss.TerminalColor
}

var Colors = Palette{
	Primary:   lipgloss.AdaptiveColor{Light: "#FFFDF5", Dark: "#FFFDF5"},
	Accent:    lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"},
	Muted:     lipgloss.AdaptiveColor{Light: "#909090", Dark: "#626262"},
	Warning:   lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#ED567A"},
	Success:   lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#35D79C"},
	Surface:   lipgloss.AdaptiveColor{Light: "#E6E6E6", Dark: "#242424"},
	Border:    lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#3C3C3C"},
	Highlight: lipgloss.AdaptiveColor{Light: "#ECFD65", Dark: "#ECFD65"},
}

const (
	ViewHorizontalPadding = 4
	ViewTopPadding        = 2
	ViewBottomPadding     = 1
	SectionSpacing        = 1
)

var (
	TitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(Colors.Primary)
	SubtitleStyle = lipgloss.NewStyle().Bold(true).Foreground(Colors.Accent)
	BodyStyle     = lipgloss.NewStyle().Foreground(Colors.Primary)
	HintStyle     = lipgloss.NewStyle().Foreground(Colors.Muted)
	WarningStyle  = lipgloss.NewStyle().Foreground(Colors.Warning).Bold(true)
	SuccessStyle  = lipgloss.NewStyle().Foreground(Colors.Success).Bold(true)
	SelectedStyle = lipgloss.NewStyle().Foreground(Colors.Surface).Background(Colors.Accent).Bold(true)
	BorderStyle   = lipgloss.NewStyle().BorderForeground(Colors.Border)
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

var (
	badgeBase     = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	BadgeTodo     = badgeBase.Foreground(Colors.Primary).Background(Colors.Border)
	BadgeProgress = badgeBase.Foreground(Colors.Surface).Background(Colors.Accent)
	BadgeDone     = badgeBase.Foreground(Colors.Surface).Background(Colors.Success)
	BadgeBlocked  = badgeBase.Foreground(Colors.Surface).Background(Colors.Muted)
	BadgeFailed   = badgeBase.Foreground(Colors.Surface).Background(Colors.Warning)
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
