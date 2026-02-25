package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func TestShouldCountFileWithExclusions(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		exclusions []string
		expected   bool
	}{
		// Default exclusions should filter data/config files
		{
			name:       "json file excluded by default",
			path:       "data/events.json",
			exclusions: defaultExclusions,
			expected:   false,
		},
		{
			name:       "jsonl file excluded by default",
			path:       "logs/events.jsonl",
			exclusions: defaultExclusions,
			expected:   false,
		},
		{
			name:       "lock file excluded by default",
			path:       "package-lock.json",
			exclusions: defaultExclusions,
			expected:   false,
		},
		{
			name:       "yarn.lock excluded by default",
			path:       "yarn.lock",
			exclusions: defaultExclusions,
			expected:   false,
		},
		{
			name:       "go.sum excluded by default",
			path:       "go.sum",
			exclusions: defaultExclusions,
			expected:   false,
		},
		{
			name:       "source file not excluded",
			path:       "cmd/orch/main.go",
			exclusions: defaultExclusions,
			expected:   true,
		},
		// Custom exclusions
		{
			name:       "custom exclusion pattern",
			path:       "internal/config.yaml",
			exclusions: []string{"*.yaml"},
			expected:   false,
		},
		{
			name:       "file matching custom exclusion",
			path:       "scripts/deploy.sh",
			exclusions: []string{"*.sh"},
			expected:   false,
		},
		// Empty exclusions should not filter these files
		{
			name:       "json file allowed with empty exclusions",
			path:       "data/events.json",
			exclusions: []string{},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldCountFileWithExclusions(tt.path, tt.exclusions)
			if result != tt.expected {
				t.Errorf("shouldCountFileWithExclusions(%q, %v) = %v, want %v", tt.path, tt.exclusions, result, tt.expected)
			}
		})
	}
}

func TestDefaultExclusions(t *testing.T) {
	// Verify default exclusions include the expected patterns
	expectedPatterns := []string{"*.jsonl", "*.json", "*.lock", "go.sum"}
	for _, pattern := range expectedPatterns {
		found := false
		for _, exclusion := range defaultExclusions {
			if exclusion == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("defaultExclusions should contain %q", pattern)
		}
	}
}

func TestMatchesExclusionPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{"exact match", "go.sum", "go.sum", true},
		{"glob extension match", "data/events.json", "*.json", true},
		{"glob extension no match", "cmd/main.go", "*.json", false},
		{"nested path glob match", "logs/2024/events.jsonl", "*.jsonl", true},
		{"lock file match", "yarn.lock", "*.lock", true},
		{"package-lock.json matches .json", "package-lock.json", "*.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesExclusionPattern(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesExclusionPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestShouldCountFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Should count
		{"cmd/orch/main.go", true},
		{"pkg/spawn/spawn.go", true},
		{"web/src/components/App.tsx", true},
		{"internal/service/handler.go", true},

		// Should not count - test files
		{"cmd/orch/main_test.go", false},
		{"web/src/App.test.ts", false},
		{"web/src/App.test.js", false},

		// Should not count - generated/vendor
		{"vendor/github.com/pkg/errors/errors.go", false},
		{"internal/generated/proto.go", false},

		// Should not count - documentation
		{"README.md", false},
		{"docs/architecture.txt", false},

		// Should not count - config
		{"package.json", false},
		{"go.mod", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := shouldCountFile(tt.path)
			if result != tt.expected {
				t.Errorf("shouldCountFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGenerateFixRecommendation(t *testing.T) {
	tests := []struct {
		file     string
		count    int
		contains string // Substring that should be in recommendation
	}{
		{"handler.go", 12, "CRITICAL"},
		{"service.go", 8, "HIGH"},
		{"utils.go", 5, "MODERATE"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			rec := generateFixRecommendation(tt.file, tt.count)
			if !hotspotContains(rec, tt.contains) {
				t.Errorf("generateFixRecommendation(%q, %d) = %q, should contain %q",
					tt.file, tt.count, rec, tt.contains)
			}
		})
	}
}

func TestGenerateInvestigationRecommendation(t *testing.T) {
	tests := []struct {
		topic    string
		count    int
		urgency  string
		contains string
	}{
		{"auth", 12, "high", "CRITICAL"},
		{"auth", 10, "low", "CRITICAL"}, // Count alone triggers critical
		{"spawn", 6, "medium", "HIGH"},
		{"config", 3, "low", "MODERATE"},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			rec := generateInvestigationRecommendation(tt.topic, tt.count, tt.urgency)
			if !hotspotContains(rec, tt.contains) {
				t.Errorf("generateInvestigationRecommendation(%q, %d, %q) = %q, should contain %q",
					tt.topic, tt.count, tt.urgency, rec, tt.contains)
			}
		})
	}
}

