package spawn

import "testing"

func TestReviewTierConstants(t *testing.T) {
	if ReviewAuto != "auto" {
		t.Errorf("ReviewAuto = %q, want %q", ReviewAuto, "auto")
	}
	if ReviewScan != "scan" {
		t.Errorf("ReviewScan = %q, want %q", ReviewScan, "scan")
	}
	if ReviewReview != "review" {
		t.Errorf("ReviewReview = %q, want %q", ReviewReview, "review")
	}
	if ReviewDeep != "deep" {
		t.Errorf("ReviewDeep = %q, want %q", ReviewDeep, "deep")
	}
}

func TestDefaultReviewTier_SkillDefaults(t *testing.T) {
	tests := []struct {
		skill     string
		issueType string
		want      string
	}{
		// auto skills
		{"capture-knowledge", "", ReviewAuto},
		{"issue-creation", "", ReviewAuto},

		// scan skills
		{"investigation", "", ReviewScan},
		{"probe", "", ReviewScan},
		{"research", "", ReviewScan},
		{"codebase-audit", "", ReviewScan},
		{"design-session", "", ReviewScan},
		{"ux-audit", "", ReviewScan},

		// review skills
		{"feature-impl", "", ReviewReview},
		{"systematic-debugging", "", ReviewReview},
		{"architect", "", ReviewReview},
		{"reliability-testing", "", ReviewReview},

		// deep skills
		{"debug-with-playwright", "", ReviewDeep},

		// Unknown skill defaults to review (conservative)
		{"unknown-skill", "", ReviewReview},
	}

	for _, tt := range tests {
		t.Run(tt.skill+"_"+tt.issueType, func(t *testing.T) {
			got := DefaultReviewTier(tt.skill, tt.issueType)
			if got != tt.want {
				t.Errorf("DefaultReviewTier(%q, %q) = %q, want %q", tt.skill, tt.issueType, got, tt.want)
			}
		})
	}
}

func TestDefaultReviewTier_IssueTypeMinimums(t *testing.T) {
	tests := []struct {
		skill     string
		issueType string
		want      string
	}{
		// Feature/bug issue type elevates scan to review
		{"investigation", "feature", ReviewReview},
		{"investigation", "bug", ReviewReview},

		// Decision issue type elevates scan to review
		{"investigation", "decision", ReviewReview},

		// No minimum for task/question - skill default prevails
		{"capture-knowledge", "task", ReviewAuto},
		{"capture-knowledge", "question", ReviewAuto},

		// feature-impl + feature stays at review (already at minimum)
		{"feature-impl", "feature", ReviewReview},

		// Investigation/probe issue types don't elevate scan skills
		{"research", "investigation", ReviewScan},
		{"research", "probe", ReviewScan},
	}

	for _, tt := range tests {
		t.Run(tt.skill+"_"+tt.issueType, func(t *testing.T) {
			got := DefaultReviewTier(tt.skill, tt.issueType)
			if got != tt.want {
				t.Errorf("DefaultReviewTier(%q, %q) = %q, want %q", tt.skill, tt.issueType, got, tt.want)
			}
		})
	}
}

func TestCompareReviewTiers(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{ReviewAuto, ReviewAuto, 0},
		{ReviewAuto, ReviewScan, -1},
		{ReviewAuto, ReviewReview, -1},
		{ReviewAuto, ReviewDeep, -1},
		{ReviewScan, ReviewAuto, 1},
		{ReviewScan, ReviewScan, 0},
		{ReviewReview, ReviewScan, 1},
		{ReviewDeep, ReviewReview, 1},
		{ReviewDeep, ReviewAuto, 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := CompareReviewTiers(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareReviewTiers(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMaxReviewTier(t *testing.T) {
	tests := []struct {
		a, b string
		want string
	}{
		{ReviewAuto, ReviewScan, ReviewScan},
		{ReviewReview, ReviewScan, ReviewReview},
		{ReviewAuto, ReviewAuto, ReviewAuto},
		{ReviewDeep, ReviewAuto, ReviewDeep},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := MaxReviewTier(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("MaxReviewTier(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsValidReviewTier(t *testing.T) {
	tests := []struct {
		tier string
		want bool
	}{
		{ReviewAuto, true},
		{ReviewScan, true},
		{ReviewReview, true},
		{ReviewDeep, true},
		{"Auto", false}, // case sensitive
		{"", false},
		{"V1", false},
		{"full", false},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			got := IsValidReviewTier(tt.tier)
			if got != tt.want {
				t.Errorf("IsValidReviewTier(%q) = %v, want %v", tt.tier, got, tt.want)
			}
		})
	}
}

func TestCompareReviewTiers_UnknownDefaultsToReview(t *testing.T) {
	// Unknown tiers should be treated as review (conservative)
	got := CompareReviewTiers("unknown", ReviewReview)
	if got != 0 {
		t.Errorf("CompareReviewTiers(%q, %q) = %d, want 0 (unknown defaults to review)", "unknown", ReviewReview, got)
	}
}
