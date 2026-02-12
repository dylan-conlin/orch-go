package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

func TestResolveBackend(t *testing.T) {
	tests := []struct {
		name            string
		backendFlag     string
		opusFlag        bool
		infraFlag       bool
		modelFlag       string
		projCfg         *config.Config
		globalCfg       *userconfig.Config
		task            string
		beadsID         string
		expectedBackend string
		expectedReason  string
		expectWarnings  bool
	}{
		{
			name:            "explicit --backend opencode flag wins",
			backendFlag:     "opencode",
			expectedBackend: "opencode",
			expectedReason:  "--backend opencode flag",
		},
		{
			name:            "explicit --backend claude flag wins",
			backendFlag:     "claude",
			expectedBackend: "claude",
			expectedReason:  "--backend claude flag",
		},
		{
			name:            "explicit --backend docker flag wins",
			backendFlag:     "docker",
			expectedBackend: "docker",
			expectedReason:  "--backend docker flag",
		},
		{
			name:            "--opus flag implies claude",
			opusFlag:        true,
			expectedBackend: "claude",
			expectedReason:  "--opus flag (implies claude backend)",
		},
		{
			name:            "--backend flag beats --opus flag",
			backendFlag:     "opencode",
			opusFlag:        true, // --opus should be ignored when --backend is set
			expectedBackend: "opencode",
			expectedReason:  "--backend opencode flag",
		},
		{
			name:            "project config spawn_mode: opencode",
			projCfg:         &config.Config{SpawnMode: "opencode"},
			expectedBackend: "opencode",
			expectedReason:  "project config (spawn_mode: opencode)",
		},
		{
			name:            "project config spawn_mode: claude",
			projCfg:         &config.Config{SpawnMode: "claude"},
			expectedBackend: "claude",
			expectedReason:  "project config (spawn_mode: claude)",
		},
		{
			name:            "project config spawn_mode: docker",
			projCfg:         &config.Config{SpawnMode: "docker"},
			expectedBackend: "docker",
			expectedReason:  "project config (spawn_mode: docker)",
		},
		{
			name:            "--opus flag beats project config",
			opusFlag:        true,
			projCfg:         &config.Config{SpawnMode: "opencode"},
			expectedBackend: "claude",
			expectedReason:  "--opus flag (implies claude backend)",
		},
		{
			name:            "global config backend: opencode",
			globalCfg:       &userconfig.Config{Backend: "opencode"},
			expectedBackend: "opencode",
			expectedReason:  "global config (backend: opencode)",
		},
		{
			name:            "global config backend: claude",
			globalCfg:       &userconfig.Config{Backend: "claude"},
			expectedBackend: "claude",
			expectedReason:  "global config (backend: claude)",
		},
		{
			name:            "global config backend: docker",
			globalCfg:       &userconfig.Config{Backend: "docker"},
			expectedBackend: "docker",
			expectedReason:  "global config (backend: docker)",
		},
		{
			name:            "project config beats global config",
			projCfg:         &config.Config{SpawnMode: "claude"},
			globalCfg:       &userconfig.Config{Backend: "opencode"},
			expectedBackend: "claude",
			expectedReason:  "project config (spawn_mode: claude)",
		},
		{
			name:            "default is opencode when no config",
			expectedBackend: "opencode",
			expectedReason:  "default (opencode for cost optimization)",
		},
		{
			name:            "invalid --backend value falls through",
			backendFlag:     "invalid",
			expectedBackend: "opencode",
			expectedReason:  "default (opencode for cost optimization)",
			expectWarnings:  true,
		},
		{
			name:            "invalid project config falls through to global",
			projCfg:         &config.Config{SpawnMode: "invalid"},
			globalCfg:       &userconfig.Config{Backend: "claude"},
			expectedBackend: "claude",
			expectedReason:  "global config (backend: claude)",
			expectWarnings:  true,
		},
		{
			name:            "infrastructure work with opencode gets warning",
			task:            "fix serve.go startup issue",
			expectedBackend: "opencode",
			expectedReason:  "default (opencode for cost optimization)",
			expectWarnings:  true, // infrastructure warning
		},
		{
			name:            "infrastructure work with claude no warning",
			backendFlag:     "claude",
			task:            "fix serve.go startup issue",
			expectedBackend: "claude",
			expectedReason:  "--backend claude flag",
			expectWarnings:  false, // no warning for claude + infra
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveBackend(
				tt.backendFlag,
				tt.opusFlag,
				tt.infraFlag,
				tt.modelFlag,
				tt.projCfg,
				tt.globalCfg,
				tt.task,
				tt.beadsID,
			)

			if result.Backend != tt.expectedBackend {
				t.Errorf("Backend: got %q, want %q", result.Backend, tt.expectedBackend)
			}

			if result.Reason != tt.expectedReason {
				t.Errorf("Reason: got %q, want %q", result.Reason, tt.expectedReason)
			}

			hasWarnings := len(result.Warnings) > 0
			if hasWarnings != tt.expectWarnings {
				t.Errorf("Warnings: got %v (len=%d), want hasWarnings=%v", result.Warnings, len(result.Warnings), tt.expectWarnings)
			}
		})
	}
}

func TestValidateBackendModelCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		backend     string
		modelFlag   string
		wantWarning bool
	}{
		{
			name:        "opencode + opus = warning",
			backend:     "opencode",
			modelFlag:   "opus",
			wantWarning: true,
		},
		{
			name:        "opencode + claude-opus = warning",
			backend:     "opencode",
			modelFlag:   "claude-opus-4",
			wantWarning: true,
		},
		{
			name:        "opencode + sonnet = ok",
			backend:     "opencode",
			modelFlag:   "sonnet",
			wantWarning: false,
		},
		{
			name:        "claude + opus = ok",
			backend:     "claude",
			modelFlag:   "opus",
			wantWarning: false,
		},
		{
			name:        "opencode + deepseek = ok",
			backend:     "opencode",
			modelFlag:   "deepseek",
			wantWarning: false,
		},
		{
			name:        "opencode + empty model = ok",
			backend:     "opencode",
			modelFlag:   "",
			wantWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning := validateBackendModelCompatibility(tt.backend, tt.modelFlag)
			gotWarning := warning != ""

			if gotWarning != tt.wantWarning {
				t.Errorf("got warning=%v (%q), want warning=%v", gotWarning, warning, tt.wantWarning)
			}
		})
	}
}

func TestResolveBackendPriorityChain(t *testing.T) {
	// This test explicitly verifies the priority chain:
	// 1) --backend flag
	// 2) --opus flag
	// 3) project config
	// 4) global config
	// 5) default opencode

	// Set up all options - most specific should win
	projCfg := &config.Config{SpawnMode: "claude"}
	globalCfg := &userconfig.Config{Backend: "claude"}

	// Priority 1: --backend beats everything
	result := resolveBackend("opencode", true, false, "", projCfg, globalCfg, "", "")
	if result.Backend != "opencode" {
		t.Errorf("Priority 1 failed: --backend should beat --opus and configs, got %s", result.Backend)
	}

	// Priority 2: --opus beats configs
	result = resolveBackend("", true, false, "", projCfg, globalCfg, "", "")
	if result.Backend != "claude" {
		t.Errorf("Priority 2 failed: --opus should beat configs, got %s", result.Backend)
	}

	// Priority 3: project config beats global config
	projCfg.SpawnMode = "opencode"
	globalCfg.Backend = "claude"
	result = resolveBackend("", false, false, "", projCfg, globalCfg, "", "")
	if result.Backend != "opencode" {
		t.Errorf("Priority 3 failed: project config should beat global config, got %s", result.Backend)
	}

	// Priority 4: global config beats default
	result = resolveBackend("", false, false, "", nil, globalCfg, "", "")
	if result.Backend != "claude" {
		t.Errorf("Priority 4 failed: global config should be used when no project config, got %s", result.Backend)
	}

	// Priority 5: default is opencode
	result = resolveBackend("", false, false, "", nil, nil, "", "")
	if result.Backend != "opencode" {
		t.Errorf("Priority 5 failed: default should be opencode, got %s", result.Backend)
	}
}

