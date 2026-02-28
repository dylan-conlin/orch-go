package spawn

import "testing"

func TestVerifyLevelConstants(t *testing.T) {
	// Verify the four levels exist with expected values
	if VerifyV0 != "V0" {
		t.Errorf("VerifyV0 = %q, want %q", VerifyV0, "V0")
	}
	if VerifyV1 != "V1" {
		t.Errorf("VerifyV1 = %q, want %q", VerifyV1, "V1")
	}
	if VerifyV2 != "V2" {
		t.Errorf("VerifyV2 = %q, want %q", VerifyV2, "V2")
	}
	if VerifyV3 != "V3" {
		t.Errorf("VerifyV3 = %q, want %q", VerifyV3, "V3")
	}
}

func TestDefaultVerifyLevel_SkillDefaults(t *testing.T) {
	tests := []struct {
		skill     string
		issueType string
		want      string
	}{
		// V0 skills
		{"issue-creation", "", VerifyV0},
		{"issue-creation", "task", VerifyV0},
		{"capture-knowledge", "", VerifyV0},

		// V1 skills
		{"investigation", "", VerifyV1},
		{"architect", "", VerifyV1},
		{"research", "", VerifyV1},
		{"codebase-audit", "", VerifyV1},
		{"design-session", "", VerifyV1},
		{"probe", "", VerifyV1},
		{"ux-audit", "", VerifyV1},

		// V2 skills
		{"feature-impl", "", VerifyV2},
		{"systematic-debugging", "", VerifyV2},
		{"reliability-testing", "", VerifyV2},

		// V3 skills
		{"debug-with-playwright", "", VerifyV3},

		// Unknown skill defaults to V1 (conservative)
		{"unknown-skill", "", VerifyV1},
	}

	for _, tt := range tests {
		t.Run(tt.skill+"_"+tt.issueType, func(t *testing.T) {
			got := DefaultVerifyLevel(tt.skill, tt.issueType)
			if got != tt.want {
				t.Errorf("DefaultVerifyLevel(%q, %q) = %q, want %q", tt.skill, tt.issueType, got, tt.want)
			}
		})
	}
}

func TestDefaultVerifyLevel_IssueTypeMinimums(t *testing.T) {
	tests := []struct {
		skill     string
		issueType string
		want      string
	}{
		// Issue type elevates V1 skill to V2 for feature/bug/decision
		{"investigation", "feature", VerifyV2},
		{"investigation", "bug", VerifyV2},
		{"investigation", "decision", VerifyV2},

		// Issue type elevates V0 skill to V1 for investigation/probe
		{"issue-creation", "investigation", VerifyV1},
		{"issue-creation", "probe", VerifyV1},

		// No minimum for task/question - skill default prevails
		{"issue-creation", "task", VerifyV0},
		{"issue-creation", "question", VerifyV0},

		// Feature-impl + feature issue type stays V2 (already at minimum)
		{"feature-impl", "feature", VerifyV2},

		// feature issue elevates investigation skill to V2
		{"architect", "feature", VerifyV2},
	}

	for _, tt := range tests {
		t.Run(tt.skill+"_"+tt.issueType, func(t *testing.T) {
			got := DefaultVerifyLevel(tt.skill, tt.issueType)
			if got != tt.want {
				t.Errorf("DefaultVerifyLevel(%q, %q) = %q, want %q", tt.skill, tt.issueType, got, tt.want)
			}
		})
	}
}

func TestVerifyLevelOrder(t *testing.T) {
	// Verify that level comparison works correctly
	tests := []struct {
		a, b string
		want int // -1 if a < b, 0 if equal, 1 if a > b
	}{
		{VerifyV0, VerifyV0, 0},
		{VerifyV0, VerifyV1, -1},
		{VerifyV0, VerifyV2, -1},
		{VerifyV0, VerifyV3, -1},
		{VerifyV1, VerifyV0, 1},
		{VerifyV1, VerifyV1, 0},
		{VerifyV2, VerifyV1, 1},
		{VerifyV3, VerifyV2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := CompareVerifyLevels(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareVerifyLevels(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMaxVerifyLevel(t *testing.T) {
	tests := []struct {
		a, b string
		want string
	}{
		{VerifyV0, VerifyV1, VerifyV1},
		{VerifyV2, VerifyV1, VerifyV2},
		{VerifyV0, VerifyV0, VerifyV0},
		{VerifyV3, VerifyV0, VerifyV3},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := MaxVerifyLevel(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("MaxVerifyLevel(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestVerifyLevelForTier(t *testing.T) {
	tests := []struct {
		name       string
		tier       string
		skillLevel string
		want       string
	}{
		// Light tier caps to V0 regardless of skill default
		{"light_caps_V2_to_V0", TierLight, VerifyV2, VerifyV0},
		{"light_caps_V1_to_V0", TierLight, VerifyV1, VerifyV0},
		{"light_keeps_V0", TierLight, VerifyV0, VerifyV0},

		// Full tier does not cap
		{"full_keeps_V2", TierFull, VerifyV2, VerifyV2},
		{"full_keeps_V1", TierFull, VerifyV1, VerifyV1},
		{"full_keeps_V3", TierFull, VerifyV3, VerifyV3},

		// Empty tier (unknown) does not cap
		{"empty_tier_keeps_V2", "", VerifyV2, VerifyV2},
		{"empty_tier_keeps_V1", "", VerifyV1, VerifyV1},

		// Unknown tier does not cap (conservative: don't reduce verification)
		{"unknown_tier_keeps_V2", "custom", VerifyV2, VerifyV2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyLevelForTier(tt.tier, tt.skillLevel)
			if got != tt.want {
				t.Errorf("VerifyLevelForTier(%q, %q) = %q, want %q", tt.tier, tt.skillLevel, got, tt.want)
			}
		})
	}
}

func TestIsValidVerifyLevel(t *testing.T) {
	tests := []struct {
		level string
		want  bool
	}{
		{VerifyV0, true},
		{VerifyV1, true},
		{VerifyV2, true},
		{VerifyV3, true},
		{"v0", false}, // case sensitive
		{"V4", false},
		{"", false},
		{"light", false},
		{"full", false},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := IsValidVerifyLevel(tt.level)
			if got != tt.want {
				t.Errorf("IsValidVerifyLevel(%q) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}
