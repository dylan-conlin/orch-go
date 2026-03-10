package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// analyzeFixCommits counts fix: commits per file in the git history.
func analyzeFixCommits(projectDir string, daysBack, threshold int) ([]Hotspot, int, error) {
	// Get fix commits from git log
	// Format: commit hash | commit message
	since := fmt.Sprintf("--since=%d days ago", daysBack)
	cmd := exec.Command("git", "log", since, "--pretty=format:%H|%s", "--name-only", "--diff-filter=ACMR")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, 0, fmt.Errorf("git log failed: %w", err)
	}

	// Parse output to count fix commits per file
	fileCounts := make(map[string]int)
	totalFixes := 0

	lines := strings.Split(string(output), "\n")
	var currentCommit string
	isFixCommit := false
	fixPattern := regexp.MustCompile(`^fix(\(.+\))?:`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a commit line (contains |)
		if strings.Contains(line, "|") {
			parts := strings.SplitN(line, "|", 2)
			if len(parts) == 2 {
				currentCommit = parts[0]
				message := strings.ToLower(parts[1])
				isFixCommit = fixPattern.MatchString(message)
				if isFixCommit {
					totalFixes++
				}
				continue
			}
		}

		// This is a file path line
		if isFixCommit && currentCommit != "" && line != "" {
			// Only count meaningful source files
			if shouldCountFile(line) {
				fileCounts[line]++
			}
		}
	}

	// Filter to files meeting threshold
	var hotspots []Hotspot
	for file, count := range fileCounts {
		if count >= threshold {
			hotspots = append(hotspots, Hotspot{
				Path:           file,
				Type:           "fix-density",
				Score:          count,
				Details:        fmt.Sprintf("%d fix commits in last %d days", count, daysBack),
				Recommendation: generateFixRecommendation(file, count),
			})
		}
	}

	// Sort by count descending
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Score > hotspots[j].Score
	})

	return hotspots, totalFixes, nil
}

// matchesExclusionPattern checks if a file path matches an exclusion pattern.
// Supports exact matches and glob patterns like *.json, *.lock.
func matchesExclusionPattern(path, pattern string) bool {
	// Exact match
	if filepath.Base(path) == pattern || path == pattern {
		return true
	}

	// Glob pattern matching (e.g., *.json)
	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:] // Remove the *
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}

	return false
}

// containsSkippedDir returns true if any directory component in the path
// matches a directory name in skipBloatDirs. This handles nested build output
// directories (e.g., "web/.svelte-kit/output/foo.js") that prefix-based checks miss.
func containsSkippedDir(path string) bool {
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		if skipBloatDirs[filepath.Base(dir)] {
			return true
		}
		dir = filepath.Dir(dir)
	}
	return false
}

// shouldCountFileWithExclusions returns true if the file should be counted in hotspot analysis,
// considering the provided exclusion patterns.
func shouldCountFileWithExclusions(path string, exclusions []string) bool {
	// Check exclusion patterns first
	for _, pattern := range exclusions {
		if matchesExclusionPattern(path, pattern) {
			return false
		}
	}

	// Skip test files - they're expected to change with fixes
	if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".test.ts") || strings.HasSuffix(path, ".test.js") {
		return false
	}
	// Skip generated files and build output directories
	if strings.Contains(path, "/generated/") {
		return false
	}
	// Check if any path segment is a build output / tool directory
	if containsSkippedDir(path) {
		return false
	}
	// Check additional multi-segment prefixes
	for _, prefix := range additionalSkipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	// Skip documentation
	if strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".txt") {
		return false
	}
	// Skip config files (usually not architectural concerns)
	if filepath.Base(path) == "package.json" || filepath.Base(path) == "go.mod" {
		return false
	}
	return true
}

// shouldCountFile returns true if the file should be counted in hotspot analysis.
// Uses the global hotspotExclude patterns.
func shouldCountFile(path string) bool {
	return shouldCountFileWithExclusions(path, hotspotExclude)
}

// isSourceFile returns true if the file is a recognized source code file type.
func isSourceFile(path string) bool {
	ext := filepath.Ext(path)
	sourceExts := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".svelte",
		".py", ".rb", ".java", ".c", ".cpp", ".h", ".hpp",
		".rs", ".swift", ".kt", ".scala", ".sh", ".bash",
		".css", ".scss", ".sass", ".less", ".html", ".vue",
	}

	for _, sourceExt := range sourceExts {
		if ext == sourceExt {
			return true
		}
	}

	return false
}

// generateFixRecommendation creates a recommendation based on fix patterns.
func generateFixRecommendation(file string, count int) string {
	if count >= 10 {
		return fmt.Sprintf("CRITICAL: Consider spawning architect to redesign %s - excessive fix churn indicates structural issues", filepath.Base(file))
	}
	if count >= 7 {
		return fmt.Sprintf("HIGH: Spawn investigation to analyze root cause of recurring issues in %s", filepath.Base(file))
	}
	return fmt.Sprintf("MODERATE: Review %s for design improvements - shows fix accumulation", filepath.Base(file))
}