func TestHotspotReportJSON(t *testing.T) {
	report := HotspotReport{
		GeneratedAt:    "2026-01-04T10:00:00Z",
		AnalysisPeriod: "Last 28 days",
		FixThreshold:   5,
		InvThreshold:   3,
		Hotspots: []Hotspot{
			{
				Path:           "cmd/orch/spawn.go",
				Type:           "fix-density",
				Score:          7,
				Details:        "7 fix commits in last 28 days",
				Recommendation: "HIGH: Spawn investigation",
			},
			{
				Path:           "auth",
				Type:           "investigation-cluster",
				Score:          5,
				Details:        "5 investigations on topic 'auth'",
				RelatedFiles:   []string{"pkg/auth/auth.go", "pkg/auth/token.go"},
				Recommendation: "HIGH: Consider design-session",
			},
		},
		TotalFixCommits:     42,
		TotalInvestigations: 15,
		HasArchitectWork:    true,
	}

	// Test JSON marshaling
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal report: %v", err)
	}

	// Unmarshal and verify
	var decoded HotspotReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal report: %v", err)
	}

	if len(decoded.Hotspots) != 2 {
		t.Errorf("Expected 2 hotspots, got %d", len(decoded.Hotspots))
	}
	if decoded.TotalFixCommits != 42 {
		t.Errorf("Expected TotalFixCommits=42, got %d", decoded.TotalFixCommits)
	}
	if !decoded.HasArchitectWork {
		t.Error("Expected HasArchitectWork=true")
	}
}

// Helper function for string contains check
func hotspotContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && hotspotContainsSubstring(s, substr))
}

func hotspotContainsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestExtractPathsFromTask(t *testing.T) {
	tests := []struct {
		name     string
		task     string
		expected []string
	}{
		{
			name:     "single file path",
			task:     "fix bug in cmd/orch/spawn.go",
			expected: []string{"cmd/orch/spawn.go"},
		},
		{
			name:     "multiple file paths",
			task:     "refactor pkg/spawn/context.go and cmd/orch/main.go",
			expected: []string{"pkg/spawn/context.go", "cmd/orch/main.go"},
		},
		{
			name:     "file path with extension",
			task:     "update web/src/components/Dashboard.tsx",
			expected: []string{"web/src/components/Dashboard.tsx"},
		},
		{
			name:     "no file paths",
			task:     "investigate auth issues",
			expected: []string{},
		},
		{
			name:     "directory path",
			task:     "reorganize pkg/daemon/ package",
			expected: []string{"pkg/daemon/"},
		},
		{
			name:     "mixed content",
			task:     "The file at cmd/orch/serve.go has 10+ conditions",
			expected: []string{"cmd/orch/serve.go"},
		},
		{
			name:     "path in quotes",
			task:     "fix issue in \"pkg/auth/token.go\"",
			expected: []string{"pkg/auth/token.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPathsFromTask(tt.task)
			if len(result) != len(tt.expected) {
				t.Errorf("extractPathsFromTask(%q) = %v, want %v", tt.task, result, tt.expected)
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("extractPathsFromTask(%q)[%d] = %q, want %q", tt.task, i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestMatchPathToHotspots(t *testing.T) {
	hotspots := []Hotspot{
		{Path: "cmd/orch/spawn.go", Type: "fix-density", Score: 7},
		{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 5},
		{Path: "auth", Type: "investigation-cluster", Score: 4},
	}

	tests := []struct {
		name          string
		path          string
		expectedMatch bool
		expectedScore int
	}{
		{
			name:          "exact match",
			path:          "cmd/orch/spawn.go",
			expectedMatch: true,
			expectedScore: 7,
		},
		{
			name:          "directory contains path",
			path:          "pkg/daemon/",
			expectedMatch: true,
			expectedScore: 5,
		},
		{
			name:          "no match",
			path:          "cmd/orch/main.go",
			expectedMatch: false,
			expectedScore: 0,
		},
		{
			name:          "topic match in path",
			path:          "pkg/auth/token.go",
			expectedMatch: true,
			expectedScore: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := matchPathToHotspots(tt.path, hotspots)
			if matched != tt.expectedMatch {
				t.Errorf("matchPathToHotspots(%q) matched = %v, want %v", tt.path, matched, tt.expectedMatch)
			}
			if score != tt.expectedScore {
				t.Errorf("matchPathToHotspots(%q) score = %d, want %d", tt.path, score, tt.expectedScore)
			}
		})
	}
}

func TestCheckSpawnHotspots(t *testing.T) {
	// Create mock hotspot data
	hotspots := []Hotspot{
		{Path: "cmd/orch/spawn.go", Type: "fix-density", Score: 7, Recommendation: "HIGH: Review spawn.go"},
		{Path: "status", Type: "investigation-cluster", Score: 5, Recommendation: "HIGH: Consider design-session for 'status'"},
	}

	result := checkSpawnHotspots("fix bug in cmd/orch/spawn.go related to status", hotspots)

	if !result.HasHotspots {
		t.Error("Expected HasHotspots=true")
	}
	if len(result.MatchedHotspots) != 2 {
		t.Errorf("Expected 2 matched hotspots, got %d", len(result.MatchedHotspots))
	}
	if result.MaxScore != 7 {
		t.Errorf("Expected MaxScore=7, got %d", result.MaxScore)
	}
}

func TestFormatHotspotWarning(t *testing.T) {
	result := &SpawnHotspotResult{
		HasHotspots: true,
		MatchedHotspots: []Hotspot{
			{Path: "cmd/orch/spawn.go", Score: 7, Recommendation: "HIGH: Review spawn.go"},
		},
		MaxScore: 7,
	}

	warning := formatHotspotWarning(result)
	if warning == "" {
		t.Error("Expected non-empty warning")
	}
	if !hotspotContains(warning, "HOTSPOT WARNING") {
		t.Errorf("Warning should contain 'HOTSPOT WARNING': %s", warning)
	}
	if !hotspotContains(warning, "architect") {
		t.Errorf("Warning should recommend architect: %s", warning)
	}
}

// TestCheckSpawnHotspots_CmdOrchMainGo tests the specific case of cmd/orch/main.go
// being detected as a hotspot when it appears in a task description.
// This validates the hotspot warning feature for high-churn files.
func TestCheckSpawnHotspots_CmdOrchMainGo(t *testing.T) {
	// Simulate the real-world scenario where cmd/orch/main.go is a hotspot
	// In the actual codebase, this file has 49 fix commits (CRITICAL level)
	hotspots := []Hotspot{
		{
			Path:           "cmd/orch/main.go",
			Type:           "fix-density",
			Score:          49,
			Details:        "49 fix commits in last 28 days",
			Recommendation: "CRITICAL: Consider spawning architect to redesign main.go - excessive fix churn indicates structural issues",
		},
	}

	tests := []struct {
		name          string
		task          string
		shouldMatch   bool
		expectedScore int
	}{
		{
			name:          "direct file reference",
			task:          "fix bug in cmd/orch/main.go",
			shouldMatch:   true,
			expectedScore: 49,
		},
		{
			name:          "file reference with context",
			task:          "refactor the spawn logic in cmd/orch/main.go to improve readability",
			shouldMatch:   true,
			expectedScore: 49,
		},
		{
			name:          "quoted file path",
			task:          "review changes to \"cmd/orch/main.go\"",
			shouldMatch:   true,
			expectedScore: 49,
		},
		{
			name:          "unrelated task",
			task:          "add new feature to the dashboard",
			shouldMatch:   false,
			expectedScore: 0,
		},
		{
			name:          "different file in same directory",
			task:          "fix bug in cmd/orch/serve.go",
			shouldMatch:   false,
			expectedScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSpawnHotspots(tt.task, hotspots)
			if result.HasHotspots != tt.shouldMatch {
				t.Errorf("checkSpawnHotspots(%q) HasHotspots = %v, want %v",
					tt.task, result.HasHotspots, tt.shouldMatch)
			}
			if result.MaxScore != tt.expectedScore {
				t.Errorf("checkSpawnHotspots(%q) MaxScore = %d, want %d",
					tt.task, result.MaxScore, tt.expectedScore)
			}
			// If it matched, verify the warning is formatted correctly
			if tt.shouldMatch && result.Warning == "" {
				t.Error("Expected non-empty warning when hotspot matches")
			}
			if tt.shouldMatch {
				// Verify warning contains key elements
				if !hotspotContains(result.Warning, "HOTSPOT WARNING") {
					t.Errorf("Warning should contain 'HOTSPOT WARNING': %s", result.Warning)
				}
				if !hotspotContains(result.Warning, "cmd/orch/main.go") {
					t.Errorf("Warning should mention the file path: %s", result.Warning)
				}
				if !hotspotContains(result.Warning, "architect") {
					t.Errorf("Warning should recommend architect review: %s", result.Warning)
				}
			}
		})
	}
}

// TestCheckSpawnHotspots_CouplingClusterMatchInTaskText validates that coupling-cluster
// hotspots are matched when the concept name appears in the task text, even without
// an explicit file path.
func TestCheckSpawnHotspots_CouplingClusterMatchInTaskText(t *testing.T) {
	hotspots := []Hotspot{
		{
			Path:         "daemon",
			Type:         "coupling-cluster",
			Score:        180,
			RelatedFiles: []string{"cmd/orch/daemon.go", "pkg/daemon/daemon.go", "web/src/lib/stores/daemon.ts"},
		},
	}

	tests := []struct {
		name          string
		task          string
		shouldMatch   bool
		expectedScore int
	}{
		{
			name:          "concept name in task text",
			task:          "refactor the daemon lifecycle management",
			shouldMatch:   true,
			expectedScore: 180,
		},
		{
			name:          "concept name with related file path",
			task:          "fix issue in cmd/orch/daemon.go",
			shouldMatch:   true,
			expectedScore: 180,
		},
		{
			name:          "unrelated task",
			task:          "add new authentication feature",
			shouldMatch:   false,
			expectedScore: 0,
		},
		{
			name:          "case insensitive match",
			task:          "Debug the Daemon spawn logic",
			shouldMatch:   true,
			expectedScore: 180,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSpawnHotspots(tt.task, hotspots)
			if result.HasHotspots != tt.shouldMatch {
				t.Errorf("checkSpawnHotspots(%q) HasHotspots = %v, want %v",
					tt.task, result.HasHotspots, tt.shouldMatch)
			}
			if result.MaxScore != tt.expectedScore {
				t.Errorf("checkSpawnHotspots(%q) MaxScore = %d, want %d",
					tt.task, result.MaxScore, tt.expectedScore)
			}
		})
	}
}

// TestCheckSpawnHotspots_MixedTypes validates that multiple hotspot types are all matched
// when relevant to the task.
func TestCheckSpawnHotspots_MixedTypes(t *testing.T) {
	hotspots := []Hotspot{
		{Path: "cmd/orch/spawn_cmd.go", Type: "fix-density", Score: 7},
		{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 1800},
		{Path: "spawn", Type: "coupling-cluster", Score: 120, RelatedFiles: []string{"cmd/orch/spawn_cmd.go", "pkg/spawn/config.go"}},
	}

	result := checkSpawnHotspots("fix bug in cmd/orch/spawn_cmd.go", hotspots)
	if !result.HasHotspots {
		t.Fatal("expected HasHotspots=true")
	}
	if len(result.MatchedHotspots) < 2 {
		t.Errorf("expected at least 2 matched hotspots (fix-density + bloat-size), got %d", len(result.MatchedHotspots))
	}
	// Should detect CRITICAL from bloat-size >1500
	if !result.HasCriticalHotspot {
		t.Error("expected HasCriticalHotspot=true for bloat-size 1800")
	}
	if result.MaxScore != 1800 {
		t.Errorf("MaxScore = %d, want 1800", result.MaxScore)
	}
}

// TestCheckSpawnHotspots_CriticalVsHighSeverity validates that different scores
// result in appropriate severity levels in warnings.
func TestCheckSpawnHotspots_CriticalVsHighSeverity(t *testing.T) {
	tests := []struct {
		name          string
		score         int
		expectedLevel string // CRITICAL, HIGH, or MODERATE
	}{
		{"score 10 is critical", 10, "CRITICAL"},
		{"score 49 is critical", 49, "CRITICAL"},
		{"score 7 is high", 7, "HIGH"},
		{"score 5 is moderate", 5, "MODERATE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hotspots := []Hotspot{
				{
					Path:           "cmd/orch/main.go",
					Type:           "fix-density",
					Score:          tt.score,
					Recommendation: generateFixRecommendation("main.go", tt.score),
				},
			}

			result := checkSpawnHotspots("fix bug in cmd/orch/main.go", hotspots)
			if !result.HasHotspots {
				t.Fatal("Expected hotspot match")
			}
			if !hotspotContains(result.MatchedHotspots[0].Recommendation, tt.expectedLevel) {
				t.Errorf("Score %d should have %s recommendation, got: %s",
					tt.score, tt.expectedLevel, result.MatchedHotspots[0].Recommendation)
			}
		})
	}
}

// TestCheckSpawnHotspots_CriticalBloatDetection validates that bloat-size files
// >1500 lines set HasCriticalHotspot and CriticalFiles.
func TestCheckSpawnHotspots_CriticalBloatDetection(t *testing.T) {
	tests := []struct {
		name           string
		lines          int
		expectCritical bool
	}{
		{"1501 lines is critical", 1501, true},
		{"2000 lines is critical", 2000, true},
		{"1500 lines is not critical", 1500, false},
		{"800 lines is not critical", 800, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hotspots := []Hotspot{
				{
					Path:           "cmd/orch/big_file.go",
					Type:           "bloat-size",
					Score:          tt.lines,
					Recommendation: generateBloatRecommendation("big_file.go", tt.lines),
				},
			}

			result := checkSpawnHotspots("fix cmd/orch/big_file.go", hotspots)
			if !result.HasHotspots {
				t.Fatal("Expected hotspot match")
			}
			if result.HasCriticalHotspot != tt.expectCritical {
				t.Errorf("Lines=%d: HasCriticalHotspot=%v, want %v",
					tt.lines, result.HasCriticalHotspot, tt.expectCritical)
			}
			if tt.expectCritical {
				if len(result.CriticalFiles) == 0 {
					t.Error("Expected CriticalFiles to be populated for critical hotspot")
				} else if result.CriticalFiles[0] != "cmd/orch/big_file.go" {
					t.Errorf("Expected CriticalFiles[0] = %q, got %q", "cmd/orch/big_file.go", result.CriticalFiles[0])
				}
			}
		})
	}
}

