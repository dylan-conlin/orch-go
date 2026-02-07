package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// DecisionItem represents a single item requiring decision/action.
type DecisionItem struct {
	ID                string   `json:"id"`                           // Agent ID or BeadsID
	BeadsID           string   `json:"beads_id,omitempty"`           // BeadsID if available
	Title             string   `json:"title,omitempty"`              // Issue title or task description
	Skill             string   `json:"skill,omitempty"`              // Skill name if applicable
	Project           string   `json:"project,omitempty"`            // Project name
	EscalationLevel   string   `json:"escalation_level,omitempty"`   // Escalation level string
	EscalationReason  string   `json:"escalation_reason,omitempty"`  // Human-readable reason
	TLDR              string   `json:"tldr,omitempty"`               // Synthesis TLDR if available
	Recommendation    string   `json:"recommendation,omitempty"`     // Synthesis recommendation
	HasWebChanges     bool     `json:"has_web_changes,omitempty"`    // For visual verification
	NextActions       []string `json:"next_actions,omitempty"`       // Follow-up items
	WorkspacePath     string   `json:"workspace_path,omitempty"`     // Path to workspace
	InvestigationPath string   `json:"investigation_path,omitempty"` // Path to investigation file
	CompletedAt       string   `json:"completed_at,omitempty"`       // Completion timestamp
}

// DecisionsAPIResponse is the JSON structure returned by /api/decisions.
type DecisionsAPIResponse struct {
	// Action-oriented categories from the design
	AbsorbKnowledge []DecisionItem `json:"absorb_knowledge"` // Knowledge-producing completions (EscalationReview)
	GiveApprovals   []DecisionItem `json:"give_approvals"`   // Visual verification needed (EscalationBlock)
	AnswerQuestions []DecisionItem `json:"answer_questions"` // Strategic questions from questions store
	HandleFailures  []DecisionItem `json:"handle_failures"`  // Failed verifications (EscalationFailed)

	// Metadata
	TotalCount int    `json:"total_count"`
	ProjectDir string `json:"project_dir,omitempty"`
	Error      string `json:"error,omitempty"`
}

