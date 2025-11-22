package run

import (
	"os"
	"path/filepath"
	"testing"
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
