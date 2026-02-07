// Package verify provides verification helpers for agent completion.
package verify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// DashboardHealthResult represents the result of dashboard health check.
type DashboardHealthResult struct {
	Passed   bool     // Whether health check passed
	Errors   []string // Error messages (blocking)
	Warnings []string // Warning messages (non-blocking)
}

// BeadsGraphResponse matches the JSON structure returned by /api/beads/graph.
type BeadsGraphResponse struct {
	Nodes     []interface{} `json:"nodes"`
	Edges     []interface{} `json:"edges"`
	NodeCount int           `json:"node_count"`
	EdgeCount int           `json:"edge_count"`
	Error     string        `json:"error,omitempty"`
}

// BeadsStatsResponse matches the JSON structure returned by /api/beads.
type BeadsStatsResponse struct {
	TotalIssues    int     `json:"total_issues"`
	OpenIssues     int     `json:"open_issues"`
	InProgress     int     `json:"in_progress_issues"`
	BlockedIssues  int     `json:"blocked_issues"`
	ReadyIssues    int     `json:"ready_issues"`
	ClosedIssues   int     `json:"closed_issues"`
	AvgLeadTimeHrs float64 `json:"avg_lead_time_hours,omitempty"`
	Error          string  `json:"error,omitempty"`
}

// hasDashboardChanges checks if any files in the list match dashboard patterns.
// Dashboard patterns include:
//   - web/ directory (frontend code)
//   - cmd/orch/serve_*.go (API endpoint files)
func hasDashboardChanges(files []string) bool {
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		// Check if file is in web/ directory
		if strings.HasPrefix(file, "web/") {
			return true
		}

		// Check if file is a serve_*.go file in cmd/orch
		if strings.Contains(file, "cmd/orch/serve_") && strings.HasSuffix(file, ".go") {
			return true
		}
	}

	return false
}

// getModifiedFiles returns the list of files modified in recent commits.
func getModifiedFiles(projectDir string) ([]string, error) {
	// Get files modified in the last commit
	// This matches the scope of what an agent just completed
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Try with just HEAD if no previous commit exists
		cmd = exec.Command("git", "diff", "--name-only", "--cached")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return nil, err
		}
	}

	lines := strings.Split(string(output), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// checkDashboardHealth verifies that dashboard API endpoints are responding correctly.
// It checks:
//  1. /api/beads/graph returns node_count > 0
//  2. /api/beads returns open_issues > 0
//
// Returns an error if any check fails.
func checkDashboardHealth(serverURL string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Check /api/beads/graph endpoint
	graphURL := strings.TrimSuffix(serverURL, "/") + "/api/beads/graph"
	resp, err := client.Get(graphURL)
	if err != nil {
		return fmt.Errorf("failed to connect to /api/beads/graph: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/api/beads/graph returned status %d (expected 200)", resp.StatusCode)
	}

	var graphResp BeadsGraphResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphResp); err != nil {
		return fmt.Errorf("failed to parse /api/beads/graph response: %v", err)
	}

	if graphResp.Error != "" {
		return fmt.Errorf("/api/beads/graph returned error: %s", graphResp.Error)
	}

	if graphResp.NodeCount == 0 {
		return fmt.Errorf("/api/beads/graph returned node_count=0 (expected > 0)")
	}

	// Check /api/beads endpoint
	beadsURL := strings.TrimSuffix(serverURL, "/") + "/api/beads"
	resp, err = client.Get(beadsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to /api/beads: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/api/beads returned status %d (expected 200)", resp.StatusCode)
	}

	var beadsResp BeadsStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&beadsResp); err != nil {
		return fmt.Errorf("failed to parse /api/beads response: %v", err)
	}

	if beadsResp.Error != "" {
		return fmt.Errorf("/api/beads returned error: %s", beadsResp.Error)
	}

	if beadsResp.OpenIssues == 0 {
		return fmt.Errorf("/api/beads returned open_issues=0 (expected > 0)")
	}

	return nil
}

// VerifyDashboardHealth checks if dashboard-touching changes require health verification.
// Returns nil if no verification needed (no dashboard changes).
// Returns DashboardHealthResult if dashboard changes detected.
//
// The verification detects changes to:
//   - web/ directory (UI code)
//   - cmd/orch/serve_*.go (API endpoints)
//
// If changes detected, it verifies the dashboard server is healthy by:
//  1. Checking /api/beads/graph returns node_count > 0
//  2. Checking /api/beads returns open_issues > 0
//
// This prevents agents from breaking the dashboard without detection.
func VerifyDashboardHealth(workspacePath, projectDir, serverURL string) *DashboardHealthResult {
	result := &DashboardHealthResult{Passed: true}

	// Get modified files from git
	files, err := getModifiedFiles(projectDir)
	if err != nil {
		result.Warnings = append(result.Warnings,
			"could not check modified files - skipping dashboard health check: "+err.Error())
		return result
	}

	// Check if any dashboard files were modified
	if !hasDashboardChanges(files) {
		// No dashboard changes - verification not needed
		return nil
	}

	// Dashboard changes detected - perform health check
	result.Warnings = append(result.Warnings,
		"dashboard-touching files modified - running health check")

	// Determine server URL if not provided
	if serverURL == "" {
		// Default to localhost:3348 (orch-go default API port)
		serverURL = "http://localhost:3348"
	}

	// Run health check
	if err := checkDashboardHealth(serverURL); err != nil {
		result.Passed = false
		result.Errors = append(result.Errors,
			"Dashboard health check failed: "+err.Error(),
			"Changes to dashboard files (web/ or serve_*.go) require verification",
			"Rebuild and restart the server, then verify endpoints are working",
		)
		return result
	}

	result.Warnings = append(result.Warnings,
		"dashboard health check passed - endpoints responding correctly")

	return result
}
