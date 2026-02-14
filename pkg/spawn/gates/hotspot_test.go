package gates

import (
	"testing"
)

func TestCheckHotspot_NilChecker(t *testing.T) {
	// Should return nil when checker is nil
	result := CheckHotspot("/some/dir", "some task", "feature-impl", false, nil)
	if result != nil {
		t.Errorf("CheckHotspot with nil checker returned non-nil result")
	}
}

func TestCheckHotspot_EmptyDir(t *testing.T) {
	// Should return nil when projectDir is empty
	checker := func(dir, task string) (*HotspotResult, error) {
		t.Fatal("checker should not be called with empty dir")
		return nil, nil
	}
	result := CheckHotspot("", "some task", "feature-impl", false, checker)
	if result != nil {
		t.Errorf("CheckHotspot with empty dir returned non-nil result")
	}
}

func TestCheckHotspot_NoHotspots(t *testing.T) {
	// Should return nil when checker finds no hotspots
	checker := func(dir, task string) (*HotspotResult, error) {
		return nil, nil
	}
	result := CheckHotspot("/some/dir", "some task", "feature-impl", false, checker)
	if result != nil {
		t.Errorf("CheckHotspot with no hotspots returned non-nil result")
	}
}

func TestCheckHotspot_WithHotspots(t *testing.T) {
	// Should return the result when hotspots are found
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots: true,
			Warning:     "test warning",
		}, nil
	}
	result := CheckHotspot("/some/dir", "some task", "feature-impl", false, checker)
	if result == nil {
		t.Fatal("CheckHotspot with hotspots returned nil")
	}
	if !result.HasHotspots {
		t.Error("expected HasHotspots to be true")
	}
}

func TestCheckHotspot_DaemonDrivenSilent(t *testing.T) {
	// Daemon-driven should still return result but suppress output
	checker := func(dir, task string) (*HotspotResult, error) {
		return &HotspotResult{
			HasHotspots: true,
			Warning:     "test warning",
		}, nil
	}
	result := CheckHotspot("/some/dir", "some task", "feature-impl", true, checker)
	if result == nil {
		t.Fatal("CheckHotspot daemon-driven returned nil")
	}
	if !result.HasHotspots {
		t.Error("expected HasHotspots to be true")
	}
}
