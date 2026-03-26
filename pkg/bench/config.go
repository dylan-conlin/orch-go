// Package bench provides a benchmark execution engine that composes
// spawn/wait/eval/rework primitives into repeatable benchmark runs.
package bench

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/model"
	"gopkg.in/yaml.v3"
)

// Config defines a benchmark suite with scenarios to execute.
type Config struct {
	Name         string     `yaml:"name"`
	Trials       int        `yaml:"trials"`
	Parallel     int        `yaml:"parallel"`
	DefaultModel string     `yaml:"default_model,omitempty"`
	Scenarios    []Scenario `yaml:"scenarios"`
	Thresholds   Thresholds `yaml:"thresholds,omitempty"`
}

// Thresholds defines configurable verdict boundaries for benchmark evaluation.
type Thresholds struct {
	PassRate      float64 `yaml:"pass_rate" json:"pass_rate"`             // minimum pass rate for PASS verdict (default: 0.8)
	MaxErrorRate  float64 `yaml:"max_error_rate" json:"max_error_rate"`   // maximum error rate before FAIL (default: 0.1)
	MaxReworkRate float64 `yaml:"max_rework_rate" json:"max_rework_rate"` // warn if rework rate exceeds this (default: 0.5)
}

// Scenario defines a single benchmark case: a skill+task to spawn,
// an eval command to judge success, and optional rework iteration.
type Scenario struct {
	Name       string `yaml:"name"`
	Skill      string `yaml:"skill"`
	Task       string `yaml:"task"`
	Eval       string `yaml:"eval"`
	Model      string `yaml:"model,omitempty"`
	MaxReworks int    `yaml:"max_reworks,omitempty"`
	Timeout    string `yaml:"timeout,omitempty"`

	// ResolvedModel is populated by ResolveModels — the full model spec after alias resolution.
	ResolvedModel model.ModelSpec `yaml:"-" json:"resolved_model,omitempty"`
}

// ParseConfig parses YAML bytes into a validated Config.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	return &cfg, nil
}

// ParseConfigFile reads and parses a YAML config file.
func ParseConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	return ParseConfig(data)
}

func (c *Config) applyDefaults() {
	if c.Trials == 0 {
		c.Trials = 1
	}
	if c.Parallel == 0 {
		c.Parallel = 1
	}
	for i := range c.Scenarios {
		if c.Scenarios[i].Timeout == "" {
			c.Scenarios[i].Timeout = "30m"
		}
	}
	if c.Thresholds.PassRate == 0 {
		c.Thresholds.PassRate = 0.8
	}
	if c.Thresholds.MaxErrorRate == 0 {
		c.Thresholds.MaxErrorRate = 0.1
	}
	if c.Thresholds.MaxReworkRate == 0 {
		c.Thresholds.MaxReworkRate = 0.5
	}
}

func (c *Config) validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Trials < 0 {
		return fmt.Errorf("trials must be >= 0 (got %d)", c.Trials)
	}
	if c.Parallel < 0 {
		return fmt.Errorf("parallel must be >= 0 (got %d)", c.Parallel)
	}
	if len(c.Scenarios) == 0 {
		return fmt.Errorf("at least one scenario is required")
	}

	seen := make(map[string]bool)
	for i, s := range c.Scenarios {
		if s.Name == "" {
			return fmt.Errorf("scenario %d: name is required", i+1)
		}
		if seen[s.Name] {
			return fmt.Errorf("scenario %q: duplicate name", s.Name)
		}
		seen[s.Name] = true
		if s.Skill == "" {
			return fmt.Errorf("scenario %q: skill is required", s.Name)
		}
		if s.Task == "" {
			return fmt.Errorf("scenario %q: task is required", s.Name)
		}
		if s.Eval == "" {
			return fmt.Errorf("scenario %q: eval is required", s.Name)
		}
		if s.Timeout != "" {
			if _, err := time.ParseDuration(s.Timeout); err != nil {
				return fmt.Errorf("scenario %q: invalid timeout %q: %w", s.Name, s.Timeout, err)
			}
		}
	}
	return nil
}

// ResolveModels resolves model aliases in all scenarios using pkg/model.Resolve.
// If a scenario has no model set, the suite's DefaultModel is used.
// After this call, each Scenario's ResolvedModel field is populated.
func (c *Config) ResolveModels() {
	for i := range c.Scenarios {
		raw := c.Scenarios[i].Model
		if raw == "" {
			raw = c.DefaultModel
		}
		if raw != "" {
			spec := model.Resolve(raw)
			c.Scenarios[i].ResolvedModel = spec
		}
	}
}

// ApplyModelOverride sets the model for all scenarios and re-resolves aliases.
func (c *Config) ApplyModelOverride(modelName string) {
	for i := range c.Scenarios {
		c.Scenarios[i].Model = modelName
	}
	c.ResolveModels()
}

// ListSuites finds benchmark YAML files in the given directory.
func ListSuites(dir string) ([]SuiteInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var suites []SuiteInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := dir + "/" + name
		cfg, err := ParseConfigFile(path)
		if err != nil {
			suites = append(suites, SuiteInfo{Path: path, Error: err.Error()})
			continue
		}
		suites = append(suites, SuiteInfo{
			Path:      path,
			Name:      cfg.Name,
			Scenarios: len(cfg.Scenarios),
			Trials:    cfg.Trials,
		})
	}
	return suites, nil
}

// SuiteInfo summarizes a discovered benchmark suite file.
type SuiteInfo struct {
	Path      string `json:"path"`
	Name      string `json:"name,omitempty"`
	Scenarios int    `json:"scenarios,omitempty"`
	Trials    int    `json:"trials,omitempty"`
	Error     string `json:"error,omitempty"`
}
