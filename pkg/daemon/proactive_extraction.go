// Package daemon provides autonomous overnight processing capabilities.
// Proactive extraction creates architect issues when files cross 1200 lines,
// before they reach the critical 1500-line threshold that blocks spawning.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

const (
	// ProactiveExtractionThreshold is the line count that triggers proactive architect issue creation.
	// Files between 1200-1500 lines get architect issues. Files >1500 are handled by the critical gate.
	ProactiveExtractionThreshold = 1200

	// CriticalExtractionThreshold is the line count handled by the existing critical extraction gate.
	// Proactive extraction skips files above this threshold to avoid duplicate issues.
	CriticalExtractionThreshold = 1500

	// ProactiveExtractionLabel is the beads label for dedup of proactive extraction issues.
	ProactiveExtractionLabel = "daemon:proactive-extraction"
)

// ProactiveExtractionFile represents a source file that crossed the extraction threshold.
type ProactiveExtractionFile struct {
	Path  string
	Lines int
}

// ProactiveExtractionResult contains the result of a proactive extraction scan.
type ProactiveExtractionResult struct {
	// Scanned is the number of files above threshold found by the scan.
	Scanned int
	// Created is the number of architect issues created.
	Created int
	// Skipped is the number of files skipped due to existing open issues (dedup).
	Skipped int
	// SkippedCritical is the number of files skipped because they exceed 1500 lines
	// (handled by the critical extraction gate already).
	SkippedCritical int
	// CreatedIssues contains the beads IDs of created issues.
	CreatedIssues []string
	// Message is a human-readable summary.
	Message string
	// Error is set if the operation failed.
	Error error
}

// ProactiveExtractionService provides I/O operations for proactive extraction scanning.
type ProactiveExtractionService interface {
	// ScanFilesAboveThreshold returns source files with line counts above the given threshold.
	ScanFilesAboveThreshold(threshold int) ([]ProactiveExtractionFile, error)
	// HasOpenExtractionIssue checks if an open architect issue already exists for this file.
	HasOpenExtractionIssue(filePath string) (bool, error)
	// CreateArchitectIssue creates a beads issue for architect review of the file.
	CreateArchitectIssue(filePath string, lines int) (string, error)
}

// RunPeriodicProactiveExtraction scans source files and creates architect issues
// for files that cross 1200 lines but are below the 1500-line critical threshold.
// This gives the orchestrator a heads-up to plan extraction before files become critical.
func (d *Daemon) RunPeriodicProactiveExtraction() *ProactiveExtractionResult {
	if !d.Scheduler.IsDue(TaskProactiveExtraction) {
		return nil
	}

	svc := d.ProactiveExtraction
	if svc == nil {
		return &ProactiveExtractionResult{
			Error:   fmt.Errorf("proactive extraction service not configured"),
			Message: "Proactive extraction: service not configured",
		}
	}

	// Scan for files above the proactive threshold
	files, err := svc.ScanFilesAboveThreshold(ProactiveExtractionThreshold)
	if err != nil {
		return &ProactiveExtractionResult{
			Error:   err,
			Message: fmt.Sprintf("Proactive extraction: scan failed: %v", err),
		}
	}

	result := &ProactiveExtractionResult{
		Scanned: len(files),
	}

	for _, f := range files {
		// Skip files above the critical threshold — those are handled by the
		// existing CRITICAL extraction gate (>1500 lines) in the spawn pipeline.
		if f.Lines > CriticalExtractionThreshold {
			result.SkippedCritical++
			continue
		}

		// Dedup: skip if an open architect/extraction issue already exists for this file.
		hasOpen, err := svc.HasOpenExtractionIssue(f.Path)
		if err != nil {
			// Non-fatal: skip on error (fail-safe)
			result.Skipped++
			continue
		}
		if hasOpen {
			result.Skipped++
			continue
		}

		// Create architect issue
		issueID, err := svc.CreateArchitectIssue(f.Path, f.Lines)
		if err != nil {
			result.Error = err
			result.Message = fmt.Sprintf("Proactive extraction: failed to create issue for %s: %v", f.Path, err)
			continue
		}

		result.Created++
		result.CreatedIssues = append(result.CreatedIssues, issueID)
	}

	// Build summary message
	if result.Created > 0 {
		result.Message = fmt.Sprintf("Proactive extraction: created %d architect issue(s) for files approaching critical size", result.Created)
	} else if result.Scanned > 0 {
		result.Message = fmt.Sprintf("Proactive extraction: %d file(s) scanned, all skipped (dedup: %d, critical: %d)",
			result.Scanned, result.Skipped, result.SkippedCritical)
	} else if result.Error == nil {
		result.Message = "Proactive extraction: no files above 1200-line threshold"
	}

	d.Scheduler.MarkRun(TaskProactiveExtraction)
	return result
}

