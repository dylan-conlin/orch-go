package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSketchybarHealthParity verifies the bash event provider's health computation
// matches the Go ComputeDaemonHealth logic for various daemon states.
// This is the integration test between the sketchybar widget and the daemon.
func TestSketchybarHealthParity(t *testing.T) {
	// Verify jq is available (required by event provider)
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not available, skipping sketchybar integration test")
	}

	now := time.Now()

	tests := []struct {
		name           string
		status         DaemonStatus
		wantHealth     string // worst signal level from Go
		wantActive     int
		wantMax        int
		wantReady      int
		wantStatus     string
		wantCompCount  string
		wantCompThresh string
	}{
		{
			name: "all_green",
			status: DaemonStatus{
				PID:    1234,
				Status: "running",
				Capacity: CapacityStatus{
					Max: 5, Active: 2, Available: 3,
				},
				LastPoll:   now.Add(-30 * time.Second),
				ReadyCount: 5,
				PhaseTimeout:      &PhaseTimeoutSnapshot{UnresponsiveCount: 0, LastCheck: now},
				QuestionDetection: &QuestionDetectionSnapshot{QuestionCount: 0, LastCheck: now},
			},
			wantHealth:     "green",
			wantActive:     2,
			wantMax:        5,
			wantReady:      5,
			wantStatus:     "running",
			wantCompCount:  "0",
			wantCompThresh: "",
		},
		{
			name: "verification_yellow",
			status: DaemonStatus{
				PID:    1234,
				Status: "running",
				Capacity: CapacityStatus{
					Max: 5, Active: 2, Available: 3,
				},
				LastPoll:   now.Add(-30 * time.Second),
				ReadyCount: 5,
			},
			wantHealth:     "yellow",
			wantActive:     2,
			wantMax:        5,
			wantReady:      5,
			wantStatus:     "running",
			wantCompCount:  "0",
			wantCompThresh: "",
		},
		{
			name: "verification_paused_red",
			status: DaemonStatus{
				PID:    1234,
				Status: "paused",
				Capacity: CapacityStatus{
					Max: 5, Active: 0, Available: 5,
				},
				LastPoll:   now.Add(-30 * time.Second),
				ReadyCount: 10,
			},
			wantHealth:     "red",
			wantActive:     0,
			wantMax:        5,
			wantReady:      10,
			wantStatus:     "paused",
			wantCompCount:  "0",
			wantCompThresh: "",
		},
		{
			name: "capacity_saturated_red",
			status: DaemonStatus{
				PID:    1234,
				Status: "running",
				Capacity: CapacityStatus{
					Max: 5, Active: 5, Available: 0,
				},
				LastPoll:   now.Add(-30 * time.Second),
				ReadyCount: 15,
			},
			wantHealth:     "red",
			wantActive:     5,
			wantMax:        5,
			wantReady:      15,
			wantStatus:     "running",
			wantCompCount:  "0",
			wantCompThresh: "",
		},
		{
			name: "capacity_yellow_80pct",
			status: DaemonStatus{
				PID:    1234,
				Status: "running",
				Capacity: CapacityStatus{
					Max: 5, Active: 4, Available: 1,
				},
				LastPoll:   now.Add(-30 * time.Second),
				ReadyCount: 3,
			},
			wantHealth:     "yellow",
			wantActive:     4,
			wantMax:        5,
			wantReady:      3,
			wantStatus:     "running",
			wantCompCount:  "0",
			wantCompThresh: "",
		},
		{
			name: "unresponsive_red",
			status: DaemonStatus{
				PID:    1234,
				Status: "running",
				Capacity: CapacityStatus{
					Max: 5, Active: 3, Available: 2,
				},
				LastPoll:   now.Add(-30 * time.Second),
				ReadyCount: 5,
				PhaseTimeout: &PhaseTimeoutSnapshot{
					UnresponsiveCount: 2,
					LastCheck:         now,
				},
			},
			wantHealth:     "red",
			wantActive:     3,
			wantMax:        5,
			wantReady:      5,
			wantStatus:     "running",
			wantCompCount:  "0",
			wantCompThresh: "",
		},
		{
			name: "questions_yellow",
			status: DaemonStatus{
				PID:    1234,
				Status: "running",
				Capacity: CapacityStatus{
					Max: 5, Active: 1, Available: 4,
				},
				LastPoll:   now.Add(-30 * time.Second),
				ReadyCount: 2,
				QuestionDetection: &QuestionDetectionSnapshot{
					QuestionCount: 2,
					LastCheck:     now,
				},
			},
			wantHealth:     "yellow",
			wantActive:     1,
			wantMax:        5,
			wantReady:      2,
			wantStatus:     "running",
			wantCompCount:  "0",
			wantCompThresh: "",
		},
		{
			name: "comprehension_threshold_red",
			status: DaemonStatus{
				PID:    1234,
				Status: "running",
				Capacity: CapacityStatus{
					Max: 5, Active: 2, Available: 3,
				},
				LastPoll:   now.Add(-30 * time.Second),
				ReadyCount: 4,
				Comprehension: &ComprehensionSnapshot{
					Count:     5,
					Threshold: 5,
				},
			},
			wantHealth:     "red",
			wantActive:     2,
			wantMax:        5,
			wantReady:      4,
			wantStatus:     "running",
			wantCompCount:  "5",
			wantCompThresh: "5",
		},
		{
			name: "queue_depth_red",
			status: DaemonStatus{
				PID:    1234,
				Status: "running",
				Capacity: CapacityStatus{
					Max: 5, Active: 1, Available: 4,
				},
				LastPoll:   now.Add(-30 * time.Second),
				ReadyCount: 60,
			},
			wantHealth:     "red",
			wantActive:     1,
			wantMax:        5,
			wantReady:      60,
			wantStatus:     "running",
			wantCompCount:  "0",
			wantCompThresh: "",
		},
	}

	// Build the bash health computation script (extracted from orch_status.sh)
	bashScript := buildHealthComputeScript()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Verify Go health computation
			goSummary := ComputeDaemonHealth(&tt.status, now)
			goWorst := worstSignalLevel(goSummary)
			if goWorst != tt.wantHealth {
				t.Errorf("Go health: want %s, got %s", tt.wantHealth, goWorst)
				for _, sig := range goSummary.Signals {
					t.Logf("  %s: %s (%s)", sig.Name, sig.Level, sig.Detail)
				}
			}

			// Step 2: Write synthetic daemon-status.json
			tmpDir := t.TempDir()
			statusPath := filepath.Join(tmpDir, "daemon-status.json")
			data, err := json.MarshalIndent(tt.status, "", "  ")
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if err := os.WriteFile(statusPath, data, 0644); err != nil {
				t.Fatalf("write: %v", err)
			}

			// Step 3: Run bash health computation
			cmd := exec.Command("bash", "-c", bashScript)
			cmd.Env = append(os.Environ(),
				"DAEMON_STATUS="+statusPath,
				fmt.Sprintf("NOW_EPOCH=%d", now.Unix()),
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("bash script failed: %v\nOutput: %s", err, string(out))
			}

			// Step 4: Parse bash output and compare
			bashVars := parseBashOutput(string(out))

			if bashVars["HEALTH_LEVEL"] != tt.wantHealth {
				t.Errorf("bash HEALTH_LEVEL: want %s, got %s", tt.wantHealth, bashVars["HEALTH_LEVEL"])
			}
			if bashVars["ACTIVE"] != fmt.Sprintf("%d", tt.wantActive) {
				t.Errorf("bash ACTIVE: want %d, got %s", tt.wantActive, bashVars["ACTIVE"])
			}
			if bashVars["MAX"] != fmt.Sprintf("%d", tt.wantMax) {
				t.Errorf("bash MAX: want %d, got %s", tt.wantMax, bashVars["MAX"])
			}
			if bashVars["READY"] != fmt.Sprintf("%d", tt.wantReady) {
				t.Errorf("bash READY: want %d, got %s", tt.wantReady, bashVars["READY"])
			}
			if bashVars["STATUS"] != tt.wantStatus {
				t.Errorf("bash STATUS: want %s, got %s", tt.wantStatus, bashVars["STATUS"])
			}
			if bashVars["COMPREHENSION_COUNT"] != tt.wantCompCount {
				t.Errorf("bash COMPREHENSION_COUNT: want %s, got %s", tt.wantCompCount, bashVars["COMPREHENSION_COUNT"])
			}
			if bashVars["COMPREHENSION_THRESHOLD"] != tt.wantCompThresh {
				t.Errorf("bash COMPREHENSION_THRESHOLD: want %s, got %s", tt.wantCompThresh, bashVars["COMPREHENSION_THRESHOLD"])
			}

			// Step 5: Verify parity between Go and bash
			if goWorst != bashVars["HEALTH_LEVEL"] {
				t.Errorf("PARITY MISMATCH: Go=%s, bash=%s", goWorst, bashVars["HEALTH_LEVEL"])
				for _, sig := range goSummary.Signals {
					t.Logf("  Go signal %s: %s", sig.Name, sig.Level)
				}
			}
		})
	}
}