func TestResolveBackendDisabledBackends(t *testing.T) {
	tests := []struct {
		name            string
		backendFlag     string
		opusFlag        bool
		infraFlag       bool
		projCfg         *config.Config
		globalCfg       *userconfig.Config
		expectedBackend string
		expectedReason  string
		expectError     bool
		expectWarnings  bool
	}{
		{
			name:        "explicit --backend docker when docker disabled returns error",
			backendFlag: "docker",
			globalCfg: &userconfig.Config{
				Backend:          "claude",
				DisabledBackends: []string{"docker"},
			},
			expectError: true,
		},
		{
			name:     "--opus ignored when claude disabled",
			opusFlag: true,
			globalCfg: &userconfig.Config{
				Backend:          "opencode",
				DisabledBackends: []string{"claude"},
			},
			expectedBackend: "opencode",
			expectedReason:  "global config (backend: opencode)",
			expectWarnings:  true, // warning about --opus being ignored
		},
		{
			name: "project config skipped when disabled",
			projCfg: &config.Config{
				SpawnMode: "docker",
			},
			globalCfg: &userconfig.Config{
				Backend:          "claude",
				DisabledBackends: []string{"docker"},
			},
			expectedBackend: "claude",
			expectedReason:  "global config (backend: claude)",
			expectWarnings:  true, // warning about project config being skipped
		},
		{
			name: "global config skipped when disabled, falls to default",
			globalCfg: &userconfig.Config{
				Backend:          "docker",
				DisabledBackends: []string{"docker"},
			},
			expectedBackend: "opencode",
			expectedReason:  "default (opencode for cost optimization)",
			expectWarnings:  true,
		},
		{
			name: "default opencode disabled, falls to claude",
			globalCfg: &userconfig.Config{
				DisabledBackends: []string{"opencode"},
			},
			expectedBackend: "claude",
			expectedReason:  "fallback (opencode disabled)",
			expectWarnings:  true,
		},
		{
			name: "opencode and claude disabled, falls to docker",
			globalCfg: &userconfig.Config{
				DisabledBackends: []string{"opencode", "claude"},
			},
			expectedBackend: "docker",
			expectedReason:  "fallback (opencode and claude disabled)",
			expectWarnings:  true,
		},
		{
			name: "all backends disabled returns error",
			globalCfg: &userconfig.Config{
				DisabledBackends: []string{"opencode", "claude", "docker"},
			},
			expectError: true,
		},
		{
			name: "disabled backend not selected even in config chain",
			projCfg: &config.Config{
				SpawnMode: "docker",
			},
			globalCfg: &userconfig.Config{
				Backend:          "docker",
				DisabledBackends: []string{"docker"},
			},
			expectedBackend: "opencode",
			expectedReason:  "default (opencode for cost optimization)",
			expectWarnings:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveBackend(
				tt.backendFlag,
				tt.opusFlag,
				tt.infraFlag,
				"",
				tt.projCfg,
				tt.globalCfg,
				"",
				"",
			)

			if tt.expectError {
				if result.Error == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
				return
			}

			if result.Backend != tt.expectedBackend {
				t.Errorf("Backend: got %q, want %q", result.Backend, tt.expectedBackend)
			}

			if result.Reason != tt.expectedReason {
				t.Errorf("Reason: got %q, want %q", result.Reason, tt.expectedReason)
			}

			hasWarnings := len(result.Warnings) > 0
			if hasWarnings != tt.expectWarnings {
				t.Errorf("Warnings: got %v (len=%d), want hasWarnings=%v", result.Warnings, len(result.Warnings), tt.expectWarnings)
			}
		})
	}
}

// mockBackendChecker is a test double for BackendAvailabilityChecker.
// availableBackends maps backend name to availability (nil = available, error = unavailable).
type mockBackendChecker struct {
	available map[string]bool
}

func (m *mockBackendChecker) IsAvailable(backend string) error {
	if m.available[backend] {
		return nil
	}
	return fmt.Errorf("%s is not available", backend)
}

