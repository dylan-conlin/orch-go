// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"os"
	"os/exec"
)

// GitHotspotChecker implements HotspotChecker using git history analysis.
// It shells out to `orch hotspot --json` to get hotspot data.
type GitHotspotChecker struct {
	// FixThreshold is the minimum fix commits to flag (default: 5).
	FixThreshold int
	// InvThreshold is the minimum investigations to flag (default: 3).
	InvThreshold int
	// DaysBack is the git history analysis period in days (default: 28).
	DaysBack int
}

// NewGitHotspotChecker creates a new GitHotspotChecker with default settings.
func NewGitHotspotChecker() *GitHotspotChecker {
	return &GitHotspotChecker{
		FixThreshold: 5,
		InvThreshold: 3,
		DaysBack:     28,
	}
}

// CheckHotspots runs `orch hotspot --json` and parses the results.
func (c *GitHotspotChecker) CheckHotspots(projectDir string) ([]HotspotWarning, error) {
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	// Build command
	cmd := exec.Command("orch", "hotspot", "--json")
	cmd.Dir = projectDir

	// Run and capture output
	output, err := cmd.Output()
	if err != nil {
		// orch hotspot might not be available or might fail
		// Return empty result (graceful degradation)
		return nil, nil
	}

	// Parse JSON output
	var report struct {
		Hotspots []struct {
			Path           string `json:"path"`
			Type           string `json:"type"`
			Score          int    `json:"score"`
			Recommendation string `json:"recommendation"`
		} `json:"hotspots"`
	}

	if err := json.Unmarshal(output, &report); err != nil {
		// Invalid JSON - graceful degradation
		return nil, nil
	}

	// Convert to HotspotWarning
	warnings := make([]HotspotWarning, len(report.Hotspots))
	for i, h := range report.Hotspots {
		warnings[i] = HotspotWarning{
			Path:           h.Path,
			Type:           h.Type,
			Score:          h.Score,
			Recommendation: h.Recommendation,
		}
	}

	return warnings, nil
}
