package specsplit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/polarzero/helm/internal/config"
	splitting "github.com/polarzero/helm/internal/specsplit"
	"github.com/polarzero/helm/internal/tui/components"
	"github.com/polarzero/helm/internal/tui/theme"
)

// ErrCanceled indicates the user exited before running the split.
var (
	ErrCanceled = errors.New("spec split canceled")
	ErrQuitAll  = errors.New("spec split quit")
)

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
	prog := tea.NewProgram(mdl, tea.WithAltScreen())
	res, err := prog.Run()
	if err != nil {
		return nil, err
	}
	finalModel, ok := res.(*model)
	if !ok {
		return nil, fmt.Errorf("unexpected model type %T", res)
	}
	if finalModel.err != nil {
		return nil, finalModel.err
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

var sessionIDRe = regexp.MustCompile(`(?i)^session id:\s*([a-f0-9-]{36})$`)

type model struct {
	opts  Options
	phase phase

	draft       string
	editorPath  string
	spinner     spinner.Model
	vp          viewport.Model
	confirmKill bool
	killKey     string

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
	sessionID   string
	resumeCmd   string
	flash       string
}

func newModel(opts Options) *model {
	sp := components.NewSpinner()

	vp := viewport.New(80, 12)

	return &model{
		opts:    opts,
		phase:   phaseIntro,
		draft:   opts.InitialInput,
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
		contentWidth := max(20, v.Width-(theme.ViewHorizontalPadding*2)-6)
		m.vp.Width = contentWidth
		m.vp.Height = max(3, min(5, v.Height-12))
		return m, nil
	case tea.KeyMsg:
		if v.Type == tea.KeyCtrlC {
			if m.phase == phaseRunning && m.cancelSplit != nil {
				m.cancelSplit()
			}
			m.canceled = true
			m.err = ErrQuitAll
			return m, tea.Quit
		}
		// Only handle q/esc here for non-running phases; running phase handles double-press confirmation.
		if m.phase != phaseRunning {
			switch v.String() {
			case "q":
				m.err = ErrQuitAll
				return m, tea.Quit
			case "esc":
				if m.phase == phaseInput {
					m.phase = phaseIntro
					m.inputErr = ""
					return m, nil
				}
				m.canceled = true
				return m, tea.Quit
			}
		}
	case editorFinishedMsg:
		if m.phase != phaseInput {
			return m, nil
		}
		if v.err != nil {
			m.inputErr = fmt.Sprintf("Editor error: %v", v.err)
			return m, nil
		}
		if m.editorPath == "" {
			m.inputErr = "Editor finished but no file to read"
			return m, nil
		}
		data, err := os.ReadFile(m.editorPath)
		if err != nil {
			m.inputErr = fmt.Sprintf("Read editor file: %v", err)
			return m, nil
		}
		m.draft = string(data)
		m.inputErr = ""
		return m, nil
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
			// Allow Option/Alt+Enter or Shift/Ctrl+Enter to insert newlines; only plain Enter starts splitting.
			if key.Alt || key.String() == "shift+enter" || key.String() == "ctrl+enter" {
				m.inputErr = ""
				return m, m.openEditorCmd()
			}
			trimmed := strings.TrimSpace(m.draft)
			if trimmed == "" {
				m.inputErr = ""
				return m, m.openEditorCmd()
			}
			m.inputErr = ""
			m.phase = phaseRunning
			m.running = true
			m.confirmKill = false
			m.killKey = ""
			m.logs = nil
			m.flash = ""
			m.sessionID = ""
			m.resumeCmd = ""
			m.vp.SetContent("")
			return m, tea.Batch(m.spinner.Tick, startSplitCmd(m.opts, m.draft))
		case tea.KeyCtrlC:
			m.canceled = true
			return m, tea.Quit
		case tea.KeyRunes:
			if strings.EqualFold(key.String(), "e") {
				m.inputErr = ""
				return m, m.openEditorCmd()
			}
		}
	}
	return m, nil
}

func (m *model) updateRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key := msg.(type) {
	case killConfirmTimeoutMsg:
		m.confirmKill = false
		m.killKey = ""
		return m, nil
	case tea.KeyMsg:
		switch key.String() {
		case "c":
			if m.resumeCmd != "" {
				m.setFlash(m.copyResumeCommand())
				return m, nil
			}
		case "esc", "q":
			if m.confirmKill && m.killKey == key.String() {
				if key.String() == "q" {
					m.err = ErrQuitAll
				}
				if m.cancelSplit != nil {
					m.cancelSplit()
				}
				return m, tea.Quit
			}
			m.confirmKill = true
			m.killKey = key.String()
			return m, killConfirmTimeoutCmd()
		}
	}
	switch v := msg.(type) {
	case splitLogMsg:
		m.appendLog(v)
		return m, listenStream(m.stream)
	case splitFinishedMsg:
		m.running = false
		m.result = v.result
		m.err = v.err
		m.phase = phaseDone
		m.confirmKill = false
		m.killKey = ""
		m.stream = nil
		if v.err != nil {
			m.appendLog(splitLogMsg{stream: "stderr", text: v.err.Error()})
		}
		return m, nil
	case splitStreamClosedMsg:
		// stream closed without finish message
		m.running = false
		m.phase = phaseDone
		m.confirmKill = false
		m.killKey = ""
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
			m.flash = ""
			m.vp.SetContent("")
			return m, nil
		case "c":
			if m.resumeCmd != "" {
				m.setFlash(m.copyResumeCommand())
				return m, nil
			}
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
	m.captureSessionID(msg.text, msg.stream)
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

func (m *model) openEditorCmd() tea.Cmd {
	path := m.editorPath
	if path == "" {
		tmp, err := os.CreateTemp("", "helm-spec-*.md")
		if err != nil {
			m.inputErr = fmt.Sprintf("Open editor: %v", err)
			return nil
		}
		path = tmp.Name()
		m.editorPath = path
		_ = tmp.Close()
	}
	if err := os.WriteFile(path, []byte(m.draft), 0o600); err != nil {
		m.inputErr = fmt.Sprintf("Save draft for editor: %v", err)
		return nil
	}
	lineno := max(1, strings.Count(m.draft, "\n")+1)
	return openEditor(path, lineno)
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

type killConfirmTimeoutMsg struct{}

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

func killConfirmTimeoutCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return killConfirmTimeoutMsg{} })
}

func (m *model) captureSessionID(line, _ string) {
	if m.sessionID != "" {
		return
	}
	match := sessionIDRe.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 2 {
		return
	}
	m.sessionID = match[1]
	m.resumeCmd = fmt.Sprintf("codex resume %s", m.sessionID)
	m.setFlash(fmt.Sprintf("Resume with: %s (press c to copy)", m.resumeCmd))
}

func (m *model) copyResumeCommand() string {
	if m.resumeCmd == "" {
		return "No session id captured yet."
	}
	if err := clipboard.WriteAll(m.resumeCmd); err != nil {
		return fmt.Sprintf("Clipboard unavailable. Command: %s", m.resumeCmd)
	}
	return fmt.Sprintf("Copied: %s", m.resumeCmd)
}

func (m *model) setFlash(msg string) {
	m.flash = msg
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
