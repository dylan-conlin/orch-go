package kbmetrics

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// OrphanCategory classifies why an investigation is orphaned.
type OrphanCategory string

const (
	// CategoryEmpty — template-only or minimal content, no real findings.
	CategoryEmpty OrphanCategory = "empty"

	// CategoryNegativeResult — investigation concluded with a negative finding
	// (no bug, already fixed, not reproducible, hypothesis disproven).
	CategoryNegativeResult OrphanCategory = "negative-result"

	// CategorySuperseded — later work covers the same ground, or explicitly
	// marked as superseded/replaced/obsoleted.
	CategorySuperseded OrphanCategory = "superseded"

	// CategoryPositiveUnlinked — has genuine findings that should feed a model
	// or decision but don't. This is actual knowledge loss.
	CategoryPositiveUnlinked OrphanCategory = "positive-unlinked"
)

// StratifiedOrphanReport extends OrphanReport with per-category counts.
type StratifiedOrphanReport struct {
	OrphanReport

	// Categories maps each orphan category to its count.
	Categories map[OrphanCategory]int `json:"categories"`

	// CategoryPaths maps each category to the file paths in it.
	CategoryPaths map[OrphanCategory][]string `json:"category_paths,omitempty"`
}

// CategoryRate returns the percentage of orphans in a given category.
func (r *StratifiedOrphanReport) CategoryRate(cat OrphanCategory) float64 {
	if r.Orphaned == 0 {
		return 0
	}
	return float64(r.Categories[cat]) / float64(r.Orphaned) * 100
}

// StratifiedSummary returns a human-readable summary of the stratified report.
func (r *StratifiedOrphanReport) StratifiedSummary() string {
	if r.Total == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Investigation Orphan Rate (Stratified)\n")
	fmt.Fprintf(&b, "======================================\n\n")
	fmt.Fprintf(&b, "Total investigations:  %d\n", r.Total)
	fmt.Fprintf(&b, "Connected:             %d\n", r.Connected)
	fmt.Fprintf(&b, "Orphaned:              %d (%.1f%%)\n\n", r.Orphaned, r.OrphanRate)
	fmt.Fprintf(&b, "Orphan Breakdown:\n")

	order := []OrphanCategory{CategoryEmpty, CategoryNegativeResult, CategorySuperseded, CategoryPositiveUnlinked}
	for _, cat := range order {
		count := r.Categories[cat]
		rate := r.CategoryRate(cat)
		fmt.Fprintf(&b, "  %-20s %4d  (%5.1f%% of orphans)\n", cat, count, rate)
	}

	fmt.Fprintf(&b, "\nActionable (positive-unlinked): %d (%.1f%% of total investigations)\n",
		r.Categories[CategoryPositiveUnlinked],
		float64(r.Categories[CategoryPositiveUnlinked])/float64(r.Total)*100)

	return b.String()
}

