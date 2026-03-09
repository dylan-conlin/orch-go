package entropy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveReport(t *testing.T) {
	dir := t.TempDir()

	report := &Report{
		GeneratedAt: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
		WindowDays:  28,
		ProjectDir:  "/tmp/test",
		HealthLevel: "healthy",
	}

	path, err := SaveReport(report, dir)
	if err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	// Verify filename format: entropy-YYYY-MM-DD.json
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "entropy-2026-03-08") {
		t.Errorf("unexpected filename: %s", base)
	}
	if !strings.HasSuffix(base, ".json") {
		t.Errorf("expected .json suffix: %s", base)
	}

	// Verify content is valid JSON with correct data
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var loaded Report
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded.HealthLevel != "healthy" {
		t.Errorf("health_level = %q, want %q", loaded.HealthLevel, "healthy")
	}
	if loaded.WindowDays != 28 {
		t.Errorf("window_days = %d, want 28", loaded.WindowDays)
	}
}

func TestSaveReport_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "entropy")

	report := &Report{
		GeneratedAt: time.Now(),
		HealthLevel: "degrading",
	}

	path, err := SaveReport(report, dir)
	if err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}