// TestRunHotspotCheckForSpawn_IncludesBloatAnalysis verifies that bloat-size
// hotspots (files >800 lines) are included in spawn-time hotspot checks.
// This tests the integration - that RunHotspotCheckForSpawn actually calls
// analyzeBloatFiles and includes bloat results in the check.
func TestRunHotspotCheckForSpawn_IncludesBloatAnalysis(t *testing.T) {
	// This test validates that the bloat analysis is performed during spawn checks.
	// We can't easily test with real file system, so we test the internal function
	// composition by checking that analyzeBloatFiles results are included.

	// Create a temporary directory with a large file
	tempDir := t.TempDir()

	// Create a Go file with >1500 lines (CRITICAL threshold)
	largeContent := "package main\n\nfunc main() {}\n"
	for i := 0; i < 1600; i++ {
		largeContent += "// line " + string(rune('0'+i%10)) + "\n"
	}

	largePath := tempDir + "/large_file.go"
	if err := writeFile(largePath, largeContent); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Initialize git repo (required for fix commit analysis)
	if err := initGitRepo(tempDir); err != nil {
		t.Skipf("Skipping test - could not init git: %v", err)
	}

	// Run the spawn hotspot check with a task that references the large file
	result, err := RunHotspotCheckForSpawn(tempDir, "fix bug in large_file.go")
	if err != nil {
		t.Fatalf("RunHotspotCheckForSpawn error: %v", err)
	}

	// Verify that the bloat-size hotspot was detected
	if result == nil {
		t.Fatal("Expected non-nil result - bloat-size hotspot should be detected")
	}

	if !result.HasHotspots {
		t.Error("Expected HasHotspots=true for large file")
	}

	// Check for bloat-size type in matched hotspots
	foundBloat := false
	for _, h := range result.MatchedHotspots {
		if h.Type == "bloat-size" {
			foundBloat = true
			if h.Score < 1500 {
				t.Errorf("Expected bloat-size score > 1500, got %d", h.Score)
			}
			break
		}
	}

	if !foundBloat {
		t.Error("Expected bloat-size hotspot in matched results - RunHotspotCheckForSpawn should include bloat analysis")
	}

	// Verify CRITICAL detection for >1500 lines
	if !result.HasCriticalHotspot {
		t.Error("Expected HasCriticalHotspot=true for file >1500 lines")
	}
}

