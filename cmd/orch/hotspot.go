package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	hotspotFormatJSON     bool
	hotspotFixThreshold   int
	hotspotInvThreshold   int
	hotspotBloatThreshold int
	hotspotDaysBack       int
	hotspotExclude        []string
)

// defaultExclusions are file patterns excluded by default from hotspot analysis.
// These are data/config files where fix commits are expected, not code hotspots.
var defaultExclusions = []string{
	"*.jsonl",
	"*.json",
	"*.lock",
	"go.sum",
}

var hotspotCmd = &cobra.Command{
	Use:   "hotspot",
	Short: "Detect areas needing architect intervention",
	Long: `Analyze git history and investigation patterns to surface areas needing architect attention.

Detection signals:
  1. Fix commit density: Files with many "fix:" commits (high churn, recurring bugs)
  2. Investigation clustering: Topics with many investigations (unclear design)
  3. Bloat size: Files exceeding line count threshold (coherence degradation)

A hotspot is flagged when:
  - A file has 5+ fix commits in the analysis period, OR
  - A topic has 3+ investigations in the knowledge base, OR
  - A file exceeds 800 lines (bloat threshold)

Use --format json for machine-readable output that can be piped to other tools.

Examples:
  orch hotspot                       # Analyze with defaults (4 weeks, 5+ fixes, 3+ investigations, 800+ lines)
  orch hotspot --days 14             # Analyze last 2 weeks only
  orch hotspot --threshold 3         # Flag files with 3+ fix commits
  orch hotspot --inv-threshold 5     # Flag topics with 5+ investigations
  orch hotspot --bloat-threshold 1000 # Flag files with 1000+ lines
  orch hotspot --format json         # Output as JSON for scripting
  orch hotspot --exclude ""          # Include all files (disable default exclusions)
  orch hotspot --exclude "*.yaml"    # Exclude only YAML files`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHotspot()
	},
}

func init() {
	hotspotCmd.Flags().BoolVar(&hotspotFormatJSON, "json", false, "Output as JSON")
	hotspotCmd.Flags().IntVar(&hotspotFixThreshold, "threshold", 5, "Minimum fix commits to flag as hotspot")
	hotspotCmd.Flags().IntVar(&hotspotInvThreshold, "inv-threshold", 3, "Minimum investigations to flag as hotspot")
	hotspotCmd.Flags().IntVar(&hotspotBloatThreshold, "bloat-threshold", 800, "Minimum lines to flag file as bloated")
	hotspotCmd.Flags().IntVar(&hotspotDaysBack, "days", 28, "Days of git history to analyze")
	hotspotCmd.Flags().StringSliceVar(&hotspotExclude, "exclude", defaultExclusions, "File patterns to exclude (e.g., *.json, go.sum)")
	rootCmd.AddCommand(hotspotCmd)
}

// Hotspot represents a detected area needing architect attention.
type Hotspot struct {
	Path           string   `json:"path"`                    // File path or topic name
	Type           string   `json:"type"`                    // "fix-density" or "investigation-cluster"
	Score          int      `json:"score"`                   // Number of occurrences
	Details        string   `json:"details,omitempty"`       // Additional context
	RelatedFiles   []string `json:"related_files,omitempty"` // Files affected (for investigation clusters)
	Recommendation string   `json:"recommendation"`          // Suggested action
}

// HotspotReport is the complete analysis output.
type HotspotReport struct {
	GeneratedAt           string    `json:"generated_at"`
	AnalysisPeriod        string    `json:"analysis_period"`
	FixThreshold          int       `json:"fix_threshold"`
	InvThreshold          int       `json:"inv_threshold"`
	BloatThreshold        int       `json:"bloat_threshold"`
	Hotspots              []Hotspot `json:"hotspots"`
	TotalFixCommits       int       `json:"total_fix_commits"`
	TotalInvestigations   int       `json:"total_investigations"`
	TotalBloatedFiles     int       `json:"total_bloated_files"`
	TotalCouplingClusters int       `json:"total_coupling_clusters"`
	HasArchitectWork      bool      `json:"has_architect_work"`
}

