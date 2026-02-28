package agent

import (
	"fmt"
	"testing"
)

func TestState_IsTerminal(t *testing.T) {
	tests := []struct {
		state    State
		terminal bool
	}{
		{StateSpawning, false},
		{StateActive, false},
		{StatePhaseComplete, false},
		{StateCompleting, false},
		{StateCompleted, true},
		{StateAbandoned, true},
		{StateOrphaned, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.terminal {
				t.Errorf("State(%q).IsTerminal() = %v, want %v", tt.state, got, tt.terminal)
			}
		})
	}
}

func TestState_IsTransient(t *testing.T) {
	tests := []struct {
		state     State
		transient bool
	}{
		{StateSpawning, true},
		{StateActive, false},
		{StatePhaseComplete, false},
		{StateCompleting, true},
		{StateCompleted, false},
		{StateAbandoned, false},
		{StateOrphaned, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsTransient(); got != tt.transient {
				t.Errorf("State(%q).IsTransient() = %v, want %v", tt.state, got, tt.transient)
			}
		})
	}
}

func TestValidateTransition_Valid(t *testing.T) {
	tests := []struct {
		from       State
		transition Transition
		wantTo     State
	}{
		{StateSpawning, TransitionSpawn, StateActive},
		{StateActive, TransitionPhaseComplete, StatePhaseComplete},
		{StatePhaseComplete, TransitionComplete, StateCompleted},
		{StateActive, TransitionAbandon, StateAbandoned},
		{StateOrphaned, TransitionForceComplete, StateCompleted},
		{StateOrphaned, TransitionForceAbandon, StateAbandoned},
	}

	for _, tt := range tests {
		t.Run(tt.transition.String(), func(t *testing.T) {
			got, err := ValidateTransition(tt.from, tt.transition)
			if err != nil {
				t.Fatalf("ValidateTransition(%q, %q) returned error: %v", tt.from, tt.transition, err)
			}
			if got != tt.wantTo {
				t.Errorf("ValidateTransition(%q, %q) = %q, want %q", tt.from, tt.transition, got, tt.wantTo)
			}
		})
	}
}

func TestValidateTransition_InvalidFromState(t *testing.T) {
	tests := []struct {
		name       string
		from       State
		transition Transition
	}{
		{"complete from active", StateActive, TransitionComplete},
		{"abandon from completed", StateCompleted, TransitionAbandon},
		{"spawn from active", StateActive, TransitionSpawn},
		{"force_complete from active", StateActive, TransitionForceComplete},
		{"phase_complete from spawning", StateSpawning, TransitionPhaseComplete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateTransition(tt.from, tt.transition)
			if err == nil {
				t.Errorf("ValidateTransition(%q, %q) expected error, got nil", tt.from, tt.transition)
			}
		})
	}
}

func TestValidateTransition_UnknownTransition(t *testing.T) {
	_, err := ValidateTransition(StateActive, Transition("unknown"))
	if err == nil {
		t.Error("ValidateTransition with unknown transition expected error, got nil")
	}
}

func TestTransitionEvent_HasCriticalFailure(t *testing.T) {
	t.Run("no effects", func(t *testing.T) {
		te := &TransitionEvent{}
		if te.HasCriticalFailure() {
			t.Error("empty TransitionEvent should not have critical failure")
		}
	})

	t.Run("non-critical failure", func(t *testing.T) {
		te := &TransitionEvent{
			Effects: []EffectResult{
				{Subsystem: "tmux", Operation: "kill_window", Success: false, Critical: false},
			},
		}
		if te.HasCriticalFailure() {
			t.Error("non-critical failure should not count as critical")
		}
	})

	t.Run("critical failure", func(t *testing.T) {
		te := &TransitionEvent{
			Effects: []EffectResult{
				{Subsystem: "beads", Operation: "close_issue", Success: false, Critical: true},
			},
		}
		if !te.HasCriticalFailure() {
			t.Error("critical failure should be detected")
		}
	})

	t.Run("critical success", func(t *testing.T) {
		te := &TransitionEvent{
			Effects: []EffectResult{
				{Subsystem: "beads", Operation: "close_issue", Success: true, Critical: true},
			},
		}
		if te.HasCriticalFailure() {
			t.Error("critical success should not be reported as failure")
		}
	})
}

func TestTransitionEvent_AddEffect(t *testing.T) {
	te := &TransitionEvent{}

	// Add successful effect
	te.AddEffect(EffectResult{
		Subsystem: "beads",
		Operation: "close_issue",
		Success:   true,
		Critical:  true,
	})

	if len(te.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(te.Effects))
	}
	if len(te.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(te.Warnings))
	}

	// Add non-critical failure — should generate warning
	te.AddEffect(EffectResult{
		Subsystem: "tmux",
		Operation: "kill_window",
		Success:   false,
		Critical:  false,
		Error:     fmt.Errorf("window not found"),
	})

	if len(te.Effects) != 2 {
		t.Fatalf("expected 2 effects, got %d", len(te.Effects))
	}
	if len(te.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(te.Warnings))
	}
}

func TestAllStates_Coverage(t *testing.T) {
	states := AllStates()
	if len(states) != 7 {
		t.Errorf("AllStates() returned %d states, expected 7", len(states))
	}

	seen := make(map[State]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate state: %s", s)
		}
		seen[s] = true
	}
}

func TestAllTransitions_Coverage(t *testing.T) {
	transitions := AllTransitions()
	if len(transitions) != 6 {
		t.Errorf("AllTransitions() returned %d transitions, expected 6", len(transitions))
	}

	seen := make(map[Transition]bool)
	for _, tr := range transitions {
		if seen[tr] {
			t.Errorf("duplicate transition: %s", tr)
		}
		seen[tr] = true
	}
}
