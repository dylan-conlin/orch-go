package gates

import (
	"testing"
)

func TestCheckDrainGate_DaemonExempt(t *testing.T) {
	// Daemon-driven spawns should never be blocked by drain gate
	err := CheckDrainGate(false, "", true)
	if err != nil {
		t.Errorf("daemon-driven spawn should be exempt from drain gate, got: %v", err)
	}
}

func TestCheckDrainGate_BypassRequiresReason(t *testing.T) {
	err := CheckDrainGate(true, "", false)
	if err == nil {
		t.Error("bypass without reason should error")
	}
}

func TestCheckDrainGate_BypassWithReason(t *testing.T) {
	// This will attempt to log but shouldn't block
	err := CheckDrainGate(true, "Dylan requested immediate spawn", false)
	if err != nil {
		t.Errorf("bypass with reason should allow spawn, got: %v", err)
	}
}

func TestShowDrainGateBlocked(t *testing.T) {
	err := showDrainGateBlocked(3)
	if err == nil {
		t.Error("showDrainGateBlocked should return error")
	}
	if err.Error() != "spawn blocked: 3 reviewable completion(s) exist — drain before spawning" {
		t.Errorf("unexpected error message: %v", err)
	}
}
