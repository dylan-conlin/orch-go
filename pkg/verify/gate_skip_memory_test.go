package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadGateSkipMemory_NoFile(t *testing.T) {
	dir := t.TempDir()
	result := ReadGateSkipMemory(dir, GateBuild)
	if result != nil {
		t.Errorf("expected nil for missing file, got %+v", result)
	}
}

func TestWriteAndReadGateSkipMemory(t *testing.T) {
	dir := t.TempDir()

	err := WriteGateSkipMemory(dir, GateBuild, "concurrent agents broke the build", "orchestrator")
	if err != nil {
		t.Fatalf("WriteGateSkipMemory failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(dir, ".orch", GateSkipFilename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("gate skip file was not created")
	}

	skip := ReadGateSkipMemory(dir, GateBuild)
	if skip == nil {
		t.Fatal("ReadGateSkipMemory returned nil")
	}
	if skip.Gate != GateBuild {
		t.Errorf("Gate = %q, want %q", skip.Gate, GateBuild)
	}
	if skip.Reason != "concurrent agents broke the build" {
		t.Errorf("Reason = %q, want %q", skip.Reason, "concurrent agents broke the build")
	}
	if skip.SetBy != "orchestrator" {
		t.Errorf("SetBy = %q, want %q", skip.SetBy, "orchestrator")
	}
	if skip.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestReadGateSkipMemory_WrongGate(t *testing.T) {
	dir := t.TempDir()
	if err := WriteGateSkipMemory(dir, GateBuild, "build broken", "orchestrator"); err != nil {
		t.Fatal(err)
	}
	result := ReadGateSkipMemory(dir, GateDashboardHealth)
	if result != nil {
		t.Errorf("expected nil for different gate, got %+v", result)
	}
}

func TestWriteGateSkipMemory_MultipleGates(t *testing.T) {
	dir := t.TempDir()
	if err := WriteGateSkipMemory(dir, GateBuild, "build broken", "orchestrator"); err != nil {
		t.Fatal(err)
	}
	if err := WriteGateSkipMemory(dir, GateDashboardHealth, "not dashboard work", "orchestrator"); err != nil {
		t.Fatal(err)
	}

	build := ReadGateSkipMemory(dir, GateBuild)
	if build == nil || build.Reason != "build broken" {
		t.Errorf("expected build skip with reason 'build broken', got %+v", build)
	}
	dashboard := ReadGateSkipMemory(dir, GateDashboardHealth)
	if dashboard == nil || dashboard.Reason != "not dashboard work" {
		t.Errorf("expected dashboard skip with reason 'not dashboard work', got %+v", dashboard)
	}
}

func TestWriteGateSkipMemory_ReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	if err := WriteGateSkipMemory(dir, GateBuild, "first reason", "agent-1"); err != nil {
		t.Fatal(err)
	}
	if err := WriteGateSkipMemory(dir, GateBuild, "updated reason", "agent-2"); err != nil {
		t.Fatal(err)
	}

	skip := ReadGateSkipMemory(dir, GateBuild)
	if skip == nil {
		t.Fatal("expected skip to exist")
	}
	if skip.Reason != "updated reason" {
		t.Errorf("Reason = %q, want %q", skip.Reason, "updated reason")
	}
	if skip.SetBy != "agent-2" {
		t.Errorf("SetBy = %q, want %q", skip.SetBy, "agent-2")
	}

	// Should only have one entry for build
	count := 0
	for _, s := range ListGateSkipMemory(dir) {
		if s.Gate == GateBuild {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 build skip entry, got %d", count)
	}
}

func TestReadGateSkipMemory_Expired(t *testing.T) {
	dir := t.TempDir()
	orchDir := filepath.Join(dir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	file := GateSkipFile{
		Skips: []GateSkip{{
			Gate: GateBuild, Reason: "old failure",
			SetAt: time.Now().Add(-3 * time.Hour), SetBy: "old-agent",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}},
	}
	data, _ := json.MarshalIndent(file, "", "  ")
	os.WriteFile(filepath.Join(orchDir, GateSkipFilename), data, 0644)

	result := ReadGateSkipMemory(dir, GateBuild)
	if result != nil {
		t.Errorf("expected nil for expired entry, got %+v", result)
	}
}

func TestReadGateSkipMemory_MixedExpiry(t *testing.T) {
	dir := t.TempDir()
	orchDir := filepath.Join(dir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	file := GateSkipFile{
		Skips: []GateSkip{
			{Gate: GateBuild, Reason: "expired", SetAt: time.Now().Add(-3 * time.Hour),
				SetBy: "old-agent", ExpiresAt: time.Now().Add(-1 * time.Hour)},
			{Gate: GateDashboardHealth, Reason: "still active", SetAt: time.Now(),
				SetBy: "orchestrator", ExpiresAt: time.Now().Add(1 * time.Hour)},
		},
	}
	data, _ := json.MarshalIndent(file, "", "  ")
	os.WriteFile(filepath.Join(orchDir, GateSkipFilename), data, 0644)

	if result := ReadGateSkipMemory(dir, GateBuild); result != nil {
		t.Errorf("expected nil for expired build skip, got %+v", result)
	}
	result := ReadGateSkipMemory(dir, GateDashboardHealth)
	if result == nil || result.Reason != "still active" {
		t.Errorf("expected active dashboard skip, got %+v", result)
	}
}

func TestReadGateSkipMemory_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	orchDir := filepath.Join(dir, ".orch")
	os.MkdirAll(orchDir, 0755)
	os.WriteFile(filepath.Join(orchDir, GateSkipFilename), []byte("not json"), 0644)

	if result := ReadGateSkipMemory(dir, GateBuild); result != nil {
		t.Errorf("expected nil for invalid JSON, got %+v", result)
	}
}

func TestClearGateSkipMemory(t *testing.T) {
	dir := t.TempDir()
	WriteGateSkipMemory(dir, GateBuild, "build broken", "orchestrator")
	WriteGateSkipMemory(dir, GateDashboardHealth, "no dashboard", "orchestrator")

	if err := ClearGateSkipMemory(dir, GateBuild); err != nil {
		t.Fatal(err)
	}
	if skip := ReadGateSkipMemory(dir, GateBuild); skip != nil {
		t.Errorf("expected nil after clear, got %+v", skip)
	}
	if skip := ReadGateSkipMemory(dir, GateDashboardHealth); skip == nil {
		t.Fatal("expected dashboard skip to still exist")
	}
}

func TestClearAllGateSkipMemory(t *testing.T) {
	dir := t.TempDir()
	WriteGateSkipMemory(dir, GateBuild, "build broken", "orchestrator")
	WriteGateSkipMemory(dir, GateDashboardHealth, "no dashboard", "orchestrator")

	if err := ClearAllGateSkipMemory(dir); err != nil {
		t.Fatal(err)
	}
	if skip := ReadGateSkipMemory(dir, GateBuild); skip != nil {
		t.Errorf("expected nil after clear all, got %+v", skip)
	}
	if skip := ReadGateSkipMemory(dir, GateDashboardHealth); skip != nil {
		t.Errorf("expected nil after clear all, got %+v", skip)
	}
	if _, err := os.Stat(gateSkipPath(dir)); !os.IsNotExist(err) {
		t.Error("gate skip file should have been removed")
	}
}

func TestClearAllGateSkipMemory_NoFile(t *testing.T) {
	dir := t.TempDir()
	if err := ClearAllGateSkipMemory(dir); err != nil {
		t.Errorf("should not error for missing file: %v", err)
	}
}

func TestListGateSkipMemory(t *testing.T) {
	dir := t.TempDir()
	if list := ListGateSkipMemory(dir); len(list) != 0 {
		t.Errorf("expected empty list, got %d entries", len(list))
	}

	WriteGateSkipMemory(dir, GateBuild, "build broken", "orchestrator")
	WriteGateSkipMemory(dir, GateDashboardHealth, "no dashboard", "orchestrator")

	if list := ListGateSkipMemory(dir); len(list) != 2 {
		t.Errorf("expected 2 entries, got %d", len(list))
	}
}

func TestWriteGateSkipMemory_CreatesOrchDir(t *testing.T) {
	dir := t.TempDir()
	if err := WriteGateSkipMemory(dir, GateBuild, "test reason", "test-agent"); err != nil {
		t.Fatalf("WriteGateSkipMemory failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".orch")); os.IsNotExist(err) {
		t.Error(".orch directory should have been created")
	}
}

func TestGateSkipPath(t *testing.T) {
	path := gateSkipPath("/projects/orch-go")
	expected := filepath.Join("/projects/orch-go", ".orch", GateSkipFilename)
	if path != expected {
		t.Errorf("gateSkipPath = %q, want %q", path, expected)
	}
}
