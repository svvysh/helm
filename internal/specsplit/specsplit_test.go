package specsplit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polarzero/helm/internal/metadata"
)

func TestSplitCreatesSpecFoldersFromPlanFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	planJSON := `{
  "specs": [
    {
      "index": 1,
      "idSuffix": "alpha",
      "name": "Alpha Spec",
      "summary": "Alpha summary",
      "dependsOn": ["spec-00-foundation"],
      "acceptanceCriteria": ["Do alpha"]
    },
    {
      "index": 2,
      "idSuffix": "beta chunk",
      "name": "Beta Spec",
      "summary": "",
      "dependsOn": ["alpha"],
      "acceptanceCriteria": ["Do beta"]
    }
  ]
}`

	planPath := filepath.Join(root, "plan.json")
	if err := os.WriteFile(planPath, []byte(planJSON), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	// Create a baseline spec so dependency names can be resolved.
	foundation := filepath.Join(root, "spec-00-foundation")
	if err := os.MkdirAll(foundation, 0o755); err != nil {
		t.Fatalf("mkdir foundation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(foundation, "SPEC.md"), []byte("# foundation"), 0o644); err != nil {
		t.Fatalf("write SPEC: %v", err)
	}
	meta := &metadata.SpecMetadata{
		ID:        "spec-00-foundation",
		Name:      "Foundation",
		Status:    metadata.StatusTodo,
		DependsOn: []string{},
	}
	if err := metadata.SaveMetadata(filepath.Join(foundation, "metadata.json"), meta); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	opts := Options{
		SpecsRoot:          root,
		RawSpec:            "# Big Spec\nMore details",
		AcceptanceCommands: []string{"go test ./...", "go vet ./..."},
		PlanPath:           planPath,
	}

	result, err := Split(context.Background(), opts)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(result.Specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(result.Specs))
	}

	alphaDir := filepath.Join(root, "spec-01-alpha")
	if _, err := os.Stat(alphaDir); err != nil {
		t.Fatalf("alpha dir missing: %v", err)
	}
	betaDir := filepath.Join(root, "spec-02-beta-chunk")
	if _, err := os.Stat(betaDir); err != nil {
		t.Fatalf("beta dir missing: %v", err)
	}

	betaMeta, err := metadata.LoadMetadata(filepath.Join(betaDir, "metadata.json"))
	if err != nil {
		t.Fatalf("load beta metadata: %v", err)
	}
	if len(betaMeta.DependsOn) != 1 || betaMeta.DependsOn[0] != "spec-01-alpha" {
		t.Fatalf("beta metadata deps = %v", betaMeta.DependsOn)
	}

	specBody, err := os.ReadFile(filepath.Join(betaDir, "SPEC.md"))
	if err != nil {
		t.Fatalf("read beta spec: %v", err)
	}
	body := string(specBody)
	if !strings.Contains(body, "(summary TBD)") {
		t.Fatalf("expected summary placeholder, got %s", body)
	}
	if !strings.Contains(body, "`spec-01-alpha`") {
		t.Fatalf("spec body missing dependency link: %s", body)
	}

	checklist, err := os.ReadFile(filepath.Join(alphaDir, "acceptance-checklist.md"))
	if err != nil {
		t.Fatalf("read checklist: %v", err)
	}
	if !strings.Contains(string(checklist), "`go test ./...`") {
		t.Fatalf("checklist missing automated command: %s", checklist)
	}
}

func TestSplitRenamesConflictingSpecIDs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Pre-create a conflicting folder.
	if err := os.MkdirAll(filepath.Join(root, "spec-01-alpha"), 0o755); err != nil {
		t.Fatalf("mkdir conflict: %v", err)
	}

	planJSON := `{"specs":[{"index":1,"idSuffix":"alpha","name":"Alpha","acceptanceCriteria":["A"]},{"index":2,"idSuffix":"beta","name":"Beta","dependsOn":["alpha"],"acceptanceCriteria":["B"]}]}`
	planPath := filepath.Join(root, "plan.json")
	if err := os.WriteFile(planPath, []byte(planJSON), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	opts := Options{
		SpecsRoot: root,
		RawSpec:   "Spec",
		PlanPath:  planPath,
	}
	result, err := Split(context.Background(), opts)
	if err != nil {
		t.Fatalf("split: %v", err)
	}

	if len(result.Specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(result.Specs))
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected warning for duplicate ID, got %v", result.Warnings)
	}

	alphaDir := filepath.Join(root, "spec-01-alpha-2")
	if _, err := os.Stat(alphaDir); err != nil {
		t.Fatalf("renamed alpha dir missing: %v", err)
	}
	betaMeta, err := metadata.LoadMetadata(filepath.Join(root, "spec-02-beta", "metadata.json"))
	if err != nil {
		t.Fatalf("load beta metadata: %v", err)
	}
	if len(betaMeta.DependsOn) != 1 || betaMeta.DependsOn[0] != "spec-01-alpha-2" {
		t.Fatalf("beta metadata deps = %v", betaMeta.DependsOn)
	}
}

func TestNormalizeSuffix(t *testing.T) {
	cases := map[string]string{
		"Alpha Feature":   "alpha-feature",
		" 123 ":           "123",
		"":                "spec",
		"Base__Feature!!": "base-feature",
	}
	for input, want := range cases {
		if got := normalizeSuffix(input); got != want {
			t.Fatalf("normalizeSuffix(%q)=%q want %q", input, got, want)
		}
	}
}