// --- Tests for bloat-size matching in matchPathToHotspots ---

func TestMatchPathToHotspots_BloatSize(t *testing.T) {
	hotspots := []Hotspot{
		{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 1800},
		{Path: "pkg/daemon/daemon.go", Type: "bloat-size", Score: 900},
	}

	tests := []struct {
		name          string
		path          string
		expectedMatch bool
		expectedScore int
	}{
		{
			name:          "exact file match",
			path:          "cmd/orch/spawn_cmd.go",
			expectedMatch: true,
			expectedScore: 1800,
		},
		{
			name:          "directory contains bloated file",
			path:          "cmd/orch/",
			expectedMatch: true,
			expectedScore: 1800,
		},
		{
			name:          "different file in pkg",
			path:          "pkg/daemon/daemon.go",
			expectedMatch: true,
			expectedScore: 900,
		},
		{
			name:          "unrelated file",
			path:          "pkg/model/model.go",
			expectedMatch: false,
			expectedScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := matchPathToHotspots(tt.path, hotspots)
			if matched != tt.expectedMatch {
				t.Errorf("matchPathToHotspots(%q) matched = %v, want %v", tt.path, matched, tt.expectedMatch)
			}
			if score != tt.expectedScore {
				t.Errorf("matchPathToHotspots(%q) score = %d, want %d", tt.path, score, tt.expectedScore)
			}
		})
	}
}

// --- Tests for coupling-cluster matching via task text ---

func TestCheckSpawnHotspots_CouplingClusterViaTaskText(t *testing.T) {
	hotspots := []Hotspot{
		{
			Path:         "daemon",
			Type:         "coupling-cluster",
			Score:        180,
			RelatedFiles: []string{"cmd/orch/daemon.go", "pkg/daemon/daemon.go"},
		},
	}

	tests := []struct {
		name        string
		task        string
		shouldMatch bool
	}{
		{
			name:        "topic appears in task text",
			task:        "refactor daemon error handling",
			shouldMatch: true,
		},
		{
			name:        "topic in task with other words",
			task:        "the daemon module needs retry logic",
			shouldMatch: true,
		},
		{
			name:        "related file referenced in task",
			task:        "fix cmd/orch/daemon.go race condition",
			shouldMatch: true,
		},
		{
			name:        "unrelated task",
			task:        "add pagination to status command",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSpawnHotspots(tt.task, hotspots)
			if result.HasHotspots != tt.shouldMatch {
				t.Errorf("checkSpawnHotspots(%q) HasHotspots = %v, want %v",
					tt.task, result.HasHotspots, tt.shouldMatch)
			}
			if tt.shouldMatch && len(result.MatchedHotspots) == 0 {
				t.Error("expected matched hotspots when HasHotspots is true")
			}
			if tt.shouldMatch && result.MatchedHotspots[0].Type != "coupling-cluster" {
				t.Errorf("expected coupling-cluster type, got %q", result.MatchedHotspots[0].Type)
			}
		})
	}
}

// --- Tests for bloat-size directory containment in checkSpawnHotspots ---

func TestCheckSpawnHotspots_BloatSizeDirectoryContainment(t *testing.T) {
	hotspots := []Hotspot{
		{
			Path:           "cmd/orch/spawn_cmd.go",
			Type:           "bloat-size",
			Score:          1800,
			Recommendation: "CRITICAL: spawn_cmd.go needs extraction",
		},
	}

	tests := []struct {
		name           string
		task           string
		shouldMatch    bool
		expectCritical bool
	}{
		{
			name:           "exact file in task",
			task:           "fix cmd/orch/spawn_cmd.go validation",
			shouldMatch:    true,
			expectCritical: true,
		},
		{
			name:           "directory reference matches",
			task:           "reorganize cmd/orch/ package",
			shouldMatch:    true,
			expectCritical: true,
		},
		{
			name:           "unrelated file in same dir",
			task:           "fix cmd/orch/status_cmd.go",
			shouldMatch:    false,
			expectCritical: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSpawnHotspots(tt.task, hotspots)
			if result.HasHotspots != tt.shouldMatch {
				t.Errorf("checkSpawnHotspots(%q) HasHotspots = %v, want %v",
					tt.task, result.HasHotspots, tt.shouldMatch)
			}
			if result.HasCriticalHotspot != tt.expectCritical {
				t.Errorf("checkSpawnHotspots(%q) HasCriticalHotspot = %v, want %v",
					tt.task, result.HasCriticalHotspot, tt.expectCritical)
			}
		})
	}
}

