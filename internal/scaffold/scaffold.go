package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/polarzero/helm/internal/config"
	"github.com/polarzero/helm/internal/metadata"
)

// Answers captures the interactive responses collected by the TUI.
type Answers struct {
	Mode                config.Mode
	AcceptanceCommands  []string
	SpecsRoot           string
	GenerateSampleGraph bool
}

// Result summarizes how the workspace was scaffolded.
type Result struct {
	SpecsRoot string
	Created   []string
	Skipped   []string
}

// Run provisions the specs workspace based on the supplied answers.
func Run(root string, answers Answers) (*Result, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}

	answers.SpecsRoot = strings.TrimSpace(answers.SpecsRoot)
	if answers.SpecsRoot == "" {
		answers.SpecsRoot = config.DefaultSpecsRoot()
	}
	answers.AcceptanceCommands = normalizeCommands(answers.AcceptanceCommands)
	if len(answers.AcceptanceCommands) == 0 {
		answers.AcceptanceCommands = defaultAcceptanceCommands()
	}
	if answers.Mode == "" {
		answers.Mode = config.ModeStrict
	}

	specsRootAbs := config.ResolveSpecsRoot(root, answers.SpecsRoot)
	if err := os.MkdirAll(specsRootAbs, 0o755); err != nil {
		return nil, fmt.Errorf("ensure specs root %s: %w", specsRootAbs, err)
	}

	result := &Result{SpecsRoot: specsRootAbs}
	record := func(path string, created bool) {
		rel, relErr := filepath.Rel(absRoot, path)
		if relErr == nil {
			path = rel
		}
		if created {
			result.Created = append(result.Created, path)
		} else {
			result.Skipped = append(result.Skipped, path)
		}
	}

	if created, err := writeFileIfMissing(filepath.Join(specsRootAbs, "README.md"), []byte(readmeTemplate()), 0o644); err != nil {
		return nil, err
	} else {
		record(filepath.Join(specsRootAbs, "README.md"), created)
	}

	if created, err := writeFileIfMissing(filepath.Join(specsRootAbs, "spec-splitting-guide.md"), []byte(specSplittingGuideTemplate()), 0o644); err != nil {
		return nil, err
	} else {
		record(filepath.Join(specsRootAbs, "spec-splitting-guide.md"), created)
	}

	implTemplate := buildImplementPrompt(answers.Mode)
	implPath := filepath.Join(specsRootAbs, "implement.prompt-template.md")
	if created, err := writeFileIfMissing(implPath, []byte(implTemplate), 0o644); err != nil {
		return nil, err
	} else {
		record(implPath, created)
	}

	reviewTemplate := buildReviewPrompt()
	reviewPath := filepath.Join(specsRootAbs, "review.prompt-template.md")
	if created, err := writeFileIfMissing(reviewPath, []byte(reviewTemplate), 0o644); err != nil {
		return nil, err
	} else {
		record(reviewPath, created)
	}

	scriptPath := filepath.Join(specsRootAbs, "implement-spec.mjs")
	if created, err := writeFileIfMissing(scriptPath, []byte(implementRunnerScript()), 0o755); err != nil {
		return nil, err
	} else {
		record(scriptPath, created)
	}

	settings := &config.Settings{
		SpecsRoot:          answers.SpecsRoot,
		Mode:               answers.Mode,
		DefaultMaxAttempts: 2,
		CodexScaffold:      config.CodexChoice{Model: "gpt-5.1", Reasoning: "medium"},
		CodexRunImpl:       config.CodexChoice{Model: "gpt-5.1-codex", Reasoning: "medium"},
		CodexRunVer:        config.CodexChoice{Model: "gpt-5.1-codex", Reasoning: "medium"},
		CodexSplit:         config.CodexChoice{Model: "gpt-5.1", Reasoning: "medium"},
		AcceptanceCommands: answers.AcceptanceCommands,
	}
	if err := config.SaveSettings(settings); err != nil {
		return nil, fmt.Errorf("save settings: %w", err)
	}

	specDir := filepath.Join(specsRootAbs, "spec-00-example")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure example spec dir %s: %w", specDir, err)
	}

	specPath := filepath.Join(specDir, "SPEC.md")
	if created, err := writeFileIfMissing(specPath, []byte(exampleSpecBody()), 0o644); err != nil {
		return nil, err
	} else {
		record(specPath, created)
	}

	checklistPath := filepath.Join(specDir, "acceptance-checklist.md")
	if created, err := writeFileIfMissing(checklistPath, []byte(exampleAcceptanceChecklist(answers.AcceptanceCommands)), 0o644); err != nil {
		return nil, err
	} else {
		record(checklistPath, created)
	}

	metadataPath := filepath.Join(specDir, "metadata.json")
	if _, err := os.Stat(metadataPath); err == nil {
		record(metadataPath, false)
	} else if errors.Is(err, os.ErrNotExist) {
		meta := metadata.SpecMetadata{
			ID:                 "spec-00-example",
			Name:               "Example spec workspace walkthrough",
			Status:             metadata.StatusTodo,
			DependsOn:          []string{},
			AcceptanceCommands: answers.AcceptanceCommands,
		}
		if err := metadata.SaveMetadata(metadataPath, &meta); err != nil {
			return nil, fmt.Errorf("write example metadata: %w", err)
		}
		record(metadataPath, true)
	} else {
		return nil, fmt.Errorf("stat metadata %s: %w", metadataPath, err)
	}

	reportPath := filepath.Join(specDir, "implementation-report.md")
	if created, err := writeFileIfMissing(reportPath, []byte(exampleImplementationReport()), 0o644); err != nil {
		return nil, err
	} else {
		record(reportPath, created)
	}

	if answers.GenerateSampleGraph {
		graphPath := filepath.Join(specsRootAbs, "sample-dependency-graph.json")
		if created, err := writeFileIfMissing(graphPath, []byte(sampleDependencyGraph()), 0o644); err != nil {
			return nil, err
		} else {
			record(graphPath, created)
		}
	}

	sort.Strings(result.Created)
	sort.Strings(result.Skipped)
	return result, nil
}

func normalizeCommands(commands []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, cmd := range commands {
		trimmed := strings.TrimSpace(cmd)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func defaultAcceptanceCommands() []string {
	return []string{"go test ./...", "go vet ./..."}
}

// DefaultAcceptanceCommands exposes the scaffold defaults for consumers like the TUI.
func DefaultAcceptanceCommands() []string {
	return cloneStrings(defaultAcceptanceCommands())
}

func writeFileIfMissing(path string, data []byte, perm os.FileMode) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("ensure dir for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	return true, nil
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}
