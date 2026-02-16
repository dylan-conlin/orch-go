package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	BeadsID   string
	IssueType string
	Title     string
}

// GetUnverifiedTier1Work returns all Tier 1 work (features/bugs/decisions) that has
// checkpoints but hasn't passed both verification gates. Only considers OPEN issues.
//
// Delegates to verify.ListUnverifiedWork() (the canonical source of truth for
// verification state) and filters to Tier 1 items only.
func GetUnverifiedTier1Work() ([]UnverifiedWork, error) {
	allUnverified, err := verify.ListUnverifiedWork()
	if err != nil {
		return nil, err
	}

	var tier1 []UnverifiedWork
	for _, item := range allUnverified {
		if item.Tier == 1 {
			tier1 = append(tier1, UnverifiedWork{
				BeadsID:   item.BeadsID,
				IssueType: item.IssueType,
				Title:     item.Title,
			})
		}
	}

	return tier1, nil
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