func TestResolveBackendWithAvailability(t *testing.T) {
	tests := []struct {
		name            string
		backendFlag     string
		opusFlag        bool
		infraFlag       bool
		projCfg         *config.Config
		globalCfg       *userconfig.Config
		available       map[string]bool // which backends are available
		expectedBackend string
		expectWarnings  bool
		expectError     bool
		reasonContains  string // substring to check in Reason
	}{
		{
			name:            "all available - uses default opencode",
			available:       map[string]bool{"opencode": true, "claude": true, "docker": true},
			expectedBackend: "opencode",
			reasonContains:  "default",
		},
		{
			name:            "opencode unavailable - falls back to claude",
			available:       map[string]bool{"opencode": false, "claude": true, "docker": true},
			expectedBackend: "claude",
			expectWarnings:  true,
			reasonContains:  "fallback",
		},
		{
			name:            "opencode and claude unavailable - falls back to docker",
			available:       map[string]bool{"opencode": false, "claude": false, "docker": true},
			expectedBackend: "docker",
			expectWarnings:  true,
			reasonContains:  "fallback",
		},
		{
			name:        "all unavailable - returns error",
			available:   map[string]bool{"opencode": false, "claude": false, "docker": false},
			expectError: true,
		},
		{
			name:            "explicit --backend flag skips fallback even when unavailable",
			backendFlag:     "docker",
			available:       map[string]bool{"opencode": true, "claude": true, "docker": false},
			expectedBackend: "docker",
			reasonContains:  "--backend docker flag",
		},
		{
			name:            "config-selected docker unavailable - falls back to opencode",
			globalCfg:       &userconfig.Config{Backend: "docker"},
			available:       map[string]bool{"opencode": true, "claude": true, "docker": false},
			expectedBackend: "opencode",
			expectWarnings:  true,
			reasonContains:  "fallback",
		},
		{
			name:            "project config docker unavailable - falls back to opencode",
			projCfg:         &config.Config{SpawnMode: "docker"},
			available:       map[string]bool{"opencode": true, "claude": true, "docker": false},
			expectedBackend: "opencode",
			expectWarnings:  true,
			reasonContains:  "fallback",
		},
		{
			name:            "--opus flag with claude unavailable - falls back to opencode",
			opusFlag:        true,
			available:       map[string]bool{"opencode": true, "claude": false, "docker": true},
			expectedBackend: "opencode",
			expectWarnings:  true,
			reasonContains:  "fallback",
		},
		{
			name:            "opencode disabled + claude unavailable - falls back to docker",
			globalCfg:       &userconfig.Config{DisabledBackends: []string{"opencode"}},
			available:       map[string]bool{"opencode": true, "claude": false, "docker": true},
			expectedBackend: "docker",
			expectWarnings:  true,
			reasonContains:  "fallback",
		},
		{
			name:            "fallback skips disabled backends",
			globalCfg:       &userconfig.Config{DisabledBackends: []string{"claude"}},
			available:       map[string]bool{"opencode": false, "claude": true, "docker": true},
			expectedBackend: "docker",
			expectWarnings:  true,
			reasonContains:  "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &mockBackendChecker{available: tt.available}
			result := resolveBackendWithAvailability(
				tt.backendFlag,
				tt.opusFlag,
				tt.infraFlag,
				"",
				tt.projCfg,
				tt.globalCfg,
				"",
				"",
				checker,
			)

			if tt.expectError {
				if result.Error == nil {
					t.Errorf("expected error, got nil (backend=%s, reason=%s)", result.Backend, result.Reason)
				}
				return
			}

			if result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
				return
			}

			if result.Backend != tt.expectedBackend {
				t.Errorf("Backend: got %q, want %q (reason: %s)", result.Backend, tt.expectedBackend, result.Reason)
			}

			if tt.reasonContains != "" && !strings.Contains(result.Reason, tt.reasonContains) {
				t.Errorf("Reason: got %q, want it to contain %q", result.Reason, tt.reasonContains)
			}

			hasWarnings := len(result.Warnings) > 0
			if hasWarnings != tt.expectWarnings {
				t.Errorf("Warnings: got %v (len=%d), want hasWarnings=%v", result.Warnings, len(result.Warnings), tt.expectWarnings)
			}
		})
	}
}

func TestIsBackendDisabled(t *testing.T) {
	tests := []struct {
		name             string
		disabledBackends []string
		backend          string
		expected         bool
	}{
		{
			name:             "empty list returns false",
			disabledBackends: nil,
			backend:          "docker",
			expected:         false,
		},
		{
			name:             "backend in list returns true",
			disabledBackends: []string{"docker"},
			backend:          "docker",
			expected:         true,
		},
		{
			name:             "backend not in list returns false",
			disabledBackends: []string{"docker"},
			backend:          "claude",
			expected:         false,
		},
		{
			name:             "multiple disabled backends",
			disabledBackends: []string{"docker", "opencode"},
			backend:          "opencode",
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &userconfig.Config{
				DisabledBackends: tt.disabledBackends,
			}
			got := cfg.IsBackendDisabled(tt.backend)
			if got != tt.expected {
				t.Errorf("IsBackendDisabled(%q) = %v, want %v", tt.backend, got, tt.expected)
			}
		})
	}
}
