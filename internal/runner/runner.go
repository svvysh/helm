package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/polarzero/helm/internal/config"
	"github.com/polarzero/helm/internal/metadata"
)

// Runner executes the worker/verifier loop for a single spec directory.
type Runner struct {
	Root                      string
	SpecsRoot                 string
	Mode                      config.Mode
	MaxAttempts               int
	WorkerChoice              config.CodexChoice
	VerifierChoice            config.CodexChoice
	DefaultAcceptanceCommands []string
	Stdout                    io.Writer
	Stderr                    io.Writer
	Codex                     CodexExecutor
	Clock                     func() time.Time
}

// CodexExecutor shells out to the Codex CLI.
type CodexExecutor interface {
	Exec(ctx context.Context, args []string, stdin string, stdout, stderr io.Writer) (string, error)
}

// ExecCodex implements CodexExecutor using os/exec.
type ExecCodex struct {
	Binary string
}

func (e ExecCodex) Exec(ctx context.Context, args []string, stdin string, stdout, stderr io.Writer) (string, error) {
	bin := e.Binary
	if bin == "" {
		bin = "codex"
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	cmd := execCommandContext(ctx, bin, args...)
	cmd.Stdin = strings.NewReader(stdin)

	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(stdout, &buf)
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("codex %s: %w", strings.Join(args, " "), err)
	}
	return buf.String(), nil
}

var execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// Run executes the worker/verifier loop for the provided spec argument.
func (r *Runner) Run(ctx context.Context, specArg string) error {
	if r == nil {
		return errors.New("runner is nil")
	}
	if specArg == "" {
		return errors.New("spec argument is required")
	}

	stdout := r.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := r.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	codex := r.Codex
	if codex == nil {
		codex = ExecCodex{}
	}

	clock := r.Clock
	if clock == nil {
		clock = time.Now
	}

	mode := r.Mode
	if mode == "" {
		mode = config.ModeStrict
	}

	workerChoice := r.WorkerChoice
	if workerChoice.Model == "" {
		workerChoice = config.CodexChoice{Model: "gpt-5.1-codex", Reasoning: "medium"}
	}

	verifierChoice := r.VerifierChoice
	if verifierChoice.Model == "" {
		verifierChoice = config.CodexChoice{Model: "gpt-5.1-codex", Reasoning: "medium"}
	}

	maxAttempts := r.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 2
	}

	root := r.Root
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}

	if r.SpecsRoot == "" {
		return errors.New("specs root is required")
	}
	absSpecsRoot, err := filepath.Abs(r.SpecsRoot)
	if err != nil {
		return fmt.Errorf("resolve specs root: %w", err)
	}

	specDir, err := resolveSpecDir(specArg, absRoot, absSpecsRoot)
	if err != nil {
		return err
	}

	spec, err := loadSpecResources(specDir, absSpecsRoot, r.DefaultAcceptanceCommands)
	if err != nil {
		return err
	}

	remaining := []string{}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		currentRemainingJSON, err := json.MarshalIndent(remaining, "", "  ")
		if err != nil {
			return fmt.Errorf("encode remaining tasks: %w", err)
		}

		workerPrompt := fillTemplate(spec.ImplementTemplate, map[string]string{
			"{{SPEC_ID}}":                  spec.ID,
			"{{SPEC_NAME}}":                spec.Name,
			"{{SPEC_BODY}}":                spec.SpecBody,
			"{{ACCEPTANCE_COMMANDS}}":      spec.AcceptanceCommandsText,
			"{{PREVIOUS_REMAINING_TASKS}}": string(currentRemainingJSON),
			"{{MODE}}":                     string(mode),
		})

		workerOutput, err := codex.Exec(ctx, workerArgs(workerChoice), workerPrompt, stdout, stderr)
		if err != nil {
			return fmt.Errorf("worker execution failed: %w", err)
		}

		reviewPrompt := fillTemplate(spec.ReviewTemplate, map[string]string{
			"{{SPEC_ID}}":               spec.ID,
			"{{SPEC_NAME}}":             spec.Name,
			"{{SPEC_BODY}}":             spec.SpecBody,
			"{{ACCEPTANCE_CHECKLIST}}":  spec.Checklist,
			"{{ACCEPTANCE_COMMANDS}}":   spec.AcceptanceCommandsText,
			"{{IMPLEMENTATION_REPORT}}": workerOutput,
			"{{MODE}}":                  string(mode),
		})

		verifierOutput, err := codex.Exec(ctx, verifierArgs(verifierChoice), reviewPrompt, stdout, stderr)
		if err != nil {
			return fmt.Errorf("verifier execution failed: %w", err)
		}

		status, newRemaining, err := parseVerifierOutput(verifierOutput)
		if err != nil {
			return err
		}

		now := clock().UTC()
		if err := updateMetadata(spec, status, newRemaining, workerOutput, attempt, now); err != nil {
			return err
		}

		if err := writeReport(spec, mode, maxAttempts, attempt, status, newRemaining, workerOutput); err != nil {
			return err
		}

		if status == "ok" {
			return nil
		}

		remaining = newRemaining
	}

	return fmt.Errorf("exhausted %d attempts without STATUS: ok", maxAttempts)
}

