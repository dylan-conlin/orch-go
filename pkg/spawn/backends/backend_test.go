package backends

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestSelect(t *testing.T) {
	tests := []struct {
		name           string
		inline         bool
		headless       bool
		tmux           bool
		attach         bool
		isOrchestrator bool
		wantBackend    string
	}{
		{
			name:           "inline flag takes precedence",
			inline:         true,
			headless:       true,
			tmux:           true,
			attach:         true,
			isOrchestrator: true,
			wantBackend:    "inline",
		},
		{
			name:           "headless flag takes precedence over tmux",
			inline:         false,
			headless:       true,
			tmux:           true,
			attach:         false,
			isOrchestrator: false,
			wantBackend:    "headless",
		},
		{
			name:           "tmux flag selects tmux backend",
			inline:         false,
			headless:       false,
			tmux:           true,
			attach:         false,
			isOrchestrator: false,
			wantBackend:    "tmux",
		},
		{
			name:           "attach flag selects tmux backend",
			inline:         false,
			headless:       false,
			tmux:           false,
			attach:         true,
			isOrchestrator: false,
			wantBackend:    "tmux",
		},
		{
			name:           "orchestrator defaults to tmux",
			inline:         false,
			headless:       false,
			tmux:           false,
			attach:         false,
			isOrchestrator: true,
			wantBackend:    "tmux",
		},
		{
			name:           "worker defaults to headless",
			inline:         false,
			headless:       false,
			tmux:           false,
			attach:         false,
			isOrchestrator: false,
			wantBackend:    "headless",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := Select(tt.inline, tt.headless, tt.tmux, tt.attach, tt.isOrchestrator)
			if backend.Name() != tt.wantBackend {
				t.Errorf("Select(%v, %v, %v, %v, %v) = %q, want %q",
					tt.inline, tt.headless, tt.tmux, tt.attach, tt.isOrchestrator,
					backend.Name(), tt.wantBackend)
			}
		})
	}
}

func TestFormatSessionTitle(t *testing.T) {
	tests := []struct {
		name          string
		workspaceName string
		beadsID       string
		want          string
	}{
		{
			name:          "with beads ID",
			workspaceName: "og-feat-test-01jan",
			beadsID:       "orch-go-1234",
			want:          "og-feat-test-01jan [orch-go-1234]",
		},
		{
			name:          "without beads ID",
			workspaceName: "og-feat-test-01jan",
			beadsID:       "",
			want:          "og-feat-test-01jan",
		},
		{
			name:          "empty workspace name with beads ID",
			workspaceName: "",
			beadsID:       "orch-go-1234",
			want:          " [orch-go-1234]",
		},
		{
			name:          "empty both",
			workspaceName: "",
			beadsID:       "",
			want:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSessionTitle(tt.workspaceName, tt.beadsID)
			if got != tt.want {
				t.Errorf("FormatSessionTitle(%q, %q) = %q, want %q",
					tt.workspaceName, tt.beadsID, got, tt.want)
			}
		})
	}
}

