package specs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/polarzero/helm/internal/metadata"
)

// SpecFolder represents a spec-* directory on disk.
type SpecFolder struct {
	ID        string
	Name      string
	Path      string
	Metadata  *metadata.SpecMetadata
	Checklist string
	CanRun    bool
	UnmetDeps []string
}

// DiscoverSpecs scans the provided root for spec-* directories and loads metadata.
func DiscoverSpecs(root string) ([]*SpecFolder, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read specs root %s: %w", root, err)
	}

	var folders []*SpecFolder

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "spec-") {
			continue
		}

		folderPath := filepath.Join(root, entry.Name())
		if err := ensureFileExists(folderPath, "SPEC.md"); err != nil {
			return nil, err
		}
		if err := ensureFileExists(folderPath, "metadata.json"); err != nil {
			return nil, err
		}

		metaPath := filepath.Join(folderPath, "metadata.json")
		meta, err := metadata.LoadMetadata(metaPath)
		if err != nil {
			return nil, err
		}

		folder := &SpecFolder{
			ID:       meta.ID,
			Name:     meta.Name,
			Path:     folderPath,
			Metadata: meta,
		}

		checklistPath := filepath.Join(folderPath, "acceptance-checklist.md")
		if _, err := os.Stat(checklistPath); err == nil {
			folder.Checklist = checklistPath
		}

		folders = append(folders, folder)
	}

	sort.Slice(folders, func(i, j int) bool {
		return folders[i].ID < folders[j].ID
	})

	return folders, nil
}

// ComputeDependencyState annotates specs with dependency state derived from metadata.
func ComputeDependencyState(specs []*SpecFolder) {
	statusByID := make(map[string]metadata.SpecStatus, len(specs))
	for _, spec := range specs {
		if spec.Metadata != nil {
			statusByID[spec.Metadata.ID] = spec.Metadata.Status
		}
	}

	for _, spec := range specs {
		spec.CanRun = false
		spec.UnmetDeps = nil
		if spec.Metadata == nil {
			continue
		}

		if spec.Metadata.Status == metadata.StatusDone {
			continue
		}

		for _, dep := range spec.Metadata.DependsOn {
			status, ok := statusByID[dep]
			if !ok || status != metadata.StatusDone {
				spec.UnmetDeps = append(spec.UnmetDeps, dep)
			}
		}

		spec.CanRun = len(spec.UnmetDeps) == 0
	}
}

func ensureFileExists(folderPath, name string) error {
	path := filepath.Join(folderPath, name)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("spec folder %s missing %s: %w", folderPath, name, err)
	}
	return nil
}
