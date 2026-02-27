// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// SessionDedupConfig holds configuration for session deduplication.
type SessionDedupConfig struct {
	// ServerURL is the OpenCode API server URL.
	ServerURL string

	// MaxAge is the maximum age of a session to consider for dedup.
	// Sessions older than this are ignored (likely stale/completed).
	// Default: 6 hours (matches typical agent work duration).
	MaxAge time.Duration

	// Verbose enables debug logging.
	Verbose bool
}

// DefaultSessionDedupConfig returns default session dedup configuration.
func DefaultSessionDedupConfig() SessionDedupConfig {
	serverURL := os.Getenv("OPENCODE_URL")
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}
	return SessionDedupConfig{
		ServerURL: serverURL,
		MaxAge:    6 * time.Hour,
		Verbose:   false,
	}
}

// SessionDedupChecker checks for existing sessions before spawning.
type SessionDedupChecker struct {
	config SessionDedupConfig
	client *http.Client
}

// NewSessionDedupChecker creates a new session dedup checker.
func NewSessionDedupChecker(config SessionDedupConfig) *SessionDedupChecker {
	return &SessionDedupChecker{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// sessionResponse represents minimal session data for dedup checking.
type sessionResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Time  struct {
		Created int64 `json:"created"` // Unix timestamp in milliseconds
		Updated int64 `json:"updated"` // Unix timestamp in milliseconds
	} `json:"time"`
}

// HasExistingSession checks if there's an existing session for the given beads ID.
// Returns true if a recent session exists (created within MaxAge), false otherwise.
// This prevents duplicate daemon spawns for the same issue.
func (c *SessionDedupChecker) HasExistingSession(beadsID string) bool {
	if beadsID == "" {
		return false
	}

	sessions, err := c.listSessions()
	if err != nil {
		// On error, allow spawn (fail-open to avoid blocking work)
		return false
	}

	now := time.Now()
	for _, s := range sessions {
		// Extract beads ID from session title
		sessionBeadsID := extractBeadsIDFromSessionTitle(s.Title)
		if sessionBeadsID != beadsID {
			continue
		}

		// Check if session is recent enough to block spawn
		createdAt := time.Unix(s.Time.Created/1000, 0)
		age := now.Sub(createdAt)
		if age <= c.config.MaxAge {
			// Found a recent session for this beads ID
			return true
		}
	}

	return false
}

// listSessions fetches all sessions from OpenCode API.
func (c *SessionDedupChecker) listSessions() ([]sessionResponse, error) {
	resp, err := c.client.Get(c.config.ServerURL + "/session")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sessions []sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

// DefaultSessionDedupChecker is a package-level checker using default config.
var defaultSessionDedupChecker *SessionDedupChecker

// initDefaultSessionDedupChecker lazily initializes the default checker.
func initDefaultSessionDedupChecker() *SessionDedupChecker {
	if defaultSessionDedupChecker == nil {
		defaultSessionDedupChecker = NewSessionDedupChecker(DefaultSessionDedupConfig())
	}
	return defaultSessionDedupChecker
}

// HasExistingSessionForBeadsID checks if there's an existing OpenCode session
// OR tmux window for the given beads ID using the default checker.
// This is the main entry point for session dedup checking in daemon.Once().
//
// Checks two layers:
// 1. OpenCode API sessions (headless backend)
// 2. Tmux windows across all sessions (Claude CLI backend)
//
// The tmux check is critical because Claude CLI spawns create tmux windows
// WITHOUT OpenCode sessions. Without this, the entire session dedup layer
// is bypassed for Claude backend spawns.
func HasExistingSessionForBeadsID(beadsID string) bool {
	if beadsID == "" {
		return false
	}

	// Layer 1: Check OpenCode sessions (headless backend)
	checker := initDefaultSessionDedupChecker()
	if checker.HasExistingSession(beadsID) {
		return true
	}

	// Layer 2: Check tmux windows (Claude CLI backend)
	// Claude CLI spawns create tmux windows with beads ID in the window name
	// but do NOT create OpenCode sessions, so Layer 1 misses them entirely.
	if HasExistingTmuxWindowForBeadsID(beadsID) {
		return true
	}

	return false
}

// HasExistingTmuxWindowForBeadsID checks if a tmux window with the given
// beads ID exists in any tmux session (workers, orchestrator, meta-orchestrator).
// Returns true if found, false otherwise. Fails-open on tmux errors.
func HasExistingTmuxWindowForBeadsID(beadsID string) bool {
	window, _, err := tmux.FindWindowByBeadsIDAllSessions(beadsID)
	if err != nil {
		// Fail-open: if tmux isn't running or has errors, allow spawn
		return false
	}
	return window != nil
}
