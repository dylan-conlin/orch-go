package gates

import (
	"fmt"
	"testing"
)

func mockProvider(score float64, grade string) HealthScoreProvider {
	return func() (float64, string, error) {
		return score, grade, nil
	}
}

func mockProviderError() HealthScoreProvider {
	return func() (float64, string, error) {
		return 0, "", fmt.Errorf("no snapshots")
	}
}

func TestCheckHealthScore_AdvisoryBelowFloor(t *testing.T) {
	err := CheckHealthScore("feature-impl", false, false, mockProvider(40, "F"))
	if err != nil {
		t.Fatalf("below-floor should warn not block, got error: %v", err)
	}
}

func TestCheckHealthScore_AllowsAboveFloor(t *testing.T) {
	err := CheckHealthScore("feature-impl", false, false, mockProvider(71, "C"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckHealthScore_AllowsAtFloor(t *testing.T) {
	err := CheckHealthScore("feature-impl", false, false, mockProvider(65, "C"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckHealthScore_ExemptSkills(t *testing.T) {
	for _, skill := range []string{"architect", "investigation", "codebase-audit"} {
		err := CheckHealthScore(skill, false, false, mockProvider(30, "F"))
		if err != nil {
			t.Fatalf("skill %s should be exempt, got error: %v", skill, err)
		}
	}
}

func TestCheckHealthScore_DaemonDrivenBypasses(t *testing.T) {
	err := CheckHealthScore("feature-impl", true, false, mockProvider(30, "F"))
	if err != nil {
		t.Fatalf("daemon-driven should bypass, got error: %v", err)
	}
}

func TestCheckHealthScore_SkipFlag(t *testing.T) {
	err := CheckHealthScore("feature-impl", false, true, mockProvider(30, "F"))
	if err != nil {
		t.Fatalf("skip flag should bypass, got error: %v", err)
	}
}

func TestCheckHealthScore_NilProvider(t *testing.T) {
	err := CheckHealthScore("feature-impl", false, false, nil)
	if err != nil {
		t.Fatalf("nil provider should pass, got error: %v", err)
	}
}

func TestCheckHealthScore_ProviderError(t *testing.T) {
	err := CheckHealthScore("feature-impl", false, false, mockProviderError())
	if err != nil {
		t.Fatalf("provider error should not block, got error: %v", err)
	}
}

func TestCheckHealthScore_SystematicDebuggingAdvisory(t *testing.T) {
	err := CheckHealthScore("systematic-debugging", false, false, mockProvider(40, "F"))
	if err != nil {
		t.Fatalf("below-floor should warn not block, got error: %v", err)
	}
}
