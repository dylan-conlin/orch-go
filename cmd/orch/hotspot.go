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
	hotspotFormatJSON    bool
	hotspotFixThreshold  int
	hotspotInvThreshold  int
	hotspotDaysBack      int
)

var hotspotCmd = &cobra.Command{
	Use:   "hotspot",
	Short: "Detect areas needing architect intervention",
	Long: `Analyze git history and investigation patterns to surface areas needing architect attention.

Detection signals:
  1. Fix commit density: Files with many "fix:" commits (high churn, recurring bugs)
  2. Investigation clustering: Topics with many investigations (unclear design)

A hotspot is flagged when:
  - A file has 5+ fix commits in the analysis period, OR
  - A topic has 3+ investigations in the knowledge base

Use --format json for machine-readable output that can be piped to other tools.

Examples:
  orch hotspot                       # Analyze with defaults (4 weeks, 5+ fixes, 3+ investigations)
  orch hotspot --days 14             # Analyze last 2 weeks only
  orch hotspot --threshold 3         # Flag files with 3+ fix commits
  orch hotspot --inv-threshold 5     # Flag topics with 5+ investigations
  orch hotspot --format json         # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHotspot()
	},
}

func init() {
	hotspotCmd.Flags().BoolVar(&hotspotFormatJSON, "json", false, "Output as JSON")
	hotspotCmd.Flags().IntVar(&hotspotFixThreshold, "threshold", 5, "Minimum fix commits to flag as hotspot")
	hotspotCmd.Flags().IntVar(&hotspotInvThreshold, "inv-threshold", 3, "Minimum investigations to flag as hotspot")
	hotspotCmd.Flags().IntVar(&hotspotDaysBack, "days", 28, "Days of git history to analyze")
	rootCmd.AddCommand(hotspotCmd)
}

// Hotspot represents a detected area needing architect attention.
type Hotspot struct {
	Path             string   `json:"path"`                        // File path or topic name
	Type             string   `json:"type"`                        // "fix-density" or "investigation-cluster"
	Score            int      `json:"score"`                       // Number of occurrences
	Details          string   `json:"details,omitempty"`           // Additional context
	RelatedFiles     []string `json:"related_files,omitempty"`     // Files affected (for investigation clusters)
	Recommendation   string   `json:"recommendation"`              // Suggested action
}

// HotspotReport is the complete analysis output.
type HotspotReport struct {
	GeneratedAt      string    `json:"generated_at"`
	AnalysisPeriod   string    `json:"analysis_period"`
	FixThreshold     int       `json:"fix_threshold"`
	InvThreshold     int       `json:"inv_threshold"`
	Hotspots         []Hotspot `json:"hotspots"`
	TotalFixCommits  int       `json:"total_fix_commits"`
	TotalInvestigations int   `json:"total_investigations"`
	HasArchitectWork bool      `json:"has_architect_work"`
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

// shouldCountFile returns true if the file should be counted in hotspot analysis.
func shouldCountFile(path string) bool {
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
			Topic       string   `json:"topic"`
			Count       int      `json:"count"`
			Files       []string `json:"files"`
			Urgency     string   `json:"urgency"`
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
