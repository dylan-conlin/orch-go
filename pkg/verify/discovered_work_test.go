package verify

import (
	"bytes"
	"strings"
	"testing"
)

func TestCollectDiscoveredWork(t *testing.T) {
	tests := []struct {
		name    string
		synth   *Synthesis
		want    int // expected number of items
		wantSrc []string // expected sources (partial match ok)
	}{
		{
			name:  "nil synthesis",
			synth: nil,
			want:  0,
		},
		{
			name:  "empty synthesis",
			synth: &Synthesis{},
			want:  0,
		},
		{
			name: "only next actions",
			synth: &Synthesis{
				NextActions: []string{
					"- Deploy to staging",
					"- Run load tests",
				},
			},
			want:    2,
			wantSrc: []string{"Next Actions", "Next Actions"},
		},
		{
			name: "only areas to explore",
			synth: &Synthesis{
				AreasToExplore: []string{
					"- Performance optimization",
				},
			},
			want:    1,
			wantSrc: []string{"Areas to Explore"},
		},
		{
			name: "only uncertainties",
			synth: &Synthesis{
				Uncertainties: []string{
					"- Edge cases with empty input",
				},
			},
			want:    1,
			wantSrc: []string{"Uncertainties"},
		},
		{
			name: "all sources combined",
			synth: &Synthesis{
				NextActions: []string{
					"1. First action",
					"2. Second action",
				},
				AreasToExplore: []string{
					"- Area one",
				},
				Uncertainties: []string{
					"- Uncertainty one",
					"- Uncertainty two",
				},
			},
			want:    5,
			wantSrc: []string{"Next Actions", "Next Actions", "Areas to Explore", "Uncertainties", "Uncertainties"},
		},
		{
			name: "recommendation close - still collects items",
			synth: &Synthesis{
				Recommendation: "close",
				NextActions: []string{
					"- Optional follow-up",
				},
			},
			want:    1,
			wantSrc: []string{"Next Actions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollectDiscoveredWork(tt.synth)

			if len(got) != tt.want {
				t.Errorf("CollectDiscoveredWork() returned %d items, want %d", len(got), tt.want)
				for i, item := range got {
					t.Logf("  [%d] %s: %s", i, item.Source, item.Description)
				}
				return
			}

			// Verify sources match
			for i, wantSrc := range tt.wantSrc {
				if !strings.Contains(got[i].Source, wantSrc) {
					t.Errorf("item[%d].Source = %q, want to contain %q", i, got[i].Source, wantSrc)
				}
			}
		})
	}
}

