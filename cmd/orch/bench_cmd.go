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
	"github.com/spf13/cobra"
)

var (
	benchOutputDir string
	benchDryRun    bool
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Benchmark execution engine",
	Long:  `Run benchmark scenarios using spawn/wait/eval/rework primitives.`,
}

var benchRunCmd = &cobra.Command{
	Use:   "run <config.yaml>",
	Short: "Execute a benchmark suite from a YAML config",
	Long: `Execute benchmark scenarios defined in a YAML config file.

Each scenario spawns an agent, waits for completion, runs an eval command,
and optionally reworks on failure. Results are written to JSONL + summary JSON.

Example config:
  name: worker-reliability
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
  orch bench run benchmarks/reliability.yaml --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBench(args[0])
	},
}

func init() {
	benchRunCmd.Flags().StringVar(&benchOutputDir, "output", "", "Output directory for results (default: ./bench-results/<timestamp>)")
	benchRunCmd.Flags().BoolVar(&benchDryRun, "dry-run", false, "Parse and validate config without executing")
	benchCmd.AddCommand(benchRunCmd)
}

func runBench(configPath string) error {
	cfg, err := bench.ParseConfigFile(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	totalRuns := len(cfg.Scenarios) * cfg.Trials
	fmt.Printf("Benchmark: %s\n", cfg.Name)
	fmt.Printf("  Scenarios: %d\n", len(cfg.Scenarios))
	fmt.Printf("  Trials: %d\n", cfg.Trials)
	fmt.Printf("  Parallel: %d\n", cfg.Parallel)
	fmt.Printf("  Total runs: %d\n", totalRuns)

	if benchDryRun {
		fmt.Println("\n--- DRY RUN (no agents spawned) ---")
		for _, s := range cfg.Scenarios {
			fmt.Printf("  %s: skill=%s model=%s eval=%q reworks=%d timeout=%s\n",
				s.Name, s.Skill, s.Model, s.Eval, s.MaxReworks, s.Timeout)
		}
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

	// Write results
	resultsPath := filepath.Join(outDir, "results.jsonl")
	summaryPath := filepath.Join(outDir, "summary.json")

	if err := result.WriteJSONL(resultsPath); err != nil {
		return fmt.Errorf("writing results: %w", err)
	}
	if err := result.WriteSummary(summaryPath); err != nil {
		return fmt.Errorf("writing summary: %w", err)
	}

	// Print summary
	s := result.Summary()
	fmt.Printf("\n--- Results ---\n")
	fmt.Printf("  Total: %d  Pass: %d  Fail: %d  Error: %d  Timeout: %d\n",
		s.Total, s.Passed, s.Failed, s.Errors, s.Timeouts)
	fmt.Printf("  Pass rate: %.0f%%\n", s.PassRate*100)
	fmt.Printf("  Total reworks: %d\n", s.TotalReworks)
	fmt.Printf("  Avg duration: %s\n", s.AvgDuration.Round(time.Second))
	fmt.Printf("  Results: %s\n", resultsPath)
	fmt.Printf("  Summary: %s\n", summaryPath)

	// Log event
	logger := events.NewLogger(events.DefaultLogPath())
	_ = logger.Log(events.Event{
		Type:      "bench.complete",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"name":      cfg.Name,
			"total":     s.Total,
			"passed":    s.Passed,
			"failed":    s.Failed,
			"pass_rate": s.PassRate,
			"reworks":   s.TotalReworks,
			"duration":  result.Duration.Seconds(),
		},
	})

	return nil
}

// makeBenchSpawnFn returns a SpawnFunc that shells out to `orch spawn`.
func makeBenchSpawnFn(projectDir string) bench.SpawnFunc {
	return func(skill, task, model string) (string, error) {
		args := []string{"spawn", skill, task,
			"--bypass-triage",
			"--reason", "benchmark-execution-engine",
			"--light",
			"--no-track",
		}
		if model != "" {
			args = append(args, "--model", model)
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
