// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// KnowledgeHealthResult contains the result of a knowledge health check.
type KnowledgeHealthResult struct {
	// TotalActive is the total number of active quick entries.
	TotalActive int

	// ByType maps entry type (decision, constraint, attempt, question) to count.
	ByType map[string]int

	// ThresholdExceeded is true when TotalActive exceeds the configured threshold.
	ThresholdExceeded bool

	// Error is set if the health check failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// KnowledgeHealthSnapshot is a point-in-time snapshot for the daemon status file.
type KnowledgeHealthSnapshot struct {
	TotalActive       int       `json:"total_active"`
	Decisions         int       `json:"decisions"`
	Constraints       int       `json:"constraints"`
	Attempts          int       `json:"attempts"`
	Questions         int       `json:"questions"`
	ThresholdExceeded bool      `json:"threshold_exceeded"`
	LastCheck         time.Time `json:"last_check"`
}

// Snapshot converts a KnowledgeHealthResult to a dashboard-ready snapshot.
func (r *KnowledgeHealthResult) Snapshot() KnowledgeHealthSnapshot {
	return KnowledgeHealthSnapshot{
		TotalActive:       r.TotalActive,
		Decisions:         r.ByType["decision"],
		Constraints:       r.ByType["constraint"],
		Attempts:          r.ByType["attempt"],
		Questions:         r.ByType["question"],
		ThresholdExceeded: r.ThresholdExceeded,
		LastCheck:         time.Now(),
	}
}

// quickEntry represents a single entry from kb quick list --json.
type quickEntry struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// DefaultKnowledgeHealthCheck shells out to kb quick list --json and counts active entries.
func DefaultKnowledgeHealthCheck() (*KnowledgeHealthResult, error) {
	cmd := exec.Command("kb", "quick", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run kb quick list: %w", err)
	}

	var entries []quickEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse kb quick list output: %w", err)
	}

	byType := make(map[string]int)
	totalActive := 0
	for _, e := range entries {
		if e.Status == "active" {
			totalActive++
			byType[e.Type]++
		}
	}

	return &KnowledgeHealthResult{
		TotalActive: totalActive,
		ByType:      byType,
	}, nil
}

// DefaultCreateKnowledgeHealthIssue creates a triage:review issue when
// knowledge entry accumulation exceeds the threshold.
// Skips creation if an open knowledge maintenance issue already exists.
func DefaultCreateKnowledgeHealthIssue(result *KnowledgeHealthResult) error {
	// Dedup: check for existing open knowledge maintenance issue
	listCmd := exec.Command("bd", "list", "--status=open", "-l", "area:knowledge")
	listOutput, err := listCmd.Output()
	if err == nil {
		lines := strings.Split(string(listOutput), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Knowledge maintenance:") {
				// Already have an open issue — skip creation
				return nil
			}
		}
	}
	// If bd list fails, proceed with creation (fail-open)

	title := fmt.Sprintf("Knowledge maintenance: %d active quick entries need promotion/pruning", result.TotalActive)
	cmd := exec.Command("bd", "create",
		"--title", title,
		"--type", "task",
		"--priority", "3",
		"-l", "triage:review",
		"-l", "area:knowledge",
	)
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to create knowledge health issue: %w", err)
	}
	return nil
}

// ShouldRunKnowledgeHealth returns true if periodic knowledge health check should run.
func (d *Daemon) ShouldRunKnowledgeHealth() bool {
	if !d.Config.KnowledgeHealthEnabled || d.Config.KnowledgeHealthInterval <= 0 {
		return false
	}
	if d.lastKnowledgeHealth.IsZero() {
		return true
	}
	return time.Since(d.lastKnowledgeHealth) >= d.Config.KnowledgeHealthInterval
}

// RunPeriodicKnowledgeHealth runs the knowledge health check if due.
// Returns the result if the check was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicKnowledgeHealth() *KnowledgeHealthResult {
	if !d.ShouldRunKnowledgeHealth() {
		return nil
	}

	kh := d.KnowledgeHealth
	if kh == nil {
		kh = &defaultKnowledgeHealthService{}
	}

	result, err := kh.Check()
	if err != nil {
		return &KnowledgeHealthResult{
			Error:   err,
			Message: fmt.Sprintf("Knowledge health check failed: %v", err),
		}
	}

	// Check threshold
	if d.Config.KnowledgeHealthThreshold > 0 && result.TotalActive >= d.Config.KnowledgeHealthThreshold {
		result.ThresholdExceeded = true
		result.Message = fmt.Sprintf("Knowledge accumulation: %d active entries (threshold: %d)", result.TotalActive, d.Config.KnowledgeHealthThreshold)

		// Create issue for knowledge maintenance
		if err := kh.CreateIssue(result); err != nil {
			result.Error = err
			result.Message += fmt.Sprintf(" (issue creation failed: %v)", err)
		}
	} else {
		result.Message = fmt.Sprintf("Knowledge health: %d active entries", result.TotalActive)
	}

	d.lastKnowledgeHealth = time.Now()

	return result
}

// LastKnowledgeHealthTime returns when knowledge health was last checked.
func (d *Daemon) LastKnowledgeHealthTime() time.Time {
	return d.lastKnowledgeHealth
}
