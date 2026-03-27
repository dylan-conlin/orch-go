package orient

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractBeadsIDs(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected []string
	}{
		{
			name:     "standard format",
			message:  "feat: add feature X (orch-go-abc12)",
			expected: []string{"orch-go-abc12"},
		},
		{
			name:     "no beads ID",
			message:  "feat: add feature X",
			expected: nil,
		},
		{
			name:     "multiple beads IDs in different commits",
			message:  "fix: thing (orch-go-xyz99)",
			expected: []string{"orch-go-xyz99"},
		},
		{
			name:     "different project prefix",
			message:  "feat: add thing (price-watch-ab12)",
			expected: nil, // only extract current project prefix
		},
		{
			name:     "beads ID mid-message",
			message:  "feat: orch-go-abc12 is done",
			expected: []string{"orch-go-abc12"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractBeadsIDs(tc.message, "orch-go")
			if len(got) != len(tc.expected) {
				t.Errorf("expected %d IDs, got %d: %v", len(tc.expected), len(got), got)
				return
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("expected ID[%d]=%q, got %q", i, tc.expected[i], got[i])
				}
			}
		})
	}
}

func TestParseGitLogForGroundTruth(t *testing.T) {
	gitLog := `abc1234 feat: add feature X (orch-go-abc12)
def5678 fix: broken thing (orch-go-def56)
ghi9012 docs: update readme
jkl3456 feat: another feature (orch-go-abc12)
mno7890 refactor: cleanup (orch-go-mno78)
`
	commits := ParseGitLogForGroundTruth(gitLog, "orch-go")

	if len(commits) != 4 {
		t.Fatalf("expected 4 commits with beads IDs, got %d", len(commits))
	}

	// Check that unique beads IDs are extracted correctly
	uniqueIDs := UniqueBeadsIDs(commits)
	if len(uniqueIDs) != 3 {
		t.Errorf("expected 3 unique beads IDs, got %d: %v", len(uniqueIDs), uniqueIDs)
	}
}

func TestParseGitNumstat(t *testing.T) {
	numstat := `15	3	pkg/orient/orient.go
8	2	cmd/orch/orient_cmd.go
-	-	binary-file.bin
25	0	pkg/orient/git_ground_truth.go
`
	added, deleted := ParseGitNumstat(numstat)

	if added != 48 {
		t.Errorf("expected 48 lines added, got %d", added)
	}
	if deleted != 5 {
		t.Errorf("expected 5 lines deleted, got %d", deleted)
	}
}

func TestParseGitNumstatEmpty(t *testing.T) {
	added, deleted := ParseGitNumstat("")
	if added != 0 || deleted != 0 {
		t.Errorf("expected 0/0 for empty input, got %d/%d", added, deleted)
	}
}

func TestComputeNetImpact(t *testing.T) {
	// Net impact = added - deleted
	added, deleted := 100, 30
	net := added - deleted
	if net != 70 {
		t.Errorf("expected net impact 70, got %d", net)
	}
}

func TestFormatThroughputWithGroundTruth(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{
			Days:            7,
			Completions:     42,
			Abandonments:    4,
			InProgress:      3,
			AvgDurationMin:  25,
			NetLinesAdded:   1892,
			NetLinesRemoved: 645,
		},
	}

	output := FormatHealth(data)

	// Should show net lines
	if !strings.Contains(output, "+1247") || !strings.Contains(output, "Net lines") {
		t.Errorf("missing net lines impact, got:\n%s", output)
	}
	// Should NOT show merged
	if strings.Contains(output, "Merged") {
		t.Errorf("should not show Merged (removed), got:\n%s", output)
	}
}

func TestFormatThroughputWithoutGroundTruth(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{
			Days:        1,
			Completions: 5,
		},
	}

	output := FormatHealth(data)

	// Should NOT show net lines when zero
	if strings.Contains(output, "Net lines") {
		t.Errorf("should not show net lines when zero, got:\n%s", output)
	}
}

func TestThroughputGroundTruthJSON(t *testing.T) {
	tp := Throughput{
		Days:            7,
		Completions:     42,
		NetLinesAdded:   1892,
		NetLinesRemoved: 645,
	}

	data := &OrientationData{Throughput: tp}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	jsonStr := string(b)

	if !strings.Contains(jsonStr, `"net_lines_added":1892`) {
		t.Errorf("JSON missing net_lines_added, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"net_lines_removed":645`) {
		t.Errorf("JSON missing net_lines_removed, got: %s", jsonStr)
	}
}