// handleDecisions aggregates agent escalation data for the Decision Center.
// Returns items grouped by action type based on escalation levels.
//
// Query params:
//   - project_dir: Optional project directory to query. If not provided, uses default.
func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get project_dir from query params (for following orchestrator context)
	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		projectDir, _ = s.currentProjectDir()
	}

	// Initialize response with empty slices
	resp := DecisionsAPIResponse{
		AbsorbKnowledge: []DecisionItem{},
		GiveApprovals:   []DecisionItem{},
		AnswerQuestions: []DecisionItem{},
		HandleFailures:  []DecisionItem{},
		ProjectDir:      projectDir,
	}

	// 1. Get completed agents from workspace directory
	workspacesDir := filepath.Join(projectDir, ".orch", "workspace")
	workspaces, err := os.ReadDir(workspacesDir)
	if err != nil {
		// If workspace dir doesn't exist, return empty response (not an error)
		resp.Error = fmt.Sprintf("No workspaces found: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 2. Scan workspaces for completed agents with escalation data
	for _, ws := range workspaces {
		if !ws.IsDir() {
			continue
		}

		workspacePath := filepath.Join(workspacesDir, ws.Name())

		// Check for completion marker (STATUS file with Complete status)
		statusPath := filepath.Join(workspacePath, "STATUS")
		statusData, err := os.ReadFile(statusPath)
		if err != nil {
			continue // Not completed yet
		}

		if !strings.Contains(string(statusData), "Complete") {
			continue // Not in Complete state
		}

		// Read MANIFEST.yaml for metadata
		manifest, err := readWorkspaceManifest(workspacePath)
		if err != nil {
			continue
		}

		// Skip if already approved/dismissed
		if manifest.Approved || manifest.Dismissed {
			continue
		}

		// Determine escalation level
		escalationLevel, escalationReason := determineWorkspaceEscalation(workspacePath, projectDir, manifest)

		// Create decision item
		item := DecisionItem{
			ID:               ws.Name(),
			BeadsID:          manifest.BeadsID,
			Title:            manifest.Task,
			Skill:            manifest.Skill,
			Project:          manifest.Project,
			EscalationLevel:  escalationLevel.String(),
			EscalationReason: escalationReason.Reason,
			WorkspacePath:    workspacePath,
			CompletedAt:      manifest.CompletedAt,
		}

		// Extract synthesis data if available
		synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
		if synthesis, err := verify.ParseSynthesis(synthesisPath); err == nil {
			item.TLDR = synthesis.TLDR
			item.Recommendation = synthesis.Recommendation
			item.NextActions = synthesis.NextActions
		}

		// Extract investigation path if available
		item.InvestigationPath = findInvestigationPath(workspacePath, projectDir, manifest.BeadsID)

		// Check for web changes
		item.HasWebChanges = hasWebChanges(workspacePath, projectDir)

		// Categorize by escalation level
		switch escalationLevel {
		case verify.EscalationReview:
			// Knowledge-producing skills → Absorb Knowledge
			if verify.IsKnowledgeProducingSkill(manifest.Skill) {
				resp.AbsorbKnowledge = append(resp.AbsorbKnowledge, item)
			}
		case verify.EscalationBlock:
			// Visual verification needed → Give Approvals
			resp.GiveApprovals = append(resp.GiveApprovals, item)
		case verify.EscalationFailed:
			// Failed verification → Handle Failures
			resp.HandleFailures = append(resp.HandleFailures, item)
		}
	}

	// 3. Add questions from questions store (strategic questions needing answers)
	questions := getOpenQuestions()
	for _, q := range questions {
		item := DecisionItem{
			ID:          q.ID,
			BeadsID:     q.ID,
			Title:       q.Title,
			Project:     projectDir,
			NextActions: q.Blocking, // Issues this question blocks
		}
		resp.AnswerQuestions = append(resp.AnswerQuestions, item)
	}

	// Calculate total
	resp.TotalCount = len(resp.AbsorbKnowledge) + len(resp.GiveApprovals) +
		len(resp.AnswerQuestions) + len(resp.HandleFailures)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode decisions: %v", err), http.StatusInternalServerError)
		return
	}
}

// workspaceManifest represents minimal manifest data we need
type workspaceManifest struct {
	BeadsID     string
	Task        string
	Skill       string
	Project     string
	CompletedAt string
	Approved    bool
	Dismissed   bool
}

// readWorkspaceManifest reads minimal manifest data from MANIFEST.yaml
func readWorkspaceManifest(workspacePath string) (*workspaceManifest, error) {
	manifestPath := filepath.Join(workspacePath, "MANIFEST.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	// Simple YAML parsing for the fields we need
	manifest := &workspaceManifest{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "beads_id:") {
			manifest.BeadsID = strings.TrimSpace(strings.TrimPrefix(line, "beads_id:"))
		} else if strings.HasPrefix(line, "task:") {
			manifest.Task = strings.TrimSpace(strings.TrimPrefix(line, "task:"))
		} else if strings.HasPrefix(line, "skill:") {
			manifest.Skill = strings.TrimSpace(strings.TrimPrefix(line, "skill:"))
		} else if strings.HasPrefix(line, "project:") {
			manifest.Project = strings.TrimSpace(strings.TrimPrefix(line, "project:"))
		} else if strings.HasPrefix(line, "completed_at:") {
			manifest.CompletedAt = strings.TrimSpace(strings.TrimPrefix(line, "completed_at:"))
		} else if strings.HasPrefix(line, "approved:") {
			manifest.Approved = strings.Contains(strings.ToLower(line), "true")
		} else if strings.HasPrefix(line, "dismissed:") {
			manifest.Dismissed = strings.Contains(strings.ToLower(line), "true")
		}
	}

	return manifest, nil
}

