package state

import (
	"database/sql"
	"fmt"
	"time"
)

// Agent represents a row in the agents table.
type Agent struct {
	// Core identity (set at spawn, immutable)
	WorkspaceName string
	BeadsID       string
	SessionID     string
	TmuxWindow    string
	Mode          string
	Skill         string
	Model         string
	Tier          string
	ProjectDir    string
	ProjectName   string
	SpawnTime     int64 // unix ms
	GitBaseline   string
	IssueTitle    string
	IssueType     string
	IssuePriority int

	// Mutable lifecycle state
	Phase            string
	PhaseSummary     string
	PhaseReportedAt  int64 // unix ms
	IsProcessing     bool
	SessionUpdatedAt int64 // unix ms
	IsCompleted      bool
	IsAbandoned      bool
	CompletedAt      int64 // unix ms
	AbandonedAt      int64 // unix ms

	// Token aggregates
	TokensInput     int
	TokensOutput    int
	TokensReasoning int
	TokensCacheRead int
	TokensTotal     int

	// Timestamps
	CreatedAt int64 // unix ms
	UpdatedAt int64 // unix ms
}

// nowMs returns the current time in unix milliseconds.
func nowMs() int64 {
	return time.Now().UnixMilli()
}

// boolToInt converts a bool to an int for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// InsertAgent inserts a new agent row. Called by orch spawn.
func (d *DB) InsertAgent(agent *Agent) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := nowMs()
	if agent.CreatedAt == 0 {
		agent.CreatedAt = now
	}
	if agent.UpdatedAt == 0 {
		agent.UpdatedAt = now
	}

	_, err := d.db.Exec(`
		INSERT INTO agents (
			workspace_name, beads_id, session_id, tmux_window, mode, skill, model,
			tier, project_dir, project_name, spawn_time, git_baseline,
			issue_title, issue_type, issue_priority,
			phase, phase_summary, phase_reported_at,
			is_processing, session_updated_at,
			is_completed, is_abandoned, completed_at, abandoned_at,
			tokens_input, tokens_output, tokens_reasoning, tokens_cache_read, tokens_total,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?
		)`,
		agent.WorkspaceName, nullString(agent.BeadsID), nullString(agent.SessionID),
		nullString(agent.TmuxWindow), agent.Mode, nullString(agent.Skill),
		nullString(agent.Model),
		nullString(agent.Tier), agent.ProjectDir, nullString(agent.ProjectName),
		agent.SpawnTime, nullString(agent.GitBaseline),
		nullString(agent.IssueTitle), nullString(agent.IssueType), agent.IssuePriority,
		nullString(agent.Phase), nullString(agent.PhaseSummary), nullInt64(agent.PhaseReportedAt),
		boolToInt(agent.IsProcessing), nullInt64(agent.SessionUpdatedAt),
		boolToInt(agent.IsCompleted), boolToInt(agent.IsAbandoned),
		nullInt64(agent.CompletedAt), nullInt64(agent.AbandonedAt),
		agent.TokensInput, agent.TokensOutput, agent.TokensReasoning,
		agent.TokensCacheRead, agent.TokensTotal,
		agent.CreatedAt, agent.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert agent %s: %w", agent.WorkspaceName, err)
	}
	return nil
}

// UpdateCompleted marks an agent as completed. Called by orch complete.
func (d *DB) UpdateCompleted(workspaceName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := nowMs()
	result, err := d.db.Exec(`
		UPDATE agents
		SET is_completed = 1, completed_at = ?, updated_at = ?
		WHERE workspace_name = ?`,
		now, now, workspaceName,
	)
	if err != nil {
		return fmt.Errorf("failed to update completed for %s: %w", workspaceName, err)
	}
	return checkRowsAffected(result, workspaceName)
}

// UpdateCompletedByBeadsID marks an agent as completed by beads ID.
func (d *DB) UpdateCompletedByBeadsID(beadsID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := nowMs()
	result, err := d.db.Exec(`
		UPDATE agents
		SET is_completed = 1, completed_at = ?, updated_at = ?
		WHERE beads_id = ?`,
		now, now, beadsID,
	)
	if err != nil {
		return fmt.Errorf("failed to update completed for beads_id %s: %w", beadsID, err)
	}
	return checkRowsAffected(result, beadsID)
}

// UpdateAbandoned marks an agent as abandoned. Called by orch abandon.
func (d *DB) UpdateAbandoned(workspaceName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := nowMs()
	result, err := d.db.Exec(`
		UPDATE agents
		SET is_abandoned = 1, abandoned_at = ?, updated_at = ?
		WHERE workspace_name = ?`,
		now, now, workspaceName,
	)
	if err != nil {
		return fmt.Errorf("failed to update abandoned for %s: %w", workspaceName, err)
	}
	return checkRowsAffected(result, workspaceName)
}

