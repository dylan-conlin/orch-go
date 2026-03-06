package daemon

import (
	"fmt"
	"time"
)

// DaemonHealthSignal represents a single health signal with a traffic-light level.
type DaemonHealthSignal struct {
	Name   string `json:"name"`
	Level  string `json:"level"`  // "green", "yellow", "red"
	Detail string `json:"detail"` // Human-readable description
}

// DaemonHealthSummary holds all 6 health signals computed from daemon-status.json.
type DaemonHealthSummary struct {
	Signals []DaemonHealthSignal `json:"signals"`
}

// ComputeDaemonHealth derives 6 health signals from the daemon status snapshot.
// Signals: Daemon Liveness, Capacity, Queue Depth, Verification, Unresponsive, Questions.
func ComputeDaemonHealth(status *DaemonStatus, now time.Time) *DaemonHealthSummary {
	if status == nil {
		return nil
	}

	return &DaemonHealthSummary{
		Signals: []DaemonHealthSignal{
			computeLiveness(status, now),
			computeCapacity(status),
			computeQueueDepth(status),
			computeVerification(status),
			computeUnresponsive(status),
			computeQuestions(status),
		},
	}
}

// computeLiveness checks how recently the daemon polled.
// Green: <2min, Yellow: 2-10min, Red: >10min
func computeLiveness(status *DaemonStatus, now time.Time) DaemonHealthSignal {
	age := now.Sub(status.LastPoll)
	sig := DaemonHealthSignal{Name: "Daemon Liveness"}

	switch {
	case age > 10*time.Minute:
		sig.Level = "red"
		sig.Detail = fmt.Sprintf("last poll %s ago", humanDuration(age))
	case age > 2*time.Minute:
		sig.Level = "yellow"
		sig.Detail = fmt.Sprintf("last poll %s ago", humanDuration(age))
	default:
		sig.Level = "green"
		sig.Detail = "polling normally"
	}
	return sig
}

// computeCapacity checks agent pool utilization.
// Green: <80%, Yellow: >=80%, Red: saturated (100%) with queued work
func computeCapacity(status *DaemonStatus) DaemonHealthSignal {
	sig := DaemonHealthSignal{Name: "Capacity"}

	if status.Capacity.Max == 0 {
		sig.Level = "green"
		sig.Detail = "no capacity limit"
		return sig
	}

	utilization := float64(status.Capacity.Active) / float64(status.Capacity.Max)
	switch {
	case status.Capacity.Available == 0 && status.ReadyCount > 0:
		sig.Level = "red"
		sig.Detail = fmt.Sprintf("%d/%d slots used, %d queued", status.Capacity.Active, status.Capacity.Max, status.ReadyCount)
	case utilization >= 0.8:
		sig.Level = "yellow"
		sig.Detail = fmt.Sprintf("%d/%d slots used", status.Capacity.Active, status.Capacity.Max)
	default:
		sig.Level = "green"
		sig.Detail = fmt.Sprintf("%d/%d slots used", status.Capacity.Active, status.Capacity.Max)
	}
	return sig
}

// computeQueueDepth checks the ready issue count.
// Green: <20, Yellow: 20-50, Red: >50
func computeQueueDepth(status *DaemonStatus) DaemonHealthSignal {
	sig := DaemonHealthSignal{Name: "Queue Depth"}

	switch {
	case status.ReadyCount > 50:
		sig.Level = "red"
		sig.Detail = fmt.Sprintf("%d issues ready", status.ReadyCount)
	case status.ReadyCount >= 20:
		sig.Level = "yellow"
		sig.Detail = fmt.Sprintf("%d issues ready", status.ReadyCount)
	default:
		sig.Level = "green"
		sig.Detail = fmt.Sprintf("%d issues ready", status.ReadyCount)
	}
	return sig
}

// computeVerification checks verification gate pressure.
// Green: >2 remaining or not configured, Yellow: 1-2 remaining, Red: paused
func computeVerification(status *DaemonStatus) DaemonHealthSignal {
	sig := DaemonHealthSignal{Name: "Verification"}

	if status.Verification == nil {
		sig.Level = "green"
		sig.Detail = "not configured"
		return sig
	}

	v := status.Verification
	switch {
	case v.IsPaused:
		sig.Level = "red"
		sig.Detail = "paused — verification required"
	case v.RemainingBeforePause <= 2:
		sig.Level = "yellow"
		sig.Detail = fmt.Sprintf("%d completions before pause", v.RemainingBeforePause)
	default:
		sig.Level = "green"
		sig.Detail = fmt.Sprintf("%d completions before pause", v.RemainingBeforePause)
	}
	return sig
}

// computeUnresponsive checks for agents that haven't reported phase.
// Green: 0, Yellow: 1, Red: >1
func computeUnresponsive(status *DaemonStatus) DaemonHealthSignal {
	sig := DaemonHealthSignal{Name: "Unresponsive"}

	count := 0
	if status.PhaseTimeout != nil {
		count = status.PhaseTimeout.UnresponsiveCount
	}

	switch {
	case count > 1:
		sig.Level = "red"
		sig.Detail = fmt.Sprintf("%d agents unresponsive", count)
	case count == 1:
		sig.Level = "yellow"
		sig.Detail = "1 agent unresponsive"
	default:
		sig.Level = "green"
		sig.Detail = "all agents responsive"
	}
	return sig
}

// computeQuestions checks for agents waiting on user input.
// Green: 0, Yellow: 1-2, Red: >2
func computeQuestions(status *DaemonStatus) DaemonHealthSignal {
	sig := DaemonHealthSignal{Name: "Questions"}

	count := 0
	if status.QuestionDetection != nil {
		count = status.QuestionDetection.QuestionCount
	}

	switch {
	case count > 2:
		sig.Level = "red"
		sig.Detail = fmt.Sprintf("%d agents waiting for input", count)
	case count >= 1:
		sig.Level = "yellow"
		sig.Detail = fmt.Sprintf("%d agent(s) waiting for input", count)
	default:
		sig.Level = "green"
		sig.Detail = "no pending questions"
	}
	return sig
}

// humanDuration formats a duration for display.
func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