// determineWorkspaceEscalation determines the escalation level for a completed workspace
func determineWorkspaceEscalation(workspacePath, projectDir string, manifest *workspaceManifest) (verify.EscalationLevel, verify.EscalationReason) {
	// Read verification result if available
	verificationPath := filepath.Join(workspacePath, "VERIFICATION.md")
	verificationPassed := true // Default to passed if no verification file
	var verificationErrors []string

	if data, err := os.ReadFile(verificationPath); err == nil {
		content := string(data)
		if strings.Contains(content, "❌") || strings.Contains(content, "FAILED") {
			verificationPassed = false
			// Extract error lines
			for _, line := range strings.Split(content, "\n") {
				if strings.Contains(line, "❌") || strings.Contains(line, "ERROR") {
					verificationErrors = append(verificationErrors, line)
				}
			}
		}
	}

	// Build escalation input
	input := verify.EscalationInput{
		VerificationPassed: verificationPassed,
		VerificationErrors: verificationErrors,
		SkillName:          manifest.Skill,
		WorkspacePath:      workspacePath,
		ProjectDir:         projectDir,
	}

	// Extract synthesis data if available
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	if synthesis, err := verify.ParseSynthesis(synthesisPath); err == nil {
		input.Outcome = synthesis.Outcome
		input.Recommendation = synthesis.Recommendation
		input.NextActions = synthesis.NextActions
	}

	// Check for web changes
	input.HasWebChanges = hasWebChanges(workspacePath, projectDir)
	input.NeedsVisualApproval = input.HasWebChanges // Simple heuristic for now

	level := verify.DetermineEscalation(input)
	reason := verify.ExplainEscalation(input)

	return level, reason
}

// hasWebChanges checks if workspace has web/ file changes
func hasWebChanges(workspacePath, projectDir string) bool {
	// Check for web/ prefix in git diff
	// This is a simplified version - in production might want to parse git log from workspace
	gitLogPath := filepath.Join(workspacePath, "git.log")
	if data, err := os.ReadFile(gitLogPath); err == nil {
		return strings.Contains(string(data), "web/")
	}
	return false
}

// findInvestigationPath attempts to find investigation file path for the workspace
func findInvestigationPath(workspacePath, projectDir, beadsID string) string {
	// Check for investigation path in beads comments (would need beads API integration)
	// For now, try to find based on beadsID pattern in .kb/investigations/
	if beadsID == "" {
		return ""
	}

	investigationsDir := filepath.Join(projectDir, ".kb", "investigations")
	entries, err := os.ReadDir(investigationsDir)
	if err != nil {
		return ""
	}

	// Look for files that reference the beadsID
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Quick scan for beadsID in filename or content
		path := filepath.Join(investigationsDir, entry.Name())
		if data, err := os.ReadFile(path); err == nil {
			if strings.Contains(string(data), beadsID) {
				return path
			}
		}
	}

	return ""
}

// getOpenQuestions fetches open questions from the beads store.
// Returns QuestionResponse items that are status=open (need answering).
func getOpenQuestions() []QuestionResponse {
	cliClient := beads.NewCLIClient()

	allQuestions, err := cliClient.List(&beads.ListArgs{
		IssueType: "question",
		Status:    "open",
		Limit:     50,
	})
	if err != nil {
		return []QuestionResponse{}
	}

	questions := make([]QuestionResponse, 0, len(allQuestions))
	for _, q := range allQuestions {
		qr := QuestionResponse{
			ID:        q.ID,
			Title:     q.Title,
			Status:    q.Status,
			Priority:  q.Priority,
			Labels:    q.Labels,
			CreatedAt: q.CreatedAt,
		}

		// Get blocking info from full issue
		if fullIssue, err := cliClient.Show(q.ID); err == nil && fullIssue.Dependencies != nil {
			var dependents []struct {
				ID string `json:"id"`
			}
			json.Unmarshal(fullIssue.Dependencies, &dependents)
			for _, dep := range dependents {
				qr.Blocking = append(qr.Blocking, dep.ID)
			}
		}

		questions = append(questions, qr)
	}

	return questions
}
