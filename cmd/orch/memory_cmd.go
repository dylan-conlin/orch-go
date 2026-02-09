package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/episodic"
	"github.com/spf13/cobra"
)

var (
	memoryEpisodesFor   string
	memoryEpisodesLimit int
	memoryEpisodesJSON  bool
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Inspect episodic memory artifacts",
}

var memoryEpisodesCmd = &cobra.Command{
	Use:   "episodes",
	Short: "Inspect episodic validation states for an issue",
	Long: `Inspect action-memory episodes and show validation states.

Examples:
  orch memory episodes --for orch-go-12345
  orch memory episodes --for orch-go-12345 --json
  orch memory episodes --for orch-go-12345 --limit 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(memoryEpisodesFor) == "" {
			return fmt.Errorf("--for is required")
		}
		return runMemoryEpisodes(memoryEpisodesFor, memoryEpisodesLimit, memoryEpisodesJSON)
	},
}

func init() {
	memoryEpisodesCmd.Flags().StringVar(&memoryEpisodesFor, "for", "", "Beads issue ID to inspect")
	memoryEpisodesCmd.Flags().IntVar(&memoryEpisodesLimit, "limit", 20, "Maximum episodes to inspect")
	memoryEpisodesCmd.Flags().BoolVar(&memoryEpisodesJSON, "json", false, "Output JSON")

	memoryCmd.AddCommand(memoryEpisodesCmd)
	rootCmd.AddCommand(memoryCmd)
}

type memoryEpisodeRow struct {
	State      episodic.ValidationState `json:"state"`
	Action     string                   `json:"action"`
	Summary    string                   `json:"summary"`
	Confidence float64                  `json:"confidence"`
	Source     string                   `json:"source"`
	Reasons    []string                 `json:"reasons,omitempty"`
}

func runMemoryEpisodes(beadsID string, limit int, jsonOutput bool) error {
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	rows, err := inspectEpisodes(beadsID, limit, projectDir)
	if err != nil {
		return err
	}

	if jsonOutput {
		data, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal episodes: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(rows) == 0 {
		fmt.Printf("No episodic entries found for %s\n", beadsID)
		return nil
	}

	accepted := 0
	degraded := 0
	rejected := 0
	for _, row := range rows {
		switch row.State {
		case episodic.ValidationStateAccepted:
			accepted++
		case episodic.ValidationStateDegraded:
			degraded++
		case episodic.ValidationStateRejected:
			rejected++
		}
	}

	fmt.Printf("Episodes for %s\n", beadsID)
	fmt.Printf("accepted=%d degraded=%d rejected=%d\n\n", accepted, degraded, rejected)

	for _, row := range rows {
		line := fmt.Sprintf("[%s] %s - %s (confidence %.2f, source %s)", row.State, row.Action, row.Summary, row.Confidence, row.Source)
		fmt.Println(line)
		if len(row.Reasons) > 0 {
			fmt.Printf("  reasons: %s\n", strings.Join(row.Reasons, ", "))
		}
	}

	return nil
}

func inspectEpisodes(beadsID string, limit int, projectDir string) ([]memoryEpisodeRow, error) {
	store := episodic.NewStore("")
	entries, err := store.QueryByBeadsID(beadsID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read episodic store: %w", err)
	}

	if len(entries) == 0 {
		return nil, nil
	}

	validated := episodic.ValidateEpisodesForReuse(entries, episodic.ValidateOptions{
		AutoInjection: true,
		MinConfidence: 0.7,
		Scope: episodic.Scope{
			Project:    scopeProjectFromDir(projectDir),
			BeadsID:    beadsID,
			ProjectDir: projectDir,
		},
	})

	rows := make([]memoryEpisodeRow, 0, len(validated))
	for _, item := range validated {
		action := strings.TrimSpace(item.Episode.Action.Name)
		if action == "" {
			action = strings.TrimSpace(item.Episode.Action.Type)
		}
		if action == "" {
			action = "action"
		}

		summary := strings.TrimSpace(item.Summary)
		if summary == "" {
			summary = strings.TrimSpace(item.Episode.Outcome.Summary)
		}

		source := "unknown"
		if strings.TrimSpace(item.Episode.Evidence.Pointer) != "" {
			source = evidenceBase(item.Episode.Evidence.Pointer)
		}

		rows = append(rows, memoryEpisodeRow{
			State:      item.State,
			Action:     action,
			Summary:    summary,
			Confidence: item.Episode.Confidence,
			Source:     source,
			Reasons:    item.Reasons,
		})
	}

	return rows, nil
}

func scopeProjectFromDir(projectDir string) string {
	trimmed := strings.TrimSpace(projectDir)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(strings.TrimRight(trimmed, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func evidenceBase(pointer string) string {
	trimmed := strings.TrimSpace(pointer)
	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	trimmed = strings.TrimPrefix(trimmed, "~/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return "unknown"
	}
	last := strings.TrimSpace(parts[len(parts)-1])
	if last == "" {
		return "unknown"
	}
	return last
}
