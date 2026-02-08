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

func TestIsInfrastructureIntervention(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{SourceManualRecovery, true},
		{SourceDoctorFix, true},
		{SourceAgentAbandoned, false},
		{"unknown_source", true}, // fail-safe: unknown = infrastructure
	}

	for _, tt := range tests {
		got := isInfrastructureIntervention(tt.source)
		if got != tt.want {
			t.Errorf("isInfrastructureIntervention(%q) = %v, want %v", tt.source, got, tt.want)
		}
	}
}

func TestComputeReport_AgentAbandonedDoesNotResetStreak(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stability.jsonl")

	now := time.Now()

	// Write snapshot 2 hours ago
	healthy := true
	writeEntry(t, path, Entry{
		Type:     TypeSnapshot,
		Ts:       now.Add(-2 * time.Hour).Unix(),
		Healthy:  &healthy,
		Services: map[string]bool{"OpenCode": true},
	})

	// Write infrastructure intervention 1 hour ago (should reset streak)
	writeEntry(t, path, Entry{
		Type:   TypeIntervention,
		Ts:     now.Add(-1 * time.Hour).Unix(),
		Source: SourceManualRecovery,
		Detail: "OpenCode restarted manually",
	})

	// Write agent abandonment 30 minutes ago (should NOT reset streak)
	writeEntry(t, path, Entry{
		Type:    TypeIntervention,
		Ts:      now.Add(-30 * time.Minute).Unix(),
		Source:  SourceAgentAbandoned,
		Detail:  "orch-go-123 abandoned",
		BeadsID: "orch-go-123",
	})

	// Write snapshot 15 minutes ago (healthy)
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

	// Should show 2 interventions in the list (both types visible)
	if len(report.Interventions) != 2 {
		t.Errorf("expected 2 interventions in list, got %d", len(report.Interventions))
	}

	// Streak should be ~1 hour (since manual_recovery, NOT since agent_abandoned)
	if report.CurrentStreak < 59*time.Minute || report.CurrentStreak > 61*time.Minute {
		t.Errorf("expected streak ~1h (ignoring agent_abandoned), got %v", report.CurrentStreak)
	}

	// LastIntervention should point to manual_recovery (1h ago), not agent_abandoned (30m ago)
	if report.LastIntervention == nil {
		t.Fatal("expected LastIntervention to be set")
	}
	expectedLastIntervention := now.Add(-1 * time.Hour).Unix()
	if report.LastIntervention.Unix() < expectedLastIntervention-5 || report.LastIntervention.Unix() > expectedLastIntervention+5 {
		t.Errorf("expected LastIntervention ~1h ago, got %v", report.LastIntervention)
	}
}

func TestComputeReport_DoctorFixResetsStreak(t *testing.T) {
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

	// Write doctor_fix intervention 20 minutes ago (should reset streak)
	writeEntry(t, path, Entry{
		Type:   TypeIntervention,
		Ts:     now.Add(-20 * time.Minute).Unix(),
		Source: SourceDoctorFix,
		Detail: "Manual orch doctor --fix invoked",
	})

	report, err := ComputeReport(path, 7)
	if err != nil {
		t.Fatalf("ComputeReport failed: %v", err)
	}

	// Streak should be ~20 minutes (since doctor_fix)
	if report.CurrentStreak < 19*time.Minute || report.CurrentStreak > 21*time.Minute {
		t.Errorf("expected streak ~20m, got %v", report.CurrentStreak)
	}
}

func TestComputeReport_OnlyAgentAbandonedInterventions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stability.jsonl")

	now := time.Now()

	// Write snapshot 2 hours ago (first data point)
	healthy := true
	writeEntry(t, path, Entry{
		Type:     TypeSnapshot,
		Ts:       now.Add(-2 * time.Hour).Unix(),
		Healthy:  &healthy,
		Services: map[string]bool{"OpenCode": true},
	})

	// Write multiple agent abandonments (none should reset streak)
	writeEntry(t, path, Entry{
		Type:    TypeIntervention,
		Ts:      now.Add(-90 * time.Minute).Unix(),
		Source:  SourceAgentAbandoned,
		Detail:  "orch-go-123 abandoned",
		BeadsID: "orch-go-123",
	})
	writeEntry(t, path, Entry{
		Type:    TypeIntervention,
		Ts:      now.Add(-45 * time.Minute).Unix(),
		Source:  SourceAgentAbandoned,
		Detail:  "orch-go-456 abandoned",
		BeadsID: "orch-go-456",
	})

	report, err := ComputeReport(path, 7)
	if err != nil {
		t.Fatalf("ComputeReport failed: %v", err)
	}

	// Should show 2 interventions in the list
	if len(report.Interventions) != 2 {
		t.Errorf("expected 2 interventions in list, got %d", len(report.Interventions))
	}

	// Streak should be ~2 hours (since first snapshot, no infrastructure interventions)
	if report.CurrentStreak < 119*time.Minute {
		t.Errorf("expected streak >= 119m (no infrastructure interventions), got %v", report.CurrentStreak)
	}

	// LastIntervention should be nil (no infrastructure interventions)
	if report.LastIntervention != nil {
		t.Errorf("expected LastIntervention=nil when only agent abandonments exist, got %v", report.LastIntervention)
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
