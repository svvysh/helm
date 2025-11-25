package run

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/polarzero/helm/internal/config"
	"github.com/polarzero/helm/internal/metadata"
	"github.com/polarzero/helm/internal/specs"
	"github.com/polarzero/helm/internal/tui/components"
)

// Options configures the run TUI.
type Options struct {
	Root      string
	SpecsRoot string
	Settings  *config.Settings
}

// Run launches the helm run TUI.
func Run(opts Options) error {
	mdl, err := newModel(opts)
	if err != nil {
		return err
	}

	prog := tea.NewProgram(mdl, tea.WithAltScreen())
	res, err := prog.Run()
	if err != nil {
		return err
	}
	finalModel, ok := res.(*model)
	if !ok {
		return errors.New("unexpected program result")
	}
	if finalModel.err != nil {
		return finalModel.err
	}
	return nil
}

type phase int

const (
	phaseList phase = iota
	phaseRunning
	phaseResult
)

var (
	sessionIDRe = regexp.MustCompile(`(?i)^session id:\s*([a-f0-9-]{36})$`)
	ErrQuitAll  = errors.New("run quit")
)

type model struct {
	opts           Options
	phase          phase
	list           list.Model
	spinner        spinner.Model
	specs          []*specs.SpecFolder
	filterRunnable bool
	confirmUnmet   bool
	width, height  int
	err            error

	viewport viewport.Model
	logs     []logEntry
	logLimit int
	stream   <-chan tea.Msg

	running     *runState
	result      *resultState
	confirmKill bool
	killKey     string
	flash       string
}

func newModel(opts Options) (*model, error) {
	if opts.Settings == nil {
		return nil, errors.New("run settings are required")
	}
	root := opts.Root
	if root == "" {
		root = "."
	}
	opts.Root = root
	specsRoot := opts.SpecsRoot
	if specsRoot == "" {
		specsRoot = config.ResolveSpecsRoot(root, opts.Settings.SpecsRoot)
	}
	opts.SpecsRoot = specsRoot

	folders, err := discoverAllSpecs(specsRoot)
	if err != nil {
		return nil, err
	}

	items := specItemsFromFolders(folders)
	lst := list.New(items, newDelegate(), 0, 0)
	lst.Title = "helm run — specs"
	lst.SetShowStatusBar(false)
	lst.SetShowPagination(false)
	lst.SetFilteringEnabled(false)
	lst.SetShowHelp(false)

	vp := viewport.New(0, 0)
	sp := components.NewSpinner()

	m := &model{
		opts:     opts,
		phase:    phaseList,
		list:     lst,
		spinner:  sp,
		specs:    folders,
		viewport: vp,
		logLimit: 2000,
	}
	return m, nil
}

func discoverAllSpecs(root string) ([]*specs.SpecFolder, error) {
	folders, err := specs.DiscoverSpecs(root)
	if err != nil {
		return nil, err
	}
	specs.ComputeDependencyState(folders)
	return folders, nil
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	}

	var cmd tea.Cmd
	switch m.phase {
	case phaseList:
		_, cmd = m.updateList(msg)
	case phaseRunning:
		_, cmd = m.updateRunning(msg)
	case phaseResult:
		_, cmd = m.updateResult(msg)
	default:
		return m, nil
	}

	if m.phase == phaseRunning {
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		if cmd != nil || spinnerCmd != nil {
			return m, tea.Batch(cmd, spinnerCmd)
		}
		return m, spinnerCmd
	}
	return m, cmd
}

