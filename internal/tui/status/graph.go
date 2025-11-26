package status

import (
	"fmt"
	"sort"

	"github.com/polarzero/helm/internal/tui/theme"
)

func buildGraphLines(entries []*entry, selectedID string) []string {
	if len(entries) == 0 {
		return []string{"No specs for this focus."}
	}

	subset := make(map[string]*entry, len(entries))
	for _, e := range entries {
		subset[e.ID] = e
	}

	depCount := make(map[string]int)
	for _, e := range entries {
		for _, dep := range e.DependsOn {
			if _, ok := subset[dep]; ok {
				depCount[dep]++
			}
		}
	}

	var roots []string
	for _, e := range entries {
		if depCount[e.ID] == 0 {
			roots = append(roots, e.ID)
		}
	}
	if len(roots) == 0 {
		for _, e := range entries {
			roots = append(roots, e.ID)
		}
	}
	sort.Strings(roots)

	builder := graphBuilder{subset: subset, selectedID: selectedID}
	for i, id := range roots {
		builder.walk(id, "", i == len(roots)-1, true, nil)
	}
	if len(builder.lines) == 0 {
		return []string{"No specs for this focus."}
	}
	return builder.lines
}

type graphBuilder struct {
	subset     map[string]*entry
	lines      []string
	selectedID string
}

func (b *graphBuilder) walk(id, prefix string, last bool, root bool, stack map[string]struct{}) {
	entry, ok := b.subset[id]
	if !ok {
		return
	}
	label := b.formatEntry(entry)
	if root {
		b.lines = append(b.lines, label)
	} else {
		connector := "├─ "
		if last {
			connector = "└─ "
		}
		b.lines = append(b.lines, prefix+connector+label)
	}

	if stack == nil {
		stack = make(map[string]struct{})
	}
	if _, seen := stack[id]; seen {
		b.lines = append(b.lines, prefix+"↺ "+label+" (cycle)")
		return
	}
	stack[id] = struct{}{}

	children := b.children(entry)
	if len(children) == 0 {
		delete(stack, id)
		return
	}

	var nextPrefix string
	if root {
		if last {
			nextPrefix = "   "
		} else {
			nextPrefix = "│  "
		}
	} else {
		if last {
			nextPrefix = prefix + "   "
		} else {
			nextPrefix = prefix + "│  "
		}
	}

	for i, child := range children {
		b.walk(child, nextPrefix, i == len(children)-1, false, stack)
	}
	delete(stack, id)
}

func (b *graphBuilder) children(e *entry) []string {
	if e == nil {
		return nil
	}
	var children []string
	for _, dep := range e.DependsOn {
		if _, ok := b.subset[dep]; ok {
			children = append(children, dep)
		}
	}
	sort.Strings(children)
	return children
}

func (b *graphBuilder) formatEntry(e *entry) string {
	if e == nil {
		return ""
	}
	cursor := "  "
	if e.ID == b.selectedID {
		cursor = "▶ "
	}
	text := fmt.Sprintf("%s%s %s — %s", cursor, e.BadgeStyle.Render(e.BadgeText), e.ID, e.Name)
	if e.HasUnmetDeps && e.BlockReason != "" {
		text += " " + theme.HintStyle.Render(fmt.Sprintf("(needs: %s)", e.BlockReason))
	}
	return text
}
