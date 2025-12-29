package spawn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectUITask(t *testing.T) {
	tests := []struct {
		name          string
		task          string
		setupDirs     []string // directories to create in temp dir
		wantIsUITask  bool
		wantAutoMCP   bool
		wantMinConf   string // minimum expected confidence
	}{
		{
			name:          "high confidence - explicit UI keyword",
			task:          "fix the UI component rendering issue",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "high confidence - svelte file",
			task:          "update the Dashboard.svelte component",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "high confidence - tsx file",
			task:          "refactor the Header.tsx component",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "high confidence - web directory",
			task:          "add new feature in web/src/routes",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "high confidence - frontend directory",
			task:          "fix bug in frontend components",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "high confidence - visual keyword",
			task:          "fix visual regression in dashboard",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "high confidence - browser keyword",
			task:          "fix browser compatibility issue",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "medium confidence - multiple UI elements",
			task:          "add button and modal to the form",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "medium",
		},
		{
			name:          "medium confidence - styling task",
			task:          "update css styles for responsive layout",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "medium",
		},
		{
			name:          "low confidence - single medium keyword",
			task:          "add a button to the API handler",
			wantIsUITask:  true,
			wantAutoMCP:   false, // low confidence doesn't auto-add
			wantMinConf:   "low",
		},
		{
			name:          "no UI - pure backend task",
			task:          "implement authentication middleware",
			wantIsUITask:  false,
			wantAutoMCP:   false,
			wantMinConf:   "none",
		},
		{
			name:          "no UI - database task",
			task:          "optimize database queries for performance",
			wantIsUITask:  false,
			wantAutoMCP:   false,
			wantMinConf:   "none",
		},
		{
			name:          "no UI - CLI task",
			task:          "add new spawn command flag",
			wantIsUITask:  false,
			wantAutoMCP:   false,
			wantMinConf:   "none",
		},
		{
			name:          "high confidence - react keyword",
			task:          "create new React component for user profile",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "high confidence - vue keyword",
			task:          "fix Vue component lifecycle issue",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "high confidence - page keyword",
			task:          "add new settings page",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "case insensitive - UI uppercase",
			task:          "fix the UI rendering",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "jsx file extension",
			task:          "update Card.jsx component",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
		{
			name:          "src/routes path",
			task:          "add route in src/routes/dashboard",
			wantIsUITask:  true,
			wantAutoMCP:   true,
			wantMinConf:   "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var projectDir string
			if len(tt.setupDirs) > 0 {
				tempDir := t.TempDir()
				for _, dir := range tt.setupDirs {
					if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
						t.Fatalf("failed to create test dir: %v", err)
					}
				}
				projectDir = tempDir
			}

			result := DetectUITask(tt.task, projectDir)

			if result.IsUITask != tt.wantIsUITask {
				t.Errorf("IsUITask = %v, want %v", result.IsUITask, tt.wantIsUITask)
			}

			if result.ShouldAutoMCP != tt.wantAutoMCP {
				t.Errorf("ShouldAutoMCP = %v, want %v", result.ShouldAutoMCP, tt.wantAutoMCP)
			}

			// Check confidence level is at least what we expect
			confOrder := map[string]int{"none": 0, "low": 1, "medium": 2, "high": 3}
			if confOrder[result.Confidence] < confOrder[tt.wantMinConf] {
				t.Errorf("Confidence = %q, want at least %q", result.Confidence, tt.wantMinConf)
			}
		})
	}
}

func TestDetectUITask_WithProjectDirs(t *testing.T) {
	tests := []struct {
		name         string
		task         string
		setupDirs    []string
		wantIsUITask bool
		wantAutoMCP  bool
	}{
		{
			name:         "project with web/ directory",
			task:         "fix issue in the project",
			setupDirs:    []string{"web/src"},
			wantIsUITask: true,
			wantAutoMCP:  false, // only medium confidence from dir alone
		},
		{
			name:         "project with frontend/ directory",
			task:         "fix issue in the project",
			setupDirs:    []string{"frontend/components"},
			wantIsUITask: true,
			wantAutoMCP:  false, // only medium confidence from dir alone
		},
		{
			name:         "project with web/ and UI task",
			task:         "update the component styles",
			setupDirs:    []string{"web/src"},
			wantIsUITask: true,
			wantAutoMCP:  true, // medium from dir + medium from keyword = auto
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			for _, dir := range tt.setupDirs {
				if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
			}

			result := DetectUITask(tt.task, tempDir)

			if result.IsUITask != tt.wantIsUITask {
				t.Errorf("IsUITask = %v, want %v", result.IsUITask, tt.wantIsUITask)
			}

			if result.ShouldAutoMCP != tt.wantAutoMCP {
				t.Errorf("ShouldAutoMCP = %v, want %v (reasons: %v)", result.ShouldAutoMCP, tt.wantAutoMCP, result.Reasons)
			}
		})
	}
}

func TestFormatUIDetectionMessage(t *testing.T) {
	tests := []struct {
		name       string
		result     *UIDetectionResult
		wantEmpty  bool
		wantContains []string
	}{
		{
			name: "auto-add message",
			result: &UIDetectionResult{
				IsUITask:      true,
				ShouldAutoMCP: true,
				Reasons:       []string{"task mentions 'component'", "task mentions 'ui'"},
			},
			wantEmpty: false,
			wantContains: []string{
				"UI task detected",
				"--mcp playwright",
				"--no-mcp",
			},
		},
		{
			name: "no auto-add - low confidence",
			result: &UIDetectionResult{
				IsUITask:      true,
				ShouldAutoMCP: false,
				Reasons:       []string{"task mentions 'button'"},
			},
			wantEmpty: true,
		},
		{
			name: "no auto-add - not UI task",
			result: &UIDetectionResult{
				IsUITask:      false,
				ShouldAutoMCP: false,
				Reasons:       []string{},
			},
			wantEmpty: true,
		},
		{
			name: "truncates many reasons",
			result: &UIDetectionResult{
				IsUITask:      true,
				ShouldAutoMCP: true,
				Reasons:       []string{"reason1", "reason2", "reason3", "reason4", "reason5"},
			},
			wantEmpty: false,
			wantContains: []string{
				"reason1",
				"reason2",
				"reason3",
				"(and more)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := FormatUIDetectionMessage(tt.result)

			if tt.wantEmpty && msg != "" {
				t.Errorf("expected empty message, got: %q", msg)
			}

			if !tt.wantEmpty && msg == "" {
				t.Error("expected non-empty message, got empty")
			}

			for _, want := range tt.wantContains {
				if !contains(msg, want) {
					t.Errorf("message should contain %q, got: %q", want, msg)
				}
			}
		})
	}
}

// Note: contains helper is defined in gap_test.go
