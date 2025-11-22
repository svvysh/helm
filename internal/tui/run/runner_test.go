package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/polarzero/helm/internal/config"
)

func TestParseAttemptLine(t *testing.T) {
	line := "=== Attempt 2 of 5 for spec-99 ==="
	cur, total, ok := parseAttemptLine(line)
	if !ok || cur != 2 || total != 5 {
		t.Fatalf("expected attempt 2/5, got %d/%d ok=%v", cur, total, ok)
	}
	if _, _, ok := parseAttemptLine("no attempt info"); ok {
		t.Fatalf("expected parseAttemptLine to fail for invalid input")
	}
}

func TestBuildRunnerEnv(t *testing.T) {
	base := []string{"PATH=/tmp"}
	settings := &config.Settings{
		DefaultMaxAttempts: 3,
		CodexRunImpl:       config.CodexChoice{Model: "gpt-impl"},
		CodexRunVer:        config.CodexChoice{Model: "gpt-ver"},
	}
	env := buildRunnerEnv(base, settings)
	wantKeys := []string{"MAX_ATTEMPTS=3", "CODEX_MODEL_IMPL=gpt-impl", "CODEX_MODEL_VER=gpt-ver"}
	for _, key := range wantKeys {
		if !containsEnv(env, key) {
			t.Fatalf("expected env to contain %s, got %v", key, env)
		}
	}

	// Ensure existing values are preserved.
	env = buildRunnerEnv([]string{"MAX_ATTEMPTS=9"}, settings)
	if !containsEnv(env, "MAX_ATTEMPTS=9") {
		t.Fatalf("expected existing MAX_ATTEMPTS to be preserved, got %v", env)
	}
}

func containsEnv(env []string, target string) bool {
	for _, kv := range env {
		if kv == target {
			return true
		}
	}
	return false
}

func TestParseRemainingTasks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "implementation-report.md")
	content := "# Report\n\n## Remaining tasks\n\n{\n  \"remainingTasks\": [\n    \"first\",\n    \"second\"\n  ]\n}\n\n## Final worker output\n\n(done)\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tasks := parseRemainingTasks(path)
	if len(tasks) != 2 || tasks[0] != "first" || tasks[1] != "second" {
		t.Fatalf("unexpected tasks %v", tasks)
	}
}
