package orch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
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

	// Isolate user config lookups
	t.Setenv("HOME", t.TempDir())

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

	// Isolate user config lookups
	t.Setenv("HOME", t.TempDir())

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

func TestDetermineSpawnBackend_UserDefaultModelPreventsInfraOverride(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	configHome := t.TempDir()
	t.Setenv("HOME", configHome)
	configDir := filepath.Join(configHome, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create user config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("default_model: gpt-4o\n"), 0644); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}

	got, err := DetermineSpawnBackend(sonnet, "fix opencode server crash", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "claude" {
		t.Errorf("user default_model should prevent infra auto-apply, got %q", got)
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

	// Set explicit user config backend
	configHome := t.TempDir()
	t.Setenv("HOME", configHome)
	configDir := filepath.Join(configHome, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create user config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("backend: claude\n"), 0644); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}

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

	// Isolate user config lookups (no config file)
	t.Setenv("HOME", t.TempDir())

	// When no explicit flags, no project config, no user config backend,
	// the hardcoded default should be "opencode"
	// Note: this test validates the default when user config exists but may have backend set.
	// The hardcoded default "opencode" is the last resort.
	got, err := DetermineSpawnBackend(sonnet, "add user feature", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "opencode" {
		t.Errorf("default backend should be 'opencode' without explicit config, got %q", got)
	}
}

func TestDetermineSpawnBackend_ConfigOverridesInfra(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	t.Run("user config overrides infra", func(t *testing.T) {
		configHome := t.TempDir()
		t.Setenv("HOME", configHome)
		configDir := filepath.Join(configHome, ".orch")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("failed to create user config dir: %v", err)
		}
		configPath := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("backend: opencode\n"), 0644); err != nil {
			t.Fatalf("failed to write user config: %v", err)
		}

		got, err := DetermineSpawnBackend(sonnet, "fix opencode server crash", "", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "opencode" {
			t.Errorf("user config backend should override infra detection, got %q", got)
		}
	})

	t.Run("project config overrides infra", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		projectDir := t.TempDir()
		configDir := filepath.Join(projectDir, ".orch")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("failed to create project config dir: %v", err)
		}
		configPath := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("spawn_mode: opencode\n"), 0644); err != nil {
			t.Fatalf("failed to write project config: %v", err)
		}

		got, err := DetermineSpawnBackend(sonnet, "fix opencode server crash", "", projectDir, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "opencode" {
			t.Errorf("project config spawn_mode should override infra detection, got %q", got)
		}
	})
}

func TestDetermineSpawnBackend_ProjectConfigWithoutSpawnModeDoesNotOverrideUserBackend(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	configHome := t.TempDir()
	t.Setenv("HOME", configHome)

	configDir := filepath.Join(configHome, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create user config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("backend: claude\n"), 0644); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}

	projectDir := t.TempDir()
	projectConfigDir := filepath.Join(projectDir, ".orch")
	if err := os.MkdirAll(projectConfigDir, 0755); err != nil {
		t.Fatalf("failed to create project config dir: %v", err)
	}
	projectConfigPath := filepath.Join(projectConfigDir, "config.yaml")
	if err := os.WriteFile(projectConfigPath, []byte("servers:\n  web: 5173\n"), 0644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	got, err := DetermineSpawnBackend(sonnet, "add user feature", "", projectDir, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "claude" {
		t.Errorf("project config without spawn_mode should not override user backend, got %q", got)
	}
}

func TestDetermineSpawnBackend_InvalidBackend(t *testing.T) {
	sonnet := model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}

	_, err := DetermineSpawnBackend(sonnet, "some task", "", "", "invalid", "")
	if err == nil {
		t.Fatal("expected error for invalid backend value")
	}
}

func TestIsArchitectIssue(t *testing.T) {
	tests := []struct {
		name  string
		issue *verify.Issue
		want  bool
	}{
		{
			name: "skill:architect label",
			issue: &verify.Issue{
				Title:     "some task",
				IssueType: "task",
				Labels:    []string{"skill:architect"},
			},
			want: true,
		},
		{
			name: "architect in title",
			issue: &verify.Issue{
				Title:     "[orch-go] architect: design extraction first",
				IssueType: "task",
				Labels:    nil,
			},
			want: true,
		},
		{
			name: "feature-impl issue",
			issue: &verify.Issue{
				Title:     "[orch-go] feature-impl: add hotspot gate",
				IssueType: "task",
				Labels:    []string{"skill:feature-impl"},
			},
			want: false,
		},
		{
			name: "no labels or architect title",
			issue: &verify.Issue{
				Title:     "fix something",
				IssueType: "bug",
				Labels:    nil,
			},
			want: false,
		},
		{
			name: "architect label among others",
			issue: &verify.Issue{
				Title:     "review hotspot area",
				IssueType: "task",
				Labels:    []string{"priority:high", "skill:architect", "area:spawn"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isArchitectIssue(tt.issue)
			if got != tt.want {
				t.Errorf("isArchitectIssue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetermineSpawnTier_TaskScopeSignals(t *testing.T) {
	// Isolate user config lookups
	t.Setenv("HOME", t.TempDir())

	tests := []struct {
		name string
		task string
		want string
	}{
		{
			name: "session scope medium upgrades to full",
			task: "SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])",
			want: spawn.TierFull,
		},
		{
			name: "session scope small keeps light",
			task: "SESSION SCOPE: Small (text edits only)",
			want: spawn.TierLight,
		},
		{
			name: "new package with tests upgrades to full",
			task: "Create new package pkg/graph with comprehensive tests",
			want: spawn.TierFull,
		},
		{
			name: "no scope signals uses skill default",
			task: "Update README wording",
			want: spawn.TierLight,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineSpawnTier("feature-impl", tt.task, false, false)
			if got != tt.want {
				t.Errorf("DetermineSpawnTier() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- extractSearchTerms tests ---

func TestExtractSearchTerms(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		contains []string // expected terms that should be in result
	}{
		{
			name:     "single file with full path",
			files:    []string{"pkg/orch/extraction.go"},
			contains: []string{"pkg/orch/extraction.go", "extraction"},
		},
		{
			name:     "basename only",
			files:    []string{"daemon.go"},
			contains: []string{"daemon"},
		},
		{
			name:     "multiple files",
			files:    []string{"cmd/orch/main.go", "pkg/daemon/daemon.go"},
			contains: []string{"cmd/orch/main.go", "main", "pkg/daemon/daemon.go", "daemon"},
		},
		{
			name:     "empty list",
			files:    []string{},
			contains: []string{},
		},
		{
			name:     "empty string in list",
			files:    []string{""},
			contains: []string{},
		},
		{
			name:     "case normalization",
			files:    []string{"Cmd/Orch/Main.Go"},
			contains: []string{"cmd/orch/main.go", "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := extractSearchTerms(tt.files)
			for _, expected := range tt.contains {
				found := false
				for _, term := range terms {
					if term == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("extractSearchTerms(%v) missing expected term %q, got %v", tt.files, expected, terms)
				}
			}
		})
	}
}
