package main

import (
	"strings"
	"testing"
)

func TestRejectCmd_InvalidCategory(t *testing.T) {
	err := runReject("fake-123", "bad work", "invalid-category", "")
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
	if !strings.Contains(err.Error(), "invalid category") {
		t.Errorf("expected 'invalid category' in error, got: %s", err.Error())
	}
}

func TestRejectCmd_ValidCategories(t *testing.T) {
	categories := []string{"quality", "scope", "approach", "stale"}
	for _, cat := range categories {
		// These will fail at the project resolution phase (no real beads),
		// but they should NOT fail at category validation.
		err := runReject("fake-123", "reason", cat, "")
		if err == nil {
			continue
		}
		if strings.Contains(err.Error(), "invalid category") {
			t.Errorf("category %q was rejected as invalid", cat)
		}
	}
}

func TestRejectCmd_RequiresArgs(t *testing.T) {
	cmd := rejectCmd
	if cmd.Args == nil {
		t.Fatal("expected Args validation on reject command")
	}
}

func TestRejectCmd_DefaultCategory(t *testing.T) {
	flag := rejectCmd.Flags().Lookup("category")
	if flag == nil {
		t.Fatal("expected --category flag")
	}
	if flag.DefValue != "quality" {
		t.Errorf("expected default category 'quality', got %q", flag.DefValue)
	}
}
