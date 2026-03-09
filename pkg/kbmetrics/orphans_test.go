package kbmetrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeOrphanRate_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	os.MkdirAll(filepath.Join(kbDir, "investigations"), 0755)

	report, err := ComputeOrphanRate(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 0 {
		t.Errorf("Total = %d, want 0", report.Total)
	}
	if report.OrphanRate != 0 {
		t.Errorf("OrphanRate = %f, want 0", report.OrphanRate)
	}
}

func TestComputeOrphanRate_AllOrphans(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(invDir, 0755)

	// Create 3 investigations with no references
	for _, name := range []string{"inv-a.md", "inv-b.md", "inv-c.md"} {
		os.WriteFile(filepath.Join(invDir, name), []byte("# Investigation\n"), 0644)
	}

	report, err := ComputeOrphanRate(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 3 {
		t.Errorf("Total = %d, want 3", report.Total)
	}
	if report.Orphaned != 3 {
		t.Errorf("Orphaned = %d, want 3", report.Orphaned)
	}
	if report.Connected != 0 {
		t.Errorf("Connected = %d, want 0", report.Connected)
	}
	if report.OrphanRate != 100.0 {
		t.Errorf("OrphanRate = %f, want 100.0", report.OrphanRate)
	}
}

func TestComputeOrphanRate_SomeConnected(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(modelDir, 0755)

	// Create 4 investigations
	os.WriteFile(filepath.Join(invDir, "inv-connected-a.md"), []byte("# Connected A\n"), 0644)
	os.WriteFile(filepath.Join(invDir, "inv-connected-b.md"), []byte("# Connected B\n"), 0644)
	os.WriteFile(filepath.Join(invDir, "inv-orphan-a.md"), []byte("# Orphan A\n"), 0644)
	os.WriteFile(filepath.Join(invDir, "inv-orphan-b.md"), []byte("# Orphan B\n"), 0644)

	// Model references 2 of them
	modelContent := `# Model: Test

## References

- .kb/investigations/inv-connected-a.md
- See also: .kb/investigations/inv-connected-b.md
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	report, err := ComputeOrphanRate(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 4 {
		t.Errorf("Total = %d, want 4", report.Total)
	}
	if report.Connected != 2 {
		t.Errorf("Connected = %d, want 2", report.Connected)
	}
	if report.Orphaned != 2 {
		t.Errorf("Orphaned = %d, want 2", report.Orphaned)
	}
	if report.OrphanRate != 50.0 {
		t.Errorf("OrphanRate = %f, want 50.0", report.OrphanRate)
	}
}

func TestComputeOrphanRate_ProbeReferences(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	probeDir := filepath.Join(kbDir, "models", "test-model", "probes")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(probeDir, 0755)

	// Create investigation
	os.WriteFile(filepath.Join(invDir, "inv-referenced.md"), []byte("# Referenced\n"), 0644)
	os.WriteFile(filepath.Join(invDir, "inv-orphan.md"), []byte("# Orphan\n"), 0644)

	// Probe references one investigation
	probeContent := `# Probe: Test
Based on findings from .kb/investigations/inv-referenced.md
`
	os.WriteFile(filepath.Join(probeDir, "probe-test.md"), []byte(probeContent), 0644)

	report, err := ComputeOrphanRate(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Connected != 1 {
		t.Errorf("Connected = %d, want 1", report.Connected)
	}
	if report.Orphaned != 1 {
		t.Errorf("Orphaned = %d, want 1", report.Orphaned)
	}
}

func TestComputeOrphanRate_SubdirectoryInvestigations(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	archivedDir := filepath.Join(invDir, "archived")
	os.MkdirAll(archivedDir, 0755)
	os.MkdirAll(filepath.Join(kbDir, "guides"), 0755)

	// Investigation in subdirectory
	os.WriteFile(filepath.Join(archivedDir, "inv-old.md"), []byte("# Old\n"), 0644)
	os.WriteFile(filepath.Join(invDir, "inv-new.md"), []byte("# New\n"), 0644)

	// Guide references the archived one
	guideContent := `# Guide
See .kb/investigations/archived/inv-old.md for details.
`
	os.WriteFile(filepath.Join(kbDir, "guides", "guide.md"), []byte(guideContent), 0644)

	report, err := ComputeOrphanRate(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 {
		t.Errorf("Total = %d, want 2", report.Total)
	}
	if report.Connected != 1 {
		t.Errorf("Connected = %d, want 1", report.Connected)
	}
}

func TestComputeOrphanRate_NoInvestigationsDir(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	os.MkdirAll(kbDir, 0755)
	// No investigations directory at all

	report, err := ComputeOrphanRate(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 0 {
		t.Errorf("Total = %d, want 0", report.Total)
	}
}

func TestComputeOrphanRate_DecisionReferences(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	decDir := filepath.Join(kbDir, "decisions")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(decDir, 0755)

	os.WriteFile(filepath.Join(invDir, "inv-connected.md"), []byte("# Connected\n"), 0644)

	// Decision references the investigation
	decContent := `# Decision
Based on .kb/investigations/inv-connected.md
`
	os.WriteFile(filepath.Join(decDir, "dec-test.md"), []byte(decContent), 0644)

	report, err := ComputeOrphanRate(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Connected != 1 {
		t.Errorf("Connected = %d, want 1", report.Connected)
	}
	if report.Orphaned != 0 {
		t.Errorf("Orphaned = %d, want 0", report.Orphaned)
	}
}

func TestOrphanReport_Summary(t *testing.T) {
	report := &OrphanReport{
		Total:      100,
		Connected:  40,
		Orphaned:   60,
		OrphanRate: 60.0,
	}
	summary := report.Summary()
	if summary == "" {
		t.Error("Summary() returned empty string")
	}
	if !containsSubstr(summary, "60.0%") {
		t.Errorf("Summary() = %q, want to contain '60.0%%'", summary)
	}
	if !containsSubstr(summary, "60/100") {
		t.Errorf("Summary() = %q, want to contain '60/100'", summary)
	}
}

func TestOrphanReport_Summary_Zero(t *testing.T) {
	report := &OrphanReport{}
	summary := report.Summary()
	if summary != "" {
		t.Errorf("Summary() = %q, want empty for zero total", summary)
	}
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
