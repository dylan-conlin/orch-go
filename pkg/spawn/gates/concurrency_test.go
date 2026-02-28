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

	// -1 = flag not set, should use default
	got := GetMaxAgents(-1)
	if got != DefaultMaxAgents {
		t.Errorf("GetMaxAgents(-1) = %d, want %d", got, DefaultMaxAgents)
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

	// -1 = flag not set, should fall through to env var
	got := GetMaxAgents(-1)
	if got != 15 {
		t.Errorf("GetMaxAgents(-1) = %d, want 15 (from env)", got)
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

	// -1 = flag not set, invalid env should fall back to default
	got := GetMaxAgents(-1)
	if got != DefaultMaxAgents {
		t.Errorf("GetMaxAgents(-1) = %d, want %d (default for invalid env)", got, DefaultMaxAgents)
	}
}

func TestGetMaxAgents_ZeroMeansUnlimited(t *testing.T) {
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	os.Setenv("ORCH_MAX_AGENTS", "20")

	// --max-agents 0 should mean unlimited (return 0), overriding env var
	got := GetMaxAgents(0)
	if got != 0 {
		t.Errorf("GetMaxAgents(0) = %d, want 0 (unlimited)", got)
	}
}

func TestGetMaxAgents_ZeroEnvMeansUnlimited(t *testing.T) {
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	os.Setenv("ORCH_MAX_AGENTS", "0")

	// Flag not set (-1), env var is 0 = unlimited
	got := GetMaxAgents(-1)
	if got != 0 {
		t.Errorf("GetMaxAgents(-1) with ORCH_MAX_AGENTS=0 = %d, want 0 (unlimited from env)", got)
	}
}
