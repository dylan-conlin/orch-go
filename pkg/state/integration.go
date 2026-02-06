package state

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// RecordSpawn inserts a new agent row into the state database.
// Called by orch spawn after workspace/beads setup but before session creation.
// Non-fatal: logs warning on failure but does not block spawn.
func RecordSpawn(cfg *spawn.Config) error {
	db, err := OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open state db: %w", err)
	}
	if db == nil {
		return nil // Could not determine DB path, skip silently
	}
	defer db.Close()

	agent := &Agent{
		WorkspaceName: cfg.WorkspaceName,
		BeadsID:       cfg.BeadsID,
		Mode:          cfg.SpawnMode,
		Skill:         cfg.SkillName,
		Model:         cfg.Model,
		Tier:          cfg.Tier,
		ProjectDir:    cfg.ProjectDir,
		ProjectName:   cfg.Project,
		SpawnTime:     time.Now().UnixMilli(),
		GitBaseline:   "", // Will be set from workspace after spawn
	}

	// Default mode to "opencode" if not set
	if agent.Mode == "" {
		agent.Mode = "opencode"
	}

	return db.InsertAgent(agent)
}

// RecordSpawnWithManifest inserts a new agent row using manifest data for richer context.
// Called by orch spawn after the AgentManifest is written.
func RecordSpawnWithManifest(cfg *spawn.Config, manifest *spawn.AgentManifest) error {
	db, err := OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open state db: %w", err)
	}
	if db == nil {
		return nil
	}
	defer db.Close()

	agent := &Agent{
		WorkspaceName: cfg.WorkspaceName,
		BeadsID:       cfg.BeadsID,
		Mode:          cfg.SpawnMode,
		Skill:         cfg.SkillName,
		Model:         cfg.Model,
		Tier:          cfg.Tier,
		ProjectDir:    cfg.ProjectDir,
		ProjectName:   cfg.Project,
		SpawnTime:     time.Now().UnixMilli(),
		GitBaseline:   manifest.GitBaseline,
	}

	if agent.Mode == "" {
		agent.Mode = "opencode"
	}

	return db.InsertAgent(agent)
}

// RecordComplete marks an agent as completed in the state database.
// Called by orch complete. Tries workspace name first, falls back to beads ID.
// Non-fatal: logs warning on failure but does not block completion.
func RecordComplete(workspaceName, beadsID string) error {
	db, err := OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open state db: %w", err)
	}
	if db == nil {
		return nil
	}
	defer db.Close()

	// Try workspace name first (more specific)
	if workspaceName != "" {
		return db.UpdateCompleted(workspaceName)
	}
	// Fall back to beads ID
	if beadsID != "" {
		return db.UpdateCompletedByBeadsID(beadsID)
	}
	return nil
}

// RecordAbandon marks an agent as abandoned in the state database.
// Called by orch abandon. Tries workspace name first, falls back to beads ID.
// Non-fatal: logs warning on failure but does not block abandonment.
func RecordAbandon(workspaceName, beadsID string) error {
	db, err := OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open state db: %w", err)
	}
	if db == nil {
		return nil
	}
	defer db.Close()

	// Try workspace name first (more specific)
	if workspaceName != "" {
		return db.UpdateAbandoned(workspaceName)
	}
	// Fall back to beads ID
	if beadsID != "" {
		return db.UpdateAbandonedByBeadsID(beadsID)
	}
	return nil
}

// RecordSessionID updates the session ID for an agent in the state database.
// Called after OpenCode session creation succeeds.
func RecordSessionID(workspaceName, sessionID string) error {
	if workspaceName == "" || sessionID == "" {
		return nil
	}
	db, err := OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open state db: %w", err)
	}
	if db == nil {
		return nil
	}
	defer db.Close()
	return db.UpdateSessionID(workspaceName, sessionID)
}

// RecordTmuxWindow updates the tmux window for an agent in the state database.
// Called after tmux window creation succeeds.
func RecordTmuxWindow(workspaceName, tmuxWindow string) error {
	if workspaceName == "" || tmuxWindow == "" {
		return nil
	}
	db, err := OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open state db: %w", err)
	}
	if db == nil {
		return nil
	}
	defer db.Close()
	return db.UpdateTmuxWindow(workspaceName, tmuxWindow)
}
