package specsplit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/polarzero/helm/internal/config"
	splitting "github.com/polarzero/helm/internal/specsplit"
)

// ErrCanceled indicates the user exited before running the split.
var ErrCanceled = errors.New("spec split canceled")

// Outcome is returned by the split TUI.
type Outcome struct {
	Result    *splitting.Result
	JumpToRun bool
}

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
func Run(opts Options) (*Outcome, error) {
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
	return &Outcome{Result: finalModel.result, JumpToRun: finalModel.jumpToRun}, nil
}

type phase int

const (
	phaseIntro phase = iota
	phaseInput
	phaseRunning
	phaseDone
)

type model struct {
	opts  Options
	phase phase

	ta      textarea.Model
	spinner spinner.Model
	vp      viewport.Model

	width  int
	height int

	inputErr string
	running  bool

	result    *splitting.Result
	err       error
	canceled  bool
	jumpToRun bool

	logs        []string
	stream      <-chan tea.Msg
	cancelSplit context.CancelFunc
}

func newModel(opts Options) *model {
	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(18)
	ta.ShowLineNumbers = true
	ta.Placeholder = "Paste the large spec here..."
	ta.SetValue(opts.InitialInput)
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(80, 12)

	return &model{
		opts:    opts,
		phase:   phaseIntro,
		ta:      ta,
		spinner: sp,
		vp:      vp,
	}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
		m.ta.SetWidth(max(20, v.Width-4))
		m.ta.SetHeight(max(10, v.Height-10))
		m.vp.Width = max(20, v.Width-4)
		m.vp.Height = max(8, v.Height-8)
		return m, nil
	case tea.KeyMsg:
		if v.String() == "q" {
			m.canceled = true
			return m, tea.Quit
		}
		if v.Type == tea.KeyEsc {
			if m.phase == phaseInput {
				m.canceled = true
				return m, tea.Quit
			}
			if m.phase == phaseRunning {
				if m.cancelSplit != nil {
					m.cancelSplit()
				}
				m.canceled = true
				return m, tea.Quit
			}
			if m.phase == phaseDone {
				return m, tea.Quit
			}
		}
	}

	// Hook up stream when runner starts
	if start, ok := msg.(runnerStartMsg); ok {
		if start.err != nil {
			m.err = start.err
			m.phase = phaseDone
			return m, tea.Quit
		}
		m.stream = start.stream
		m.cancelSplit = start.cancel
		return m, listenStream(m.stream)
	}

	switch m.phase {
	case phaseIntro:
		return m.updateIntro(msg)
	case phaseInput:
		return m.updateInput(msg)
	case phaseRunning:
		return m.updateRunning(msg)
	case phaseDone:
		return m.updateDone(msg)
	default:
		return m, nil
	}
}

func (m *model) updateIntro(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEnter:
			m.phase = phaseInput
		case tea.KeyCtrlC:
			m.canceled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEnter:
			trimmed := strings.TrimSpace(m.ta.Value())
			if trimmed == "" {
				m.inputErr = "Spec content is required"
				return m, nil
			}
			m.inputErr = ""
			m.phase = phaseRunning
			m.running = true
			m.logs = nil
			m.vp.SetContent("")
			return m, tea.Batch(m.spinner.Tick, startSplitCmd(m.opts, m.ta.Value()))
		case tea.KeyCtrlC:
			m.canceled = true
			return m, tea.Quit
		case tea.KeyCtrlL:
			m.ta.SetValue("")
		case tea.KeyCtrlO:
			// Load from file prompt: reuse textarea as quick path input.
			path := strings.TrimSpace(m.ta.Value())
			if path == "" {
				m.inputErr = "Enter a file path then press Ctrl+O"
				return m, nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				m.inputErr = fmt.Sprintf("read file: %v", err)
				return m, nil
			}
			m.ta.SetValue(string(data))
			m.ta.CursorEnd()
			m.inputErr = fmt.Sprintf("Loaded %s", path)
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m *model) updateRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case splitLogMsg:
		m.appendLog(v)
		return m, listenStream(m.stream)
	case splitFinishedMsg:
		m.running = false
		m.result = v.result
		m.err = v.err
		m.phase = phaseDone
		m.stream = nil
		if v.err != nil {
			m.appendLog(splitLogMsg{stream: "stderr", text: v.err.Error()})
		}
		return m, nil
	case splitStreamClosedMsg:
		// stream closed without finish message
		m.running = false
		m.phase = phaseDone
		return m, nil
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *model) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "r":
			m.jumpToRun = true
			return m, tea.Quit
		case "n":
			m.phase = phaseInput
			m.result = nil
			m.err = nil
			m.logs = nil
			m.vp.SetContent("")
			return m, nil
		case "q", "enter", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) appendLog(msg splitLogMsg) {
	if msg.text == "" {
		return
	}
	m.logs = append(m.logs, fmt.Sprintf("%s: %s", msg.stream, msg.text))
	if len(m.logs) > 2000 {
		m.logs = m.logs[len(m.logs)-2000:]
	}
	m.vp.SetContent(strings.Join(m.logs, "\n"))
	m.vp.GotoBottom()
}

func (m *model) View() string {
	switch m.phase {
	case phaseIntro:
		return introView()
	case phaseInput:
		return inputView(m)
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
		stream := make(chan tea.Msg)
		out := newLineEmitter("stdout", stream)
		errw := newLineEmitter("stderr", stream)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			defer close(stream)
			defer out.close()
			defer errw.close()
			res, err := splitting.Split(ctx, splitting.Options{
				SpecsRoot:          opts.SpecsRoot,
				GuidePath:          opts.GuidePath,
				RawSpec:            rawSpec,
				AcceptanceCommands: opts.AcceptanceCommands,
				CodexChoice:        opts.CodexChoice,
				PlanPath:           opts.PlanPath,
				Stdout:             out,
				Stderr:             errw,
			})
			stream <- splitFinishedMsg{result: res, err: err}
		}()

		return runnerStartMsg{stream: stream, cancel: cancel}
	}
}

type runnerStartMsg struct {
	stream <-chan tea.Msg
	err    error
	cancel context.CancelFunc
}

type splitLogMsg struct {
	stream string
	text   string
}

type splitFinishedMsg struct {
	result *splitting.Result
	err    error
}

type splitStreamClosedMsg struct{}

func listenStream(ch <-chan tea.Msg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return splitStreamClosedMsg{}
		}
		return msg
	}
}

// lineEmitter buffers writes and emits per-line messages to the TUI.
type lineEmitter struct {
	stream string
	ch     chan<- tea.Msg
	buf    bytes.Buffer
	closed bool
}

func newLineEmitter(stream string, ch chan<- tea.Msg) *lineEmitter {
	return &lineEmitter{stream: stream, ch: ch}
}

func (w *lineEmitter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, io.EOF
	}
	n, _ := w.buf.Write(p)
	for {
		data := w.buf.Bytes()
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			line := string(data[:idx])
			w.ch <- splitLogMsg{stream: w.stream, text: line}
			w.buf.Next(idx + 1)
		} else {
			break
		}
	}
	return n, nil
}

func (w *lineEmitter) close() {
	if w.closed {
		return
	}
	if rem := strings.TrimSpace(w.buf.String()); rem != "" {
		w.ch <- splitLogMsg{stream: w.stream, text: rem}
	}
	w.closed = true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
