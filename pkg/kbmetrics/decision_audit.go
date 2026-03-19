package kbmetrics

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DecisionType distinguishes architectural-principle from implementation decisions.
type DecisionType string

const (
	DecisionArchitectural  DecisionType = "architectural"
	DecisionImplementation DecisionType = "implementation"
)

// FindingStatus indicates whether an expected artifact was found.
type FindingStatus string

const (
	FindingPresent FindingStatus = "present"
	FindingMissing FindingStatus = "missing"
	FindingWeak    FindingStatus = "weak" // found reference but not strong enforcement
)

// Decision holds parsed metadata for one decision file.
type Decision struct {
	Path          string   `json:"path"`
	Title         string   `json:"title"`
	Date          string   `json:"date"`
	Status        string   `json:"status"`
	Type          DecisionType `json:"type"`
	BlockPatterns []string `json:"block_patterns,omitempty"`
	FileRefs      []string `json:"file_refs,omitempty"`
}

// Finding is one validation result for a decision.
type Finding struct {
	Check       string        `json:"check"`
	Status      FindingStatus `json:"status"`
	Detail      string        `json:"detail,omitempty"`
}

// DecisionReport is the audit result for one decision.
type DecisionReport struct {
	Decision Decision  `json:"decision"`
	Type     DecisionType `json:"type"`
	Findings []Finding `json:"findings"`
	Score    string    `json:"score"` // "enforced", "partial", "unanchored"
}

var (
	titleRe    = regexp.MustCompile(`^#\s+(?:Decision:\s*)?(.+)`)
	dateRe     = regexp.MustCompile(`\*\*Date:\*\*\s*(\S+)`)
	statusRe   = regexp.MustCompile(`\*\*Status:\*\*\s*(.+)`)
	fileRefRe  = regexp.MustCompile(`(?:^|[\s\x60(])([a-zA-Z_./]+\.(?:go|ts|js|yaml|yml|md|sh|sql))\b`)
	pathLikeRe = regexp.MustCompile(`(?:cmd|pkg|web|skills)/[a-zA-Z0-9_/.-]+\.(?:go|ts|js|yaml|yml|sh)`)

	// Architectural signal words in titles
	architecturalKeywords = []string{
		"principle", "pattern", "model", "architecture",
		"philosophy", "constraint", "boundary", "taxonomy",
	}
)

// AuditDecisions scans .kb/decisions/ and validates each decision.
// kbDir is the path to .kb/, projectDir is the repo root.
func AuditDecisions(kbDir, projectDir string) ([]DecisionReport, error) {
	decDir := filepath.Join(kbDir, "decisions")
	entries, err := os.ReadDir(decDir)
	if err != nil {
		return nil, fmt.Errorf("read decisions dir: %w", err)
	}

	var reports []DecisionReport
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(decDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		d, err := parseDecision(e.Name(), string(data))
		if err != nil {
			continue
		}

		var findings []Finding
		switch d.Type {
		case DecisionArchitectural:
			findings = validateArchitectural(d, projectDir)
		case DecisionImplementation:
			findings = validateImplementation(d, projectDir)
		}

		score := scoreFindings(findings)
		reports = append(reports, DecisionReport{
			Decision: d,
			Type:     d.Type,
			Findings: findings,
			Score:    score,
		})
	}

	sort.Slice(reports, func(i, j int) bool {
		// Unanchored first, then by date descending
		if reports[i].Score != reports[j].Score {
			return scoreOrder(reports[i].Score) < scoreOrder(reports[j].Score)
		}
		return reports[i].Decision.Date > reports[j].Decision.Date
	})

	return reports, nil
}

func scoreOrder(s string) int {
	switch s {
	case "unanchored":
		return 0
	case "partial":
		return 1
	case "enforced":
		return 2
	default:
		return 3
	}
}

func scoreFindings(findings []Finding) string {
	if len(findings) == 0 {
		return "unanchored"
	}
	present, total := 0, 0
	for _, f := range findings {
		total++
		if f.Status == FindingPresent {
			present++
		}
	}
	if total == 0 {
		return "unanchored"
	}
	ratio := float64(present) / float64(total)
	if ratio >= 0.5 {
		return "enforced"
	}
	if present > 0 {
		return "partial"
	}
	return "unanchored"
}

