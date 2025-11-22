package settings

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/polarzero/helm/internal/config"
)

// ErrCanceled is returned when the user exits without saving.
var ErrCanceled = errors.New("settings canceled")

// Options configures the settings TUI.
type Options struct {
	Initial *config.Settings
}

// Run launches the settings TUI and saves on confirmation.
func Run(opts Options) (*config.Settings, error) {
	initial := opts.Initial
	if initial == nil {
		defaults := config.DefaultSpecsRoot()
		initial = &config.Settings{SpecsRoot: defaults}
	}
	config.ApplyDefaults(initial)
	initialModel := newModel(*initial)
	prog := tea.NewProgram(initialModel)
	res, err := prog.Run()
	if err != nil {
		return nil, err
	}
	m, ok := res.(model)
	if !ok {
		return nil, fmt.Errorf("unexpected program result")
	}
	if !m.saved {
		return nil, ErrCanceled
	}
	if err := config.Validate(&m.settings); err != nil {
		return nil, err
	}
	if err := config.SaveSettings(&m.settings); err != nil {
		return nil, err
	}
	return &m.settings, nil
}

type field int

const (
	fieldSpecsRoot field = iota
	fieldMode
	fieldMaxAttempts
	fieldAcceptance
	fieldScaffoldModel
	fieldScaffoldReasoning
	fieldRunImplModel
	fieldRunImplReasoning
	fieldRunVerModel
	fieldRunVerReasoning
	fieldSplitModel
	fieldSplitReasoning
	fieldSave
)

var modelOptions = []string{"gpt-5.1", "gpt-5.1-codex", "gpt-5.1-codex-mini", "git-5.1-codex-max"}

var reasoningByModel = map[string][]string{
	"gpt-5.1":            {"low", "medium", "high"},
	"gpt-5.1-codex":      {"low", "medium", "high"},
	"gpt-5.1-codex-mini": {"medium", "high"},
	"git-5.1-codex-max":  {"low", "medium", "high", "xhigh"},
}

type model struct {
	settings  config.Settings
	cursor    field
	saved     bool
	rootInput textinput.Model
	maxInput  textinput.Model
	acInput   textinput.Model
}

func newModel(settings config.Settings) model {
	root := textinput.New()
	root.Placeholder = "docs/specs"
	root.SetValue(settings.SpecsRoot)
	root.Prompt = ""

	max := textinput.New()
	max.Placeholder = "2"
	max.SetValue(strconv.Itoa(settings.DefaultMaxAttempts))
	max.Prompt = ""

	ac := textinput.New()
	ac.Placeholder = "go test ./..."
	ac.SetValue(strings.Join(settings.AcceptanceCommands, ","))
	ac.Prompt = ""

	return model{settings: settings, rootInput: root, maxInput: max, acInput: ac}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < fieldSave {
				m.cursor++
			}
		case "left", "h":
			m.cycle(-1)
		case "right", "l":
			m.cycle(1)
		case "enter":
			if m.cursor == fieldSave {
				if err := m.persistInputs(); err != nil {
					// keep focus and show inline error via prompt change
					m.maxInput.Placeholder = err.Error()
					return m, nil
				}
				m.saved = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	if m.cursor == fieldSpecsRoot {
		m.rootInput, cmd = m.rootInput.Update(msg)
	} else if m.cursor == fieldMaxAttempts {
		m.maxInput, cmd = m.maxInput.Update(msg)
	} else if m.cursor == fieldAcceptance {
		m.acInput, cmd = m.acInput.Update(msg)
	}
	return m, cmd
}

func (m *model) persistInputs() error {
	m.settings.SpecsRoot = strings.TrimSpace(m.rootInput.Value())
	if m.settings.SpecsRoot == "" {
		m.settings.SpecsRoot = config.DefaultSpecsRoot()
	}
	max, err := strconv.Atoi(strings.TrimSpace(m.maxInput.Value()))
	if err != nil || max <= 0 {
		return fmt.Errorf("max attempts must be >0")
	}
	m.settings.DefaultMaxAttempts = max
	ac := strings.TrimSpace(m.acInput.Value())
	if ac == "" {
		m.settings.AcceptanceCommands = []string{}
	} else {
		parts := strings.Split(ac, ",")
		var cmds []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cmds = append(cmds, p)
			}
		}
		m.settings.AcceptanceCommands = cmds
	}
	return nil
}

