package bench

import (
	"strings"
	"testing"
)

func TestParseConfig_ValidYAML(t *testing.T) {
	input := `
name: worker-reliability
trials: 5
parallel: 2
scenarios:
  - name: simple-feature
    skill: feature-impl
    task: "Add a hello world endpoint"
    eval: "go test ./..."
    model: opus
    max_reworks: 2
    timeout: 30m
  - name: bug-fix
    skill: systematic-debugging
    task: "Fix sorting bug"
    eval: "make test"
    timeout: 20m
`
	cfg, err := ParseConfig([]byte(input))
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if cfg.Name != "worker-reliability" {
		t.Errorf("Name = %q, want %q", cfg.Name, "worker-reliability")
	}
	if cfg.Trials != 5 {
		t.Errorf("Trials = %d, want 5", cfg.Trials)
	}
	if cfg.Parallel != 2 {
		t.Errorf("Parallel = %d, want 2", cfg.Parallel)
	}
	if len(cfg.Scenarios) != 2 {
		t.Fatalf("len(Scenarios) = %d, want 2", len(cfg.Scenarios))
	}

	s := cfg.Scenarios[0]
	if s.Name != "simple-feature" {
		t.Errorf("Scenarios[0].Name = %q, want %q", s.Name, "simple-feature")
	}
	if s.Skill != "feature-impl" {
		t.Errorf("Scenarios[0].Skill = %q, want %q", s.Skill, "feature-impl")
	}
	if s.MaxReworks != 2 {
		t.Errorf("Scenarios[0].MaxReworks = %d, want 2", s.MaxReworks)
	}
	if s.Timeout != "30m" {
		t.Errorf("Scenarios[0].Timeout = %q, want %q", s.Timeout, "30m")
	}
}

func TestParseConfig_Defaults(t *testing.T) {
	input := `
name: minimal
scenarios:
  - name: test-one
    skill: feature-impl
    task: "do something"
    eval: "echo ok"
`
	cfg, err := ParseConfig([]byte(input))
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if cfg.Trials != 1 {
		t.Errorf("default Trials = %d, want 1", cfg.Trials)
	}
	if cfg.Parallel != 1 {
		t.Errorf("default Parallel = %d, want 1", cfg.Parallel)
	}
	if cfg.Scenarios[0].MaxReworks != 0 {
		t.Errorf("default MaxReworks = %d, want 0", cfg.Scenarios[0].MaxReworks)
	}
	if cfg.Scenarios[0].Timeout != "30m" {
		t.Errorf("default Timeout = %q, want %q", cfg.Scenarios[0].Timeout, "30m")
	}
}

func TestParseConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "missing name",
			input:   "scenarios:\n  - name: x\n    skill: y\n    task: z\n    eval: e",
			wantErr: "name is required",
		},
		{
			name:    "no scenarios",
			input:   "name: test",
			wantErr: "at least one scenario",
		},
		{
			name:    "scenario missing name",
			input:   "name: test\nscenarios:\n  - skill: x\n    task: y\n    eval: e",
			wantErr: "scenario 1: name is required",
		},
		{
			name:    "scenario missing skill",
			input:   "name: test\nscenarios:\n  - name: x\n    task: y\n    eval: e",
			wantErr: "scenario \"x\": skill is required",
		},
		{
			name:    "scenario missing task",
			input:   "name: test\nscenarios:\n  - name: x\n    skill: y\n    eval: e",
			wantErr: "scenario \"x\": task is required",
		},
		{
			name:    "scenario missing eval",
			input:   "name: test\nscenarios:\n  - name: x\n    skill: y\n    task: t",
			wantErr: "scenario \"x\": eval is required",
		},
		{
			name:    "negative trials",
			input:   "name: test\ntrials: -1\nscenarios:\n  - name: x\n    skill: y\n    task: t\n    eval: e",
			wantErr: "trials must be >= 0",
		},
		{
			name:    "negative parallel",
			input:   "name: test\nparallel: -1\nscenarios:\n  - name: x\n    skill: y\n    task: t\n    eval: e",
			wantErr: "parallel must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig([]byte(tt.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseConfig_Thresholds(t *testing.T) {
	input := `
name: with-thresholds
thresholds:
  pass_rate: 0.9
  max_error_rate: 0.05
  max_rework_rate: 0.3
scenarios:
  - name: test
    skill: feature-impl
    task: "do thing"
    eval: "echo ok"
`
	cfg, err := ParseConfig([]byte(input))
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if cfg.Thresholds.PassRate != 0.9 {
		t.Errorf("PassRate = %f, want 0.9", cfg.Thresholds.PassRate)
	}
	if cfg.Thresholds.MaxErrorRate != 0.05 {
		t.Errorf("MaxErrorRate = %f, want 0.05", cfg.Thresholds.MaxErrorRate)
	}
	if cfg.Thresholds.MaxReworkRate != 0.3 {
		t.Errorf("MaxReworkRate = %f, want 0.3", cfg.Thresholds.MaxReworkRate)
	}
}

func TestParseConfig_ThresholdDefaults(t *testing.T) {
	input := `
name: no-thresholds
scenarios:
  - name: test
    skill: feature-impl
    task: "do thing"
    eval: "echo ok"
`
	cfg, err := ParseConfig([]byte(input))
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if cfg.Thresholds.PassRate != 0.8 {
		t.Errorf("default PassRate = %f, want 0.8", cfg.Thresholds.PassRate)
	}
	if cfg.Thresholds.MaxErrorRate != 0.1 {
		t.Errorf("default MaxErrorRate = %f, want 0.1", cfg.Thresholds.MaxErrorRate)
	}
	if cfg.Thresholds.MaxReworkRate != 0.5 {
		t.Errorf("default MaxReworkRate = %f, want 0.5", cfg.Thresholds.MaxReworkRate)
	}
}

func TestParseConfigFile_NotFound(t *testing.T) {
	_, err := ParseConfigFile("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