func parseDecision(filename, content string) (Decision, error) {
	d := Decision{Path: filename}

	// Parse frontmatter
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content[3:], "---", 2)
		if len(parts) == 2 {
			fm := parts[0]
			content = parts[1]
			d.BlockPatterns = extractFrontmatterPatterns(fm)
		}
	}

	// Parse title
	for _, line := range strings.Split(content, "\n") {
		if m := titleRe.FindStringSubmatch(line); len(m) > 1 {
			d.Title = strings.TrimSpace(m[1])
			break
		}
	}

	// Parse date
	if m := dateRe.FindStringSubmatch(content); len(m) > 1 {
		d.Date = m[1]
	}

	// Parse status
	if m := statusRe.FindStringSubmatch(content); len(m) > 1 {
		d.Status = strings.TrimSpace(m[1])
	}

	// Extract file references from body
	d.FileRefs = extractFileReferences(content)

	// Classify
	d.Type = classifyDecision(d.Title, content)

	return d, nil
}

func extractFrontmatterPatterns(fm string) []string {
	var patterns []string
	inPatterns := false
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "patterns:") {
			inPatterns = true
			continue
		}
		if inPatterns {
			if strings.HasPrefix(trimmed, "- ") {
				// Could be next list item or a pattern value
				val := strings.TrimPrefix(trimmed, "- ")
				val = strings.Trim(val, "\"'")
				if strings.Contains(val, "*") || strings.Contains(val, "/") {
					patterns = append(patterns, val)
				} else {
					inPatterns = false
				}
			} else if !strings.HasPrefix(trimmed, "#") && trimmed != "" && !strings.HasPrefix(trimmed, "\"") {
				inPatterns = false
			}
		}
	}
	return patterns
}

func classifyDecision(title, content string) DecisionType {
	lower := strings.ToLower(title)
	for _, kw := range architecturalKeywords {
		if strings.Contains(lower, kw) {
			return DecisionArchitectural
		}
	}

	// If it has block patterns or "What Changes" section, it's implementation
	if strings.Contains(content, "blocks:") && strings.Contains(content, "patterns:") {
		return DecisionImplementation
	}
	if strings.Contains(content, "## What Changes") || strings.Contains(content, "### What Changes") {
		return DecisionImplementation
	}
	if strings.Contains(content, "### Exact Code Changes") {
		return DecisionImplementation
	}

	// If stability: foundational in frontmatter, architectural
	if strings.Contains(content, "stability: foundational") {
		return DecisionArchitectural
	}

	// Default: if it has file references, implementation; otherwise architectural
	refs := extractFileReferences(content)
	if len(refs) >= 3 {
		return DecisionImplementation
	}

	return DecisionArchitectural
}

func extractFileReferences(content string) []string {
	seen := make(map[string]bool)
	var refs []string
	for _, m := range pathLikeRe.FindAllString(content, -1) {
		if !seen[m] {
			seen[m] = true
			refs = append(refs, m)
		}
	}
	return refs
}

// validateArchitectural checks if an architectural principle is reflected
// in gates, hooks, tests, or CLAUDE.md.
func validateArchitectural(d Decision, projectDir string) []Finding {
	var findings []Finding

	// Derive search terms from the title
	terms := architecturalSearchTerms(d.Title)
	if len(terms) == 0 {
		return findings
	}

	// Check 1: Is the principle mentioned in test files?
	testHits := searchFiles(projectDir, terms, "*_test.go")
	if len(testHits) > 0 {
		findings = append(findings, Finding{
			Check:  "test enforcement",
			Status: FindingPresent,
			Detail: fmt.Sprintf("found in %d test file(s): %s", len(testHits), joinMax(testHits, 3)),
		})
	} else {
		findings = append(findings, Finding{
			Check:  "test enforcement",
			Status: FindingMissing,
			Detail: "no test files reference this principle",
		})
	}

	// Check 2: Is it reflected in gates (pkg/spawn/gates/ or pkg/verify/)?
	gateHits := searchDirs(projectDir, terms, []string{
		"pkg/spawn/gates",
		"pkg/verify",
	})
	if len(gateHits) > 0 {
		findings = append(findings, Finding{
			Check:  "gate/verify enforcement",
			Status: FindingPresent,
			Detail: fmt.Sprintf("found in: %s", joinMax(gateHits, 3)),
		})
	}

	// Check 3: Is it in CLAUDE.md or hooks?
	docHits := searchFiles(projectDir, terms, "CLAUDE.md")
	hookHits := searchDirs(projectDir, terms, []string{".claude"})
	if len(docHits) > 0 || len(hookHits) > 0 {
		var where []string
		if len(docHits) > 0 {
			where = append(where, "CLAUDE.md")
		}
		if len(hookHits) > 0 {
			where = append(where, joinMax(hookHits, 2))
		}
		findings = append(findings, Finding{
			Check:  "documentation/hooks",
			Status: FindingPresent,
			Detail: fmt.Sprintf("found in: %s", strings.Join(where, ", ")),
		})
	} else {
		findings = append(findings, Finding{
			Check:  "documentation/hooks",
			Status: FindingMissing,
			Detail: "not found in CLAUDE.md or hooks",
		})
	}

	// Check 4: Is it in daemon config or skill files?
	configHits := searchDirs(projectDir, terms, []string{
		"pkg/daemon",
		"pkg/daemonconfig",
		"skills/src",
	})
	if len(configHits) > 0 {
		findings = append(findings, Finding{
			Check:  "daemon/skill config",
			Status: FindingPresent,
			Detail: fmt.Sprintf("found in: %s", joinMax(configHits, 3)),
		})
	}

	return findings
}