// analyzeInvestigationClusters scans .kb/investigations/ directly and clusters
// by meaningful keywords extracted from filenames, filtering out generic stop words.
func analyzeInvestigationClusters(projectDir string, threshold int) ([]Hotspot, int, error) {
	kbDir := filepath.Join(projectDir, ".kb", "investigations")
	entries, err := os.ReadDir(kbDir)
	if err != nil {
		// .kb/investigations/ might not exist
		return nil, 0, nil
	}

	// Extract keywords from each investigation file and map keyword → files
	keywordToFiles := make(map[string][]string)
	totalInv := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		totalInv++

		keywords := extractInvestigationKeywords(entry.Name())
		seen := make(map[string]bool) // deduplicate within single file
		for _, kw := range keywords {
			if !seen[kw] {
				keywordToFiles[kw] = append(keywordToFiles[kw], entry.Name())
				seen[kw] = true
			}
		}
	}

	// Build hotspots from keyword clusters meeting threshold
	var hotspots []Hotspot
	for keyword, files := range keywordToFiles {
		if len(files) >= threshold {
			hotspots = append(hotspots, Hotspot{
				Path:           keyword,
				Type:           "investigation-cluster",
				Score:          len(files),
				Details:        fmt.Sprintf("%d investigations on topic '%s'", len(files), keyword),
				RelatedFiles:   files,
				Recommendation: generateInvestigationRecommendation(keyword, len(files), ""),
			})
		}
	}

	// Sort by score descending for deterministic output
	sort.Slice(hotspots, func(i, j int) bool {
		if hotspots[i].Score != hotspots[j].Score {
			return hotspots[i].Score > hotspots[j].Score
		}
		return hotspots[i].Path < hotspots[j].Path
	})

	return hotspots, totalInv, nil
}

// extractInvestigationKeywords extracts meaningful topic keywords from an
// investigation filename by stripping date/type prefixes and filtering stop words.
// Example: "2026-02-19-design-coupling-hotspot-analysis-system.md" → ["coupling", "hotspot", "analysis", "system"]
func extractInvestigationKeywords(filename string) []string {
	// Strip .md extension
	name := strings.TrimSuffix(filename, ".md")

	// Strip date prefix (YYYY-MM-DD-)
	if len(name) > 11 && name[4] == '-' && name[7] == '-' && name[10] == '-' {
		name = name[11:]
	}

	// Strip type prefix (inv-, design-, audit-, spike-, synthesis-, debug-)
	typePrefixes := []string{"inv-", "design-", "audit-", "spike-", "synthesis-", "debug-"}
	for _, prefix := range typePrefixes {
		if strings.HasPrefix(name, prefix) {
			name = name[len(prefix):]
			break
		}
	}

	// Strip secondary type/priority prefixes (p0-, p1-, p2-)
	for _, prefix := range []string{"p0-", "p1-", "p2-", "p3-", "p4-"} {
		if strings.HasPrefix(name, prefix) {
			name = name[len(prefix):]
			break
		}
	}

	// Split on hyphens
	tokens := strings.Split(name, "-")

	// Filter stop words and empty tokens
	var keywords []string
	for _, token := range tokens {
		if token != "" && !isInvestigationStopWord(token) {
			keywords = append(keywords, token)
		}
	}

	return keywords
}

// isInvestigationStopWord returns true if a word is too generic to be a meaningful
// investigation topic keyword.
func isInvestigationStopWord(word string) bool {
	w := strings.ToLower(word)
	_, found := investigationStopWords[w]
	return found
}

// investigationStopWords contains words that are too generic for investigation clustering.
// These are common English words plus investigation-naming conventions that don't
// represent coherent topic areas.
var investigationStopWords = map[string]bool{
	// Common English
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
	"are": true, "be": true, "been": true, "being": true, "have": true, "has": true,
	"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
	"could": true, "should": true, "may": true, "might": true, "can": true,
	"not": true, "no": true, "all": true, "each": true, "every": true,
	"this": true, "that": true, "these": true, "those": true,
	"it": true, "its": true, "vs": true, "about": true, "before": true, "after": true,
	"how": true, "why": true, "what": true, "when": true, "where": true,
	"need": true, "needs": true,

	// Generic investigation/task verbs
	"add": true, "fix": true, "implement": true, "integrate": true,
	"investigate": true, "review": true, "check": true, "update": true,
	"create": true, "remove": true, "move": true, "change": true,
	"test": true, "verify": true, "validate": true, "ensure": true,
	"enhance": true, "improve": true, "refactor": true, "consider": true,
	"use": true, "using": true, "used": true, "scope": true,
	"document": true, "comprehensive": true, "design": true, "audit": true,

	// Generic descriptors
	"new": true, "old": true, "current": true, "existing": true,
	"phase": true, "step": true, "process": true, "approach": true,
	"into": true, "ready": true, "during": true,

	// Project-specific generics (appear in nearly all investigation filenames)
	"orch": true, "go": true,
}

// generateInvestigationRecommendation creates a recommendation based on investigation patterns.
func generateInvestigationRecommendation(topic string, count int, urgency string) string {
	if count >= 10 || urgency == "high" {
		return fmt.Sprintf("CRITICAL: Spawn architect session to synthesize %d investigations on '%s' - needs design decision", count, topic)
	}
	if count >= 5 {
		return fmt.Sprintf("HIGH: Consider design-session to consolidate understanding of '%s'", topic)
	}
	return fmt.Sprintf("MODERATE: '%s' has accumulated investigations - may need synthesis", topic)
}
