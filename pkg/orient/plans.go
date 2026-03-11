package orient

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PlanSummary holds metadata for an active coordination plan.
type PlanSummary struct {
	Name     string      `json:"name"`
	Title    string      `json:"title"`
	Status   string      `json:"status"`
	TLDR     string      `json:"tldr,omitempty"`
	Projects []string    `json:"projects,omitempty"`
	Phases   []PlanPhase `json:"phases,omitempty"`
	Progress string      `json:"progress,omitempty"` // e.g. "1/4 complete" — set by ApplyBeadsProgress
}

// PlanPhase holds status for a single phase within a plan.
type PlanPhase struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	BeadsIDs []string `json:"beads_ids,omitempty"`
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
			// Parse phase beads like: **Beads:** orch-go-abc1, orch-go-def2
			if strings.HasPrefix(line, "**Beads:**") && len(plan.Phases) > 0 {
				plan.Phases[len(plan.Phases)-1].BeadsIDs = parseBeadsIDs(strings.TrimPrefix(line, "**Beads:**"))
			}
		}
	}

	plan.TLDR = strings.Join(tldrLines, " ")

	return plan, scanner.Err()
}

// parseBeadsIDs extracts comma-separated beads IDs from a value string.
func parseBeadsIDs(val string) []string {
	val = strings.TrimSpace(val)
	if val == "" || val == "none" {
		return nil
	}
	var ids []string
	for _, id := range strings.Split(val, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// CollectPlanBeadsIDs gathers all beads IDs from all phases of all plans.
func CollectPlanBeadsIDs(plans []PlanSummary) []string {
	var ids []string
	for _, plan := range plans {
		for _, phase := range plan.Phases {
			ids = append(ids, phase.BeadsIDs...)
		}
	}
	return ids
}

// ApplyBeadsProgress updates plan phase statuses and progress summaries
// based on beads issue statuses. statusMap maps beads ID -> status string
// (e.g. "closed", "in_progress", "open").
func ApplyBeadsProgress(plans []PlanSummary, statusMap map[string]string) {
	if len(statusMap) == 0 {
		return
	}

	for i := range plans {
		hydrated := 0
		complete := 0

		for j := range plans[i].Phases {
			phase := &plans[i].Phases[j]
			if len(phase.BeadsIDs) == 0 {
				continue
			}
			hydrated++

			allClosed := true
			anyInProgress := false

			for _, id := range phase.BeadsIDs {
				status := statusMap[id]
				switch status {
				case "closed":
					// ok
				case "in_progress":
					allClosed = false
					anyInProgress = true
				default:
					allClosed = false
				}
			}

			if allClosed {
				phase.Status = "complete"
				complete++
			} else if anyInProgress {
				phase.Status = "in-progress"
			} else {
				phase.Status = "ready"
			}
		}

		if hydrated > 0 {
			plans[i].Progress = fmt.Sprintf("%d/%d complete", complete, hydrated)
		}
	}
}
