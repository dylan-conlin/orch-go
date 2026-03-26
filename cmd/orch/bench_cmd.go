package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/bench"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/spf13/cobra"
)

var (
	benchOutputDir    string
	benchDryRun       bool
	benchModelOveride string
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Benchmark execution engine",
	Long: `Run benchmark scenarios using spawn/wait/eval/rework primitives.

Model aliases (opus, sonnet, haiku, gpt-5, etc.) are resolved to full model
IDs at config load time. Use --model to override the model for all scenarios.`,
}

var benchRunCmd = &cobra.Command{
	Use:   "run <config.yaml>",
	Short: "Execute a benchmark suite from a YAML config",
	Long: `Execute benchmark scenarios defined in a YAML config file.

Each scenario spawns an agent, waits for completion, runs an eval command,
and optionally reworks on failure. Results are written to JSONL + summary JSON.

Model aliases are resolved automatically:
  opus → claude-opus-4-5-20251101
  sonnet → claude-sonnet-4-5-20250929
  gpt-5 → gpt-5.2
  etc.

Example config:
  name: worker-reliability
  default_model: opus
  trials: 3
  parallel: 2
  scenarios:
    - name: simple-feature
      skill: feature-impl
      task: "Add hello endpoint"
      eval: "go test ./..."
      model: opus
      max_reworks: 1
      timeout: 30m

Examples:
  orch bench run benchmarks/reliability.yaml
  orch bench run benchmarks/reliability.yaml --output ./results
  orch bench run benchmarks/reliability.yaml --dry-run
  orch bench run benchmarks/reliability.yaml --model sonnet`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBench(args[0])
	},
}

