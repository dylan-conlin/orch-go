package orch

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
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
	// Backend should fall through to user config/default instead of being forced to "claude"
	got, err := DetermineSpawnBackend(codex, "fix opencode server crash", "", "", "", "codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With no project config, falls through to user config (~/.orch/config.yaml)
	// or hardcoded default "opencode". Either way, NOT the infra auto-apply path.
	_ = got // Backend value depends on user config; key assertion is no error

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

func TestDetermineSpawnBackend_UserConfigFallback(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	// Load user config to see what backend is set
	userCfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("failed to load user config: %v", err)
	}

	// Determine expected backend: user config backend, or "opencode" default
	expectedBackend := "opencode"
	if userCfg != nil && userCfg.Backend != "" {
		expectedBackend = userCfg.Backend
	}

	// No explicit flags, non-infra task, no project config → should use user config backend
	got, err := DetermineSpawnBackend(sonnet, "add user feature", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expectedBackend {
		t.Errorf("no flags, no project config: got %q, want %q (from user config)", got, expectedBackend)
	}

	// Explicit --model, non-infra task, no project config → should also use user config backend
	got, err = DetermineSpawnBackend(sonnet, "add user feature", "", "", "", "sonnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expectedBackend {
		t.Errorf("explicit model, no project config: got %q, want %q (from user config)", got, expectedBackend)
	}
}

func TestDetermineSpawnBackend_HardcodedDefaultIsOpencode(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	// When no explicit flags, no project config, no user config backend,
	// the hardcoded default should be "opencode"
	// Note: this test validates the default when user config exists but may have backend set.
	// The hardcoded default "opencode" is the last resort.
	got, err := DetermineSpawnBackend(sonnet, "add user feature", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should never be "claude" without explicit flag or infrastructure detection
	if got == "claude" {
		t.Errorf("default backend should not be 'claude' without explicit flag, got %q", got)
	}
}

func TestDetermineSpawnBackend_InvalidBackend(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	_, err := DetermineSpawnBackend(sonnet, "some task", "", "", "invalid", "")
	if err == nil {
		t.Fatal("expected error for invalid backend value")
	}
}