func TestPromptDiscoveredWorkDisposition(t *testing.T) {
	t.Run("empty items list returns immediately", func(t *testing.T) {
		items := []DiscoveredWorkItem{}
		input := strings.NewReader("")
		output := &bytes.Buffer{}

		result, err := PromptDiscoveredWorkDisposition(items, input, output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Dispositions) != 0 {
			t.Errorf("expected empty dispositions, got %d", len(result.Dispositions))
		}
		if result.AllDispositioned {
			t.Error("expected AllDispositioned to be false for empty input (nothing to disposition)")
		}
	})

	t.Run("user says yes to each item", func(t *testing.T) {
		items := []DiscoveredWorkItem{
			{Description: "- Deploy to staging", Source: "Next Actions"},
			{Description: "- Run tests", Source: "Next Actions"},
		}
		input := strings.NewReader("y\ny\n")
		output := &bytes.Buffer{}

		result, err := PromptDiscoveredWorkDisposition(items, input, output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.AllDispositioned {
			t.Error("expected AllDispositioned to be true")
		}
		if len(result.Dispositions) != 2 {
			t.Errorf("expected 2 dispositions, got %d", len(result.Dispositions))
		}
		for i, d := range result.Dispositions {
			if d.Action != DispositionFileIssue {
				t.Errorf("disposition[%d].Action = %v, want DispositionFileIssue", i, d.Action)
			}
		}
	})

	t.Run("user skips individual items", func(t *testing.T) {
		items := []DiscoveredWorkItem{
			{Description: "- Item 1", Source: "Next Actions"},
			{Description: "- Item 2", Source: "Next Actions"},
		}
		input := strings.NewReader("n\nn\n")
		output := &bytes.Buffer{}

		result, err := PromptDiscoveredWorkDisposition(items, input, output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.AllDispositioned {
			t.Error("expected AllDispositioned to be true (items were skipped but still dispositioned)")
		}
		for i, d := range result.Dispositions {
			if d.Action != DispositionSkip {
				t.Errorf("disposition[%d].Action = %v, want DispositionSkip", i, d.Action)
			}
		}
	})

	t.Run("skip-all requires reason", func(t *testing.T) {
		items := []DiscoveredWorkItem{
			{Description: "- Item 1", Source: "Next Actions"},
			{Description: "- Item 2", Source: "Next Actions"},
			{Description: "- Item 3", Source: "Areas to Explore"},
		}
		// User types 's' for skip-all, then provides reason
		input := strings.NewReader("s\nalready tracked in another issue\n")
		output := &bytes.Buffer{}

		result, err := PromptDiscoveredWorkDisposition(items, input, output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.AllDispositioned {
			t.Error("expected AllDispositioned to be true")
		}
		if result.SkipAllReason != "already tracked in another issue" {
			t.Errorf("SkipAllReason = %q, want %q", result.SkipAllReason, "already tracked in another issue")
		}
		// All items should be marked as skip-all
		for i, d := range result.Dispositions {
			if d.Action != DispositionSkipAll {
				t.Errorf("disposition[%d].Action = %v, want DispositionSkipAll", i, d.Action)
			}
		}
	})

	t.Run("skip-all empty reason prompts again", func(t *testing.T) {
		items := []DiscoveredWorkItem{
			{Description: "- Item 1", Source: "Next Actions"},
		}
		// User types 's', then empty, then a real reason
		input := strings.NewReader("s\n\nactual reason\n")
		output := &bytes.Buffer{}

		result, err := PromptDiscoveredWorkDisposition(items, input, output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.SkipAllReason != "actual reason" {
			t.Errorf("SkipAllReason = %q, want %q", result.SkipAllReason, "actual reason")
		}
		// Output should contain the "reason required" message
		if !strings.Contains(output.String(), "reason required") {
			t.Error("expected output to contain 'reason required' message")
		}
	})

	t.Run("mixed responses", func(t *testing.T) {
		items := []DiscoveredWorkItem{
			{Description: "- Item 1", Source: "Next Actions"},
			{Description: "- Item 2", Source: "Areas to Explore"},
			{Description: "- Item 3", Source: "Uncertainties"},
		}
		// y, n, y
		input := strings.NewReader("y\nn\ny\n")
		output := &bytes.Buffer{}

		result, err := PromptDiscoveredWorkDisposition(items, input, output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.AllDispositioned {
			t.Error("expected AllDispositioned to be true")
		}
		if result.Dispositions[0].Action != DispositionFileIssue {
			t.Errorf("disposition[0] should be FileIssue, got %v", result.Dispositions[0].Action)
		}
		if result.Dispositions[1].Action != DispositionSkip {
			t.Errorf("disposition[1] should be Skip, got %v", result.Dispositions[1].Action)
		}
		if result.Dispositions[2].Action != DispositionFileIssue {
			t.Errorf("disposition[2] should be FileIssue, got %v", result.Dispositions[2].Action)
		}
	})

	t.Run("EOF before all items dispositioned", func(t *testing.T) {
		items := []DiscoveredWorkItem{
			{Description: "- Item 1", Source: "Next Actions"},
			{Description: "- Item 2", Source: "Next Actions"},
		}
		// Only provide one response, EOF after
		input := strings.NewReader("y\n")
		output := &bytes.Buffer{}

		result, err := PromptDiscoveredWorkDisposition(items, input, output)
		// Should return error because not all items dispositioned
		if err == nil {
			t.Fatal("expected error for incomplete disposition")
		}
		if result.AllDispositioned {
			t.Error("AllDispositioned should be false when EOF before completion")
		}
	})
}

func TestDiscoveredWorkResult_FiledItems(t *testing.T) {
	result := &DiscoveredWorkResult{
		AllDispositioned: true,
		Dispositions: []DiscoveredWorkDisposition{
			{Item: DiscoveredWorkItem{Description: "- Item 1"}, Action: DispositionFileIssue},
			{Item: DiscoveredWorkItem{Description: "- Item 2"}, Action: DispositionSkip},
			{Item: DiscoveredWorkItem{Description: "- Item 3"}, Action: DispositionFileIssue},
		},
	}

	filed := result.FiledItems()
	if len(filed) != 2 {
		t.Errorf("FiledItems() returned %d items, want 2", len(filed))
	}
}

func TestDiscoveredWorkResult_SkippedItems(t *testing.T) {
	result := &DiscoveredWorkResult{
		AllDispositioned: true,
		Dispositions: []DiscoveredWorkDisposition{
			{Item: DiscoveredWorkItem{Description: "- Item 1"}, Action: DispositionFileIssue},
			{Item: DiscoveredWorkItem{Description: "- Item 2"}, Action: DispositionSkip},
			{Item: DiscoveredWorkItem{Description: "- Item 3"}, Action: DispositionSkipAll},
		},
	}

	skipped := result.SkippedItems()
	if len(skipped) != 2 {
		t.Errorf("SkippedItems() returned %d items, want 2", len(skipped))
	}
}
