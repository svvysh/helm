package runner

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/polarzero/helm/internal/config"
	"github.com/polarzero/helm/internal/metadata"
)

func TestRunnerSuccessWritesMetadataAndReport(t *testing.T) {
	root := t.TempDir()
	specsRoot := filepath.Join(root, "docs", "specs")
	specDir := filepath.Join(specsRoot, "spec-99-demo")
	mustMkdirAll(t, specDir)
	writeSpecFiles(t, specsRoot, specDir, []string{"make all"})

	fake := &fakeCodex{
		t: t,
		responses: []fakeResponse{
			{stdout: "worker log\nall good\n"},
			{stdout: "STATUS: ok\n{\"remainingTasks\":[]}\ncommentary"},
		},
	}

	now := time.Date(2025, 11, 22, 8, 0, 0, 0, time.UTC)

	r := &Runner{
		Root:                      root,
		SpecsRoot:                 specsRoot,
		Mode:                      config.ModeStrict,
		MaxAttempts:               2,
		WorkerModel:               "impl",
		VerifierModel:             "ver",
		DefaultAcceptanceCommands: []string{"make all"},
		Stdout:                    io.Discard,
		Stderr:                    io.Discard,
		Codex:                     fake,
		Clock: func() time.Time {
			return now
		},
	}

	if err := r.Run(context.Background(), "spec-99-demo"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	meta, err := metadata.LoadMetadata(filepath.Join(specDir, "metadata.json"))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v", err)
	}
	if meta.Status != metadata.StatusDone {
		t.Fatalf("metadata status = %s, want done", meta.Status)
	}
	if meta.LastRun == nil || !meta.LastRun.Equal(now) {
		t.Fatalf("metadata lastRun mismatch: %+v", meta.LastRun)
	}
	if !strings.Contains(meta.Notes, "all good") {
		t.Fatalf("notes missing worker summary: %s", meta.Notes)
	}

	report := slurp(t, filepath.Join(specDir, "implementation-report.md"))
	if !strings.Contains(report, "Final verifier status: ok") {
		t.Fatalf("report missing status: %s", report)
	}
	if !strings.Contains(report, "\"remainingTasks\": []") {
		t.Fatalf("report missing remaining tasks: %s", report)
	}
	if !strings.Contains(report, "worker log") {
		t.Fatalf("report missing worker output")
	}
}

func TestRunnerMissingPropagatesRemainingTasks(t *testing.T) {
	root := t.TempDir()
	specsRoot := filepath.Join(root, "docs", "specs")
	specDir := filepath.Join(specsRoot, "spec-99-demo")
	mustMkdirAll(t, specDir)
	writeSpecFiles(t, specsRoot, specDir, nil)

	fake := &fakeCodex{
		t: t,
		responses: []fakeResponse{
			{stdout: "worker attempt 1"},
			{stdout: "STATUS: missing\n{\"remainingTasks\":[\"write tests\"]}\ncommentary"},
		},
	}

	r := &Runner{
		Root:        root,
		SpecsRoot:   specsRoot,
		MaxAttempts: 1,
		Mode:        config.ModeStrict,
		Stdout:      io.Discard,
		Stderr:      io.Discard,
		Codex:       fake,
		Clock: func() time.Time {
			return time.Date(2025, 11, 22, 8, 0, 0, 0, time.UTC)
		},
	}

	err := r.Run(context.Background(), "spec-99-demo")
	if err == nil {
		t.Fatalf("Run() expected error")
	}

	meta, err := metadata.LoadMetadata(filepath.Join(specDir, "metadata.json"))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v", err)
	}
	if meta.Status != metadata.StatusInProgress {
		t.Fatalf("metadata status = %s, want in-progress", meta.Status)
	}
	if !strings.Contains(meta.Notes, "write tests") {
		t.Fatalf("notes missing remaining tasks: %s", meta.Notes)
	}

	report := slurp(t, filepath.Join(specDir, "implementation-report.md"))
	if !strings.Contains(report, "Final verifier status: missing") {
		t.Fatalf("report missing status: %s", report)
	}
	if !strings.Contains(report, "write tests") {
		t.Fatalf("report missing remaining tasks JSON: %s", report)
	}
}

