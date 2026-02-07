package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHasExistingSession(t *testing.T) {
	tests := []struct {
		name       string
		beadsID    string
		sessions   []sessionResponse
		maxAge     time.Duration
		wantExists bool
	}{
		{
			name:       "no sessions",
			beadsID:    "orch-go-abc123",
			sessions:   []sessionResponse{},
			maxAge:     6 * time.Hour,
			wantExists: false,
		},
		{
			name:    "matching session exists - recent",
			beadsID: "orch-go-abc123",
			sessions: []sessionResponse{
				{
					ID:    "session-1",
					Title: "og-feat-test-15jan [orch-go-abc123]",
					Time: struct {
						Created int64 `json:"created"`
						Updated int64 `json:"updated"`
					}{
						Created: time.Now().Add(-1 * time.Hour).UnixMilli(),
						Updated: time.Now().Add(-30 * time.Minute).UnixMilli(),
					},
				},
			},
			maxAge:     6 * time.Hour,
			wantExists: true,
		},
		{
			name:    "matching session exists - too old",
			beadsID: "orch-go-abc123",
			sessions: []sessionResponse{
				{
					ID:    "session-1",
					Title: "og-feat-test-15jan [orch-go-abc123]",
					Time: struct {
						Created int64 `json:"created"`
						Updated int64 `json:"updated"`
					}{
						Created: time.Now().Add(-10 * time.Hour).UnixMilli(),
						Updated: time.Now().Add(-8 * time.Hour).UnixMilli(),
					},
				},
			},
			maxAge:     6 * time.Hour,
			wantExists: false,
		},
		{
			name:    "no matching session - different beads ID",
			beadsID: "orch-go-abc123",
			sessions: []sessionResponse{
				{
					ID:    "session-1",
					Title: "og-feat-other-15jan [orch-go-xyz789]",
					Time: struct {
						Created int64 `json:"created"`
						Updated int64 `json:"updated"`
					}{
						Created: time.Now().Add(-1 * time.Hour).UnixMilli(),
						Updated: time.Now().Add(-30 * time.Minute).UnixMilli(),
					},
				},
			},
			maxAge:     6 * time.Hour,
			wantExists: false,
		},
		{
			name:    "multiple sessions - one matches",
			beadsID: "orch-go-target",
			sessions: []sessionResponse{
				{
					ID:    "session-1",
					Title: "og-feat-other-15jan [orch-go-other]",
					Time: struct {
						Created int64 `json:"created"`
						Updated int64 `json:"updated"`
					}{
						Created: time.Now().Add(-1 * time.Hour).UnixMilli(),
						Updated: time.Now().UnixMilli(),
					},
				},
				{
					ID:    "session-2",
					Title: "og-feat-target-15jan [orch-go-target]",
					Time: struct {
						Created int64 `json:"created"`
						Updated int64 `json:"updated"`
					}{
						Created: time.Now().Add(-2 * time.Hour).UnixMilli(),
						Updated: time.Now().UnixMilli(),
					},
				},
			},
			maxAge:     6 * time.Hour,
			wantExists: true,
		},
		{
			name:       "empty beads ID",
			beadsID:    "",
			sessions:   []sessionResponse{},
			maxAge:     6 * time.Hour,
			wantExists: false,
		},
		{
			name:    "session without beads ID in title",
			beadsID: "orch-go-abc123",
			sessions: []sessionResponse{
				{
					ID:    "session-1",
					Title: "og-feat-test-15jan", // No [beads-id]
					Time: struct {
						Created int64 `json:"created"`
						Updated int64 `json:"updated"`
					}{
						Created: time.Now().Add(-1 * time.Hour).UnixMilli(),
						Updated: time.Now().UnixMilli(),
					},
				},
			},
			maxAge:     6 * time.Hour,
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.sessions)
			}))
			defer server.Close()

			// Create checker with mock server
			config := SessionDedupConfig{
				ServerURL: server.URL,
				MaxAge:    tt.maxAge,
				Verbose:   false,
			}
			checker := NewSessionDedupChecker(config)

			got := checker.HasExistingSession(tt.beadsID)
			if got != tt.wantExists {
				t.Errorf("HasExistingSession(%q) = %v, want %v", tt.beadsID, got, tt.wantExists)
			}
		})
	}
}

func TestHasExistingSession_ServerError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := SessionDedupConfig{
		ServerURL: server.URL,
		MaxAge:    6 * time.Hour,
		Verbose:   false,
	}
	checker := NewSessionDedupChecker(config)

	// Should return true (fail-safe) on server error to prevent duplicate spawns
	got := checker.HasExistingSession("orch-go-abc123")
	if got != true {
		t.Errorf("HasExistingSession() on server error = %v, want true (fail-safe to prevent duplicates)", got)
	}
}

func TestExtractBeadsIDFromSessionTitle_Additional(t *testing.T) {
	tests := []struct {
		title  string
		wantID string
	}{
		{
			title:  "og-feat-test-15jan [orch-go-abc123]",
			wantID: "orch-go-abc123",
		},
		{
			title:  "workspace-name [proj-xyz789]",
			wantID: "proj-xyz789",
		},
		{
			title:  "og-feat-test-15jan", // No beads ID
			wantID: "",
		},
		{
			title:  "workspace [beads-id] extra text", // Beads ID not at end
			wantID: "beads-id",
		},
		{
			title:  "[ ]", // Empty brackets
			wantID: "",
		},
		{
			title:  "test [  spaced-id  ]", // Whitespace in brackets
			wantID: "spaced-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := extractBeadsIDFromSessionTitle(tt.title)
			if got != tt.wantID {
				t.Errorf("extractBeadsIDFromSessionTitle(%q) = %q, want %q", tt.title, got, tt.wantID)
			}
		})
	}
}
