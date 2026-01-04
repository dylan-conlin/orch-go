package main

import (
	"encoding/json"
	"testing"
)

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
