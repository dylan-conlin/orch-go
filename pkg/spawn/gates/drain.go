package gates

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// CheckDrainGate enforces that no reviewable completions exist before spawning.
// This prevents the pattern where new work is spawned while completed work sits
// unreviewed, producing motion without value extraction.
//
// Reviewable = workspace with SYNTHESIS.md or light-tier Phase: Complete,
// counted via `bd list` issues with daemon:ready-review label.
//
// Returns nil if spawn is allowed, or an error if reviewable work blocks the spawn.
func CheckDrainGate(bypass bool, bypassReason string, daemonDriven bool) error {
	// Daemon-driven spawns are exempt — the daemon has its own verification pause
	if daemonDriven {
		return nil
	}

	if bypass {
		if bypassReason == "" {
			return fmt.Errorf("--force-drain-bypass requires --bypass-reason with justification")
		}
		LogDrainBypass(bypassReason)
		return nil
	}

	count, err := CountReviewableCompletions()
	if err != nil {
		// Don't block spawn on infrastructure error
		fmt.Fprintf(os.Stderr, "Warning: drain gate check failed: %v\n", err)
		return nil
	}

	if count > 0 {
		return showDrainGateBlocked(count)
	}

	return nil
}

// CountReviewableCompletions counts open issues with the daemon:ready-review label.
// This is the same signal the daemon uses for its verification pause.
func CountReviewableCompletions() (int, error) {
	cmd := exec.Command("bd", "list", "--no-db")
	cmd.Env = append(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("bd list failed: %w", err)
	}

	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "daemon:ready-review") && strings.Contains(line, " open ") {
			count++
		}
	}
	return count, nil
}

// LogDrainBypass logs a drain gate bypass event.
func LogDrainBypass(reason string) {
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "gate.bypass",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"gate":   "drain",
			"reason": reason,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log drain bypass: %v\n", err)
	}

	// Also log to violations log
	violationsPath := getDrainViolationsPath()
	if err := os.MkdirAll(filepath.Dir(violationsPath), 0755); err != nil {
		return
	}
	f, err := os.OpenFile(violationsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("%s\tdrain_bypassed\t%s\n", timestamp, reason)
	f.WriteString(line)
}

func getDrainViolationsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/metrics/drain-violations.log"
	}
	return filepath.Join(home, ".orch", "metrics", "drain-violations.log")
}

func showDrainGateBlocked(count int) error {
	fmt.Fprintf(os.Stderr, `
┌─────────────────────────────────────────────────────────────────────────────┐
│  🚫 DRAIN GATE BLOCKED                                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│  Cannot spawn new agent: %d completion(s) awaiting review.                 │
│                                                                             │
│  Review queue must be empty before spawning new work.                       │
│  Completions without review produce motion without value extraction.        │
│                                                                             │
│  Next steps:                                                                │
│    1. Review completions: orch review                                       │
│    2. Complete each: orch complete <beads-id>                               │
│    3. Or bypass: orch spawn --force-drain-bypass --reason "..."             │
│                                                                             │
│  See: closure-discipline thread                                             │
└─────────────────────────────────────────────────────────────────────────────┘

`, count)

	return fmt.Errorf("spawn blocked: %d reviewable completion(s) exist — drain before spawning", count)
}
