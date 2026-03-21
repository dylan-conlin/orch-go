// Package decisions provides decision lifecycle management:
// enforcement type classification, staleness detection, and budget cap enforcement.
package decisions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// EnforcementType is the mechanism by which a decision is enforced.
type EnforcementType string

const (
	EnforcementGate        EnforcementType = "gate"
	EnforcementHook        EnforcementType = "hook"
	EnforcementConvention  EnforcementType = "convention"
	EnforcementContextOnly EnforcementType = "context-only"
	EnforcementUnknown     EnforcementType = ""
)

// BudgetCap is the target maximum number of active decisions.
const BudgetCap = 30

// StaleThresholdDays is how many days a context-only decision can go uncited before being flagged stale.
const StaleThresholdDays = 30

// Decision represents a parsed decision file with lifecycle metadata.
type Decision struct {
	Path        string          // Full file path
	Name        string          // Filename without extension
	Date        time.Time       // Parsed from filename YYYY-MM-DD prefix
	Status      string          // Accepted, Proposed, Superseded, etc.
	Enforcement EnforcementType // gate, hook, convention, context-only
	Title       string          // From first heading
	CitedBy     int             // Number of files that reference this decision
}

// regexes for parsing decision files
var (
	regexStatus      = regexp.MustCompile(`(?m)^\*\*Status:\*\*\s*(.+)$`)
	regexEnforcement = regexp.MustCompile(`(?m)^\*\*Enforcement:\*\*\s*(.+)$`)
	regexTitle       = regexp.MustCompile(`(?m)^#\s+(?:Decision:\s*)?(.+)$`)
	regexDatePrefix  = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-`)
)

// ListActiveDecisions lists all non-archived decision files in .kb/decisions/.
func ListActiveDecisions(projectDir string) ([]Decision, error) {
	decDir := filepath.Join(projectDir, ".kb", "decisions")
	entries, err := os.ReadDir(decDir)
	if err != nil {
		return nil, fmt.Errorf("reading decisions dir: %w", err)
	}

	var decisions []Decision
	for _, e := range entries {
		if e.IsDir() {
			continue // skip archived/ subdirectory
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		d := parseDecisionFile(filepath.Join(decDir, e.Name()))
		decisions = append(decisions, d)
	}

	sort.Slice(decisions, func(i, j int) bool {
		return decisions[i].Date.After(decisions[j].Date)
	})
	return decisions, nil
}

// parseDecisionFile extracts metadata from a decision file.
func parseDecisionFile(path string) Decision {
	d := Decision{
		Path: path,
		Name: strings.TrimSuffix(filepath.Base(path), ".md"),
	}

	// Parse date from filename
	if matches := regexDatePrefix.FindStringSubmatch(filepath.Base(path)); len(matches) >= 2 {
		if t, err := time.Parse("2006-01-02", matches[1]); err == nil {
			d.Date = t
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return d
	}
	content := string(data)

	// Parse status
	if matches := regexStatus.FindStringSubmatch(content); len(matches) >= 2 {
		d.Status = strings.TrimSpace(matches[1])
	}

	// Parse enforcement
	if matches := regexEnforcement.FindStringSubmatch(content); len(matches) >= 2 {
		d.Enforcement = EnforcementType(strings.ToLower(strings.TrimSpace(matches[1])))
	}

	// Parse title
	if matches := regexTitle.FindStringSubmatch(content); len(matches) >= 2 {
		d.Title = strings.TrimSpace(matches[1])
	}

	return d
}

// CountCitations counts how many files in the project reference a decision filename.
// Uses simple filename grep across .kb/, skills/, and CLAUDE.md.
func CountCitations(decisionName, projectDir string) int {
	count := 0
	searchPaths := []string{
		filepath.Join(projectDir, ".kb"),
		filepath.Join(projectDir, "skills"),
		filepath.Join(projectDir, "CLAUDE.md"),
	}

	for _, searchPath := range searchPaths {
		info, err := os.Stat(searchPath)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			// Single file
			data, err := os.ReadFile(searchPath)
			if err == nil && strings.Contains(string(data), decisionName) {
				count++
			}
			continue
		}
		filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".md") {
				return nil
			}
			// Don't count self-citation
			if filepath.Base(path) == decisionName+".md" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if strings.Contains(string(data), decisionName) {
				count++
			}
			return nil
		})
	}
	return count
}

// StaleResult contains the results of a staleness analysis.
type StaleResult struct {
	Stale   []Decision // Context-only decisions past threshold with 0 citations
	Active  int        // Total active decisions
	Budget  int        // Budget cap
	OverBy  int        // How many over budget (0 if under)
}

// FindStale identifies context-only decisions that are past StaleThresholdDays
// with zero citations, and calculates budget status.
func FindStale(projectDir string) (*StaleResult, error) {
	decisions, err := ListActiveDecisions(projectDir)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result := &StaleResult{
		Active: len(decisions),
		Budget: BudgetCap,
	}

	if result.Active > BudgetCap {
		result.OverBy = result.Active - BudgetCap
	}

	for i := range decisions {
		d := &decisions[i]
		// Count citations for all decisions
		d.CitedBy = CountCitations(d.Name, projectDir)

		// Stale = context-only + >30d old + 0 citations
		if d.Enforcement == EnforcementContextOnly &&
			!d.Date.IsZero() &&
			now.Sub(d.Date).Hours()/24 > float64(StaleThresholdDays) &&
			d.CitedBy == 0 {
			result.Stale = append(result.Stale, *d)
		}
	}

	return result, nil
}

// BudgetStatus returns a summary of decision budget usage.
type BudgetStatus struct {
	Active      int
	Cap         int
	OverBy      int
	ByType      map[EnforcementType]int
	Unclassified int // Decisions without enforcement field
}

// CheckBudget analyzes the current decision budget.
func CheckBudget(projectDir string) (*BudgetStatus, error) {
	decisions, err := ListActiveDecisions(projectDir)
	if err != nil {
		return nil, err
	}

	status := &BudgetStatus{
		Active: len(decisions),
		Cap:    BudgetCap,
		ByType: make(map[EnforcementType]int),
	}

	if status.Active > BudgetCap {
		status.OverBy = status.Active - BudgetCap
	}

	for _, d := range decisions {
		if d.Enforcement == EnforcementUnknown {
			status.Unclassified++
		} else {
			status.ByType[d.Enforcement]++
		}
	}

	return status, nil
}
