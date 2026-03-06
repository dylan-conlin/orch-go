package verify

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestParseFrictionComments(t *testing.T) {
	tests := []struct {
		name     string
		comments []Comment
		want     []FrictionItem
	}{
		{
			name: "single bug friction",
			comments: []Comment{
				{Text: "Phase: Planning - starting work"},
				{Text: "Friction: bug: beads dir resolution fails from nested repos"},
				{Text: "Phase: Complete - done"},
			},
			want: []FrictionItem{
				{Category: "bug", Description: "beads dir resolution fails from nested repos"},
			},
		},
		{
			name: "multiple categories",
			comments: []Comment{
				{Text: "Friction: bug: git hook false positive"},
				{Text: "Friction: ceremony: 12 line fix took 30min due to process overhead"},
				{Text: "Friction: tooling: bd sync error noise"},
			},
			want: []FrictionItem{
				{Category: "bug", Description: "git hook false positive"},
				{Category: "ceremony", Description: "12 line fix took 30min due to process overhead"},
				{Category: "tooling", Description: "bd sync error noise"},
			},
		},
		{
			name: "friction none",
			comments: []Comment{
				{Text: "Friction: none"},
			},
			want: nil,
		},
		{
			name: "no friction comments",
			comments: []Comment{
				{Text: "Phase: Planning - starting"},
				{Text: "Phase: Complete - done"},
			},
			want: nil,
		},
		{
			name: "gap category",
			comments: []Comment{
				{Text: "Friction: gap: SYNTHESIS.md not gitignored"},
			},
			want: []FrictionItem{
				{Category: "gap", Description: "SYNTHESIS.md not gitignored"},
			},
		},
		{
			name: "ignores non-friction lines",
			comments: []Comment{
				{Text: "Phase: Planning - starting"},
				{Text: "Friction: bug: something broken"},
				{Text: "Found a performance issue"},
				{Text: "Friction: none"},  // This is after a real friction, but none means no additional
			},
			want: []FrictionItem{
				{Category: "bug", Description: "something broken"},
			},
		},
		{
			name: "category only no description",
			comments: []Comment{
				{Text: "Friction: tooling"},
			},
			want: []FrictionItem{
				{Category: "tooling", Description: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFrictionComments(tt.comments)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseFrictionComments() returned %d items, want %d", len(got), len(tt.want))
			}
			for i, item := range got {
				if item.Category != tt.want[i].Category {
					t.Errorf("item[%d].Category = %q, want %q", i, item.Category, tt.want[i].Category)
				}
				if item.Description != tt.want[i].Description {
					t.Errorf("item[%d].Description = %q, want %q", i, item.Description, tt.want[i].Description)
				}
			}
		})
	}
}

func TestFormatFrictionAdvisory(t *testing.T) {
	t.Run("multiple items grouped by category", func(t *testing.T) {
		items := []FrictionItem{
			{Category: "bug", Description: "beads dir resolution fails"},
			{Category: "bug", Description: "git hook false positive"},
			{Category: "ceremony", Description: "process overhead"},
			{Category: "tooling", Description: "bd sync noise"},
		}

		output := FormatFrictionAdvisory(items)

		if !strings.Contains(output, "Friction Report") {
			t.Error("expected output to contain 'Friction Report'")
		}
		if !strings.Contains(output, "bug") {
			t.Error("expected output to contain 'bug' category")
		}
		if !strings.Contains(output, "ceremony") {
			t.Error("expected output to contain 'ceremony' category")
		}
		if !strings.Contains(output, "beads dir resolution fails") {
			t.Error("expected output to contain bug description")
		}
	})

	t.Run("empty items returns empty", func(t *testing.T) {
		output := FormatFrictionAdvisory(nil)
		if output != "" {
			t.Errorf("expected empty output for nil items, got %q", output)
		}
	})

	t.Run("single item", func(t *testing.T) {
		items := []FrictionItem{
			{Category: "gap", Description: "missing capability"},
		}
		output := FormatFrictionAdvisory(items)
		if !strings.Contains(output, "gap") {
			t.Error("expected output to contain 'gap'")
		}
		if !strings.Contains(output, "missing capability") {
			t.Error("expected output to contain description")
		}
	})
}

func TestFetchAndParseFriction(t *testing.T) {
	// This tests the integration function that takes beadsID + projectDir
	// We can't test actual beads calls, but verify it handles empty comments
	t.Run("returns nil for empty beads ID", func(t *testing.T) {
		items := FetchAndParseFriction("", "")
		if items != nil {
			t.Error("expected nil for empty beads ID")
		}
	})
}

// Verify Comment type alias works with beads.Comment
func TestFrictionCommentTypeCompatibility(t *testing.T) {
	c := beads.Comment{
		Text: "Friction: bug: test",
	}
	comments := []Comment{c}
	items := ParseFrictionComments(comments)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Category != "bug" {
		t.Errorf("expected category 'bug', got %q", items[0].Category)
	}
}
