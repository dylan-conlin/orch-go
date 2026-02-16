package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// CheckVerificationGate enforces that no unverified Tier 1 work exists before spawning.
// This prevents the cascade pattern where each agent's fix alters ground truth without verification.
//
// From verifiability-first decision (Phase 3): Block spawn if unverified Tier 1 work exists.
// Simple serialization model: daemon blocks new spawns while unverified Tier 1 work exists.
// Independent parallel work can use --bypass-verification to override.
//
// Returns nil if spawn is allowed, or an error if unverified work blocks the spawn.
func CheckVerificationGate(bypassVerification bool, bypassReason string) error {
	// If bypass flag is set, allow spawn but log the bypass
	if bypassVerification {
		if bypassReason == "" {
			return fmt.Errorf("--bypass-verification requires --bypass-reason with justification")
		}
		LogVerificationBypass(bypassReason)
		return nil
	}

	// Check for unverified Tier 1 work
	unverified, err := GetUnverifiedTier1Work()
	if err != nil {
		// Log warning but don't block spawn on infrastructure error
		fmt.Fprintf(os.Stderr, "Warning: verification gate check failed: %v\n", err)
		return nil
	}

	if len(unverified) > 0 {
		return showVerificationGateBlocked(unverified)
	}

	return nil
}

// UnverifiedWork represents a Tier 1 deliverable awaiting verification.
type UnverifiedWork struct {
	BeadsID     string
	IssueType   string
	Title       string
	CompletedAt time.Time
}

// GetUnverifiedTier1Work returns all Tier 1 work (features/bugs/decisions) that completed
// but hasn't passed both verification gates.
//
// Algorithm:
// 1. Get all closed issues with status=closed
// 2. Filter to Tier 1 work (features/bugs/decisions)
// 3. Check verification checkpoints for each
// 4. Return those without both gate1 and gate2 complete
func GetUnverifiedTier1Work() ([]UnverifiedWork, error) {
	// Get all recently closed issues (last 30 days to avoid scanning full history)
	// This uses beads CLI to list closed issues
	closedIssues, err := getRecentClosedIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to get closed issues: %w", err)
	}

	var unverified []UnverifiedWork

	for _, issue := range closedIssues {
		// Skip non-Tier 1 work
		if !checkpoint.IsTier1Work(issue.IssueType) {
			continue
		}

		// Check if verification checkpoint exists with both gates complete
		cp, err := checkpoint.HasCheckpoint(issue.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check checkpoint for %s: %w", issue.ID, err)
		}

		// If no checkpoint exists, or either gate is incomplete, it's unverified
		if cp == nil || !cp.Gate1Complete || !cp.Gate2Complete {
			unverified = append(unverified, UnverifiedWork{
				BeadsID:     issue.ID,
				IssueType:   issue.IssueType,
				Title:       issue.Title,
				CompletedAt: issue.ClosedAt,
			})
		}
	}

	return unverified, nil
}

// ClosedIssue represents a closed beads issue.
type ClosedIssue struct {
	ID        string
	Title     string
	IssueType string
	ClosedAt  time.Time
}

// getRecentClosedIssues retrieves closed issues from the last 30 days.
// Uses beads CLI to list issues with status=closed.
func getRecentClosedIssues() ([]ClosedIssue, error) {
	// Get all closed issues from beads
	// We'll use the verify package's GetIssue function for individual issues,
	// but first we need to get a list of issue IDs.
	// For now, we'll check all issues mentioned in verification checkpoints
	// since those are the ones that went through orch complete.

	checkpoints, err := checkpoint.ReadCheckpoints()
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoints: %w", err)
	}

	// Deduplicate beads IDs from checkpoints
	beadsIDSet := make(map[string]bool)
	for _, cp := range checkpoints {
		beadsIDSet[cp.BeadsID] = true
	}

	var closedIssues []ClosedIssue
	cutoff := time.Now().AddDate(0, 0, -30) // Last 30 days

	// Fetch each issue and check if it's closed and recent
	for beadsID := range beadsIDSet {
		issue, err := verify.GetIssue(beadsID)
		if err != nil {
			// Skip issues we can't fetch (may have been deleted)
			continue
		}

		if issue.Status != "closed" {
			continue
		}

		// Parse close timestamp from comments or use checkpoint timestamp as proxy
		closedAt := getIssueClosedTime(beadsID, checkpoints)
		if closedAt.Before(cutoff) {
			continue
		}

		closedIssues = append(closedIssues, ClosedIssue{
			ID:        issue.ID,
			Title:     issue.Title,
			IssueType: issue.IssueType,
			ClosedAt:  closedAt,
		})
	}

	return closedIssues, nil
}

