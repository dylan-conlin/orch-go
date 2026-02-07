// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
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
		// Fail-safe: on error, assume session exists to prevent duplicate spawns.
		// Better to skip one spawn cycle than create a duplicate agent.
		fmt.Fprintf(os.Stderr, "warning: session dedup check failed for %s (assuming exists to prevent duplicate): %v\n", beadsID, err)
		return true
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
// for the given beads ID using the default checker.
// This is the main entry point for session dedup checking in daemon.Once().
func HasExistingSessionForBeadsID(beadsID string) bool {
	checker := initDefaultSessionDedupChecker()
	return checker.HasExistingSession(beadsID)
}