func (m *model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.confirmUnmet {
				m.confirmUnmet = false
				return m, nil
			}
			return m, tea.Quit
		case "ctrl+c", "q":
			m.err = ErrQuitAll
			return m, tea.Quit
		case "f":
			preserve := ""
			if item := m.currentItem(); item != nil && item.folder.Metadata != nil {
				preserve = item.folder.Metadata.ID
			}
			m.filterRunnable = !m.filterRunnable
			m.confirmUnmet = false
			m.refreshItems(preserve)
			return m, nil
		case "up", "down", "k", "j", "pgup", "pgdown":
			if m.confirmUnmet {
				m.confirmUnmet = false
			}
		case "enter":
			if m.confirmUnmet {
				return m, nil
			}
			item := m.currentItem()
			if item == nil {
				return m, nil
			}
			if !canStartRun(item.folder) {
				return m, nil
			}
			if len(item.folder.UnmetDeps) > 0 {
				m.confirmUnmet = true
				return m, nil
			}
			return m, m.startRun(item.folder)
		case "y":
			if m.confirmUnmet {
				m.confirmUnmet = false
				item := m.currentItem()
				if item != nil {
					if !canStartRun(item.folder) {
						return m, nil
					}
					return m, m.startRun(item.folder)
				}
			}
		case "n":
			if m.confirmUnmet {
				m.confirmUnmet = false
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) updateRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.running == nil {
		return m, nil
	}

	switch msg := msg.(type) {
	case killConfirmTimeoutMsg:
		m.confirmKill = false
		m.killKey = ""
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if m.confirmKill && m.killKey == msg.String() {
				m.killProcess()
				// Exit back to caller; run keeps streaming but TUI closes.
				if msg.String() == "q" {
					m.err = ErrQuitAll
				}
				return m, tea.Quit
			}
			m.confirmKill = true
			m.killKey = msg.String()
			return m, killConfirmTimeoutCmd()
		case "c":
			if m.running != nil && m.running.resumeCmd != "" {
				m.setFlash(m.copyResumeCommand())
				return m, nil
			}
		case "n":
			if m.confirmKill {
				m.confirmKill = false
				return m, nil
			}
		}
	}

	switch msg := msg.(type) {
	case runnerStartMsg:
		if msg.err != nil {
			specID, specName := "", ""
			if m.running != nil && m.running.spec != nil && m.running.spec.Metadata != nil {
				specID = m.running.spec.Metadata.ID
				specName = m.running.spec.Metadata.Name
			}
			m.phase = phaseResult
			m.result = &resultState{specID: specID, specName: specName, err: msg.err}
			m.confirmKill = false
			m.resize()
			m.running = nil
			return m, nil
		}
		m.stream = msg.stream
		return m, m.listenForLogs()
	case runnerLogMsg:
		m.appendLog(msg)
		return m, m.listenForLogs()
	case runnerFinishedMsg:
		if m.running != nil {
			m.running.finished = true
			m.running.exitErr = msg.err
			m.running.exitCode = msg.exitCode
		}
		m.stream = nil
		m.phase = phaseResult
		result := &resultState{}
		if m.running != nil && m.running.spec != nil && m.running.spec.Metadata != nil {
			result.specID = m.running.spec.Metadata.ID
			result.specName = m.running.spec.Metadata.Name
			result.resumeCmd = m.running.resumeCmd
		}
		result.exitErr = msg.err
		result.exitCode = msg.exitCode
		m.result = result
		m.confirmKill = false
		if err := m.refreshSpecs(result.specID); err != nil {
			result.err = err
		} else {
			folder := m.findSpec(result.specID)
			if folder != nil && folder.Metadata != nil {
				result.status = folder.Metadata.Status
				result.specName = folder.Metadata.Name
				report := filepath.Join(folder.Path, "implementation-report.md")
				result.remaining = parseRemainingTasks(report)
			}
		}
		m.resize()
		return m, nil
	case runnerStreamClosedMsg:
		m.stream = nil
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			if msg.String() == "q" {
				m.err = ErrQuitAll
			}
			return m, tea.Quit
		case "c":
			if m.result != nil && m.result.resumeCmd != "" {
				m.setFlash(m.copyResumeCommandResult())
				return m, nil
			}
		case "enter", "r":
			m.phase = phaseList
			m.result = nil
			m.running = nil
			m.logs = nil
			m.flash = ""
			m.viewport.SetContent("")
			m.confirmKill = false
			m.resize()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	switch m.phase {
	case phaseList:
		return m.listView()
	case phaseRunning:
		return m.runningView()
	case phaseResult:
		return m.resultView()
	default:
		return ""
	}
}

func (m *model) resize() {
	if m.width == 0 || m.height == 0 {
		return
	}
	switch m.phase {
	case phaseList:
		h := m.height - 2
		if h < 3 {
			h = m.height
		}
		m.list.SetSize(m.width, h)
	case phaseRunning, phaseResult:
		header := 7
		h := m.height - header
		if h < 3 {
			h = m.height
		}
		m.viewport.Width = m.width
		m.viewport.Height = h
	}
}

func (m *model) refreshItems(preserve string) {
	m.list.SetItems(filterItems(m.specs, m.filterRunnable))
	if preserve != "" {
		m.selectByID(preserve)
	}
}

func (m *model) refreshSpecs(preserve string) error {
	folders, err := discoverAllSpecs(m.opts.SpecsRoot)
	if err != nil {
		return err
	}
	m.specs = folders
	m.refreshItems(preserve)
	return nil
}

func (m *model) currentItem() *specItem {
	item, ok := m.list.SelectedItem().(specItem)
	if !ok {
		return nil
	}
	return &item
}

