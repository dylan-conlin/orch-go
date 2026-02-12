package cleanup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/state"
)

func TestReconcileStateMarksStaleRows(t *testing.T) {
	now := time.Now().UnixMilli()

	dbPath := filepath.Join(t.TempDir(), "state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("open state DB: %v", err)
	}

	agents := []*state.Agent{
		{WorkspaceName: "ws-live", BeadsID: "orch-go-live1", SessionID: "ses_live", Mode: "opencode", ProjectDir: t.TempDir(), SpawnTime: now},
		{WorkspaceName: "ws-live-beads", BeadsID: "orch-go-live2", SessionID: "", Mode: "opencode", ProjectDir: t.TempDir(), SpawnTime: now},
		{WorkspaceName: "ws-stale", BeadsID: "orch-go-stale1", SessionID: "ses_dead", Mode: "opencode", ProjectDir: t.TempDir(), SpawnTime: now},
		{WorkspaceName: "ws-complete", BeadsID: "orch-go-stale2", SessionID: "ses_dead2", Mode: "opencode", ProjectDir: t.TempDir(), SpawnTime: now, Phase: "Complete - done"},
		{WorkspaceName: "ws-claude", BeadsID: "orch-go-claude", SessionID: "", Mode: "claude", ProjectDir: t.TempDir(), SpawnTime: now},
	}

	for _, agent := range agents {
		if err := db.InsertAgent(agent); err != nil {
			t.Fatalf("insert agent %s: %v", agent.WorkspaceName, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close state DB: %v", err)
	}

	server := newSessionServer([]map[string]any{
		{"id": "ses_live", "title": "Worker [orch-go-live1]", "time": map[string]any{"created": now, "updated": now}},
		{"id": "ses_other", "title": "Other [orch-go-live2]", "time": map[string]any{"created": now, "updated": now}},
		{"id": "ses_old", "title": "Old [orch-go-stale1]", "time": map[string]any{"created": now, "updated": now - int64((2 * time.Hour).Milliseconds())}},
	})
	defer server.Close()

	result, err := ReconcileState(ReconcileStateOptions{
		ServerURL: server.URL,
		DBPath:    dbPath,
		Quiet:     true,
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if result.ActiveRows != 5 {
		t.Fatalf("ActiveRows = %d, want 5", result.ActiveRows)
	}
	if result.ReconcilableRows != 4 {
		t.Fatalf("ReconcilableRows = %d, want 4", result.ReconcilableRows)
	}
	if result.LiveRows != 2 {
		t.Fatalf("LiveRows = %d, want 2", result.LiveRows)
	}
	if result.StaleRows != 2 {
		t.Fatalf("StaleRows = %d, want 2", result.StaleRows)
	}
	if result.CompletedRows != 1 {
		t.Fatalf("CompletedRows = %d, want 1", result.CompletedRows)
	}
	if result.AbandonedRows != 1 {
		t.Fatalf("AbandonedRows = %d, want 1", result.AbandonedRows)
	}
	if result.SkippedRows != 1 {
		t.Fatalf("SkippedRows = %d, want 1", result.SkippedRows)
	}
	if result.OpenMinusLiveBefore != 2 {
		t.Fatalf("OpenMinusLiveBefore = %d, want 2", result.OpenMinusLiveBefore)
	}
	if result.OpenMinusLiveAfter != 0 {
		t.Fatalf("OpenMinusLiveAfter = %d, want 0", result.OpenMinusLiveAfter)
	}

	verifyDB, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen state DB: %v", err)
	}
	defer verifyDB.Close()

	assertAgentState(t, verifyDB, "ws-live", false, false)
	assertAgentState(t, verifyDB, "ws-live-beads", false, false)
	assertAgentState(t, verifyDB, "ws-stale", false, true)
	assertAgentState(t, verifyDB, "ws-complete", true, false)
	assertAgentState(t, verifyDB, "ws-claude", false, false)
}

func TestReconcileStateDryRunDoesNotMutate(t *testing.T) {
	now := time.Now().UnixMilli()

	dbPath := filepath.Join(t.TempDir(), "state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("open state DB: %v", err)
	}
	if err := db.InsertAgent(&state.Agent{
		WorkspaceName: "ws-stale",
		BeadsID:       "orch-go-stale",
		SessionID:     "ses_dead",
		Mode:          "opencode",
		ProjectDir:    t.TempDir(),
		SpawnTime:     now,
	}); err != nil {
		t.Fatalf("insert stale row: %v", err)
	}
	db.Close()

	server := newSessionServer([]map[string]any{})
	defer server.Close()

	result, err := ReconcileState(ReconcileStateOptions{
		ServerURL: server.URL,
		DBPath:    dbPath,
		DryRun:    true,
		Quiet:     true,
	})
	if err != nil {
		t.Fatalf("reconcile dry-run: %v", err)
	}
	if result.AbandonedRows != 1 {
		t.Fatalf("AbandonedRows = %d, want 1", result.AbandonedRows)
	}
	if result.OpenMinusLiveAfter != result.OpenMinusLiveBefore {
		t.Fatalf("OpenMinusLiveAfter = %d, want %d", result.OpenMinusLiveAfter, result.OpenMinusLiveBefore)
	}

	verifyDB, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen state DB: %v", err)
	}
	defer verifyDB.Close()
	assertAgentState(t, verifyDB, "ws-stale", false, false)
}

func TestReconcileStateChecksSessionEndpointBeforeAbandon(t *testing.T) {
	now := time.Now().UnixMilli()

	dbPath := filepath.Join(t.TempDir(), "state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("open state DB: %v", err)
	}
	if err := db.InsertAgent(&state.Agent{
		WorkspaceName: "ws-race",
		BeadsID:       "orch-go-race",
		SessionID:     "ses_race",
		Mode:          "headless",
		ProjectDir:    t.TempDir(),
		SpawnTime:     now,
	}); err != nil {
		t.Fatalf("insert state row: %v", err)
	}
	db.Close()

	server := newSessionServerWithDetails(
		[]map[string]any{},
		map[string]map[string]any{
			"ses_race": {
				"id":    "ses_race",
				"title": "Worker [orch-go-race]",
				"time": map[string]any{
					"created": now,
					"updated": now,
				},
			},
		},
	)
	defer server.Close()

	result, err := ReconcileState(ReconcileStateOptions{
		ServerURL: server.URL,
		DBPath:    dbPath,
		Quiet:     true,
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if result.LiveRows != 1 {
		t.Fatalf("LiveRows = %d, want 1", result.LiveRows)
	}
	if result.StaleRows != 0 {
		t.Fatalf("StaleRows = %d, want 0", result.StaleRows)
	}
	if result.AbandonedRows != 0 {
		t.Fatalf("AbandonedRows = %d, want 0", result.AbandonedRows)
	}

	verifyDB, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen state DB: %v", err)
	}
	defer verifyDB.Close()
	assertAgentState(t, verifyDB, "ws-race", false, false)
}

func TestReconcileStateUpdatesRegistryBySessionLiveness(t *testing.T) {
	now := time.Now().UnixMilli()
	registryPath := filepath.Join(t.TempDir(), "sessions.json")
	reg := session.NewRegistry(registryPath)

	seed := []session.OrchestratorSession{
		{WorkspaceName: "orch-live", SessionID: "ses_live", ProjectDir: t.TempDir(), SpawnTime: time.Now(), Status: "active"},
		{WorkspaceName: "orch-stale", SessionID: "ses_dead", ProjectDir: t.TempDir(), SpawnTime: time.Now(), Status: "active"},
		{WorkspaceName: "orch-noid", SessionID: "", ProjectDir: t.TempDir(), SpawnTime: time.Now(), Status: "active"},
	}
	for _, item := range seed {
		if err := reg.Register(item); err != nil {
			t.Fatalf("seed registry %s: %v", item.WorkspaceName, err)
		}
	}

	server := newSessionServer([]map[string]any{
		{"id": "ses_live", "title": "Live", "time": map[string]any{"created": now, "updated": now}},
	})
	defer server.Close()

	result, err := ReconcileState(ReconcileStateOptions{
		ServerURL:         server.URL,
		RegistryPath:      registryPath,
		ReconcileRegistry: true,
		Quiet:             true,
	})
	if err != nil {
		t.Fatalf("reconcile registry: %v", err)
	}

	if result.RegistryActive != 3 {
		t.Fatalf("RegistryActive = %d, want 3", result.RegistryActive)
	}
	if result.RegistryUpdated != 1 {
		t.Fatalf("RegistryUpdated = %d, want 1", result.RegistryUpdated)
	}
	if result.RegistrySkipped != 1 {
		t.Fatalf("RegistrySkipped = %d, want 1", result.RegistrySkipped)
	}

	live, err := reg.Get("orch-live")
	if err != nil {
		t.Fatalf("get orch-live: %v", err)
	}
	if live.Status != "active" {
		t.Fatalf("orch-live status = %q, want active", live.Status)
	}

	stale, err := reg.Get("orch-stale")
	if err != nil {
		t.Fatalf("get orch-stale: %v", err)
	}
	if stale.Status != "abandoned" {
		t.Fatalf("orch-stale status = %q, want abandoned", stale.Status)
	}

	noID, err := reg.Get("orch-noid")
	if err != nil {
		t.Fatalf("get orch-noid: %v", err)
	}
	if noID.Status != "active" {
		t.Fatalf("orch-noid status = %q, want active", noID.Status)
	}
}

func newSessionServer(payload []map[string]any) *httptest.Server {
	return newSessionServerWithDetails(payload, nil)
}

func newSessionServerWithDetails(payload []map[string]any, details map[string]map[string]any) *httptest.Server {
	sessionsByID := make(map[string]map[string]any)
	for _, item := range payload {
		id, _ := item["id"].(string)
		if id != "" {
			sessionsByID[id] = item
		}
	}
	for id, item := range details {
		sessionsByID[id] = item
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/session/") {
			sessionID := strings.TrimPrefix(r.URL.Path, "/session/")
			item, ok := sessionsByID[sessionID]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(item)
			return
		}

		http.NotFound(w, r)
		return
	}))
}

func assertAgentState(t *testing.T, db *state.DB, workspace string, wantCompleted, wantAbandoned bool) {
	t.Helper()
	agent, err := db.GetAgent(workspace)
	if err != nil {
		t.Fatalf("GetAgent(%s): %v", workspace, err)
	}
	if agent.IsCompleted != wantCompleted {
		t.Fatalf("%s IsCompleted = %v, want %v", workspace, agent.IsCompleted, wantCompleted)
	}
	if agent.IsAbandoned != wantAbandoned {
		t.Fatalf("%s IsAbandoned = %v, want %v", workspace, agent.IsAbandoned, wantAbandoned)
	}
}
