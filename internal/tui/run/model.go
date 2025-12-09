package run

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/polarzero/helm/internal/config"
	"github.com/polarzero/helm/internal/metadata"
	"github.com/polarzero/helm/internal/specs"
	"github.com/polarzero/helm/internal/tui/components"
	"github.com/polarzero/helm/internal/tui/theme"
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

	prog := tea.NewProgram(mdl, tea.WithAltScreen(), tea.WithMouseCellMotion())
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
		width:    80,
		height:   24,
	}

	m.viewport.MouseWheelEnabled = true
	m.resize()
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
		return m, tea.ClearScreen
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
	needClear := false
	switch msg := msg.(type) {
	case tea.KeyMsg:
		msg = components.NormalizeKey(msg)
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
			needClear = true
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
	if needClear {
		return m, tea.Batch(cmd, tea.ClearScreen)
	}
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
		msg = components.NormalizeKey(msg)
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
		msg = components.NormalizeKey(msg)
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
	var view string
	switch m.phase {
	case phaseList:
		view = m.listView()
	case phaseRunning:
		view = m.runningView()
	case phaseResult:
		view = m.resultView()
	default:
		view = ""
	}
	view = components.ClampHeight(view, m.height)
	return components.PadToHeight(view, m.height)
}

func (m *model) resize() {
	if m.width == 0 || m.height == 0 {
		return
	}

	bodyArea := components.ContentArea(m.width, m.height)

	switch m.phase {
	case phaseList:
		chrome := m.listChromeHeight()
		_, listArea := components.SplitVertical(bodyArea, components.Fixed(chrome))
		available := listArea.Dy()
		minHeight := max(3, bodyArea.Dy())
		if available < 3 {
			available = minHeight
		}
		m.list.SetSize(bodyArea.Dx(), available)
	case phaseRunning, phaseResult:
		chrome := m.logsChromeHeight()
		_, logsArea := components.SplitVertical(bodyArea, components.Fixed(chrome))
		available := logsArea.Dy()
		if available < 3 {
			available = 3 // keep a minimal viewport while avoiding overflow amplification
		}
		m.viewport.Width = components.ViewportInnerWidth(m.width, theme.DefaultCardBorder)
		m.viewport.Height = available
		if len(m.logs) > 0 {
			m.viewport.SetContent(components.FitStyledContent(buildLogContent(m.logs), m.viewport.Width, true, "…"))
		}

		// Measure the rendered view; if it still overflows the terminal, shrink
		// the viewport by the overflow amount (clamped to a small minimum).
		rendered := ""
		switch m.phase {
		case phaseRunning:
			rendered = m.runningView()
		case phaseResult:
			rendered = m.resultView()
		}
		if rendered != "" {
			if over := lipgloss.Height(rendered) - m.height; over > 0 && m.viewport.Height > 3 {
				newHeight := max(3, m.viewport.Height-over)
				if newHeight < m.viewport.Height {
					m.viewport.Height = newHeight
					if len(m.logs) > 0 {
						m.viewport.SetContent(components.FitStyledContent(buildLogContent(m.logs), m.viewport.Width, true, "…"))
					}
				}
			}
		}
	}
}

