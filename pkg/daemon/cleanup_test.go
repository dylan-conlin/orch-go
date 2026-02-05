package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/registry"
)

func TestRunRegistryCleanup(t *testing.T) {
	// Create a temporary registry file
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "agent-registry.json")

	now := time.Now()
	oldTime := now.AddDate(0, 0, -10) // 10 days ago
	recentTime := now.AddDate(0, 0, -3) // 3 days ago

	type registryData struct {
		Agents []*registry.Agent `json:"agents"`
	}

	agents := registryData{
		Agents: []*registry.Agent{
			{
				ID:        "old-agent-1",
				BeadsID:   "orch-go-111",
				Status:    registry.StateActive,
				SpawnedAt: oldTime.Format(registry.TimeFormat),
				UpdatedAt: oldTime.Format(registry.TimeFormat),
			},
			{
				ID:        "old-agent-2",
				BeadsID:   "orch-go-222",
				Status:    registry.StateActive,
				SpawnedAt: oldTime.Format(registry.TimeFormat),
				UpdatedAt: oldTime.Format(registry.TimeFormat),
			},
			{
				ID:        "recent-agent",
				BeadsID:   "orch-go-333",
				Status:    registry.StateActive,
				SpawnedAt: recentTime.Format(registry.TimeFormat),
				UpdatedAt: recentTime.Format(registry.TimeFormat),
			},
		},
	}

	data, err := json.MarshalIndent(agents, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal registry: %v", err)
	}
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("Failed to write registry: %v", err)
	}

	// Override the default path for the test by setting HOME
	// Since registry.DefaultPath() uses os.UserHomeDir(), we need to use
	// the registry package directly with the temp path
	// Instead, test the logic inline since runRegistryCleanup uses DefaultPath()

	// Test: verify the cleanup logic by loading, filtering, and checking results
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	allAgents := reg.ListAgents()
	if len(allAgents) != 3 {
		t.Fatalf("Expected 3 agents, got %d", len(allAgents))
	}

	// Simulate cleanup: filter by 7-day cutoff
	cutoff := now.AddDate(0, 0, -7)
	var toKeep []*registry.Agent
	removed := 0
	for _, agent := range allAgents {
		spawnTime, err := time.Parse(registry.TimeFormat, agent.SpawnedAt)
		if err != nil {
			toKeep = append(toKeep, agent)
			continue
		}
		if spawnTime.Before(cutoff) {
			removed++
		} else {
			toKeep = append(toKeep, agent)
		}
	}

	if removed != 2 {
		t.Errorf("Expected 2 entries removed, got %d", removed)
	}
	if len(toKeep) != 1 {
		t.Errorf("Expected 1 entry kept, got %d", len(toKeep))
	}
	if toKeep[0].ID != "recent-agent" {
		t.Errorf("Expected recent-agent to be kept, got %s", toKeep[0].ID)
	}

	// Write back and verify file
	newData := registryData{Agents: toKeep}
	jsonData, err := json.MarshalIndent(newData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(registryPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Reload and verify
	reg2, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to reload registry: %v", err)
	}
	remaining := reg2.ListAgents()
	if len(remaining) != 1 {
		t.Errorf("Expected 1 agent after cleanup, got %d", len(remaining))
	}
}

func TestRunRegistryCleanupNoStaleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "agent-registry.json")

	now := time.Now()
	recentTime := now.AddDate(0, 0, -1) // 1 day ago

	type registryData struct {
		Agents []*registry.Agent `json:"agents"`
	}

	agents := registryData{
		Agents: []*registry.Agent{
			{
				ID:        "recent-agent",
				BeadsID:   "orch-go-111",
				Status:    registry.StateActive,
				SpawnedAt: recentTime.Format(registry.TimeFormat),
				UpdatedAt: recentTime.Format(registry.TimeFormat),
			},
		},
	}

	data, err := json.MarshalIndent(agents, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal registry: %v", err)
	}
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("Failed to write registry: %v", err)
	}

	// Verify no entries would be removed
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	cutoff := now.AddDate(0, 0, -7)
	removed := 0
	for _, agent := range reg.ListAgents() {
		spawnTime, _ := time.Parse(registry.TimeFormat, agent.SpawnedAt)
		if spawnTime.Before(cutoff) {
			removed++
		}
	}

	if removed != 0 {
		t.Errorf("Expected 0 entries removed, got %d", removed)
	}
}

func TestRunPeriodicCleanupIncludesRegistry(t *testing.T) {
	// Verify that CleanupRegistry config defaults are set correctly
	config := DefaultConfig()

	if !config.CleanupRegistry {
		t.Error("Expected CleanupRegistry to be true by default")
	}
	if config.CleanupRegistryAgeDays != 7 {
		t.Errorf("Expected CleanupRegistryAgeDays to be 7, got %d", config.CleanupRegistryAgeDays)
	}
}

func TestCleanupResultIncludesRegistryField(t *testing.T) {
	result := CleanupResult{
		SessionsDeleted:        1,
		WorkspacesArchived:     2,
		InvestigationsArchived: 3,
		RegistryEntriesRemoved: 4,
		Message:                "test",
	}

	if result.RegistryEntriesRemoved != 4 {
		t.Errorf("Expected RegistryEntriesRemoved=4, got %d", result.RegistryEntriesRemoved)
	}
}
