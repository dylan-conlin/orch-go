// Package registry provides persistent agent tracking for orch-go.
//
// This is a minimal registry for agent management that stores:
// - Agent ID (workspace name) <-> session_id mapping
// - Basic agent metadata (beads_id for lifecycle tracking)
// - Status tracking (active, completed, abandoned, deleted)
//
// Beads is the source of truth for detailed agent state and lifecycle.
// This registry provides local agent tracking for CLI operations.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// AgentState represents the lifecycle state of an agent.
type AgentState string

const (
	StateActive    AgentState = "active"
	StateCompleted AgentState = "completed"
	StateAbandoned AgentState = "abandoned"
	StateDeleted   AgentState = "deleted"
)

// TimeFormat is the timestamp format used in the registry.
// Using RFC3339Nano for sub-second precision in concurrent operations.
const TimeFormat = time.RFC3339Nano

// Agent represents a tracked agent in the registry.
type Agent struct {
	// Core identification
	ID        string `json:"id"`                   // Unique identifier (workspace name)
	BeadsID   string `json:"beads_id,omitempty"`   // Foreign key to beads issue
	SessionID string `json:"session_id,omitempty"` // OpenCode session ID (for headless agents)
	WindowID  string `json:"window_id,omitempty"`  // Tmux window ID for operations
	Window    string `json:"window,omitempty"`     // Tmux window name (session:window format)

	// State
	Status AgentState `json:"status"`

	// Timestamps
	SpawnedAt   string `json:"spawned_at"`
	UpdatedAt   string `json:"updated_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	AbandonedAt string `json:"abandoned_at,omitempty"`
	DeletedAt   string `json:"deleted_at,omitempty"`

	// Optional metadata (only stored for non-beads agents)
	ProjectDir      string `json:"project_dir,omitempty"`
	Skill           string `json:"skill,omitempty"`
	PrimaryArtifact string `json:"primary_artifact,omitempty"`
	IsInteractive   bool   `json:"is_interactive,omitempty"`
	BeadsDBPath     string `json:"beads_db_path,omitempty"` // For cross-project lookups
}

// registryData is the on-disk format.
type registryData struct {
	Agents []*Agent `json:"agents"`
}

// Registry manages persistent state for spawned agents.
type Registry struct {
	path    string
	agents  []*Agent
	mu      sync.RWMutex
	timeout time.Duration
}

// DefaultPath returns the default registry file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "agent-registry.json")
}

// New creates a new Registry with the given path.
// If path is empty, uses DefaultPath().
func New(path string) (*Registry, error) {
	if path == "" {
		path = DefaultPath()
	}

	r := &Registry{
		path:    path,
		agents:  make([]*Agent, 0),
		timeout: 10 * time.Second,
	}

	if err := r.load(); err != nil {
		return nil, err
	}

	return r, nil
}

// load reads the registry from disk with shared file locking.
func (r *Registry) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(r.path); os.IsNotExist(err) {
		r.agents = make([]*Agent, 0)
		return nil
	}

	// Open with shared lock for reading
	f, err := os.OpenFile(r.path, os.O_RDONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			r.agents = make([]*Agent, 0)
			return nil
		}
		return fmt.Errorf("failed to open registry: %w", err)
	}
	defer f.Close()

	// Acquire shared lock for reading
	if err := r.lockWithTimeout(f, syscall.LOCK_SH); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, err := os.ReadFile(r.path)
	if err != nil {
		return fmt.Errorf("failed to read registry: %w", err)
	}

	if len(data) == 0 {
		r.agents = make([]*Agent, 0)
		return nil
	}

	var rd registryData
	if err := json.Unmarshal(data, &rd); err != nil {
		return fmt.Errorf("failed to parse registry: %w", err)
	}

	r.agents = rd.Agents
	if r.agents == nil {
		r.agents = make([]*Agent, 0)
	}

	return nil
}

// Save persists the registry to disk with file locking.
func (r *Registry) Save() error {
	return r.save(false)
}

// save writes the registry to disk.
// If skipMerge is true, overwrites without merging (used for delete operations).
func (r *Registry) save(skipMerge bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open or create the file
	var f *os.File
	var err error
	if _, err = os.Stat(r.path); os.IsNotExist(err) {
		f, err = os.OpenFile(r.path, os.O_RDWR|os.O_CREATE, 0644)
	} else {
		f, err = os.OpenFile(r.path, os.O_RDWR, 0644)
	}
	if err != nil {
		return fmt.Errorf("failed to open registry file: %w", err)
	}
	defer f.Close()

	// Acquire exclusive lock with timeout
	if err := r.lockWithTimeout(f, syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	var agentsToWrite []*Agent

	if skipMerge {
		agentsToWrite = r.agents
	} else {
		// Re-read file content while holding the lock
		// Read directly from the locked file descriptor
		if _, err := f.Seek(0, 0); err != nil {
			return fmt.Errorf("failed to seek for read: %w", err)
		}
		stat, err := f.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat: %w", err)
		}
		currentData := make([]byte, stat.Size())
		if stat.Size() > 0 {
			n, err := f.Read(currentData)
			if err != nil {
				return fmt.Errorf("failed to read current data: %w", err)
			}
			currentData = currentData[:n]
		}

		var currentRD registryData
		if len(currentData) > 0 {
			if err := json.Unmarshal(currentData, &currentRD); err == nil {
				agentsToWrite = r.mergeAgents(currentRD.Agents, r.agents)
			} else {
				agentsToWrite = r.agents
			}
		} else {
			agentsToWrite = r.agents
		}
	}

	// Write to file
	rd := registryData{Agents: agentsToWrite}
	data, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}

// lockWithTimeout acquires a file lock with timeout.
func (r *Registry) lockWithTimeout(f *os.File, lockType int) error {
	deadline := time.Now().Add(r.timeout)
	for {
		err := syscall.Flock(int(f.Fd()), lockType|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if err != syscall.EWOULDBLOCK {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("could not acquire registry lock after %v", r.timeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// mergeAgents combines current (from disk) and ours (in-memory) agents.
// Newer entries (by UpdatedAt) win in conflicts.
func (r *Registry) mergeAgents(current, ours []*Agent) []*Agent {
	merged := make(map[string]*Agent)

	// Index our agents
	oursByID := make(map[string]*Agent)
	for _, a := range ours {
		oursByID[a.ID] = a
	}

	// Process current agents from disk
	for _, currentAgent := range current {
		ourAgent, exists := oursByID[currentAgent.ID]
		if exists {
			// Compare timestamps, newer wins
			if currentAgent.UpdatedAt > ourAgent.UpdatedAt {
				merged[currentAgent.ID] = currentAgent
			} else {
				merged[currentAgent.ID] = ourAgent
			}
		} else {
			merged[currentAgent.ID] = currentAgent
		}
	}

	// Add agents only we have
	for _, ourAgent := range ours {
		if _, exists := merged[ourAgent.ID]; !exists {
			merged[ourAgent.ID] = ourAgent
		}
	}

	result := make([]*Agent, 0, len(merged))
	for _, a := range merged {
		result = append(result, a)
	}
	return result
}

// Register adds a new agent to the registry.
// Returns error if agent ID already exists and is active.
// Abandoned, completed, or deleted agents can be re-registered (respawned).
func (r *Registry) Register(agent *Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().Format(TimeFormat)

	// Check for duplicate by agent ID
	var existingAgent *Agent
	for _, a := range r.agents {
		if a.ID == agent.ID {
			existingAgent = a
			break
		}
	}

	if existingAgent != nil {
		// If the existing agent is active, reject the registration
		if existingAgent.Status == StateActive {
			return fmt.Errorf("agent '%s' already registered", agent.ID)
		}
		// If the existing agent is abandoned, completed, or deleted, reuse the slot
		// This allows respawning with the same agent ID
		existingAgent.BeadsID = agent.BeadsID
		existingAgent.SessionID = agent.SessionID
		existingAgent.WindowID = agent.WindowID
		existingAgent.Window = agent.Window
		existingAgent.Status = StateActive
		existingAgent.SpawnedAt = now
		existingAgent.UpdatedAt = now
		existingAgent.CompletedAt = ""
		existingAgent.AbandonedAt = ""
		existingAgent.DeletedAt = ""
		existingAgent.ProjectDir = agent.ProjectDir
		existingAgent.Skill = agent.Skill
		existingAgent.PrimaryArtifact = agent.PrimaryArtifact
		existingAgent.IsInteractive = agent.IsInteractive
		existingAgent.BeadsDBPath = agent.BeadsDBPath
		return nil
	}

	// Check for window_id reuse - abandon existing agent
	if agent.WindowID != "" {
		for _, a := range r.agents {
			if a.Status == StateActive && a.WindowID == agent.WindowID {
				a.Status = StateAbandoned
				a.AbandonedAt = now
				a.UpdatedAt = now
			}
		}
	}

	// Set timestamps
	agent.SpawnedAt = now
	agent.UpdatedAt = now
	if agent.Status == "" {
		agent.Status = StateActive
	}

	r.agents = append(r.agents, agent)
	return nil
}

// Find looks up an agent by ID or beads_id.
// Returns nil if not found.
func (r *Registry) Find(identifier string) *Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First: exact match on agent ID
	for _, a := range r.agents {
		if a.ID == identifier {
			return a
		}
	}

	// Second: match on beads_id
	for _, a := range r.agents {
		if a.BeadsID == identifier {
			return a
		}
	}

	return nil
}

// ListAgents returns all non-deleted agents.
func (r *Registry) ListAgents() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Agent, 0)
	for _, a := range r.agents {
		if a.Status != StateDeleted {
			result = append(result, a)
		}
	}
	return result
}

// ListActive returns only active agents.
func (r *Registry) ListActive() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Agent, 0)
	for _, a := range r.agents {
		if a.Status == StateActive {
			result = append(result, a)
		}
	}
	return result
}

// ListCompleted returns only completed agents (not active, abandoned, or deleted).
func (r *Registry) ListCompleted() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Agent, 0)
	for _, a := range r.agents {
		if a.Status == StateCompleted {
			result = append(result, a)
		}
	}
	return result
}

// ListCleanable returns agents that can be cleaned (completed or abandoned).
func (r *Registry) ListCleanable() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Agent, 0)
	for _, a := range r.agents {
		if a.Status == StateCompleted || a.Status == StateAbandoned {
			result = append(result, a)
		}
	}
	return result
}

// Abandon marks an agent as abandoned.
// Returns true if agent was found and abandoned, false otherwise.
func (r *Registry) Abandon(agentID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, a := range r.agents {
		if a.ID == agentID && a.Status == StateActive {
			now := time.Now().Format(TimeFormat)
			a.Status = StateAbandoned
			a.AbandonedAt = now
			a.UpdatedAt = now
			return true
		}
	}
	return false
}

// Complete marks an agent as completed.
// Returns true if agent was found and marked completed, false otherwise.
func (r *Registry) Complete(agentID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, a := range r.agents {
		if a.ID == agentID && a.Status == StateActive {
			now := time.Now().Format(TimeFormat)
			a.Status = StateCompleted
			a.CompletedAt = now
			a.UpdatedAt = now
			return true
		}
	}
	return false
}

// Remove marks an agent as deleted (tombstone pattern).
// Returns true if agent was found and deleted, false otherwise.
func (r *Registry) Remove(agentID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, a := range r.agents {
		if a.ID == agentID {
			now := time.Now().Format(TimeFormat)
			a.Status = StateDeleted
			a.DeletedAt = now
			a.UpdatedAt = now
			return true
		}
	}
	return false
}

// HeadlessWindowID is the special window ID marker for headless spawns.
// This is used to identify agents spawned via HTTP API (without a terminal/TUI).
const HeadlessWindowID = "headless"

// SaveSkipMerge saves without merging (used for delete operations).
func (r *Registry) SaveSkipMerge() error {
	return r.save(true)
}
