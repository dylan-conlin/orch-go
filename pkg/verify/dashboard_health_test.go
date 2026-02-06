package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestHasDashboardChanges tests detection of dashboard-touching files
func TestHasDashboardChanges(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected bool
	}{
		{
			name:     "web files should trigger health check",
			files:    []string{"web/src/App.tsx", "web/package.json"},
			expected: true,
		},
		{
			name:     "serve_beads.go should trigger health check",
			files:    []string{"cmd/orch/serve_beads.go"},
			expected: true,
		},
		{
			name:     "serve_attention.go should trigger health check",
			files:    []string{"cmd/orch/serve_attention.go"},
			expected: true,
		},
		{
			name:     "non-dashboard files should not trigger",
			files:    []string{"cmd/orch/spawn.go", "pkg/session/session.go"},
			expected: false,
		},
		{
			name:     "empty file list should not trigger",
			files:    []string{},
			expected: false,
		},
		{
			name:     "mixed files - should trigger if any match",
			files:    []string{"cmd/orch/spawn.go", "web/src/App.tsx"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasDashboardChanges(tt.files)
			if result != tt.expected {
				t.Errorf("hasDashboardChanges(%v) = %v, want %v", tt.files, result, tt.expected)
			}
		})
	}
}

// TestCheckDashboardHealth tests the HTTP health check logic
func TestCheckDashboardHealth(t *testing.T) {
	tests := []struct {
		name        string
		graphResp   BeadsGraphResponse
		beadsResp   BeadsStatsResponse
		graphStatus int
		beadsStatus int
		wantErr     bool
		errContains string
	}{
		{
			name: "healthy dashboard - both endpoints return data",
			graphResp: BeadsGraphResponse{
				NodeCount: 5,
				EdgeCount: 3,
			},
			beadsResp: BeadsStatsResponse{
				OpenIssues: 2,
			},
			graphStatus: http.StatusOK,
			beadsStatus: http.StatusOK,
			wantErr:     false,
		},
		{
			name: "zero node count should fail",
			graphResp: BeadsGraphResponse{
				NodeCount: 0,
				EdgeCount: 0,
			},
			beadsResp: BeadsStatsResponse{
				OpenIssues: 2,
			},
			graphStatus: http.StatusOK,
			beadsStatus: http.StatusOK,
			wantErr:     true,
			errContains: "node_count",
		},
		{
			name: "zero open issues should fail",
			graphResp: BeadsGraphResponse{
				NodeCount: 5,
				EdgeCount: 3,
			},
			beadsResp: BeadsStatsResponse{
				OpenIssues: 0,
			},
			graphStatus: http.StatusOK,
			beadsStatus: http.StatusOK,
			wantErr:     true,
			errContains: "open_issues",
		},
		{
			name:        "graph endpoint error should fail",
			graphResp:   BeadsGraphResponse{},
			beadsResp:   BeadsStatsResponse{OpenIssues: 2},
			graphStatus: http.StatusInternalServerError,
			beadsStatus: http.StatusOK,
			wantErr:     true,
			errContains: "/api/beads/graph",
		},
		{
			name:        "beads endpoint error should fail",
			graphResp:   BeadsGraphResponse{NodeCount: 5},
			beadsResp:   BeadsStatsResponse{},
			graphStatus: http.StatusOK,
			beadsStatus: http.StatusInternalServerError,
			wantErr:     true,
			errContains: "/api/beads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/beads/graph" {
					w.WriteHeader(tt.graphStatus)
					if tt.graphStatus == http.StatusOK {
						json.NewEncoder(w).Encode(tt.graphResp)
					}
				} else if r.URL.Path == "/api/beads" {
					w.WriteHeader(tt.beadsStatus)
					if tt.beadsStatus == http.StatusOK {
						json.NewEncoder(w).Encode(tt.beadsResp)
					}
				}
			}))
			defer server.Close()

			err := checkDashboardHealth(server.URL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("checkDashboardHealth() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("checkDashboardHealth() error = %v, should contain %s", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("checkDashboardHealth() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestVerifyDashboardHealth tests the full verification flow
func TestVerifyDashboardHealth(t *testing.T) {
	// Create a temporary workspace with SPAWN_CONTEXT.md
	tempDir := t.TempDir()
	workspacePath := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}

	projectDir := tempDir

	// Initialize git repo for testing
	setupGitRepo(t, projectDir)

	t.Run("no dashboard changes - should pass", func(t *testing.T) {
		// Commit a non-dashboard file
		testFile := filepath.Join(projectDir, "pkg", "test.go")
		os.MkdirAll(filepath.Dir(testFile), 0755)
		os.WriteFile(testFile, []byte("package test"), 0644)
		runGitCmd(t, projectDir, "add", ".")
		runGitCmd(t, projectDir, "commit", "-m", "test")

		result := VerifyDashboardHealth(workspacePath, projectDir, "")
		if result != nil && !result.Passed {
			t.Errorf("VerifyDashboardHealth() should pass for non-dashboard changes, got: %v", result.Errors)
		}
	})

	t.Run("dashboard changes but no server - should fail", func(t *testing.T) {
		// Commit a dashboard file
		testFile := filepath.Join(projectDir, "web", "src", "App.tsx")
		os.MkdirAll(filepath.Dir(testFile), 0755)
		os.WriteFile(testFile, []byte("export default App"), 0644)
		runGitCmd(t, projectDir, "add", ".")
		runGitCmd(t, projectDir, "commit", "-m", "dashboard change")

		result := VerifyDashboardHealth(workspacePath, projectDir, "http://localhost:9999")
		if result == nil || result.Passed {
			t.Error("VerifyDashboardHealth() should fail when server is unreachable")
		}
	})

	t.Run("dashboard changes with healthy server - should pass", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/beads/graph" {
				json.NewEncoder(w).Encode(BeadsGraphResponse{NodeCount: 5, EdgeCount: 3})
			} else if r.URL.Path == "/api/beads" {
				json.NewEncoder(w).Encode(BeadsStatsResponse{OpenIssues: 2})
			}
		}))
		defer server.Close()

		result := VerifyDashboardHealth(workspacePath, projectDir, server.URL)
		if result != nil && !result.Passed {
			t.Errorf("VerifyDashboardHealth() should pass with healthy server, got: %v", result.Errors)
		}
	})
}

// Helper functions
// (contains function is defined in check_test.go and reused here)

func setupGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@example.com")
	runGitCmd(t, dir, "config", "user.name", "Test User")
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
}