// listChromeHeight measures all non-list chrome for the list phase at current width.
func (m *model) listChromeHeight() int {
	filterLabel := "all specs"
	if m.filterRunnable {
		filterLabel = "runnable only"
	}

	bodyParts := []string{""} // placeholder where the list content will render
	if m.confirmUnmet {
		deps := "None"
		if item := m.currentItem(); item != nil {
			if len(item.folder.UnmetDeps) > 0 {
				deps = strings.Join(item.folder.UnmetDeps, ", ")
			}
		}
		bodyParts = append(bodyParts, components.Modal(components.ModalConfig{
			Width: m.width,
			Title: "Run with unmet dependencies?",
			Body: []string{
				fmt.Sprintf("Unmet deps: %s", deps),
				"Press y to run anyway or n/esc to cancel.",
			},
		}))
	}
	if m.flash != "" {
		bodyParts = append(bodyParts, components.Flash(components.FlashInfo, m.flash))
	}

	help := []components.HelpEntry{
		{Key: "↑/↓", Label: "move"},
		{Key: "enter", Label: "run"},
		{Key: "f", Label: fmt.Sprintf("filter (%s)", filterLabel)},
		{Key: "q", Label: "quit"},
	}
	if m.confirmUnmet {
		help = append(help,
			components.HelpEntry{Key: "y", Label: "confirm"},
			components.HelpEntry{Key: "n/esc", Label: "cancel"},
		)
	}

	title := components.TitleBar(components.TitleConfig{Title: "helm run"})
	body := strings.Join(bodyParts, "\n\n")
	helpBar := components.HelpBar(components.ContentWidth(m.width), help...)

	sections := make([]string, 0, 3)
	if strings.TrimSpace(title) != "" {
		sections = append(sections, title)
	}
	if strings.TrimSpace(body) != "" {
		sections = append(sections, body)
	}
	if strings.TrimSpace(helpBar) != "" {
		sections = append(sections, helpBar)
	}
	return lipgloss.Height(strings.Join(sections, "\n\n"))
}