var benchValidateCmd = &cobra.Command{
	Use:   "validate <config.yaml>",
	Short: "Validate a benchmark config without executing",
	Long:  `Parse and validate a benchmark YAML config. Reports errors or prints the resolved configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBenchValidate(args[0])
	},
}

var benchListCmd = &cobra.Command{
	Use:   "list [directory]",
	Short: "List benchmark suite files",
	Long:  `Discover and list benchmark YAML files in a directory (default: ./benchmarks).`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "benchmarks"
		if len(args) > 0 {
			dir = args[0]
		}
		return runBenchList(dir)
	},
}

func init() {
	benchRunCmd.Flags().StringVar(&benchOutputDir, "output", "", "Output directory for results (default: ./bench-results/<timestamp>)")
	benchRunCmd.Flags().BoolVar(&benchDryRun, "dry-run", false, "Parse and validate config without executing")
	benchRunCmd.Flags().StringVar(&benchModelOveride, "model", "", "Override model for all scenarios (accepts aliases: opus, sonnet, etc.)")
	benchCmd.AddCommand(benchRunCmd)
	benchCmd.AddCommand(benchValidateCmd)
	benchCmd.AddCommand(benchListCmd)
}

func loadAndResolveConfig(configPath string) (*bench.Config, error) {
	cfg, err := bench.ParseConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Apply CLI model override if provided
	if benchModelOveride != "" {
		cfg.ApplyModelOverride(benchModelOveride)
	} else {
		cfg.ResolveModels()
	}

	return cfg, nil
}

func runBench(configPath string) error {
	cfg, err := loadAndResolveConfig(configPath)
	if err != nil {
		return err
	}

	totalRuns := len(cfg.Scenarios) * cfg.Trials
	fmt.Printf("Benchmark: %s\n", cfg.Name)
	fmt.Printf("  Scenarios: %d\n", len(cfg.Scenarios))
	fmt.Printf("  Trials: %d\n", cfg.Trials)
	fmt.Printf("  Parallel: %d\n", cfg.Parallel)
	fmt.Printf("  Total runs: %d\n", totalRuns)

	if benchDryRun {
		printDryRun(cfg)
		return nil
	}

	// Resolve output directory
	outDir := benchOutputDir
	if outDir == "" {
		outDir = filepath.Join("bench-results", time.Now().Format("20060102-150405"))
	}

	projectDir, _ := os.Getwd()

	engine := &bench.Engine{
		SpawnFn:  makeBenchSpawnFn(projectDir),
		WaitFn:   makeBenchWaitFn(),
		EvalFn:   makeBenchEvalFn(projectDir),
		ReworkFn: makeBenchReworkFn(projectDir),
	}

	fmt.Printf("\nStarting benchmark run...\n")
	result, err := engine.Run(cfg)
	if err != nil {
		return fmt.Errorf("benchmark failed: %w", err)
	}

	// Write raw results
	resultsPath := filepath.Join(outDir, "results.jsonl")
	summaryPath := filepath.Join(outDir, "summary.json")

	if err := result.WriteJSONL(resultsPath); err != nil {
		return fmt.Errorf("writing results: %w", err)
	}
	if err := result.WriteSummary(summaryPath); err != nil {
		return fmt.Errorf("writing summary: %w", err)
	}

	// Write config snapshot for reproducibility
	configSnapshotPath := filepath.Join(outDir, "config.yaml")
	if err := bench.WriteConfigSnapshot(cfg, configSnapshotPath); err != nil {
		return fmt.Errorf("writing config snapshot: %w", err)
	}

	// Collect run metadata
	runID := filepath.Base(outDir)
	meta := bench.RunMetadata{
		RunID:     runID,
		GitSHA:    gitOutput(projectDir, "rev-parse", "HEAD"),
		GitBranch: gitOutput(projectDir, "rev-parse", "--abbrev-ref", "HEAD"),
		StartedAt: result.StartedAt,
		ConfigRef: "config.yaml",
	}

	// Generate and write report
	report := bench.GenerateReport(result, cfg, meta)
	reportPath := filepath.Join(outDir, "report.json")
	if err := bench.WriteReport(report, reportPath); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	// Print human-readable report
	fmt.Println()
	fmt.Print(bench.FormatReport(report))
	fmt.Printf("\nArtifacts:\n")
	fmt.Printf("  %s/\n", outDir)
	fmt.Printf("    results.jsonl    — per-trial JSONL\n")
	fmt.Printf("    summary.json     — aggregate stats\n")
	fmt.Printf("    report.json      — full report with verdicts\n")
	fmt.Printf("    config.yaml      — config snapshot\n")

	// Log event
	logger := events.NewLogger(events.DefaultLogPath())
	_ = logger.Log(events.Event{
		Type:      "bench.complete",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"name":      cfg.Name,
			"total":     report.Summary.Total,
			"passed":    report.Summary.Passed,
			"failed":    report.Summary.Failed,
			"pass_rate": report.Summary.PassRate,
			"reworks":   report.Summary.TotalReworks,
			"duration":  result.Duration.Seconds(),
			"verdict":   report.Verdict.Overall,
		},
	})

	return nil
}

func printDryRun(cfg *bench.Config) {
	fmt.Println("\n--- DRY RUN (no agents spawned) ---")
	for _, s := range cfg.Scenarios {
		resolved := "(default)"
		if s.ResolvedModel.ModelID != "" {
			resolved = s.ResolvedModel.Format()
		}
		fmt.Printf("\n  %s:\n", s.Name)
		fmt.Printf("    skill:    %s\n", s.Skill)
		fmt.Printf("    model:    %s → %s\n", displayModelInput(s.Model, cfg.DefaultModel), resolved)
		fmt.Printf("    eval:     %s\n", s.Eval)
		fmt.Printf("    reworks:  %d\n", s.MaxReworks)
		fmt.Printf("    timeout:  %s\n", s.Timeout)
	}
	fmt.Println()
	fmt.Printf("Thresholds:\n")
	fmt.Printf("    pass_rate:      %.0f%%\n", cfg.Thresholds.PassRate*100)
	fmt.Printf("    max_error_rate: %.0f%%\n", cfg.Thresholds.MaxErrorRate*100)
	fmt.Printf("    max_rework_rate:%.0f%%\n", cfg.Thresholds.MaxReworkRate*100)
}

// displayModelInput returns the model alias as written, or notes the default source.
func displayModelInput(scenarioModel, defaultModel string) string {
	if scenarioModel != "" {
		return scenarioModel
	}
	if defaultModel != "" {
		return defaultModel + " (suite default)"
	}
	return "(none)"
}

func runBenchValidate(configPath string) error {
	cfg, err := loadAndResolveConfig(configPath)
	if err != nil {
		return err
	}

	fmt.Printf("Config: %s ✓\n", configPath)
	fmt.Printf("  Name:      %s\n", cfg.Name)
	fmt.Printf("  Scenarios: %d\n", len(cfg.Scenarios))
	fmt.Printf("  Trials:    %d\n", cfg.Trials)
	fmt.Printf("  Parallel:  %d\n", cfg.Parallel)
	if cfg.DefaultModel != "" {
		spec := model.Resolve(cfg.DefaultModel)
		fmt.Printf("  Default model: %s → %s\n", cfg.DefaultModel, spec.Format())
	}

	fmt.Println("\nScenarios:")
	for _, s := range cfg.Scenarios {
		resolved := "(default)"
		if s.ResolvedModel.ModelID != "" {
			resolved = s.ResolvedModel.Format()
		}
		fmt.Printf("  %-20s model=%s  eval=%q  timeout=%s\n",
			s.Name, resolved, s.Eval, s.Timeout)
	}
	return nil
}

func runBenchList(dir string) error {
	suites, err := bench.ListSuites(dir)
	if err != nil {
		return err
	}

	if len(suites) == 0 {
		fmt.Printf("No benchmark suites found in %s/\n", dir)
		return nil
	}

	fmt.Printf("Benchmark suites in %s/:\n\n", dir)
	for _, s := range suites {
		if s.Error != "" {
			fmt.Printf("  %-30s  ✗ %s\n", filepath.Base(s.Path), s.Error)
			continue
		}
		fmt.Printf("  %-30s  %s (%d scenarios × %d trials)\n",
			filepath.Base(s.Path), s.Name, s.Scenarios, s.Trials)
	}
	return nil
}

// gitOutput runs a git command and returns trimmed stdout, or empty string on error.
func gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// makeBenchSpawnFn returns a SpawnFunc that shells out to `orch spawn`.
func makeBenchSpawnFn(projectDir string) bench.SpawnFunc {
	return func(skill, task, modelSpec string) (string, error) {
		args := []string{"spawn", skill, task,
			"--bypass-triage",
			"--reason", "benchmark-execution-engine",
			"--light",
			"--no-track",
		}
		if modelSpec != "" {
			args = append(args, "--model", modelSpec)
		}

		cmd := exec.Command("orch", args...)
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("spawn failed: %w\noutput: %s", err, output)
		}

		// Parse beads ID from spawn output (last line typically has the ID)
		beadsID := extractBeadsID(string(output))
		if beadsID == "" {
			return "", fmt.Errorf("could not extract beads ID from spawn output:\n%s", output)
		}
		return beadsID, nil
	}
}

// makeBenchWaitFn returns a WaitFunc that shells out to `orch wait`.
func makeBenchWaitFn() bench.WaitFunc {
	return func(beadsID string, timeout time.Duration) error {
		args := []string{"wait", beadsID,
			"--timeout", timeout.String(),
			"-q",
		}
		cmd := exec.Command("orch", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("wait failed: %w\noutput: %s", err, output)
		}
		return nil
	}
}

// makeBenchEvalFn returns an EvalFunc that runs the eval command in a shell.
func makeBenchEvalFn(projectDir string) bench.EvalFunc {
	return func(evalCmd string) (int, string, error) {
		cmd := exec.Command("sh", "-c", evalCmd)
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode(), outputStr, nil
			}
			return -1, outputStr, err
		}
		return 0, outputStr, nil
	}
}

// makeBenchReworkFn returns a ReworkFunc that shells out to `orch rework`.
func makeBenchReworkFn(projectDir string) bench.ReworkFunc {
	return func(beadsID, feedback string) error {
		args := []string{"rework", beadsID, feedback,
			"--bypass-triage",
			"--force",
		}
		cmd := exec.Command("orch", args...)
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("rework failed: %w\noutput: %s", err, output)
		}
		return nil
	}
}

// extractBeadsID parses a beads ID from spawn output.
// Looks for patterns like "orch-go-XXXXX" or "Created: orch-go-XXXXX".
func extractBeadsID(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		// Look for beads ID pattern in each word
		for _, word := range strings.Fields(line) {
			cleaned := strings.Trim(word, "\"'(),;:")
			if isBeadsID(cleaned) {
				return cleaned
			}
		}
	}
	return ""
}

// isBeadsID checks if a string looks like a beads ID (e.g., "orch-go-abc12").
func isBeadsID(s string) bool {
	// Pattern: project-name-XXXXX where XXXXX is alphanumeric
	parts := strings.Split(s, "-")
	if len(parts) < 3 {
		return false
	}
	last := parts[len(parts)-1]
	if len(last) < 4 || len(last) > 6 {
		return false
	}
	for _, c := range last {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}
