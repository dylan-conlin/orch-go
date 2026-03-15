package daemonconfig

import (
	"testing"
)

func TestComplianceLevelString(t *testing.T) {
	tests := []struct {
		level ComplianceLevel
		want  string
	}{
		{ComplianceStrict, "strict"},
		{ComplianceStandard, "standard"},
		{ComplianceRelaxed, "relaxed"},
		{ComplianceAutonomous, "autonomous"},
		{ComplianceLevel(99), "strict"}, // unknown defaults to strict
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("ComplianceLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestParseComplianceLevel(t *testing.T) {
	tests := []struct {
		input string
		want  ComplianceLevel
		ok    bool
	}{
		{"strict", ComplianceStrict, true},
		{"standard", ComplianceStandard, true},
		{"relaxed", ComplianceRelaxed, true},
		{"autonomous", ComplianceAutonomous, true},
		{"Strict", ComplianceStrict, true},
		{"STANDARD", ComplianceStandard, true},
		{"", ComplianceStrict, false},
		{"invalid", ComplianceStrict, false},
	}
	for _, tt := range tests {
		got, ok := ParseComplianceLevel(tt.input)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ParseComplianceLevel(%q) = (%v, %v), want (%v, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestComplianceConfigResolve_Default(t *testing.T) {
	cfg := ComplianceConfig{Default: ComplianceStandard}
	if got := cfg.Resolve("feature-impl", "opus"); got != ComplianceStandard {
		t.Errorf("Resolve with only default = %v, want standard", got)
	}
}

func TestComplianceConfigResolve_SkillOverride(t *testing.T) {
	cfg := ComplianceConfig{
		Default: ComplianceStrict,
		Skills:  map[string]ComplianceLevel{"investigation": ComplianceRelaxed},
	}
	if got := cfg.Resolve("investigation", "opus"); got != ComplianceRelaxed {
		t.Errorf("Resolve with skill override = %v, want relaxed", got)
	}
	// Unmatched skill falls through to default
	if got := cfg.Resolve("feature-impl", "opus"); got != ComplianceStrict {
		t.Errorf("Resolve without skill override = %v, want strict", got)
	}
}

func TestComplianceConfigResolve_ModelOverride(t *testing.T) {
	cfg := ComplianceConfig{
		Default: ComplianceStrict,
		Models:  map[string]ComplianceLevel{"opus": ComplianceStandard},
	}
	if got := cfg.Resolve("feature-impl", "opus"); got != ComplianceStandard {
		t.Errorf("Resolve with model override = %v, want standard", got)
	}
	// Unmatched model falls through to default
	if got := cfg.Resolve("feature-impl", "sonnet"); got != ComplianceStrict {
		t.Errorf("Resolve without model override = %v, want strict", got)
	}
}

func TestComplianceConfigResolve_SkillBeatsModel(t *testing.T) {
	cfg := ComplianceConfig{
		Default: ComplianceStrict,
		Skills:  map[string]ComplianceLevel{"architect": ComplianceStrict},
		Models:  map[string]ComplianceLevel{"opus": ComplianceRelaxed},
	}
	// Skill override takes precedence over model override
	if got := cfg.Resolve("architect", "opus"); got != ComplianceStrict {
		t.Errorf("Resolve skill > model = %v, want strict", got)
	}
}

func TestComplianceConfigResolve_ComboHighestPrecedence(t *testing.T) {
	cfg := ComplianceConfig{
		Default: ComplianceStrict,
		Skills:  map[string]ComplianceLevel{"feature-impl": ComplianceStandard},
		Models:  map[string]ComplianceLevel{"opus": ComplianceStandard},
		Combos:  map[string]ComplianceLevel{"opus+feature-impl": ComplianceAutonomous},
	}
	if got := cfg.Resolve("feature-impl", "opus"); got != ComplianceAutonomous {
		t.Errorf("Resolve combo > skill > model = %v, want autonomous", got)
	}
	// Non-matching combo falls through to skill
	if got := cfg.Resolve("feature-impl", "sonnet"); got != ComplianceStandard {
		t.Errorf("Resolve without combo match = %v, want standard (from skill)", got)
	}
}

func TestComplianceConfigResolve_ZeroValue(t *testing.T) {
	// Zero-value ComplianceConfig should resolve to Strict (default)
	cfg := ComplianceConfig{}
	if got := cfg.Resolve("feature-impl", "opus"); got != ComplianceStrict {
		t.Errorf("Resolve zero-value config = %v, want strict", got)
	}
}

func TestComplianceConfigResolve_NilMaps(t *testing.T) {
	cfg := ComplianceConfig{Default: ComplianceRelaxed}
	// Should not panic with nil maps
	if got := cfg.Resolve("anything", "anything"); got != ComplianceRelaxed {
		t.Errorf("Resolve with nil maps = %v, want relaxed", got)
	}
}

func TestDeriveVerificationThreshold(t *testing.T) {
	tests := []struct {
		level ComplianceLevel
		want  int
	}{
		{ComplianceStrict, 3},
		{ComplianceStandard, 8},
		{ComplianceRelaxed, 20},
		{ComplianceAutonomous, 0},
	}
	for _, tt := range tests {
		if got := DeriveVerificationThreshold(tt.level); got != tt.want {
			t.Errorf("DeriveVerificationThreshold(%v) = %d, want %d", tt.level, got, tt.want)
		}
	}
}

func TestDeriveInvariantThreshold(t *testing.T) {
	tests := []struct {
		level ComplianceLevel
		want  int
	}{
		{ComplianceStrict, 3},
		{ComplianceStandard, 5},
		{ComplianceRelaxed, 10},
		{ComplianceAutonomous, 0},
	}
	for _, tt := range tests {
		if got := DeriveInvariantThreshold(tt.level); got != tt.want {
			t.Errorf("DeriveInvariantThreshold(%v) = %d, want %d", tt.level, got, tt.want)
		}
	}
}

func TestDeriveArchitectEscalationEnabled(t *testing.T) {
	tests := []struct {
		level ComplianceLevel
		want  bool
	}{
		{ComplianceStrict, true},
		{ComplianceStandard, true},
		{ComplianceRelaxed, false},
		{ComplianceAutonomous, false},
	}
	for _, tt := range tests {
		if got := DeriveArchitectEscalationEnabled(tt.level); got != tt.want {
			t.Errorf("DeriveArchitectEscalationEnabled(%v) = %v, want %v", tt.level, got, tt.want)
		}
	}
}

func TestDeriveSynthesisRequired(t *testing.T) {
	tests := []struct {
		level ComplianceLevel
		want  bool
	}{
		{ComplianceStrict, true},
		{ComplianceStandard, true},
		{ComplianceRelaxed, false},
		{ComplianceAutonomous, false},
	}
	for _, tt := range tests {
		if got := DeriveSynthesisRequired(tt.level); got != tt.want {
			t.Errorf("DeriveSynthesisRequired(%v) = %v, want %v", tt.level, got, tt.want)
		}
	}
}

func TestDerivePhaseEnforcement(t *testing.T) {
	tests := []struct {
		level ComplianceLevel
		want  string
	}{
		{ComplianceStrict, "required"},
		{ComplianceStandard, "required"},
		{ComplianceRelaxed, "advisory"},
		{ComplianceAutonomous, "advisory"},
	}
	for _, tt := range tests {
		if got := DerivePhaseEnforcement(tt.level); got != tt.want {
			t.Errorf("DerivePhaseEnforcement(%v) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestDeriveTriggerBudget(t *testing.T) {
	tests := []struct {
		level ComplianceLevel
		want  int
	}{
		{ComplianceStrict, 10},
		{ComplianceStandard, 10},
		{ComplianceRelaxed, 15},
		{ComplianceAutonomous, 20},
	}
	for _, tt := range tests {
		if got := DeriveTriggerBudget(tt.level); got != tt.want {
			t.Errorf("DeriveTriggerBudget(%v) = %d, want %d", tt.level, got, tt.want)
		}
	}
}
