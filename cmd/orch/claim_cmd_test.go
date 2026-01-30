package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestSanitizeForWorkspaceName(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Add New Feature", 20, "add-new-feature"},
		{"Multiple   Spaces", 20, "multiple-spaces"},
		{"Special!@#$Chars", 20, "specialchars"},
		{"Underscores_Here", 20, "underscores-here"},
		{"This is a very long title that needs truncation", 20, "this-is-a-very-long"},
		{"Trailing-Hyphen-", 20, "trailing-hyphen"},
		{"--Leading-Hyphens", 20, "leading-hyphens"},
	}

	for _, tt := range tests {
		got := sanitizeForWorkspaceName(tt.input, tt.maxLen)
		if got != tt.expected {
			t.Errorf("sanitizeForWorkspaceName(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
		}
	}
}

func TestFormatSessionTitleWithBeadsID(t *testing.T) {
	tests := []struct {
		title    string
		beadsID  string
		expected string
	}{
		{"Add feature", "orch-go-123", "Add feature [orch-go-123]"},
		{"Fix bug [old-id]", "orch-go-456", "Fix bug [orch-go-456]"},
		{"Title without ID", "proj-789", "Title without ID [proj-789]"},
	}

	for _, tt := range tests {
		got := formatSessionTitleWithBeadsID(tt.title, tt.beadsID)
		if got != tt.expected {
			t.Errorf("formatSessionTitleWithBeadsID(%q, %q) = %q, want %q", tt.title, tt.beadsID, got, tt.expected)
		}
	}
}

func TestGenerateClaimWorkspaceName(t *testing.T) {
	// This test just verifies the function doesn't panic and produces reasonable output
	projectDir := "/Users/test/orch-go"
	sessionTitle := "Add new dashboard feature"
	beadsID := "orch-go-21029"

	result, err := generateClaimWorkspaceName(projectDir, sessionTitle, beadsID)
	if err != nil {
		t.Fatalf("generateClaimWorkspaceName failed: %v", err)
	}

	// Verify format: <project>-claimed-<description>-<date>-<hash>
	if len(result) == 0 {
		t.Error("workspace name is empty")
	}

	// Should contain "claimed"
	if !strings.Contains(result, "claimed") {
		t.Errorf("workspace name %q doesn't contain 'claimed'", result)
	}

	// Should contain hash from beads ID
	if !strings.Contains(result, "21029") {
		t.Errorf("workspace name %q doesn't contain hash '21029'", result)
	}
}

func TestWriteClaimWorkspaceFiles(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "claim-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	workspaceName := "test-claimed-feature-29jan-123"
	sessionID := "ses_test123"
	beadsID := "orch-go-test"
	projectDir := tmpDir
	sessionTitle := "Test Session Title"

	err = writeClaimWorkspaceFiles(tmpDir, workspaceName, sessionID, beadsID, projectDir, sessionTitle)
	if err != nil {
		t.Fatalf("writeClaimWorkspaceFiles failed: %v", err)
	}

	// Verify .session_id
	readSessionID := spawn.ReadSessionID(tmpDir)
	if readSessionID != sessionID {
		t.Errorf("session ID = %q, want %q", readSessionID, sessionID)
	}

	// Verify .beads_id
	beadsIDPath := filepath.Join(tmpDir, ".beads_id")
	content, err := os.ReadFile(beadsIDPath)
	if err != nil {
		t.Fatalf("failed to read .beads_id: %v", err)
	}
	readBeadsID := string(content)
	if readBeadsID != beadsID+"\n" {
		t.Errorf("beads ID = %q, want %q", readBeadsID, beadsID+"\n")
	}

	// Verify .tier
	tier := spawn.ReadTier(tmpDir)
	if tier != spawn.TierLight {
		t.Errorf("tier = %q, want %q", tier, spawn.TierLight)
	}

	// Verify AGENT_MANIFEST.json exists
	manifestPath := filepath.Join(tmpDir, "AGENT_MANIFEST.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("AGENT_MANIFEST.json doesn't exist")
	}

	// Verify SPAWN_CONTEXT.md exists
	spawnContextPath := filepath.Join(tmpDir, "SPAWN_CONTEXT.md")
	if _, err := os.Stat(spawnContextPath); os.IsNotExist(err) {
		t.Error("SPAWN_CONTEXT.md doesn't exist")
	}
}

func TestExtractHashFromBeadsID(t *testing.T) {
	tests := []struct {
		beadsID  string
		expected string
	}{
		{"orch-go-21029", "21029"},
		{"proj-456", "456"},
		{"single", "single"},
		{"", "unkn"},
	}

	for _, tt := range tests {
		got := extractHashFromBeadsID(tt.beadsID)
		if got != tt.expected {
			t.Errorf("extractHashFromBeadsID(%q) = %q, want %q", tt.beadsID, got, tt.expected)
		}
	}
}