func TestBackendNames(t *testing.T) {
	tests := []struct {
		backend Backend
		want    string
	}{
		{&InlineBackend{}, "inline"},
		{&HeadlessBackend{}, "headless"},
		{&TmuxBackend{}, "tmux"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.backend.Name(); got != tt.want {
				t.Errorf("backend.Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAddGapAnalysisToEventData(t *testing.T) {
	t.Run("nil gap analysis", func(t *testing.T) {
		eventData := make(map[string]interface{})
		AddGapAnalysisToEventData(eventData, nil)

		// Should not add any fields
		if len(eventData) != 0 {
			t.Errorf("expected empty map, got %d fields", len(eventData))
		}
	})

	t.Run("gap analysis without gaps", func(t *testing.T) {
		eventData := make(map[string]interface{})
		gapAnalysis := &spawn.GapAnalysis{
			HasGaps:        false,
			ContextQuality: 85,
		}
		AddGapAnalysisToEventData(eventData, gapAnalysis)

		if eventData["gap_has_gaps"] != false {
			t.Errorf("gap_has_gaps = %v, want false", eventData["gap_has_gaps"])
		}
		if eventData["gap_context_quality"] != 85 {
			t.Errorf("gap_context_quality = %v, want 85", eventData["gap_context_quality"])
		}
	})

	t.Run("gap analysis with gaps", func(t *testing.T) {
		eventData := make(map[string]interface{})
		gapAnalysis := &spawn.GapAnalysis{
			HasGaps:        true,
			ContextQuality: 25,
			MatchStats: spawn.MatchStatistics{
				TotalMatches:       5,
				ConstraintCount:    2,
				DecisionCount:      2,
				InvestigationCount: 1,
			},
			Gaps: []spawn.Gap{
				{Type: spawn.GapTypeNoConstraints},
				{Type: spawn.GapTypeNoDecisions},
			},
		}
		AddGapAnalysisToEventData(eventData, gapAnalysis)

		if eventData["gap_has_gaps"] != true {
			t.Errorf("gap_has_gaps = %v, want true", eventData["gap_has_gaps"])
		}
		if eventData["gap_context_quality"] != 25 {
			t.Errorf("gap_context_quality = %v, want 25", eventData["gap_context_quality"])
		}
		if eventData["gap_match_total"] != 5 {
			t.Errorf("gap_match_total = %v, want 5", eventData["gap_match_total"])
		}
		if eventData["gap_match_constraints"] != 2 {
			t.Errorf("gap_match_constraints = %v, want 2", eventData["gap_match_constraints"])
		}

		gapTypes, ok := eventData["gap_types"].([]string)
		if !ok {
			t.Errorf("gap_types is not []string")
		}
		if len(gapTypes) != 2 {
			t.Errorf("gap_types length = %d, want 2", len(gapTypes))
		}
	})
}

func TestAddUsageInfoToEventData(t *testing.T) {
	t.Run("nil usage info", func(t *testing.T) {
		eventData := make(map[string]interface{})
		AddUsageInfoToEventData(eventData, nil)

		// Should not add any fields
		if len(eventData) != 0 {
			t.Errorf("expected empty map, got %d fields", len(eventData))
		}
	})

	t.Run("basic usage info", func(t *testing.T) {
		eventData := make(map[string]interface{})
		usageInfo := &spawn.UsageInfo{
			FiveHourUsed: 15.5,
			SevenDayUsed: 42.3,
			AccountEmail: "test@example.com",
		}
		AddUsageInfoToEventData(eventData, usageInfo)

		if eventData["usage_5h_used"] != 15.5 {
			t.Errorf("usage_5h_used = %v, want 15.5", eventData["usage_5h_used"])
		}
		if eventData["usage_weekly_used"] != 42.3 {
			t.Errorf("usage_weekly_used = %v, want 42.3", eventData["usage_weekly_used"])
		}
		if eventData["usage_account"] != "test@example.com" {
			t.Errorf("usage_account = %v, want test@example.com", eventData["usage_account"])
		}
	})

	t.Run("auto-switched account", func(t *testing.T) {
		eventData := make(map[string]interface{})
		usageInfo := &spawn.UsageInfo{
			FiveHourUsed: 85.2,
			SevenDayUsed: 90.1,
			AutoSwitched: true,
			SwitchReason: "Rate limit approached",
		}
		AddUsageInfoToEventData(eventData, usageInfo)

		if eventData["usage_auto_switched"] != true {
			t.Errorf("usage_auto_switched = %v, want true", eventData["usage_auto_switched"])
		}
		if eventData["usage_switch_reason"] != "Rate limit approached" {
			t.Errorf("usage_switch_reason = %v, want 'Rate limit approached'", eventData["usage_switch_reason"])
		}
	})
}

func TestFormatContextQualitySummary(t *testing.T) {
	tests := []struct {
		name         string
		gapAnalysis  *spawn.GapAnalysis
		wantContains string
	}{
		{
			name:         "nil gap analysis",
			gapAnalysis:  nil,
			wantContains: "not checked",
		},
		{
			name: "zero quality - critical",
			gapAnalysis: &spawn.GapAnalysis{
				ContextQuality: 0,
			},
			wantContains: "CRITICAL - No context",
		},
		{
			name: "poor quality (15)",
			gapAnalysis: &spawn.GapAnalysis{
				ContextQuality: 15,
			},
			wantContains: "poor",
		},
		{
			name: "limited quality (35)",
			gapAnalysis: &spawn.GapAnalysis{
				ContextQuality: 35,
			},
			wantContains: "limited",
		},
		{
			name: "moderate quality (55)",
			gapAnalysis: &spawn.GapAnalysis{
				ContextQuality: 55,
			},
			wantContains: "moderate",
		},
		{
			name: "good quality (75)",
			gapAnalysis: &spawn.GapAnalysis{
				ContextQuality: 75,
			},
			wantContains: "good",
		},
		{
			name: "excellent quality (95)",
			gapAnalysis: &spawn.GapAnalysis{
				ContextQuality: 95,
			},
			wantContains: "excellent",
		},
		{
			name: "with match stats",
			gapAnalysis: &spawn.GapAnalysis{
				ContextQuality: 75,
				MatchStats: spawn.MatchStatistics{
					TotalMatches:    10,
					ConstraintCount: 3,
				},
			},
			wantContains: "10 matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatContextQualitySummary(tt.gapAnalysis)
			if got == "" {
				t.Error("got empty string")
			}
			// Check that the result contains the expected substring
			if tt.wantContains != "" {
				if !contains(got, tt.wantContains) {
					t.Errorf("FormatContextQualitySummary() = %q, want to contain %q", got, tt.wantContains)
				}
			}
		})
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
