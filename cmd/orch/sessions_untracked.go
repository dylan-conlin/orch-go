package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/execution"
	"github.com/spf13/cobra"
)

var (
	untrackedSessionsJSON    bool
	untrackedSessionsLimit   int
	untrackedSessionsProject string
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List untracked OpenCode sessions (orchestrator/ad-hoc/no-track)",
	Long: `List untracked OpenCode sessions.

This command shows sessions that do NOT map to beads-tracked work items:
- orchestrator sessions (role=orchestrator)
- ad-hoc sessions (no beads_id)
- --no-track sessions (explicit opt-out)

Two-lane split:
  orch status   = tracked work only
  orch sessions = untracked sessions only

Examples:
  orch sessions
  orch sessions --limit 50
  orch sessions --project orch-go
  orch sessions --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUntrackedSessions(serverURL)
	},
}

func init() {
	sessionsCmd.Flags().BoolVar(&untrackedSessionsJSON, "json", false, "Output as JSON")
	sessionsCmd.Flags().IntVar(&untrackedSessionsLimit, "limit", 25, "Maximum number of sessions to show")
	sessionsCmd.Flags().StringVar(&untrackedSessionsProject, "project", "", "Filter by project (name or path)")

	rootCmd.AddCommand(sessionsCmd)
}

type UntrackedSessionOutput struct {
	ID            string `json:"id"`
	Title         string `json:"title,omitempty"`
	Category      string `json:"category"`
	Role          string `json:"role,omitempty"`
	BeadsID       string `json:"beads_id,omitempty"`
	Tier          string `json:"tier,omitempty"`
	SpawnMode     string `json:"spawn_mode,omitempty"`
	Skill         string `json:"skill,omitempty"`
	Model         string `json:"model,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	ProjectDir    string `json:"project_dir,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

func runUntrackedSessions(serverURL string) error {
	client := execution.NewOpenCodeAdapter(serverURL)
	projectDir, _ := os.Getwd()

	untracked, err := listUntrackedSessions(client, projectDir)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	filters := parseProjectFilters(untrackedSessionsProject)
	filtered := make([]UntrackedSessionOutput, 0, len(untracked))
	for _, entry := range untracked {
		if len(filters) > 0 && !filterByProject(entry.Session.Directory, filters) {
			continue
		}
		filtered = append(filtered, entry.toOutput())
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UpdatedAt > filtered[j].UpdatedAt
	})

	if untrackedSessionsLimit > 0 && len(filtered) > untrackedSessionsLimit {
		filtered = filtered[:untrackedSessionsLimit]
	}

	if untrackedSessionsJSON {
		data, err := json.MarshalIndent(filtered, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(filtered) == 0 {
		fmt.Println("No untracked sessions found")
		return nil
	}

	fmt.Printf("Found %d untracked sessions:\n\n", len(filtered))
	for _, session := range filtered {
		updated := session.UpdatedAt
		if updated == "" {
			updated = "unknown"
		}
		fmt.Printf("%-22s  %s\n", session.ID, truncate(session.Title, 70))
		fmt.Printf("  Category: %s | Updated: %s\n", session.Category, updated)
		if session.ProjectDir != "" {
			fmt.Printf("  Project:  %s\n", session.ProjectDir)
		}
		if session.Role != "" || session.Skill != "" || session.Tier != "" || session.SpawnMode != "" {
			fmt.Printf("  Meta:     role=%s skill=%s tier=%s mode=%s\n",
				emptyFallback(session.Role, "-"),
				emptyFallback(session.Skill, "-"),
				emptyFallback(session.Tier, "-"),
				emptyFallback(session.SpawnMode, "-"))
		}
		fmt.Println()
	}

	return nil
}

func parseProjectFilters(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func emptyFallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
