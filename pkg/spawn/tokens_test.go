package spawn

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name      string
		charCount int
		want      int
	}{
		{"empty", 0, 0},
		{"small", 100, 25},
		{"medium", 4000, 1000},
		{"large", 400000, 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.charCount)
			if got != tt.want {
				t.Errorf("EstimateTokens(%d) = %d, want %d", tt.charCount, got, tt.want)
			}
		})
	}
}

func TestTokenEstimate_ExceedsWarning(t *testing.T) {
	tests := []struct {
		name             string
		estimatedTokens  int
		warningThreshold int
		want             bool
	}{
		{"below threshold", 50000, 100000, false},
		{"at threshold", 100000, 100000, true},
		{"above threshold", 150000, 100000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &TokenEstimate{
				EstimatedTokens:  tt.estimatedTokens,
				WarningThreshold: tt.warningThreshold,
			}
			got := e.ExceedsWarning()
			if got != tt.want {
				t.Errorf("ExceedsWarning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenEstimate_ExceedsError(t *testing.T) {
	tests := []struct {
		name            string
		estimatedTokens int
		errorThreshold  int
		want            bool
	}{
		{"below threshold", 100000, 150000, false},
		{"at threshold", 150000, 150000, true},
		{"above threshold", 200000, 150000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &TokenEstimate{
				EstimatedTokens: tt.estimatedTokens,
				ErrorThreshold:  tt.errorThreshold,
			}
			got := e.ExceedsError()
			if got != tt.want {
				t.Errorf("ExceedsError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenEstimate_UtilizationPercent(t *testing.T) {
	tests := []struct {
		name             string
		estimatedTokens  int
		warningThreshold int
		want             float64
	}{
		{"zero threshold", 100, 0, 0},
		{"50 percent", 50000, 100000, 50},
		{"100 percent", 100000, 100000, 100},
		{"150 percent", 150000, 100000, 150},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &TokenEstimate{
				EstimatedTokens:  tt.estimatedTokens,
				WarningThreshold: tt.warningThreshold,
			}
			got := e.UtilizationPercent()
			if got != tt.want {
				t.Errorf("UtilizationPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenEstimate_FormatSummary(t *testing.T) {
	t.Run("small context", func(t *testing.T) {
		e := &TokenEstimate{
			EstimatedTokens:  5000,
			WarningThreshold: 100000,
			Components:       map[string]int{"template": 750, "task": 250, "skill": 4000},
		}
		summary := e.FormatSummary()
		if !strings.Contains(summary, "5k tokens") {
			t.Error("summary should contain token count")
		}
		// Small contexts don't show breakdown
		if strings.Contains(summary, "Components:") {
			t.Error("small context should not show component breakdown")
		}
	})

	t.Run("large context with breakdown", func(t *testing.T) {
		e := &TokenEstimate{
			EstimatedTokens:  50000,
			WarningThreshold: 100000,
			Components:       map[string]int{"template": 750, "task": 250, "skill": 45000, "kb_context": 4000},
		}
		summary := e.FormatSummary()
		if !strings.Contains(summary, "50k tokens") {
			t.Error("summary should contain token count")
		}
		if !strings.Contains(summary, "Components:") {
			t.Error("large context should show component breakdown")
		}
		if !strings.Contains(summary, "skill") {
			t.Error("should show skill component (> 1000 tokens)")
		}
		if !strings.Contains(summary, "kb_context") {
			t.Error("should show kb_context component (> 1000 tokens)")
		}
	})
}

func TestEstimateContextTokens(t *testing.T) {
	t.Run("minimal config", func(t *testing.T) {
		cfg := &Config{
			Task: "test task",
		}
		estimate := EstimateContextTokens(cfg)

		if estimate.EstimatedTokens == 0 {
			t.Error("estimated tokens should be > 0")
		}
		if estimate.Components["template"] == 0 {
			t.Error("should have template component")
		}
		if estimate.Components["task"] == 0 {
			t.Error("should have task component")
		}
		if estimate.WarningThreshold != DefaultTokenWarningThreshold {
			t.Errorf("warning threshold = %d, want %d", estimate.WarningThreshold, DefaultTokenWarningThreshold)
		}
	})

	t.Run("full config", func(t *testing.T) {
		cfg := &Config{
			Task:          strings.Repeat("x", 400),   // 100 tokens
			SkillContent:  strings.Repeat("x", 40000), // 10000 tokens
			KBContext:     strings.Repeat("x", 8000),  // 2000 tokens
			ServerContext: strings.Repeat("x", 400),   // 100 tokens
		}
		estimate := EstimateContextTokens(cfg)

		if estimate.Components["skill"] == 0 {
			t.Error("should have skill component")
		}
		if estimate.Components["kb_context"] == 0 {
			t.Error("should have kb_context component")
		}
		if estimate.Components["server_context"] == 0 {
			t.Error("should have server_context component")
		}

		// Total should be sum of components
		total := 0
		for _, tokens := range estimate.Components {
			total += tokens
		}
		if estimate.EstimatedTokens != total {
			t.Errorf("EstimatedTokens (%d) should equal sum of components (%d)", estimate.EstimatedTokens, total)
		}
	})
}

func TestEstimateContentTokens(t *testing.T) {
	content := strings.Repeat("x", 4000) // 1000 tokens
	estimate := EstimateContentTokens(content)

	if estimate.CharCount != 4000 {
		t.Errorf("CharCount = %d, want 4000", estimate.CharCount)
	}
	if estimate.EstimatedTokens != 1000 {
		t.Errorf("EstimatedTokens = %d, want 1000", estimate.EstimatedTokens)
	}
	if estimate.Components["content"] != 1000 {
		t.Errorf("content component = %d, want 1000", estimate.Components["content"])
	}
}

func TestValidateContextSize(t *testing.T) {
	t.Run("acceptable size", func(t *testing.T) {
		cfg := &Config{
			Task:         strings.Repeat("x", 400),   // 100 tokens
			SkillContent: strings.Repeat("x", 40000), // 10000 tokens
		}
		err := ValidateContextSize(cfg)
		if err != nil {
			t.Errorf("unexpected error for acceptable size: %v", err)
		}
	})

	t.Run("exceeds error threshold", func(t *testing.T) {
		// Create config that exceeds 150k tokens
		// Need 600k characters (150k * 4)
		cfg := &Config{
			Task:         strings.Repeat("x", 4000),
			SkillContent: strings.Repeat("x", 600000),
		}
		err := ValidateContextSize(cfg)
		if err == nil {
			t.Error("expected error for oversized context")
		}

		// Should be a ContextTooLargeError
		ctlErr, ok := err.(*ContextTooLargeError)
		if !ok {
			t.Fatalf("expected *ContextTooLargeError, got %T", err)
		}
		if ctlErr.LargestComponent != "skill" {
			t.Errorf("LargestComponent = %s, want skill", ctlErr.LargestComponent)
		}
	})
}

func TestContextTooLargeError_Error(t *testing.T) {
	err := &ContextTooLargeError{
		Estimate: &TokenEstimate{
			EstimatedTokens: 160000,
			ErrorThreshold:  150000,
			Components: map[string]int{
				"skill": 155000,
				"task":  5000,
			},
		},
		LargestComponent: "skill",
	}

	msg := err.Error()
	if !strings.Contains(msg, "160k") {
		t.Error("error message should contain estimated tokens")
	}
	if !strings.Contains(msg, "150k") {
		t.Error("error message should contain limit")
	}
	if !strings.Contains(msg, "skill") {
		t.Error("error message should mention largest component")
	}
}

func TestShouldWarnAboutSize(t *testing.T) {
	t.Run("small context - no warning", func(t *testing.T) {
		cfg := &Config{
			Task:         strings.Repeat("x", 400),
			SkillContent: strings.Repeat("x", 4000),
		}
		shouldWarn, msg := ShouldWarnAboutSize(cfg)
		if shouldWarn {
			t.Error("should not warn for small context")
		}
		if msg != "" {
			t.Errorf("message should be empty, got: %s", msg)
		}
	})

	t.Run("large context with skill - warning", func(t *testing.T) {
		// Create config that exceeds 100k tokens (warning threshold)
		// Need 400k characters
		cfg := &Config{
			Task:         strings.Repeat("x", 4000),
			SkillContent: strings.Repeat("x", 400000),
		}
		shouldWarn, msg := ShouldWarnAboutSize(cfg)
		if !shouldWarn {
			t.Error("should warn for large context")
		}
		if !strings.Contains(msg, "Large spawn context") {
			t.Error("warning should mention large context")
		}
		if !strings.Contains(msg, "skill") {
			t.Error("warning should mention skill as the issue")
		}
	})

	t.Run("large context with kb_context - warning", func(t *testing.T) {
		cfg := &Config{
			Task:      strings.Repeat("x", 4000),
			KBContext: strings.Repeat("x", 400000),
		}
		shouldWarn, msg := ShouldWarnAboutSize(cfg)
		if !shouldWarn {
			t.Error("should warn for large context")
		}
		if !strings.Contains(msg, "kb_context") {
			t.Error("warning should mention kb_context as the issue")
		}
	})
}

func TestFindLargestComponent(t *testing.T) {
	tests := []struct {
		name       string
		components map[string]int
		want       string
	}{
		{"empty", map[string]int{}, ""},
		{"single", map[string]int{"skill": 100}, "skill"},
		{"multiple", map[string]int{"skill": 100, "task": 50, "kb_context": 200}, "kb_context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findLargestComponent(tt.components)
			if got != tt.want {
				t.Errorf("findLargestComponent() = %s, want %s", got, tt.want)
			}
		})
	}
}
