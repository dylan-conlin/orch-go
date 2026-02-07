package stability

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stability.jsonl")
	r := NewRecorder(path)

	services := map[string]bool{
		"OpenCode":   true,
		"orch serve": true,
	}

	if err := r.RecordSnapshot(true, services); err != nil {
		t.Fatalf("RecordSnapshot failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if entry.Type != TypeSnapshot {
		t.Errorf("expected type %q, got %q", TypeSnapshot, entry.Type)
	}
	if entry.Healthy == nil || !*entry.Healthy {
		t.Error("expected healthy=true")
	}
	if !entry.Services["OpenCode"] {
		t.Error("expected OpenCode service to be true")
	}
}

func TestRecordIntervention(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stability.jsonl")
	r := NewRecorder(path)

	if err := r.RecordIntervention(SourceManualRecovery, "OpenCode restarted manually", []string{"OpenCode"}, ""); err != nil {
		t.Fatalf("RecordIntervention failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if entry.Type != TypeIntervention {
		t.Errorf("expected type %q, got %q", TypeIntervention, entry.Type)
	}
	if entry.Source != SourceManualRecovery {
		t.Errorf("expected source %q, got %q", SourceManualRecovery, entry.Source)
	}
}

func TestComputeReport_NoFile(t *testing.T) {
	report, err := ComputeReport("/nonexistent/path.jsonl", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.HasData {
		t.Error("expected HasData=false for missing file")
	}
}

func TestComputeReport_SnapshotsOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stability.jsonl")
	r := NewRecorder(path)

	now := time.Now()

	// Write some snapshots
	for i := 0; i < 10; i++ {
		healthy := true
		entry := Entry{
			Type:     TypeSnapshot,
			Ts:       now.Add(-time.Duration(10-i) * time.Minute).Unix(),
			Healthy:  &healthy,
			Services: map[string]bool{"OpenCode": true},
		}
		writeEntry(t, path, entry)
	}
	_ = r // recorder used indirectly via writeEntry

	report, err := ComputeReport(path, 7)
	if err != nil {
		t.Fatalf("ComputeReport failed: %v", err)
	}

	if !report.HasData {
		t.Error("expected HasData=true")
	}
	if report.SnapshotsTotal != 10 {
		t.Errorf("expected 10 snapshots, got %d", report.SnapshotsTotal)
	}
	if report.SnapshotsHealthy != 10 {
		t.Errorf("expected 10 healthy snapshots, got %d", report.SnapshotsHealthy)
	}
	if report.HealthPercent != 100 {
		t.Errorf("expected 100%% health, got %.1f%%", report.HealthPercent)
	}
	if len(report.Interventions) != 0 {
		t.Errorf("expected 0 interventions, got %d", len(report.Interventions))
	}
	// Streak should be ~10 minutes (since first snapshot, no interventions)
	if report.CurrentStreak < 9*time.Minute {
		t.Errorf("expected streak >= 9m, got %v", report.CurrentStreak)
	}
}

func TestComputeReport_WithIntervention(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stability.jsonl")

	now := time.Now()

	// Write snapshot 1 hour ago
	healthy := true
	writeEntry(t, path, Entry{
		Type:     TypeSnapshot,
		Ts:       now.Add(-1 * time.Hour).Unix(),
		Healthy:  &healthy,
		Services: map[string]bool{"OpenCode": true},
	})

	// Write intervention 30 minutes ago
	writeEntry(t, path, Entry{
		Type:   TypeIntervention,
		Ts:     now.Add(-30 * time.Minute).Unix(),
		Source: SourceManualRecovery,
		Detail: "OpenCode restarted manually",
	})

	// Write snapshot 15 minutes ago (healthy again)
	writeEntry(t, path, Entry{
		Type:     TypeSnapshot,
		Ts:       now.Add(-15 * time.Minute).Unix(),
		Healthy:  &healthy,
		Services: map[string]bool{"OpenCode": true},
	})

	report, err := ComputeReport(path, 7)
	if err != nil {
		t.Fatalf("ComputeReport failed: %v", err)
	}

	if len(report.Interventions) != 1 {
		t.Fatalf("expected 1 intervention, got %d", len(report.Interventions))
	}

	// Streak should be ~30 minutes (since the intervention)
	if report.CurrentStreak < 29*time.Minute || report.CurrentStreak > 31*time.Minute {
		t.Errorf("expected streak ~30m, got %v", report.CurrentStreak)
	}

	// Progress should be small (30min / 7days ≈ 0.3%)
	if report.ProgressPercent > 1 {
		t.Errorf("expected progress < 1%%, got %.1f%%", report.ProgressPercent)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0m"},
		{5 * time.Minute, "5m"},
		{2*time.Hour + 30*time.Minute, "2h 30m"},
		{3*24*time.Hour + 14*time.Hour + 22*time.Minute, "3d 14h 22m"},
		{7 * 24 * time.Hour, "7d 0h 0m"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.d)
		if got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		percent float64
		width   int
		want    string
	}{
		{0, 20, "--------------------"},
		{50, 20, "##########----------"},
		{100, 20, "####################"},
		{120, 20, "####################"}, // capped at 100%
	}

	for _, tt := range tests {
		got := ProgressBar(tt.percent, tt.width)
		if got != tt.want {
			t.Errorf("ProgressBar(%.0f%%, %d) = %q, want %q", tt.percent, tt.width, got, tt.want)
		}
	}
}

func writeEntry(t *testing.T, path string, entry Entry) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	f.Write(append(data, '\n'))
}
