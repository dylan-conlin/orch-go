package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

// ============================================================================
// Session End Command
// ============================================================================

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: "End the current session",
	Long: `End the current orchestrator work session.

This clears the session state. Use before:
- Taking a break
- Handing off to another orchestrator
- Changing focus to a different goal

The session summary is logged for posterity.

Examples:
  orch session end`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionEnd()
	},
}

func runSessionEnd() error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	if !store.IsActive() {
		fmt.Println("No active session to end")
		return nil
	}

	// Gate: Check for accumulated investigation promotion candidates
	// This prevents backlog accumulation by prompting triage before session end
	if err := gateInvestigationPromotions(); err != nil {
		return err
	}

	// Get session info before ending - IMPORTANT: Get the session object to access WindowName
	// which was captured at session start. This is used for archiving, NOT GetCurrentWindowName().
	sess := store.Get()
	duration := store.Duration()
	spawnCount := store.SpawnCount()

	// Get spawn statuses for final summary
	statuses := store.GetSpawnStatuses(serverURL)
	activeCount := 0
	for _, s := range statuses {
		if s.State == "active" {
			activeCount++
		}
	}

	// Get project directory
	projectDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get project directory: %v\n", err)
	} else {
		// Use the stored window name from session start, NOT GetCurrentWindowName()
		// This ensures we archive to the correct directory even if called from a different window
		windowName := sess.WindowName
		if windowName == "" {
			// Fallback for sessions created before WindowName was added
			windowName, err = tmux.GetCurrentWindowName()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get window name: %v\n", err)
				windowName = "default"
			}
		}

		// Complete and archive the session handoff
		// This validates unfilled sections, prompts for completion, then archives
		if err := completeAndArchiveHandoff(projectDir, windowName); err != nil {
			// Only warn - not all sessions will have active handoffs (pre-active-pattern sessions)
			fmt.Fprintf(os.Stderr, "Warning: failed to complete/archive session handoff: %v\n", err)
		}
	}

	// End the session
	ended, err := store.End()
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Log the session end
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.ended",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"goal":          ended.Goal,
			"started_at":    ended.StartedAt.Format(session.TimeFormat),
			"duration":      duration.String(),
			"spawn_count":   spawnCount,
			"active_at_end": activeCount,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Session ended: %s\n", ended.Goal)
	fmt.Printf("  Duration:  %s\n", formatSessionDuration(duration))
	fmt.Printf("  Spawns:    %d total\n", spawnCount)

	if activeCount > 0 {
		fmt.Printf("\n⚠️  %d agent(s) still active. Use 'orch status' to monitor.\n", activeCount)
	}

	// Show checkpoint advice based on session duration using orchestrator thresholds
	orchThresholds := session.DefaultOrchestratorThresholds()
	if duration >= orchThresholds.Max {
		fmt.Printf("\n⛔ Session exceeded %s checkpoint max.\n", formatSessionDuration(orchThresholds.Max))
		fmt.Println("   Consider shorter sessions to maintain quality.")
	} else if duration >= orchThresholds.Strong {
		fmt.Printf("\n🟡 Session was %s+. Good to hand off, but review quality of late work.\n", formatSessionDuration(orchThresholds.Strong))
	}

	return nil
}

// InvestigationPromotionThreshold is the count above which session end will warn.
// Gates accumulation of promotion candidates that need triage.
const InvestigationPromotionThreshold = 5

// InvestigationPromotionItem represents a single investigation promotion candidate.
type InvestigationPromotionItem struct {
	File       string `json:"file"`
	Title      string `json:"title"`
	AgeDays    int    `json:"age_days"`
	Suggestion string `json:"suggestion"`
}

// InvestigationPromotionResult holds the JSON output from kb reflect --type investigation-promotion.
type InvestigationPromotionResult struct {
	InvestigationPromotion []InvestigationPromotionItem `json:"investigation_promotion"`
}

// checkInvestigationPromotions runs kb reflect --type investigation-promotion --format json
// and returns the count of promotion candidates. Returns 0 and logs warning on error.
func checkInvestigationPromotions() int {
	cmd := exec.Command("kb", "reflect", "--type", "investigation-promotion", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		// kb reflect may not be available or may fail - not critical, just skip
		return 0
	}

	var result InvestigationPromotionResult
	if err := json.Unmarshal(output, &result); err != nil {
		// Parse error - skip silently
		return 0
	}

	return len(result.InvestigationPromotion)
}

// gateInvestigationPromotions checks for accumulated investigation promotion candidates
// and prompts user to triage if above threshold. Returns error if user aborts.
// This is a gate at session end to prevent accumulation of promotion candidates.
func gateInvestigationPromotions() error {
	count := checkInvestigationPromotions()
	if count <= InvestigationPromotionThreshold {
		return nil // Below threshold, proceed
	}

	fmt.Println()
	fmt.Printf("⚠️  INVESTIGATION PROMOTION BACKLOG\n")
	fmt.Printf("   %d investigations need promotion review (threshold: %d)\n", count, InvestigationPromotionThreshold)
	fmt.Printf("   Run 'kb reflect --type investigation-promotion' to triage.\n")
	fmt.Println()

	// Prompt user to confirm proceeding
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("   Continue ending session anyway? (y/N): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("   Session end aborted. Please triage investigation promotions first.")
		return fmt.Errorf("session end aborted: investigation promotion backlog needs triage")
	}

	return nil
}
