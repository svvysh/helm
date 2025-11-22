package specs

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/polarzero/helm/internal/metadata"
)

func TestDiscoverSpecsAndDependencyState(t *testing.T) {
	root := t.TempDir()
	specsRoot := filepath.Join(root, "docs", "specs")

	makeSpec := func(dir string, meta *metadata.SpecMetadata, includeChecklist bool) {
		folder := filepath.Join(specsRoot, dir)
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", folder, err)
		}
		specFile := filepath.Join(folder, "SPEC.md")
		if err := os.WriteFile(specFile, []byte("# spec"), 0o644); err != nil {
			t.Fatalf("write SPEC.md: %v", err)
		}
		if includeChecklist {
			checklist := filepath.Join(folder, "acceptance-checklist.md")
			if err := os.WriteFile(checklist, []byte("- [ ] task"), 0o644); err != nil {
				t.Fatalf("write checklist: %v", err)
			}
		}
		metaPath := filepath.Join(folder, "metadata.json")
		if err := metadata.SaveMetadata(metaPath, meta); err != nil {
			t.Fatalf("save metadata: %v", err)
		}
	}

	now := time.Now().UTC()
	makeSpec("spec-00-foundation", &metadata.SpecMetadata{
		ID:        "spec-00-foundation",
		Name:      "Foundation",
		Status:    metadata.StatusDone,
		DependsOn: nil,
		LastRun:   &now,
	}, true)

	makeSpec("spec-01-config-metadata", &metadata.SpecMetadata{
		ID:        "spec-01-config-metadata",
		Name:      "Config",
		Status:    metadata.StatusTodo,
		DependsOn: []string{"spec-00-foundation"},
	}, false)

	makeSpec("spec-02-extra", &metadata.SpecMetadata{
		ID:        "spec-02-extra",
		Name:      "Extra",
		Status:    metadata.StatusTodo,
		DependsOn: []string{"spec-01-config-metadata", "spec-missing"},
	}, false)

	specs, err := DiscoverSpecs(specsRoot)
	if err != nil {
		t.Fatalf("DiscoverSpecs() error = %v", err)
	}

	if len(specs) != 3 {
		t.Fatalf("expected 3 specs, got %d", len(specs))
	}

	ComputeDependencyState(specs)

	lookup := func(id string) *SpecFolder {
		for _, s := range specs {
			if s.Metadata != nil && s.Metadata.ID == id {
				return s
			}
		}
		return nil
	}

	s0 := lookup("spec-00-foundation")
	if s0 == nil || s0.Checklist == "" {
		t.Fatalf("expected checklist path for foundation spec")
	}
	if s0.CanRun {
		t.Fatalf("done spec should not be runnable")
	}

	s1 := lookup("spec-01-config-metadata")
	if s1 == nil {
		t.Fatalf("spec-01-config-metadata not discovered")
	}
	if !s1.CanRun || len(s1.UnmetDeps) != 0 {
		t.Fatalf("spec-01-config-metadata should be runnable with no unmet deps: %+v", s1)
	}

	s2 := lookup("spec-02-extra")
	if s2 == nil {
		t.Fatalf("spec-02-extra not discovered")
	}
	if s2.CanRun {
		t.Fatalf("spec-02-extra should not be runnable")
	}
	if len(s2.UnmetDeps) != 2 {
		t.Fatalf("expected 2 unmet deps, got %v", s2.UnmetDeps)
	}
}