func runHotspot() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	report := HotspotReport{
		GeneratedAt:    time.Now().Format(time.RFC3339),
		AnalysisPeriod: fmt.Sprintf("Last %d days", hotspotDaysBack),
		FixThreshold:   hotspotFixThreshold,
		InvThreshold:   hotspotInvThreshold,
		BloatThreshold: hotspotBloatThreshold,
		Hotspots:       []Hotspot{},
	}

	// 1. Analyze git log for fix commit density
	fixHotspots, totalFixes, err := analyzeFixCommits(projectDir, hotspotDaysBack, hotspotFixThreshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to analyze git history: %v\n", err)
	} else {
		report.TotalFixCommits = totalFixes
		report.Hotspots = append(report.Hotspots, fixHotspots...)
	}

	// 2. Query kb reflect for investigation clustering
	invHotspots, totalInv, err := analyzeInvestigationClusters(projectDir, hotspotInvThreshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to analyze investigations: %v\n", err)
	} else {
		report.TotalInvestigations = totalInv
		report.Hotspots = append(report.Hotspots, invHotspots...)
	}

	// 3. Analyze file sizes for bloat detection
	bloatHotspots, totalBloat, err := analyzeBloatFiles(projectDir, hotspotBloatThreshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to analyze file sizes: %v\n", err)
	} else {
		report.TotalBloatedFiles = totalBloat
		report.Hotspots = append(report.Hotspots, bloatHotspots...)
	}

	// 4. Analyze cross-layer coupling clusters
	couplingHotspots, totalCoupling, err := analyzeCouplingClusters(projectDir, hotspotDaysBack)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to analyze coupling clusters: %v\n", err)
	} else {
		report.TotalCouplingClusters = totalCoupling
		report.Hotspots = append(report.Hotspots, couplingHotspots...)
	}

	// Sort hotspots by score (descending)
	sort.Slice(report.Hotspots, func(i, j int) bool {
		return report.Hotspots[i].Score > report.Hotspots[j].Score
	})

	report.HasArchitectWork = len(report.Hotspots) > 0

	// Output
	if hotspotFormatJSON {
		return outputJSON(report)
	}
	return outputText(report)
}

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
	// Skip generated files
	if strings.Contains(path, "/generated/") || strings.HasPrefix(path, "vendor/") {
		return false
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

// analyzeInvestigationClusters uses kb reflect to find investigation clusters.
func analyzeInvestigationClusters(projectDir string, threshold int) ([]Hotspot, int, error) {
	// Run kb reflect --type synthesis --format json
	cmd := exec.Command("kb", "reflect", "--type", "synthesis", "--format", "json")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// kb reflect might not be installed or might fail
		// This is not a fatal error - just skip this analysis
		return nil, 0, nil
	}

	// Parse JSON output
	var reflectResult struct {
		Synthesis []struct {
			Topic   string   `json:"topic"`
			Count   int      `json:"count"`
			Files   []string `json:"files"`
			Urgency string   `json:"urgency"`
		} `json:"synthesis"`
	}

	if err := json.Unmarshal(output, &reflectResult); err != nil {
		// Try parsing as array directly (different kb versions)
		var synthesisList []struct {
			Topic   string   `json:"topic"`
			Count   int      `json:"count"`
			Files   []string `json:"files"`
			Urgency string   `json:"urgency"`
		}
		if err := json.Unmarshal(output, &synthesisList); err != nil {
			return nil, 0, nil // Silent failure - kb output format might differ
		}
		reflectResult.Synthesis = synthesisList
	}

	var hotspots []Hotspot
	totalInv := 0

	for _, s := range reflectResult.Synthesis {
		totalInv += s.Count

		if s.Count >= threshold {
			hotspots = append(hotspots, Hotspot{
				Path:           s.Topic,
				Type:           "investigation-cluster",
				Score:          s.Count,
				Details:        fmt.Sprintf("%d investigations on topic '%s'", s.Count, s.Topic),
				RelatedFiles:   s.Files,
				Recommendation: generateInvestigationRecommendation(s.Topic, s.Count, s.Urgency),
			})
		}
	}

	return hotspots, totalInv, nil
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

// analyzeBloatFiles scans source files and flags files exceeding the bloat threshold.
func analyzeBloatFiles(projectDir string, threshold int) ([]Hotspot, int, error) {
	var hotspots []Hotspot
	totalBloat := 0

	// Walk the project directory to count lines in source files
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip common non-source directories
			if info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path from project directory
		relPath, err := filepath.Rel(projectDir, path)
		if err != nil {
			return nil // Skip files we can't get relative path for
		}

		// Only count files that pass the exclusion filter
		if !shouldCountFile(relPath) {
			return nil
		}

		// Only count recognized source file types for bloat detection
		if !isSourceFile(relPath) {
			return nil
		}

		// Count lines in the file
		lineCount, err := countLines(path)
		if err != nil {
			return nil // Skip files we can't count (permissions, etc.)
		}

		// Check if file exceeds threshold
		if lineCount >= threshold {
			totalBloat++
			hotspots = append(hotspots, Hotspot{
				Path:           relPath,
				Type:           "bloat-size",
				Score:          lineCount,
				Details:        fmt.Sprintf("%d lines (threshold: %d)", lineCount, threshold),
				Recommendation: generateBloatRecommendation(relPath, lineCount),
			})
		}

		return nil
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort by line count descending
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Score > hotspots[j].Score
	})

	return hotspots, totalBloat, nil
}

