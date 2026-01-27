package verify

import (
	"testing"
)

func TestHasBehavioralEvidence(t *testing.T) {
	tests := []struct {
		name         string
		comments     []Comment
		wantHas      bool
		wantEvidence int // expected number of evidence items
	}{
		{
			name:         "no comments",
			comments:     nil,
			wantHas:      false,
			wantEvidence: 0,
		},
		{
			name: "vague claim without evidence",
			comments: []Comment{
				{Text: "tests pass"},
				{Text: "all done"},
			},
			wantHas:      false,
			wantEvidence: 0,
		},
		{
			name: "behavior verified pattern",
			comments: []Comment{
				{Text: "Behavior verified: clicked button → modal opened"},
			},
			wantHas:      true,
			wantEvidence: 1,
		},
		{
			name: "ran locally pattern",
			comments: []Comment{
				{Text: "Ran locally: started scraper → second instance blocked"},
			},
			wantHas:      true,
			wantEvidence: 1,
		},
		{
			name: "visual verification pattern",
			comments: []Comment{
				{Text: "Visual verification: dashboard shows updated stats"},
			},
			wantHas:      true,
			wantEvidence: 1,
		},
		{
			name: "curl test pattern",
			comments: []Comment{
				{Text: "curl test: GET /api/health returns 200"},
			},
			wantHas:      true,
			wantEvidence: 1,
		},
		{
			name: "multiple evidence patterns",
			comments: []Comment{
				{Text: "Behavior verified: form submits correctly"},
				{Text: "Visual verification: success message displayed"},
			},
			wantHas:      true,
			wantEvidence: 2,
		},
		{
			name: "smoke test pattern",
			comments: []Comment{
				{Text: "Smoke test: full workflow completes without errors"},
			},
			wantHas:      true,
			wantEvidence: 1,
		},
		{
			name: "lock verification pattern",
			comments: []Comment{
				{Text: "Lock acquired successfully, second process waited"},
			},
			wantHas:      true,
			wantEvidence: 1,
		},
		{
			name: "redis verification pattern",
			comments: []Comment{
				{Text: "Redis connected and working properly"},
			},
			wantHas:      true,
			wantEvidence: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasEvidence, evidence := HasBehavioralEvidence(tt.comments)
			if hasEvidence != tt.wantHas {
				t.Errorf("HasBehavioralEvidence() hasEvidence = %v, want %v", hasEvidence, tt.wantHas)
			}
			if len(evidence) != tt.wantEvidence {
				t.Errorf("HasBehavioralEvidence() len(evidence) = %d, want %d", len(evidence), tt.wantEvidence)
			}
		})
	}
}

func TestDetectBehaviorChangeType(t *testing.T) {
	tests := []struct {
		name            string
		files           []string
		wantType        BehavioralValidationType
		wantMatchCount  int // minimum number of matched files
	}{
		{
			name:            "no files",
			files:           nil,
			wantType:        "",
			wantMatchCount:  0,
		},
		{
			name:            "markdown only",
			files:           []string{"README.md", "docs/guide.md"},
			wantType:        "",
			wantMatchCount:  0,
		},
		{
			name:            "UI svelte files",
			files:           []string{"src/components/Button.svelte", "src/pages/Home.svelte"},
			wantType:        BehavioralTypeUI,
			wantMatchCount:  2,
		},
		{
			name:            "UI web directory",
			files:           []string{"web/index.html", "web/app.js"},
			wantType:        BehavioralTypeUI,
			wantMatchCount:  2,
		},
		{
			name:            "API routes",
			files:           []string{"api/users.go", "routes/auth.go"},
			wantType:        BehavioralTypeAPI,
			wantMatchCount:  2,
		},
		{
			name:            "concurrency lock files",
			files:           []string{"pkg/lock/redis_lock.go", "internal/mutex.go"},
			wantType:        BehavioralTypeConcurrency,
			wantMatchCount:  2,
		},
		{
			name:            "integration redis",
			files:           []string{"pkg/redis/client.go", "internal/redis_cache.go"},
			wantType:        BehavioralTypeIntegration,
			wantMatchCount:  2,
		},
		{
			name:            "CLI commands",
			files:           []string{"cmd/orch/main.go", "cmd/orch/spawn.go"},
			wantType:        BehavioralTypeCLI,
			wantMatchCount:  2,
		},
		{
			name:            "mixed - UI takes priority",
			files:           []string{"web/app.js", "cmd/orch/main.go"},
			wantType:        BehavioralTypeUI,
			wantMatchCount:  1,
		},
		{
			name:            "tsx files are UI",
			files:           []string{"src/App.tsx", "src/components/Header.tsx"},
			wantType:        BehavioralTypeUI,
			wantMatchCount:  2,
		},
		{
			name:            "database files",
			files:           []string{"db/migrations/001.sql", "pkg/database/conn.go"},
			wantType:        BehavioralTypeIntegration,
			wantMatchCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotFiles := DetectBehaviorChangeType(tt.files)
			if gotType != tt.wantType {
				t.Errorf("DetectBehaviorChangeType() type = %v, want %v", gotType, tt.wantType)
			}
			if len(gotFiles) < tt.wantMatchCount {
				t.Errorf("DetectBehaviorChangeType() matched files = %d, want >= %d", len(gotFiles), tt.wantMatchCount)
			}
		})
	}
}