// UpdateAbandonedByBeadsID marks an agent as abandoned by beads ID.
func (d *DB) UpdateAbandonedByBeadsID(beadsID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := nowMs()
	result, err := d.db.Exec(`
		UPDATE agents
		SET is_abandoned = 1, abandoned_at = ?, updated_at = ?
		WHERE beads_id = ?`,
		now, now, beadsID,
	)
	if err != nil {
		return fmt.Errorf("failed to update abandoned for beads_id %s: %w", beadsID, err)
	}
	return checkRowsAffected(result, beadsID)
}

// UpdateSessionID sets the session ID for an agent. Called after OpenCode session creation.
func (d *DB) UpdateSessionID(workspaceName, sessionID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := nowMs()
	result, err := d.db.Exec(`
		UPDATE agents
		SET session_id = ?, updated_at = ?
		WHERE workspace_name = ?`,
		sessionID, now, workspaceName,
	)
	if err != nil {
		return fmt.Errorf("failed to update session_id for %s: %w", workspaceName, err)
	}
	return checkRowsAffected(result, workspaceName)
}

// UpdateTmuxWindow sets the tmux window for an agent. Called after tmux window creation.
func (d *DB) UpdateTmuxWindow(workspaceName, tmuxWindow string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := nowMs()
	result, err := d.db.Exec(`
		UPDATE agents
		SET tmux_window = ?, updated_at = ?
		WHERE workspace_name = ?`,
		tmuxWindow, now, workspaceName,
	)
	if err != nil {
		return fmt.Errorf("failed to update tmux_window for %s: %w", workspaceName, err)
	}
	return checkRowsAffected(result, workspaceName)
}

// UpdatePhase updates the phase for an agent. Called by orch phase command.
func (d *DB) UpdatePhase(workspaceName, phase, summary string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := nowMs()
	result, err := d.db.Exec(`
		UPDATE agents
		SET phase = ?, phase_summary = ?, phase_reported_at = ?, updated_at = ?
		WHERE workspace_name = ?`,
		phase, nullString(summary), now, now, workspaceName,
	)
	if err != nil {
		return fmt.Errorf("failed to update phase for %s: %w", workspaceName, err)
	}
	return checkRowsAffected(result, workspaceName)
}

// UpdatePhaseByBeadsID updates the phase for an agent by beads ID.
func (d *DB) UpdatePhaseByBeadsID(beadsID, phase, summary string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := nowMs()
	result, err := d.db.Exec(`
		UPDATE agents
		SET phase = ?, phase_summary = ?, phase_reported_at = ?, updated_at = ?
		WHERE beads_id = ?`,
		phase, nullString(summary), now, now, beadsID,
	)
	if err != nil {
		return fmt.Errorf("failed to update phase for beads_id %s: %w", beadsID, err)
	}
	return checkRowsAffected(result, beadsID)
}

// GetAgent returns an agent by workspace name.
func (d *DB) GetAgent(workspaceName string) (*Agent, error) {
	row := d.db.QueryRow(`SELECT * FROM agents WHERE workspace_name = ?`, workspaceName)
	return scanAgent(row)
}

// GetAgentByBeadsID returns an agent by beads ID.
func (d *DB) GetAgentByBeadsID(beadsID string) (*Agent, error) {
	row := d.db.QueryRow(`SELECT * FROM agents WHERE beads_id = ?`, beadsID)
	return scanAgent(row)
}

// ListActiveAgents returns all agents that are not completed or abandoned.
func (d *DB) ListActiveAgents() ([]*Agent, error) {
	rows, err := d.db.Query(`
		SELECT * FROM agents
		WHERE is_completed = 0 AND is_abandoned = 0
		ORDER BY spawn_time DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query active agents: %w", err)
	}
	defer rows.Close()
	return scanAgents(rows)
}

// ListAgentsByProject returns all agents for a given project name.
func (d *DB) ListAgentsByProject(projectName string) ([]*Agent, error) {
	rows, err := d.db.Query(`
		SELECT * FROM agents
		WHERE project_name = ?
		ORDER BY spawn_time DESC`, projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents by project: %w", err)
	}
	defer rows.Close()
	return scanAgents(rows)
}

// ListAllAgents returns all agents.
func (d *DB) ListAllAgents() ([]*Agent, error) {
	rows, err := d.db.Query(`SELECT * FROM agents ORDER BY spawn_time DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query all agents: %w", err)
	}
	defer rows.Close()
	return scanAgents(rows)
}

