// Package focus provides north star tracking for multi-project prioritization.
//
// The focus package stores the current priority goal that guides work selection.
// It provides:
// - Set/Get/Clear operations for the current focus
// - Drift detection (comparing active work against focus)
// - Next action suggestions based on current state
package focus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TimeFormat is the timestamp format used in focus storage.
const TimeFormat = time.RFC3339Nano

// Focus represents the current north star priority.
type Focus struct {
	// Goal is the human-readable description of the priority
	Goal string `json:"goal"`

	// BeadsID is the optional beads issue ID associated with the focus
	BeadsID string `json:"beads_id,omitempty"`

	// SetAt is when the focus was established
	SetAt string `json:"set_at"`
}

// DriftResult represents the result of a drift check.
type DriftResult struct {
	// IsDrifting is true if active work doesn't align with focus
	IsDrifting bool `json:"is_drifting"`

	// FocusedIssue is the beads ID from the current focus (if any)
	FocusedIssue string `json:"focused_issue,omitempty"`

	// ActiveIssues are the beads IDs of currently active work
	ActiveIssues []string `json:"active_issues,omitempty"`

	// Goal is the current focus goal
	Goal string `json:"goal,omitempty"`
}

// Suggestion represents a next action recommendation.
type Suggestion struct {
	// Action is the recommended action type
	// Possible values: set-focus, start-work, continue, refocus
	Action string `json:"action"`

	// Description is a human-readable explanation
	Description string `json:"description"`

	// BeadsID is the relevant beads issue (if applicable)
	BeadsID string `json:"beads_id,omitempty"`

	// Goal is the current focus goal (if any)
	Goal string `json:"goal,omitempty"`
}

// storeData is the on-disk format.
type storeData struct {
	Focus *Focus `json:"focus"`
}

// Store manages persistent focus state.
type Store struct {
	path  string
	focus *Focus
	mu    sync.RWMutex
}

// DefaultPath returns the default focus file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "focus.json")
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

// load reads the focus from disk.
func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		s.focus = nil
		return nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		s.focus = nil
		return nil
	}

	var sd storeData
	if err := json.Unmarshal(data, &sd); err != nil {
		return err
	}

	s.focus = sd.Focus
	return nil
}

// save writes the focus to disk.
func (s *Store) save() error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	sd := storeData{Focus: s.focus}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// Get returns the current focus, or nil if not set.
func (s *Store) Get() *Focus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.focus
}

// Set sets the current focus.
func (s *Store) Set(f *Focus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set timestamp
	f.SetAt = time.Now().Format(TimeFormat)
	s.focus = f

	return s.save()
}

// Clear removes the current focus.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.focus = nil
	return s.save()
}

// CheckDrift compares active work against the current focus.
// Returns a DriftResult indicating whether work is aligned with focus.
func (s *Store) CheckDrift(activeIssues []string) DriftResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := DriftResult{
		ActiveIssues: activeIssues,
	}

	// No focus set = no drift possible
	if s.focus == nil {
		result.IsDrifting = false
		return result
	}

	result.Goal = s.focus.Goal
	result.FocusedIssue = s.focus.BeadsID

	// Focus without specific issue = any work is fine
	if s.focus.BeadsID == "" {
		result.IsDrifting = false
		return result
	}

	// Check if focused issue is among active issues
	for _, issue := range activeIssues {
		if issue == s.focus.BeadsID {
			result.IsDrifting = false
			return result
		}
	}

	// Focused issue not found in active work = drift
	result.IsDrifting = true
	return result
}

// SuggestNext recommends the next action based on current state.
func (s *Store) SuggestNext(activeIssues []string) Suggestion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// No focus = suggest setting one
	if s.focus == nil {
		return Suggestion{
			Action:      "set-focus",
			Description: "No focus set. Use 'orch focus <goal>' to set a north star priority.",
		}
	}

	suggestion := Suggestion{
		Goal: s.focus.Goal,
	}

	// Focus with specific issue
	if s.focus.BeadsID != "" {
		suggestion.BeadsID = s.focus.BeadsID

		// Check if working on focused issue
		for _, issue := range activeIssues {
			if issue == s.focus.BeadsID {
				suggestion.Action = "continue"
				suggestion.Description = "Already working on focused issue. Keep going!"
				return suggestion
			}
		}

		// Check if any work is active
		if len(activeIssues) == 0 {
			suggestion.Action = "start-work"
			suggestion.Description = "No active work. Start work on focused issue: " + s.focus.BeadsID
			return suggestion
		}

		// Working on something else = suggest refocus
		suggestion.Action = "refocus"
		suggestion.Description = "Currently working on other issues. Consider switching to focused issue: " + s.focus.BeadsID
		return suggestion
	}

	// Focus without specific issue - any work is fine
	if len(activeIssues) == 0 {
		suggestion.Action = "start-work"
		suggestion.Description = "No active work. Start working toward: " + s.focus.Goal
		return suggestion
	}

	suggestion.Action = "continue"
	suggestion.Description = "Working toward: " + s.focus.Goal
	return suggestion
}
