package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestExtractWorkspaceKeywords(t *testing.T) {
	tests := []struct {
		name          string
		workspaceName string
		wantKeywords  []string
	}{
		{
			name:          "standard_investigation_workspace",
			workspaceName: "og-inv-skillc-deploy-06jan-ed96",
			wantKeywords:  []string{"skillc", "deploy"},
		},
		{
			name:          "feature_workspace",
			workspaceName: "og-feat-dashboard-auto-discover-06jan-dfc6",
			wantKeywords:  []string{"dashboard", "auto", "discover"},
		},
		{
			name:          "debug_workspace",
			workspaceName: "og-debug-status-polling-05dec-ab12",
			wantKeywords:  []string{"status", "polling"},
		},
		{
			name:          "short_workspace_name",
			workspaceName: "og-inv",
			wantKeywords:  nil,
		},
		{
			name:          "empty_workspace_name",
			workspaceName: "",
			wantKeywords:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWorkspaceKeywords(tt.workspaceName)
			if len(got) != len(tt.wantKeywords) {
				t.Errorf("extractWorkspaceKeywords(%q) = %v, want %v", tt.workspaceName, got, tt.wantKeywords)
				return
			}
			for i := range got {
				if got[i] != tt.wantKeywords[i] {
					t.Errorf("extractWorkspaceKeywords(%q)[%d] = %q, want %q", tt.workspaceName, i, got[i], tt.wantKeywords[i])
				}
			}
		})
	}
}

func TestIsHexLike(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abcd", true},
		{"1234", true},
		{"a1b2", true},
		{"ed96", true},
		{"dfc6", true},
		{"ABCD", false},
		{"ghij", false},
		{"test", false},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isHexLike(tt.input); got != tt.expected {
				t.Errorf("isHexLike(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDiscoverInvestigationPath(t *testing.T) {
	tmpDir := t.TempDir()

	invDir := filepath.Join(tmpDir, ".kb", "investigations")
	if err := os.MkdirAll(invDir, 0755); err != nil {
		t.Fatalf("Failed to create investigations dir: %v", err)
	}

	invFiles := []string{
		"2026-01-06-inv-dashboard-auto-discover.md",
		"2026-01-05-inv-status-polling.md",
		"2026-01-04-inv-skillc-deploy-structure.md",
	}
	for _, name := range invFiles {
		if err := os.WriteFile(filepath.Join(invDir, name), []byte("# Investigation"), 0644); err != nil {
			t.Fatalf("Failed to create investigation file: %v", err)
		}
	}

	simpleDir := filepath.Join(invDir, "simple")
	if err := os.MkdirAll(simpleDir, 0755); err != nil {
		t.Fatalf("Failed to create simple dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simpleDir, "2026-01-06-simple-test.md"), []byte("# Simple"), 0644); err != nil {
		t.Fatalf("Failed to create simple investigation: %v", err)
	}

	wsDir := filepath.Join(tmpDir, ".orch", "workspace", "og-inv-my-workspace-06jan-1234")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "SPAWN_CONTEXT.md"), []byte("# Context"), 0644); err != nil {
		t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "SYNTHESIS.md"), []byte("# Synthesis"), 0644); err != nil {
		t.Fatalf("Failed to create SYNTHESIS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "inv-local-findings.md"), []byte("# Findings"), 0644); err != nil {
		t.Fatalf("Failed to create local investigation: %v", err)
	}

	tests := []struct {
		name          string
		workspaceName string
		beadsID       string
		projectDir    string
		wantFound     bool
		wantContains  string
	}{
		{
			name:          "match_by_workspace_keywords",
			workspaceName: "og-feat-dashboard-auto-discover-06jan-dfc6",
			beadsID:       "orch-go-wrrks",
			projectDir:    tmpDir,
			wantFound:     true,
			wantContains:  "dashboard-auto-discover",
		},
		{
			name:          "match_by_workspace_keywords_skillc",
			workspaceName: "og-inv-skillc-deploy-structure-06jan-ed96",
			beadsID:       "orch-go-xyz",
			projectDir:    tmpDir,
			wantFound:     true,
			wantContains:  "skillc-deploy-structure",
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
			name:          "no_matching_investigation",
			workspaceName: "og-inv-nonexistent-topic-06jan-1234",
			beadsID:       "orch-go-nomatch",
			projectDir:    tmpDir,
			wantFound:     false,
			wantContains:  "",
		},
		{
			name:          "workspace_with_local_inv_file",
			workspaceName: "og-inv-my-workspace-06jan-1234",
			beadsID:       "orch-go-local",
			projectDir:    tmpDir,
			wantFound:     true,
			wantContains:  "inv-local-findings.md",
		},
	}

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
			if tt.wantFound && tt.wantContains != "" && !filepath.IsAbs(got) {
				t.Errorf("discoverInvestigationPath() = %q, want absolute path", got)
			}
		})
	}
}

