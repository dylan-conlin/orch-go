package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/registry"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "days suffix",
			input:    "7d",
			expected: 7 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "single day",
			input:    "1d",
			expected: 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "30 days",
			input:    "30d",
			expected: 30 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "hours suffix",
			input:    "168h",
			expected: 168 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "minutes suffix",
			input:    "60m",
			expected: 60 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "seconds suffix",
			input:    "3600s",
			expected: 3600 * time.Second,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			input:    "abc",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid days format",
			input:    "xd",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestShouldRemoveRegistryEntry(t *testing.T) {
	now := time.Now()
	old := now.Add(-10 * 24 * time.Hour) // 10 days ago
	recent := now.Add(-1 * time.Hour)    // 1 hour ago
	cutoff := now.Add(-7 * 24 * time.Hour)

	tests := []struct {
		name      string
		agent     *registry.Agent
		untracked bool
		olderThan time.Time
		want      bool
	}{
		{
			name: "untracked agent matches --untracked",
			agent: &registry.Agent{
				ID:        "test-1",
				BeadsID:   "orch-go-untracked-12345",
				Status:    registry.StateActive,
				SpawnedAt: recent.Format(registry.TimeFormat),
			},
			untracked: true,
			olderThan: time.Time{},
			want:      true,
		},
		{
			name: "tracked agent does not match --untracked",
			agent: &registry.Agent{
				ID:        "test-2",
				BeadsID:   "orch-go-12345",
				Status:    registry.StateActive,
				SpawnedAt: recent.Format(registry.TimeFormat),
			},
			untracked: true,
			olderThan: time.Time{},
			want:      false,
		},
		{
			name: "old agent matches --older-than",
			agent: &registry.Agent{
				ID:        "test-3",
				BeadsID:   "orch-go-12345",
				Status:    registry.StateActive,
				SpawnedAt: old.Format(registry.TimeFormat),
			},
			untracked: false,
			olderThan: cutoff,
			want:      true,
		},
		{
			name: "recent agent does not match --older-than",
			agent: &registry.Agent{
				ID:        "test-4",
				BeadsID:   "orch-go-12345",
				Status:    registry.StateActive,
				SpawnedAt: recent.Format(registry.TimeFormat),
			},
			untracked: false,
			olderThan: cutoff,
			want:      false,
		},
		{
			name: "both filters: untracked recent matches via untracked",
			agent: &registry.Agent{
				ID:        "test-5",
				BeadsID:   "orch-go-untracked-12345",
				Status:    registry.StateActive,
				SpawnedAt: recent.Format(registry.TimeFormat),
			},
			untracked: true,
			olderThan: cutoff,
			want:      true,
		},
		{
			name: "both filters: tracked old matches via age",
			agent: &registry.Agent{
				ID:        "test-6",
				BeadsID:   "orch-go-12345",
				Status:    registry.StateActive,
				SpawnedAt: old.Format(registry.TimeFormat),
			},
			untracked: true,
			olderThan: cutoff,
			want:      true,
		},
		{
			name: "neither filter active returns false",
			agent: &registry.Agent{
				ID:        "test-7",
				BeadsID:   "orch-go-12345",
				Status:    registry.StateActive,
				SpawnedAt: recent.Format(registry.TimeFormat),
			},
			untracked: false,
			olderThan: time.Time{},
			want:      false,
		},
		{
			name: "empty beads ID does not match untracked",
			agent: &registry.Agent{
				ID:        "test-8",
				BeadsID:   "",
				Status:    registry.StateActive,
				SpawnedAt: recent.Format(registry.TimeFormat),
			},
			untracked: true,
			olderThan: time.Time{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRemoveRegistryEntry(tt.agent, tt.untracked, tt.olderThan)
			if got != tt.want {
				t.Errorf("shouldRemoveRegistryEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegistryCleanUntracked(t *testing.T) {
	// Create a temp registry file with mixed entries
	tmpDir := t.TempDir()
	regPath := filepath.Join(tmpDir, "agent-registry.json")

	now := time.Now()

	type regData struct {
		Agents []*registry.Agent `json:"agents"`
	}

	data := regData{
		Agents: []*registry.Agent{
			{
				ID:        "og-feat-tracked-01jan",
				BeadsID:   "orch-go-12345",
				Status:    registry.StateActive,
				SpawnedAt: now.Format(registry.TimeFormat),
				UpdatedAt: now.Format(registry.TimeFormat),
			},
			{
				ID:        "og-feat-untracked-01jan",
				BeadsID:   "orch-go-untracked-9999999",
				Status:    registry.StateActive,
				SpawnedAt: now.Format(registry.TimeFormat),
				UpdatedAt: now.Format(registry.TimeFormat),
			},
			{
				ID:        "og-feat-another-tracked-01jan",
				BeadsID:   "orch-go-67890",
				Status:    registry.StateActive,
				SpawnedAt: now.Format(registry.TimeFormat),
				UpdatedAt: now.Format(registry.TimeFormat),
			},
		},
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if err := os.WriteFile(regPath, bytes, 0644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Load registry and purge untracked
	reg, err := registry.New(regPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	all := reg.ListAll()
	if len(all) != 3 {
		t.Fatalf("ListAll() = %d, want 3", len(all))
	}

	// Purge untracked entries
	removed := reg.Purge(func(a *registry.Agent) bool {
		return isUntrackedRegistryEntry(a)
	})
	if removed != 1 {
		t.Errorf("Purge() removed %d, want 1", removed)
	}

	if err := reg.SaveSkipMerge(); err != nil {
		t.Fatalf("SaveSkipMerge() error: %v", err)
	}

	// Reload and verify
	reg2, err := registry.New(regPath)
	if err != nil {
		t.Fatalf("New() reload error: %v", err)
	}

	remaining := reg2.ListAll()
	if len(remaining) != 2 {
		t.Fatalf("After purge: ListAll() = %d, want 2", len(remaining))
	}

	// Verify the correct entries survived
	for _, a := range remaining {
		if isUntrackedRegistryEntry(a) {
			t.Errorf("Untracked entry %s survived purge", a.ID)
		}
	}
}

func TestIsUntrackedRegistryEntry(t *testing.T) {
	tests := []struct {
		name  string
		agent *registry.Agent
		want  bool
	}{
		{
			name:  "untracked beads ID",
			agent: &registry.Agent{BeadsID: "orch-go-untracked-12345"},
			want:  true,
		},
		{
			name:  "tracked beads ID",
			agent: &registry.Agent{BeadsID: "orch-go-12345"},
			want:  false,
		},
		{
			name:  "empty beads ID",
			agent: &registry.Agent{BeadsID: ""},
			want:  false,
		},
		{
			name:  "kb-cli untracked",
			agent: &registry.Agent{BeadsID: "kb-cli-untracked-99999"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUntrackedRegistryEntry(tt.agent)
			if got != tt.want {
				t.Errorf("isUntrackedRegistryEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}
