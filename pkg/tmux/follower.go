// Package tmux provides tmux session and window management for agent spawning.
// This file implements the tmux follower functionality - polling the orchestrator's
// current working directory and emitting events when the project changes.

package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ProjectChangeEvent is emitted when the orchestrator changes projects.
type ProjectChangeEvent struct {
	Cwd        string `json:"cwd"`
	ProjectDir string `json:"project_dir"` // Directory containing .beads/ or .orch/
	PrevCwd    string `json:"prev_cwd,omitempty"`
	PrevDir    string `json:"prev_project_dir,omitempty"`
}

// FollowerOptions configures the tmux follower.
type FollowerOptions struct {
	// PollInterval is how often to poll tmux (default: 500ms)
	PollInterval time.Duration
	// StabilityThreshold is how many consecutive polls with same cwd
	// before emitting a change event (default: 2, prevents flicker)
	StabilityThreshold int
	// SessionName is the tmux session to follow (default: "orchestrator")
	SessionName string
}

// DefaultFollowerOptions returns the default follower options.
func DefaultFollowerOptions() FollowerOptions {
	return FollowerOptions{
		PollInterval:       500 * time.Millisecond,
		StabilityThreshold: 2,
		SessionName:        OrchestratorSessionName,
	}
}

// FollowerState tracks the current state of the follower.
type FollowerState struct {
	mu sync.RWMutex

	// Current confirmed project
	currentCwd        string
	currentProjectDir string

	// Stability tracking - only switch after N consecutive polls with same cwd
	stabilityCwd        string
	stabilityProjectDir string
	stabilityCounter    int

	// Configuration
	options FollowerOptions

	// Callbacks
	onChange func(event ProjectChangeEvent)
	onError  func(err error)

	// Lifecycle
	running bool
	stopCh  chan struct{}
}

// NewFollower creates a new tmux follower.
func NewFollower(opts FollowerOptions) *FollowerState {
	if opts.PollInterval == 0 {
		opts.PollInterval = 500 * time.Millisecond
	}
	if opts.StabilityThreshold == 0 {
		opts.StabilityThreshold = 2
	}
	if opts.SessionName == "" {
		opts.SessionName = OrchestratorSessionName
	}

	return &FollowerState{
		options: opts,
		stopCh:  make(chan struct{}),
	}
}

// SetOnChange sets the callback for project changes.
func (f *FollowerState) SetOnChange(fn func(event ProjectChangeEvent)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onChange = fn
}

// SetOnError sets the callback for errors.
func (f *FollowerState) SetOnError(fn func(err error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onError = fn
}

// Start begins polling tmux for project changes.
func (f *FollowerState) Start() {
	f.mu.Lock()
	if f.running {
		f.mu.Unlock()
		return
	}
	f.running = true
	f.stopCh = make(chan struct{})
	f.mu.Unlock()

	go f.pollLoop()
}

// Stop stops the follower.
func (f *FollowerState) Stop() {
	f.mu.Lock()
	if !f.running {
		f.mu.Unlock()
		return
	}
	f.running = false
	close(f.stopCh)
	f.mu.Unlock()
}

// GetCurrentProject returns the current project directory.
func (f *FollowerState) GetCurrentProject() (cwd, projectDir string) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.currentCwd, f.currentProjectDir
}

// pollLoop runs the polling loop.
func (f *FollowerState) pollLoop() {
	ticker := time.NewTicker(f.options.PollInterval)
	defer ticker.Stop()

	// Initial poll immediately
	f.poll()

	for {
		select {
		case <-f.stopCh:
			return
		case <-ticker.C:
			f.poll()
		}
	}
}

// poll performs a single poll iteration.
func (f *FollowerState) poll() {
	cwd, err := GetTmuxCwd(f.options.SessionName)
	if err != nil {
		f.mu.RLock()
		onError := f.onError
		f.mu.RUnlock()
		if onError != nil {
			onError(err)
		}
		return
	}

	f.processCwdPoll(cwd)
}

// processCwdPoll processes a cwd poll result with stability checking.
func (f *FollowerState) processCwdPoll(cwd string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Find project directory (contains .beads/ or .orch/)
	projectDir := findProjectDir(cwd)

	// Sticky behavior: if no project found (e.g., in home dir), ignore this poll
	if projectDir == "" {
		return
	}

	// If same as current project, no change needed - reset stability tracking
	if projectDir == f.currentProjectDir {
		f.stabilityCwd = ""
		f.stabilityProjectDir = ""
		f.stabilityCounter = 0
		return
	}

	// First detection - emit immediately (no stability check needed)
	if f.currentProjectDir == "" {
		f.emitChange(cwd, projectDir)
		return
	}

	// Check if this poll matches what we're tracking for stability
	if projectDir == f.stabilityProjectDir {
		// Same as tracked - increment stability counter
		f.stabilityCounter++

		// If we've reached stability threshold, emit the change
		if f.stabilityCounter >= f.options.StabilityThreshold {
			f.emitChange(cwd, projectDir)
			f.stabilityCwd = ""
			f.stabilityProjectDir = ""
			f.stabilityCounter = 0
		}
	} else {
		// Different from tracked - start fresh stability tracking
		f.stabilityCwd = cwd
		f.stabilityProjectDir = projectDir
		f.stabilityCounter = 1
	}
}

// emitChange emits a project change event (caller must hold lock).
func (f *FollowerState) emitChange(cwd, projectDir string) {
	event := ProjectChangeEvent{
		Cwd:        cwd,
		ProjectDir: projectDir,
		PrevCwd:    f.currentCwd,
		PrevDir:    f.currentProjectDir,
	}

	f.currentCwd = cwd
	f.currentProjectDir = projectDir

	onChange := f.onChange
	if onChange != nil {
		// Call outside lock to prevent deadlocks
		go onChange(event)
	}
}