// ComputeStratifiedOrphanRate computes orphan rates and classifies each orphan.
func ComputeStratifiedOrphanRate(kbDir string) (*StratifiedOrphanReport, error) {
	invDir := filepath.Join(kbDir, "investigations")

	invFiles, err := collectInvestigationFiles(invDir)
	if err != nil {
		return &StratifiedOrphanReport{Categories: make(map[OrphanCategory]int)}, nil
	}
	if len(invFiles) == 0 {
		return &StratifiedOrphanReport{Categories: make(map[OrphanCategory]int)}, nil
	}

	// Build investigation relative paths
	invRelPaths := make(map[string]bool, len(invFiles))
	relToAbs := make(map[string]string, len(invFiles))
	for _, f := range invFiles {
		rel, err := filepath.Rel(filepath.Dir(kbDir), f)
		if err != nil {
			continue
		}
		invRelPaths[rel] = false
		relToAbs[rel] = f
	}

	// Scan for references (same logic as ComputeOrphanRate)
	err = filepath.Walk(kbDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		rel, _ := filepath.Rel(filepath.Dir(kbDir), path)
		if strings.HasPrefix(rel, ".kb/investigations/") || strings.HasPrefix(rel, filepath.Join(".kb", "investigations")+string(filepath.Separator)) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		for relPath := range invRelPaths {
			searchPath := filepath.ToSlash(relPath)
			if strings.Contains(content, searchPath) {
				invRelPaths[relPath] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning kb dir: %w", err)
	}

	// Classify orphans
	categories := make(map[OrphanCategory]int)
	categoryPaths := make(map[OrphanCategory][]string)
	connected := 0

	for relPath, isConnected := range invRelPaths {
		if isConnected {
			connected++
			continue
		}
		absPath := relToAbs[relPath]
		cat := ClassifyOrphan(absPath)
		categories[cat]++
		categoryPaths[cat] = append(categoryPaths[cat], relPath)
	}

	total := len(invRelPaths)
	orphaned := total - connected
	rate := 0.0
	if total > 0 {
		rate = float64(orphaned) / float64(total) * 100
	}

	return &StratifiedOrphanReport{
		OrphanReport: OrphanReport{
			Total:      total,
			Connected:  connected,
			Orphaned:   orphaned,
			OrphanRate: rate,
		},
		Categories:    categories,
		CategoryPaths: categoryPaths,
	}, nil
}

// ClassifyOrphan reads an investigation file and classifies it into a category.
func ClassifyOrphan(path string) OrphanCategory {
	data, err := os.ReadFile(path)
	if err != nil {
		return CategoryPositiveUnlinked // can't read → assume positive
	}

	content := string(data)

	if isEmptyInvestigation(content) {
		return CategoryEmpty
	}
	if isSuperseded(content) {
		return CategorySuperseded
	}
	if isNegativeResult(content) {
		return CategoryNegativeResult
	}

	return CategoryPositiveUnlinked
}

// isEmptyInvestigation returns true if the file is mostly template with no findings.
func isEmptyInvestigation(content string) bool {
	lines := strings.Split(content, "\n")

	// Count lines with substantive prose (not template, headers, metadata, empty,
	// or structural markdown).
	contentLines := 0
	inComment := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Track HTML comments spanning multiple lines
		if strings.HasPrefix(trimmed, "<!--") {
			inComment = true
		}
		if inComment {
			if strings.Contains(trimmed, "-->") {
				inComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Metadata lines
		if strings.HasPrefix(trimmed, "**Status:**") ||
			strings.HasPrefix(trimmed, "**Created:**") ||
			strings.HasPrefix(trimmed, "**Beads") ||
			strings.HasPrefix(trimmed, "**Skill") ||
			strings.HasPrefix(trimmed, "**Model:**") ||
			strings.HasPrefix(trimmed, "**Date:**") ||
			strings.HasPrefix(trimmed, "**TLDR:**") {
			continue
		}
		// Template placeholders
		if strings.HasPrefix(trimmed, "- [ ]") {
			continue
		}
		// Table structure
		if strings.HasPrefix(trimmed, "| ---") {
			continue
		}
		// Horizontal rules
		if trimmed == "---" {
			continue
		}
		contentLines++
	}

	return contentLines < 5
}

var negativeResultPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)already (fixed|resolved|implemented|exists|added|working|present)`),
	regexp.MustCompile(`(?i)not (a bug|reproducible|an issue|broken)`),
	regexp.MustCompile(`(?i)no (bug|issue|problem|regression|error) found`),
	regexp.MustCompile(`(?i)(false positive|won't fix|wontfix|not needed)`),
	regexp.MustCompile(`(?i)negative result`),
	regexp.MustCompile(`(?i)hypothesis (disproven|rejected|invalidated|wrong)`),
	regexp.MustCompile(`(?i)doesn't reproduce`),
	regexp.MustCompile(`(?i)works (as expected|correctly|as designed|as intended)`),
	regexp.MustCompile(`(?i)no (action|change) (needed|required|necessary)`),
}

// isNegativeResult returns true if the investigation concluded with a negative finding.
func isNegativeResult(content string) bool {
	matches := 0
	for _, pat := range negativeResultPatterns {
		if pat.MatchString(content) {
			matches++
		}
	}
	// Require at least 2 distinct signals to classify as negative result.
	return matches >= 2
}

var supersededPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)superseded by`),
	regexp.MustCompile(`(?i)replaced by`),
	regexp.MustCompile(`(?i)obsoleted by`),
	regexp.MustCompile(`(?i)see instead[:\s]`),
	regexp.MustCompile(`(?i)covered by.*later`),
	regexp.MustCompile(`(?i)merged into`),
	regexp.MustCompile(`(?i)this investigation (is|was) (superseded|replaced|obsolete)`),
}

// isSuperseded returns true if the investigation is explicitly marked as superseded.
func isSuperseded(content string) bool {
	for _, pat := range supersededPatterns {
		if pat.MatchString(content) {
			return true
		}
	}
	return false
}