// getIssueClosedTime returns the time an issue was closed.
// Uses the latest checkpoint timestamp as a proxy for close time.
func getIssueClosedTime(beadsID string, checkpoints []checkpoint.Checkpoint) time.Time {
	var latest time.Time
	for _, cp := range checkpoints {
		if cp.BeadsID == beadsID && cp.Timestamp.After(latest) {
			latest = cp.Timestamp
		}
	}
	if latest.IsZero() {
		return time.Now() // Fallback if no checkpoint found
	}
	return latest
}

// LogVerificationBypass logs a verification bypass event to the violations log.
func LogVerificationBypass(reason string) {
	// Log to events.jsonl
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "spawn.verification_bypassed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"reason": reason,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log verification bypass: %v\n", err)
	}

	// Also log to dedicated violations log
	violationsPath := getVerificationViolationsPath()
	if err := os.MkdirAll(filepath.Dir(violationsPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create violations log directory: %v\n", err)
		return
	}

	f, err := os.OpenFile(violationsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open violations log: %v\n", err)
		return
	}
	defer f.Close()

	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("%s\tverification_bypassed\t%s\n", timestamp, reason)
	if _, err := f.WriteString(line); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write to violations log: %v\n", err)
	}
}

// getVerificationViolationsPath returns the path to the verification violations log.
func getVerificationViolationsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/metrics/verification-violations.log"
	}
	return filepath.Join(home, ".orch", "metrics", "verification-violations.log")
}

// showVerificationGateBlocked displays the error message when spawn is blocked.
func showVerificationGateBlocked(unverified []UnverifiedWork) error {
	fmt.Fprintf(os.Stderr, `
┌─────────────────────────────────────────────────────────────────────────────┐
│  🚫 VERIFICATION GATE BLOCKED                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  Cannot spawn new agent: unverified Tier 1 work exists.                    │
│                                                                             │
│  The verifiability-first constraint prevents autonomous progression        │
│  when completed work hasn't been verified. This blocks the cascade         │
│  pattern where each agent's fix alters ground truth without human          │
│  verification in between.                                                  │
│                                                                             │
│  Unverified work:                                                           │
`)

	for i, work := range unverified {
		if i >= 5 {
			fmt.Fprintf(os.Stderr, "│    ... and %d more                                                           │\n", len(unverified)-5)
			break
		}
		truncatedTitle := work.Title
		if len(truncatedTitle) > 50 {
			truncatedTitle = truncatedTitle[:47] + "..."
		}
		fmt.Fprintf(os.Stderr, "│    • %s (%s)%s│\n",
			work.BeadsID,
			work.IssueType,
			spaces(66-len(work.BeadsID)-len(work.IssueType)-7))
		fmt.Fprintf(os.Stderr, "│      %s%s│\n",
			truncatedTitle,
			spaces(71-len(truncatedTitle)))
	}

	fmt.Fprintf(os.Stderr, `│                                                                             │
│  Next steps:                                                                │
│    1. Verify completed work: orch verify complete <beads-id> --explain "..." │
│    2. Or bypass for independent work: orch spawn --bypass-verification ... │
│       (requires --bypass-reason "justification")                            │
│                                                                             │
│  See: .kb/decisions/2026-02-14-verifiability-first-hard-constraint.md      │
└─────────────────────────────────────────────────────────────────────────────┘

`)

	return fmt.Errorf("spawn blocked: %d unverified Tier 1 deliverable(s) exist", len(unverified))
}

// spaces returns a string of n spaces for formatting.
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("%*s", n, "")
}