// countLines counts the number of lines in a file.
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	buf := make([]byte, 32*1024) // 32KB buffer
	lineSep := []byte{'\n'}

	for {
		c, err := file.Read(buf)
		count += strings.Count(string(buf[:c]), string(lineSep))

		if err != nil {
			if err == os.ErrClosed || strings.Contains(err.Error(), "EOF") {
				break
			}
			return 0, err
		}
	}

	return count, nil
}

// generateBloatRecommendation creates a recommendation based on file size.
func generateBloatRecommendation(file string, lines int) string {
	if lines > 1500 {
		return fmt.Sprintf("CRITICAL: %s (%d lines) - Recommend architect session for structural redesign", filepath.Base(file), lines)
	}
	if lines >= 800 {
		return fmt.Sprintf("MODERATE: %s (%d lines) - See .kb/guides/code-extraction-patterns.md for extraction workflow", filepath.Base(file), lines)
	}
	return fmt.Sprintf("INFO: %s (%d lines)", filepath.Base(file), lines)
}

// outputJSON prints the report as JSON.
func outputJSON(report HotspotReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// outputText prints the report in human-readable format.
func outputText(report HotspotReport) error {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  🔥 HOTSPOT ANALYSIS                                                         ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Period: %-66s ║\n", report.AnalysisPeriod)
	fmt.Printf("║  Fix Commits Analyzed: %-53d ║\n", report.TotalFixCommits)
	fmt.Printf("║  Investigations Analyzed: %-50d ║\n", report.TotalInvestigations)
	fmt.Printf("║  Bloated Files (>%d lines): %-48d ║\n", report.BloatThreshold, report.TotalBloatedFiles)
	fmt.Printf("║  Coupling Clusters: %-56d ║\n", report.TotalCouplingClusters)
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	if len(report.Hotspots) == 0 {
		fmt.Println("║  ✓ No hotspots detected - codebase appears healthy                          ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
		return nil
	}

	fmt.Printf("║  🚨 %d HOTSPOT(S) DETECTED                                                    ║\n", len(report.Hotspots))
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	for i, h := range report.Hotspots {
		if i >= 10 {
			fmt.Printf("║  ... and %d more (use --format json for full list)                          ║\n", len(report.Hotspots)-10)
			break
		}

		// Type indicator
		typeIcon := "🔧"
		if h.Type == "investigation-cluster" {
			typeIcon = "📚"
		} else if h.Type == "bloat-size" {
			typeIcon = "📏"
		} else if h.Type == "coupling-cluster" {
			typeIcon = "🔗"
		}

		// Truncate path for display
		displayPath := h.Path
		if len(displayPath) > 50 {
			displayPath = "..." + displayPath[len(displayPath)-47:]
		}

		fmt.Printf("║  %s [%2d] %-50s            ║\n", typeIcon, h.Score, displayPath)

		// Format recommendation (may need wrapping)
		rec := h.Recommendation
		if len(rec) > 72 {
			fmt.Printf("║      %s ║\n", rec[:72])
			remaining := rec[72:]
			for len(remaining) > 72 {
				fmt.Printf("║      %s ║\n", remaining[:72])
				remaining = remaining[72:]
			}
			if len(remaining) > 0 {
				fmt.Printf("║      %-72s ║\n", remaining)
			}
		} else {
			fmt.Printf("║      %-72s ║\n", rec)
		}
	}

	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  RECOMMENDED ACTIONS:                                                        ║")
	fmt.Println("║    1. Review hotspots above for architectural patterns                       ║")
	fmt.Println("║    2. Spawn architect session for critical items:                            ║")
	fmt.Println("║       orch spawn architect \"Review hotspots from orch hotspot\"               ║")
	fmt.Println("║    3. Consider design-session for recurring investigation topics             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	return nil
}

// Helper for output formatting - converts int to string with padding
func intToStr(n int) string {
	return strconv.Itoa(n)
}

// SpawnHotspotResult contains the result of checking hotspots for a spawn task.
type SpawnHotspotResult struct {
	HasHotspots        bool      `json:"has_hotspots"`
	HasCriticalHotspot bool      `json:"has_critical_hotspot"` // True when any matched bloat-size file >1500 lines
	MatchedHotspots    []Hotspot `json:"matched_hotspots,omitempty"`
	CriticalFiles      []string  `json:"critical_files,omitempty"` // File paths of CRITICAL hotspots (>1500 lines)
	MaxScore           int       `json:"max_score"`
	Warning            string    `json:"warning,omitempty"`
}

// extractPathsFromTask extracts file/directory paths from a task description.
// Returns a list of paths found in the task text.
func extractPathsFromTask(task string) []string {
	var paths []string

	// Pattern matches file paths like:
	// - cmd/orch/spawn.go
	// - pkg/daemon/daemon.go
	// - web/src/components/Dashboard.tsx
	// - "pkg/auth/token.go" (quoted)
	// - pkg/daemon/ (directories)
	pathPattern := regexp.MustCompile(`(?:^|[\s"'])([a-zA-Z0-9_\-./]+(?:\.[a-zA-Z0-9]+|/))(?:[\s"']|$)`)

	matches := pathPattern.FindAllStringSubmatch(task, -1)
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			path := strings.Trim(match[1], `"'`)
			// Validate it looks like a real path (has at least one directory separator or extension)
			if (strings.Contains(path, "/") || strings.Contains(path, ".")) && !seen[path] {
				// Filter out common non-path patterns
				if !isLikelyNotAPath(path) {
					paths = append(paths, path)
					seen[path] = true
				}
			}
		}
	}

	return paths
}

// isLikelyNotAPath returns true if the string is unlikely to be a file path.
func isLikelyNotAPath(s string) bool {
	// URLs
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return true
	}
	// Very short paths are probably not real
	if len(s) < 4 {
		return true
	}
	// Common words that might match the pattern but aren't paths
	nonPaths := []string{"e.g.", "i.e.", "etc.", "vs.", "no."}
	for _, np := range nonPaths {
		if s == np {
			return true
		}
	}
	return false
}

