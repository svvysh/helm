package scaffold

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/polarzero/helm/internal/config"
	innerscaffold "github.com/polarzero/helm/internal/scaffold"
)

// ErrCanceled is returned when the user aborts the scaffold flow.
var ErrCanceled = errors.New("scaffold canceled")

// Options configures the scaffold TUI.
type Options struct {
	Root     string
	Defaults *config.Settings
}

// Run launches the scaffold TUI and returns the resulting workspace summary.
func Run(opts Options) (*innerscaffold.Result, error) {
	initial := newModel(opts)
	prog := tea.NewProgram(initial)
	finalModel, err := prog.Run()
	if err != nil {
		return nil, err
	}
	m := finalModel.(*model)
	if m.cancelled {
		return nil, ErrCanceled
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type step int

const (
	stepIntro step = iota
	stepMode
	stepCommands
	stepOptions
	stepConfirm
	stepRunning
	stepComplete
)

type model struct {
	opts        Options
	step        step
	modeIndex   int
	modeChoices []config.Mode

	commands      []string
	commandInput  textinput.Model
	specsRoot     textinput.Model
	focusIndex    int
	makeGraph     bool
	optionsErr    string
	result        *innerscaffold.Result
	spinner       spinner.Model
	cancelled     bool
	err           error
	running       bool
	width, height int
}

func newModel(opts Options) *model {
	defaults := opts.Defaults
	if defaults == nil {
		defaults = &config.Settings{Mode: config.ModeStrict, SpecsRoot: config.DefaultSpecsRoot()}
	}
	m := &model{
		opts:        opts,
		modeChoices: []config.Mode{config.ModeStrict, config.ModeParallel},
		commands:    cloneCommands(defaultAcceptanceCommands(defaults)),
	}
	for i, choice := range m.modeChoices {
		if choice == defaults.Mode {
			m.modeIndex = i
			break
		}
	}
	m.commandInput = textinput.New()
	m.commandInput.Placeholder = "e.g. go test ./..."
	m.commandInput.Prompt = "↪ "
	m.commandInput.Focus()
	m.specsRoot = textinput.New()
	root := defaults.SpecsRoot
	if root == "" {
		root = config.DefaultSpecsRoot()
	}
	m.specsRoot.SetValue(root)
	m.specsRoot.Placeholder = config.DefaultSpecsRoot()
	m.specsRoot.Prompt = "↪ "
	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
	return m
}

func defaultAcceptanceCommands(settings *config.Settings) []string {
	if settings != nil && len(settings.AcceptanceCommands) > 0 {
		return settings.AcceptanceCommands
	}
	return innerscaffold.DefaultAcceptanceCommands()
}

func cloneCommands(src []string) []string {
	out := make([]string, len(src))
	copy(out, src)
	return out
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	if m.running && m.step == stepRunning {
		return m.updateRunning(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if keyCancel(msg) {
			m.cancelled = true
			return m, tea.Quit
		}
	}

	switch m.step {
	case stepIntro:
		return m.updateIntro(msg)
	case stepMode:
		return m.updateMode(msg)
	case stepCommands:
		return m.updateCommands(msg)
	case stepOptions:
		return m.updateOptions(msg)
	case stepConfirm:
		return m.updateConfirm(msg)
	case stepComplete:
		return m.updateComplete(msg)
	default:
		return m, nil
	}
}

func keyCancel(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return true
	default:
		return false
	}
}

func (m *model) updateIntro(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			m.step = stepMode
		}
	}
	return m, nil
}

func (m *model) updateMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "shift+tab":
			m.modeIndex = (m.modeIndex - 1 + len(m.modeChoices)) % len(m.modeChoices)
		case "down", "tab":
			m.modeIndex = (m.modeIndex + 1) % len(m.modeChoices)
		case "enter":
			m.step = stepCommands
		}
	}
	return m, nil
}

func (m *model) updateCommands(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		value := strings.TrimSpace(m.commandInput.Value())
		switch msg.String() {
		case "enter":
			if value == "" {
				m.step = stepOptions
				return m, nil
			}
			m.commands = append(m.commands, value)
			m.commandInput.SetValue("")
		case "ctrl+w":
			if value == "" && len(m.commands) > 0 {
				m.commands = m.commands[:len(m.commands)-1]
			}
		}
	}
	var cmd tea.Cmd
	m.commandInput, cmd = m.commandInput.Update(msg)
	return m, cmd
}

func (m *model) updateOptions(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.focusIndex = (m.focusIndex + 1) % 2
		case "up", "shift+tab":
			m.focusIndex = (m.focusIndex - 1 + 2) % 2
		case "enter":
			if m.focusIndex == 0 {
				if err := m.validateSpecsRoot(); err != nil {
					m.optionsErr = err.Error()
					break
				}
				m.step = stepConfirm
			} else {
				m.makeGraph = !m.makeGraph
			}
		case " ":
			if m.focusIndex == 1 {
				m.makeGraph = !m.makeGraph
			}
		}
	}
	var cmd tea.Cmd
	if m.focusIndex == 0 {
		m.specsRoot, cmd = m.specsRoot.Update(msg)
	}
	return m, cmd
}

func (m *model) validateSpecsRoot() error {
	if strings.TrimSpace(m.specsRoot.Value()) == "" {
		return errors.New("specs root cannot be empty")
	}
	return nil
}

func (m *model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			if err := m.validateSpecsRoot(); err != nil {
				m.optionsErr = err.Error()
				return m, nil
			}
			m.optionsErr = ""
			m.step = stepRunning
			m.running = true
			cmd := tea.Batch(m.spinner.Tick, runScaffoldCmd(m.opts.Root, m.answers()))
			return m, cmd
		}
	}
	return m, nil
}

func (m *model) updateRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case scaffoldFinishedMsg:
		m.running = false
		m.result = msg.result
		m.err = msg.err
		m.step = stepComplete
		return m, nil
	}
	return m, nil
}

func (m *model) updateComplete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string {
	if m.cancelled {
		return "Scaffold canceled."
	}
	switch m.step {
	case stepIntro:
		return introView()
	case stepMode:
		return modeView(m.modeChoices, m.modeIndex)
	case stepCommands:
		return commandsView(m.commands, m.commandInput.View())
	case stepOptions:
		return optionsView(m.specsRoot.View(), m.focusIndex, m.makeGraph, m.optionsErr)
	case stepConfirm:
		return confirmView(m.answers())
	case stepRunning:
		return runningView(m.spinner.View())
	case stepComplete:
		return completeView(m.result, m.err)
	default:
		return ""
	}
}

func (m *model) answers() innerscaffold.Answers {
	cmds := cloneCommands(m.commands)
	trimmed := cmds[:0]
	for _, c := range cmds {
		if t := strings.TrimSpace(c); t != "" {
			trimmed = append(trimmed, t)
		}
	}
	mode := m.modeChoices[m.modeIndex]
	return innerscaffold.Answers{
		Mode:                mode,
		AcceptanceCommands:  trimmed,
		SpecsRoot:           strings.TrimSpace(m.specsRoot.Value()),
		GenerateSampleGraph: m.makeGraph,
	}
}

type scaffoldFinishedMsg struct {
	result *innerscaffold.Result
	err    error
}

func runScaffoldCmd(root string, answers innerscaffold.Answers) tea.Cmd {
	return func() tea.Msg {
		res, err := innerscaffold.Run(root, answers)
		return scaffoldFinishedMsg{result: res, err: err}
	}
}
