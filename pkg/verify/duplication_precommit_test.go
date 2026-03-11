package verify

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/dupdetect"
)

func TestFormatStagedDuplicationWarning_NilResult(t *testing.T) {
	result := FormatStagedDuplicationWarning(nil)
	if result != "" {
		t.Errorf("expected empty string for nil result, got %q", result)
	}
}

func TestFormatStagedDuplicationWarning_NoPairs(t *testing.T) {
	result := FormatStagedDuplicationWarning(&DuplicationPrecommitResult{
		Passed: true,
	})
	if result != "" {
		t.Errorf("expected empty string for no warnings, got %q", result)
	}
}

func TestFormatStagedDuplicationWarning_WithPairs(t *testing.T) {
	dr := &DuplicationPrecommitResult{
		Passed: true,
		Warnings: []dupdetect.DupPair{
			{
				FuncA:      dupdetect.FuncInfo{Name: "processItems", File: "pkg/a.go", StartLine: 10, Lines: 15},
				FuncB:      dupdetect.FuncInfo{Name: "handleEntries", File: "pkg/b.go", StartLine: 5, Lines: 15},
				Similarity: 0.92,
			},
		},
	}

	result := FormatStagedDuplicationWarning(dr)
	if result == "" {
		t.Fatal("expected non-empty warning")
	}

	for _, want := range []string{"WARNING", "processItems", "handleEntries", "92%", "orch dupdetect"} {
		if !strings.Contains(result, want) {
			t.Errorf("warning missing %q", want)
		}
	}
}

func TestCheckStagedDuplication_EmptyDir(t *testing.T) {
	result := CheckStagedDuplication("")
	if result != nil {
		t.Error("expected nil for empty dir")
	}
}