// matchPathToHotspots checks if a path matches any hotspot.
// Returns true and the highest matching score if a match is found.
func matchPathToHotspots(path string, hotspots []Hotspot) (bool, int) {
	maxScore := 0
	matched := false

	for _, h := range hotspots {
		switch h.Type {
		case "fix-density":
			// For fix-density, check for:
			// 1. Exact file match
			// 2. Path is a directory that contains the hotspot file
			// 3. Hotspot is a directory that the path is in
			if path == h.Path {
				// Exact match
				matched = true
			} else if strings.HasSuffix(path, "/") && strings.HasPrefix(h.Path, path) {
				// Path is a directory containing the hotspot
				matched = true
			} else if strings.HasSuffix(h.Path, "/") && strings.HasPrefix(path, h.Path) {
				// Hotspot is a directory containing the path
				matched = true
			}
			if matched && h.Score > maxScore {
				maxScore = h.Score
			}
		case "investigation-cluster":
			// For investigation clusters, check if the topic appears in the path
			if strings.Contains(strings.ToLower(path), strings.ToLower(h.Path)) {
				matched = true
				if h.Score > maxScore {
					maxScore = h.Score
				}
			}
		case "bloat-size":
			// For bloat-size, check for exact file match or directory containment
			if path == h.Path {
				// Exact match
				matched = true
			} else if strings.HasSuffix(path, "/") && strings.HasPrefix(h.Path, path) {
				// Path is a directory containing the hotspot file
				matched = true
			} else if strings.HasSuffix(h.Path, "/") && strings.HasPrefix(path, h.Path) {
				// Hotspot is a directory containing the path
				matched = true
			}
			if matched && h.Score > maxScore {
				maxScore = h.Score
			}
		case "coupling-cluster":
			// For coupling clusters, check if the concept appears in the path
			// or if the path matches any of the related files
			if strings.Contains(strings.ToLower(path), strings.ToLower(h.Path)) {
				matched = true
			}
			// Also check related files for exact or directory matches
			for _, rf := range h.RelatedFiles {
				if path == rf || (strings.HasSuffix(path, "/") && strings.HasPrefix(rf, path)) {
					matched = true
					break
				}
			}
			if matched && h.Score > maxScore {
				maxScore = h.Score
			}
		}
	}

	return matched, maxScore
}

