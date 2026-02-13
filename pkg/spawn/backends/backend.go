// Package backends provides spawn backend implementations for different
// execution modes (inline, headless, tmux).
package backends

import (
	"context"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// Backend is the interface for spawn backends.
// Each backend creates an agent session via a different mechanism.
type Backend interface {
	// Name returns the backend name for logging ("inline", "headless", "tmux").
	Name() string

	// Spawn creates a new agent session.
	// The context can be used for cancellation/timeout.
	// Returns Result on success, or error on failure.
	Spawn(ctx context.Context, req *SpawnRequest) (*Result, error)
}

// SpawnRequest contains all inputs needed for spawning an agent.
type SpawnRequest struct {
	// Config is the spawn configuration.
	Config *spawn.Config

	// ServerURL is the OpenCode server URL.
	ServerURL string

	// MinimalPrompt is the prompt to send to the agent.
	MinimalPrompt string

	// BeadsID is the beads issue ID for tracking.
	BeadsID string

	// SkillName is the skill being spawned.
	SkillName string

	// Task is the user's task description.
	Task string

	// Attach is true to attach terminal after spawn (tmux only).
	Attach bool
}

// Result contains the output of a successful spawn.
type Result struct {
	// SessionID is the OpenCode session ID.
	SessionID string

	// SpawnMode is "inline", "headless", or "tmux".
	SpawnMode string

	// TmuxInfo contains tmux-specific metadata (nil for non-tmux backends).
	TmuxInfo *TmuxInfo

	// RetryAttempts is the number of retry attempts taken (for headless backend).
	RetryAttempts int
}

// TmuxInfo contains tmux-specific spawn results.
type TmuxInfo struct {
	// SessionName is the tmux session name (e.g., "workers-orch-go").
	SessionName string

	// WindowTarget is the window target (e.g., "workers-orch-go:1").
	WindowTarget string

	// WindowID is tmux's internal window identifier.
	WindowID string
}

// Select returns the appropriate backend based on spawn mode flags.
// Priority: inline > headless > (tmux/attach/isOrchestrator) > headless (default).
func Select(inline, headless, tmux, attach bool, isOrchestrator bool) Backend {
	if inline {
		return &InlineBackend{}
	}
	if headless {
		return &HeadlessBackend{}
	}
	// Orchestrators default to tmux; workers default to headless
	useTmux := tmux || attach || isOrchestrator
	if useTmux {
		return &TmuxBackend{}
	}
	return &HeadlessBackend{}
}
