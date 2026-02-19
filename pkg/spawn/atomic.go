package spawn

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// AtomicSpawnOpts holds parameters for atomic spawn.
type AtomicSpawnOpts struct {
	Config    *Config
	BeadsID   string
	NoTrack   bool
	ServerURL string
}

// AtomicSpawnResult holds the output of a successful atomic spawn.
type AtomicSpawnResult struct {
	SessionID     string
	WorkspacePath string
	ManifestPath  string
}

// AtomicSpawnPhase1 performs the common pre-session writes:
//  1. Tag beads issue with orch:agent label
//  2. Write workspace (SPAWN_CONTEXT.md, dotfiles, AGENT_MANIFEST.json)
//
// Returns a rollback function that undoes all Phase 1 writes on failure.
// Must be called before session creation.
func AtomicSpawnPhase1(opts *AtomicSpawnOpts) (rollback func(), err error) {
	var cleanups []func()

	rollback = func() {
		// Execute cleanups in reverse order
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	// Step 1: Tag beads issue with orch:agent
	if !opts.NoTrack && opts.BeadsID != "" {
		if err := tagBeadsAgent(opts.BeadsID); err != nil {
			return rollback, fmt.Errorf("beads tag failed: %w", err)
		}
		cleanups = append(cleanups, func() {
			untagBeadsAgent(opts.BeadsID)
		})
	}

	// Step 2: Write workspace (SPAWN_CONTEXT.md + manifest without session_id)
	if err := WriteContext(opts.Config); err != nil {
		rollback()
		return nil, fmt.Errorf("workspace write failed: %w", err)
	}
	cleanups = append(cleanups, func() {
		os.RemoveAll(opts.Config.WorkspacePath())
	})

	return rollback, nil
}

// AtomicSpawnPhase2 performs the session-specific writes after session creation.
// Updates the manifest with session_id and writes session ID file.
// This is called by each backend after session creation succeeds.
// Phase 2 is best-effort: if writes fail, the session is already running
// and metadata is supplementary. The rollback responsibility stays with the caller
// for the session itself (DeleteSession if needed).
func AtomicSpawnPhase2(opts *AtomicSpawnOpts, sessionID string) error {
	workspacePath := opts.Config.WorkspacePath()

	// Step 3: Write session ID to dotfile
	if sessionID != "" {
		if err := WriteSessionID(workspacePath, sessionID); err != nil {
			return fmt.Errorf("session ID write failed: %w", err)
		}
	}

	// Step 4: Update manifest with session_id
	manifest, err := ReadAgentManifest(workspacePath)
	if err == nil && manifest != nil {
		manifest.SessionID = sessionID
		if writeErr := WriteAgentManifest(workspacePath, *manifest); writeErr != nil {
			// Non-fatal: manifest still has all other fields
			fmt.Fprintf(os.Stderr, "Warning: failed to update manifest with session ID: %v\n", writeErr)
		}
	}

	return nil
}

// tagBeadsAgent adds the orch:agent label to a beads issue.
func tagBeadsAgent(beadsID string) error {
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			return client.AddLabel(beadsID, "orch:agent")
		}
	}
	// Fallback to CLI
	return beads.FallbackAddLabel(beadsID, "orch:agent")
}

// untagBeadsAgent removes the orch:agent label from a beads issue.
// Used during rollback when spawn fails after tagging.
func untagBeadsAgent(beadsID string) {
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			_ = client.RemoveLabel(beadsID, "orch:agent")
			return
		}
	}
	// Fallback to CLI
	_ = beads.FallbackRemoveLabel(beadsID, "orch:agent")
}
