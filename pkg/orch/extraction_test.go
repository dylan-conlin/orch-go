package orch

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/model"
)

func TestDetermineSpawnBackend_ExplicitBackendWins(t *testing.T) {
	// Explicit --backend should always win, even when --opus is set
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	tests := []struct {
		name        string
		backendFlag string
		opusFlag    bool
		task        string // non-infra task
		want        string
	}{
		{
			name:        "explicit opencode wins over opus flag",
			backendFlag: "opencode",
			opusFlag:    true,
			task:        "add user feature",
			want:        "opencode",
		},
		{
			name:        "explicit claude with opus flag",
			backendFlag: "claude",
			opusFlag:    true,
			task:        "add user feature",
			want:        "claude",
		},
		{
			name:        "explicit opencode without opus",
			backendFlag: "opencode",
			opusFlag:    false,
			task:        "add user feature",
			want:        "opencode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetermineSpawnBackend(sonnet, tt.task, "", "", tt.backendFlag, "", tt.opusFlag)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("DetermineSpawnBackend() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetermineSpawnBackend_InfraAdvisory(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	// When --backend is explicit AND infra work detected, backend is respected (advisory only)
	got, err := DetermineSpawnBackend(sonnet, "fix opencode server crash", "", "", "opencode", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "opencode" {
		t.Errorf("explicit --backend opencode should win over infra detection, got %q", got)
	}

	// When --backend is NOT set AND infra work detected, auto-apply claude
	got, err = DetermineSpawnBackend(sonnet, "fix opencode server crash", "", "", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "claude" {
		t.Errorf("infra detection without explicit --backend should auto-apply claude, got %q", got)
	}
}

func TestDetermineSpawnBackend_InvalidBackend(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	_, err := DetermineSpawnBackend(sonnet, "some task", "", "", "invalid", "", false)
	if err == nil {
		t.Fatal("expected error for invalid backend value")
	}
}
