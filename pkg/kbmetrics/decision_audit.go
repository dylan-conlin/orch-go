package kbmetrics

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DecisionAuditReport is the top-level result of auditing decisions.
type DecisionAuditReport struct {
	TotalDecisions    int                  `json:"total_decisions"`
	AcceptedDecisions int                  `json:"accepted_decisions"`
	WithIssues        int                  `json:"with_issues"`
	Entries           []DecisionAuditEntry `json:"entries,omitempty"`
}

// DecisionAuditEntry is the audit result for a single decision.
type DecisionAuditEntry struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Status       string   `json:"status"`
	Title        string   `json:"title,omitempty"`
	MissingFiles []string `json:"missing_files,omitempty"`
	ExistingFiles []string `json:"existing_files,omitempty"`
}

var (
	// Frontmatter status: "status: accepted"
	fmStatusRe = regexp.MustCompile(`(?m)^status:\s*(.+)$`)

	// Body status: "**Status:** Accepted" or "**Status:** Partially Implemented (notes)"
	bodyStatusRe = regexp.MustCompile(`(?m)^\*\*Status:\*\*\s*(.+)$`)

	// File references in backtick-delimited paths
	// Matches: `path/to/file.ext`, `.kb/guides/foo.md`, `cmd/orch/bar.go:funcName()`
	backtickPathRe = regexp.MustCompile("`([a-zA-Z0-9_./-]+(?:\\.[a-zA-Z0-9]+)(?::[a-zA-Z0-9_()]+)?)`")

	// Decision title: "# Decision: Title" or "# Title"
	decisionTitleRe = regexp.MustCompile(`(?m)^#\s+(?:Decision:\s*)?(.+)$`)
)

// parseDecisionStatus extracts the status from a decision file.
// Checks frontmatter first, then body **Status:** lines.
// Returns lowercase normalized status (e.g., "accepted", "superseded", "proposed").
func parseDecisionStatus(content string) string {
	// Check if content has frontmatter
	if strings.HasPrefix(content, "---") {
		endIdx := strings.Index(content[3:], "---")
		if endIdx > 0 {
			frontmatter := content[3 : 3+endIdx]
			if m := fmStatusRe.FindStringSubmatch(frontmatter); m != nil {
				return strings.ToLower(strings.TrimSpace(m[1]))
			}
		}
	}

	// Check body for **Status:** line
	if m := bodyStatusRe.FindStringSubmatch(content); m != nil {
		statusText := strings.TrimSpace(m[1])
		// Normalize: take the first word(s) before parenthetical or dash
		statusText = strings.ToLower(statusText)
		// Handle "Partially Implemented (reviewed ...)" → "partially implemented"
		// Handle "Superseded (partially)" → "superseded"
		// Handle "Active" → "active"
		for _, prefix := range []string{"partially implemented", "superseded", "accepted", "proposed", "active", "not implemented", "deprecated"} {
			if strings.HasPrefix(statusText, prefix) {
				return prefix
			}
		}
		// Fallback: first word
		parts := strings.Fields(statusText)
		if len(parts) > 0 {
			return parts[0]
		}
	}

	return ""
}

// extractFileReferences finds file paths referenced in backtick-delimited text.
// Strips function/line suffixes (e.g., `:archiveStaleWorkspaces()` → `cmd/orch/clean_cmd.go`).
func extractFileReferences(content string) []string {
	matches := backtickPathRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var refs []string

	for _, m := range matches {
		ref := m[1]
		// Strip function/line suffix after colon
		if idx := strings.Index(ref, ":"); idx > 0 {
			ref = ref[:idx]
		}
		// Skip obvious non-file-paths
		if !looksLikeFilePath(ref) {
			continue
		}
		if !seen[ref] {
			seen[ref] = true
			refs = append(refs, ref)
		}
	}

	sort.Strings(refs)
	return refs
}

// looksLikeFilePath returns true if the string looks like a relative file path.
func looksLikeFilePath(s string) bool {
	// Must contain a slash or start with a dot
	if !strings.Contains(s, "/") && !strings.HasPrefix(s, ".") {
		return false
	}
	// Must have a file extension
	ext := filepath.Ext(s)
	if ext == "" {
		return false
	}
	// Common file extensions
	validExts := map[string]bool{
		".go": true, ".md": true, ".yaml": true, ".yml": true,
		".json": true, ".ts": true, ".tsx": true, ".js": true,
		".jsx": true, ".svelte": true, ".sql": true, ".sh": true,
		".py": true, ".toml": true, ".css": true, ".html": true,
	}
	return validExts[ext]
}

