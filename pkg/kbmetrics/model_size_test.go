package kbmetrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditModelSize_FlagsOversizedModels(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, "models")

	// Create a small model (under threshold)
	smallDir := filepath.Join(modelsDir, "small-model", "probes")
	os.MkdirAll(smallDir, 0o755)
	os.WriteFile(filepath.Join(modelsDir, "small-model", "model.md"),
		[]byte("# Small Model\n\nSmall content.\n"), 0o644)

	// Create a large model (over 30KB) with recent Last Updated
	bigDir := filepath.Join(modelsDir, "big-recent", "probes")
	os.MkdirAll(bigDir, 0o755)
	today := time.Now().Format("2006-01-02")
	bigContent := "# Big Recent\n\n**Last Updated:** " + today + "\n\n" + strings.Repeat("x", 35000)
	os.WriteFile(filepath.Join(modelsDir, "big-recent", "model.md"),
		[]byte(bigContent), 0o644)

	// Create a large model (over 30KB) with stale Last Updated (>2 weeks)
	staleDir := filepath.Join(modelsDir, "big-stale", "probes")
	os.MkdirAll(staleDir, 0o755)
	staleDate := time.Now().AddDate(0, 0, -20).Format("2006-01-02")
	staleContent := "# Big Stale\n\n**Last Updated:** " + staleDate + "\n\n" + strings.Repeat("y", 40000)
	os.WriteFile(filepath.Join(modelsDir, "big-stale", "model.md"),
		[]byte(staleContent), 0o644)

	// Create a large model with NO Last Updated (should flag — unknown = stale)
	noDateDir := filepath.Join(modelsDir, "big-nodate", "probes")
	os.MkdirAll(noDateDir, 0o755)
	noDateContent := "# Big No Date\n\n" + strings.Repeat("z", 32000)
	os.WriteFile(filepath.Join(modelsDir, "big-nodate", "model.md"),
		[]byte(noDateContent), 0o644)

	reports, err := AuditModelSize(dir, 30*1024, 14)
	if err != nil {
		t.Fatalf("AuditModelSize: %v", err)
	}

	if len(reports) == 0 {
		t.Fatal("expected at least one report")
	}

	// Find each model in the reports
	byName := map[string]ModelSizeReport{}
	for _, r := range reports {
		byName[r.Name] = r
	}

	// small-model should not appear (under threshold)
	if _, ok := byName["small-model"]; ok {
		t.Error("small-model should not appear in oversized report")
	}

	// big-recent: oversized but recently updated → not flagged for review
	if r, ok := byName["big-recent"]; !ok {
		t.Error("big-recent should appear in report")
	} else if r.NeedsReview {
		t.Error("big-recent was recently updated, should not need review")
	}

	// big-stale: oversized AND stale → flagged
	if r, ok := byName["big-stale"]; !ok {
		t.Error("big-stale should appear in report")
	} else {
		if !r.NeedsReview {
			t.Error("big-stale should need review (oversized + stale)")
		}
		if r.SizeBytes < 40000 {
			t.Errorf("big-stale size wrong: got %d", r.SizeBytes)
		}
	}

	// big-nodate: oversized AND no date (treat as stale) → flagged
	if r, ok := byName["big-nodate"]; !ok {
		t.Error("big-nodate should appear in report")
	} else if !r.NeedsReview {
		t.Error("big-nodate should need review (oversized + unknown date = stale)")
	}
}

func TestAuditModelSize_SortedBySizeDescending(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, "models")

	for _, m := range []struct {
		name string
		size int
	}{
		{"medium", 31000},
		{"large", 50000},
		{"huge", 70000},
	} {
		os.MkdirAll(filepath.Join(modelsDir, m.name, "probes"), 0o755)
		content := "# " + m.name + "\n\n" + strings.Repeat("x", m.size)
		os.WriteFile(filepath.Join(modelsDir, m.name, "model.md"),
			[]byte(content), 0o644)
	}

	reports, err := AuditModelSize(dir, 30*1024, 14)
	if err != nil {
		t.Fatal(err)
	}

	if len(reports) != 3 {
		t.Fatalf("expected 3 reports, got %d", len(reports))
	}

	if reports[0].SizeBytes < reports[1].SizeBytes || reports[1].SizeBytes < reports[2].SizeBytes {
		t.Errorf("reports not sorted by size descending: %d, %d, %d",
			reports[0].SizeBytes, reports[1].SizeBytes, reports[2].SizeBytes)
	}
}

func TestFormatModelSizeText(t *testing.T) {
	reports := []ModelSizeReport{
		{
			Name:        "big-stale",
			Path:        ".kb/models/big-stale/model.md",
			SizeBytes:   65000,
			SizeKB:      63.5,
			LastUpdated: "2026-02-20",
			DaysSince:   26,
			NeedsReview: true,
		},
		{
			Name:        "big-recent",
			Path:        ".kb/models/big-recent/model.md",
			SizeBytes:   35000,
			SizeKB:      34.2,
			LastUpdated: "2026-03-17",
			DaysSince:   1,
			NeedsReview: false,
		},
	}

	out := FormatModelSizeText(reports)

	if !strings.Contains(out, "Model Size Audit") {
		t.Error("missing header")
	}
	if !strings.Contains(out, "big-stale") {
		t.Error("missing big-stale")
	}
	if !strings.Contains(out, "REVIEW") {
		t.Error("missing REVIEW flag for stale model")
	}
	if !strings.Contains(out, "big-recent") {
		t.Error("missing big-recent")
	}
}

func TestAuditModelSize_NoModelsDir(t *testing.T) {
	dir := t.TempDir()
	reports, err := AuditModelSize(dir, 30*1024, 14)
	if err == nil {
		t.Fatal("expected error for missing models dir")
	}
	if reports != nil {
		t.Error("expected nil reports")
	}
}

func TestAuditModelSize_ChecksGlobalModels(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, "models")
	os.MkdirAll(modelsDir, 0o755)

	globalDir := filepath.Join(dir, "global", "models", "global-big", "probes")
	os.MkdirAll(globalDir, 0o755)
	content := "# Global Big\n\n" + strings.Repeat("g", 35000)
	os.WriteFile(filepath.Join(dir, "global", "models", "global-big", "model.md"),
		[]byte(content), 0o644)

	reports, err := AuditModelSize(dir, 30*1024, 14)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, r := range reports {
		if r.Name == "global-big" {
			found = true
		}
	}
	if !found {
		t.Error("expected global model to be included in audit")
	}
}