// TestSketchybarDaemonNotRunning verifies the widget shows "off" when daemon-status.json
// is missing (daemon not running).
func TestSketchybarDaemonNotRunning(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not available")
	}

	bashScript := buildHealthComputeScript()

	cmd := exec.Command("bash", "-c", bashScript)
	cmd.Env = append(os.Environ(),
		"DAEMON_STATUS=/tmp/nonexistent-daemon-status-test.json",
		fmt.Sprintf("NOW_EPOCH=%d", time.Now().Unix()),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash script failed: %v\nOutput: %s", err, string(out))
	}

	vars := parseBashOutput(string(out))
	if vars["STATUS"] != "off" {
		t.Errorf("STATUS: want 'off' when daemon not running, got %q", vars["STATUS"])
	}
	if vars["ACTIVE"] != "0" {
		t.Errorf("ACTIVE: want 0 when daemon not running, got %s", vars["ACTIVE"])
	}
	if vars["MAX"] != "0" {
		t.Errorf("MAX: want 0 when daemon not running, got %s", vars["MAX"])
	}
}

// TestSketchybarCapacityCacheParsing verifies account usage extraction from capacity-cache.json.
func TestSketchybarCapacityCacheParsing(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not available")
	}

	tmpDir := t.TempDir()

	// Write synthetic daemon-status.json (minimal, running)
	statusPath := filepath.Join(tmpDir, "daemon-status.json")
	status := DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max: 5, Active: 1, Available: 4,
		},
		LastPoll:   time.Now().Add(-10 * time.Second),
		ReadyCount: 3,
	}
	data, _ := json.MarshalIndent(status, "", "  ")
	os.WriteFile(statusPath, data, 0644)

	// Write synthetic capacity-cache.json
	capPath := filepath.Join(tmpDir, "capacity-cache.json")
	capData := `{
  "fetched_at": "2026-03-24T10:00:00Z",
  "accounts": [
    {
      "name": "work",
      "email": "test@example.com",
      "is_default": true,
      "capacity": {
        "FiveHourUsed": 65,
        "FiveHourRemaining": 35,
        "SevenDayUsed": 43,
        "SevenDayRemaining": 57,
        "Email": "test@example.com"
      }
    }
  ]
}`
	os.WriteFile(capPath, []byte(capData), 0644)

	bashScript := buildCapacityScript()

	cmd := exec.Command("bash", "-c", bashScript)
	cmd.Env = append(os.Environ(),
		"DAEMON_STATUS="+statusPath,
		"CAPACITY_CACHE="+capPath,
		fmt.Sprintf("NOW_EPOCH=%d", time.Now().Unix()),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash script failed: %v\nOutput: %s", err, string(out))
	}

	vars := parseBashOutput(string(out))
	if vars["ACCOUNT_5H"] != "35" {
		t.Errorf("ACCOUNT_5H: want 35, got %q", vars["ACCOUNT_5H"])
	}
	if vars["ACCOUNT_7D"] != "57" {
		t.Errorf("ACCOUNT_7D: want 57, got %q", vars["ACCOUNT_7D"])
	}
}