func TestRunnerUsesPreviousRemainingTasksInPrompt(t *testing.T) {
	root := t.TempDir()
	specsRoot := filepath.Join(root, "docs", "specs")
	specDir := filepath.Join(specsRoot, "spec-99-demo")
	mustMkdirAll(t, specDir)
	writeSpecFiles(t, specsRoot, specDir, nil)

	fake := &fakeCodex{
		t: t,
		responses: []fakeResponse{
			{stdout: "first worker"},
			{stdout: "STATUS: missing\n{\"remainingTasks\":[\"fix lint\"]}"},
			{stdout: "second worker"},
			{stdout: "STATUS: ok\n{\"remainingTasks\":[]}"},
		},
	}

	r := &Runner{
		Root:                      root,
		SpecsRoot:                 specsRoot,
		MaxAttempts:               2,
		Mode:                      config.ModeStrict,
		DefaultAcceptanceCommands: []string{"go test ./..."},
		Stdout:                    io.Discard,
		Stderr:                    io.Discard,
		Codex:                     fake,
		Clock:                     func() time.Time { return time.Date(2025, 11, 22, 8, 0, 0, 0, time.UTC) },
	}

	if err := r.Run(context.Background(), "spec-99-demo"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(fake.calls) != 4 {
		t.Fatalf("expected 4 codex calls, got %d", len(fake.calls))
	}
	workerPrompt := fake.calls[2].prompt
	if !strings.Contains(workerPrompt, "\"fix lint\"") {
		t.Fatalf("second worker prompt missing remaining tasks: %s", workerPrompt)
	}
	if !strings.Contains(workerPrompt, "- go test ./...") {
		t.Fatalf("worker prompt missing acceptance commands: %s", workerPrompt)
	}
}

type fakeCodex struct {
	t         *testing.T
	responses []fakeResponse
	calls     []codexCall
}

type fakeResponse struct {
	stdout string
	err    error
}

type codexCall struct {
	args   []string
	prompt string
}

func (f *fakeCodex) Exec(_ context.Context, args []string, stdin string, stdout, stderr io.Writer) (string, error) {
	if len(f.responses) == 0 {
		f.t.Fatalf("unexpected codex call: %+v", args)
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	f.calls = append(f.calls, codexCall{args: append([]string(nil), args...), prompt: stdin})
	if stdout != nil {
		if _, err := io.WriteString(stdout, resp.stdout); err != nil {
			f.t.Fatalf("write stdout: %v", err)
		}
	}
	if stderr != nil {
		if _, err := io.WriteString(stderr, ""); err != nil {
			f.t.Fatalf("write stderr: %v", err)
		}
	}
	if resp.err != nil {
		return "", resp.err
	}
	return resp.stdout, nil
}

func writeSpecFiles(t *testing.T, specsRoot, specDir string, acceptance []string) {
	t.Helper()
	mustMkdirAll(t, specsRoot)
	if err := os.WriteFile(filepath.Join(specDir, "SPEC.md"), []byte("# Demo Spec\nbody"), 0o644); err != nil {
		t.Fatalf("write SPEC.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "acceptance-checklist.md"), []byte("- [ ] check"), 0o644); err != nil {
		t.Fatalf("write checklist: %v", err)
	}
	meta := &metadata.SpecMetadata{
		ID:                 "spec-99-demo",
		Name:               "Demo Runner",
		Status:             metadata.StatusTodo,
		AcceptanceCommands: acceptance,
	}
	if err := metadata.SaveMetadata(filepath.Join(specDir, "metadata.json"), meta); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	implTpl := "{{SPEC_ID}}\n{{SPEC_BODY}}\n{{ACCEPTANCE_COMMANDS}}\n{{PREVIOUS_REMAINING_TASKS}}\n{{MODE}}"
	if err := os.WriteFile(filepath.Join(specsRoot, "implement.prompt-template.md"), []byte(implTpl), 0o644); err != nil {
		t.Fatalf("write implement template: %v", err)
	}
	reviewTpl := "{{SPEC_ID}}\n{{IMPLEMENTATION_REPORT}}\n{{ACCEPTANCE_COMMANDS}}\n{{MODE}}"
	if err := os.WriteFile(filepath.Join(specsRoot, "review.prompt-template.md"), []byte(reviewTpl), 0o644); err != nil {
		t.Fatalf("write review template: %v", err)
	}
}

func slurp(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(data)
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
