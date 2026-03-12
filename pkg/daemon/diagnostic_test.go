package daemon

import (
	"testing"
	"time"
)

func TestClassifyFailureMode(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		agent    DiagnosticAgent
		want     FailureMode
		wantDesc string // substring of description
	}{
		{
			name: "QUESTION deadlock - agent in QUESTION phase with session",
			agent: DiagnosticAgent{
				BeadsID:    "test-001",
				Phase:      "QUESTION - Should we use JWT or session-based auth?",
				HasSession: true,
				UpdatedAt:  now.Add(-45 * time.Minute),
				Model:      "anthropic/claude-opus-4-5",
			},
			want: FailureModeQuestionDeadlock,
		},
		{
			name: "BLOCKED with session - concurrency ceiling",
			agent: DiagnosticAgent{
				BeadsID:    "test-002",
				Phase:      "BLOCKED - Concurrency limit reached",
				HasSession: true,
				UpdatedAt:  now.Add(-20 * time.Minute),
				Model:      "anthropic/claude-opus-4-5",
			},
			want: FailureModeConcurrencyCeiling,
		},
		{
			name: "non-Anthropic model stalled in Implementing",
			agent: DiagnosticAgent{
				BeadsID:    "test-003",
				Phase:      "Implementing - Adding middleware",
				HasSession: true,
				UpdatedAt:  now.Add(-45 * time.Minute),
				Model:      "openai/gpt-5.2-codex",
				Skill:      "feature-impl",
			},
			want: FailureModeModelIncompatibility,
		},
		{
			name: "non-Anthropic model stalled in Planning",
			agent: DiagnosticAgent{
				BeadsID:    "test-004",
				Phase:      "Planning - Analyzing codebase",
				HasSession: true,
				UpdatedAt:  now.Add(-45 * time.Minute),
				Model:      "openai/gpt-4o",
				Skill:      "architect",
			},
			want: FailureModeModelIncompatibility,
		},
		{
			name: "Anthropic model stalled in Implementing - generic phase stall",
			agent: DiagnosticAgent{
				BeadsID:    "test-005",
				Phase:      "Implementing - Adding tests",
				HasSession: true,
				UpdatedAt:  now.Add(-45 * time.Minute),
				Model:      "anthropic/claude-opus-4-5",
			},
			want: FailureModePhaseStall,
		},
		{
			name: "Phase Complete but no SYNTHESIS - compliance gap",
			agent: DiagnosticAgent{
				BeadsID:      "test-006",
				Phase:        "Complete - All tests passing",
				HasSession:   false,
				HasSynthesis: false,
				IsFullTier:   true,
				UpdatedAt:    now.Add(-2 * time.Hour),
				Model:        "anthropic/claude-opus-4-5",
			},
			want: FailureModeSynthesisGap,
		},
		{
			name: "Phase Complete with SYNTHESIS - not a failure",
			agent: DiagnosticAgent{
				BeadsID:      "test-007",
				Phase:        "Complete - Done",
				HasSession:   false,
				HasSynthesis: true,
				IsFullTier:   true,
				UpdatedAt:    now.Add(-1 * time.Hour),
			},
			want: FailureModeNone,
		},
		{
			name: "Light tier Complete without SYNTHESIS - not a failure",
			agent: DiagnosticAgent{
				BeadsID:      "test-008",
				Phase:        "Complete - Done",
				HasSession:   false,
				HasSynthesis: false,
				IsFullTier:   false,
				UpdatedAt:    now.Add(-1 * time.Hour),
			},
			want: FailureModeNone,
		},
		{
			name: "no phase reported, old agent - silent failure",
			agent: DiagnosticAgent{
				BeadsID:    "test-009",
				Phase:      "",
				HasSession: false,
				UpdatedAt:  now.Add(-2 * time.Hour),
				Model:      "anthropic/claude-opus-4-5",
			},
			want: FailureModeSilentFailure,
		},
		{
			name: "no phase but recent - not yet a failure",
			agent: DiagnosticAgent{
				BeadsID:    "test-010",
				Phase:      "",
				HasSession: true,
				UpdatedAt:  now.Add(-5 * time.Minute),
				Model:      "anthropic/claude-opus-4-5",
			},
			want: FailureModeNone,
		},
		{
			name: "Exploration phase with prior art indicator",
			agent: DiagnosticAgent{
				BeadsID:        "test-011",
				Phase:          "Exploration - Found prior agents completed overlapping work",
				HasSession:     true,
				UpdatedAt:      now.Add(-45 * time.Minute),
				Model:          "anthropic/claude-opus-4-5",
				HasPriorAgents: true,
			},
			want: FailureModePriorArtConfusion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyFailureMode(tt.agent)
			if result.Mode != tt.want {
				t.Errorf("ClassifyFailureMode() = %v, want %v (description: %s)", result.Mode, tt.want, result.Description)
			}
			if tt.wantDesc != "" && result.Description == "" {
				t.Errorf("expected description containing %q, got empty", tt.wantDesc)
			}
		})
	}
}

