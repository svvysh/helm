package status

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/polarzero/helm/internal/metadata"
	"github.com/polarzero/helm/internal/specs"
	"github.com/polarzero/helm/internal/tui/components"
	"github.com/polarzero/helm/internal/tui/theme"
)

// ErrQuitAll signals the caller to quit the entire CLI.
var ErrQuitAll = errors.New("status quit")

// Options configure the status TUI.
type Options struct {
	SpecsRoot string
}

// Run launches the status overview TUI.
func Run(opts Options) error {
	if opts.SpecsRoot == "" {
		return errors.New("status: specs root is required")
	}

	folders, err := loadFolders(opts.SpecsRoot)
	if err != nil {
		return err
	}

	mdl := newModel(opts, folders)
	prog := tea.NewProgram(mdl, tea.WithAltScreen())
	res, err := prog.Run()
	if err != nil {
		return err
	}
	final, ok := res.(*model)
	if !ok {
		return errors.New("status: unexpected program result")
	}
	return final.err
}

type viewMode int

const (
	viewTable viewMode = iota
	viewGraph
)

type focusMode int

const (
	focusAll focusMode = iota
	focusRunnable
	focusSubtree
)

type model struct {
	opts Options

	table         table.Model
	graphViewport viewport.Model

	viewMode  viewMode
	focusMode focusMode

	focusTarget string

	width  int
	height int

	entries   []*entry
	entryByID map[string]*entry
	visible   []*entry
	summary   summaryCounts

	infoMessage string

	err error
}

type entry struct {
	folder          *specs.SpecFolder
	ID              string
	Name            string
	DependsOn       []string
	Dependents      []string
	DepsDisplay     string
	LastRunDisplay  string
	BlockReason     string
	CanRun          bool
	BadgeText       string
	BadgeStyle      lipgloss.Style
	StatusCategory  theme.StatusCategory
	HasUnmetDeps    bool
	LastRunUnixNano int64
}

type summaryCounts struct {
	counts map[theme.StatusCategory]int
	total  int
}

func loadFolders(root string) ([]*specs.SpecFolder, error) {
	folders, err := specs.DiscoverSpecs(root)
	if err != nil {
		return nil, err
	}
	specs.ComputeDependencyState(folders)
	return folders, nil
}

func newModel(opts Options, folders []*specs.SpecFolder) *model {
	entries := buildEntries(folders)
	linkDependents(entries)
	entryByID := make(map[string]*entry, len(entries))
	for _, e := range entries {
		entryByID[e.ID] = e
	}

	tbl := table.New(
		table.WithColumns(defaultColumns()),
		table.WithFocused(true),
	)
	tbl.SetStyles(components.TableStyles())

	vp := viewport.New(0, 0)

	m := &model{
		opts:          opts,
		table:         tbl,
		graphViewport: vp,
		viewMode:      viewTable,
		focusMode:     focusAll,
		entries:       entries,
		entryByID:     entryByID,
		summary:       buildSummary(entries),
	}
	m.refreshVisible("")
	return m
}

