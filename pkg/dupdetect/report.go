// Package dupdetect provides AST-based function duplication detection.
// report.go integrates detection results with beads issue tracking.
package dupdetect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// ReportConfig controls how duplicates are reported to beads.
type ReportConfig struct {
	// Threshold overrides the detector's threshold for reporting purposes.
	// Pairs already filtered by the detector; this is informational.
	Threshold float64

	// DryRun prints what would be created without creating issues.
	DryRun bool

	// ProjectDir is the project root (used for constructing relative paths).
	ProjectDir string
}

// ReportResult summarizes what was reported to beads.
type ReportResult struct {
	Created  int      // Number of new issues created
	Skipped  int      // Number of pairs skipped (existing issue)
	IssueIDs []string // IDs of created issues
	Errors   []error  // Non-fatal errors encountered
}

// ReportToBeads creates beads issues for each duplicate pair.
// It uses title-based dedup (beads CreateArgs.Force=false) to avoid
// creating duplicate issues for the same pair.
func ReportToBeads(client beads.BeadsClient, pairs []DupPair, cfg ReportConfig) (*ReportResult, error) {
	result := &ReportResult{}

	for _, pair := range pairs {
		title := DupPairTitle(pair)
		desc := dupPairDescription(pair, cfg)

		issue, err := client.Create(&beads.CreateArgs{
			Title:     title,
			IssueType: "task",
			Priority:  3,
			Labels:    []string{"dupdetect", "triage:review"},
			Description: desc,
			Force:     false, // let beads dedup by title
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("create issue for %s <-> %s: %w", pair.FuncA.Name, pair.FuncB.Name, err))
			continue
		}

		if issue != nil {
			result.Created++
			result.IssueIDs = append(result.IssueIDs, issue.ID)
		}
	}

	return result, nil
}

// DupPairTitle generates a stable, dedup-friendly title for a duplicate pair.
// The title is deterministic: functions are sorted alphabetically so the same
// pair always produces the same title regardless of detection order.
func DupPairTitle(pair DupPair) string {
	a := pair.FuncA.Name
	b := pair.FuncB.Name
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("Extract shared logic: %s / %s (%.0f%% similar)", a, b, pair.Similarity*100)
}

// dupPairDescription generates a description for the beads issue.
func dupPairDescription(pair DupPair, cfg ReportConfig) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Duplicate function pair detected (%.0f%% structural similarity).\n\n", pair.Similarity*100))
	sb.WriteString(fmt.Sprintf("**Function A:** `%s` in `%s` (line %d, %d lines)\n", pair.FuncA.Name, pair.FuncA.File, pair.FuncA.StartLine, pair.FuncA.Lines))
	sb.WriteString(fmt.Sprintf("**Function B:** `%s` in `%s` (line %d, %d lines)\n", pair.FuncB.Name, pair.FuncB.File, pair.FuncB.StartLine, pair.FuncB.Lines))
	sb.WriteString("\n**Action:** Extract shared logic into a common function or package.\n")
	sb.WriteString("\n_Auto-created by dupdetect (Harness Layer 2)_\n")
	return sb.String()
}

// ScanProject walks all Go packages under projectDir and returns duplicate pairs.
func (d *Detector) ScanProject(projectDir string) ([]DupPair, error) {
	var allFuncs []FuncInfo

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		// Skip directories that shouldn't be scanned
		if info.IsDir() {
			base := info.Name()
			if base == ".git" || base == "vendor" || base == "node_modules" ||
				base == ".orch" || base == ".beads" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process non-test Go files
		name := info.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Use relative path for readable output
		relPath, relErr := filepath.Rel(projectDir, path)
		if relErr != nil {
			relPath = path
		}

		funcs, err := d.ParseSource(relPath, string(src))
		if err != nil {
			return nil // skip unparseable files
		}
		allFuncs = append(allFuncs, funcs...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk project: %w", err)
	}

	return d.FindDuplicates(allFuncs), nil
}
