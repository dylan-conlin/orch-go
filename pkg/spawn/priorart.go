// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// PriorCompletion represents a completed prior agent's work on the same or related issue.
type PriorCompletion struct {
	// BeadsID is the issue ID this completion was tracked under.
	BeadsID string
	// Skill is the skill used by the prior agent.
	Skill string
	// Summary is the extracted TLDR from SYNTHESIS.md or the close_reason fallback.
	Summary string
	// Workspace is the archived workspace name for reference.
	Workspace string
	// SpawnTime is when the prior agent was spawned.
	SpawnTime time.Time
}

// GatherPriorArt finds prior completed work related to the given beads issue.
// It scans archived workspaces for matching beads IDs and extracts summaries.
// When a beads client is provided, it also looks up close_reason as a fallback
// when SYNTHESIS.md is not available (light tier spawns).
// Returns formatted markdown for injection into SPAWN_CONTEXT.md, or empty string if none found.
func GatherPriorArt(beadsID string, projectDir string, client beads.BeadsClient) string {
	if beadsID == "" {
		return ""
	}

	// Find all archived workspaces with matching beads ID
	archivedWorkspaces := FindArchivedWorkspacesByBeadsID(projectDir, beadsID)

	var completions []PriorCompletion

	// Try to get close_reason from beads as fallback (fetched once, used per-workspace)
	var issueCloseReason string
	if client != nil {
		if issue, err := client.Show(beadsID); err == nil && issue.Status == "closed" {
			issueCloseReason = issue.CloseReason
		}
	}

	for _, ws := range archivedWorkspaces {
		completion := extractCompletionFromWorkspace(ws, issueCloseReason)
		if completion != nil {
			completions = append(completions, *completion)
		}
	}

	if len(completions) == 0 {
		return ""
	}

	// Sort by spawn time (oldest first) to show progression
	sort.Slice(completions, func(i, j int) bool {
		return completions[i].SpawnTime.Before(completions[j].SpawnTime)
	})

	return FormatPriorCompletions(completions)
}

// archivedWorkspace holds metadata about a found archived workspace.
type archivedWorkspace struct {
	Path      string
	Manifest  AgentManifest
	SpawnTime time.Time
}

// FindArchivedWorkspacesByBeadsID scans the archived workspace directory for all
// workspaces matching the given beads ID. Returns all matches (not just most recent).
func FindArchivedWorkspacesByBeadsID(projectDir, beadsID string) []archivedWorkspace {
	if beadsID == "" {
		return nil
	}

	archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")
	entries, err := os.ReadDir(archivedDir)
	if err != nil {
		return nil
	}

	var matches []archivedWorkspace
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wsPath := filepath.Join(archivedDir, entry.Name())
		manifest := ReadAgentManifestWithFallback(wsPath)
		if manifest == nil || manifest.BeadsID != beadsID {
			continue
		}

		spawnTime := manifest.ParseSpawnTime()
		if spawnTime.IsZero() {
			if info, err := entry.Info(); err == nil {
				spawnTime = info.ModTime()
			}
		}

		matches = append(matches, archivedWorkspace{
			Path:      wsPath,
			Manifest:  *manifest,
			SpawnTime: spawnTime,
		})
	}

	return matches
}

// extractCompletionFromWorkspace extracts a PriorCompletion from an archived workspace.
// Prefers SYNTHESIS.md TLDR, falls back to close_reason from beads.
// Returns nil if no usable summary can be extracted.
func extractCompletionFromWorkspace(ws archivedWorkspace, closeReason string) *PriorCompletion {
	summary := ""

	// Try SYNTHESIS.md first
	synthesisPath := filepath.Join(ws.Path, "SYNTHESIS.md")
	if content, err := os.ReadFile(synthesisPath); err == nil {
		summary = ExtractTLDRFromSynthesis(string(content))
	}

	// Fall back to close_reason
	if summary == "" && closeReason != "" {
		summary = closeReason
	}

	// No usable summary
	if summary == "" {
		return nil
	}

	return &PriorCompletion{
		BeadsID:   ws.Manifest.BeadsID,
		Skill:     ws.Manifest.Skill,
		Summary:   summary,
		Workspace: ws.Manifest.WorkspaceName,
		SpawnTime: ws.SpawnTime,
	}
}

// ExtractTLDRFromSynthesis extracts the TLDR section content from a SYNTHESIS.md file.
// Returns empty string if no TLDR section is found.
func ExtractTLDRFromSynthesis(content string) string {
	lines := strings.Split(content, "\n")
	return extractSection(lines, "tldr")
}

// FormatPriorCompletions formats a slice of PriorCompletion into markdown
// suitable for injection into SPAWN_CONTEXT.md.
func FormatPriorCompletions(completions []PriorCompletion) string {
	if len(completions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## PRIOR COMPLETIONS\n\n")
	sb.WriteString("Prior agents completed work on this issue. Review before starting to avoid re-doing completed work.\n")
	sb.WriteString("**Do NOT re-do work that was already completed** — build on it.\n\n")

	for i, c := range completions {
		sb.WriteString(fmt.Sprintf("### Prior Agent %d", i+1))
		if c.Skill != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", c.Skill))
		}
		sb.WriteString("\n")

		if c.BeadsID != "" {
			sb.WriteString(fmt.Sprintf("- **Issue:** %s\n", c.BeadsID))
		}
		if c.Workspace != "" {
			sb.WriteString(fmt.Sprintf("- **Workspace:** %s\n", c.Workspace))
		}
		sb.WriteString(fmt.Sprintf("- **Summary:** %s\n", c.Summary))
		sb.WriteString("\n")
	}

	return sb.String()
}