func TestListActiveIssuesSingleProject(t *testing.T) {
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	defer func() {
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
	}()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(projectDir string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-open":    {ID: "orch-go-open", Status: "open"},
			"orch-go-active":  {ID: "orch-go-active", Status: "in_progress"},
			"orch-go-blocked": {ID: "orch-go-blocked", Status: "blocked"},
			"orch-go-closed":  {ID: "orch-go-closed", Status: "closed"},
		}, nil
	}

	issues, projectDirs := listActiveIssues([]string{"/tmp/project"})
	if len(projectDirs) != 3 {
		t.Fatalf("Expected 3 project dirs (open + in_progress + blocked), got %d", len(projectDirs))
	}
	if _, ok := issues["orch-go-active"]; !ok {
		t.Fatal("Expected in_progress issue to be included")
	}
	if _, ok := issues["orch-go-open"]; !ok {
		t.Fatal("Expected open issue to be included (newly spawned agents may have open status)")
	}
	if _, ok := issues["orch-go-blocked"]; !ok {
		t.Fatal("Expected blocked issue to be included (blocked agents should be visible in dashboard)")
	}
	if _, ok := issues["orch-go-closed"]; ok {
		t.Fatal("Expected closed issue to be excluded")
	}
}

func TestListActiveIssuesCrossProjectDedup(t *testing.T) {
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	defer func() {
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
	}()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(projectDir string) (map[string]*verify.Issue, error) {
		switch projectDir {
		case "/tmp/project-a":
			return map[string]*verify.Issue{
				"orch-go-a1":     {ID: "orch-go-a1", Status: "in_progress"},
				"orch-go-shared": {ID: "orch-go-shared", Status: "in_progress"},
			}, nil
		case "/tmp/project-b":
			return map[string]*verify.Issue{
				"orch-go-shared": {ID: "orch-go-shared", Status: "in_progress"},
				"orch-go-b1":     {ID: "orch-go-b1", Status: "in_progress"},
			}, nil
		default:
			return nil, nil
		}
	}

	issues, projectDirs := listActiveIssues([]string{"/tmp/project-a", "/tmp/project-b"})
	if len(issues) != 3 {
		t.Fatalf("Expected 3 deduplicated issues, got %d", len(issues))
	}
	if projectDirs["orch-go-shared"] != "/tmp/project-a" {
		t.Fatalf("Expected shared issue to keep first project dir, got %s", projectDirs["orch-go-shared"])
	}
}

func TestListActiveIssuesEmptyProjectDirs(t *testing.T) {
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	defer func() {
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
	}()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-active":  {ID: "orch-go-active", Status: "in_progress"},
			"orch-go-blocked": {ID: "orch-go-blocked", Status: "blocked"},
			"orch-go-closed":  {ID: "orch-go-closed", Status: "closed"},
		}, nil
	}
	listOpenIssuesWithDir = func(projectDir string) (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}

	issues, projectDirs := listActiveIssues(nil)
	if len(projectDirs) != 0 {
		t.Fatalf("Expected no project dir mappings, got %d", len(projectDirs))
	}
	if _, ok := issues["orch-go-active"]; !ok {
		t.Fatal("Expected in_progress issue to be included")
	}
	if _, ok := issues["orch-go-blocked"]; !ok {
		t.Fatal("Expected blocked issue to be included (blocked agents should be visible in dashboard)")
	}
	if _, ok := issues["orch-go-closed"]; ok {
		t.Fatal("Expected closed issue to be excluded")
	}
}

func TestListActiveIssuesErrorHandling(t *testing.T) {
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	defer func() {
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
	}()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(projectDir string) (map[string]*verify.Issue, error) {
		if projectDir == "/tmp/project-a" {
			return nil, fmt.Errorf("boom")
		}
		return map[string]*verify.Issue{
			"orch-go-ok": {ID: "orch-go-ok", Status: "in_progress"},
		}, nil
	}

	issues, _ := listActiveIssues([]string{"/tmp/project-a", "/tmp/project-b"})
	if _, ok := issues["orch-go-ok"]; !ok {
		t.Fatal("Expected in_progress issue from healthy project")
	}
}
