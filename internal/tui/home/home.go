package home

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/polarzero/helm/internal/tui/components"
)

// Selection enumerates home-menu choices.
type Selection int

const (
	SelectRun Selection = iota
	SelectBreakdown
	SelectStatus
	SelectQuit
)

// Options configures the home TUI; fields are carried so future panes can reuse them.
type Options struct {
	Root               string
	SpecsRoot          string
	Settings           interface{}
	AcceptanceCommands []string
}

// Result captures the chosen selection.
type Result struct {
	Selection Selection
}

// ErrCanceled indicates the user quit without making a selection.
var ErrCanceled = errors.New("home canceled")

// Run presents a minimal home menu and returns the chosen action.
func Run(_ Options) (*Result, error) {
	m := &model{cursor: 0}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	finalModel, ok := final.(*model)
	if !ok {
		return nil, errors.New("unexpected program model")
	}
	if finalModel.canceled {
		return nil, ErrCanceled
	}
	return &Result{Selection: finalModel.selection}, nil
}

type model struct {
	cursor    int
	selection Selection
	canceled  bool
	hasChosen bool
	width     int
	height    int
}

var items = []struct {
	title string
	sel   Selection
}{
	{"Run specs", SelectRun},
	{"Breakdown specs", SelectBreakdown},
	{"Status overview", SelectStatus},
	{"Quit", SelectQuit},
}

func (m model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, tea.ClearScreen
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch msg.String() {
		case "ctrl+c", "q":
			m.canceled = true
			return m, tea.Quit
		case "esc":
			m.canceled = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				cmd = tea.ClearScreen
			}
		case "down", "j":
			if m.cursor < len(items)-1 {
				m.cursor++
				cmd = tea.ClearScreen
			}
		case "enter":
			m.selection = items[m.cursor].sel
			m.hasChosen = true
			return m, tea.Quit
		}
		return m, cmd
	}
	return m, nil
}

func (m *model) View() string {
	menuItems := make([]components.MenuItem, len(items))
	for i, item := range items {
		menuItems[i] = components.MenuItem{Title: item.title}
	}
	body := components.MenuList(m.width, menuItems, m.cursor)
	help := []components.HelpEntry{
		{Key: "↑/↓", Label: "move"},
		{Key: "enter", Label: "select"},
		{Key: "q", Label: "quit"},
	}
	view := components.PageShell(components.PageShellOptions{
		Width:       m.width,
		Title:       components.TitleConfig{Title: "Helm — choose an action"},
		Body:        body,
		HelpEntries: help,
	})
	return components.PadToHeight(view, m.height)
}
