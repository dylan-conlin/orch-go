package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiscoverInvestigationPath tests auto-discovery of investigation files.
// NOTE: Keyword-based matching was REMOVED (Jan 2026) because it caused wrong
// investigations to be shown. Now we only match by beads ID or workspace directory files.
func TestDiscoverInvestigationPath(t *testing.T) {
	// Create a temporary project directory structure
	tmpDir := t.TempDir()

	// Create .kb/investigations/ directory with some investigation files
	invDir := filepath.Join(tmpDir, ".kb", "investigations")
	if err := os.MkdirAll(invDir, 0755); err != nil {
		t.Fatalf("Failed to create investigations dir: %v", err)
	}

	// Create investigation files - some with beads IDs in the name
	invFiles := []string{
		"2026-01-06-inv-dashboard-auto-discover.md",      // No beads ID
		"2026-01-05-inv-status-polling.md",               // No beads ID
		"2026-01-04-inv-skillc-deploy-structure.md",      // No beads ID
		"2026-01-07-inv-orch-go-abc123-specific-task.md", // Has beads ID "abc123"
		"2026-01-08-inv-feature-orch-go-def456.md",       // Has beads ID "def456"
	}
	for _, name := range invFiles {
		if err := os.WriteFile(filepath.Join(invDir, name), []byte("# Investigation"), 0644); err != nil {
			t.Fatalf("Failed to create investigation file: %v", err)
		}
	}

	// Create workspace directory with .md files
	wsDir := filepath.Join(tmpDir, ".orch", "workspace", "og-inv-my-workspace-06jan-1234")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}
	// Create standard workspace files that should be skipped
	if err := os.WriteFile(filepath.Join(wsDir, "SPAWN_CONTEXT.md"), []byte("# Context"), 0644); err != nil {
		t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "SYNTHESIS.md"), []byte("# Synthesis"), 0644); err != nil {
		t.Fatalf("Failed to create SYNTHESIS.md: %v", err)
	}
	// Create an investigation file in workspace
	if err := os.WriteFile(filepath.Join(wsDir, "inv-local-findings.md"), []byte("# Findings"), 0644); err != nil {
		t.Fatalf("Failed to create local investigation: %v", err)
	}

	tests := []struct {
		name          string
		workspaceName string
		beadsID       string
		projectDir    string
		wantFound     bool
		wantContains  string // substring that should be in the result
	}{
		{
			name:          "match_by_beads_id_full",
			workspaceName: "og-inv-some-task-07jan-xxxx",
			beadsID:       "orch-go-abc123",
			projectDir:    tmpDir,
			wantFound:     true,
			wantContains:  "abc123",
		},
		{
			name:          "match_by_beads_id_short",
			workspaceName: "og-inv-another-task-08jan-yyyy",
			beadsID:       "orch-go-def456",
			projectDir:    tmpDir,
			wantFound:     true,
			wantContains:  "def456",
		},
		{
			name:          "no_project_dir",
			workspaceName: "og-inv-test",
			beadsID:       "test-123",
			projectDir:    "",
			wantFound:     false,
			wantContains:  "",
		},
		{
			name:          "no_matching_investigation_no_keyword_fallback",
			workspaceName: "og-feat-dashboard-auto-discover-06jan-dfc6", // Would have matched by keywords before
			beadsID:       "orch-go-nomatch",                            // But no beads ID match in files
			projectDir:    tmpDir,
			wantFound:     false, // No longer matches - keyword fallback removed
			wantContains:  "",
		},
		{
			name:          "workspace_with_local_inv_file",
			workspaceName: "og-inv-my-workspace-06jan-1234",
			beadsID:       "orch-go-local",
			projectDir:    tmpDir,
			wantFound:     true, // Matches workspace directory file
			wantContains:  "inv-local-findings.md",
		},
		{
			name:          "no_beads_id_no_workspace_file_no_match",
			workspaceName: "og-inv-nonexistent-workspace-06jan-zzzz",
			beadsID:       "orch-go-xyz789", // No file contains this ID
			projectDir:    tmpDir,
			wantFound:     false,
			wantContains:  "",
		},
	}

	// Build cache for the test project directory
	cache := buildInvestigationDirCache([]string{tmpDir})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := discoverInvestigationPath(tt.workspaceName, tt.beadsID, tt.projectDir, cache)
			if tt.wantFound && got == "" {
				t.Errorf("discoverInvestigationPath() = empty, want path containing %q", tt.wantContains)
			}
			if !tt.wantFound && got != "" {
				t.Errorf("discoverInvestigationPath() = %q, want empty", got)
			}
			if tt.wantFound && tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("discoverInvestigationPath() = %q, want path containing %q", got, tt.wantContains)
			}
			if tt.wantFound && !filepath.IsAbs(got) {
				t.Errorf("discoverInvestigationPath() = %q, want absolute path", got)
			}
		})
	}
}