// TestSketchybarWorstLevelFunction tests the bash worst_level function directly.
func TestSketchybarWorstLevelFunction(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	tests := []struct {
		a, b, want string
	}{
		{"green", "green", "green"},
		{"green", "yellow", "yellow"},
		{"yellow", "green", "yellow"},
		{"green", "red", "red"},
		{"red", "green", "red"},
		{"yellow", "red", "red"},
		{"red", "yellow", "red"},
		{"red", "red", "red"},
	}

	script := `
worst_level() {
  local a="$1" b="$2"
  if [ "$a" = "red" ] || [ "$b" = "red" ]; then
    echo "red"
  elif [ "$a" = "yellow" ] || [ "$b" = "yellow" ]; then
    echo "yellow"
  else
    echo "green"
  fi
}
`

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.a, tt.b), func(t *testing.T) {
			cmd := exec.Command("bash", "-c", script+fmt.Sprintf("worst_level %s %s", tt.a, tt.b))
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("bash failed: %v", err)
			}
			got := strings.TrimSpace(string(out))
			if got != tt.want {
				t.Errorf("worst_level(%s, %s): want %s, got %s", tt.a, tt.b, tt.want, got)
			}
		})
	}
}

// worstSignalLevel returns the worst level across all signals.
func worstSignalLevel(summary *DaemonHealthSummary) string {
	if summary == nil {
		return "green"
	}
	worst := "green"
	for _, sig := range summary.Signals {
		switch sig.Level {
		case "red":
			return "red"
		case "yellow":
			worst = "yellow"
		}
	}
	return worst
}

