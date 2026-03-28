package compose

import "testing"

func TestClassifyBriefCategory_BugsAreMaintenance(t *testing.T) {
	tests := []struct {
		name  string
		skill string
		title string
	}{
		{"bug with feature-impl skill", "feature-impl", "Login page broken"},
		{"bug with systematic-debugging skill", "systematic-debugging", "Fix auth crash"},
		{"bug with empty skill", "", "Something is broken"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyBriefCategory("bug", tt.skill, tt.title)
			if got != CategoryMaintenance {
				t.Errorf("ClassifyBriefCategory(bug, %q, %q) = %q, want %q",
					tt.skill, tt.title, got, CategoryMaintenance)
			}
		})
	}
}

func TestClassifyBriefCategory_DebuggingIsMaintenance(t *testing.T) {
	got := ClassifyBriefCategory("task", "systematic-debugging", "Investigate slow query")
	if got != CategoryMaintenance {
		t.Errorf("got %q, want %q", got, CategoryMaintenance)
	}
}

func TestClassifyBriefCategory_KnowledgeSkills(t *testing.T) {
	tests := []struct {
		skill string
	}{
		{"investigation"},
		{"research"},
		{"architect"},
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			got := ClassifyBriefCategory("task", tt.skill, "Design new API")
			if got != CategoryKnowledge {
				t.Errorf("ClassifyBriefCategory(task, %q, ...) = %q, want %q",
					tt.skill, got, CategoryKnowledge)
			}
		})
	}
}

func TestClassifyBriefCategory_KnowledgeIssueTypes(t *testing.T) {
	tests := []struct {
		issueType string
	}{
		{"investigation"},
		{"question"},
		{"experiment"},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			got := ClassifyBriefCategory(tt.issueType, "feature-impl", "Something")
			if got != CategoryKnowledge {
				t.Errorf("ClassifyBriefCategory(%q, feature-impl, ...) = %q, want %q",
					tt.issueType, got, CategoryKnowledge)
			}
		})
	}
}

func TestClassifyBriefCategory_FeatureImplMaintenanceTitleKeywords(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"fix broken auth middleware", CategoryMaintenance},
		{"Fix spawn gate test flakiness", CategoryMaintenance},
		{"add test coverage for compose", CategoryMaintenance},
		{"update config defaults", CategoryMaintenance},
		{"infra: add health check endpoint", CategoryMaintenance},
		{"cleanup stale workspace directories", CategoryMaintenance},
		{"lint fixes for new rules", CategoryMaintenance},
		{"wire compose into daemon periodic loop", CategoryMaintenance},
		{"plumb issue type through completion pipeline", CategoryMaintenance},
		{"migrate JSON storage to SQLite", CategoryMaintenance},
		{"update dep: bump cobra to v1.8", CategoryMaintenance},
		{"bump go version to 1.22", CategoryMaintenance},
		{"rename CompletionTarget fields", CategoryMaintenance},
		// Knowledge titles (no maintenance keywords)
		{"implement brief composition layer", CategoryKnowledge},
		{"add orch orient command", CategoryKnowledge},
		{"design daemon between-session composition", CategoryKnowledge},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := ClassifyBriefCategory("feature", "feature-impl", tt.title)
			if got != tt.expected {
				t.Errorf("ClassifyBriefCategory(feature, feature-impl, %q) = %q, want %q",
					tt.title, got, tt.expected)
			}
		})
	}
}

func TestClassifyBriefCategory_TaskRefactorKeywords(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"refactor completion pipeline", CategoryMaintenance},
		{"extract model_drift_reflection.go to pkg/modeldrift/", CategoryMaintenance},
		{"move spawn context to pkg/spawn", CategoryMaintenance},
		{"reorganize daemon config", CategoryMaintenance},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := ClassifyBriefCategory("task", "feature-impl", tt.title)
			if got != tt.expected {
				t.Errorf("ClassifyBriefCategory(task, feature-impl, %q) = %q, want %q",
					tt.title, got, tt.expected)
			}
		})
	}
}

func TestClassifyBriefCategory_TaskRefactorNotTriggeredForFeatureType(t *testing.T) {
	// Refactor keywords only apply when issue type is "task", not "feature"
	got := ClassifyBriefCategory("feature", "feature-impl", "refactor the auth system")
	if got != CategoryKnowledge {
		t.Errorf("got %q, want %q (refactor keyword should not match for feature type)", got, CategoryKnowledge)
	}
}

func TestClassifyBriefCategory_DefaultIsKnowledge(t *testing.T) {
	got := ClassifyBriefCategory("task", "feature-impl", "implement new dashboard widget")
	if got != CategoryKnowledge {
		t.Errorf("got %q, want %q", got, CategoryKnowledge)
	}
}

func TestClassifyBriefCategory_EmptyInputs(t *testing.T) {
	got := ClassifyBriefCategory("", "", "")
	if got != CategoryKnowledge {
		t.Errorf("empty inputs should default to knowledge, got %q", got)
	}
}