func workerArgs(choice config.CodexChoice) []string {
	args := []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "--model", choice.Model}
	if choice.Reasoning != "" {
		args = append(args, "--reasoning", choice.Reasoning)
	}
	args = append(args, "--stdin")
	return args
}

func verifierArgs(choice config.CodexChoice) []string {
	args := []string{"exec", "--sandbox", "read-only", "--model", choice.Model}
	if choice.Reasoning != "" {
		args = append(args, "--reasoning", choice.Reasoning)
	}
	args = append(args, "--stdin")
	return args
}

func fillTemplate(tpl string, replacements map[string]string) string {
	pairs := make([]string, 0, len(replacements)*2)
	for k, v := range replacements {
		pairs = append(pairs, k, v)
	}
	return strings.NewReplacer(pairs...).Replace(tpl)
}

func resolveSpecDir(arg, root, specsRoot string) (string, error) {
	if filepath.IsAbs(arg) {
		if _, err := os.Stat(arg); err != nil {
			return "", fmt.Errorf("could not find spec directory %s: %w", arg, err)
		}
		return arg, nil
	}

	candidate := filepath.Join(root, arg)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat spec %s: %w", candidate, err)
	}

	alt := filepath.Join(specsRoot, arg)
	if _, err := os.Stat(alt); err == nil {
		return alt, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("could not find spec directory at %s or %s", candidate, alt)
	} else {
		return "", fmt.Errorf("stat spec %s: %w", alt, err)
	}
}

func loadSpecResources(specDir, specsRoot string, defaultCommands []string) (*specResources, error) {
	metaPath := filepath.Join(specDir, "metadata.json")
	specPath := filepath.Join(specDir, "SPEC.md")
	reportPath := filepath.Join(specDir, "implementation-report.md")

	meta, err := metadata.LoadMetadata(metaPath)
	if err != nil {
		return nil, err
	}

	specBody, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read SPEC.md: %w", err)
	}

	checklistPath := filepath.Join(specDir, "acceptance-checklist.md")
	checklistData, _ := os.ReadFile(checklistPath)

	implementTplPath := filepath.Join(specsRoot, "implement.prompt-template.md")
	reviewTplPath := filepath.Join(specsRoot, "review.prompt-template.md")

	implementTpl, err := os.ReadFile(implementTplPath)
	if err != nil {
		return nil, fmt.Errorf("read implement prompt template: %w", err)
	}
	reviewTpl, err := os.ReadFile(reviewTplPath)
	if err != nil {
		return nil, fmt.Errorf("read review prompt template: %w", err)
	}

	commands := meta.AcceptanceCommands
	if len(commands) == 0 && len(defaultCommands) > 0 {
		commands = defaultCommands
	}

	acceptanceText := formatAcceptanceCommands(commands)

	specID := meta.ID
	if specID == "" {
		specID = filepath.Base(specDir)
	}

	specName := meta.Name
	if specName == "" {
		specName = extractSpecTitle(string(specBody))
		if specName == "" {
			specName = "(unnamed spec)"
		}
	}

	return &specResources{
		Dir:                    specDir,
		Metadata:               meta,
		MetadataPath:           metaPath,
		SpecBody:               string(specBody),
		Checklist:              string(checklistData),
		ImplementTemplate:      string(implementTpl),
		ReviewTemplate:         string(reviewTpl),
		ReportPath:             reportPath,
		AcceptanceCommands:     commands,
		AcceptanceCommandsText: acceptanceText,
		ID:                     specID,
		Name:                   specName,
	}, nil
}