// buildHealthComputeScript extracts the health computation from orch_status.sh
// into a standalone script that reads from DAEMON_STATUS env var and prints results.
func buildHealthComputeScript() string {
	return `#!/bin/bash
# Extracted health computation from orch_status.sh for testing

worst_level() {
  local a="$1" b="$2"
  if [ "$a" = "red" ] || [ "$b" = "red" ]; then
    echo "red"
  elif [ "$a" = "yellow" ] || [ "$b" = "yellow" ]; then
    echo "yellow"
  else
    echo "green"
  fi
}

ACTIVE=0
MAX=0
AVAILABLE=0
READY=0
STATUS="off"
HEALTH_LEVEL="green"
	UNRESPONSIVE=0
	QUESTIONS=0
	VERIFICATION_REMAINING=""
	COMPREHENSION_COUNT=0
	COMPREHENSION_THRESHOLD=""

	if [ -f "$DAEMON_STATUS" ]; then
  DS=$(cat "$DAEMON_STATUS" 2>/dev/null)

  if [ -n "$DS" ] && echo "$DS" | jq -e '.status' >/dev/null 2>&1; then
    ACTIVE=$(echo "$DS" | jq -r '.capacity.active // 0')
    MAX=$(echo "$DS" | jq -r '.capacity.max // 0')
    AVAILABLE=$(echo "$DS" | jq -r '.capacity.available // 0')
    READY=$(echo "$DS" | jq -r '.ready_count // 0')
    STATUS=$(echo "$DS" | jq -r '.status // "unknown"')

    UNRESPONSIVE=$(echo "$DS" | jq -r '.phase_timeout.unresponsive_count // 0')
	    QUESTIONS=$(echo "$DS" | jq -r '.question_detection.question_count // 0')
	    COMPREHENSION_COUNT=$(echo "$DS" | jq -r '.comprehension.count // 0')

	    COMPREHENSION_THRESHOLD=$(echo "$DS" | jq -r '
	      if .comprehension then (.comprehension.threshold | tostring)
	      else "" end
	    ')

	    VERIFICATION_REMAINING=$(echo "$DS" | jq -r '
      if .verification then (.verification.remaining_before_pause | tostring)
      else "" end
    ')

    # 1. Liveness: use NOW_EPOCH from env for deterministic testing
    LAST_POLL_ISO=$(echo "$DS" | jq -r '.last_poll // empty' | cut -c1-19)
    LAST_POLL_EPOCH=""
    if [ -n "$LAST_POLL_ISO" ]; then
      LAST_POLL_EPOCH=$(date -j -f "%Y-%m-%dT%H:%M:%S" "$LAST_POLL_ISO" "+%s" 2>/dev/null)
    fi
    LIVENESS_LEVEL="green"
    if [ -n "$LAST_POLL_EPOCH" ] && [ -n "$NOW_EPOCH" ]; then
      POLL_AGE=$((NOW_EPOCH - LAST_POLL_EPOCH))
      if [ "$POLL_AGE" -gt 600 ]; then
        LIVENESS_LEVEL="red"
      elif [ "$POLL_AGE" -gt 120 ]; then
        LIVENESS_LEVEL="yellow"
      fi
    fi

    # 2. Capacity
    CAP_LEVEL="green"
    if [ "$MAX" -gt 0 ]; then
      if [ "$AVAILABLE" -eq 0 ] && [ "$READY" -gt 0 ]; then
        CAP_LEVEL="red"
      elif [ "$((ACTIVE * 100 / MAX))" -ge 80 ]; then
        CAP_LEVEL="yellow"
      fi
    fi

    # 3. Queue depth
    QUEUE_LEVEL="green"
    if [ "$READY" -gt 50 ]; then
      QUEUE_LEVEL="red"
    elif [ "$READY" -ge 20 ]; then
      QUEUE_LEVEL="yellow"
    fi

    # 4. Verification
    VERIFY_LEVEL="green"
    IS_PAUSED=$(echo "$DS" | jq -r '.verification.is_paused // false')
    if [ "$IS_PAUSED" = "true" ]; then
      VERIFY_LEVEL="red"
    elif [ -n "$VERIFICATION_REMAINING" ] && [ "$VERIFICATION_REMAINING" != "" ]; then
      if [ "$VERIFICATION_REMAINING" -le 2 ] 2>/dev/null; then
        VERIFY_LEVEL="yellow"
      fi
    fi

    # 5. Unresponsive
    UNRESPONSIVE_LEVEL="green"
    if [ "$UNRESPONSIVE" -gt 1 ]; then
      UNRESPONSIVE_LEVEL="red"
    elif [ "$UNRESPONSIVE" -eq 1 ]; then
      UNRESPONSIVE_LEVEL="yellow"
    fi

    # 6. Questions
	    QUESTION_LEVEL="green"
	    if [ "$QUESTIONS" -gt 2 ]; then
	      QUESTION_LEVEL="red"
	    elif [ "$QUESTIONS" -ge 1 ]; then
	      QUESTION_LEVEL="yellow"
	    fi

	    COMPREHENSION_LEVEL="green"
	    if [ -n "$COMPREHENSION_THRESHOLD" ] && [ "$COMPREHENSION_THRESHOLD" != "" ] && [ "$COMPREHENSION_COUNT" -ge "$COMPREHENSION_THRESHOLD" ] 2>/dev/null; then
	      COMPREHENSION_LEVEL="red"
	    elif [ "$COMPREHENSION_COUNT" -gt 0 ]; then
	      COMPREHENSION_LEVEL="yellow"
	    fi

	    # Worst across all
	    HEALTH_LEVEL="green"
	    for LVL in "$LIVENESS_LEVEL" "$CAP_LEVEL" "$QUEUE_LEVEL" "$VERIFY_LEVEL" "$UNRESPONSIVE_LEVEL" "$QUESTION_LEVEL" "$COMPREHENSION_LEVEL"; do
	      HEALTH_LEVEL=$(worst_level "$HEALTH_LEVEL" "$LVL")
	    done
  fi
fi

echo "ACTIVE=$ACTIVE"
echo "MAX=$MAX"
echo "AVAILABLE=$AVAILABLE"
echo "READY=$READY"
echo "STATUS=$STATUS"
echo "HEALTH_LEVEL=$HEALTH_LEVEL"
echo "UNRESPONSIVE=$UNRESPONSIVE"
echo "QUESTIONS=$QUESTIONS"
echo "VERIFICATION_REMAINING=$VERIFICATION_REMAINING"
echo "COMPREHENSION_COUNT=$COMPREHENSION_COUNT"
echo "COMPREHENSION_THRESHOLD=$COMPREHENSION_THRESHOLD"
`
}

