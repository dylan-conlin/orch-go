package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindWorkspaceByPartialName(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	workspaceBase := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceBase, 0755); err != nil {
		t.Fatalf("Failed to create workspace base: %v", err)
	}

	// Create test workspaces
	workspaces := []string{
		"og-feat-auth-06jan-abc1",
		"og-feat-login-06jan-def2",
		"og-inv-test-06jan-ghi3",
	}
	for _, ws := range workspaces {
		if err := os.MkdirAll(filepath.Join(workspaceBase, ws), 0755); err != nil {
			t.Fatalf("Failed to create workspace %s: %v", ws, err)
		}
	}

	tests := []struct {
		name        string
		partialName string
		want        string
		wantErr     bool
	}{
		{
			name:        "exact match",
			partialName: "og-feat-auth-06jan-abc1",
			want:        "og-feat-auth-06jan-abc1",
			wantErr:     false,
		},
		{
			name:        "partial match - unique",
			partialName: "auth",
			want:        "og-feat-auth-06jan-abc1",
			wantErr:     false,
		},
		{
			name:        "partial match - unique inv",
			partialName: "inv-test",
			want:        "og-inv-test-06jan-ghi3",
			wantErr:     false,
		},
		{
			name:        "no match",
			partialName: "nonexistent",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "multiple matches",
			partialName: "06jan",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "multiple matches - og-feat",
			partialName: "og-feat",
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindWorkspaceByPartialName(tmpDir, tt.partialName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindWorkspaceByPartialName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FindWorkspaceByPartialName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsPartialMatch(t *testing.T) {
	tests := []struct {
		name    string
		wsName  string
		partial string
		want    bool
	}{
		{
			name:    "exact match",
			wsName:  "og-feat-auth-06jan",
			partial: "og-feat-auth-06jan",
			want:    true,
		},
		{
			name:    "substring match",
			wsName:  "og-feat-auth-06jan",
			partial: "auth",
			want:    true,
		},
		{
			name:    "substring match - prefix",
			wsName:  "og-feat-auth-06jan",
			partial: "og-feat",
			want:    true,
		},
		{
			name:    "substring match - suffix",
			wsName:  "og-feat-auth-06jan",
			partial: "06jan",
			want:    true,
		},
		{
			name:    "no match",
			wsName:  "og-feat-auth-06jan",
			partial: "xyz",
			want:    false,
		},
		{
			name:    "empty partial",
			wsName:  "og-feat-auth-06jan",
			partial: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsPartialMatch(tt.wsName, tt.partial); got != tt.want {
				t.Errorf("containsPartialMatch(%q, %q) = %v, want %v", tt.wsName, tt.partial, got, tt.want)
			}
		})
	}
}

func TestAttachCommand_WorkspaceNotFound(t *testing.T) {
	// Create temp directory without workspace
	tmpDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldDir)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .orch/workspace directory but no workspace
	workspaceBase := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceBase, 0755); err != nil {
		t.Fatalf("Failed to create workspace base: %v", err)
	}

	err = runAttach("nonexistent-workspace")
	if err == nil {
		t.Error("Expected error for nonexistent workspace, got nil")
	}
}

func TestAttachCommand_NoSessionID(t *testing.T) {
	// Create temp directory with workspace but no session ID
	tmpDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldDir)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create workspace directory without .session_id
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "test-workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	err = runAttach("test-workspace")
	if err == nil {
		t.Error("Expected error for workspace without session ID, got nil")
	}
}
