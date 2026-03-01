package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestCollectCompletionTelemetry_DurationFromManifest(t *testing.T) {
	tmpDir := t.TempDir()

	// Write AGENT_MANIFEST.json with spawn time 30 minutes ago
	spawnTime := time.Now().Add(-30 * time.Minute)
	manifest := spawn.AgentManifest{
		WorkspaceName: "og-test-agent",
		Skill:         "feature-impl",
		BeadsID:       "orch-go-test1",
		ProjectDir:    tmpDir,
		SpawnTime:     spawnTime.Format(time.RFC3339),
		Tier:          "full",
		SpawnMode:     "claude",
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENT_MANIFEST.json"), data, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	durationSecs, _, _, outcome := collectCompletionTelemetry(tmpDir, false, true)

	// Duration should be approximately 30 minutes (1800 seconds), allow 60s tolerance
	if durationSecs < 1740 || durationSecs > 1860 {
		t.Errorf("expected duration ~1800s (30 min), got %d", durationSecs)
	}
	if outcome != "success" {
		t.Errorf("expected outcome 'success', got %q", outcome)
	}
}

func TestCollectCompletionTelemetry_DurationFromDotfile(t *testing.T) {
	tmpDir := t.TempDir()

	// Write .spawn_time dotfile (legacy format: Unix nanos)
	spawnTime := time.Now().Add(-45 * time.Minute)
	nanos := spawnTime.UnixNano()
	if err := os.WriteFile(filepath.Join(tmpDir, ".spawn_time"), []byte(fmt.Sprintf("%d", nanos)), 0644); err != nil {
		t.Fatalf("failed to write spawn_time: %v", err)
	}
	// Write .tier to satisfy manifest reading
	if err := os.WriteFile(filepath.Join(tmpDir, ".tier"), []byte("full"), 0644); err != nil {
		t.Fatalf("failed to write tier: %v", err)
	}

	durationSecs, _, _, _ := collectCompletionTelemetry(tmpDir, false, true)

	// Duration should be approximately 45 minutes (2700 seconds), allow 60s tolerance
	if durationSecs < 2640 || durationSecs > 2760 {
		t.Errorf("expected duration ~2700s (45 min), got %d", durationSecs)
	}
}

func TestCollectCompletionTelemetry_NoManifest(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty directory - no manifest, no dotfiles
	durationSecs, _, _, outcome := collectCompletionTelemetry(tmpDir, false, true)

	if durationSecs != 0 {
		t.Errorf("expected 0 duration with no manifest, got %d", durationSecs)
	}
	if outcome != "success" {
		t.Errorf("expected outcome 'success', got %q", outcome)
	}
}

func TestCollectCompletionTelemetry_ArchivedWorkspace(t *testing.T) {
	// This test verifies the BUG: if workspace is archived (moved),
	// collecting telemetry from the ORIGINAL path returns 0 duration.
	tmpDir := t.TempDir()

	// Write manifest to original workspace path
	originalPath := filepath.Join(tmpDir, "workspace", "og-test-agent")
	archivedPath := filepath.Join(tmpDir, "workspace", "archived", "og-test-agent")
	if err := os.MkdirAll(originalPath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	spawnTime := time.Now().Add(-20 * time.Minute)
	manifest := spawn.AgentManifest{
		WorkspaceName: "og-test-agent",
		SpawnTime:     spawnTime.Format(time.RFC3339),
		Tier:          "full",
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(originalPath, "AGENT_MANIFEST.json"), data, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Telemetry from ORIGINAL path should work
	durationBefore, _, _, _ := collectCompletionTelemetry(originalPath, false, true)
	if durationBefore < 1140 || durationBefore > 1260 {
		t.Errorf("expected duration ~1200s before archive, got %d", durationBefore)
	}

	// Simulate archiving (move workspace)
	if err := os.MkdirAll(filepath.Dir(archivedPath), 0755); err != nil {
		t.Fatalf("failed to create archived dir: %v", err)
	}
	if err := os.Rename(originalPath, archivedPath); err != nil {
		t.Fatalf("failed to rename workspace: %v", err)
	}

	// Telemetry from ORIGINAL path should now return 0 (workspace gone)
	durationAfter, _, _, _ := collectCompletionTelemetry(originalPath, false, true)
	if durationAfter != 0 {
		t.Errorf("expected 0 duration from archived path, got %d", durationAfter)
	}

	// Telemetry from ARCHIVED path should still work
	durationArchived, _, _, _ := collectCompletionTelemetry(archivedPath, false, true)
	if durationArchived < 1140 || durationArchived > 1260 {
		t.Errorf("expected duration ~1200s from archived path, got %d", durationArchived)
	}
}

func TestCollectCompletionTelemetry_OutcomeValues(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		forced   bool
		passed   bool
		expected string
	}{
		{"success", false, true, "success"},
		{"forced", true, true, "forced"},
		{"forced_failed", true, false, "forced"},
		{"failed", false, false, "failed"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, outcome := collectCompletionTelemetry(tmpDir, tc.forced, tc.passed)
			if outcome != tc.expected {
				t.Errorf("expected outcome %q, got %q", tc.expected, outcome)
			}
		})
	}
}