// GetTmuxCwd gets the current working directory of the active window in a tmux session.
// It uses a two-step approach:
// 1. Get the active window index via #{window_index}
// 2. Get that specific window's pane cwd via session:index target
//
// This is necessary because using just the session name (-t session) returns the
// first pane in the session, not the active window's pane.
func GetTmuxCwd(sessionName string) (string, error) {
	// Step 1: Get active window index
	indexCmd, err := tmuxCommand("display-message", "-t", sessionName, "-p", "#{window_index}")
	if err != nil {
		return "", fmt.Errorf("failed to create tmux command for window index: %w", err)
	}
	indexOutput, err := indexCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get active window index: %w", err)
	}
	windowIndex := strings.TrimSpace(string(indexOutput))

	// Empty window index indicates the session doesn't exist
	// (tmux display-message returns empty for non-existent sessions)
	if windowIndex == "" {
		return "", fmt.Errorf("session %q not found or has no windows", sessionName)
	}

	// Step 2: Get that window's pane cwd
	target := fmt.Sprintf("%s:%s", sessionName, windowIndex)
	cwdCmd, err := tmuxCommand("display-message", "-t", target, "-p", "#{pane_current_path}")
	if err != nil {
		return "", fmt.Errorf("failed to create tmux command for pane cwd: %w", err)
	}
	cwdOutput, err := cwdCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get pane cwd: %w", err)
	}
	cwd := strings.TrimSpace(string(cwdOutput))

	// Fallback: if pane_current_path is empty (e.g., running Claude Code),
	// use lsof to get the process's actual working directory
	if cwd == "" {
		cwd, _ = getPaneCwdViaLsof(sessionName, windowIndex)
	}

	return cwd, nil
}

// getPaneCwdViaLsof gets the pane's working directory by querying the process directly.
// This is a fallback for when #{pane_current_path} is empty (e.g., running Claude Code).
func getPaneCwdViaLsof(sessionName, windowIndex string) (string, error) {
	target := fmt.Sprintf("%s:%s", sessionName, windowIndex)
	pidCmd, err := tmuxCommand("display-message", "-t", target, "-p", "#{pane_pid}")
	if err != nil {
		return "", err
	}
	pidOutput, err := pidCmd.Output()
	if err != nil {
		return "", err
	}
	pid := strings.TrimSpace(string(pidOutput))
	if pid == "" {
		return "", fmt.Errorf("no pane pid")
	}

	// Use lsof to get the cwd of the process
	lsofCmd := exec.Command("lsof", "-p", pid)
	lsofOutput, err := lsofCmd.Output()
	if err != nil {
		return "", err
	}

	// Parse lsof output for cwd line
	for _, line := range strings.Split(string(lsofOutput), "\n") {
		if strings.Contains(line, "cwd") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[len(fields)-1], nil
			}
		}
	}
	return "", fmt.Errorf("cwd not found in lsof output")
}

// findProjectDir walks up from cwd to find a directory containing .beads/ or .orch/.
// Returns empty string if no project directory found (e.g., in home directory).
func findProjectDir(cwd string) string {
	current := cwd
	homeDir := os.Getenv("HOME")

	for {
		// Check for .beads/ directory (beads-managed project)
		beadsPath := filepath.Join(current, ".beads")
		if info, err := os.Stat(beadsPath); err == nil && info.IsDir() {
			return current
		}

		// Check for .orch/ directory (orchestrator workspace)
		orchPath := filepath.Join(current, ".orch")
		if info, err := os.Stat(orchPath); err == nil && info.IsDir() {
			return current
		}

		// Don't go above home directory
		if current == homeDir {
			return ""
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root without finding project
			return ""
		}
		current = parent
	}
}

// MultiProjectConfig defines which projects should be shown when following a given project.
// This handles the special case where orch-go should see agents from multiple repos.
type MultiProjectConfig struct {
	// Project is the project name (e.g., "orch-go")
	Project string `json:"project"`
	// IncludeProjects lists additional projects to include (e.g., ["orch-cli", "beads", "kb-cli"])
	IncludeProjects []string `json:"include_projects"`
}

// DefaultMultiProjectConfigs returns the default multi-project configurations.
// These define which orchestration project sees agents from multiple repos.
// Also includes aliases for projects where the directory name differs from the beads ID prefix.
func DefaultMultiProjectConfigs() []MultiProjectConfig {
	return []MultiProjectConfig{
		{
			// orch-go is the orchestration home - it should see agents from all ecosystem repos
			Project: "orch-go",
			IncludeProjects: []string{
				"orch-cli",
				"beads",
				"kb-cli",
				"orch-knowledge",
				"opencode",
				"price-watch", // Include price-watch agents when viewing from orch-go
				"pw",          // Alias: price-watch directory uses "pw" beads ID prefix
			},
		},
		{
			// price-watch directory uses "pw" as beads ID prefix
			// Include "pw" so agents with beads IDs like "pw-xxxx" are visible
			Project:         "price-watch",
			IncludeProjects: []string{"pw"},
		},
	}
}

// GetIncludedProjects returns the list of projects to include for a given project.
// If no multi-project config exists, returns just the project itself.
func GetIncludedProjects(project string, configs []MultiProjectConfig) []string {
	for _, cfg := range configs {
		if cfg.Project == project {
			// Include the base project plus configured additional projects
			result := []string{project}
			result = append(result, cfg.IncludeProjects...)
			return result
		}
	}
	// No multi-project config - just the single project
	return []string{project}
}
