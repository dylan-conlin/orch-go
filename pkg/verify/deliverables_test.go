package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDeliverablesSchema(t *testing.T) {
	schema := DefaultDeliverablesSchema()

	// Verify defaults exist
	if len(schema.Defaults.Required) == 0 {
		t.Error("defaults should have required deliverables")
	}

	// Verify bug.feature-impl config
	cfg := schema.GetConfigForIssue("bug", "feature-impl")
	if len(cfg.Required) == 0 {
		t.Error("bug.feature-impl should have required deliverables")
	}

	// Check that code_committed and tests_pass are required for bug.feature-impl
	hasCodeCommitted := false
	hasTestsPass := false
	for _, d := range cfg.Required {
		if d == DeliverableCodeCommitted {
			hasCodeCommitted = true
		}
		if d == DeliverableTestsPass {
			hasTestsPass = true
		}
	}
	if !hasCodeCommitted {
		t.Error("bug.feature-impl should require code_committed")
	}
	if !hasTestsPass {
		t.Error("bug.feature-impl should require tests_pass")
	}
}

func TestGetConfigForIssue(t *testing.T) {
	schema := DefaultDeliverablesSchema()

	tests := []struct {
		name           string
		issueType      string
		skill          string
		expectRequired []DeliverableType
	}{
		{
			name:           "exact match bug.feature-impl",
			issueType:      "bug",
			skill:          "feature-impl",
			expectRequired: []DeliverableType{DeliverableCodeCommitted, DeliverableTestsPass},
		},
		{
			name:           "wildcard type *.investigation",
			issueType:      "task",
			skill:          "investigation",
			expectRequired: []DeliverableType{DeliverableInvestigationArtifact},
		},
		{
			name:           "wildcard type *.architect",
			issueType:      "bug",
			skill:          "architect",
			expectRequired: []DeliverableType{DeliverableInvestigationArtifact},
		},
		{
			name:           "fallback to defaults",
			issueType:      "unknown",
			skill:          "unknown-skill",
			expectRequired: []DeliverableType{DeliverableCodeCommitted},
		},
		{
			name:           "case insensitive",
			issueType:      "BUG",
			skill:          "Feature-Impl",
			expectRequired: []DeliverableType{DeliverableCodeCommitted, DeliverableTestsPass},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := schema.GetConfigForIssue(tt.issueType, tt.skill)
			if len(cfg.Required) != len(tt.expectRequired) {
				t.Errorf("expected %d required deliverables, got %d", len(tt.expectRequired), len(cfg.Required))
				return
			}
			for i, expected := range tt.expectRequired {
				if cfg.Required[i] != expected {
					t.Errorf("expected required[%d] = %s, got %s", i, expected, cfg.Required[i])
				}
			}
		})
	}
}

func TestDetectSynthesisExists(t *testing.T) {
	// Create temp dir for testing
	tmpDir := t.TempDir()

	// Test non-existent file
	satisfied, evidence := detectSynthesisExists(tmpDir)
	if satisfied {
		t.Error("should not be satisfied when SYNTHESIS.md doesn't exist")
	}
	if evidence != "SYNTHESIS.md not found" {
		t.Errorf("unexpected evidence: %s", evidence)
	}

	// Create empty file
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")
	if err := os.WriteFile(synthesisPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	satisfied, evidence = detectSynthesisExists(tmpDir)
	if satisfied {
		t.Error("should not be satisfied when SYNTHESIS.md is empty")
	}
	if evidence != "SYNTHESIS.md is empty" {
		t.Errorf("unexpected evidence: %s", evidence)
	}

	// Write content
	if err := os.WriteFile(synthesisPath, []byte("# Session Summary"), 0644); err != nil {
		t.Fatal(err)
	}

	satisfied, evidence = detectSynthesisExists(tmpDir)
	if !satisfied {
		t.Error("should be satisfied when SYNTHESIS.md has content")
	}
}

func TestDetectTestsPass(t *testing.T) {
	tests := []struct {
		name     string
		comments []Comment
		want     bool
	}{
		{
			name:     "no comments",
			comments: []Comment{},
			want:     false,
		},
		{
			name: "with test evidence",
			comments: []Comment{
				{Text: "Tests: go test ./... - PASS (15 passed, 0 failed)"},
			},
			want: true,
		},
		{
			name: "without test evidence",
			comments: []Comment{
				{Text: "Phase: Implementing - working on feature"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			satisfied, _ := detectTestsPass(tt.comments)
			if satisfied != tt.want {
				t.Errorf("detectTestsPass() = %v, want %v", satisfied, tt.want)
			}
		})
	}
}

func TestDetectVisualVerified(t *testing.T) {
	tests := []struct {
		name     string
		comments []Comment
		want     bool
	}{
		{
			name:     "no comments",
			comments: []Comment{},
			want:     false,
		},
		{
			name: "with visual verification",
			comments: []Comment{
				{Text: "Visual verification: screenshot shows correct layout"},
			},
			want: true,
		},
		{
			name: "without visual verification",
			comments: []Comment{
				{Text: "Phase: Complete - all tests passing"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			satisfied, _ := detectVisualVerified(tt.comments)
			if satisfied != tt.want {
				t.Errorf("detectVisualVerified() = %v, want %v", satisfied, tt.want)
			}
		})
	}
}

func TestCheckDeliverables(t *testing.T) {
	// Create temp workspace
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Test with no deliverables satisfied
	result, err := CheckDeliverables("test-123", "bug", "feature-impl", workspacePath, tmpDir, nil)
	if err != nil {
		t.Fatalf("CheckDeliverables failed: %v", err)
	}

	if result.IssueID != "test-123" {
		t.Errorf("expected issue_id 'test-123', got '%s'", result.IssueID)
	}

	if result.AllSatisfied {
		t.Error("should not be all satisfied when no deliverables are met")
	}

	if result.Required == 0 {
		t.Error("should have required deliverables for bug.feature-impl")
	}

	// Test with investigation skill (different deliverables)
	result, err = CheckDeliverables("inv-456", "task", "investigation", workspacePath, tmpDir, nil)
	if err != nil {
		t.Fatalf("CheckDeliverables failed: %v", err)
	}

	hasInvestigation := false
	for _, d := range result.Deliverables {
		if d.Type == DeliverableInvestigationArtifact {
			hasInvestigation = true
			break
		}
	}
	if !hasInvestigation {
		t.Error("investigation skill should expect investigation_artifact deliverable")
	}
}
