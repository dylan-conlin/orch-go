package orient

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanModelFreshness(t *testing.T) {
	// Create a temp directory structure mimicking .kb/models/
	dir := t.TempDir()

	// Create a fresh model (updated today)
	freshDir := filepath.Join(dir, "fresh-model")
	os.MkdirAll(freshDir, 0755)
	today := time.Now().Format("2006-01-02")
	os.WriteFile(filepath.Join(freshDir, "model.md"), []byte(
		"# Model: Fresh Model\n\n**Domain:** Testing\n**Last Updated:** "+today+"\n\n---\n\n## Summary (30 seconds)\n\nThis is a fresh model.\n",
	), 0644)
	// Add a recent probe
	probesDir := filepath.Join(freshDir, "probes")
	os.MkdirAll(probesDir, 0755)
	os.WriteFile(filepath.Join(probesDir, "2026-02-27-probe-something.md"), []byte("probe"), 0644)

	// Create a stale model (updated 20 days ago, no recent probes)
	staleDir := filepath.Join(dir, "stale-model")
	os.MkdirAll(staleDir, 0755)
	staleDate := time.Now().AddDate(0, 0, -20).Format("2006-01-02")
	os.WriteFile(filepath.Join(staleDir, "model.md"), []byte(
		"# Model: Stale Model\n\n**Domain:** Testing\n**Last Updated:** "+staleDate+"\n\n---\n\n## Summary (30 seconds)\n\nThis model is stale.\n",
	), 0644)

	// Create a very stale model (updated 40 days ago, old probes)
	veryStaleDir := filepath.Join(dir, "very-stale-model")
	os.MkdirAll(veryStaleDir, 0755)
	veryStaleDate := time.Now().AddDate(0, 0, -40).Format("2006-01-02")
	os.WriteFile(filepath.Join(veryStaleDir, "model.md"), []byte(
		"# Model: Very Stale Model\n\n**Domain:** Testing\n**Last Updated:** "+veryStaleDate+"\n\n---\n\n## Summary (30 seconds)\n\nThis model is very stale with no recent probes.\n",
	), 0644)
	oldProbesDir := filepath.Join(veryStaleDir, "probes")
	os.MkdirAll(oldProbesDir, 0755)
	os.WriteFile(filepath.Join(oldProbesDir, "2025-12-01-probe-old.md"), []byte("old probe"), 0644)

	results, err := ScanModelFreshness(dir)
	if err != nil {
		t.Fatalf("ScanModelFreshness failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 models, got %d", len(results))
	}

	// Build lookup by name
	byName := make(map[string]ModelFreshness)
	for _, r := range results {
		byName[r.Name] = r
	}

	// Fresh model should not be stale
	fresh := byName["fresh-model"]
	if fresh.IsStale() {
		t.Errorf("fresh-model should not be stale, age=%d days", fresh.AgeDays)
	}

	// Stale model (20 days, no probes)
	stale := byName["stale-model"]
	if !stale.IsStale() {
		t.Errorf("stale-model should be stale (updated %d days ago, no probes)", stale.AgeDays)
	}

	// Very stale model
	veryStale := byName["very-stale-model"]
	if !veryStale.IsStale() {
		t.Errorf("very-stale-model should be stale (updated %d days ago)", veryStale.AgeDays)
	}
	if veryStale.HasRecentProbes {
		t.Errorf("very-stale-model should not have recent probes")
	}
}

func TestScanModelFreshness_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := ScanModelFreshness(dir)
	if err != nil {
		t.Fatalf("ScanModelFreshness failed on empty dir: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 models, got %d", len(results))
	}
}

func TestScanModelFreshness_MissingDir(t *testing.T) {
	results, err := ScanModelFreshness("/nonexistent/path")
	if err != nil {
		t.Fatalf("ScanModelFreshness should not error on missing dir: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 models, got %d", len(results))
	}
}

func TestExtractLastUpdated(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantDate string
		wantOK   bool
	}{
		{
			name:     "standard format",
			content:  "# Model\n\n**Last Updated:** 2026-02-27\n",
			wantDate: "2026-02-27",
			wantOK:   true,
		},
		{
			name:     "no date",
			content:  "# Model\n\nSome content\n",
			wantDate: "",
			wantOK:   false,
		},
		{
			name:     "different date",
			content:  "**Last Updated:** 2025-12-15\n",
			wantDate: "2025-12-15",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractLastUpdated(tt.content)
			if ok != tt.wantOK {
				t.Errorf("extractLastUpdated() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got.Format("2006-01-02") != tt.wantDate {
				t.Errorf("extractLastUpdated() = %v, want %v", got.Format("2006-01-02"), tt.wantDate)
			}
		})
	}
}

func TestExtractSummary(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "standard summary section",
			content: `# Model: Test

## Summary (30 seconds)

Agent state exists across four independent layers. These layers fall into two distinct categories.

---

## Core Mechanism`,
			want: "Agent state exists across four independent layers. These layers fall into two distinct categories.",
		},
		{
			name: "summary without time hint",
			content: `# Model: Test

## Summary

This is a brief summary of the model.

---`,
			want: "This is a brief summary of the model.",
		},
		{
			name:    "no summary section",
			content: "# Model\n\nSome content without summary.\n",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSummary(tt.content)
			if got != tt.want {
				t.Errorf("extractSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterStaleModels(t *testing.T) {
	models := []ModelFreshness{
		{Name: "fresh", AgeDays: 2, HasRecentProbes: true},
		{Name: "mildly-stale", AgeDays: 16, HasRecentProbes: false},
		{Name: "very-stale", AgeDays: 40, HasRecentProbes: false},
	}

	stale := FilterStaleModels(models, 2)
	if len(stale) != 2 {
		t.Fatalf("expected 2 stale models, got %d", len(stale))
	}
	// Should be sorted by age descending (stalest first)
	if stale[0].Name != "very-stale" {
		t.Errorf("expected very-stale first, got %s", stale[0].Name)
	}
	if stale[1].Name != "mildly-stale" {
		t.Errorf("expected mildly-stale second, got %s", stale[1].Name)
	}
}

func TestFilterStaleModels_Limit(t *testing.T) {
	models := []ModelFreshness{
		{Name: "a", AgeDays: 20, HasRecentProbes: false},
		{Name: "b", AgeDays: 30, HasRecentProbes: false},
		{Name: "c", AgeDays: 40, HasRecentProbes: false},
	}

	stale := FilterStaleModels(models, 1)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale model (limit), got %d", len(stale))
	}
	if stale[0].Name != "c" {
		t.Errorf("expected stalest model 'c', got %s", stale[0].Name)
	}
}

func TestHumanAge(t *testing.T) {
	tests := []struct {
		days int
		want string
	}{
		{0, "today"},
		{1, "1d ago"},
		{7, "7d ago"},
		{14, "14d ago"},
	}

	for _, tt := range tests {
		got := HumanAge(tt.days)
		if got != tt.want {
			t.Errorf("HumanAge(%d) = %q, want %q", tt.days, got, tt.want)
		}
	}
}