// validateImplementation checks if referenced files/patterns still exist.
func validateImplementation(d Decision, projectDir string) []Finding {
	var findings []Finding

	// Check block patterns from frontmatter
	for _, pat := range d.BlockPatterns {
		matches := globPattern(projectDir, pat)
		if len(matches) > 0 {
			findings = append(findings, Finding{
				Check:  fmt.Sprintf("pattern: %s", pat),
				Status: FindingPresent,
				Detail: fmt.Sprintf("%d file(s) match", len(matches)),
			})
		} else {
			findings = append(findings, Finding{
				Check:  fmt.Sprintf("pattern: %s", pat),
				Status: FindingMissing,
				Detail: "no files match this pattern",
			})
		}
	}

	// Check file references extracted from body
	for _, ref := range d.FileRefs {
		fullPath := filepath.Join(projectDir, ref)
		if _, err := os.Stat(fullPath); err == nil {
			findings = append(findings, Finding{
				Check:  fmt.Sprintf("file: %s", ref),
				Status: FindingPresent,
			})
		} else {
			findings = append(findings, Finding{
				Check:  fmt.Sprintf("file: %s", ref),
				Status: FindingMissing,
				Detail: "file not found",
			})
		}
	}

	return findings
}

// architecturalSearchTerms extracts meaningful search terms from a decision title.
func architecturalSearchTerms(title string) []string {
	// Remove common noise words, split into searchable phrases
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}

	// Build terms from title words, excluding noise
	noise := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"for": true, "and": true, "or": true, "not": true, "in": true,
		"of": true, "to": true, "with": true, "from": true, "by": true,
		"on": true, "at": true, "as": true, "over": true, "vs": true,
		"decision": true,
	}

	words := strings.Fields(strings.ToLower(title))
	var significant []string
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?()\"'")
		if len(w) > 2 && !noise[w] {
			significant = append(significant, w)
		}
	}

	if len(significant) == 0 {
		return nil
	}

	// Use the full title phrase only — individual words are too broad
	// and cause massive false positive rates
	var terms []string
	fullPhrase := strings.Join(significant, " ")
	terms = append(terms, fullPhrase)

	// Also add 2-word bigrams for partial matching
	if len(significant) >= 2 {
		for i := 0; i < len(significant)-1; i++ {
			bigram := significant[i] + " " + significant[i+1]
			// Skip bigrams with very common words
			if len(significant[i]) > 3 && len(significant[i+1]) > 3 {
				terms = append(terms, bigram)
			}
		}
	}

	return terms
}

func isArchitecturalKeyword(w string) bool {
	for _, kw := range architecturalKeywords {
		if w == kw {
			return true
		}
	}
	return false
}

func searchFiles(projectDir string, terms []string, fileGlob string) []string {
	var hits []string
	seen := make(map[string]bool)

	filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Skip hidden dirs except .claude
		base := filepath.Base(path)
		if info.IsDir() {
			if strings.HasPrefix(base, ".") && base != ".claude" {
				return filepath.SkipDir
			}
			if base == "vendor" || base == "node_modules" || base == "worktrees" {
				return filepath.SkipDir
			}
			return nil
		}
		matched, _ := filepath.Match(fileGlob, base)
		if !matched {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lower := strings.ToLower(string(data))
		for _, term := range terms {
			if strings.Contains(lower, term) {
				rel, _ := filepath.Rel(projectDir, path)
				if !seen[rel] {
					seen[rel] = true
					hits = append(hits, rel)
				}
				break
			}
		}
		return nil
	})
	return hits
}

