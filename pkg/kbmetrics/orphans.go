package kbmetrics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// OrphanReport summarizes the orphan rate for investigations.
type OrphanReport struct {
	Total      int     `json:"total"`
	Connected  int     `json:"connected"`
	Orphaned   int     `json:"orphaned"`
	OrphanRate float64 `json:"orphan_rate"` // percentage 0-100
}

// Summary returns a human-readable summary of the orphan rate.
func (r *OrphanReport) Summary() string {
	if r.Total == 0 {
		return ""
	}
	return fmt.Sprintf("%.1f%% orphan rate (%d/%d investigations unconnected)",
		r.OrphanRate, r.Orphaned, r.Total)
}

// ComputeOrphanRate counts investigations referenced by other .kb/ files
// vs total investigations. An investigation is "connected" if its relative
// path (e.g., .kb/investigations/inv-foo.md) appears in any other .kb/ file.
func ComputeOrphanRate(kbDir string) (*OrphanReport, error) {
	invDir := filepath.Join(kbDir, "investigations")

	// Collect all investigation file paths (relative to parent of kbDir)
	invFiles, err := collectInvestigationFiles(invDir)
	if err != nil {
		// No investigations directory is fine
		return &OrphanReport{}, nil
	}

	if len(invFiles) == 0 {
		return &OrphanReport{}, nil
	}

	// Build set of investigation basenames and relative paths for matching
	// We match on ".kb/investigations/..." patterns in file content
	invRelPaths := make(map[string]bool, len(invFiles))
	for _, f := range invFiles {
		// Get path relative to kbDir's parent (so it starts with .kb/)
		rel, err := filepath.Rel(filepath.Dir(kbDir), f)
		if err != nil {
			continue
		}
		invRelPaths[rel] = false // false = orphaned (not yet found)
	}

	// Scan all .kb/ markdown files for references to investigations
	err = filepath.Walk(kbDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Skip investigation files themselves — self-references don't count
		rel, _ := filepath.Rel(filepath.Dir(kbDir), path)
		if strings.HasPrefix(rel, ".kb/investigations/") || strings.HasPrefix(rel, filepath.Join(".kb", "investigations")+string(filepath.Separator)) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)

		// Check each investigation path
		for relPath := range invRelPaths {
			// Normalize to forward slashes for matching
			searchPath := filepath.ToSlash(relPath)
			if strings.Contains(content, searchPath) {
				invRelPaths[relPath] = true // connected
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning kb dir: %w", err)
	}

	// Count connected vs orphaned
	connected := 0
	for _, isConnected := range invRelPaths {
		if isConnected {
			connected++
		}
	}

	total := len(invRelPaths)
	orphaned := total - connected
	rate := 0.0
	if total > 0 {
		rate = float64(orphaned) / float64(total) * 100
	}

	return &OrphanReport{
		Total:      total,
		Connected:  connected,
		Orphaned:   orphaned,
		OrphanRate: rate,
	}, nil
}

// collectInvestigationFiles returns all .md files under the investigations directory.
func collectInvestigationFiles(invDir string) ([]string, error) {
	if _, err := os.Stat(invDir); os.IsNotExist(err) {
		return nil, nil
	}

	var files []string
	err := filepath.Walk(invDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
