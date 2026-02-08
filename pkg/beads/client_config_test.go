package beads

import "testing"

func TestResolveBDSubprocessLimit_Default(t *testing.T) {
	t.Setenv(bdSubprocessLimitEnvVar, "")

	got := resolveBDSubprocessLimit()
	if got != defaultMaxBDSubprocess {
		t.Fatalf("resolveBDSubprocessLimit() = %d, want %d", got, defaultMaxBDSubprocess)
	}
}

func TestResolveBDSubprocessLimit_FromEnv(t *testing.T) {
	t.Setenv(bdSubprocessLimitEnvVar, "21")

	got := resolveBDSubprocessLimit()
	if got != 21 {
		t.Fatalf("resolveBDSubprocessLimit() = %d, want 21", got)
	}
}

func TestResolveBDSubprocessLimit_InvalidEnvFallsBack(t *testing.T) {
	t.Setenv(bdSubprocessLimitEnvVar, "abc")

	got := resolveBDSubprocessLimit()
	if got != defaultMaxBDSubprocess {
		t.Fatalf("resolveBDSubprocessLimit() = %d, want %d", got, defaultMaxBDSubprocess)
	}
}

func TestResolveBDSandboxMode_DefaultEnabled(t *testing.T) {
	t.Setenv(bdDisableSandboxEnvVar, "")

	if !resolveBDSandboxMode() {
		t.Fatal("resolveBDSandboxMode() = false, want true")
	}
}

func TestResolveBDSandboxMode_DisabledByEnv(t *testing.T) {
	t.Setenv(bdDisableSandboxEnvVar, "true")

	if resolveBDSandboxMode() {
		t.Fatal("resolveBDSandboxMode() = true, want false")
	}
}

func TestPrependSandboxArg(t *testing.T) {
	original := useBDSandboxMode
	t.Cleanup(func() {
		useBDSandboxMode = original
	})

	useBDSandboxMode = true

	args := prependSandboxArg([]string{"show", "orch-go-123", "--json"})
	if len(args) < 2 || args[0] != "--sandbox" || args[1] != "show" {
		t.Fatalf("prependSandboxArg returned unexpected args: %v", args)
	}

	already := prependSandboxArg([]string{"--sandbox", "show", "orch-go-123", "--json"})
	if len(already) == 0 || already[0] != "--sandbox" {
		t.Fatalf("prependSandboxArg should not duplicate --sandbox: %v", already)
	}

	useBDSandboxMode = false
	unchanged := prependSandboxArg([]string{"show", "orch-go-123", "--json"})
	if len(unchanged) == 0 || unchanged[0] != "show" {
		t.Fatalf("prependSandboxArg should keep args unchanged when sandbox disabled: %v", unchanged)
	}
}
