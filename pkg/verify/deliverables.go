// Package verify provides deliverables schema and tracking for issue completion.
// This implements Work Graph Phase 2: Deliverables schema and tracking.
//
// Deliverables are expected outputs per issue type + skill combination.
// The schema defines what's expected, detection checks if they're met,
// and override logging captures when completions happen with gaps.
package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DeliverableType represents a type of deliverable.
type DeliverableType string

const (
	// DeliverableCodeCommitted indicates code changes were committed.
	DeliverableCodeCommitted DeliverableType = "code_committed"
	// DeliverableTestsPass indicates tests are passing.
	DeliverableTestsPass DeliverableType = "tests_pass"
	// DeliverableSynthesisExists indicates SYNTHESIS.md exists.
	DeliverableSynthesisExists DeliverableType = "synthesis_exists"
	// DeliverableVisualVerified indicates visual verification was performed (for UI work).
	DeliverableVisualVerified DeliverableType = "visual_verified"
	// DeliverableInvestigationArtifact indicates an investigation artifact was created.
	DeliverableInvestigationArtifact DeliverableType = "investigation_artifact"
	// DeliverableDesignDocument indicates a design document was created.
	DeliverableDesignDocument DeliverableType = "design_document"
	// DeliverableDecisionRecord indicates a decision record was created.
	DeliverableDecisionRecord DeliverableType = "decision_record"
)

// DeliverableStatus represents the status of a single deliverable.
type DeliverableStatus struct {
	Type        DeliverableType `json:"type"`
	Expected    bool            `json:"expected"`    // Is this deliverable expected?
	Satisfied   bool            `json:"satisfied"`   // Is this deliverable satisfied?
	Description string          `json:"description"` // Human-readable description
	Evidence    string          `json:"evidence"`    // Evidence of satisfaction (e.g., commit SHA, test output)
	Optional    bool            `json:"optional"`    // Is this deliverable optional?
}

// DeliverablesResult contains the full deliverables status for an issue.
type DeliverablesResult struct {
	IssueID      string              `json:"issue_id"`
	IssueType    string              `json:"issue_type"`
	Skill        string              `json:"skill"`
	Deliverables []DeliverableStatus `json:"deliverables"`
	AllSatisfied bool                `json:"all_satisfied"`
	Required     int                 `json:"required"`  // Count of required deliverables
	Satisfied    int                 `json:"satisfied"` // Count of satisfied deliverables
	Missing      []DeliverableType   `json:"missing"`   // List of missing required deliverables
}

// DeliverableConfig defines expected deliverables for a type+skill combination.
type DeliverableConfig struct {
	Required []DeliverableType `yaml:"required"`
	Optional []DeliverableType `yaml:"optional"`
}

// DeliverablesSchema holds the full configuration for all type+skill combinations.
type DeliverablesSchema struct {
	// Defaults are used when no specific type+skill config exists.
	Defaults DeliverableConfig `yaml:"defaults"`
	// ByType maps issue types to skill-specific configs.
	// Format: "type.skill" -> config (e.g., "bug.feature-impl")
	// If skill is "*", applies to all skills for that type.
	ByTypeSkill map[string]DeliverableConfig `yaml:"by_type_skill"`
}

// DefaultDeliverablesSchema returns the built-in default schema.
// This is used when no user config exists.
func DefaultDeliverablesSchema() *DeliverablesSchema {
	return &DeliverablesSchema{
		Defaults: DeliverableConfig{
			Required: []DeliverableType{DeliverableCodeCommitted},
			Optional: []DeliverableType{DeliverableSynthesisExists},
		},
		ByTypeSkill: map[string]DeliverableConfig{
			// Bug + feature-impl: full verification
			"bug.feature-impl": {
				Required: []DeliverableType{
					DeliverableCodeCommitted,
					DeliverableTestsPass,
				},
				Optional: []DeliverableType{
					DeliverableVisualVerified,
					DeliverableSynthesisExists,
				},
			},
			// Task + feature-impl: code and tests
			"task.feature-impl": {
				Required: []DeliverableType{
					DeliverableCodeCommitted,
					DeliverableTestsPass,
				},
				Optional: []DeliverableType{
					DeliverableSynthesisExists,
				},
			},
			// Investigation skill: investigation artifact
			"*.investigation": {
				Required: []DeliverableType{
					DeliverableInvestigationArtifact,
				},
				Optional: []DeliverableType{},
			},
			// Architect skill: investigation + decision
			"*.architect": {
				Required: []DeliverableType{
					DeliverableInvestigationArtifact,
				},
				Optional: []DeliverableType{
					DeliverableDecisionRecord,
				},
			},
			// Design session: design document
			"*.design-session": {
				Required: []DeliverableType{
					DeliverableDesignDocument,
				},
				Optional: []DeliverableType{},
			},
			// Systematic debugging: code committed if fix found
			"*.systematic-debugging": {
				Required: []DeliverableType{
					DeliverableInvestigationArtifact,
				},
				Optional: []DeliverableType{
					DeliverableCodeCommitted,
					DeliverableTestsPass,
				},
			},
		},
	}
}

