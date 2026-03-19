package kbmetrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeSessionOrphans_FiltersByDate(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(modelDir, 0755)

	// Old investigation (before cutoff)
	os.WriteFile(filepath.Join(invDir, "2026-03-10-old-inv.md"), []byte("# Old\n"), 0644)
	// Recent investigations (after cutoff)
	os.WriteFile(filepath.Join(invDir, "2026-03-18-new-orphan.md"), []byte("# New orphan\n"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-03-19-new-connected.md"), []byte("# New connected\n"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-03-19-new-orphan2.md"), []byte("# New orphan 2\n"), 0644)

	// Model references one recent investigation
	modelContent := "# Model\nSee .kb/investigations/2026-03-19-new-connected.md\n"
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	since := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	report, err := ComputeSessionOrphans(kbDir, since)
	if err != nil {
		t.Fatal(err)
	}
	if report.Investigations != 3 {
		t.Errorf("Investigations = %d, want 3", report.Investigations)
	}
	if report.Orphaned != 2 {
		t.Errorf("Orphaned = %d, want 2", report.Orphaned)
	}
}

func TestComputeSessionOrphans_NoRecentInvestigations(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)

	// Only old investigations
	os.WriteFile(filepath.Join(invDir, "2026-03-01-old.md"), []byte("# Old\n"), 0644)

	since := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	report, err := ComputeSessionOrphans(kbDir, since)
	if err != nil {
		t.Fatal(err)
	}
	if report.Investigations != 0 {
		t.Errorf("Investigations = %d, want 0", report.Investigations)
	}
	if report.Orphaned != 0 {
		t.Errorf("Orphaned = %d, want 0", report.Orphaned)
	}
}

func TestComputeSessionOrphans_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	os.MkdirAll(filepath.Join(kbDir, "investigations"), 0755)

	since := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	report, err := ComputeSessionOrphans(kbDir, since)
	if err != nil {
		t.Fatal(err)
	}
	if report.Investigations != 0 {
		t.Errorf("Investigations = %d, want 0", report.Investigations)
	}
}

func TestComputeSessionOrphans_AllConnected(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(modelDir, 0755)

	os.WriteFile(filepath.Join(invDir, "2026-03-19-inv-a.md"), []byte("# A\n"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-03-19-inv-b.md"), []byte("# B\n"), 0644)

	modelContent := "# Model\n- .kb/investigations/2026-03-19-inv-a.md\n- .kb/investigations/2026-03-19-inv-b.md\n"
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	since := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	report, err := ComputeSessionOrphans(kbDir, since)
	if err != nil {
		t.Fatal(err)
	}
	if report.Investigations != 2 {
		t.Errorf("Investigations = %d, want 2", report.Investigations)
	}
	if report.Orphaned != 0 {
		t.Errorf("Orphaned = %d, want 0", report.Orphaned)
	}
}