// buildCapacityScript extends the health script to also parse capacity-cache.json.
func buildCapacityScript() string {
	return buildHealthComputeScript() + `
ACCOUNT_5H=""
ACCOUNT_7D=""

if [ -f "$CAPACITY_CACHE" ]; then
  CC=$(cat "$CAPACITY_CACHE" 2>/dev/null)
  if [ -n "$CC" ] && echo "$CC" | jq -e '.accounts' >/dev/null 2>&1; then
    ACCT=$(echo "$CC" | jq -r '(.accounts[] | select(.is_default == true)) // .accounts[0] // empty')
    if [ -n "$ACCT" ]; then
      ACCOUNT_5H=$(echo "$ACCT" | jq -r '.capacity.FiveHourRemaining // empty' | xargs printf "%.0f" 2>/dev/null)
      ACCOUNT_7D=$(echo "$ACCT" | jq -r '.capacity.SevenDayRemaining // empty' | xargs printf "%.0f" 2>/dev/null)
    fi
  fi
fi

echo "ACCOUNT_5H=$ACCOUNT_5H"
echo "ACCOUNT_7D=$ACCOUNT_7D"
`
}

// parseBashOutput parses KEY=VALUE lines from bash output into a map.
func parseBashOutput(output string) map[string]string {
	vars := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "="); idx > 0 {
			key := line[:idx]
			val := line[idx+1:]
			vars[key] = val
		}
	}
	return vars
}
