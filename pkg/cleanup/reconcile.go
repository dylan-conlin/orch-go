package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/state"
)

// ReconcileStateOptions configures state/session reconciliation.
type ReconcileStateOptions struct {
	// ServerURL is the OpenCode server URL.
	ServerURL string
	// Client optionally injects a preconfigured OpenCode client.
	Client opencode.ClientInterface
	// DBPath overrides state DB path. Empty uses default (~/.orch/state.db).
	DBPath string
	// RegistryPath overrides sessions registry path. Empty uses default (~/.orch/sessions.json).
	RegistryPath string
	// DryRun reports changes without mutating state.
	DryRun bool
	// Quiet suppresses progress output.
	Quiet bool
	// ReconcileRegistry enables sessions.json active-status reconciliation.
	ReconcileRegistry bool
}

// ReconcileStateResult summarizes reconciliation outcomes.
type ReconcileStateResult struct {
	// State DB metrics
	ActiveRows          int
	ReconcilableRows    int
	LiveRows            int
	StaleRows           int
	CompletedRows       int
	AbandonedRows       int
	SkippedRows         int
	OpenMinusLiveBefore int
	OpenMinusLiveAfter  int

	// sessions.json metrics
	RegistryActive  int
	RegistryUpdated int
	RegistrySkipped int
}

// ReconcileState closes stale active rows in state.db by checking liveness against
// live OpenCode sessions (updated within state.DefaultMaxIdleTime).
//
// Policy:
//   - reconcilable rows (opencode/headless or rows with session_id) with no live
//     OpenCode session are marked closed:
//   - rows with completion signals are marked completed
//   - all other stale rows are marked abandoned
//   - non-OpenCode rows without session IDs are skipped (for example tmux-only modes)
func ReconcileState(opts ReconcileStateOptions) (*ReconcileStateResult, error) {
	result := &ReconcileStateResult{}
	client := opts.Client
	if client == nil {
		client = opencode.NewClient(opts.ServerURL)
	}

	liveSessions, err := listRecentlyActiveSessions(client, state.DefaultMaxIdleTime)
	if err != nil {
		return nil, fmt.Errorf("failed to list live OpenCode sessions: %w", err)
	}
	liveSessionIDs, liveTitles := indexLiveSessions(liveSessions)

	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = state.DefaultDBPath()
	}

	if dbPath != "" {
		if _, err := os.Stat(dbPath); err == nil {
			if err := reconcileStateDBRows(dbPath, liveSessionIDs, liveTitles, opts.DryRun, result); err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to stat state DB %s: %w", dbPath, err)
		}
	}

	if opts.ReconcileRegistry {
		registryPath := opts.RegistryPath
		if registryPath == "" {
			registryPath = session.RegistryPath()
		}
		if _, err := os.Stat(registryPath); err == nil {
			if err := reconcileRegistryRows(registryPath, liveSessionIDs, opts.DryRun, result); err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to stat sessions registry %s: %w", registryPath, err)
		}
	}

	if !opts.Quiet {
		verb := "Marked"
		if opts.DryRun {
			verb = "Would mark"
		}
		fmt.Printf("\nReconciling state cache against live OpenCode sessions...\n")
		fmt.Printf("  Active rows: %d (reconcilable: %d, skipped: %d)\n", result.ActiveRows, result.ReconcilableRows, result.SkippedRows)
		fmt.Printf("  Live matches: %d, stale rows: %d\n", result.LiveRows, result.StaleRows)
		fmt.Printf("  %s completed: %d, abandoned: %d\n", verb, result.CompletedRows, result.AbandonedRows)
		if opts.ReconcileRegistry {
			fmt.Printf("  sessions.json active: %d (updated: %d, skipped: %d)\n", result.RegistryActive, result.RegistryUpdated, result.RegistrySkipped)
		}
		fmt.Printf("  open_minus_live: %d -> %d\n", result.OpenMinusLiveBefore, result.OpenMinusLiveAfter)
	}

	return result, nil
}

func listRecentlyActiveSessions(client opencode.ClientInterface, maxIdle time.Duration) ([]opencode.Session, error) {
	sessions, err := client.ListSessions("")
	if err != nil {
		return nil, err
	}
	now := time.Now()
	live := make([]opencode.Session, 0, len(sessions))
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdle {
			live = append(live, s)
		}
	}
	return live, nil
}

func indexLiveSessions(sessions []opencode.Session) (map[string]bool, []string) {
	liveSessionIDs := make(map[string]bool, len(sessions))
	liveTitles := make([]string, 0, len(sessions))
	for _, s := range sessions {
		if s.ID != "" {
			liveSessionIDs[s.ID] = true
		}
		if s.Title != "" {
			liveTitles = append(liveTitles, s.Title)
		}
	}
	return liveSessionIDs, liveTitles
}

