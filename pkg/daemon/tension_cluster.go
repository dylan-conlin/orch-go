package daemon

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/claims"
)

// TensionClusterResult holds the outcome of a tension cluster scan cycle.
type TensionClusterResult struct {
	Error        error
	Message      string
	ClusterCount int    // total clusters found meeting threshold
	IssueCreated string // beads ID of created issue (max 1 per cycle)
}

// Snapshot returns a snapshot for status file writing.
func (r *TensionClusterResult) Snapshot() TensionClusterSnapshot {
	return TensionClusterSnapshot{
		ClusterCount: r.ClusterCount,
		IssueCreated: r.IssueCreated,
		Message:      r.Message,
	}
}

// TensionClusterSnapshot is the daemon status snapshot for tension cluster scanning.
type TensionClusterSnapshot struct {
	ClusterCount int    `json:"cluster_count"`
	IssueCreated string `json:"issue_created,omitempty"`
	Message      string `json:"message"`
}

// TensionClusterIssue holds the data needed to create a beads issue for a tension cluster.
type TensionClusterIssue struct {
	ClusterID   string
	Summary     string
	Description string
}

// TensionClusterService is the interface for checking/creating tension cluster issues.
type TensionClusterService interface {
	// HasOpenClusterIssue returns true if there is already an open issue
	// for the given cluster ID (labeled daemon:tension-cluster).
	HasOpenClusterIssue(clusterID string) (bool, error)

	// CreateClusterIssue creates an architect issue for the tension cluster.
	// Returns the beads issue ID.
	CreateClusterIssue(issue TensionClusterIssue) (string, error)
}

// RunPeriodicTensionClusterScan runs the tension cluster scan if due.
func (d *Daemon) RunPeriodicTensionClusterScan() *TensionClusterResult {
	if !d.Scheduler.IsDue(TaskTensionClusterScan) {
		return nil
	}

	result := d.runTensionClusterScan()

	if result != nil && result.Error == nil {
		d.Scheduler.MarkRun(TaskTensionClusterScan)
	}

	return result
}

// runTensionClusterScan scans claims for tension clusters and creates architect issues.
// Max 1 issue per cycle.
func (d *Daemon) runTensionClusterScan() *TensionClusterResult {
	if d.TensionClusterService == nil {
		return &TensionClusterResult{
			Message: "tension cluster service not configured",
		}
	}

	modelsDir := findModelsDir()
	if modelsDir == "" {
		return &TensionClusterResult{
			Message: "no .kb/models/ directory found",
		}
	}

	files, err := claims.ScanAll(modelsDir)
	if err != nil {
		return &TensionClusterResult{
			Error:   err,
			Message: fmt.Sprintf("scan claims failed: %v", err),
		}
	}
	if len(files) == 0 {
		return &TensionClusterResult{
			Message: "no claims.yaml files found",
		}
	}

	threshold := d.Config.TensionClusterThreshold
	if threshold < 2 {
		threshold = 3
	}

	clusters := claims.FindClusters(files, threshold)
	if len(clusters) == 0 {
		return &TensionClusterResult{
			Message: "no tension clusters found meeting threshold",
		}
	}

	result := TensionClusterResult{
		ClusterCount: len(clusters),
	}

	// Create issue for first cluster that doesn't already have one (max 1 per cycle)
	for _, c := range clusters {
		hasOpen, err := d.TensionClusterService.HasOpenClusterIssue(c.ID)
		if err != nil {
			continue
		}
		if hasOpen {
			continue
		}

		issue := TensionClusterIssue{
			ClusterID:   c.ID,
			Summary:     formatClusterSummary(c),
			Description: formatClusterDescription(c),
		}

		issueID, err := d.TensionClusterService.CreateClusterIssue(issue)
		if err != nil {
			result.Error = err
			result.Message = fmt.Sprintf("create cluster issue failed: %v", err)
			return &result
		}

		result.IssueCreated = issueID
		result.Message = fmt.Sprintf("created issue for tension cluster %s (score=%.0f, %d claims from %d models)",
			c.ID, c.Score, len(c.Claims), len(c.Models))
		return &result
	}

	result.Message = fmt.Sprintf("found %d clusters, all already have open issues", len(clusters))
	return &result
}

// formatClusterSummary returns a one-line summary for the beads issue title.
func formatClusterSummary(c claims.TensionCluster) string {
	return fmt.Sprintf("Tension cluster: %s (%d claims from %d models, score=%.0f)",
		c.TargetClaim, len(c.Claims), len(c.Models), c.Score)
}

// formatClusterDescription returns the full context for the architect issue body.
func formatClusterDescription(c claims.TensionCluster) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Tension Cluster: %s\n\n", c.ID))
	b.WriteString(fmt.Sprintf("### Convergence Point\n%s (in %s)\n\n", c.TargetClaim, c.TargetModel))
	b.WriteString(fmt.Sprintf("### Tensions (%d claims from %d models)\n", len(c.Claims), len(c.Models)))
	for _, m := range c.Claims {
		b.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", m.ClaimID, m.ModelName, m.Text))
		b.WriteString(fmt.Sprintf("  Tension: %s — %s\n", m.TensionType, m.Note))
	}
	if len(c.DomainTags) > 0 {
		b.WriteString(fmt.Sprintf("\n### Domain Tags\n%s\n", strings.Join(c.DomainTags, ", ")))
	}
	b.WriteString(fmt.Sprintf("\n### Question for Architect\n"))
	b.WriteString(fmt.Sprintf("These %d claims from %d models converge on %s. ",
		len(c.Claims), len(c.Models), c.TargetClaim))
	b.WriteString("Design: what implementation work resolves, strengthens, or restructures this tension area?\n")
	return b.String()
}
