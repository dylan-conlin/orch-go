package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseGitArtifactModifiedAt_UsesNewestCommitPerPath(t *testing.T) {
	output := strings.Join([]string{
		"__TS__1700000100",
		".kb/investigations/newest.md",
		".kb/decisions/shared.md",
		"",
		"__TS__1699990000",
		".kb/decisions/shared.md",
		".kb/models/older.md",
	}, "\n")

	result := parseGitArtifactModifiedAt(output)
	if len(result) != 3 {
		t.Fatalf("expected 3 parsed paths, got %d", len(result))
	}

	if result[".kb/decisions/shared.md"] != time.Unix(1700000100, 0).UTC() {
		t.Fatalf("expected newest commit timestamp for shared path, got %s", result[".kb/decisions/shared.md"])
	}
}

func TestSortArtifactsByRecency_NewestFirstAndPathTieBreak(t *testing.T) {
	now := time.Now()
	shared := now.Add(-2 * time.Hour)

	artifacts := []ArtifactFeedItem{
		{Path: ".kb/decisions/older.md", ModifiedAt: now.Add(-6 * time.Hour)},
		{Path: ".kb/decisions/zeta.md", ModifiedAt: shared},
		{Path: ".kb/decisions/newest.md", ModifiedAt: now.Add(-30 * time.Minute)},
		{Path: ".kb/decisions/alpha.md", ModifiedAt: shared},
	}

	sortArtifactsByRecency(artifacts)

	got := []string{artifacts[0].Path, artifacts[1].Path, artifacts[2].Path, artifacts[3].Path}
	want := []string{
		".kb/decisions/newest.md",
		".kb/decisions/alpha.md",
		".kb/decisions/zeta.md",
		".kb/decisions/older.md",
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected order at index %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestFilterRecent_PreservesRecencyOrder(t *testing.T) {
	now := time.Now()
	artifacts := []ArtifactFeedItem{
		{Path: "old.md", ModifiedAt: now.Add(-9 * 24 * time.Hour)},
		{Path: "recent-b.md", ModifiedAt: now.Add(-3 * time.Hour)},
		{Path: "recent-a.md", ModifiedAt: now.Add(-1 * time.Hour)},
	}

	sortArtifactsByRecency(artifacts)
	recent := filterRecent(artifacts, 7*24*time.Hour)

	if len(recent) != 2 {
		t.Fatalf("expected 2 recent artifacts, got %d", len(recent))
	}

	if recent[0].Path != "recent-a.md" || recent[1].Path != "recent-b.md" {
		t.Fatalf("unexpected filtered order: got [%s, %s]", recent[0].Path, recent[1].Path)
	}
}
