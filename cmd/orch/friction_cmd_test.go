package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/friction"
)

func TestFindBeadsIssuesJSONL(t *testing.T) {
	// Create a temp directory with .beads/issues.jsonl
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "issues.jsonl"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save and restore working directory
	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	os.Chdir(dir)
	path := findBeadsIssuesJSONL()
	if path == "" {
		t.Error("expected to find .beads/issues.jsonl, got empty")
	}
}

func TestFormatFrictionText(t *testing.T) {
	report := &friction.Report{
		TotalIssues:   3,
		TotalComments: 14,
		FrictionCount: 4,
		NoneCount:     10,
		FrictionRate:  0.2857,
		Days:          7,
		Categories: []friction.CategoryCount{
			{Category: "tooling", Count: 2, Percentage: 50},
			{Category: "ceremony", Count: 1, Percentage: 25},
			{Category: "bug", Count: 1, Percentage: 25},
		},
		TopSources: []friction.Source{
			{Pattern: "governance hooks blocked valid action", Count: 2, Example: "hook blocked my edit"},
		},
		SkillRates: []friction.SkillRate{
			{Skill: "feat", Total: 8, FrictionCount: 3, Rate: 0.375},
			{Skill: "debug", Total: 4, FrictionCount: 1, Rate: 0.25},
		},
		WeeklyTrend: []friction.WeekBucket{
			{Week: "2026-W12", FrictionCount: 2},
			{Week: "2026-W13", FrictionCount: 2},
		},
	}

	output := formatFrictionText(report)

	// Check key sections exist
	checks := []string{
		"FRICTION REPORT",
		"last 7 days",
		"SUMMARY",
		"Friction rate:",
		"CATEGORIES",
		"tooling",
		"ceremony",
		"TOP RECURRING SOURCES",
		"governance hooks",
		"PER-SKILL FRICTION",
		"feat",
		"WEEKLY TREND",
		"2026-W12",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q", check)
		}
	}
}

func TestFormatFrictionTextAllTime(t *testing.T) {
	report := &friction.Report{
		Days: 0, // all time
	}
	output := formatFrictionText(report)
	if !strings.Contains(output, "all time") {
		t.Error("expected 'all time' for days=0")
	}
}