// --- Test for multiple hotspot types in one task ---

func TestCheckSpawnHotspots_MultipleHotspotTypes(t *testing.T) {
	hotspots := []Hotspot{
		{
			Path:           "cmd/orch/spawn_cmd.go",
			Type:           "fix-density",
			Score:          12,
			Recommendation: "CRITICAL: fix churn",
		},
		{
			Path:           "cmd/orch/spawn_cmd.go",
			Type:           "bloat-size",
			Score:          1800,
			Recommendation: "CRITICAL: needs extraction",
		},
		{
			Path:         "spawn",
			Type:         "coupling-cluster",
			Score:        45,
			RelatedFiles: []string{"cmd/orch/spawn_cmd.go", "pkg/spawn/config.go"},
		},
	}

	result := checkSpawnHotspots("refactor spawn logic in cmd/orch/spawn_cmd.go", hotspots)

	if !result.HasHotspots {
		t.Fatal("expected HasHotspots=true for multi-type match")
	}

	// Should match all three hotspot types
	if len(result.MatchedHotspots) != 3 {
		t.Errorf("expected 3 matched hotspots (fix-density + bloat-size + coupling-cluster), got %d", len(result.MatchedHotspots))
	}

	// Should detect CRITICAL from bloat-size >1500
	if !result.HasCriticalHotspot {
		t.Error("expected HasCriticalHotspot=true (bloat-size >1500)")
	}

	// CriticalFiles should contain the bloated file
	if len(result.CriticalFiles) == 0 {
		t.Error("expected CriticalFiles to be populated")
	}

	// MaxScore should be the highest (1800 from bloat-size)
	if result.MaxScore != 1800 {
		t.Errorf("expected MaxScore=1800, got %d", result.MaxScore)
	}

	// Warning should be non-empty
	if result.Warning == "" {
		t.Error("expected non-empty warning")
	}
}

// --- Test for coupling-cluster matching via RelatedFiles ---

func TestMatchPathToHotspots_CouplingClusterRelatedFiles(t *testing.T) {
	hotspots := []Hotspot{
		{
			Path:         "agent-status",
			Type:         "coupling-cluster",
			Score:        60,
			RelatedFiles: []string{"cmd/orch/serve_agents.go", "web/src/lib/stores/agents.ts"},
		},
	}

	tests := []struct {
		name          string
		path          string
		expectedMatch bool
	}{
		{
			name:          "exact related file match",
			path:          "cmd/orch/serve_agents.go",
			expectedMatch: true,
		},
		{
			name:          "directory containing related file",
			path:          "cmd/orch/",
			expectedMatch: true,
		},
		{
			name:          "web related file",
			path:          "web/src/lib/stores/agents.ts",
			expectedMatch: true,
		},
		{
			name:          "unrelated file",
			path:          "pkg/model/model.go",
			expectedMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _ := matchPathToHotspots(tt.path, hotspots)
			if matched != tt.expectedMatch {
				t.Errorf("matchPathToHotspots(%q) matched = %v, want %v", tt.path, matched, tt.expectedMatch)
			}
		})
	}
}

// --- Test checkSpawnHotspots with empty hotspots ---

func TestCheckSpawnHotspots_NoHotspots(t *testing.T) {
	result := checkSpawnHotspots("fix cmd/orch/main.go", nil)
	if result.HasHotspots {
		t.Error("expected HasHotspots=false with nil hotspots")
	}
	if result.HasCriticalHotspot {
		t.Error("expected HasCriticalHotspot=false with nil hotspots")
	}

	result = checkSpawnHotspots("fix cmd/orch/main.go", []Hotspot{})
	if result.HasHotspots {
		t.Error("expected HasHotspots=false with empty hotspots")
	}
}

// --- Test formatHotspotWarning edge cases ---

func TestFormatHotspotWarning_NoHotspots(t *testing.T) {
	result := &SpawnHotspotResult{
		HasHotspots: false,
	}
	warning := formatHotspotWarning(result)
	if warning != "" {
		t.Errorf("expected empty warning for no hotspots, got: %s", warning)
	}
}

func TestFormatHotspotWarning_EmptyMatchedHotspots(t *testing.T) {
	result := &SpawnHotspotResult{
		HasHotspots:     true,
		MatchedHotspots: []Hotspot{},
	}
	warning := formatHotspotWarning(result)
	if warning != "" {
		t.Errorf("expected empty warning for empty matched hotspots, got: %s", warning)
	}
}

// Helper to write file
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// --- Tests for basename/suffix matching in matchPathToHotspots ---