func reconcileStateDBRows(
	dbPath string,
	liveSessionIDs map[string]bool,
	liveTitles []string,
	dryRun bool,
	result *ReconcileStateResult,
) error {
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open state DB %s: %w", dbPath, err)
	}
	defer db.Close()

	activeRows, err := db.ListActiveAgents()
	if err != nil {
		return fmt.Errorf("failed to list active state rows: %w", err)
	}
	result.ActiveRows = len(activeRows)

	for _, row := range activeRows {
		if !isReconcilableStateRow(row) {
			result.SkippedRows++
			continue
		}

		result.ReconcilableRows++
		if isStateRowLive(row, liveSessionIDs, liveTitles) {
			result.LiveRows++
			continue
		}

		result.StaleRows++
		markCompleted := shouldMarkCompleted(row)

		if markCompleted {
			result.CompletedRows++
			if !dryRun {
				if err := db.UpdateCompleted(row.WorkspaceName); err != nil {
					return fmt.Errorf("failed to mark completed for %s: %w", row.WorkspaceName, err)
				}
			}
			continue
		}

		result.AbandonedRows++
		if !dryRun {
			if err := db.UpdateAbandoned(row.WorkspaceName); err != nil {
				return fmt.Errorf("failed to mark abandoned for %s: %w", row.WorkspaceName, err)
			}
		}
	}

	result.OpenMinusLiveBefore = result.ReconcilableRows - result.LiveRows

	if dryRun {
		result.OpenMinusLiveAfter = result.OpenMinusLiveBefore
		return nil
	}

	activeAfter, err := db.ListActiveAgents()
	if err != nil {
		return fmt.Errorf("failed to re-list active rows after reconciliation: %w", err)
	}
	result.OpenMinusLiveAfter = computeOpenMinusLive(activeAfter, liveSessionIDs, liveTitles)
	return nil
}

func isReconcilableStateRow(row *state.Agent) bool {
	mode := strings.ToLower(strings.TrimSpace(row.Mode))
	if row.SessionID != "" {
		return true
	}
	return mode == "" || mode == "opencode" || mode == "headless"
}

func isStateRowLive(row *state.Agent, liveSessionIDs map[string]bool, liveTitles []string) bool {
	if row.SessionID != "" && liveSessionIDs[row.SessionID] {
		return true
	}
	if row.BeadsID != "" {
		for _, title := range liveTitles {
			if strings.Contains(title, row.BeadsID) {
				return true
			}
		}
	}
	return false
}

func computeOpenMinusLive(activeRows []*state.Agent, liveSessionIDs map[string]bool, liveTitles []string) int {
	reconcilable := 0
	live := 0
	for _, row := range activeRows {
		if !isReconcilableStateRow(row) {
			continue
		}
		reconcilable++
		if isStateRowLive(row, liveSessionIDs, liveTitles) {
			live++
		}
	}
	return reconcilable - live
}

func shouldMarkCompleted(row *state.Agent) bool {
	phase := strings.ToLower(strings.TrimSpace(row.Phase))
	if strings.HasPrefix(phase, "complete") {
		return true
	}
	if row.ProjectDir == "" || row.WorkspaceName == "" {
		return false
	}
	workspacePath := filepath.Join(row.ProjectDir, ".orch", "workspace", row.WorkspaceName)
	if fileExists(filepath.Join(workspacePath, "SYNTHESIS.md")) {
		return true
	}
	archivedPath := filepath.Join(row.ProjectDir, ".orch", "workspace", "archived", row.WorkspaceName)
	if dirExists(archivedPath) {
		return true
	}
	return false
}

func reconcileRegistryRows(
	registryPath string,
	liveSessionIDs map[string]bool,
	dryRun bool,
	result *ReconcileStateResult,
) error {
	registry := session.NewRegistry(registryPath)
	active, err := registry.ListActive()
	if err != nil {
		return fmt.Errorf("failed to list active sessions from registry: %w", err)
	}

	result.RegistryActive = len(active)
	for _, item := range active {
		if strings.TrimSpace(item.SessionID) == "" {
			result.RegistrySkipped++
			continue
		}
		if liveSessionIDs[item.SessionID] {
			continue
		}

		newStatus := "abandoned"
		if item.ArchivedPath != "" || discoverArchivedWorkspacePath(item.ProjectDir, item.WorkspaceName) != "" {
			newStatus = "completed"
		}

		result.RegistryUpdated++
		if dryRun {
			continue
		}
		if err := registry.Update(item.WorkspaceName, func(s *session.OrchestratorSession) {
			s.Status = newStatus
			if s.ArchivedPath == "" {
				s.ArchivedPath = discoverArchivedWorkspacePath(s.ProjectDir, s.WorkspaceName)
			}
		}); err != nil {
			return fmt.Errorf("failed to update registry session %s: %w", item.WorkspaceName, err)
		}
	}

	return nil
}

func discoverArchivedWorkspacePath(projectDir, workspaceName string) string {
	if projectDir == "" || workspaceName == "" {
		return ""
	}
	path := filepath.Join(projectDir, ".orch", "workspace", "archived", workspaceName)
	if dirExists(path) {
		return path
	}
	return ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
