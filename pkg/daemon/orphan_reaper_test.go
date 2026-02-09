package daemon

import (
	"net/http"
	"net/http/httptest"
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

func TestGetActiveSessionInfo_ExtractsIDsAndTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"id":"ses_abc","title":"og-feat-auth [orch-go-12345]"},
			{"id":"ses_xyz","title":"og-debug-fix"}
		]`))
	}))
	defer server.Close()

	activeIDs, activeTitles, err := getActiveSessionInfo(server.URL)
	if err != nil {
		t.Fatalf("getActiveSessionInfo() error: %v", err)
	}

	// Verify session IDs
	if !activeIDs["ses_abc"] {
		t.Error("expected ses_abc in activeIDs")
	}
	if !activeIDs["ses_xyz"] {
		t.Error("expected ses_xyz in activeIDs")
	}

	// Verify titles include full title and extracted workspace name
	if !activeTitles["og-feat-auth [orch-go-12345]"] {
		t.Error("expected full title in activeTitles")
	}
	if !activeTitles["og-feat-auth"] {
		t.Error("expected workspace name extracted from title")
	}
	if !activeTitles["og-debug-fix"] {
		t.Error("expected og-debug-fix in activeTitles")
	}
}

func TestReapOrphanProcesses_LedgerSweptField(t *testing.T) {
	// Verify the LedgerSwept field is included in the result structure
	result := &OrphanReapResult{
		Found:       3,
		Killed:      2,
		LedgerSwept: 1,
		Message:     "test",
	}
	if result.LedgerSwept != 1 {
		t.Errorf("LedgerSwept = %d, want 1", result.LedgerSwept)
	}
}
