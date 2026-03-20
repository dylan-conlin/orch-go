package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/claims"
)

// ClaimProbeResult holds the outcome of a claim probe generation cycle.
type ClaimProbeResult struct {
	Error       error
	Message     string
	ProbeCount  int
	ProbeIssues []string // beads IDs of created probe issues
}

// Snapshot returns a snapshot for status file writing.
func (r *ClaimProbeResult) Snapshot() ClaimProbeSnapshot {
	return ClaimProbeSnapshot{
		ProbeCount:  r.ProbeCount,
		ProbeIssues: r.ProbeIssues,
		Message:     r.Message,
	}
}

// ClaimProbeSnapshot is the daemon status snapshot for claim probe generation.
type ClaimProbeSnapshot struct {
	ProbeCount  int      `json:"probe_count"`
	ProbeIssues []string `json:"probe_issues,omitempty"`
	Message     string   `json:"message"`
}

// ClaimProbeService is the interface for checking existing probe issues.
type ClaimProbeService interface {
	// HasOpenProbeForClaim returns true if there is already an open issue
	// labeled with the given claim ID.
	HasOpenProbeForClaim(claimID, modelName string) (bool, error)

	// CreateProbeIssue creates a probe investigation issue for the given claim.
	// Returns the beads issue ID.
	CreateProbeIssue(claimID, claimText, falsifiesIf, modelName string) (string, error)
}

// RunPeriodicClaimProbeGeneration runs claim probe generation if due.
func (d *Daemon) RunPeriodicClaimProbeGeneration() *ClaimProbeResult {
	if !d.Scheduler.IsDue(TaskClaimProbeGeneration) {
		return nil
	}

	result := d.runClaimProbeGeneration()

	if result != nil && result.Error == nil {
		d.Scheduler.MarkRun(TaskClaimProbeGeneration)
	}

	return result
}

// runClaimProbeGeneration scans claims.yaml files in active models and generates
// probe issues for stale or unconfirmed claims. Max 1 probe per cycle.
func (d *Daemon) runClaimProbeGeneration() *ClaimProbeResult {
	if d.ClaimProbeService == nil {
		return &ClaimProbeResult{
			Message: "claim probe service not configured",
		}
	}

	// Find models directory
	modelsDir := findModelsDir()
	if modelsDir == "" {
		return &ClaimProbeResult{
			Message: "no .kb/models/ directory found",
		}
	}

	// Scan all claims.yaml files
	files, err := claims.ScanAll(modelsDir)
	if err != nil {
		return &ClaimProbeResult{
			Error:   err,
			Message: fmt.Sprintf("scan claims failed: %v", err),
		}
	}
	if len(files) == 0 {
		return &ClaimProbeResult{
			Message: "no claims.yaml files found",
		}
	}

	now := time.Now()
	var result ClaimProbeResult

	// Check each model's claims for probe eligibility
	for modelName, f := range files {
		// Check model has activity signal (updated recently or referenced in spawns)
		if !hasModelActivity(modelsDir, modelName) {
			continue
		}

		for _, c := range f.Claims {
			if !c.IsProbeEligible(now) {
				continue
			}
			if c.FalsifiesIf == "" {
				continue // No falsification condition = no probe question
			}

			// Dedup: check if open probe already exists for this claim
			hasOpen, err := d.ClaimProbeService.HasOpenProbeForClaim(c.ID, modelName)
			if err != nil || hasOpen {
				continue
			}

			// Create probe issue (max 1 per cycle)
			issueID, err := d.ClaimProbeService.CreateProbeIssue(c.ID, c.Text, c.FalsifiesIf, modelName)
			if err != nil {
				result.Error = err
				result.Message = fmt.Sprintf("create probe issue failed: %v", err)
				return &result
			}

			result.ProbeCount++
			result.ProbeIssues = append(result.ProbeIssues, issueID)
			result.Message = fmt.Sprintf("generated probe for claim %s in model %s", c.ID, modelName)
			return &result // Max 1 probe per cycle
		}
	}

	result.Message = "no probe-eligible claims found"
	return &result
}

// hasModelActivity returns true if the model has been updated within 14 days.
func hasModelActivity(modelsDir, modelName string) bool {
	modelPath := filepath.Join(modelsDir, modelName, "model.md")
	info, err := os.Stat(modelPath)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < 14*24*time.Hour
}

// findModelsDir returns the .kb/models/ directory path if it exists.
func findModelsDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	modelsDir := filepath.Join(cwd, ".kb", "models")
	if _, err := os.Stat(modelsDir); err != nil {
		return ""
	}
	return modelsDir
}
