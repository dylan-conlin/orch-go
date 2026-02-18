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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/state"
)

// TimeFormat is the timestamp format used in session storage.
const TimeFormat = time.RFC3339Nano

// DefaultInactivityTimeout marks sessions inactive after this long with no spawns.
const DefaultInactivityTimeout = 24 * time.Hour

// SessionType indicates the type of session for checkpoint threshold selection.
type SessionType string

const (
	// SessionTypeAgent is for implementation agents that accumulate code context.
	SessionTypeAgent SessionType = "agent"

	// SessionTypeOrchestrator is for orchestrator sessions that coordinate work.
	// Orchestrators delegate to agents, so their context doesn't degrade as quickly.
	SessionTypeOrchestrator SessionType = "orchestrator"
)

// CheckpointThresholds holds the duration thresholds for session checkpoints.
type CheckpointThresholds struct {
	Warning time.Duration
	Strong  time.Duration
	Max     time.Duration
}

// DefaultAgentThresholds returns the default checkpoint thresholds for agent sessions.
// Agents accumulate implementation context which degrades over time.
func DefaultAgentThresholds() CheckpointThresholds {
	return CheckpointThresholds{
		Warning: 2 * time.Hour,
		Strong:  3 * time.Hour,
		Max:     4 * time.Hour,
	}
}

// DefaultOrchestratorThresholds returns the default checkpoint thresholds for orchestrator sessions.
// Orchestrators coordinate work and delegate to agents, so context persists longer.
func DefaultOrchestratorThresholds() CheckpointThresholds {
	return CheckpointThresholds{
		Warning: 4 * time.Hour,
		Strong:  6 * time.Hour,
		Max:     8 * time.Hour,
	}
}

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

	// WindowName is the tmux window where the session was started.
	// This is captured at session start and used for archiving at session end,
	// ensuring the correct directory is used even if the user runs session end
	// from a different window.
	WindowName string `json:"window_name,omitempty"`

	// WorkspacePath is the absolute path to the session workspace directory.
	// This is the active workspace directory created at session start:
	// {project}/.orch/session/{sessionName}/active/
	// Storing this improves discoverability and enables 'orch session end'
	// to easily find and validate the workspace.
	WorkspacePath string `json:"workspace_path,omitempty"`

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

	if _, err := s.ExpireIfInactive(DefaultInactivityTimeout); err != nil {
		return nil, err
	}

	return s, nil
}

// ExpireIfInactive clears the session if there has been no activity for staleAfter.
// Returns true if the session was expired.
func (s *Store) ExpireIfInactive(staleAfter time.Duration) (bool, error) {
	if staleAfter <= 0 {
		return false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		return false, nil
	}

	lastActivity := s.session.StartedAt
	if len(s.session.Spawns) > 0 {
		lastActivity = s.session.Spawns[len(s.session.Spawns)-1].SpawnedAt
	}

	if time.Since(lastActivity) <= staleAfter {
		return false, nil
	}

	s.session = nil
	if err := s.save(); err != nil {
		return false, err
	}

	return true, nil
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

// Start begins a new session with the given goal, window name, and workspace path.
// The windowName should be the tmux window where the session is being started,
// captured after renaming the window to the session name. This allows session end
// to find the correct active directory even if called from a different window.
// The workspacePath is the absolute path to the active workspace directory.
// If a session is already active, it is replaced.
func (s *Store) Start(goal, windowName, workspacePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.session = &Session{
		Goal:          goal,
		StartedAt:     time.Now(),
		WindowName:    windowName,
		WorkspacePath: workspacePath,
		Spawns:        []SpawnRecord{},
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

// GetCheckpointStatusWithType returns checkpoint status using type-appropriate thresholds.
// For orchestrator sessions, use longer thresholds since coordination context persists better.
// For agent sessions, use shorter thresholds since implementation context degrades faster.
// Returns nil if no session is active.
func (s *Store) GetCheckpointStatusWithType(sessionType SessionType) *CheckpointStatus {
	var thresholds CheckpointThresholds
	switch sessionType {
	case SessionTypeOrchestrator:
		thresholds = DefaultOrchestratorThresholds()
	default:
		thresholds = DefaultAgentThresholds()
	}
	return s.GetCheckpointStatusWithThresholds(thresholds)
}

// GetCheckpointStatusWithThresholds returns checkpoint status using custom thresholds.
// This allows callers to provide their own thresholds (e.g., from config).
// Returns nil if no session is active.
func (s *Store) GetCheckpointStatusWithThresholds(thresholds CheckpointThresholds) *CheckpointStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return nil
	}

	duration := time.Since(s.session.StartedAt)

	status := &CheckpointStatus{
		Duration: duration,
	}

	// Format duration for messages
	formatDuration := func(d time.Duration) string {
		hours := int(d.Hours())
		if hours > 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}

	switch {
	case duration >= thresholds.Max:
		status.Level = "exceeded"
		status.Message = fmt.Sprintf("Session has exceeded %s max - handoff immediately", formatDuration(thresholds.Max))
		status.NextThreshold = 0

	case duration >= thresholds.Strong:
		status.Level = "strong"
		status.Message = fmt.Sprintf("Session at %s+ - strongly recommend handoff", formatDuration(thresholds.Strong))
		status.NextThreshold = thresholds.Max - duration

	case duration >= thresholds.Warning:
		status.Level = "warning"
		status.Message = fmt.Sprintf("Session at %s+ - consider checkpoint or handoff", formatDuration(thresholds.Warning))
		status.NextThreshold = thresholds.Strong - duration

	default:
		status.Level = "ok"
		status.Message = "Session within normal duration"
		status.NextThreshold = thresholds.Warning - duration
	}

	return status
}

// GenerateSessionName generates a session name in the format {project}-{count}.
// Count is based on existing session directories for this project.
// projectDir is the full path to the project directory.
func GenerateSessionName(projectDir string) (string, error) {
	// Extract project name from directory path
	projectName := filepath.Base(projectDir)

	// Session directories are in {projectDir}/.orch/session/
	sessionBaseDir := filepath.Join(projectDir, ".orch", "session")

	// Count existing session directories matching {project}-{number}
	count, err := countProjectSessions(sessionBaseDir, projectName)
	if err != nil {
		return "", fmt.Errorf("failed to count sessions: %w", err)
	}

	// Generate name as {project}-{count+1}
	return fmt.Sprintf("%s-%d", projectName, count+1), nil
}

// countProjectSessions counts existing session directories matching {project}-{number} pattern.
// Returns the highest number found, so next session would be count+1.
func countProjectSessions(sessionBaseDir, projectName string) (int, error) {
	// Check if session directory exists
	if _, err := os.Stat(sessionBaseDir); os.IsNotExist(err) {
		return 0, nil // No sessions yet
	}

	// List all directories in session base
	entries, err := os.ReadDir(sessionBaseDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read session directory: %w", err)
	}

	// Pattern: {project}-{number}
	pattern := regexp.MustCompile(fmt.Sprintf(`^%s-(\d+)$`, regexp.QuoteMeta(projectName)))

	maxCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory matches pattern
		matches := pattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		// Extract number
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			continue // Skip invalid numbers
		}

		// Track maximum count
		if num > maxCount {
			maxCount = num
		}
	}

	return maxCount, nil
}