// parseDecisionTitle extracts the title from a decision file.
func parseDecisionTitle(content string) string {
	if m := decisionTitleRe.FindStringSubmatch(content); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// isAcceptedLike returns true for statuses that imply implementation should exist.
func isAcceptedLike(status string) bool {
	switch status {
	case "accepted", "active", "partially implemented":
		return true
	}
	return false
}

// AuditDecisions scans decision files in .kb/decisions/ and .kb/global/decisions/
// for implementation evidence. For each Accepted decision, checks that referenced
// files exist. Returns a report with divergences.
func AuditDecisions(projectDir string) (*DecisionAuditReport, error) {
	kbDir := filepath.Join(projectDir, ".kb")

	// Collect decision files from both local and global directories
	dirs := []string{
		filepath.Join(kbDir, "decisions"),
		filepath.Join(kbDir, "global", "decisions"),
	}

	report := &DecisionAuditReport{}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read decisions dir %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			decPath := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(decPath)
			if err != nil {
				continue
			}

			content := string(data)
			status := parseDecisionStatus(content)
			report.TotalDecisions++

			if !isAcceptedLike(status) {
				continue
			}

			report.AcceptedDecisions++
			title := parseDecisionTitle(content)
			refs := extractFileReferences(content)

			auditEntry := DecisionAuditEntry{
				Name:   entry.Name(),
				Path:   decPath,
				Status: status,
				Title:  title,
			}

			for _, ref := range refs {
				// Resolve path relative to project root
				fullPath := filepath.Join(projectDir, ref)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					auditEntry.MissingFiles = append(auditEntry.MissingFiles, ref)
				} else {
					auditEntry.ExistingFiles = append(auditEntry.ExistingFiles, ref)
				}
			}

			report.Entries = append(report.Entries, auditEntry)
		}
	}

	// Count entries with issues
	for _, e := range report.Entries {
		if len(e.MissingFiles) > 0 {
			report.WithIssues++
		}
	}

	// Sort: entries with missing files first
	sort.Slice(report.Entries, func(i, j int) bool {
		iMissing := len(report.Entries[i].MissingFiles)
		jMissing := len(report.Entries[j].MissingFiles)
		if iMissing != jMissing {
			return iMissing > jMissing
		}
		return report.Entries[i].Name < report.Entries[j].Name
	})

	return report, nil
}

// FormatDecisionAuditText produces a human-readable decision audit report.
func FormatDecisionAuditText(report *DecisionAuditReport, verbose bool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Decision Implementation Audit — %d decisions\n", report.TotalDecisions))
	b.WriteString(strings.Repeat("=", 55) + "\n\n")
	b.WriteString(fmt.Sprintf("Total decisions:     %d\n", report.TotalDecisions))
	b.WriteString(fmt.Sprintf("Accepted/Active:     %d\n", report.AcceptedDecisions))
	b.WriteString(fmt.Sprintf("With missing refs:   %d\n\n", report.WithIssues))

	if report.WithIssues == 0 && !verbose {
		b.WriteString("All accepted decisions have valid file references.\n")
		return b.String()
	}

	// Show entries with missing files
	for _, e := range report.Entries {
		if len(e.MissingFiles) == 0 && !verbose {
			continue
		}

		title := e.Title
		if title == "" {
			title = e.Name
		}
		if len(title) > 60 {
			title = title[:60] + "..."
		}

		if len(e.MissingFiles) > 0 {
			b.WriteString(fmt.Sprintf("⚠ %s [%s]\n", title, e.Status))
			b.WriteString(fmt.Sprintf("  File: %s\n", e.Name))
			for _, mf := range e.MissingFiles {
				b.WriteString(fmt.Sprintf("  ✗ %s\n", mf))
			}
			if verbose {
				for _, ef := range e.ExistingFiles {
					b.WriteString(fmt.Sprintf("  ✓ %s\n", ef))
				}
			}
			b.WriteString("\n")
		} else if verbose {
			b.WriteString(fmt.Sprintf("✓ %s [%s]\n", title, e.Status))
			b.WriteString(fmt.Sprintf("  File: %s\n", e.Name))
			for _, ef := range e.ExistingFiles {
				b.WriteString(fmt.Sprintf("  ✓ %s\n", ef))
			}
			b.WriteString("\n")
		}
	}

	// Summary of no-ref decisions
	noRefCount := 0
	for _, e := range report.Entries {
		if len(e.MissingFiles) == 0 && len(e.ExistingFiles) == 0 {
			noRefCount++
		}
	}
	if noRefCount > 0 {
		b.WriteString(fmt.Sprintf("Note: %d accepted decisions have no file references at all.\n", noRefCount))
		if verbose {
			for _, e := range report.Entries {
				if len(e.MissingFiles) == 0 && len(e.ExistingFiles) == 0 {
					title := e.Title
					if title == "" {
						title = e.Name
					}
					b.WriteString(fmt.Sprintf("  - %s\n", title))
				}
			}
		}
	}

	return b.String()
}