// scanAgent scans a single agent row from a QueryRow result.
func scanAgent(row *sql.Row) (*Agent, error) {
	a := &Agent{}
	var beadsID, sessionID, tmuxWindow, skill, model, tier sql.NullString
	var projectName, gitBaseline, issueTitle, issueType sql.NullString
	var phase, phaseSummary sql.NullString
	var phaseReportedAt, sessionUpdatedAt, completedAt, abandonedAt sql.NullInt64
	var isProcessing, isCompleted, isAbandoned int

	err := row.Scan(
		&a.WorkspaceName, &beadsID, &sessionID, &tmuxWindow,
		&a.Mode, &skill, &model, &tier,
		&a.ProjectDir, &projectName, &a.SpawnTime, &gitBaseline,
		&issueTitle, &issueType, &a.IssuePriority,
		&phase, &phaseSummary, &phaseReportedAt,
		&isProcessing, &sessionUpdatedAt,
		&isCompleted, &isAbandoned, &completedAt, &abandonedAt,
		&a.TokensInput, &a.TokensOutput, &a.TokensReasoning,
		&a.TokensCacheRead, &a.TokensTotal,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("failed to scan agent: %w", err)
	}

	a.BeadsID = beadsID.String
	a.SessionID = sessionID.String
	a.TmuxWindow = tmuxWindow.String
	a.Skill = skill.String
	a.Model = model.String
	a.Tier = tier.String
	a.ProjectName = projectName.String
	a.GitBaseline = gitBaseline.String
	a.IssueTitle = issueTitle.String
	a.IssueType = issueType.String
	a.Phase = phase.String
	a.PhaseSummary = phaseSummary.String
	a.PhaseReportedAt = phaseReportedAt.Int64
	a.IsProcessing = isProcessing != 0
	a.SessionUpdatedAt = sessionUpdatedAt.Int64
	a.IsCompleted = isCompleted != 0
	a.IsAbandoned = isAbandoned != 0
	a.CompletedAt = completedAt.Int64
	a.AbandonedAt = abandonedAt.Int64

	return a, nil
}

// scanAgents scans multiple agent rows.
func scanAgents(rows *sql.Rows) ([]*Agent, error) {
	var agents []*Agent
	for rows.Next() {
		a := &Agent{}
		var beadsID, sessionID, tmuxWindow, skill, model, tier sql.NullString
		var projectName, gitBaseline, issueTitle, issueType sql.NullString
		var phase, phaseSummary sql.NullString
		var phaseReportedAt, sessionUpdatedAt, completedAt, abandonedAt sql.NullInt64
		var isProcessing, isCompleted, isAbandoned int

		err := rows.Scan(
			&a.WorkspaceName, &beadsID, &sessionID, &tmuxWindow,
			&a.Mode, &skill, &model, &tier,
			&a.ProjectDir, &projectName, &a.SpawnTime, &gitBaseline,
			&issueTitle, &issueType, &a.IssuePriority,
			&phase, &phaseSummary, &phaseReportedAt,
			&isProcessing, &sessionUpdatedAt,
			&isCompleted, &isAbandoned, &completedAt, &abandonedAt,
			&a.TokensInput, &a.TokensOutput, &a.TokensReasoning,
			&a.TokensCacheRead, &a.TokensTotal,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent row: %w", err)
		}

		a.BeadsID = beadsID.String
		a.SessionID = sessionID.String
		a.TmuxWindow = tmuxWindow.String
		a.Skill = skill.String
		a.Model = model.String
		a.Tier = tier.String
		a.ProjectName = projectName.String
		a.GitBaseline = gitBaseline.String
		a.IssueTitle = issueTitle.String
		a.IssueType = issueType.String
		a.Phase = phase.String
		a.PhaseSummary = phaseSummary.String
		a.PhaseReportedAt = phaseReportedAt.Int64
		a.IsProcessing = isProcessing != 0
		a.SessionUpdatedAt = sessionUpdatedAt.Int64
		a.IsCompleted = isCompleted != 0
		a.IsAbandoned = isAbandoned != 0
		a.CompletedAt = completedAt.Int64
		a.AbandonedAt = abandonedAt.Int64

		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// nullString converts an empty string to a sql.NullString for nullable columns.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullInt64 converts a zero int64 to a sql.NullInt64 for nullable columns.
func nullInt64(n int64) sql.NullInt64 {
	if n == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: n, Valid: true}
}

// checkRowsAffected returns an error if no rows were affected by an update.
func checkRowsAffected(result sql.Result, identifier string) error {
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("agent %s not found in state database", identifier)
	}
	return nil
}

// OpenDefault opens the state database at the default path.
// Returns nil, nil if the database path cannot be determined (graceful degradation).
func OpenDefault() (*DB, error) {
	path := DefaultDBPath()
	if path == "" {
		return nil, nil
	}
	return Open(path)
}