func TestMatchPathToHotspots_BasenameSuffixMatch(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		hotspots      []Hotspot
		expectedMatch bool
		expectedScore int
	}{
		{
			name:          "basename matches fix-density full path",
			path:          "complete_cmd.go",
			hotspots:      []Hotspot{{Path: "cmd/orch/complete_cmd.go", Type: "fix-density", Score: 12}},
			expectedMatch: true,
			expectedScore: 12,
		},
		{
			name:          "partial path suffix matches fix-density",
			path:          "orch/complete_cmd.go",
			hotspots:      []Hotspot{{Path: "cmd/orch/complete_cmd.go", Type: "fix-density", Score: 12}},
			expectedMatch: true,
			expectedScore: 12,
		},
		{
			name:          "basename matches bloat-size full path",
			path:          "hotspot.go",
			hotspots:      []Hotspot{{Path: "cmd/orch/hotspot.go", Type: "bloat-size", Score: 1800}},
			expectedMatch: true,
			expectedScore: 1800,
		},
		{
			name:          "partial path suffix matches bloat-size",
			path:          "orch/hotspot.go",
			hotspots:      []Hotspot{{Path: "cmd/orch/hotspot.go", Type: "bloat-size", Score: 1800}},
			expectedMatch: true,
			expectedScore: 1800,
		},
		{
			name:          "unrelated basename does not match",
			path:          "main.go",
			hotspots:      []Hotspot{{Path: "cmd/orch/complete_cmd.go", Type: "fix-density", Score: 12}},
			expectedMatch: false,
			expectedScore: 0,
		},
		{
			name:          "exact match still works",
			path:          "cmd/orch/complete_cmd.go",
			hotspots:      []Hotspot{{Path: "cmd/orch/complete_cmd.go", Type: "fix-density", Score: 12}},
			expectedMatch: true,
			expectedScore: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := matchPathToHotspots(tt.path, tt.hotspots)
			if matched != tt.expectedMatch {
				t.Errorf("matchPathToHotspots(%q) matched = %v, want %v", tt.path, matched, tt.expectedMatch)
			}
			if score != tt.expectedScore {
				t.Errorf("matchPathToHotspots(%q) score = %d, want %d", tt.path, score, tt.expectedScore)
			}
		})
	}
}

func TestMatchPathToHotspots_CouplingClusterRelatedFilesSuffix(t *testing.T) {
	hotspots := []Hotspot{
		{
			Path:         "agent-status",
			Type:         "coupling-cluster",
			Score:        60,
			RelatedFiles: []string{"cmd/orch/serve_agents.go", "web/src/lib/stores/agents.ts"},
		},
	}

	tests := []struct {
		name          string
		path          string
		expectedMatch bool
	}{
		{
			name:          "basename matches related file",
			path:          "serve_agents.go",
			expectedMatch: true,
		},
		{
			name:          "partial suffix matches related file",
			path:          "orch/serve_agents.go",
			expectedMatch: true,
		},
		{
			name:          "unrelated basename",
			path:          "main.go",
			expectedMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _ := matchPathToHotspots(tt.path, hotspots)
			if matched != tt.expectedMatch {
				t.Errorf("matchPathToHotspots(%q) matched = %v, want %v", tt.path, matched, tt.expectedMatch)
			}
		})
	}
}

// TestCheckSpawnHotspots_BasenameInTaskText validates the end-to-end flow:
// task text contains bare filename → extractPathsFromTask extracts it → matchPathToHotspots matches via suffix.
func TestCheckSpawnHotspots_BasenameInTaskText(t *testing.T) {
	hotspots := []Hotspot{
		{
			Path:           "cmd/orch/complete_cmd.go",
			Type:           "bloat-size",
			Score:          1800,
			Recommendation: "CRITICAL: complete_cmd.go needs extraction",
		},
		{
			Path:           "cmd/orch/complete_cmd.go",
			Type:           "fix-density",
			Score:          12,
			Recommendation: "CRITICAL: fix churn in complete_cmd.go",
		},
	}

	tests := []struct {
		name        string
		task        string
		shouldMatch bool
	}{
		{
			name:        "bare filename in task text",
			task:        "fix validation in complete_cmd.go",
			shouldMatch: true,
		},
		{
			name:        "full path still works",
			task:        "fix cmd/orch/complete_cmd.go validation",
			shouldMatch: true,
		},
		{
			name:        "unrelated file",
			task:        "fix main.go startup",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSpawnHotspots(tt.task, hotspots)
			if result.HasHotspots != tt.shouldMatch {
				t.Errorf("checkSpawnHotspots(%q) HasHotspots = %v, want %v",
					tt.task, result.HasHotspots, tt.shouldMatch)
			}
		})
	}
}

// Helper to initialize a minimal git repo
func initGitRepo(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	return cmd.Run()
}

