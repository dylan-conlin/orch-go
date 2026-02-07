// Package session provides orchestrator session state management.
//
// The Registry type tracks orchestrator sessions (spawned agents) in a persistent
// JSON file with file locking for concurrent access. This is separate from the
// Store type which tracks orchestrator work sessions (goals, focus blocks).

package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RegistryPath returns the default session registry path.
func RegistryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "sessions.json")
}

// RegistryLockPath returns the default session registry lock file path.
func RegistryLockPath() string {
	return RegistryPath() + ".lock"
}

// OrchestratorSession represents an active orchestrator session.
type OrchestratorSession struct {
	// WorkspaceName is the unique identifier for this session (e.g., "og-orch-ship-feature-05jan")
	WorkspaceName string `json:"workspace_name"`

	// SessionID is the OpenCode session ID (e.g., "ses_abc123...")
	SessionID string `json:"session_id"`

	// ProjectDir is the absolute path to the project directory
	ProjectDir string `json:"project_dir"`

	// SpawnTime is when the session was started
	SpawnTime time.Time `json:"spawn_time"`

	// Goal is the session's objective
	Goal string `json:"goal"`

	// Status is the session state: "active", "completed", or "abandoned"
	Status string `json:"status"`

	// ArchivedPath is set when the workspace has been archived (empty if not archived).
	// Format: absolute path to the archived workspace directory.
	ArchivedPath string `json:"archived_path,omitempty"`
}

// RegistrySchema provides inline documentation for the registry format.
// This makes the file self-describing for agents and humans who encounter it.
type RegistrySchema struct {
	Version      string `json:"version"`
	Description  string `json:"description"`
	PrimaryKey   string `json:"primary_key"`
	StatusValues string `json:"status_values"`
	SafeOps      string `json:"safe_operations"`
	Modify       string `json:"to_modify"`
}

// DefaultRegistrySchema returns the schema metadata for sessions.json.
func DefaultRegistrySchema() RegistrySchema {
	return RegistrySchema{
		Version:      "1.0",
		Description:  "Orchestrator session registry - tracks spawned agent sessions",
		PrimaryKey:   "workspace_name (unique identifier for each session)",
		StatusValues: "active | completed | abandoned",
		SafeOps:      "read-only; file uses locking for concurrent access",
		Modify:       "use orch commands (orch spawn, orch complete, orch abandon)",
	}
}

// RegistryData is the on-disk format of the session registry.
type RegistryData struct {
	Schema   RegistrySchema        `json:"_schema,omitempty"`
	Sessions []OrchestratorSession `json:"sessions"`
}

// Registry manages persistent orchestrator session state.
type Registry struct {
	path     string
	lockPath string
	mu       sync.Mutex
}

// Common errors
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrLockFailed      = errors.New("failed to acquire lock")
)

// NewRegistry creates a new Registry with the given path.
// If path is empty, uses RegistryPath().
func NewRegistry(path string) *Registry {
	if path == "" {
		path = RegistryPath()
	}
	return &Registry{
		path:     path,
		lockPath: path + ".lock",
	}
}

// withLock executes fn while holding a file lock.
// This protects against concurrent access from multiple processes.
func (r *Registry) withLock(fn func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(r.lockPath), 0755); err != nil {
		return err
	}

	// Try to acquire file lock with retries
	var lockFile *os.File
	var err error
	for i := 0; i < 10; i++ {
		lockFile, err = os.OpenFile(r.lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			break
		}
		if !os.IsExist(err) {
			return err
		}
		// Check if lock is stale (older than 60 seconds)
		if info, statErr := os.Stat(r.lockPath); statErr == nil {
			if time.Since(info.ModTime()) > 60*time.Second {
				os.Remove(r.lockPath)
				continue
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lockFile == nil {
		return ErrLockFailed
	}
	defer func() {
		lockFile.Close()
		os.Remove(r.lockPath)
	}()

	return fn()
}

// load reads the registry from disk. Must be called under lock.
func (r *Registry) load() (*RegistryData, error) {
	data, err := os.ReadFile(r.path)
	if os.IsNotExist(err) {
		return &RegistryData{Sessions: []OrchestratorSession{}}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return &RegistryData{Sessions: []OrchestratorSession{}}, nil
	}

	var reg RegistryData
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// save writes the registry to disk. Must be called under lock.
func (r *Registry) save(data *RegistryData) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return err
	}

	// Always include schema for self-describing artifact
	data.Schema = DefaultRegistrySchema()

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, bytes, 0644)
}

// Register adds a new orchestrator session to the registry.
func (r *Registry) Register(session OrchestratorSession) error {
	return r.withLock(func() error {
		data, err := r.load()
		if err != nil {
			return err
		}

		// Check if session already exists and update it
		for i, s := range data.Sessions {
			if s.WorkspaceName == session.WorkspaceName {
				data.Sessions[i] = session
				return r.save(data)
			}
		}

		// Add new session
		data.Sessions = append(data.Sessions, session)
		return r.save(data)
	})
}

// Update modifies an existing session in the registry.
func (r *Registry) Update(workspaceName string, updateFn func(*OrchestratorSession)) error {
	return r.withLock(func() error {
		data, err := r.load()
		if err != nil {
			return err
		}

		for i, s := range data.Sessions {
			if s.WorkspaceName == workspaceName {
				updateFn(&data.Sessions[i])
				return r.save(data)
			}
		}
		return ErrSessionNotFound
	})
}

// Unregister removes a session from the registry.
func (r *Registry) Unregister(workspaceName string) error {
	return r.withLock(func() error {
		data, err := r.load()
		if err != nil {
			return err
		}

		for i, s := range data.Sessions {
			if s.WorkspaceName == workspaceName {
				data.Sessions = append(data.Sessions[:i], data.Sessions[i+1:]...)
				return r.save(data)
			}
		}
		return ErrSessionNotFound
	})
}

// List returns all sessions in the registry.
func (r *Registry) List() ([]OrchestratorSession, error) {
	var sessions []OrchestratorSession
	err := r.withLock(func() error {
		data, err := r.load()
		if err != nil {
			return err
		}
		sessions = make([]OrchestratorSession, len(data.Sessions))
		copy(sessions, data.Sessions)
		return nil
	})
	return sessions, err
}

// Get returns a specific session by workspace name.
func (r *Registry) Get(workspaceName string) (*OrchestratorSession, error) {
	var session *OrchestratorSession
	err := r.withLock(func() error {
		data, err := r.load()
		if err != nil {
			return err
		}
		for _, s := range data.Sessions {
			if s.WorkspaceName == workspaceName {
				cp := s
				session = &cp
				return nil
			}
		}
		return ErrSessionNotFound
	})
	return session, err
}

// ListActive returns only active sessions.
func (r *Registry) ListActive() ([]OrchestratorSession, error) {
	sessions, err := r.List()
	if err != nil {
		return nil, err
	}
	var active []OrchestratorSession
	for _, s := range sessions {
		if s.Status == "active" {
			active = append(active, s)
		}
	}
	return active, nil
}

// ListByProject returns sessions for a specific project directory.
func (r *Registry) ListByProject(projectDir string) ([]OrchestratorSession, error) {
	sessions, err := r.List()
	if err != nil {
		return nil, err
	}
	var filtered []OrchestratorSession
	for _, s := range sessions {
		if s.ProjectDir == projectDir {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}