// LoadDeliverablesSchema loads the schema from ~/.orch/deliverables.yaml.
// Falls back to default schema if file doesn't exist.
func LoadDeliverablesSchema() (*DeliverablesSchema, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultDeliverablesSchema(), nil
	}

	configPath := filepath.Join(home, ".orch", "deliverables.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultDeliverablesSchema(), nil
		}
		return nil, fmt.Errorf("failed to read deliverables config: %w", err)
	}

	var schema DeliverablesSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse deliverables config: %w", err)
	}

	return &schema, nil
}

// GetConfigForIssue returns the deliverable config for a specific issue type and skill.
// It checks in order:
// 1. Exact match: "type.skill"
// 2. Wildcard type: "*.skill"
// 3. Wildcard skill: "type.*"
// 4. Defaults
func (s *DeliverablesSchema) GetConfigForIssue(issueType, skill string) DeliverableConfig {
	// Normalize inputs
	issueType = strings.ToLower(strings.TrimSpace(issueType))
	skill = strings.ToLower(strings.TrimSpace(skill))

	// 1. Exact match
	if cfg, ok := s.ByTypeSkill[issueType+"."+skill]; ok {
		return cfg
	}

	// 2. Wildcard type
	if cfg, ok := s.ByTypeSkill["*."+skill]; ok {
		return cfg
	}

	// 3. Wildcard skill
	if cfg, ok := s.ByTypeSkill[issueType+".*"]; ok {
		return cfg
	}

	// 4. Defaults
	return s.Defaults
}

// CheckDeliverables checks all deliverables for an issue.
// Parameters:
//   - issueID: the beads issue ID
//   - issueType: the issue type (bug, task, epic, question)
//   - skill: the skill used for the work
//   - workspacePath: path to the agent's workspace
//   - projectDir: path to the project root
//   - beadsComments: pre-fetched beads comments (for test/visual evidence)
func CheckDeliverables(issueID, issueType, skill, workspacePath, projectDir string, beadsComments []Comment) (*DeliverablesResult, error) {
	schema, err := LoadDeliverablesSchema()
	if err != nil {
		return nil, err
	}

	config := schema.GetConfigForIssue(issueType, skill)
	result := &DeliverablesResult{
		IssueID:      issueID,
		IssueType:    issueType,
		Skill:        skill,
		Deliverables: []DeliverableStatus{},
		Missing:      []DeliverableType{},
	}

	// Check required deliverables
	for _, dtype := range config.Required {
		status := checkDeliverable(dtype, workspacePath, projectDir, beadsComments)
		status.Expected = true
		status.Optional = false
		result.Deliverables = append(result.Deliverables, status)
		result.Required++
		if status.Satisfied {
			result.Satisfied++
		} else {
			result.Missing = append(result.Missing, dtype)
		}
	}

	// Check optional deliverables
	for _, dtype := range config.Optional {
		status := checkDeliverable(dtype, workspacePath, projectDir, beadsComments)
		status.Expected = true
		status.Optional = true
		result.Deliverables = append(result.Deliverables, status)
		// Optional deliverables don't affect AllSatisfied
	}

	result.AllSatisfied = len(result.Missing) == 0
	return result, nil
}

// checkDeliverable checks a single deliverable type.
func checkDeliverable(dtype DeliverableType, workspacePath, projectDir string, comments []Comment) DeliverableStatus {
	status := DeliverableStatus{
		Type: dtype,
	}

	switch dtype {
	case DeliverableCodeCommitted:
		status.Description = "Code changes committed to git"
		status.Satisfied, status.Evidence = detectCodeCommitted(projectDir)

	case DeliverableTestsPass:
		status.Description = "Tests are passing"
		status.Satisfied, status.Evidence = detectTestsPass(comments)

	case DeliverableSynthesisExists:
		status.Description = "SYNTHESIS.md exists in workspace"
		status.Satisfied, status.Evidence = detectSynthesisExists(workspacePath)

	case DeliverableVisualVerified:
		status.Description = "Visual verification performed for UI changes"
		status.Satisfied, status.Evidence = detectVisualVerified(comments)

	case DeliverableInvestigationArtifact:
		status.Description = "Investigation artifact created"
		status.Satisfied, status.Evidence = detectInvestigationArtifact(workspacePath, projectDir)

	case DeliverableDesignDocument:
		status.Description = "Design document created"
		status.Satisfied, status.Evidence = detectDesignDocument(workspacePath, projectDir)

	case DeliverableDecisionRecord:
		status.Description = "Decision record created"
		status.Satisfied, status.Evidence = detectDecisionRecord(projectDir)

	default:
		status.Description = fmt.Sprintf("Unknown deliverable type: %s", dtype)
		status.Satisfied = false
	}

	return status
}