// --- Default production implementation ---

// defaultProactiveExtractionService is the production implementation that scans
// the project directory and creates beads issues.
type defaultProactiveExtractionService struct{}

// NewDefaultProactiveExtractionService creates a production ProactiveExtractionService.
func NewDefaultProactiveExtractionService() ProactiveExtractionService {
	return &defaultProactiveExtractionService{}
}

func (s *defaultProactiveExtractionService) ScanFilesAboveThreshold(threshold int) ([]ProactiveExtractionFile, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	var files []ProactiveExtractionFile

	err = filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			// Skip non-source directories (same set as hotspot.go skipBloatDirs)
			switch name {
			case ".git", "node_modules", "vendor", ".svelte-kit", "dist", "build",
				"__pycache__", ".next", ".nuxt", ".output", ".opencode", ".orch", ".beads", ".claude":
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(projectDir, path)
		if err != nil {
			return nil
		}

		// Only count Go source files (the primary language of this project)
		if !strings.HasSuffix(relPath, ".go") {
			return nil
		}
		// Skip test files — they can be long without needing extraction
		if strings.HasSuffix(relPath, "_test.go") {
			return nil
		}

		lineCount, err := countFileLines(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		if lineCount >= threshold {
			files = append(files, ProactiveExtractionFile{
				Path:  relPath,
				Lines: lineCount,
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	return files, nil
}

// countFileLines counts newlines in a file. Reuses the same buffer approach as hotspot.go countLines.
func countFileLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	buf := make([]byte, 32*1024)

	for {
		c, err := file.Read(buf)
		for i := 0; i < c; i++ {
			if buf[i] == '\n' {
				count++
			}
		}
		if err != nil {
			break
		}
	}

	return count, nil
}

func (s *defaultProactiveExtractionService) HasOpenExtractionIssue(filePath string) (bool, error) {
	issues, err := ListIssuesWithLabel(ProactiveExtractionLabel)
	if err != nil {
		return false, err
	}
	baseName := filepath.Base(filePath)
	for _, issue := range issues {
		titleLower := strings.ToLower(issue.Title)
		if strings.Contains(titleLower, strings.ToLower(filePath)) ||
			strings.Contains(titleLower, strings.ToLower(baseName)) {
			return true, nil
		}
	}
	return false, nil
}

func (s *defaultProactiveExtractionService) CreateArchitectIssue(filePath string, lines int) (string, error) {
	title := fmt.Sprintf("Architect: plan extraction for %s (%d lines, approaching 1500-line limit)",
		filePath, lines)

	desc := fmt.Sprintf("%s has grown to %d lines. The critical extraction threshold is 1500 lines, "+
		"at which point spawn gates block feature-impl and systematic-debugging skills. "+
		"Plan extraction now to avoid blocking future work. "+
		"See .kb/guides/code-extraction-patterns.md for extraction workflow.",
		filePath, lines)

	// Try RPC first, fallback to CLI
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Create(&beads.CreateArgs{
				Title:       title,
				Description: desc,
				IssueType:   "task",
				Priority:    3,
				Labels:      []string{ProactiveExtractionLabel, "triage:ready"},
			})
			if err == nil {
				return issue.ID, nil
			}
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackCreate(title, desc, "task", 3, []string{ProactiveExtractionLabel, "triage:ready"}, "")
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}
