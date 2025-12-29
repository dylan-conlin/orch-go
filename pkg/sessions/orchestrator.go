// Orchestrator session management for tracking human-orchestrator collaborative sessions.
// This is distinct from OpenCode session search (sessions.go) - these are "focus blocks"
// that track when an orchestrator started working, their goal, and spawned agents.

package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// OrchestratorSession represents an orchestrator's work session (a "focus block").
// Unlike worker sessions (spawn→complete), orchestrator sessions are composite:
// they may include multiple agent spawns and have a strategic goal.
type OrchestratorSession struct {
	// ID is a unique identifier for this session (format: sess_YYYYMMDD_HHMMSS)
	ID string `json:"id"`

	// Started is when the session began
	Started time.Time `json:"started"`

	// Goal is the human-readable description of what this session aims to accomplish
	Goal string `json:"goal"`

	// FocusID is the ID from the focus store (if focus was set)
	FocusID string `json:"focus_id,omitempty"`

	// Spawns tracks agent spawns during this session
	Spawns []SpawnRecord `json:"spawns,omitempty"`
}

// SpawnRecord tracks an agent spawn during an orchestrator session.
type SpawnRecord struct {
	// BeadsID is the beads issue ID for the spawn
	BeadsID string `json:"beads_id"`

	// Skill is the skill used for the spawn
	Skill string `json:"skill"`

	// SpawnedAt is when the agent was spawned
	SpawnedAt time.Time `json:"spawned_at"`

	// SessionID is the OpenCode session ID (if available)
	SessionID string `json:"session_id,omitempty"`
}

// orchestratorStoreData is the on-disk format for orchestrator sessions.
type orchestratorStoreData struct {
	Session *OrchestratorSession `json:"session,omitempty"`
}

// OrchestratorStore manages persistent orchestrator session state.
type OrchestratorStore struct {
	path    string
	session *OrchestratorSession
	mu      sync.RWMutex
}

// DefaultOrchestratorPath returns the default orchestrator session file path.
func DefaultOrchestratorPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "session.json")
}

// NewOrchestratorStore creates a new OrchestratorStore with the given path.
// If path is empty, uses DefaultOrchestratorPath().
func NewOrchestratorStore(path string) (*OrchestratorStore, error) {
	if path == "" {
		path = DefaultOrchestratorPath()
	}

	s := &OrchestratorStore{
		path: path,
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// load reads the session from disk.
func (s *OrchestratorStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		s.session = nil
		return nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		s.session = nil
		return nil
	}

	var sd orchestratorStoreData
	if err := json.Unmarshal(data, &sd); err != nil {
		return err
	}

	s.session = sd.Session
	return nil
}

// save writes the session to disk.
func (s *OrchestratorStore) save() error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	sd := orchestratorStoreData{Session: s.session}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// Get returns the current orchestrator session, or nil if not active.
func (s *OrchestratorStore) Get() *OrchestratorSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.session
}

// GenerateSessionID creates a new session ID based on current time.
// Format: sess_YYYYMMDD_HHMMSS
func GenerateSessionID() string {
	now := time.Now()
	return fmt.Sprintf("sess_%s_%s",
		now.Format("20060102"),
		now.Format("150405"))
}

// Start begins a new orchestrator session with the given goal.
// If a session is already active, returns an error.
func (s *OrchestratorStore) Start(goal string) (*OrchestratorSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Note: We allow starting a new session even if one exists
	// The previous session's data will be overwritten
	// Future: Could archive to session history

	session := &OrchestratorSession{
		ID:      GenerateSessionID(),
		Started: time.Now(),
		Goal:    goal,
		Spawns:  []SpawnRecord{},
	}

	s.session = session

	if err := s.save(); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// SetFocusID updates the focus ID for the current session.
// This is called after successfully setting focus.
func (s *OrchestratorStore) SetFocusID(focusID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		return fmt.Errorf("no active session")
	}

	s.session.FocusID = focusID
	return s.save()
}

// RecordSpawn adds a spawn record to the current session.
func (s *OrchestratorStore) RecordSpawn(beadsID, skill, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		// No active session - silently ignore (spawns can happen outside sessions)
		return nil
	}

	s.session.Spawns = append(s.session.Spawns, SpawnRecord{
		BeadsID:   beadsID,
		Skill:     skill,
		SpawnedAt: time.Now(),
		SessionID: sessionID,
	})

	return s.save()
}

// End ends the current session and returns it.
// The session data remains on disk but is marked as inactive.
func (s *OrchestratorStore) End() (*OrchestratorSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.session
	if session == nil {
		return nil, fmt.Errorf("no active session")
	}

	s.session = nil
	if err := s.save(); err != nil {
		return nil, fmt.Errorf("failed to save: %w", err)
	}

	return session, nil
}

// Clear removes the current session without returning it.
func (s *OrchestratorStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.session = nil
	return s.save()
}

// Duration returns how long the current session has been active.
// Returns 0 if no session is active.
func (s *OrchestratorStore) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return 0
	}

	return time.Since(s.session.Started)
}

// IsActive returns true if there's an active orchestrator session.
func (s *OrchestratorStore) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.session != nil
}
