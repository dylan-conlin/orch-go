// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/artifactsync"
	"github.com/dylan-conlin/orch-go/pkg/beads"
)

const (
	// ArtifactSyncLabel is the beads label used for artifact sync issues (for dedup).
	ArtifactSyncLabel = "artifact-sync"
)

// defaultArtifactSyncService is the production ArtifactSyncService.
type defaultArtifactSyncService struct{}

func (s *defaultArtifactSyncService) Analyze(projectDir string) (*ArtifactSyncResult, error) {
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Load manifest
	manifest, err := artifactsync.LoadManifest(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	// Read drift events
	driftLogPath := artifactsync.DefaultDriftLogPath()
	events, err := artifactsync.ReadDriftEvents(driftLogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read drift events: %w", err)
	}

	if len(events) == 0 {
		return &ArtifactSyncResult{
			Message: "Artifact sync: no drift events found",
		}, nil
	}

	// Analyze drift
	report := artifactsync.AnalyzeDrift(manifest, events)

	if len(report.Entries) == 0 {
		return &ArtifactSyncResult{
			EventsCount: len(events),
			Message:     fmt.Sprintf("Artifact sync: %d events but no manifest matches", len(events)),
		}, nil
	}

	return &ArtifactSyncResult{
		DriftDetected: true,
		EntriesCount:  len(report.Entries),
		EventsCount:   len(events),
		Report:        report,
		Message:       fmt.Sprintf("Artifact sync: %d entries affected by %d events", len(report.Entries), len(events)),
	}, nil
}

func (s *defaultArtifactSyncService) HasOpenIssue() (bool, error) {
	issues, err := ListIssuesWithLabel(ArtifactSyncLabel)
	if err != nil {
		return false, err
	}
	return len(issues) > 0, nil
}

func (s *defaultArtifactSyncService) CreateIssue(report *artifactsync.DriftReport) (string, error) {
	title := "Artifact sync: update drifted documentation"
	description := formatArtifactSyncDescription(report)

	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Create(&beads.CreateArgs{
				Title:       title,
				Description: description,
				IssueType:   "task",
				Priority:    3,
				Labels:      []string{ArtifactSyncLabel, "triage:ready"},
			})
			if err == nil {
				return issue.ID, nil
			}
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackCreate(title, description, "task", 3, []string{ArtifactSyncLabel, "triage:ready"}, "")
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}

func (s *defaultArtifactSyncService) SpawnSyncAgent(report *artifactsync.DriftReport) error {
	task := buildArtifactSyncTask(report)
	// Shell out to orch spawn with artifact-sync skill, light tier, bypass triage
	cmd := exec.Command("orch", "spawn", "--bypass-triage", "--light", "artifact-sync", task)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to spawn artifact-sync agent: %w: %s", err, string(output))
	}
	return nil
}

func (s *defaultArtifactSyncService) SpawnBudgetAwareSyncAgent(report *artifactsync.DriftReport, currentLines, budget int) error {
	task := buildBudgetAwareSyncTask(report, currentLines, budget)
	cmd := exec.Command("orch", "spawn", "--bypass-triage", "--light", "artifact-sync", task)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to spawn budget-aware artifact-sync agent: %w: %s", err, string(output))
	}
	return nil
}

func (s *defaultArtifactSyncService) CLAUDEMDLineCount(projectDir string) (int, error) {
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return 0, fmt.Errorf("failed to get working directory: %w", err)
		}
	}
	data, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	if err != nil {
		return 0, fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}
	return bytes.Count(data, []byte("\n")) + 1, nil
}

func formatArtifactSyncDescription(report *artifactsync.DriftReport) string {
	var b strings.Builder
	b.WriteString("Artifact drift detected. The following artifacts may need updating:\n\n")

	for _, entry := range report.Entries {
		label := entry.ArtifactPath
		if entry.SectionName != "" {
			label = fmt.Sprintf("%s:%s", entry.ArtifactPath, entry.SectionName)
		}
		b.WriteString(fmt.Sprintf("- %s (triggers: %s)\n", label, strings.Join(entry.Triggers, ", ")))
	}

	b.WriteString(fmt.Sprintf("\nSource: %s\n", artifactsync.DefaultDriftLogPath()))
	return b.String()
}

func buildArtifactSyncTask(report *artifactsync.DriftReport) string {
	var lines []string
	lines = append(lines, "Update the following drifted artifacts based on recent code changes:")
	lines = append(lines, "")

	for _, entry := range report.Entries {
		label := entry.ArtifactPath
		if entry.SectionName != "" {
			label = fmt.Sprintf("%s:%s", entry.ArtifactPath, entry.SectionName)
		}

		var commitRanges []string
		seen := make(map[string]bool)
		for _, ev := range entry.Events {
			if ev.CommitRange != "" && !seen[ev.CommitRange] {
				seen[ev.CommitRange] = true
				commitRanges = append(commitRanges, ev.CommitRange)
			}
		}

		line := fmt.Sprintf("- %s (triggers: %s)", label, strings.Join(entry.Triggers, ", "))
		if len(commitRanges) > 0 {
			line += fmt.Sprintf(" [commits: %s]", strings.Join(commitRanges, ", "))
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func buildBudgetAwareSyncTask(report *artifactsync.DriftReport, currentLines, budget int) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("CLAUDE.md LINE BUDGET EXCEEDED: %d lines (budget: %d). Before adding new content, you MUST remove lowest-relevance content to bring the file under budget.", currentLines, budget))
	lines = append(lines, "")
	lines = append(lines, "Steps:")
	lines = append(lines, "1. Read CLAUDE.md and identify sections with lowest relevance to agent behavior")
	lines = append(lines, "2. Remove or condense those sections until line count is under the budget")
	lines = append(lines, "3. Then update the following drifted artifacts:")
	lines = append(lines, "")

	for _, entry := range report.Entries {
		label := entry.ArtifactPath
		if entry.SectionName != "" {
			label = fmt.Sprintf("%s:%s", entry.ArtifactPath, entry.SectionName)
		}

		var commitRanges []string
		seen := make(map[string]bool)
		for _, ev := range entry.Events {
			if ev.CommitRange != "" && !seen[ev.CommitRange] {
				seen[ev.CommitRange] = true
				commitRanges = append(commitRanges, ev.CommitRange)
			}
		}

		line := fmt.Sprintf("- %s (triggers: %s)", label, strings.Join(entry.Triggers, ", "))
		if len(commitRanges) > 0 {
			line += fmt.Sprintf(" [commits: %s]", strings.Join(commitRanges, ", "))
		}
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("4. Verify final line count is under %d lines", budget))

	return strings.Join(lines, "\n")
}
