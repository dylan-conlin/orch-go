package gates

import (
	"os"
	"testing"
)

func TestGetMaxAgents_Default(t *testing.T) {
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	os.Unsetenv("ORCH_MAX_AGENTS")

	got := GetMaxAgents(0)
	if got != DefaultMaxAgents {
		t.Errorf("GetMaxAgents(0) = %d, want %d", got, DefaultMaxAgents)
	}
}

func TestGetMaxAgents_FlagOverridesEnv(t *testing.T) {
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	os.Setenv("ORCH_MAX_AGENTS", "20")

	got := GetMaxAgents(10)
	if got != 10 {
		t.Errorf("GetMaxAgents(10) = %d, want 10 (flag takes precedence)", got)
	}
}

func TestGetMaxAgents_EnvVar(t *testing.T) {
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	os.Setenv("ORCH_MAX_AGENTS", "15")

	got := GetMaxAgents(0)
	if got != 15 {
		t.Errorf("GetMaxAgents(0) = %d, want 15 (from env)", got)
	}
}

func TestGetMaxAgents_InvalidEnvFallsBack(t *testing.T) {
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	os.Setenv("ORCH_MAX_AGENTS", "not-a-number")

	got := GetMaxAgents(0)
	if got != DefaultMaxAgents {
		t.Errorf("GetMaxAgents(0) = %d, want %d (default for invalid env)", got, DefaultMaxAgents)
	}
}
