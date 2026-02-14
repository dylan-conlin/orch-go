package gates

import (
	"testing"
)

func TestCheckTriageBypass_DaemonDriven(t *testing.T) {
	// Daemon-driven spawns always pass regardless of bypass flag
	err := CheckTriageBypass(true, false, "investigation", "some task")
	if err != nil {
		t.Errorf("CheckTriageBypass(daemonDriven=true) returned error: %v", err)
	}
}

func TestCheckTriageBypass_ManualWithBypass(t *testing.T) {
	// Manual spawn with --bypass-triage should pass
	err := CheckTriageBypass(false, true, "investigation", "some task")
	if err != nil {
		t.Errorf("CheckTriageBypass(bypassTriage=true) returned error: %v", err)
	}
}

func TestCheckTriageBypass_ManualWithoutBypass(t *testing.T) {
	// Manual spawn without --bypass-triage should fail
	err := CheckTriageBypass(false, false, "investigation", "some task")
	if err == nil {
		t.Error("CheckTriageBypass(bypassTriage=false) expected error, got nil")
	}
}
