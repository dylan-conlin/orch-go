package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStaleBugResult_IsPotentiallyStale(t *testing.T) {
	tests := []struct {
		name     string
		result   StaleBugResult
		expected bool
	}{
		{
			name: "no related commits",
			result: StaleBugResult{
				RelatedCommits: nil,
			},
			expected: false,
		},
		{
			name: "has related commits",
			result: StaleBugResult{
				RelatedCommits: []RelatedCommit{
					{Hash: "abc123", Subject: "fix: auth issue"},
				},
			},
			expected: true,
		},
		{
			name: "multiple related commits",
			result: StaleBugResult{
				RelatedCommits: []RelatedCommit{
					{Hash: "abc123", Subject: "fix: auth issue"},
					{Hash: "def456", Subject: "chore: cleanup auth"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsPotentiallyStale(); got != tt.expected {
				t.Errorf("IsPotentiallyStale() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCheckStaleBug(t *testing.T) {
	// Create a temp git repository for testing
	tempDir := t.TempDir()
	
	// Initialize git repo
	initGit := exec.Command("git", "init")
	initGit.Dir = tempDir
	if err := initGit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	
	// Configure git user for commits
	configName := exec.Command("git", "config", "user.name", "Test User")
	configName.Dir = tempDir
	_ = configName.Run()
	
	configEmail := exec.Command("git", "config", "user.email", "test@example.com")
	configEmail.Dir = tempDir
	_ = configEmail.Run()
	
	// Create initial commit
	if err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = tempDir
	_ = addCmd.Run()
	
	commitCmd := exec.Command("git", "commit", "-m", "initial commit")
	commitCmd.Dir = tempDir
	_ = commitCmd.Run()

	t.Run("no matching commits returns not stale", func(t *testing.T) {
		result, err := CheckStaleBug(tempDir, "test-xyz", "completely unrelated keywords", time.Now().Add(-24*time.Hour))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.IsPotentiallyStale() {
			t.Error("Expected not stale when no matching commits")
		}
	})

	t.Run("matching issue ID in commit message", func(t *testing.T) {
		// Create a commit that mentions the issue ID
		if err := os.WriteFile(filepath.Join(tempDir, "fix.txt"), []byte("fix"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		addCmd := exec.Command("git", "add", ".")
		addCmd.Dir = tempDir
		_ = addCmd.Run()
		
		commitCmd := exec.Command("git", "commit", "-m", "fix: resolve auth bug [issue-abc]")
		commitCmd.Dir = tempDir
		if err := commitCmd.Run(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
		
		// Check for issue-abc - should find the commit
		result, err := CheckStaleBug(tempDir, "issue-abc", "", time.Now().Add(-24*time.Hour))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.IsPotentiallyStale() {
			t.Error("Expected stale when issue ID found in commit")
		}
		if len(result.RelatedCommits) != 1 {
			t.Errorf("Expected 1 related commit, got %d", len(result.RelatedCommits))
		}
	})

	t.Run("matching keywords in commit message", func(t *testing.T) {
		// Create a commit with matching keywords
		if err := os.WriteFile(filepath.Join(tempDir, "auth.txt"), []byte("auth fix"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		addCmd := exec.Command("git", "add", ".")
		addCmd.Dir = tempDir
		_ = addCmd.Run()
		
		commitCmd := exec.Command("git", "commit", "-m", "fix: authentication middleware timeout")
		commitCmd.Dir = tempDir
		if err := commitCmd.Run(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
		
		// Check for keywords that match
		result, err := CheckStaleBug(tempDir, "different-id", "authentication timeout", time.Now().Add(-24*time.Hour))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.IsPotentiallyStale() {
			t.Error("Expected stale when keywords found in commit")
		}
	})

	t.Run("respects since time filter", func(t *testing.T) {
		// Check with a time after the commits - should find nothing
		result, err := CheckStaleBug(tempDir, "", "authentication", time.Now().Add(1*time.Hour))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.IsPotentiallyStale() {
			t.Error("Expected not stale when checking after commit time")
		}
	})

	t.Run("handles empty issue ID and keywords gracefully", func(t *testing.T) {
		result, err := CheckStaleBug(tempDir, "", "", time.Now().Add(-24*time.Hour))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		// With no issue ID and no keywords, should not match anything
		if result.IsPotentiallyStale() {
			t.Error("Expected not stale when no search criteria")
		}
	})
}

func TestExtractKeywordsFromTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected []string
	}{
		{
			name:     "simple title",
			title:    "Fix authentication bug",
			expected: []string{"authentication"}, // "Fix" and "bug" are filtered as common issue words
		},
		{
			name:     "title with stop words",
			title:    "The login is broken for some users",
			expected: []string{"login", "broken", "users"},
		},
		{
			name:     "title with short words",
			title:    "Fix UI in the application",
			expected: []string{"application"}, // "Fix", "UI", "in", "the" are filtered (too short or stop words)
		},
		{
			name:     "empty title",
			title:    "",
			expected: []string{},
		},
		{
			name:     "only stop words",
			title:    "the is a an",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKeywordsFromTitle(tt.title)
			if len(got) != len(tt.expected) {
				t.Errorf("ExtractKeywordsFromTitle() = %v, want %v", got, tt.expected)
				return
			}
			for i, keyword := range got {
				if keyword != tt.expected[i] {
					t.Errorf("Keyword[%d] = %s, want %s", i, keyword, tt.expected[i])
				}
			}
		})
	}
}

func TestFormatStaleBugWarning(t *testing.T) {
	t.Run("nil result returns empty", func(t *testing.T) {
		result := FormatStaleBugWarning(nil)
		if result != "" {
			t.Errorf("Expected empty string for nil result")
		}
	})

	t.Run("not stale returns empty", func(t *testing.T) {
		result := FormatStaleBugWarning(&StaleBugResult{
			RelatedCommits: nil,
		})
		if result != "" {
			t.Errorf("Expected empty string for non-stale result")
		}
	})

	t.Run("stale result shows warning", func(t *testing.T) {
		result := FormatStaleBugWarning(&StaleBugResult{
			IssueID: "test-123",
			RelatedCommits: []RelatedCommit{
				{Hash: "abc123", Subject: "fix: auth issue", Author: "Test User", Date: time.Now()},
			},
		})
		if result == "" {
			t.Error("Expected warning message for stale result")
		}
		// Should contain key elements
		if !containsString(result, "STALE BUG") && !containsString(result, "POTENTIALLY STALE") {
			t.Errorf("Expected warning to mention stale bug: %s", result)
		}
		if !containsString(result, "abc123") {
			t.Errorf("Expected warning to show commit hash: %s", result)
		}
	})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || containsString(s[1:], substr)))
}
