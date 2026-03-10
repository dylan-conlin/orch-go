package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// skipBloatDirs are directory names excluded from bloat scanning (filepath.Walk).
// These contain build output, tool files, or vendored code that shouldn't be
// flagged as source code hotspots.
var skipBloatDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	// Build output directories
	".svelte-kit": true,
	"dist":        true,
	"build":       true,
	"__pycache__":  true,
	".next":       true,
	".nuxt":       true,
	".output":     true,
	// Tool/workspace directories (not source code)
	".opencode": true,
	".orch":     true,
	".beads":    true,
}

// additionalSkipPrefixes are multi-segment path prefixes that can't be expressed
// as single directory names. Used alongside containsSkippedDir for defense-in-depth
// filtering of git log paths.
var additionalSkipPrefixes = []string{
	"public/assets/",
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
			if skipBloatDirs[info.Name()] {
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

