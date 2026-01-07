// Package session provides orchestrator session state management.
//
// A session represents a focused work period with a goal, start time, and
// tracked spawns. Unlike agent-registry which tracks ALL spawns, session
// only tracks spawns made during the current session.
//
// Key principle: spawn states are derived at query time via GetLiveness(),
// not stored. This prevents stale state - the session file only records
// spawn history, not spawn status.
package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/state"
)

// TimeFormat is the timestamp format used in session storage.
const TimeFormat = time.RFC3339Nano

// Checkpoint thresholds for orchestrator session duration discipline.
// Sessions exceeding these durations should checkpoint or hand off.
const (
	// CheckpointWarningDuration is when to start suggesting checkpoints (2 hours).
	CheckpointWarningDuration = 2 * time.Hour

	// CheckpointStrongDuration is when to strongly recommend handoff (3 hours).
	CheckpointStrongDuration = 3 * time.Hour

	// CheckpointMaxDuration is the maximum recommended session duration (4 hours).
	CheckpointMaxDuration = 4 * time.Hour
)

// DefaultPath returns the default session file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "session.json")
}

// Session represents an orchestrator work session.
type Session struct {
	// Goal is the session's north star priority
	Goal string `json:"goal"`

	// StartedAt is when the session began
	StartedAt time.Time `json:"started_at"`

	// Spawns are agents spawned during this session
	// Note: status is NOT stored here - derived at query time
	Spawns []SpawnRecord `json:"spawns"`
}

// SpawnRecord records that a spawn occurred during this session.
// Status is intentionally NOT stored - use GetSpawnStatuses() to derive status.
type SpawnRecord struct {
	// BeadsID is the beads issue ID tracking this agent
	BeadsID string `json:"beads_id"`

	// Skill is the skill used to spawn the agent
	Skill string `json:"skill"`

	// Task is the task description given to the agent
	Task string `json:"task,omitempty"`

	// SpawnedAt is when the agent was spawned
	SpawnedAt time.Time `json:"spawned_at"`

	// ProjectDir is the project directory the agent was spawned in
	ProjectDir string `json:"project_dir,omitempty"`
}

// SpawnStatus extends SpawnRecord with derived state information.
// This is returned by GetSpawnStatuses() which queries actual agent state.
type SpawnStatus struct {
	SpawnRecord

	// State is the derived agent state: "active", "completed", or "phantom"
	State string `json:"state"`

	// LivenessDetails contains the raw liveness check results
	LivenessDetails *state.LivenessResult `json:"liveness,omitempty"`
}

// storeData is the on-disk format.
type storeData struct {
	Session *Session `json:"session"`
}

// Store manages persistent session state.
type Store struct {
	path    string
	session *Session
	mu      sync.RWMutex
}

// New creates a new Store with the given path.
// If path is empty, uses DefaultPath().
func New(path string) (*Store, error) {
	if path == "" {
		path = DefaultPath()
	}

	s := &Store{
		path: path,
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// load reads the session from disk.
func (s *Store) load() error {
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

	var sd storeData
	if err := json.Unmarshal(data, &sd); err != nil {
		return err
	}

	s.session = sd.Session
	return nil
}

// save writes the session to disk.
func (s *Store) save() error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	sd := storeData{Session: s.session}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// Get returns the current session, or nil if no session is active.
func (s *Store) Get() *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return nil
	}

	// Return a copy to prevent external mutation
	session := *s.session
	session.Spawns = make([]SpawnRecord, len(s.session.Spawns))
	copy(session.Spawns, s.session.Spawns)
	return &session
}

// Start begins a new session with the given goal.
// If a session is already active, it is replaced.
func (s *Store) Start(goal string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.session = &Session{
		Goal:      goal,
		StartedAt: time.Now(),
		Spawns:    []SpawnRecord{},
	}

	return s.save()
}

// End ends the current session.
// Returns the ended session for logging/handoff purposes.
func (s *Store) End() (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		return nil, nil
	}

	ended := s.session
	s.session = nil

	if err := s.save(); err != nil {
		return nil, err
	}

	return ended, nil
}

// RecordSpawn records that an agent was spawned during this session.
// If no session is active, this is a no-op.
func (s *Store) RecordSpawn(beadsID, skill, task, projectDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		return nil // No active session, ignore
	}

	s.session.Spawns = append(s.session.Spawns, SpawnRecord{
		BeadsID:    beadsID,
		Skill:      skill,
		Task:       task,
		SpawnedAt:  time.Now(),
		ProjectDir: projectDir,
	})

	return s.save()
}

// GetSpawnStatuses returns spawn records with their derived states.
// This queries actual agent liveness for each spawn - it does NOT trust stored state.
func (s *Store) GetSpawnStatuses(serverURL string) []SpawnStatus {
	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()

	if session == nil {
		return nil
	}

	statuses := make([]SpawnStatus, len(session.Spawns))
	for i, spawn := range session.Spawns {
		statuses[i] = SpawnStatus{
			SpawnRecord: spawn,
		}

		// Query actual liveness
		liveness := state.GetLiveness(spawn.BeadsID, serverURL, spawn.ProjectDir)
		statuses[i].LivenessDetails = &liveness

		// Derive state from liveness
		if liveness.IsAlive() {
			statuses[i].State = "active"
		} else if liveness.IsPhantom() {
			statuses[i].State = "phantom"
		} else {
			statuses[i].State = "completed"
		}
	}

	return statuses
}

// Duration returns how long the current session has been running.
// Returns 0 if no session is active.
func (s *Store) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return 0
	}

	return time.Since(s.session.StartedAt)
}

// IsActive returns true if a session is currently active.
func (s *Store) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.session != nil
}

// SpawnCount returns the number of spawns in the current session.
func (s *Store) SpawnCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return 0
	}
	return len(s.session.Spawns)
}

// CheckpointStatus describes the current checkpoint state for a session.
type CheckpointStatus struct {
	// Level is the checkpoint urgency level: "ok", "warning", "strong", "exceeded"
	Level string

	// Message is a human-readable checkpoint recommendation
	Message string

	// SinceLastCheckpoint is the duration since session start (checkpoint tracking not yet implemented)
	Duration time.Duration

	// NextThreshold is the duration until the next checkpoint threshold
	NextThreshold time.Duration
}

// GetCheckpointStatus returns checkpoint status for the current session.
// Returns nil if no session is active.
func (s *Store) GetCheckpointStatus() *CheckpointStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return nil
	}

	duration := time.Since(s.session.StartedAt)

	status := &CheckpointStatus{
		Duration: duration,
	}

	switch {
	case duration >= CheckpointMaxDuration:
		status.Level = "exceeded"
		status.Message = "Session has exceeded 4h max - handoff immediately"
		status.NextThreshold = 0

	case duration >= CheckpointStrongDuration:
		status.Level = "strong"
		status.Message = "Session at 3h+ - strongly recommend handoff"
		status.NextThreshold = CheckpointMaxDuration - duration

	case duration >= CheckpointWarningDuration:
		status.Level = "warning"
		status.Message = "Session at 2h+ - consider checkpoint or handoff"
		status.NextThreshold = CheckpointStrongDuration - duration

	default:
		status.Level = "ok"
		status.Message = "Session within normal duration"
		status.NextThreshold = CheckpointWarningDuration - duration
	}

	return status
}
