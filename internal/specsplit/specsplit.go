package specsplit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/polarzero/helm/internal/config"
	"github.com/polarzero/helm/internal/metadata"
	"github.com/polarzero/helm/internal/runner"
	"github.com/polarzero/helm/internal/specs"
)

// Plan represents the Codex response describing derived specs.
type Plan struct {
	Specs []PlanSpec `json:"specs"`
}

// PlanSpec captures a single spec entry returned by Codex.
type PlanSpec struct {
	Index              int      `json:"index"`
	IDSuffix           string   `json:"idSuffix"`
	Name               string   `json:"name"`
	Summary            string   `json:"summary"`
	DependsOn          []string `json:"dependsOn"`
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
}

// Options configures how Split operates.
type Options struct {
	SpecsRoot          string
	RawSpec            string
	GuidePath          string
	AcceptanceCommands []string
	CodexChoice        config.CodexChoice
	Codex              runner.CodexExecutor
	Plan               *Plan
	PlanPath           string
	Stdout             io.Writer
	Stderr             io.Writer
}

// Result summarizes what was created by Split.
type Result struct {
	Specs    []GeneratedSpec
	Warnings []string
	Plan     *Plan
}

// GeneratedSpec describes an emitted spec folder.
type GeneratedSpec struct {
	ID        string
	Name      string
	Path      string
	DependsOn []string
}