// logsChromeHeight measures chrome surrounding the log viewport for running/result phases.
func (m *model) logsChromeHeight() int {
	var sections []string

	title := ""
	switch m.phase {
	case phaseRunning:
		if m.running != nil && m.running.spec != nil && m.running.spec.Metadata != nil {
			title = components.TitleBar(components.TitleConfig{
				Title: fmt.Sprintf("Running %s — %s", m.running.spec.Metadata.ID, m.running.spec.Metadata.Name),
			})
		}
	case phaseResult:
		if m.result != nil {
			title = components.TitleBar(components.TitleConfig{
				Title: fmt.Sprintf("Run result — %s", m.result.specID),
			})
		}
	}
	if strings.TrimSpace(title) != "" {
		sections = append(sections, title)
	}

	bodyParts := []string{}
	if m.phase == phaseRunning {
		attemptLine := "Waiting for attempts to start"
		stage := ""
		if m.running != nil {
			stage = strings.TrimSpace(m.running.stage)
			if stage != "" {
				stage = cases.Title(language.Und).String(stage)
			}
		}
		if m.running != nil && m.running.attempt > 0 && m.running.totalAttempts > 0 {
			if stage != "" {
				attemptLine = fmt.Sprintf("%s — attempt %d of %d", stage, m.running.attempt, m.running.totalAttempts)
			} else {
				attemptLine = fmt.Sprintf("Attempt %d of %d", m.running.attempt, m.running.totalAttempts)
			}
		} else if stage != "" {
			attemptLine = stage
		} else if m.running != nil && m.running.started {
			attemptLine = "Streaming Codex logs..."
		}
		bodyParts = append(bodyParts, components.SpinnerLine(m.spinner.View(), attemptLine))
		if m.running != nil {
			if chip := components.ResumeChip(m.running.resumeCmd); chip != "" {
				bodyParts = append(bodyParts, chip)
			}
		}
	} else if m.phase == phaseResult {
		if m.result != nil {
			if m.result.err != nil {
				bodyParts = append(bodyParts, components.Flash(components.FlashDanger, fmt.Sprintf("Error: %v", m.result.err)))
			} else {
				statusLabel := strings.ToUpper(string(m.result.status))
				if statusLabel == "" {
					statusLabel = "UNKNOWN"
				}
				bodyParts = append(bodyParts, fmt.Sprintf("Spec status: %s", statusLabel))
				if m.result.exitErr != nil {
					bodyParts = append(bodyParts, components.Flash(components.FlashDanger, fmt.Sprintf("implement-spec exited with code %d: %v", m.result.exitCode, m.result.exitErr)))
				} else {
					bodyParts = append(bodyParts, components.Flash(components.FlashSuccess, "implement-spec exited successfully."))
				}
			}
			if len(m.result.remaining) > 0 {
				bodyParts = append(bodyParts, components.BulletList(m.result.remaining))
			}
			if chip := components.ResumeChip(m.result.resumeCmd); chip != "" {
				bodyParts = append(bodyParts, chip)
			}
		}
	}

	if m.flash != "" {
		bodyParts = append(bodyParts, components.Flash(components.FlashInfo, m.flash))
	}

	if m.phase == phaseRunning && m.confirmKill {
		bodyParts = append(bodyParts, components.Modal(components.ModalConfig{
			Width: m.width,
			Title: "Stop the current run?",
			Body: []string{
				"implement-spec will be terminated.",
				"Press ESC again within 2s to stop; press Q twice to quit Helm.",
			},
		}))
	}

	var help []components.HelpEntry
	if m.phase == phaseRunning {
		help = []components.HelpEntry{
			{Key: "↑/↓ PgUp/PgDn", Label: "scroll"},
			{Key: "mouse", Label: "scroll"},
			{Key: "c", Label: "copy resume"},
			{Key: "esc×2", Label: "stop run"},
			{Key: "q×2", Label: "quit"},
		}
	} else {
		help = []components.HelpEntry{
			{Key: "enter/r", Label: "back to list"},
			{Key: "c", Label: "copy resume"},
			{Key: "q", Label: "quit"},
		}
	}

	body := strings.Join(bodyParts, "\n\n")
	helpBar := components.HelpBar(components.ContentWidth(m.width), help...)

	if strings.TrimSpace(body) != "" {
		sections = append(sections, body)
	}
	if strings.TrimSpace(helpBar) != "" {
		sections = append(sections, helpBar)
	}

	chromeHeight := lipgloss.Height(strings.Join(sections, "\n\n"))
	hasStatus := true
	return chromeHeight + components.ViewportChromeHeight(m.width, theme.DefaultCardBorder, hasStatus)
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
	m.running = &runState{
		spec:          folder,
		attempt:       1,
		totalAttempts: resolveMaxAttempts(m.opts.Settings),
		stage:         "implementing",
		started:       true,
	}
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
	wasAtBottom := m.viewport.AtBottom()
	content := buildLogContent(m.logs)
	clamped := components.FitStyledContent(content, m.viewport.Width, true, "…")
	m.viewport.SetContent(clamped)
	if wasAtBottom {
		m.viewport.GotoBottom()
	}
	if m.running != nil {
		if attempt, total, ok := parseAttemptLine(entry.text); ok {
			m.running.attempt = attempt
			m.running.totalAttempts = total
		}
		if stage := classifyStage(entry.text); stage != "" {
			m.running.stage = stage
		}
		// consider the run "started" as soon as any log arrives, so the spinner line
		// can fall back to a useful message even if attempt/stage markers are absent.
		if !m.running.started {
			m.running.started = true
		}
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

var stageRe = regexp.MustCompile(`(?i)^stage:\s*(implementing|verifying)`)

func classifyStage(line string) string {
	m := stageRe.FindStringSubmatch(strings.TrimSpace(line))
	if len(m) == 2 {
		return strings.ToLower(m[1])
	}
	return ""
}

// resolveMaxAttempts returns the configured max attempts from settings or env.
func resolveMaxAttempts(settings *config.Settings) int {
	if settings == nil {
		return 0
	}
	if v := strings.TrimSpace(os.Getenv("MAX_ATTEMPTS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	if settings.DefaultMaxAttempts > 0 {
		return settings.DefaultMaxAttempts
	}
	return 0
}

type logEntry struct {
	stream string
	text   string
}

type runState struct {
	spec          *specs.SpecFolder
	attempt       int
	totalAttempts int
	started       bool
	stage         string
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