func (m *model) startRun(folder *specs.SpecFolder) tea.Cmd {
	m.phase = phaseRunning
	m.confirmUnmet = false
	m.logs = nil
	m.flash = ""
	m.viewport.SetContent("")
	m.viewport.GotoTop()
	m.running = &runState{spec: folder}
	m.resize()
	return tea.Batch(m.spinner.Tick, startRunnerCmd(m.opts, folder))
}

func (m *model) appendLog(msg runnerLogMsg) {
	if msg.text == "" {
		return
	}
	trimmed := strings.TrimRight(msg.text, "\n")
	entry := logEntry{stream: msg.stream, text: trimmed}
	m.captureSessionID(trimmed, msg.stream)
	m.logs = append(m.logs, entry)
	if len(m.logs) > m.logLimit {
		m.logs = m.logs[len(m.logs)-m.logLimit:]
	}
	content := buildLogContent(m.logs)
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()

	if attempt, total, ok := parseAttemptLine(entry.text); ok {
		m.running.attempt = attempt
		m.running.totalAttempts = total
	}
}

func (m *model) killProcess() {
	// Go runner is not cancelable yet; just clear the confirmation flag.
	m.confirmKill = false
}

func (m *model) selectByID(id string) {
	if id == "" {
		return
	}
	for idx, item := range m.list.Items() {
		specItem, ok := item.(specItem)
		if !ok || specItem.folder.Metadata == nil {
			continue
		}
		if specItem.folder.Metadata.ID == id {
			m.list.Select(idx)
			return
		}
	}
}

func canStartRun(folder *specs.SpecFolder) bool {
	if folder == nil || folder.Metadata == nil {
		return false
	}
	switch folder.Metadata.Status {
	case metadata.StatusDone, metadata.StatusInProgress:
		return false
	default:
		return true
	}
}

func (m *model) findSpec(id string) *specs.SpecFolder {
	for _, folder := range m.specs {
		if folder.Metadata != nil && folder.Metadata.ID == id {
			return folder
		}
	}
	return nil
}

func (m *model) listenForLogs() tea.Cmd {
	if m.stream == nil {
		return nil
	}
	return listenStream(m.stream)
}

func (m *model) captureSessionID(line, _ string) {
	if m.running == nil || m.running.sessionID != "" {
		return
	}
	match := sessionIDRe.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 2 {
		return
	}
	id := match[1]
	m.running.sessionID = id
	m.running.resumeCmd = fmt.Sprintf("codex resume %s", id)
	m.setFlash(fmt.Sprintf("Resume with: %s (press c to copy)", m.running.resumeCmd))
}

func (m *model) copyResumeCommand() string {
	if m.running == nil || m.running.resumeCmd == "" {
		return "No session id captured yet."
	}
	if err := clipboard.WriteAll(m.running.resumeCmd); err != nil {
		return fmt.Sprintf("Clipboard unavailable. Command: %s", m.running.resumeCmd)
	}
	return fmt.Sprintf("Copied: %s", m.running.resumeCmd)
}

func (m *model) copyResumeCommandResult() string {
	if m.result == nil || m.result.resumeCmd == "" {
		return "No session id captured yet."
	}
	if err := clipboard.WriteAll(m.result.resumeCmd); err != nil {
		return fmt.Sprintf("Clipboard unavailable. Command: %s", m.result.resumeCmd)
	}
	return fmt.Sprintf("Copied: %s", m.result.resumeCmd)
}

func (m *model) setFlash(msg string) {
	m.flash = msg
}

func killConfirmTimeoutCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return killConfirmTimeoutMsg{} })
}

func buildLogContent(entries []logEntry) string {
	var b strings.Builder
	for i, entry := range entries {
		prefix := "stdout"
		if entry.stream == "stderr" {
			prefix = "stderr"
		}
		b.WriteString(prefix)
		b.WriteString(": ")
		b.WriteString(entry.text)
		if i < len(entries)-1 {
			b.WriteRune('\n')
		}
	}
	return b.String()
}

type logEntry struct {
	stream string
	text   string
}

type runState struct {
	spec          *specs.SpecFolder
	attempt       int
	totalAttempts int
	finished      bool
	exitErr       error
	exitCode      int
	sessionID     string
	resumeCmd     string
}

type resultState struct {
	specID    string
	specName  string
	status    metadata.SpecStatus
	exitErr   error
	exitCode  int
	err       error
	remaining []string
	resumeCmd string
}

type killConfirmTimeoutMsg struct{}