// Split orchestrates plan acquisition and folder generation.
func Split(ctx context.Context, opts Options) (*Result, error) {
	specsRoot := strings.TrimSpace(opts.SpecsRoot)
	if specsRoot == "" {
		return nil, errors.New("specs root is required")
	}
	absRoot, err := filepath.Abs(specsRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve specs root: %w", err)
	}
	if info, err := os.Stat(absRoot); err != nil {
		return nil, fmt.Errorf("stat specs root %s: %w", absRoot, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("specs root %s is not a directory", absRoot)
	}

	rawSpec := strings.TrimSpace(opts.RawSpec)
	if rawSpec == "" {
		return nil, errors.New("spec input cannot be empty")
	}

	plan, err := resolvePlan(ctx, absRoot, rawSpec, opts)
	if err != nil {
		return nil, err
	}
	if len(plan.Specs) == 0 {
		return nil, errors.New("split plan contained no specs")
	}

	acceptance := cloneStrings(opts.AcceptanceCommands)
	res, err := applyPlan(absRoot, plan, acceptance)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func resolvePlan(ctx context.Context, specsRoot, rawSpec string, opts Options) (*Plan, error) {
	if opts.Plan != nil {
		normalized := normalizePlan(opts.Plan)
		if err := normalized.Validate(); err != nil {
			return nil, err
		}
		return normalized, nil
	}

	if opts.PlanPath != "" {
		data, err := os.ReadFile(opts.PlanPath)
		if err != nil {
			return nil, fmt.Errorf("read plan file %s: %w", opts.PlanPath, err)
		}
		plan, err := parsePlan(data)
		if err != nil {
			return nil, err
		}
		return plan, nil
	}

	guidePath := opts.GuidePath
	if guidePath == "" {
		guidePath = filepath.Join(specsRoot, "spec-splitting-guide.md")
	}
	guideData, err := os.ReadFile(guidePath)
	if err != nil {
		return nil, fmt.Errorf("read splitting guide %s: %w", guidePath, err)
	}

	prompt := buildPlanPrompt(string(guideData), rawSpec, opts.AcceptanceCommands)

	codex := opts.Codex
	if codex == nil {
		codex = runner.ExecCodex{}
	}
	choice := opts.CodexChoice
	if choice.Model == "" {
		choice = config.CodexChoice{Model: "gpt-5.1", Reasoning: "medium"}
	}

	args := []string{"exec", "--sandbox", "read-only", "--model", choice.Model}
	if choice.Reasoning != "" {
		args = append(args, "--reasoning", choice.Reasoning)
	}
	args = append(args, "--stdin")

	stdout := opts.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	output, err := codex.Exec(ctx, args, prompt, stdout, stderr)
	if err != nil {
		return nil, fmt.Errorf("codex split plan failed: %w", err)
	}

	plan, err := parsePlan([]byte(output))
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func parsePlan(data []byte) (*Plan, error) {
	trimmed := bytes.TrimSpace(data)
	plan := &Plan{}
	if err := json.Unmarshal(trimmed, plan); err != nil {
		start := bytes.IndexByte(trimmed, '{')
		end := bytes.LastIndexByte(trimmed, '}')
		if start >= 0 && end > start {
			var alt Plan
			if err2 := json.Unmarshal(trimmed[start:end+1], &alt); err2 == nil {
				normalized := normalizePlan(&alt)
				if err := normalized.Validate(); err != nil {
					return nil, err
				}
				return normalized, nil
			}
		}
		return nil, fmt.Errorf("parse split plan: %w", err)
	}
	normalized := normalizePlan(plan)
	if err := normalized.Validate(); err != nil {
		return nil, err
	}
	return normalized, nil
}

// Validate ensures the plan is well-formed.
func (p *Plan) Validate() error {
	if p == nil {
		return errors.New("plan is nil")
	}
	if len(p.Specs) == 0 {
		return errors.New("plan.specs must not be empty")
	}
	for i, spec := range p.Specs {
		if strings.TrimSpace(spec.Name) == "" {
			return fmt.Errorf("plan.specs[%d].name is required", i)
		}
		if spec.Index < 0 {
			return fmt.Errorf("plan.specs[%d].index must be >=0", i)
		}
	}
	return nil
}

func normalizePlan(plan *Plan) *Plan {
	if plan == nil {
		return nil
	}
	out := &Plan{Specs: make([]PlanSpec, len(plan.Specs))}
	for i, spec := range plan.Specs {
		normalized := spec
		normalized.Name = strings.TrimSpace(normalized.Name)
		normalized.Summary = strings.TrimSpace(normalized.Summary)
		normalized.IDSuffix = strings.TrimSpace(normalized.IDSuffix)
		if normalized.IDSuffix == "" {
			normalized.IDSuffix = normalized.Name
		}
		normalized.DependsOn = cloneStrings(normalized.DependsOn)
		normalized.AcceptanceCriteria = cloneStrings(normalized.AcceptanceCriteria)
		out.Specs[i] = normalized
	}
	return out
}

func applyPlan(specsRoot string, plan *Plan, acceptance []string) (*Result, error) {
	if plan == nil {
		return nil, errors.New("plan is nil")
	}

	existingNames := map[string]string{}
	if folders, err := specs.DiscoverSpecs(specsRoot); err == nil {
		for _, folder := range folders {
			if folder.Metadata != nil {
				existingNames[folder.Metadata.ID] = folder.Metadata.Name
			}
		}
	}

	usedIDs := map[string]struct{}{}
	drafts := make([]*specDraft, 0, len(plan.Specs))
	suffixMap := map[string]*specDraft{}
	indexMap := map[int]*specDraft{}
	idMap := map[string]*specDraft{}
	var warnings []string

	for _, spec := range plan.Specs {
		suffix := normalizeSuffix(spec.IDSuffix)
		baseID := fmt.Sprintf("spec-%02d-%s", spec.Index, suffix)
		finalID, warn, err := ensureUniqueID(specsRoot, baseID, usedIDs)
		if err != nil {
			return nil, err
		}
		if warn != "" {
			warnings = append(warnings, warn)
		}
		draft := &specDraft{
			Plan:    spec,
			ID:      finalID,
			Dir:     filepath.Join(specsRoot, finalID),
			Summary: spec.Summary,
		}
		drafts = append(drafts, draft)
		suffixMap[suffix] = draft
		indexMap[spec.Index] = draft
		idMap[strings.ToLower(finalID)] = draft
	}

	for _, draft := range drafts {
		draft.Depends = resolveDependencies(draft.Plan.DependsOn, indexMap, suffixMap, idMap)
	}

	nameMap := make(map[string]string, len(existingNames)+len(drafts))
	for k, v := range existingNames {
		nameMap[k] = v
	}

	var results []GeneratedSpec
	for _, draft := range drafts {
		if err := os.MkdirAll(draft.Dir, 0o755); err != nil {
			return nil, fmt.Errorf("ensure spec dir %s: %w", draft.Dir, err)
		}

		if err := writeSpecFiles(draft, acceptance, nameMap); err != nil {
			return nil, err
		}

		nameMap[draft.ID] = draft.Plan.Name
		results = append(results, GeneratedSpec{
			ID:        draft.ID,
			Name:      draft.Plan.Name,
			Path:      draft.Dir,
			DependsOn: cloneStrings(draft.Depends),
		})
	}

	sort.Strings(warnings)

	return &Result{Specs: results, Warnings: warnings, Plan: plan}, nil
}

type specDraft struct {
	Plan    PlanSpec
	ID      string
	Dir     string
	Summary string
	Depends []string
}

func writeSpecFiles(draft *specDraft, acceptance []string, nameByID map[string]string) error {
	if draft == nil {
		return errors.New("draft is nil")
	}

	specPath := filepath.Join(draft.Dir, "SPEC.md")
	if err := os.WriteFile(specPath, []byte(renderSpecMarkdown(draft, nameByID)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", specPath, err)
	}

	checklistPath := filepath.Join(draft.Dir, "acceptance-checklist.md")
	if err := os.WriteFile(checklistPath, []byte(renderChecklist(draft, acceptance)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", checklistPath, err)
	}

	depends := cloneStrings(draft.Depends)
	if depends == nil {
		depends = []string{}
	}
	cmds := cloneStrings(acceptance)
	if cmds == nil {
		cmds = []string{}
	}

	meta := &metadata.SpecMetadata{
		ID:                 draft.ID,
		Name:               draft.Plan.Name,
		Status:             metadata.StatusTodo,
		DependsOn:          depends,
		AcceptanceCommands: cmds,
	}

	if err := metadata.SaveMetadata(filepath.Join(draft.Dir, "metadata.json"), meta); err != nil {
		return fmt.Errorf("write metadata for %s: %w", draft.ID, err)
	}

	reportPath := filepath.Join(draft.Dir, "implementation-report.md")
	report := "Implementation report will be generated by `implement-spec.mjs` after this spec is run.\n"
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", reportPath, err)
	}

	return nil
}

func renderSpecMarkdown(draft *specDraft, nameByID map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", draft.Plan.Name)
	b.WriteString("## Summary\n\n")
	summary := strings.TrimSpace(draft.Summary)
	if summary == "" {
		summary = "(summary TBD)"
	}
	b.WriteString(summary)
	b.WriteString("\n\n")

	b.WriteString("## Acceptance Criteria\n\n")
	criteria := filterNonEmpty(draft.Plan.AcceptanceCriteria)
	if len(criteria) == 0 {
		b.WriteString("- (acceptance criteria TBD)\n\n")
	} else {
		for _, item := range criteria {
			fmt.Fprintf(&b, "- %s\n", item)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Depends on\n\n")
	if len(draft.Depends) == 0 {
		b.WriteString("- _None_\n")
	} else {
		for _, dep := range draft.Depends {
			label := fmt.Sprintf("`%s`", dep)
			if name := nameByID[dep]; name != "" {
				label = fmt.Sprintf("`%s` — %s", dep, name)
			}
			fmt.Fprintf(&b, "- %s\n", label)
		}
	}
	b.WriteString("\n")
	return b.String()
}

func renderChecklist(draft *specDraft, commands []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Acceptance Checklist — %s\n\n", draft.ID)
	b.WriteString("## Automated commands\n\n")
	cmds := filterNonEmpty(commands)
	if len(cmds) == 0 {
		b.WriteString("- [ ] (none provided)\n\n")
	} else {
		for _, cmd := range cmds {
			fmt.Fprintf(&b, "- [ ] `%s`\n", cmd)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Spec criteria\n\n")
	crit := filterNonEmpty(draft.Plan.AcceptanceCriteria)
	if len(crit) == 0 {
		b.WriteString("- [ ] (add acceptance criteria)\n")
	} else {
		for _, c := range crit {
			fmt.Fprintf(&b, "- [ ] %s\n", c)
		}
	}
	b.WriteString("\n")
	return b.String()
}

func ensureUniqueID(specsRoot, baseID string, used map[string]struct{}) (string, string, error) {
	candidate := baseID
	suffix := 2
	for {
		if _, ok := used[candidate]; ok {
			candidate = fmt.Sprintf("%s-%d", baseID, suffix)
			suffix++
			continue
		}
		path := filepath.Join(specsRoot, candidate)
		if _, err := os.Stat(path); err == nil {
			candidate = fmt.Sprintf("%s-%d", baseID, suffix)
			suffix++
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("stat %s: %w", path, err)
		}
		used[candidate] = struct{}{}
		warn := ""
		if candidate != baseID {
			warn = fmt.Sprintf("spec folder %s existed; created %s instead", baseID, candidate)
		}
		return candidate, warn, nil
	}
}

func resolveDependencies(raw []string, indexMap map[int]*specDraft, suffixMap, idMap map[string]*specDraft) []string {
	seen := map[string]struct{}{}
	var deps []string
	for _, item := range raw {
		dep := strings.TrimSpace(item)
		if dep == "" {
			continue
		}
		resolved := resolveDependency(dep, indexMap, suffixMap, idMap)
		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		deps = append(deps, resolved)
	}
	return deps
}

func resolveDependency(dep string, indexMap map[int]*specDraft, suffixMap, idMap map[string]*specDraft) string {
	if draft := idMap[strings.ToLower(dep)]; draft != nil {
		return draft.ID
	}
	if idx, err := strconv.Atoi(dep); err == nil {
		if draft := indexMap[idx]; draft != nil {
			return draft.ID
		}
	}
	if draft := suffixMap[normalizeSuffix(dep)]; draft != nil {
		return draft.ID
	}
	return dep
}

func normalizeSuffix(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return "spec"
	}
	var b strings.Builder
	lastHyphen := false
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "spec"
	}
	return result
}

func buildPlanPrompt(guide, rawSpec string, acceptance []string) string {
	var b strings.Builder
	b.WriteString("# Spec Splitting Request\n\n")
	b.WriteString("You are assisting with breaking down a large product spec into smaller, incremental specs for the Helm CLI. Use the provided guide and acceptance commands to structure the split. Respond with **JSON only** matching the described schema.\n\n")

	b.WriteString("## Splitting Guide\n\n")
	b.WriteString(guide)
	if !strings.HasSuffix(guide, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("## Acceptance Commands\n\n")
	cmds := filterNonEmpty(acceptance)
	if len(cmds) == 0 {
		b.WriteString("- (none provided)\n\n")
	} else {
		for _, cmd := range cmds {
			fmt.Fprintf(&b, "- %s\n", cmd)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Raw Spec\n\n")
	b.WriteString("```markdown\n")
	b.WriteString(rawSpec)
	if !strings.HasSuffix(rawSpec, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("```\n\n")

	b.WriteString("## JSON Response Contract\n\n")
	b.WriteString("Return JSON with this shape (no extra commentary):\n\n")
	b.WriteString("```json\n")
	b.WriteString(`{
  "specs": [
    {
      "index": 0,
      "idSuffix": "foundation",
      "name": "Go module and CLI skeleton",
      "summary": "Short human-readable summary",
      "dependsOn": ["spec-00-foundation"],
      "acceptanceCriteria": [
        "CLI binary exposes scaffold/run/spec/status subcommands",
        "go test ./... passes"
      ]
    }
  ]
}`)
	b.WriteString("\n```)\n\n")

	b.WriteString("Rules:\n")
	b.WriteString("- `index` must be unique per spec and roughly follow execution order.\n")
	b.WriteString("- `idSuffix` should be a short slug used for the folder name.\n")
	b.WriteString("- `dependsOn` may reference existing spec IDs or other entries via `index` or `idSuffix`.\n")
	b.WriteString("- Provide at least one acceptance criterion per spec.\n")
	b.WriteString("- Keep each spec focused and implementation-ready.\n")
	return b.String()
}

func filterNonEmpty(values []string) []string {
	var out []string
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}