func TestClassifyFailureMode_RecommendedActions(t *testing.T) {
	now := time.Now()

	t.Run("QUESTION deadlock recommends notification", func(t *testing.T) {
		result := ClassifyFailureMode(DiagnosticAgent{
			BeadsID:    "test-q1",
			Phase:      "QUESTION - Need API key format",
			HasSession: true,
			UpdatedAt:  now.Add(-30 * time.Minute),
		})
		if len(result.RecommendedActions) == 0 {
			t.Fatal("expected recommended actions for QUESTION deadlock")
		}
		found := false
		for _, a := range result.RecommendedActions {
			if a.Action == ActionNotifyUser {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected ActionNotifyUser in recommended actions")
		}
	})

	t.Run("model incompatibility recommends respawn with Anthropic", func(t *testing.T) {
		result := ClassifyFailureMode(DiagnosticAgent{
			BeadsID:    "test-m1",
			Phase:      "Implementing - Stuck",
			HasSession: true,
			UpdatedAt:  now.Add(-45 * time.Minute),
			Model:      "openai/gpt-4o",
			Skill:      "investigation",
		})
		if len(result.RecommendedActions) == 0 {
			t.Fatal("expected recommended actions for model incompatibility")
		}
		found := false
		for _, a := range result.RecommendedActions {
			if a.Action == ActionRespawnWithModel {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected ActionRespawnWithModel in recommended actions")
		}
	})

	t.Run("concurrency ceiling recommends wait", func(t *testing.T) {
		result := ClassifyFailureMode(DiagnosticAgent{
			BeadsID:    "test-c1",
			Phase:      "BLOCKED - Max agents reached",
			HasSession: true,
			UpdatedAt:  now.Add(-20 * time.Minute),
		})
		found := false
		for _, a := range result.RecommendedActions {
			if a.Action == ActionWaitForSlot {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected ActionWaitForSlot in recommended actions")
		}
	})
}

func TestDiagnosticResult_String(t *testing.T) {
	result := DiagnosticResult{
		Mode:        FailureModeQuestionDeadlock,
		Description: "Agent waiting for answer",
		Severity:    SeverityActionable,
	}
	s := result.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}

func TestRunDiagnostics(t *testing.T) {
	now := time.Now()
	agents := []DiagnosticAgent{
		{
			BeadsID:    "a1",
			Phase:      "QUESTION - Need clarification",
			HasSession: true,
			UpdatedAt:  now.Add(-45 * time.Minute),
		},
		{
			BeadsID:    "a2",
			Phase:      "Complete - Done",
			HasSession: false,
			UpdatedAt:  now.Add(-1 * time.Hour),
			IsFullTier: true,
			HasSynthesis: true,
		},
		{
			BeadsID:    "a3",
			Phase:      "Implementing - In progress",
			HasSession: true,
			UpdatedAt:  now.Add(-45 * time.Minute),
			Model:      "openai/gpt-5.2-codex",
		},
	}

	report := RunDiagnostics(agents)
	if report.TotalAgents != 3 {
		t.Errorf("TotalAgents = %d, want 3", report.TotalAgents)
	}
	if report.HealthyCount != 1 {
		t.Errorf("HealthyCount = %d, want 1", report.HealthyCount)
	}
	if report.FailingCount != 2 {
		t.Errorf("FailingCount = %d, want 2", report.FailingCount)
	}
	if len(report.ByMode) == 0 {
		t.Error("ByMode should not be empty")
	}
	if _, ok := report.ByMode[FailureModeQuestionDeadlock]; !ok {
		t.Error("expected QUESTION deadlock in ByMode")
	}
	if _, ok := report.ByMode[FailureModeModelIncompatibility]; !ok {
		t.Error("expected model incompatibility in ByMode")
	}
}

func TestIsNonAnthropicModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"anthropic/claude-opus-4-5", false},
		{"anthropic/claude-sonnet-4-5", false},
		{"openai/gpt-4o", true},
		{"openai/gpt-5.2-codex", true},
		{"google/gemini-2.5-flash", true},
		{"", false}, // unknown defaults to not-non-Anthropic
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := isNonAnthropicModel(tt.model); got != tt.want {
				t.Errorf("isNonAnthropicModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestIsProtocolHeavySkill(t *testing.T) {
	tests := []struct {
		skill string
		want  bool
	}{
		{"architect", true},
		{"investigation", true},
		{"feature-impl", true},
		{"hello", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			if got := isProtocolHeavySkill(tt.skill); got != tt.want {
				t.Errorf("isProtocolHeavySkill(%q) = %v, want %v", tt.skill, got, tt.want)
			}
		})
	}
}