func TestExtractInvestigationKeywords(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected []string
	}{
		{
			name:     "standard investigation with date and type prefix",
			filename: "2026-02-19-design-coupling-hotspot-analysis-system.md",
			expected: []string{"coupling", "hotspot", "analysis", "system"},
		},
		{
			name:     "investigation with inv prefix",
			filename: "2026-01-04-inv-integrate-hotspot-detection-into-orch.md",
			expected: []string{"hotspot", "detection"},
		},
		{
			name:     "investigation with audit prefix",
			filename: "2026-01-03-audit-comprehensive-orch-go-bugs-reliability-architecture.md",
			expected: []string{"bugs", "reliability", "architecture"},
		},
		{
			name:     "investigation with generic terms filtered",
			filename: "2026-01-03-inv-document-changelog-pattern-ecosystem-expansion.md",
			expected: []string{"changelog", "pattern", "ecosystem", "expansion"},
		},
		{
			name:     "spike type prefix",
			filename: "2026-02-24-spike-claude-code-hooks-orchestrator-guard.md",
			expected: []string{"claude", "code", "hooks", "orchestrator", "guard"},
		},
		{
			name:     "filters comprehensive, document, integrate",
			filename: "2026-02-14-audit-comprehensive-orch-go-codebase-audit.md",
			expected: []string{"codebase"},
		},
		{
			name:     "doctor topic preserved",
			filename: "2025-12-26-inv-add-orch-doctor-command-check.md",
			expected: []string{"doctor", "command"},
		},
		{
			name:     "workers topic preserved",
			filename: "2025-12-22-inv-workers-stall-during-build-phase.md",
			expected: []string{"workers", "stall", "build"},
		},
		{
			name:     "handoff topic preserved",
			filename: "2025-12-22-inv-orch-handoff-generates-stale-incorrect.md",
			expected: []string{"handoff", "generates", "stale", "incorrect"},
		},
		{
			name:     "p0 prefix stripped",
			filename: "2026-01-09-inv-p0-implement-orch-doctor-health.md",
			expected: []string{"doctor", "health"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractInvestigationKeywords(tt.filename)
			if len(result) != len(tt.expected) {
				t.Errorf("extractInvestigationKeywords(%q) = %v (len %d), want %v (len %d)",
					tt.filename, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i, kw := range result {
				if kw != tt.expected[i] {
					t.Errorf("extractInvestigationKeywords(%q)[%d] = %q, want %q",
						tt.filename, i, kw, tt.expected[i])
				}
			}
		})
	}
}

func TestIsInvestigationStopWord(t *testing.T) {
	// These should be filtered out
	stopWords := []string{
		"comprehensive", "document", "integrate", "design",
		"investigate", "add", "fix", "implement",
		"into", "review", "check", "update",
		"orch", "go", "the", "and", "for",
		"new", "use", "how", "why", "what",
	}
	for _, word := range stopWords {
		if !isInvestigationStopWord(word) {
			t.Errorf("isInvestigationStopWord(%q) = false, want true", word)
		}
	}

	// These should NOT be filtered out
	keepWords := []string{
		"doctor", "workers", "handoff", "entropy",
		"daemon", "spawn", "hotspot", "coupling",
		"auth", "token", "session", "config",
		"changelog", "dashboard", "status",
	}
	for _, word := range keepWords {
		if isInvestigationStopWord(word) {
			t.Errorf("isInvestigationStopWord(%q) = true, want false", word)
		}
	}
}

func TestAnalyzeInvestigationClusters_DirectScan(t *testing.T) {
	// Create a temp directory with mock investigation files
	tempDir := t.TempDir()
	kbDir := tempDir + "/.kb/investigations"
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatalf("Failed to create kb dir: %v", err)
	}

	// Create mock investigation files that should cluster on "doctor"
	doctorFiles := []string{
		"2025-12-26-inv-add-orch-doctor-command-check.md",
		"2026-01-09-inv-p0-implement-orch-doctor-health.md",
		"2026-01-10-inv-p0-implement-orch-doctor-health.md",
	}
	for _, f := range doctorFiles {
		if err := os.WriteFile(kbDir+"/"+f, []byte("# Test\n"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Create files that would falsely cluster on "comprehensive" with old approach
	compFiles := []string{
		"2026-01-03-audit-comprehensive-orch-go-bugs-reliability-architecture.md",
		"2026-01-15-inv-add-comprehensive-orch-clean-all.md",
		"2026-02-14-audit-comprehensive-orch-go-codebase-audit.md",
	}
	for _, f := range compFiles {
		if err := os.WriteFile(kbDir+"/"+f, []byte("# Test\n"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	hotspots, totalInv, err := analyzeInvestigationClusters(tempDir, 3)
	if err != nil {
		t.Fatalf("analyzeInvestigationClusters error: %v", err)
	}

	if totalInv != 6 {
		t.Errorf("totalInv = %d, want 6", totalInv)
	}

	// "doctor" should be a hotspot (3 investigations)
	foundDoctor := false
	foundComprehensive := false
	for _, h := range hotspots {
		if h.Path == "doctor" {
			foundDoctor = true
			if h.Score != 3 {
				t.Errorf("doctor score = %d, want 3", h.Score)
			}
		}
		if h.Path == "comprehensive" {
			foundComprehensive = true
		}
	}

	if !foundDoctor {
		t.Errorf("Expected 'doctor' hotspot, got: %v", hotspots)
	}
	if foundComprehensive {
		t.Error("Should NOT have 'comprehensive' hotspot - it's a generic stop word")
	}
}
