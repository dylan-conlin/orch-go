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
	t.Setenv("ORCH_DEBUG", "")

	useBDSandboxMode = true

	args := prependSandboxArg([]string{"show", "orch-go-123", "--json"})
	if len(args) < 3 || args[0] != "--sandbox" || args[1] != "--quiet" || args[2] != "show" {
		t.Fatalf("prependSandboxArg returned unexpected args: %v", args)
	}

	already := prependSandboxArg([]string{"--sandbox", "show", "orch-go-123", "--json"})
	if !hasCLIArg(already, "--sandbox") {
		t.Fatalf("prependSandboxArg should include --sandbox: %v", already)
	}
	sandboxCount := 0
	for _, arg := range already {
		if arg == "--sandbox" {
			sandboxCount++
		}
	}
	if sandboxCount != 1 {
		t.Fatalf("prependSandboxArg should not duplicate --sandbox: %v", already)
	}
	if !hasCLIArg(already, "--quiet") {
		t.Fatalf("prependSandboxArg should include --quiet when ORCH_DEBUG is not set: %v", already)
	}

	useBDSandboxMode = false
	unchanged := prependSandboxArg([]string{"show", "orch-go-123", "--json"})
	if len(unchanged) == 0 || unchanged[0] != "--quiet" {
		t.Fatalf("prependSandboxArg should still include --quiet when sandbox disabled: %v", unchanged)
	}
}

func TestPrependSandboxArg_DisablesQuietWhenDebugEnabled(t *testing.T) {
	original := useBDSandboxMode
	t.Cleanup(func() {
		useBDSandboxMode = original
	})

	t.Setenv("ORCH_DEBUG", "1")
	useBDSandboxMode = true

	args := prependSandboxArg([]string{"show", "orch-go-123", "--json"})
	if hasCLIArg(args, "--quiet") || hasCLIArg(args, "-q") {
		t.Fatalf("prependSandboxArg should not add --quiet when ORCH_DEBUG is set: %v", args)
	}
}