func TestHasBehaviorChangeCommitPattern(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
		want     bool
	}{
		{
			name:     "no messages",
			messages: nil,
			want:     false,
		},
		{
			name:     "docs only",
			messages: []string{"docs: update readme", "chore: update deps"},
			want:     false,
		},
		{
			name:     "feat commit",
			messages: []string{"feat: add user authentication"},
			want:     true,
		},
		{
			name:     "fix commit",
			messages: []string{"fix: resolve race condition"},
			want:     true,
		},
		{
			name:     "refactor commit",
			messages: []string{"refactor(api): restructure handlers"},
			want:     true,
		},
		{
			name:     "add endpoint",
			messages: []string{"added new endpoint for user data"},
			want:     true,
		},
		{
			name:     "implement lock",
			messages: []string{"implement distributed lock for concurrency"},
			want:     true,
		},
		{
			name:     "ui change",
			messages: []string{"ui update for dashboard"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasBehaviorChangeCommitPattern(tt.messages)
			if got != tt.want {
				t.Errorf("HasBehaviorChangeCommitPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSuggestedValidationSteps(t *testing.T) {
	tests := []struct {
		name       string
		valType    BehavioralValidationType
		wantSteps  bool // should have steps
		wantMinLen int  // minimum number of steps
	}{
		{
			name:       "empty type",
			valType:    "",
			wantSteps:  false,
			wantMinLen: 0,
		},
		{
			name:       "UI type",
			valType:    BehavioralTypeUI,
			wantSteps:  true,
			wantMinLen: 3,
		},
		{
			name:       "API type",
			valType:    BehavioralTypeAPI,
			wantSteps:  true,
			wantMinLen: 3,
		},
		{
			name:       "concurrency type",
			valType:    BehavioralTypeConcurrency,
			wantSteps:  true,
			wantMinLen: 3,
		},
		{
			name:       "integration type",
			valType:    BehavioralTypeIntegration,
			wantSteps:  true,
			wantMinLen: 3,
		},
		{
			name:       "CLI type",
			valType:    BehavioralTypeCLI,
			wantSteps:  true,
			wantMinLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := GetSuggestedValidationSteps(tt.valType)
			hasSteps := len(steps) > 0
			if hasSteps != tt.wantSteps {
				t.Errorf("GetSuggestedValidationSteps() hasSteps = %v, want %v", hasSteps, tt.wantSteps)
			}
			if len(steps) < tt.wantMinLen {
				t.Errorf("GetSuggestedValidationSteps() len(steps) = %d, want >= %d", len(steps), tt.wantMinLen)
			}
		})
	}
}

func TestCheckBehavioralValidationWithComments_NoChanges(t *testing.T) {
	// Test with empty workspace and project dir - should return empty result
	result := CheckBehavioralValidationWithComments("test-123", "", "", nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.BehavioralValidationSuggested {
		t.Error("expected BehavioralValidationSuggested to be false when no changes")
	}
}

func TestCheckBehavioralValidationForCompletion_NotSuggested(t *testing.T) {
	// When not suggested, should return nil (to match pattern of other completion checks)
	result := CheckBehavioralValidationForCompletion("test-123", "", "", nil)
	if result != nil {
		t.Error("expected nil when behavioral validation not suggested")
	}
}
