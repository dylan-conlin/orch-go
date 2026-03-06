package verify

import (
	"testing"
	"time"
)

func TestVerifyLiveness(t *testing.T) {
	now := time.Now()
	fiveMinAgo := now.Add(-6 * time.Minute)
	twoMinAgo := now.Add(-2 * time.Minute)
	thirtyOneMinAgo := now.Add(-31 * time.Minute)

	tests := []struct {
		name       string
		input      LivenessInput
		wantStatus string
		wantReason string
		wantAlive  bool
	}{
		{
			name: "phase reported (non-complete) = active",
			input: LivenessInput{
				Comments:  []Comment{{Text: "Phase: Implementing - working on tests", CreatedAt: twoMinAgo.Format(time.RFC3339)}},
				SpawnTime: fiveMinAgo,
				Now:       now,
			},
			wantStatus: LivenessActive,
			wantReason: ReasonPhaseReported,
			wantAlive:  true,
		},
		{
			name: "phase complete = completed",
			input: LivenessInput{
				Comments:  []Comment{{Text: "Phase: Complete - All tests passing"}},
				SpawnTime: fiveMinAgo,
				Now:       now,
			},
			wantStatus: LivenessCompleted,
			wantReason: ReasonPhaseComplete,
			wantAlive:  false,
		},
		{
			name: "recently spawned, no phase yet = active",
			input: LivenessInput{
				Comments:  []Comment{},
				SpawnTime: twoMinAgo,
				Now:       now,
			},
			wantStatus: LivenessActive,
			wantReason: ReasonRecentlySpawned,
			wantAlive:  true,
		},
		{
			name: "no phase, spawned long ago = dead",
			input: LivenessInput{
				Comments:  []Comment{},
				SpawnTime: fiveMinAgo,
				Now:       now,
			},
			wantStatus: LivenessDead,
			wantReason: ReasonNoPhaseReported,
			wantAlive:  false,
		},
		{
			name: "no phase, zero spawn time, old = dead",
			input: LivenessInput{
				Comments:  []Comment{},
				SpawnTime: time.Time{},
				Now:       now,
			},
			wantStatus: LivenessDead,
			wantReason: ReasonNoPhaseReported,
			wantAlive:  false,
		},
		{
			name: "phase reported but stale (31 min ago) = still active",
			input: LivenessInput{
				Comments:  []Comment{{Text: "Phase: Implementing - building feature", CreatedAt: thirtyOneMinAgo.Format(time.RFC3339)}},
				SpawnTime: fiveMinAgo,
				Now:       now,
			},
			wantStatus: LivenessActive,
			wantReason: ReasonPhaseReported,
			wantAlive:  true,
		},
		{
			name: "multiple phases, latest is complete",
			input: LivenessInput{
				Comments: []Comment{
					{Text: "Phase: Planning - started"},
					{Text: "Phase: Implementing - coding"},
					{Text: "Phase: Complete - done"},
				},
				SpawnTime: fiveMinAgo,
				Now:       now,
			},
			wantStatus: LivenessCompleted,
			wantReason: ReasonPhaseComplete,
			wantAlive:  false,
		},
		{
			name: "multiple phases, latest is not complete",
			input: LivenessInput{
				Comments: []Comment{
					{Text: "Phase: Planning - started"},
					{Text: "Phase: Implementing - coding"},
				},
				SpawnTime: fiveMinAgo,
				Now:       now,
			},
			wantStatus: LivenessActive,
			wantReason: ReasonPhaseReported,
			wantAlive:  true,
		},
		{
			name: "non-phase comments only, recently spawned",
			input: LivenessInput{
				Comments:  []Comment{{Text: "FRAME: Started working on X"}},
				SpawnTime: twoMinAgo,
				Now:       now,
			},
			wantStatus: LivenessActive,
			wantReason: ReasonRecentlySpawned,
			wantAlive:  true,
		},
		{
			name: "case insensitive phase complete",
			input: LivenessInput{
				Comments:  []Comment{{Text: "Phase: complete - all done"}},
				SpawnTime: fiveMinAgo,
				Now:       now,
			},
			wantStatus: LivenessCompleted,
			wantReason: ReasonPhaseComplete,
			wantAlive:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyLiveness(tt.input)

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", result.Status, tt.wantStatus)
			}
			if result.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
			}
			if result.IsAlive() != tt.wantAlive {
				t.Errorf("IsAlive() = %v, want %v", result.IsAlive(), tt.wantAlive)
			}
		})
	}
}

func TestLivenessResult_Warning(t *testing.T) {
	now := time.Now()
	twoMinAgo := now.Add(-2 * time.Minute)

	result := VerifyLiveness(LivenessInput{
		Comments:  []Comment{{Text: "Phase: Implementing - working on tests", CreatedAt: twoMinAgo.Format(time.RFC3339)}},
		SpawnTime: now.Add(-10 * time.Minute),
		Now:       now,
	})

	warning := result.Warning()
	if warning == "" {
		t.Error("expected non-empty warning for active agent")
	}

	// Completed agent should have no warning
	completedResult := VerifyLiveness(LivenessInput{
		Comments:  []Comment{{Text: "Phase: Complete - done"}},
		SpawnTime: now.Add(-10 * time.Minute),
		Now:       now,
	})
	if completedResult.Warning() != "" {
		t.Errorf("expected empty warning for completed agent, got %q", completedResult.Warning())
	}

	// Dead agent should have no warning
	deadResult := VerifyLiveness(LivenessInput{
		Comments:  []Comment{},
		SpawnTime: now.Add(-10 * time.Minute),
		Now:       now,
	})
	if deadResult.Warning() != "" {
		t.Errorf("expected empty warning for dead agent, got %q", deadResult.Warning())
	}
}

func TestVerifyLivenessGracePeriod(t *testing.T) {
	now := time.Now()

	// Exactly at grace period boundary (5 min)
	result := VerifyLiveness(LivenessInput{
		Comments:  []Comment{},
		SpawnTime: now.Add(-5 * time.Minute),
		Now:       now,
	})

	// At exactly 5 minutes, agent should be dead (grace period is < 5 min)
	if result.Status != LivenessDead {
		t.Errorf("at 5 min boundary: Status = %q, want %q", result.Status, LivenessDead)
	}

	// Just under 5 minutes = still in grace period
	result2 := VerifyLiveness(LivenessInput{
		Comments:  []Comment{},
		SpawnTime: now.Add(-4*time.Minute - 59*time.Second),
		Now:       now,
	})
	if result2.Status != LivenessActive {
		t.Errorf("at 4m59s: Status = %q, want %q", result2.Status, LivenessActive)
	}
}