func searchDirs(projectDir string, terms []string, dirs []string) []string {
	var hits []string
	seen := make(map[string]bool)

	for _, dir := range dirs {
		absDir := filepath.Join(projectDir, dir)
		filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				base := filepath.Base(path)
				if base == "worktrees" || base == "node_modules" || base == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".md") &&
				!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".json") &&
				!strings.HasSuffix(path, ".sh") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			lower := strings.ToLower(string(data))
			for _, term := range terms {
				if strings.Contains(lower, term) {
					rel, _ := filepath.Rel(projectDir, path)
					if !seen[rel] {
						seen[rel] = true
						hits = append(hits, rel)
					}
					break
				}
			}
			return nil
		})
	}
	return hits
}

func globPattern(projectDir string, pattern string) []string {
	// Convert ** glob patterns to Walk-based matching
	var matches []string
	// Simple approach: walk and match against the pattern suffix
	suffix := pattern
	for strings.HasPrefix(suffix, "**/") {
		suffix = suffix[3:]
	}

	filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		base := filepath.Base(path)
		if info.IsDir() {
			if strings.HasPrefix(base, ".") || base == "worktrees" || base == "vendor" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(projectDir, path)
		matched, _ := filepath.Match(suffix, rel)
		if !matched {
			// Also try matching against just the path suffix
			matched, _ = filepath.Match(suffix, filepath.Join(filepath.Base(filepath.Dir(path)), base))
			if !matched {
				// Try matching the full relative path components
				parts := strings.Split(rel, string(filepath.Separator))
				for i := range parts {
					subpath := filepath.Join(parts[i:]...)
					if m, _ := filepath.Match(suffix, subpath); m {
						matched = true
						break
					}
				}
			}
		}
		if matched {
			matches = append(matches, rel)
		}
		return nil
	})
	return matches
}

func joinMax(items []string, max int) string {
	if len(items) <= max {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:max], ", ") + fmt.Sprintf(" (+%d more)", len(items)-max)
}

// FormatDecisionAuditText renders a human-readable report.
func FormatDecisionAuditText(reports []DecisionReport) string {
	var b strings.Builder

	// Summary counts
	archCount, implCount := 0, 0
	enforced, partial, unanchored := 0, 0, 0
	for _, r := range reports {
		switch r.Type {
		case DecisionArchitectural:
			archCount++
		case DecisionImplementation:
			implCount++
		}
		switch r.Score {
		case "enforced":
			enforced++
		case "partial":
			partial++
		case "unanchored":
			unanchored++
		}
	}

	fmt.Fprintf(&b, "Decision Audit: %d decisions (%d architectural, %d implementation)\n",
		len(reports), archCount, implCount)
	fmt.Fprintf(&b, "Scores: %d enforced, %d partial, %d unanchored\n\n",
		enforced, partial, unanchored)

	// Show unanchored first (the actionable ones)
	for _, score := range []string{"unanchored", "partial", "enforced"} {
		var group []DecisionReport
		for _, r := range reports {
			if r.Score == score {
				group = append(group, r)
			}
		}
		if len(group) == 0 {
			continue
		}

		label := strings.ToUpper(score)
		fmt.Fprintf(&b, "--- %s (%d) ---\n\n", label, len(group))

		for _, r := range group {
			typeTag := "ARCH"
			if r.Type == DecisionImplementation {
				typeTag = "IMPL"
			}
			fmt.Fprintf(&b, "[%s] %s  (%s, %s)\n", typeTag, r.Decision.Title, r.Decision.Date, r.Decision.Status)

			for _, f := range r.Findings {
				icon := "?"
				switch f.Status {
				case FindingPresent:
					icon = "+"
				case FindingMissing:
					icon = "-"
				case FindingWeak:
					icon = "~"
				}
				if f.Detail != "" {
					fmt.Fprintf(&b, "  %s %s: %s\n", icon, f.Check, f.Detail)
				} else {
					fmt.Fprintf(&b, "  %s %s\n", icon, f.Check)
				}
			}
			fmt.Fprintln(&b)
		}
	}

	return b.String()
}
