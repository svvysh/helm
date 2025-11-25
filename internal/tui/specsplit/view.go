package specsplit

import (
	"fmt"
	"strings"

	splitting "github.com/polarzero/helm/internal/specsplit"
	"github.com/polarzero/helm/internal/tui/components"
)

func introView() string {
	body := strings.Join([]string{
		"This command collects a large spec (paste or file) and streams Codex progress while it generates smaller Helm specs.",
		"",
		"Paste a spec on the next screen to get started.",
	}, "\n")
	help := []components.HelpEntry{
		{Key: "enter", Label: "begin"},
		{Key: "q/esc", Label: "quit"},
	}
	return components.PageShell(components.PageShellOptions{
		Title:       components.TitleConfig{Title: "helm spec — split large specs"},
		Body:        body,
		HelpEntries: help,
	})
}

func inputView(m *model) string {
	lines := []string{
		"Press e to open your editor and paste the large spec. Enter will start splitting once the draft is non-empty.",
	}
	if m.opts.PlanPath != "" {
		lines = append(lines, fmt.Sprintf("Dev mode: plan will be loaded from %s", m.opts.PlanPath))
	}
	if m.inputErr != "" {
		lines = append(lines, components.Flash(components.FlashDanger, fmt.Sprintf("Error: %s", m.inputErr)))
	}
	body, status := previewDraft(m.draft)
	lines = append(lines, "", components.ViewportCard(components.ViewportCardOptions{
		Width:   m.width,
		Content: body,
		Status:  status,
	}))
	help := []components.HelpEntry{
		{Key: "enter", Label: "split"},
		{Key: "e", Label: "edit in $EDITOR"},
		{Key: "q", Label: "quit"},
		{Key: "esc", Label: "back"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       m.width,
		Title:       components.TitleConfig{Title: "Provide the large spec"},
		Body:        strings.Join(lines, "\n"),
		HelpEntries: help,
	})
}

func runningView(m *model) string {
	status := "Splitting via Codex..."
	if m.opts.PlanPath != "" {
		status = fmt.Sprintf("Reading plan from %s...", m.opts.PlanPath)
	}
	lines := []string{
		components.SpinnerLine(m.spinner.View(), status),
	}
	if chip := components.ResumeChip(m.resumeCmd); chip != "" {
		lines = append(lines, chip)
	}
	if m.flash != "" {
		lines = append(lines, components.Flash(components.FlashInfo, m.flash))
	}
	lines = append(lines, components.ViewportCard(components.ViewportCardOptions{
		Width:   m.width,
		Content: m.vp.View(),
		Status:  "PgUp/PgDn to scroll logs",
	}))
	if m.confirmKill {
		lines = append(lines, components.Modal(components.ModalConfig{
			Width: m.width,
			Title: "Stop the current split?",
			Body: []string{
				"Codex will be terminated.",
				"Press ESC again within 2s to stop; press Q twice to quit Helm.",
			},
		}))
	}
	help := []components.HelpEntry{
		{Key: "PgUp/PgDn", Label: "scroll logs"},
		{Key: "c", Label: "copy resume"},
		{Key: "esc×2", Label: "stop split"},
		{Key: "q×2", Label: "quit helm"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       m.width,
		Title:       components.TitleConfig{Title: "Splitting spec"},
		Body:        strings.Join(lines, "\n\n"),
		HelpEntries: help,
	})
}

func doneView(m *model) string {
	if m.err != nil {
		lines := []string{
			components.Flash(components.FlashDanger, fmt.Sprintf("Split failed: %v", m.err)),
		}
		if chip := components.ResumeChip(m.resumeCmd); chip != "" {
			lines = append(lines, chip)
		}
		if m.flash != "" {
			lines = append(lines, components.Flash(components.FlashInfo, m.flash))
		}
		if len(m.logs) > 0 {
			lines = append(lines, components.ViewportCard(components.ViewportCardOptions{
				Width:   m.width,
				Content: renderLogTail(m.logs, 15),
				Status:  "Recent logs",
			}))
		}
		help := []components.HelpEntry{
			{Key: "enter/q/esc", Label: "exit"},
			{Key: "n", Label: "split another"},
			{Key: "r", Label: "jump to Run"},
		}
		return components.PageShell(components.PageShellOptions{
			Width:       m.width,
			Title:       components.TitleConfig{Title: "Split failed"},
			Body:        strings.Join(lines, "\n\n"),
			HelpEntries: help,
		})
	}
	if m.result == nil {
		help := []components.HelpEntry{
			{Key: "enter/q/esc", Label: "exit"},
		}
		return components.PageShell(components.PageShellOptions{
			Width:       m.width,
			Title:       components.TitleConfig{Title: "No specs were created."},
			Body:        "Press Enter/q/Esc to exit.",
			HelpEntries: help,
		})
	}

	lines := []string{
		renderSummaryTable(m.result.Specs),
	}
	if m.resumeCmd != "" {
		lines = append(lines, components.ResumeChip(m.resumeCmd))
	}
	if m.flash != "" {
		lines = append(lines, components.Flash(components.FlashInfo, m.flash))
	}
	if len(m.result.Warnings) > 0 {
		lines = append(lines, components.BulletList(m.result.Warnings))
	}
	if len(m.logs) > 0 {
		lines = append(lines, components.ViewportCard(components.ViewportCardOptions{
			Width:   m.width,
			Content: renderLogTail(m.logs, 15),
			Status:  "Recent logs",
		}))
	}
	help := []components.HelpEntry{
		{Key: "enter/q/esc", Label: "exit"},
		{Key: "r", Label: "jump to Run"},
		{Key: "n", Label: "split another"},
	}
	return components.PageShell(components.PageShellOptions{
		Width:       m.width,
		Title:       components.TitleConfig{Title: "Spec split complete!"},
		Body:        strings.Join(lines, "\n\n"),
		HelpEntries: help,
	})
}

func renderSummaryTable(specs []splitting.GeneratedSpec) string {
	if len(specs) == 0 {
		return "(no specs created)"
	}
	rows := make([][]string, 0, len(specs))
	for _, spec := range specs {
		rows = append(rows, []string{spec.ID, spec.Name, strings.Join(spec.DependsOn, ", ")})
	}
	return components.SummaryTable(components.SummaryTableData{
		Headers: []string{"Spec ID", "Name", "Depends on"},
		Rows:    rows,
	})
}

func renderLogTail(lines []string, max int) string {
	if len(lines) == 0 {
		return "(no logs)"
	}
	if len(lines) > max {
		lines = lines[len(lines)-max:]
	}
	return strings.Join(lines, "\n")
}

func previewDraft(draft string) (string, string) {
	if strings.TrimSpace(draft) == "" {
		return "(draft is empty — press e to open your editor)", "Preview"
	}
	lines := strings.Split(draft, "\n")
	total := len(lines)
	limit := min(total, 10)
	visible := min(limit, 5)
	content := lines[:visible]
	if limit > visible {
		content = append(content, "…")
	}
	status := fmt.Sprintf("Preview: first %d lines (showing %d of %d)", limit, visible, total)
	return strings.Join(content, "\n"), status
}
