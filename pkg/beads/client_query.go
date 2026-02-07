package beads

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Ready retrieves ready issues from the daemon.
func (c *Client) Ready(args *ReadyArgs) ([]Issue, error) {
	if args == nil {
		args = &ReadyArgs{}
	}

	resp, err := c.execute(OpReady, args)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(resp.Data, &issues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ready issues: %w", err)
	}

	return issues, nil
}

// List retrieves issues matching the given criteria.
func (c *Client) List(args *ListArgs) ([]Issue, error) {
	if args == nil {
		args = &ListArgs{}
	}

	resp, err := c.execute(OpList, args)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(resp.Data, &issues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issues: %w", err)
	}

	return issues, nil
}

// Stats retrieves beads statistics.
func (c *Client) Stats() (*Stats, error) {
	resp, err := c.execute(OpStats, nil)
	if err != nil {
		return nil, err
	}

	// RPC returns flat stats (no "summary" wrapper), CLI returns wrapped.
	// Try flat format first (RPC), then wrapped format (CLI fallback compatibility).
	var summary StatsSummary
	if err := json.Unmarshal(resp.Data, &summary); err == nil && summary.TotalIssues > 0 {
		// RPC format: flat StatsSummary
		return &Stats{Summary: summary}, nil
	}

	// Try wrapped format (CLI format)
	var stats Stats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	return &stats, nil
}

// Stale retrieves stale issues.
func (c *Client) Stale(args *StaleArgs) ([]Issue, error) {
	if args == nil {
		args = &StaleArgs{}
	}

	resp, err := c.execute(OpStale, args)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(resp.Data, &issues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stale issues: %w", err)
	}

	return issues, nil
}

// Count counts issues matching the given criteria.
func (c *Client) Count(args *CountArgs) (*CountResponse, error) {
	if args == nil {
		args = &CountArgs{}
	}

	resp, err := c.execute(OpCount, args)
	if err != nil {
		return nil, err
	}

	var countResp CountResponse
	if err := json.Unmarshal(resp.Data, &countResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal count response: %w", err)
	}

	return &countResp, nil
}

// ResolveID resolves a partial issue ID to a full ID.
func (c *Client) ResolveID(partialID string) (string, error) {
	args := ResolveIDArgs{ID: partialID}

	resp, err := c.execute(OpResolveID, args)
	if err != nil {
		return "", err
	}

	// The response data is the resolved ID as a string
	var resolvedID string
	if err := json.Unmarshal(resp.Data, &resolvedID); err != nil {
		return "", fmt.Errorf("failed to unmarshal resolved ID: %w", err)
	}

	return resolvedID, nil
}

// FallbackReady retrieves ready issues via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackReady() ([]Issue, error) {
	// Use --limit 0 to get ALL ready issues (bd ready defaults to limit 10)
	output, err := runBDOutput(DefaultDir, "ready", "--json", "--limit", "0")
	if err != nil {
		if IsCLITimeout(err) {
			return nil, fmt.Errorf("bd ready timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd ready failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd ready failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd ready output: %w", err)
	}

	return issues, nil
}

// FallbackList retrieves issues via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
// Uses --limit 0 to get ALL issues (bd list defaults to 50 most recent).
func FallbackList(status string) ([]Issue, error) {
	// Use --limit 0 to get ALL issues. Without this, bd list returns only
	// the 50 most recent issues, which can miss in_progress issues when
	// the repo has many recent closed issues (discovered in orch-go-20942).
	args := []string{"list", "--json", "--limit", "0"}
	if status != "" {
		args = append(args, "--status", status)
	}

	output, err := runBDOutput(DefaultDir, args...)
	if err != nil {
		if IsCLITimeout(err) {
			return nil, fmt.Errorf("bd list timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackListByIDs retrieves specific issues by ID via bd CLI.
// Uses --id and --all flags to fetch issues regardless of status.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackListByIDs(ids []string) ([]Issue, error) {
	if len(ids) == 0 {
		return []Issue{}, nil
	}

	// Use --id with comma-separated IDs and --all to include closed issues
	args := []string{"list", "--json", "--all", "--id", strings.Join(ids, ",")}

	output, err := runBDOutput(DefaultDir, args...)
	if err != nil {
		if IsCLITimeout(err) {
			return nil, fmt.Errorf("bd list --id timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list --id failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list --id failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackListByParent retrieves children of a parent issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackListByParent(parentID string) ([]Issue, error) {
	if parentID == "" {
		return []Issue{}, nil
	}

	// Use --parent and --all to include closed children
	// Use --limit 0 to get all children
	args := []string{"list", "--json", "--limit", "0", "--parent", parentID}

	output, err := runBDOutput(DefaultDir, args...)
	if err != nil {
		if IsCLITimeout(err) {
			return nil, fmt.Errorf("bd list --parent timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list --parent failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list --parent failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackStats retrieves stats via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackStats() (*Stats, error) {
	output, err := runBDOutput(DefaultDir, "stats", "--json")
	if err != nil {
		if IsCLITimeout(err) {
			return nil, fmt.Errorf("bd stats timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd stats failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd stats failed: %w", err)
	}

	var stats Stats
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse bd stats output: %w", err)
	}

	return &stats, nil
}