func (m *model) cycle(delta int) {
	switch m.cursor {
	case fieldMode:
		if m.settings.Mode == config.ModeStrict {
			m.settings.Mode = config.ModeParallel
		} else {
			m.settings.Mode = config.ModeStrict
		}
	case fieldScaffoldModel:
		m.settings.CodexScaffold.Model = rotate(modelOptions, m.settings.CodexScaffold.Model, delta)
		m.settings.CodexScaffold.Reasoning = ensureReasoning(m.settings.CodexScaffold)
	case fieldScaffoldReasoning:
		m.settings.CodexScaffold.Reasoning = rotate(reasoningByModel[m.settings.CodexScaffold.Model], m.settings.CodexScaffold.Reasoning, delta)
	case fieldRunImplModel:
		m.settings.CodexRunImpl.Model = rotate(modelOptions, m.settings.CodexRunImpl.Model, delta)
		m.settings.CodexRunImpl.Reasoning = ensureReasoning(m.settings.CodexRunImpl)
	case fieldRunImplReasoning:
		m.settings.CodexRunImpl.Reasoning = rotate(reasoningByModel[m.settings.CodexRunImpl.Model], m.settings.CodexRunImpl.Reasoning, delta)
	case fieldRunVerModel:
		m.settings.CodexRunVer.Model = rotate(modelOptions, m.settings.CodexRunVer.Model, delta)
		m.settings.CodexRunVer.Reasoning = ensureReasoning(m.settings.CodexRunVer)
	case fieldRunVerReasoning:
		m.settings.CodexRunVer.Reasoning = rotate(reasoningByModel[m.settings.CodexRunVer.Model], m.settings.CodexRunVer.Reasoning, delta)
	case fieldSplitModel:
		m.settings.CodexSplit.Model = rotate(modelOptions, m.settings.CodexSplit.Model, delta)
		m.settings.CodexSplit.Reasoning = ensureReasoning(m.settings.CodexSplit)
	case fieldSplitReasoning:
		m.settings.CodexSplit.Reasoning = rotate(reasoningByModel[m.settings.CodexSplit.Model], m.settings.CodexSplit.Reasoning, delta)
	}
}

func ensureReasoning(choice config.CodexChoice) string {
	allowed := reasoningByModel[choice.Model]
	for _, r := range allowed {
		if r == choice.Reasoning {
			return r
		}
	}
	return allowed[0]
}

func rotate(list []string, current string, delta int) string {
	if len(list) == 0 {
		return current
	}
	idx := 0
	for i, v := range list {
		if v == current {
			idx = i
			break
		}
	}
	idx = (idx + delta) % len(list)
	if idx < 0 {
		idx += len(list)
	}
	return list[idx]
}

func (m model) View() string {
	lines := []string{
		"Use up/down to move, left/right to change options, enter to save, esc to cancel.",
		"",
		label(m.cursor == fieldSpecsRoot, "Specs root", m.rootInput.View()),
		label(m.cursor == fieldMode, "Mode", string(m.settings.Mode)),
		label(m.cursor == fieldMaxAttempts, "Default max attempts", m.maxInput.View()),
		label(m.cursor == fieldAcceptance, "Acceptance commands (comma-separated)", m.acInput.View()),
		label(m.cursor == fieldScaffoldModel, "Scaffold model", m.settings.CodexScaffold.Model),
		label(m.cursor == fieldScaffoldReasoning, "Scaffold reasoning", m.settings.CodexScaffold.Reasoning),
		label(m.cursor == fieldRunImplModel, "Run worker model", m.settings.CodexRunImpl.Model),
		label(m.cursor == fieldRunImplReasoning, "Run worker reasoning", m.settings.CodexRunImpl.Reasoning),
		label(m.cursor == fieldRunVerModel, "Run verifier model", m.settings.CodexRunVer.Model),
		label(m.cursor == fieldRunVerReasoning, "Run verifier reasoning", m.settings.CodexRunVer.Reasoning),
		label(m.cursor == fieldSplitModel, "Split model", m.settings.CodexSplit.Model),
		label(m.cursor == fieldSplitReasoning, "Split reasoning", m.settings.CodexSplit.Reasoning),
		label(m.cursor == fieldSave, "Save", "press enter"),
	}
	return strings.Join(lines, "\n") + "\n"
}

func label(focused bool, name, value string) string {
	if focused {
		return fmt.Sprintf("> %s: %s", name, value)
	}
	return fmt.Sprintf("  %s: %s", name, value)
}
