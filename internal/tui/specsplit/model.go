package specsplit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/polarzero/helm/internal/config"
	splitting "github.com/polarzero/helm/internal/specsplit"
)

// ErrCanceled indicates the user exited before running the split.
var ErrCanceled = errors.New("spec split canceled")

// Options configures the spec split TUI.
type Options struct {
	SpecsRoot          string
	GuidePath          string
	AcceptanceCommands []string
	CodexChoice        config.CodexChoice
	InitialInput       string
	PlanPath           string
}

// Run launches the spec split TUI.
func Run(opts Options) (*splitting.Result, error) {
	mdl := newModel(opts)
	prog := tea.NewProgram(mdl)
	res, err := prog.Run()
	if err != nil {
		return nil, err
	}
	finalModel, ok := res.(*model)
	if !ok {
		return nil, fmt.Errorf("unexpected model type %T", res)
	}
	if finalModel.canceled {
		return nil, ErrCanceled
	}
	if finalModel.err != nil {
		return nil, finalModel.err
	}
	return finalModel.result, nil
}

type phase int

const (
	phaseIntro phase = iota
	phaseInput
	phasePreview
	phaseRunning
	phaseDone
)

type model struct {
	opts  Options
	phase phase

	ta      textarea.Model
	spinner spinner.Model

	width  int
	height int

	inputErr string
	running  bool

	result   *splitting.Result
	err      error
	canceled bool

	preview []string
}

func newModel(opts Options) *model {
	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.ShowLineNumbers = true
	ta.Placeholder = "Paste the large spec here..."
	ta.SetValue(opts.InitialInput)
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return &model{
		opts:    opts,
		phase:   phaseIntro,
		ta:      ta,
		spinner: sp,
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ta.SetWidth(max(20, msg.Width-4))
		m.ta.SetHeight(max(10, msg.Height-8))
		return m, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		if key.Type == tea.KeyCtrlC {
			if m.phase != phaseDone {
				m.canceled = true
			}
			return m, tea.Quit
		}
	}

	switch m.phase {
	case phaseIntro:
		return m.updateIntro(msg)
	case phaseInput:
		return m.updateInput(msg)
	case phasePreview:
		return m.updatePreview(msg)
	case phaseRunning:
		return m.updateRunning(msg)
	case phaseDone:
		return m.updateDone(msg)
	default:
		return m, nil
	}
}

func (m *model) updateIntro(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.phase = phaseInput
		case tea.KeyEsc:
			m.canceled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.canceled = true
			return m, tea.Quit
		case tea.KeyCtrlD:
			trimmed := strings.TrimSpace(m.ta.Value())
			if trimmed == "" {
				m.inputErr = "Spec content is required"
				return m, nil
			}
			m.inputErr = ""
			m.preview = buildPreviewLines(m.ta.Value(), 60)
			m.phase = phasePreview
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m *model) updatePreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.phase = phaseRunning
			m.running = true
			m.spinner = spinner.New()
			m.spinner.Spinner = spinner.Dot
			return m, tea.Batch(m.spinner.Tick, startSplitCmd(m.opts, m.ta.Value()))
		case tea.KeyCtrlB, tea.KeyEsc:
			m.phase = phaseInput
			return m, nil
		}
		if msg.String() == "b" {
			m.phase = phaseInput
			return m, nil
		}
	}
	return m, nil
}

func (m *model) updateRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case splitCompleteMsg:
		m.running = false
		if msg.err != nil {
			m.err = msg.err
			m.phase = phaseDone
			return m, nil
		}
		m.result = msg.result
		m.phase = phaseDone
		return m, nil
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *model) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, tea.Quit
		}
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string {
	switch m.phase {
	case phaseIntro:
		return introView()
	case phaseInput:
		return inputView(m)
	case phasePreview:
		return previewView(m)
	case phaseRunning:
		return runningView(m)
	case phaseDone:
		return doneView(m)
	default:
		return ""
	}
}

func startSplitCmd(opts Options, rawSpec string) tea.Cmd {
	return func() tea.Msg {
		res, err := splitting.Split(context.Background(), splitting.Options{
			SpecsRoot:          opts.SpecsRoot,
			GuidePath:          opts.GuidePath,
			RawSpec:            rawSpec,
			AcceptanceCommands: opts.AcceptanceCommands,
			CodexChoice:        opts.CodexChoice,
			PlanPath:           opts.PlanPath,
		})
		return splitCompleteMsg{result: res, err: err}
	}
}

type splitCompleteMsg struct {
	result *splitting.Result
	err    error
}

func buildPreviewLines(text string, maxLines int) []string {
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return lines
	}
	preview := make([]string, 0, maxLines+1)
	preview = append(preview, lines[:maxLines]...)
	preview = append(preview, fmt.Sprintf("... (%d more lines)", len(lines)-maxLines))
	return preview
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
