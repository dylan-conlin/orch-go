// Package cleanup provides utilities for cleaning up investigation files.
package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArchiveEmptyInvestigationsOptions configures the investigation archival behavior.
type ArchiveEmptyInvestigationsOptions struct {
	// ProjectDir is the root directory of the project (contains .kb/investigations/)
	ProjectDir string
	// DryRun if true, only reports what would be archived without actually archiving
	DryRun bool
	// Quiet if true, suppresses progress output (for daemon use)
	Quiet bool
}

// emptyInvestigationPlaceholders are patterns that indicate an investigation file was never filled in.
// These are template placeholders from kb create investigation that agents should replace.
var emptyInvestigationPlaceholders = []string{
	"[Brief, descriptive title]",
	"[Clear, specific question",
	"[Concrete observations, data, examples]",
	"[File paths with line numbers",
	"[Explanation of the insight",
}

// isEmptyInvestigation checks if an investigation file still has template placeholders.
// Returns true if the file contains multiple placeholder patterns, indicating it was never filled in.
func isEmptyInvestigation(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	contentStr := string(content)
	placeholderCount := 0
	for _, placeholder := range emptyInvestigationPlaceholders {
		if strings.Contains(contentStr, placeholder) {
			placeholderCount++
		}
	}

	// Require at least 2 placeholder patterns to be considered empty
	// (to avoid false positives from files that just mention placeholders in documentation)
	return placeholderCount >= 2
}

// ArchiveEmptyInvestigations moves empty investigation files to .kb/investigations/archived/.
// Returns the number of files archived and any error encountered.
func ArchiveEmptyInvestigations(opts ArchiveEmptyInvestigationsOptions) (int, error) {
	investigationsDir := filepath.Join(opts.ProjectDir, ".kb", "investigations")
	archivedDir := filepath.Join(investigationsDir, "archived")

	// Check if investigations directory exists
	if _, err := os.Stat(investigationsDir); os.IsNotExist(err) {
		if !opts.Quiet {
			fmt.Println("\nNo .kb/investigations directory found")
		}
		return 0, nil
	}

	if !opts.Quiet {
		fmt.Println("\nScanning for empty investigation files...")
	}

	// Find all empty investigation files
	var emptyFiles []string
	err := filepath.Walk(investigationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories and non-markdown files
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Skip files already in archived folder
		if strings.Contains(path, "/archived/") {
			return nil
		}

		if isEmptyInvestigation(path) {
			emptyFiles = append(emptyFiles, path)
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to scan investigations: %w", err)
	}

	if len(emptyFiles) == 0 {
		if !opts.Quiet {
			fmt.Println("  No empty investigation files found")
		}
		return 0, nil
	}

	if !opts.Quiet {
		fmt.Printf("  Found %d empty investigation files:\n", len(emptyFiles))
	}

	// Create archived directory if needed
	if !opts.DryRun {
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create archived directory: %w", err)
		}
	}

	// Archive empty files
	archived := 0
	for _, path := range emptyFiles {
		filename := filepath.Base(path)

		// Preserve subdirectory structure (e.g., simple/)
		relPath, _ := filepath.Rel(investigationsDir, path)
		destDir := filepath.Join(archivedDir, filepath.Dir(relPath))
		destPath := filepath.Join(destDir, filename)

		if opts.DryRun {
			if !opts.Quiet {
				fmt.Printf("    [DRY-RUN] Would archive: %s\n", relPath)
			}
			archived++
			continue
		}

		// Create destination subdirectory if needed
		if err := os.MkdirAll(destDir, 0755); err != nil {
			if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "    Warning: failed to create directory %s: %v\n", destDir, err)
			}
			continue
		}

		// Check if destination already exists
		finalDestPath := destPath
		if _, err := os.Stat(destPath); err == nil {
			// Destination exists - add timestamp suffix to make it unique
			suffix := time.Now().Format("150405") // HHMMSS format
			// Insert suffix before .md extension
			baseName := strings.TrimSuffix(filename, ".md")
			finalDestPath = filepath.Join(destDir, baseName+"-"+suffix+".md")
			if !opts.Quiet {
				fmt.Printf("    Note: Archive destination exists, using: %s-%s.md\n", baseName, suffix)
			}
		}

		// Move file to archived
		if err := os.Rename(path, finalDestPath); err != nil {
			if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", relPath, err)
			}
			continue
		}

		if !opts.Quiet {
			fmt.Printf("    Archived: %s\n", relPath)
		}
		archived++
	}

	return archived, nil
}
