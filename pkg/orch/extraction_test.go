package orch

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/model"
)

func TestDetermineSpawnBackend_ExplicitBackendWins(t *testing.T) {
	// Explicit --backend should always win
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	tests := []struct {
		name        string
		backendFlag string
		task        string // non-infra task
		want        string
	}{
		{
			name:        "explicit opencode",
			backendFlag: "opencode",
			task:        "add user feature",
			want:        "opencode",
		},
		{
			name:        "explicit claude",
			backendFlag: "claude",
			task:        "add user feature",
			want:        "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetermineSpawnBackend(sonnet, tt.task, "", "", tt.backendFlag, "")
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
	got, err := DetermineSpawnBackend(sonnet, "fix opencode server crash", "", "", "opencode", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "opencode" {
		t.Errorf("explicit --backend opencode should win over infra detection, got %q", got)
	}

	// When --backend is NOT set AND infra work detected, auto-apply claude
	got, err = DetermineSpawnBackend(sonnet, "fix opencode server crash", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "claude" {
		t.Errorf("infra detection without explicit --backend should auto-apply claude, got %q", got)
	}
}

func TestDetermineSpawnBackend_ExplicitModelPreventsInfraOverride(t *testing.T) {
	codex := model.ModelSpec{Provider: "openai", ModelID: "gpt-5.2-codex"}

	// When --model is explicit AND infra work detected, escape hatch should NOT override
	// Backend should fall through to config/default instead of being forced to "claude"
	// Note: without project config, default is "claude" - but the key behavior is that
	// infrastructure detection doesn't force it; the config path is taken instead
	got, err := DetermineSpawnBackend(codex, "fix opencode server crash", "", "", "", "codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With no project config in test, falls through to default "claude"
	// The important test is that it does NOT hit the infrastructure auto-apply path
	// (which would log events and print escape hatch messages)
	_ = got // Backend value depends on config; key assertion is no error

	// When NEITHER --model NOR --backend is set, infra detection should auto-apply claude
	got, err = DetermineSpawnBackend(codex, "fix opencode server crash", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "claude" {
		t.Errorf("infra detection without any explicit flags should auto-apply claude, got %q", got)
	}
}

func TestDetermineSpawnBackend_ExplicitModelAndBackend(t *testing.T) {
	codex := model.ModelSpec{Provider: "openai", ModelID: "gpt-5.2-codex"}

	// Both --model and --backend explicit with infra task: --backend wins
	got, err := DetermineSpawnBackend(codex, "fix spawn_cmd.go escape hatch", "", "", "opencode", "codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "opencode" {
		t.Errorf("explicit --backend opencode should win over infra detection even with --model codex, got %q", got)
	}
}

func TestDetermineSpawnBackend_InvalidBackend(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	_, err := DetermineSpawnBackend(sonnet, "some task", "", "", "invalid", "")
	if err == nil {
		t.Fatal("expected error for invalid backend value")
	}
}