func formatAcceptanceCommands(commands []string) string {
	if len(commands) == 0 {
		return "- (none specified)"
	}
	lines := make([]string, len(commands))
	for i, cmd := range commands {
		lines[i] = "- " + cmd
	}
	return strings.Join(lines, "\n")
}

func extractSpecTitle(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}

func parseVerifierOutput(output string) (string, []string, error) {
	lines := strings.Split(output, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	if len(filtered) < 2 {
		return "", nil, errors.New("verifier output missing required lines")
	}

	statusLine := filtered[0]
	var status string
	switch statusLine {
	case "STATUS: ok":
		status = "ok"
	case "STATUS: missing":
		status = "missing"
	default:
		return "", nil, fmt.Errorf("unexpected verifier status line: %s", statusLine)
	}

	var payload struct {
		RemainingTasks []string `json:"remainingTasks"`
	}
	if err := json.Unmarshal([]byte(filtered[1]), &payload); err != nil {
		return "", nil, fmt.Errorf("parse verifier JSON: %w", err)
	}
	if payload.RemainingTasks == nil {
		payload.RemainingTasks = []string{}
	}

	return status, payload.RemainingTasks, nil
}

func updateMetadata(spec *specResources, status string, remaining []string, workerOutput string, attempt int, now time.Time) error {
	if spec == nil || spec.Metadata == nil {
		return errors.New("spec metadata is nil")
	}
	meta := spec.Metadata
	meta.LastRun = &now

	switch status {
	case "ok":
		meta.Status = metadata.StatusDone
		summary := summarizeWorkerOutput(workerOutput)
		note := fmt.Sprintf("[%s] attempt %d ok — %s", now.Format(time.RFC3339), attempt, summary)
		appendNote(meta, note)
	case "missing":
		meta.Status = metadata.StatusInProgress
		summary := "none"
		if len(remaining) > 0 {
			summary = strings.Join(remaining, "; ")
		}
		note := fmt.Sprintf("[%s] attempt %d remaining tasks: %s", now.Format(time.RFC3339), attempt, summary)
		appendNote(meta, note)
	default:
		return fmt.Errorf("unknown status %s", status)
	}

	if err := metadata.SaveMetadata(spec.MetadataPath, meta); err != nil {
		return fmt.Errorf("save metadata: %w", err)
	}
	return nil
}

func appendNote(meta *metadata.SpecMetadata, note string) {
	note = strings.TrimSpace(note)
	if note == "" {
		return
	}
	if strings.TrimSpace(meta.Notes) == "" {
		meta.Notes = note
	} else {
		meta.Notes = strings.TrimSpace(meta.Notes) + "\n" + note
	}
}

func summarizeWorkerOutput(workerOutput string) string {
	lines := strings.Split(workerOutput, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" {
			return trimmed
		}
	}
	return "worker output empty"
}

func writeReport(spec *specResources, mode config.Mode, maxAttempts, attempt int, status string, remaining []string, workerOutput string) error {
	if spec == nil {
		return errors.New("spec resources nil")
	}
	data, err := json.MarshalIndent(map[string]any{"remainingTasks": remaining}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode remaining tasks: %w", err)
	}

	builder := &strings.Builder{}
	fmt.Fprintf(builder, "# Implementation Report for %s — %s\n\n", spec.ID, spec.Name)
	fmt.Fprintf(builder, "- Mode: %s\n", mode)
	fmt.Fprintf(builder, "- Max attempts: %d\n", maxAttempts)
	fmt.Fprintf(builder, "- Attempts performed: %d\n", attempt)
	fmt.Fprintf(builder, "- Final verifier status: %s\n\n", status)
	builder.WriteString("## Remaining tasks\n\n")
	builder.Write(data)
	builder.WriteString("\n\n## Final worker output\n\n")
	builder.WriteString(workerOutput)
	builder.WriteString("\n")

	if err := os.WriteFile(spec.ReportPath, []byte(builder.String()), 0o644); err != nil {
		return fmt.Errorf("write implementation report: %w", err)
	}
	return nil
}

type specResources struct {
	Dir                    string
	Metadata               *metadata.SpecMetadata
	MetadataPath           string
	SpecBody               string
	Checklist              string
	ImplementTemplate      string
	ReviewTemplate         string
	ReportPath             string
	AcceptanceCommands     []string
	AcceptanceCommandsText string
	ID                     string
	Name                   string
}
