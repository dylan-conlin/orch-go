package orient

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// PlanSummary holds metadata for an active coordination plan.
type PlanSummary struct {
	Name     string       `json:"name"`
	Title    string       `json:"title"`
	Status   string       `json:"status"`
	TLDR     string       `json:"tldr,omitempty"`
	Projects []string     `json:"projects,omitempty"`
	Phases   []PlanPhase  `json:"phases,omitempty"`
}

// PlanPhase holds status for a single phase within a plan.
type PlanPhase struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ScanActivePlans reads .kb/plans/ and returns summaries for plans with status: active.
func ScanActivePlans(plansDir string) ([]PlanSummary, error) {
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return nil, err
	}

	var plans []PlanSummary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(plansDir, entry.Name())
		plan, err := parsePlanFile(path)
		if err != nil {
			continue
		}

		if plan.Status == "active" {
			plan.Name = strings.TrimSuffix(entry.Name(), ".md")
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

// parsePlanFile extracts metadata from a plan markdown file.
func parsePlanFile(path string) (PlanSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return PlanSummary{}, err
	}
	defer file.Close()

	var plan PlanSummary
	scanner := bufio.NewScanner(file)
	inTLDR := false
	inPhases := false
	var tldrLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Parse title from H1 header
		if strings.HasPrefix(line, "# Coordination Plan:") {
			plan.Title = strings.TrimSpace(strings.TrimPrefix(line, "# Coordination Plan:"))
			continue
		}

		// Parse frontmatter-style fields (only outside sections)
		if strings.HasPrefix(line, "**Status:**") && !inPhases && !inTLDR {
			plan.Status = strings.TrimSpace(strings.TrimPrefix(line, "**Status:**"))
			continue
		}
		if strings.HasPrefix(line, "**Projects:**") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "**Projects:**"))
			for _, p := range strings.Split(raw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					plan.Projects = append(plan.Projects, p)
				}
			}
			continue
		}

		// Parse TLDR section
		if strings.HasPrefix(line, "## TLDR") {
			inTLDR = true
			inPhases = false
			continue
		}

		// Parse Phases section
		if strings.HasPrefix(line, "## Phases") {
			inPhases = true
			inTLDR = false
			continue
		}

		// Any other H2 ends both sections
		if strings.HasPrefix(line, "## ") {
			inTLDR = false
			inPhases = false
			continue
		}

		if inTLDR {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				tldrLines = append(tldrLines, trimmed)
			}
			continue
		}

		if inPhases {
			// Parse phase headers like: ### Phase 1: Setup
			if strings.HasPrefix(line, "### Phase") || strings.HasPrefix(line, "### ") {
				phaseName := strings.TrimPrefix(line, "### ")
				plan.Phases = append(plan.Phases, PlanPhase{Name: phaseName})
				continue
			}
			// Parse phase status like: **Status:** in-progress
			if strings.HasPrefix(line, "**Status:**") && len(plan.Phases) > 0 {
				plan.Phases[len(plan.Phases)-1].Status = strings.TrimSpace(strings.TrimPrefix(line, "**Status:**"))
			}
		}
	}

	plan.TLDR = strings.Join(tldrLines, " ")

	return plan, scanner.Err()
}
