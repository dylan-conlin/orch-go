package kbmetrics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionOrphanReport summarizes orphans scoped to a time window.
type SessionOrphanReport struct {
	Investigations int `json:"investigations"` // investigations created since cutoff
	Orphaned       int `json:"orphaned"`       // of those, how many are unlinked
}

// ComputeSessionOrphans counts investigations created since the given time
// and determines how many are orphaned (unlinked by other .kb/ files).
// It uses the YYYY-MM-DD date prefix in investigation filenames.
func ComputeSessionOrphans(kbDir string, since time.Time) (*SessionOrphanReport, error) {
	invDir := filepath.Join(kbDir, "investigations")

	invFiles, err := collectInvestigationFiles(invDir)
	if err != nil {
		return &SessionOrphanReport{}, nil
	}

	if len(invFiles) == 0 {
		return &SessionOrphanReport{}, nil
	}

	cutoffDate := since.Format("2006-01-02")

	// Filter to investigations created on or after the cutoff date
	var recentFiles []string
	for _, f := range invFiles {
		base := filepath.Base(f)
		if len(base) >= 10 {
			fileDate := base[:10]
			if fileDate >= cutoffDate {
				recentFiles = append(recentFiles, f)
			}
		}
	}

	if len(recentFiles) == 0 {
		return &SessionOrphanReport{}, nil
	}

	// Build set of recent investigation relative paths
	recentRelPaths := make(map[string]bool, len(recentFiles))
	for _, f := range recentFiles {
		rel, err := filepath.Rel(filepath.Dir(kbDir), f)
		if err != nil {
			continue
		}
		recentRelPaths[rel] = false // false = orphaned
	}

	// Scan all .kb/ markdown files for references
	err = filepath.Walk(kbDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Skip investigation files themselves
		rel, _ := filepath.Rel(filepath.Dir(kbDir), path)
		if strings.HasPrefix(rel, ".kb/investigations/") || strings.HasPrefix(rel, filepath.Join(".kb", "investigations")+string(filepath.Separator)) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)

		for relPath := range recentRelPaths {
			searchPath := filepath.ToSlash(relPath)
			if strings.Contains(content, searchPath) {
				recentRelPaths[relPath] = true
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning kb dir: %w", err)
	}

	orphaned := 0
	for _, isConnected := range recentRelPaths {
		if !isConnected {
			orphaned++
		}
	}

	return &SessionOrphanReport{
		Investigations: len(recentRelPaths),
		Orphaned:       orphaned,
	}, nil
}