// detectCodeCommitted checks if there are git commits since the workspace was created.
// Returns true if there are uncommitted changes or recent commits.
func detectCodeCommitted(projectDir string) (bool, string) {
	if projectDir == "" {
		return false, "no project directory"
	}

	// Check for staged or unstaged changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Sprintf("git status failed: %v", err)
	}

	if len(output) > 0 {
		return true, "uncommitted changes present"
	}

	// Check for recent commits (within last hour)
	// This uses git log to find commits since a reasonable time ago
	cmd = exec.Command("git", "log", "--oneline", "--since=1 hour ago", "-1")
	cmd.Dir = projectDir
	output, err = cmd.Output()
	if err != nil {
		return false, fmt.Sprintf("git log failed: %v", err)
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		commitLine := strings.TrimSpace(string(output))
		// Extract just the commit hash (first word)
		parts := strings.SplitN(commitLine, " ", 2)
		if len(parts) > 0 {
			return true, fmt.Sprintf("recent commit: %s", parts[0])
		}
		return true, "recent commit found"
	}

	return false, "no recent commits"
}

// detectTestsPass checks if test execution evidence exists in beads comments.
func detectTestsPass(comments []Comment) (bool, string) {
	hasEvidence, _ := HasTestExecutionEvidence(comments)
	if hasEvidence {
		return true, "test evidence found in beads comments"
	}
	return false, "no test execution evidence"
}

// detectSynthesisExists checks if SYNTHESIS.md exists and has content.
func detectSynthesisExists(workspacePath string) (bool, string) {
	if workspacePath == "" {
		return false, "no workspace path"
	}

	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	info, err := os.Stat(synthesisPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "SYNTHESIS.md not found"
		}
		return false, fmt.Sprintf("error checking SYNTHESIS.md: %v", err)
	}

	if info.Size() == 0 {
		return false, "SYNTHESIS.md is empty"
	}

	return true, fmt.Sprintf("SYNTHESIS.md exists (%d bytes)", info.Size())
}

// detectVisualVerified checks if visual verification evidence exists in beads comments.
func detectVisualVerified(comments []Comment) (bool, string) {
	hasVisual, _ := HasVisualVerificationEvidence(comments)
	if hasVisual {
		return true, "visual verification evidence found"
	}
	return false, "no visual verification evidence"
}

// detectInvestigationArtifact checks if an investigation file was created.
func detectInvestigationArtifact(workspacePath, projectDir string) (bool, string) {
	// Check workspace for INVESTIGATION.md or similar
	if workspacePath != "" {
		patterns := []string{
			"INVESTIGATION.md",
			"investigation.md",
		}
		for _, pattern := range patterns {
			path := filepath.Join(workspacePath, pattern)
			if _, err := os.Stat(path); err == nil {
				return true, fmt.Sprintf("found %s in workspace", pattern)
			}
		}
	}

	// Check .kb/investigations for recently modified files
	if projectDir != "" {
		investigationsDir := filepath.Join(projectDir, ".kb", "investigations")
		if entries, err := os.ReadDir(investigationsDir); err == nil {
			// Look for files modified in the last hour
			cutoff := time.Now().Add(-1 * time.Hour)
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					info, err := entry.Info()
					if err == nil && info.ModTime().After(cutoff) {
						return true, fmt.Sprintf("recent investigation: %s", entry.Name())
					}
				}
			}
		}
	}

	return false, "no investigation artifact found"
}

// detectDesignDocument checks if a design document was created.
func detectDesignDocument(workspacePath, projectDir string) (bool, string) {
	// Check workspace for DESIGN.md or similar
	if workspacePath != "" {
		patterns := []string{
			"DESIGN.md",
			"design.md",
			"DESIGN_BRIEF.md",
		}
		for _, pattern := range patterns {
			path := filepath.Join(workspacePath, pattern)
			if _, err := os.Stat(path); err == nil {
				return true, fmt.Sprintf("found %s in workspace", pattern)
			}
		}
	}

	// Check docs/designs for recently modified files
	if projectDir != "" {
		designsDir := filepath.Join(projectDir, "docs", "designs")
		if entries, err := os.ReadDir(designsDir); err == nil {
			cutoff := time.Now().Add(-1 * time.Hour)
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					info, err := entry.Info()
					if err == nil && info.ModTime().After(cutoff) {
						return true, fmt.Sprintf("recent design: %s", entry.Name())
					}
				}
			}
		}
	}

	return false, "no design document found"
}

// detectDecisionRecord checks if a decision record was created.
func detectDecisionRecord(projectDir string) (bool, string) {
	if projectDir == "" {
		return false, "no project directory"
	}

	// Check .kb/decisions for recently modified files
	decisionsDir := filepath.Join(projectDir, ".kb", "decisions")
	if entries, err := os.ReadDir(decisionsDir); err == nil {
		cutoff := time.Now().Add(-1 * time.Hour)
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				info, err := entry.Info()
				if err == nil && info.ModTime().After(cutoff) {
					return true, fmt.Sprintf("recent decision: %s", entry.Name())
				}
			}
		}
	}

	return false, "no decision record found"
}
