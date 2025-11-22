package home

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
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
	p := tea.NewProgram(m)
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
	case tea.KeyMsg:
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
			}
		case "down", "j":
			if m.cursor < len(items)-1 {
				m.cursor++
			}
		case "enter":
			m.selection = items[m.cursor].sel
			m.hasChosen = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string {
	var out string
	out += "Helm — choose an action:\n\n"
	for i, item := range items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		out += cursor + " " + item.title + "\n"
	}
	out += "\n↑/↓ to move, enter to select, q to quit"
	return out
}
