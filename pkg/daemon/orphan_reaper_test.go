package daemon

import (
	"testing"
	"time"
)

func TestReapOrphanProcesses_DisabledByConfig(t *testing.T) {
	d := &Daemon{
		Config: Config{
			OrphanReapEnabled: false,
		},
	}

	result := d.ReapOrphanProcesses()
	if result != nil {
		t.Errorf("ReapOrphanProcesses() should return nil when disabled, got %+v", result)
	}
}

func TestReapOrphanProcesses_SkipsWhenNotDue(t *testing.T) {
	d := &Daemon{
		Config: Config{
			OrphanReapEnabled:  true,
			OrphanReapInterval: 5 * time.Minute,
		},
		lastOrphanReap: time.Now(), // Just ran
	}

	result := d.ReapOrphanProcesses()
	if result != nil {
		t.Errorf("ReapOrphanProcesses() should return nil when not due, got %+v", result)
	}
}

func TestReapOrphanProcesses_RunsWhenDue(t *testing.T) {
	d := &Daemon{
		Config: Config{
			OrphanReapEnabled:  true,
			OrphanReapInterval: 5 * time.Minute,
			CleanupServerURL:   "http://127.0.0.1:99999", // Will fail - no server
		},
		lastOrphanReap: time.Now().Add(-10 * time.Minute), // Overdue
	}

	result := d.ReapOrphanProcesses()
	// Should return a result (even if error) since it's due
	if result == nil {
		t.Fatal("ReapOrphanProcesses() should return result when due")
	}
	// Expect an error since we're connecting to a non-existent server
	if result.Error == nil {
		t.Error("ReapOrphanProcesses() should return error when server unavailable")
	}
}

func TestReapOrphanProcesses_RunsOnFirstCall(t *testing.T) {
	d := &Daemon{
		Config: Config{
			OrphanReapEnabled:  true,
			OrphanReapInterval: 5 * time.Minute,
			CleanupServerURL:   "http://127.0.0.1:99999", // Will fail - no server
		},
		// lastOrphanReap is zero value - should run on first call
	}

	result := d.ReapOrphanProcesses()
	if result == nil {
		t.Fatal("ReapOrphanProcesses() should return result on first call")
	}
}

func TestReapOrphanProcesses_DefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if !config.OrphanReapEnabled {
		t.Error("DefaultConfig().OrphanReapEnabled should be true")
	}
	if config.OrphanReapInterval != 5*time.Minute {
		t.Errorf("DefaultConfig().OrphanReapInterval = %v, want 5m", config.OrphanReapInterval)
	}
}

func TestGetActiveSessionTitles_InvalidServer(t *testing.T) {
	_, err := getActiveSessionTitles("http://127.0.0.1:99999")
	if err == nil {
		t.Error("getActiveSessionTitles() should return error for invalid server")
	}
}
