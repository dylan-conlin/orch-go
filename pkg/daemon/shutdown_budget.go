package daemon

import "time"

// ShutdownBudget enforces explicit time budgets for daemon shutdown phases.
// The total budget (4s) is derived from launchd's ExitTimeOut (5s) minus a 1s
// safety margin. Each phase has a dedicated allocation to prevent any single
// defer from consuming the entire budget and triggering SIGKILL.
type ShutdownBudget struct {
	Total         time.Duration // Total shutdown budget (launchd 5s - 1s safety)
	Reflection    time.Duration // kb reflect analysis
	StatusCleanup time.Duration // Remove status file, PID lock
	LogFlush      time.Duration // Flush and close daemon log

	start time.Time // Set by Begin()
}

// NewShutdownBudget returns a budget with production defaults.
func NewShutdownBudget() *ShutdownBudget {
	return &ShutdownBudget{
		Total:         4 * time.Second,
		Reflection:    2500 * time.Millisecond,
		StatusCleanup: 500 * time.Millisecond,
		LogFlush:      500 * time.Millisecond,
	}
}

// Begin marks the start of the shutdown sequence.
func (b *ShutdownBudget) Begin() {
	b.start = time.Now()
}

// Remaining returns how much of the total budget remains.
// Returns 0 if the budget is expired or Begin() was never called.
func (b *ShutdownBudget) Remaining() time.Duration {
	if b.start.IsZero() {
		return 0
	}
	elapsed := time.Since(b.start)
	if elapsed >= b.Total {
		return 0
	}
	return b.Total - elapsed
}