func buildEntries(folders []*specs.SpecFolder) []*entry {
	if len(folders) == 0 {
		return nil
	}
	out := make([]*entry, 0, len(folders))
	for _, folder := range folders {
		if folder == nil || folder.Metadata == nil {
			continue
		}
		blocked := len(folder.UnmetDeps) > 0 && folder.Metadata.Status != metadata.StatusDone
		label, style, category := theme.StatusBadge(folder.Metadata, blocked)
		ent := &entry{
			folder:         folder,
			ID:             folder.Metadata.ID,
			Name:           folder.Metadata.Name,
			DependsOn:      cloneStrings(folder.Metadata.DependsOn),
			DepsDisplay:    renderDeps(folder.Metadata.DependsOn),
			LastRunDisplay: renderLastRun(folder.Metadata.LastRun),
			BlockReason:    strings.Join(folder.UnmetDeps, ", "),
			CanRun:         folder.CanRun,
			BadgeText:      label,
			BadgeStyle:     style,
			StatusCategory: category,
			HasUnmetDeps:   len(folder.UnmetDeps) > 0,
		}
		if folder.Metadata.LastRun != nil {
			ent.LastRunUnixNano = folder.Metadata.LastRun.UnixNano()
		}
		out = append(out, ent)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func linkDependents(entries []*entry) {
	if len(entries) == 0 {
		return
	}
	index := make(map[string]*entry, len(entries))
	for _, e := range entries {
		if e == nil || e.ID == "" {
			continue
		}
		index[e.ID] = e
	}
	for _, e := range entries {
		if e == nil {
			continue
		}
		for _, dep := range e.DependsOn {
			if depEntry, ok := index[dep]; ok {
				depEntry.Dependents = append(depEntry.Dependents, e.ID)
			}
		}
	}
	for _, e := range entries {
		if len(e.Dependents) == 0 {
			continue
		}
		sort.Strings(e.Dependents)
		e.Dependents = dedupeSortedStrings(e.Dependents)
	}
}

func dedupeSortedStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	n := 1
	for i := 1; i < len(values); i++ {
		if values[i] == values[n-1] {
			continue
		}
		values[n] = values[i]
		n++
	}
	return values[:n]
}

func renderDeps(depends []string) string {
	if len(depends) == 0 {
		return "-"
	}
	return strings.Join(depends, ", ")
}

func renderLastRun(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.In(time.Now().Location()).Format("2006-01-02 15:04")
}

func buildSummary(entries []*entry) summaryCounts {
	counts := map[theme.StatusCategory]int{
		theme.StatusTodo:       0,
		theme.StatusInProgress: 0,
		theme.StatusDone:       0,
		theme.StatusBlocked:    0,
		theme.StatusFailed:     0,
	}
	for _, e := range entries {
		counts[e.StatusCategory]++
	}
	return summaryCounts{counts: counts, total: len(entries)}
}

func defaultColumns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 16},
		{Title: "Name", Width: 24},
		{Title: "Status", Width: 14},
		{Title: "Deps", Width: 24},
		{Title: "Last Run", Width: 18},
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "q":
			m.err = ErrQuitAll
			return m, tea.Quit
		case "tab", "shift+tab":
			m.toggleView()
			return m, nil
		case "f":
			m.cycleFocus(m.currentSelectionID())
			return m, nil
		case "enter":
			if m.viewMode == viewTable && m.setSubtreeFromSelection() {
				return m, nil
			}
		case "r":
			m.reloadData()
			return m, nil
		}
	}

	if m.viewMode == viewTable {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.graphViewport, cmd = m.graphViewport.Update(msg)
	return m, cmd
}

func (m *model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	contentHeight := m.height - 6
	if contentHeight < 5 {
		contentHeight = m.height - 3
		if contentHeight < 3 {
			contentHeight = 3
		}
	}

	m.table.SetHeight(contentHeight)
	m.graphViewport.Height = contentHeight
	m.graphViewport.Width = max(20, m.width-2)

	m.table.SetColumns(m.computeColumns())
	m.graphViewport.SetContent(strings.Join(buildGraphLines(m.visible), "\n"))
}

func (m *model) refreshVisible(preserve string) {
	m.visible = m.filterEntries()

	rows := make([]table.Row, 0, len(m.visible))
	for _, e := range m.visible {
		rows = append(rows, table.Row{
			e.ID,
			e.Name,
			e.BadgeStyle.Render(e.BadgeText),
			e.DepsDisplay,
			e.LastRunDisplay,
		})
	}
	m.table.SetRows(rows)
	m.selectRow(preserve)

	content := buildGraphLines(m.visible)
	m.graphViewport.SetContent(strings.Join(content, "\n"))
	if len(content) == 0 {
		m.graphViewport.SetContent("No specs discovered.")
	}
	m.graphViewport.GotoTop()
}

func (m *model) filterEntries() []*entry {
	switch m.focusMode {
	case focusAll:
		return cloneEntries(m.entries)
	case focusRunnable:
		var out []*entry
		for _, e := range m.entries {
			if e.CanRun {
				out = append(out, e)
			}
		}
		return out
	case focusSubtree:
		if m.focusTarget == "" {
			return nil
		}
		set := m.collectSubtree(m.focusTarget)
		if len(set) == 0 {
			return nil
		}
		var out []*entry
		for _, e := range m.entries {
			if _, ok := set[e.ID]; ok {
				out = append(out, e)
			}
		}
		return out
	default:
		return cloneEntries(m.entries)
	}
}