// checkSpawnHotspots checks if a task description references any hotspot areas.
// Returns a SpawnHotspotResult with details about matched hotspots.
func checkSpawnHotspots(task string, hotspots []Hotspot) *SpawnHotspotResult {
	result := &SpawnHotspotResult{}

	// Extract paths from task
	paths := extractPathsFromTask(task)

	// Also check for investigation-cluster topic matches directly in task text
	taskLower := strings.ToLower(task)

	// Check each hotspot
	for _, h := range hotspots {
		matched := false

		// Check extracted paths against this hotspot
		for _, path := range paths {
			if pathMatches, _ := matchPathToHotspots(path, []Hotspot{h}); pathMatches {
				matched = true
				break
			}
		}

		// For investigation clusters and coupling clusters, also check if topic appears in task text
		if !matched && (h.Type == "investigation-cluster" || h.Type == "coupling-cluster") {
			if strings.Contains(taskLower, strings.ToLower(h.Path)) {
				matched = true
			}
		}

		if matched {
			result.HasHotspots = true
			result.MatchedHotspots = append(result.MatchedHotspots, h)
			if h.Score > result.MaxScore {
				result.MaxScore = h.Score
			}
			// Track CRITICAL hotspots: bloat-size files >1500 lines
			if h.Type == "bloat-size" && h.Score > 1500 {
				result.HasCriticalHotspot = true
				result.CriticalFiles = append(result.CriticalFiles, h.Path)
			}
		}
	}

	if result.HasHotspots {
		result.Warning = formatHotspotWarning(result)
	}

	return result
}

// formatHotspotWarning formats a warning message for hotspot matches.
func formatHotspotWarning(result *SpawnHotspotResult) string {
	if !result.HasHotspots || len(result.MatchedHotspots) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("┌─────────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  🔥 HOTSPOT WARNING: Task targets high-churn area                          │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────────────────────┤\n")

	for _, h := range result.MatchedHotspots {
		typeIcon := "🔧"
		if h.Type == "investigation-cluster" {
			typeIcon = "📚"
		} else if h.Type == "bloat-size" {
			typeIcon = "📏"
		} else if h.Type == "coupling-cluster" {
			typeIcon = "🔗"
		}
		line := fmt.Sprintf("│  %s [%d] %s", typeIcon, h.Score, h.Path)
		// Pad to box width
		if len(line) < 78 {
			line += strings.Repeat(" ", 78-len(line))
		}
		sb.WriteString(line + "│\n")
	}

	sb.WriteString("├─────────────────────────────────────────────────────────────────────────────┤\n")
	sb.WriteString("│  💡 RECOMMENDATION: Consider spawning architect first to review design     │\n")
	sb.WriteString("│     orch spawn architect \"Review design for [area]\"                        │\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}

// RunHotspotCheckForSpawn runs hotspot analysis and checks task against results.
// This is the main entry point for spawn integration.
// Returns nil if no hotspots detected, otherwise returns the result with warning.
func RunHotspotCheckForSpawn(projectDir, task string) (*SpawnHotspotResult, error) {
	// Run hotspot analysis (reuse existing logic)
	report := HotspotReport{
		GeneratedAt:    time.Now().Format(time.RFC3339),
		AnalysisPeriod: fmt.Sprintf("Last %d days", 28), // Default to 28 days
		FixThreshold:   5,                               // Default threshold
		InvThreshold:   3,                               // Default threshold
		BloatThreshold: 800,                             // Default bloat threshold
		Hotspots:       []Hotspot{},
	}

	// Analyze git history for fix commit density
	fixHotspots, _, err := analyzeFixCommits(projectDir, 28, 5)
	if err == nil {
		report.Hotspots = append(report.Hotspots, fixHotspots...)
	}

	// Analyze investigation clusters (silent failure if kb not available)
	invHotspots, _, _ := analyzeInvestigationClusters(projectDir, 3)
	report.Hotspots = append(report.Hotspots, invHotspots...)

	// Analyze coupling clusters
	couplingHotspots, _, _ := analyzeCouplingClusters(projectDir, 28)
	report.Hotspots = append(report.Hotspots, couplingHotspots...)

	// Analyze file sizes for bloat detection (CRITICAL files >1500 lines trigger spawn blocking)
	bloatHotspots, _, _ := analyzeBloatFiles(projectDir, 800)
	report.Hotspots = append(report.Hotspots, bloatHotspots...)

	if len(report.Hotspots) == 0 {
		return nil, nil
	}

	// Check task against hotspots
	result := checkSpawnHotspots(task, report.Hotspots)

	if !result.HasHotspots {
		return nil, nil
	}

	return result, nil
}
