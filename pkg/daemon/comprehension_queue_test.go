package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// mockComprehensionQuerier is a test double for ComprehensionQuerier.
type mockComprehensionQuerier struct {
	count int
	err   error
}

func (m *mockComprehensionQuerier) CountPending() (int, error) {
	return m.count, m.err
}

func TestCheckComprehensionThrottle_NilQuerier(t *testing.T) {
	allowed, count, threshold := CheckComprehensionThrottle(nil, 5)
	if !allowed {
		t.Error("nil querier should allow spawning")
	}
	if count != 0 {
		t.Errorf("nil querier count = %d, want 0", count)
	}
	if threshold != 5 {
		t.Errorf("threshold = %d, want 5", threshold)
	}
}

func TestCheckComprehensionThrottle_BelowThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 3}
	allowed, count, threshold := CheckComprehensionThrottle(q, 5)
	if !allowed {
		t.Error("should allow when below threshold")
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if threshold != 5 {
		t.Errorf("threshold = %d, want 5", threshold)
	}
}

func TestCheckComprehensionThrottle_AtThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 5}
	allowed, _, _ := CheckComprehensionThrottle(q, 5)
	if allowed {
		t.Error("should block when at threshold")
	}
}

func TestCheckComprehensionThrottle_AboveThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 8}
	allowed, _, _ := CheckComprehensionThrottle(q, 5)
	if allowed {
		t.Error("should block when above threshold")
	}
}

func TestCheckComprehensionThrottle_ErrorFailsOpen(t *testing.T) {
	q := &mockComprehensionQuerier{count: 0, err: fmt.Errorf("bd failed")}
	allowed, _, _ := CheckComprehensionThrottle(q, 5)
	if !allowed {
		t.Error("should fail-open on error")
	}
}

func TestCheckComprehensionThrottle_DefaultThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 3}
	_, _, threshold := CheckComprehensionThrottle(q, 0)
	if threshold != DefaultComprehensionThreshold {
		t.Errorf("default threshold = %d, want %d", threshold, DefaultComprehensionThreshold)
	}
}

func TestComprehensionLabelConstants(t *testing.T) {
	if LabelComprehensionUnread != "comprehension:unread" {
		t.Errorf("LabelComprehensionUnread = %q, want %q", LabelComprehensionUnread, "comprehension:unread")
	}
	if LabelComprehensionProcessed != "comprehension:processed" {
		t.Errorf("LabelComprehensionProcessed = %q, want %q", LabelComprehensionProcessed, "comprehension:processed")
	}
	if LabelComprehensionPending != "comprehension:pending" {
		t.Errorf("LabelComprehensionPending = %q, want %q", LabelComprehensionPending, "comprehension:pending")
	}
}

func TestRecordBriefFeedback_ValidRatings(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(filepath.Join(projectDir, ".kb", "briefs"), 0755)

	// Test valid ratings
	for _, rating := range []string{"shallow", "good"} {
		err := RecordBriefFeedback("test-123", rating, projectDir)
		if err != nil {
			t.Errorf("RecordBriefFeedback(%q) failed: %v", rating, err)
		}

		got, err := ReadBriefFeedback("test-123", projectDir)
		if err != nil {
			t.Errorf("ReadBriefFeedback failed: %v", err)
		}
		if got != rating {
			t.Errorf("ReadBriefFeedback = %q, want %q", got, rating)
		}
	}
}

func TestRecordBriefFeedback_InvalidRating(t *testing.T) {
	tmpDir := t.TempDir()
	err := RecordBriefFeedback("test-123", "invalid", tmpDir)
	if err == nil {
		t.Error("expected error for invalid rating")
	}
}

func TestParseBriefSignalCount(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			"frontmatter with signals",
			"---\nbeads_id: test-123\nsignal_count: 5\nsignal_total: 6\n---\n\n# Brief",
			5,
		},
		{
			"zero signals",
			"---\nbeads_id: test-123\nsignal_count: 0\nsignal_total: 6\n---\n\n# Brief",
			0,
		},
		{
			"no frontmatter",
			"# Brief: test-123\n\n## Frame\n",
			0,
		},
		{
			"malformed frontmatter",
			"---\nno closing delimiter",
			0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseBriefSignalCount(tc.content)
			if got != tc.want {
				t.Errorf("ParseBriefSignalCount() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestReadBriefFeedback_NoFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	rating, err := ReadBriefFeedback("nonexistent", tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if rating != "" {
		t.Errorf("expected empty rating, got %q", rating)
	}
}
