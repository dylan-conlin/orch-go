package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/dupdetect"
)

func TestFormatDuplicationAdvisory_EmptyPairs(t *testing.T) {
	result := dupdetect.FormatDuplicationAdvisory(nil)
	if result != "" {
		t.Errorf("expected empty string for nil pairs, got %q", result)
	}
}

func TestFindDuplicationInModifiedFiles_EmptyProjectDir(t *testing.T) {
	pairs := findDuplicationInModifiedFiles("", "")
	if len(pairs) != 0 {
		t.Errorf("expected nil for empty projectDir, got %d pairs", len(pairs))
	}
}

func TestCountDuplicationAdvisoryMatches_EmptyProjectDir(t *testing.T) {
	count := countDuplicationAdvisoryMatches("", "")
	if count != 0 {
		t.Errorf("expected 0 for empty projectDir, got %d", count)
	}
}