func cloneEntries(entries []*entry) []*entry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]*entry, len(entries))
	copy(out, entries)
	return out
}

func (m *model) selectRow(preserve string) {
	if len(m.visible) == 0 {
		m.table.SetCursor(0)
		return
	}
	if preserve != "" {
		for idx, e := range m.visible {
			if e.ID == preserve {
				m.table.SetCursor(idx)
				return
			}
		}
	}
	m.table.SetCursor(0)
}

func (m *model) toggleView() {
	if m.viewMode == viewTable {
		m.viewMode = viewGraph
	} else {
		m.viewMode = viewTable
	}
}

func (m *model) cycleFocus(preserve string) {
	next := (int(m.focusMode) + 1) % 3
	mode := focusMode(next)
	if mode == focusSubtree {
		target := m.focusTarget
		if target == "" {
			target = preserve
			if target == "" && len(m.entries) > 0 {
				target = m.entries[0].ID
			}
		}
		if target == "" {
			mode = focusAll
		} else {
			m.focusTarget = target
		}
	}
	m.focusMode = mode
	m.refreshVisible(preserve)
}

func (m *model) setSubtreeFromSelection() bool {
	id := m.currentSelectionID()
	if id == "" {
		return false
	}
	if _, ok := m.entryByID[id]; !ok {
		return false
	}
	m.focusMode = focusSubtree
	m.focusTarget = id
	m.refreshVisible(id)
	return true
}

func (m *model) currentSelectionID() string {
	if len(m.table.Rows()) == 0 {
		return ""
	}
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return ""
	}
	return row[0]
}

func (m *model) collectSubtree(root string) map[string]struct{} {
	visited := make(map[string]struct{})
	var walk func(id string)
	walk = func(id string) {
		if id == "" {
			return
		}
		if _, ok := visited[id]; ok {
			return
		}
		entry, ok := m.entryByID[id]
		if !ok {
			return
		}
		visited[id] = struct{}{}
		for _, dep := range entry.Dependents {
			walk(dep)
		}
	}
	walk(root)
	return visited
}

func (m *model) reloadData() {
	folders, err := loadFolders(m.opts.SpecsRoot)
	if err != nil {
		m.infoMessage = fmt.Sprintf("Reload failed: %v", err)
		return
	}
	entries := buildEntries(folders)
	linkDependents(entries)
	m.entries = entries
	m.entryByID = make(map[string]*entry, len(entries))
	for _, e := range entries {
		m.entryByID[e.ID] = e
	}
	m.summary = buildSummary(entries)

	if m.focusMode == focusSubtree && m.focusTarget != "" {
		if _, ok := m.entryByID[m.focusTarget]; !ok {
			m.focusTarget = ""
			m.focusMode = focusAll
		}
	}
	preserve := m.currentSelectionID()
	m.refreshVisible(preserve)
	m.infoMessage = fmt.Sprintf("Reloaded %d spec(s)", len(entries))
}

func (m *model) computeColumns() []table.Column {
	total := m.width - 4
	if total < 40 {
		total = 40
	}

	idWidth := 16
	statusWidth := 14
	lastRunWidth := 18
	depsWidth := 22

	nameWidth := total - (idWidth + statusWidth + lastRunWidth + depsWidth)
	if nameWidth < 20 {
		shortage := 20 - nameWidth
		depsWidth -= shortage
		if depsWidth < 12 {
			extra := 12 - depsWidth
			lastRunWidth -= extra
			depsWidth = 12
			if lastRunWidth < 14 {
				lastRunWidth = 14
			}
		}
		nameWidth = 20
	}

	return []table.Column{
		{Title: "ID", Width: idWidth},
		{Title: "Name", Width: nameWidth},
		{Title: "Status", Width: statusWidth},
		{Title: "Deps", Width: depsWidth},
		{Title: "Last Run", Width: lastRunWidth},
	}
}

func (m *model) View() string {
	return renderView(m)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
