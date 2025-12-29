// Package main provides the CLI entry point for orch-go.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/experiment"
	"github.com/spf13/cobra"
)

var experimentCmd = &cobra.Command{
	Use:   "experiment",
	Short: "Run behavioral experiments on agent context/attention",
	Long: `Run behavioral experiments to understand how agents process context.

This command helps you systematically test hypotheses about agent behavior,
such as whether constraint position or framing affects recognition.

Subcommands:
  create   - Scaffold a new experiment
  list     - Show all experiments with status
  run      - Execute one condition of an experiment
  status   - Show runs completed per condition
  analyze  - Generate/update analysis.md with results

Example workflow:
  1. orch experiment create context-attention --hypothesis "Position affects recognition"
  2. Edit .orch/experiments/<date>-context-attention/experiment.yaml to define conditions
  3. orch experiment run <experiment> --condition A
  4. Manually record metrics in the generated run file
  5. orch experiment analyze <experiment>`,
}

var (
	experimentCreateHypothesis string
)

var experimentCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Scaffold a new experiment",
	Long: `Create a new experiment directory with template files.

Creates:
  .orch/experiments/<date>-<name>/
    experiment.yaml   - Configuration with conditions, metrics
    runs/             - Directory for run data
    analysis.md       - Template for analysis

Examples:
  orch experiment create context-attention
  orch experiment create context-attention --hypothesis "Position affects recognition"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return runExperimentCreate(name, experimentCreateHypothesis)
	},
}

var experimentListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all experiments with status",
	Long: `List all experiments in .orch/experiments/ with their status and progress.

Shows:
  - Experiment name and status
  - Creation date
  - Number of runs completed vs target

Examples:
  orch experiment list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExperimentList()
	},
}

var (
	experimentRunCondition string
	experimentRunSessionID string
	experimentRunBeadsID   string
	experimentRunNotes     string
)

var experimentRunCmd = &cobra.Command{
	Use:   "run <experiment>",
	Short: "Execute one condition of an experiment",
	Long: `Record a run of one experimental condition.

For v1, this creates the run file structure and prompts you to fill in
the metrics manually after observing the agent. Future versions may
integrate with session introspection for automatic metric capture.

The run file is saved to:
  .orch/experiments/<experiment>/runs/<condition>-<n>.json

Examples:
  orch experiment run 2025-12-28-context-attention --condition A
  orch experiment run 2025-12-28-context-attention --condition A --session-id ses_xxx
  orch experiment run 2025-12-28-context-attention --condition A --notes "Agent struggled initially"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		expName := args[0]
		return runExperimentRun(expName, experimentRunCondition, experimentRunSessionID, experimentRunBeadsID, experimentRunNotes)
	},
}

var experimentStatusCmd = &cobra.Command{
	Use:   "status <experiment>",
	Short: "Show runs completed per condition",
	Long: `Display the current status of an experiment including:
  - Runs completed per condition
  - Progress toward target runs
  - Quick metrics summary

Examples:
  orch experiment status 2025-12-28-context-attention`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		expName := args[0]
		return runExperimentStatus(expName)
	},
}

var experimentAnalyzeCmd = &cobra.Command{
	Use:   "analyze <experiment>",
	Short: "Generate/update analysis.md with results",
	Long: `Generate or update the analysis.md file with current results.

Calculates per-condition metrics and creates summary tables:
  - Average tool calls before recognition
  - Recognition rate (% that found existing answer)
  - Average time to recognition

Examples:
  orch experiment analyze 2025-12-28-context-attention`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		expName := args[0]
		return runExperimentAnalyze(expName)
	},
}

func init() {
	experimentCreateCmd.Flags().StringVar(&experimentCreateHypothesis, "hypothesis", "", "Hypothesis being tested")

	experimentRunCmd.Flags().StringVar(&experimentRunCondition, "condition", "", "Condition ID (e.g., A, B)")
	experimentRunCmd.Flags().StringVar(&experimentRunSessionID, "session-id", "", "OpenCode session ID (optional)")
	experimentRunCmd.Flags().StringVar(&experimentRunBeadsID, "beads-id", "", "Beads issue ID (optional)")
	experimentRunCmd.Flags().StringVar(&experimentRunNotes, "notes", "", "Notes about this run (optional)")
	experimentRunCmd.MarkFlagRequired("condition")

	experimentCmd.AddCommand(experimentCreateCmd)
	experimentCmd.AddCommand(experimentListCmd)
	experimentCmd.AddCommand(experimentRunCmd)
	experimentCmd.AddCommand(experimentStatusCmd)
	experimentCmd.AddCommand(experimentAnalyzeCmd)

	rootCmd.AddCommand(experimentCmd)
}

func runExperimentCreate(name, hypothesis string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if hypothesis == "" {
		hypothesis = fmt.Sprintf("Hypothesis for %s experiment", name)
	}

	expDir, err := experiment.CreateExperiment(projectDir, name, hypothesis)
	if err != nil {
		return err
	}

	fmt.Printf("Created experiment: %s\n", filepath.Base(expDir))
	fmt.Printf("  Directory: %s\n", expDir)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit %s/experiment.yaml to define conditions\n", expDir)
	fmt.Printf("  2. Run conditions: orch experiment run %s --condition A\n", filepath.Base(expDir))
	fmt.Printf("  3. Record metrics in the generated run file\n")
	fmt.Printf("  4. Analyze results: orch experiment analyze %s\n", filepath.Base(expDir))

	return nil
}

func runExperimentList() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	experiments, err := experiment.ListExperiments(projectDir)
	if err != nil {
		return err
	}

	if len(experiments) == 0 {
		fmt.Println("No experiments found.")
		fmt.Println()
		fmt.Println("Create one with: orch experiment create <name>")
		return nil
	}

	fmt.Println("Experiments:")
	fmt.Println()

	for _, exp := range experiments {
		// Get run counts
		expDir := experiment.ExperimentDir(projectDir, fmt.Sprintf("%s-%s", exp.Created, exp.Name))
		runs, _ := experiment.LoadRuns(expDir)
		byCondition := experiment.RunsByCondition(runs)

		// Calculate total target runs
		totalTarget := exp.RunsPerCondition * len(exp.Conditions)
		totalCompleted := len(runs)

		statusEmoji := "🔄"
		switch exp.Status {
		case experiment.StatusComplete:
			statusEmoji = "✅"
		case experiment.StatusAbandoned:
			statusEmoji = "❌"
		}

		fmt.Printf("%s %s-%s [%s]\n", statusEmoji, exp.Created, exp.Name, exp.Status)
		fmt.Printf("   Hypothesis: %s\n", truncateString(exp.Hypothesis, 60))
		fmt.Printf("   Progress:   %d/%d runs", totalCompleted, totalTarget)

		// Show per-condition breakdown
		if len(exp.Conditions) > 0 {
			var parts []string
			var condIDs []string
			for id := range exp.Conditions {
				condIDs = append(condIDs, id)
			}
			sort.Strings(condIDs)
			for _, id := range condIDs {
				count := len(byCondition[id])
				parts = append(parts, fmt.Sprintf("%s:%d/%d", id, count, exp.RunsPerCondition))
			}
			fmt.Printf(" (%s)", strings.Join(parts, ", "))
		}
		fmt.Println()
		fmt.Println()
	}

	return nil
}

func runExperimentRun(expName, condition, sessionID, beadsID, notes string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find experiment directory (handle both full name and partial name)
	expDir, err := findExperimentDir(projectDir, expName)
	if err != nil {
		return err
	}

	exp, err := experiment.LoadExperiment(expDir)
	if err != nil {
		return err
	}

	// Validate condition
	if _, ok := exp.Conditions[condition]; !ok {
		var validConditions []string
		for id := range exp.Conditions {
			validConditions = append(validConditions, id)
		}
		sort.Strings(validConditions)
		return fmt.Errorf("invalid condition '%s'. Valid conditions: %s", condition, strings.Join(validConditions, ", "))
	}

	// Check if already at target runs
	runs, _ := experiment.LoadRuns(expDir)
	byCondition := experiment.RunsByCondition(runs)
	currentCount := len(byCondition[condition])
	if currentCount >= exp.RunsPerCondition {
		fmt.Printf("Warning: Condition %s already has %d/%d runs completed.\n", condition, currentCount, exp.RunsPerCondition)
		fmt.Println("Continue anyway? (y/N)")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return fmt.Errorf("aborted")
		}
	}

	// Get next run number
	runNumber, err := experiment.GetNextRunNumber(expDir, condition)
	if err != nil {
		return err
	}

	// Create run with placeholder metrics (to be filled in manually)
	run := &experiment.Run{
		Condition: condition,
		RunNumber: runNumber,
		Timestamp: time.Now().Format(time.RFC3339),
		SessionID: sessionID,
		BeadsID:   beadsID,
		Metrics: experiment.RunMetrics{
			ToolCallsBeforeRecognition: 0,
			RecognizedExistingAnswer:   false,
			TimeToRecognitionSeconds:   0,
		},
		Notes: notes,
	}

	if err := experiment.SaveRun(expDir, run); err != nil {
		return err
	}

	runFile := filepath.Join(expDir, "runs", fmt.Sprintf("%s-%d.json", condition, runNumber))

	fmt.Printf("Created run file: %s\n", runFile)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Spawn an agent with the condition's spawn context modifications")
	fmt.Println("  2. Observe the agent's behavior")
	fmt.Println("  3. Edit the run file to record metrics:")
	fmt.Printf("     - tool_calls_before_recognition: count of tool calls before citing the constraint\n")
	fmt.Printf("     - recognized_existing_answer: true/false\n")
	fmt.Printf("     - time_to_recognition_seconds: seconds until recognition (or total time if not recognized)\n")
	fmt.Println()

	// Show the run file content for reference
	data, _ := json.MarshalIndent(run, "", "  ")
	fmt.Println("Run file content:")
	fmt.Println(string(data))

	return nil
}

func runExperimentStatus(expName string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	expDir, err := findExperimentDir(projectDir, expName)
	if err != nil {
		return err
	}

	exp, err := experiment.LoadExperiment(expDir)
	if err != nil {
		return err
	}

	runs, err := experiment.LoadRuns(expDir)
	if err != nil {
		return err
	}

	byCondition := experiment.RunsByCondition(runs)

	fmt.Printf("Experiment: %s-%s\n", exp.Created, exp.Name)
	fmt.Printf("Status: %s\n", exp.Status)
	fmt.Printf("Hypothesis: %s\n", exp.Hypothesis)
	fmt.Println()

	// Get sorted condition IDs
	var condIDs []string
	for id := range exp.Conditions {
		condIDs = append(condIDs, id)
	}
	sort.Strings(condIDs)

	fmt.Println("Progress by Condition:")
	fmt.Println()

	totalCompleted := 0
	totalTarget := 0

	for _, id := range condIDs {
		cond := exp.Conditions[id]
		condRuns := byCondition[id]
		completed := len(condRuns)
		target := exp.RunsPerCondition
		totalCompleted += completed
		totalTarget += target

		progress := fmt.Sprintf("%d/%d", completed, target)
		bar := makeProgressBar(completed, target, 20)

		fmt.Printf("  %s: %s %s\n", id, bar, progress)
		fmt.Printf("     %s\n", cond.Description)

		// Show quick metrics if runs exist
		if completed > 0 {
			var totalToolCalls, recognizedCount, totalTime int
			for _, r := range condRuns {
				totalToolCalls += r.Metrics.ToolCallsBeforeRecognition
				if r.Metrics.RecognizedExistingAnswer {
					recognizedCount++
				}
				totalTime += r.Metrics.TimeToRecognitionSeconds
			}
			avgToolCalls := float64(totalToolCalls) / float64(completed)
			recognitionRate := float64(recognizedCount) / float64(completed) * 100
			avgTime := float64(totalTime) / float64(completed)
			fmt.Printf("     Avg tool calls: %.1f | Recognition: %.0f%% | Avg time: %.0fs\n", avgToolCalls, recognitionRate, avgTime)
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d/%d runs completed\n", totalCompleted, totalTarget)

	if totalCompleted >= totalTarget && exp.Status == experiment.StatusRunning {
		fmt.Println()
		fmt.Println("All runs completed! Consider:")
		fmt.Printf("  1. Run analysis: orch experiment analyze %s\n", expName)
		fmt.Println("  2. Update experiment status to 'complete' in experiment.yaml")
	}

	return nil
}

func runExperimentAnalyze(expName string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	expDir, err := findExperimentDir(projectDir, expName)
	if err != nil {
		return err
	}

	analysisPath, err := experiment.GenerateAnalysis(expDir)
	if err != nil {
		return err
	}

	fmt.Printf("Updated analysis: %s\n", analysisPath)
	fmt.Println()
	fmt.Println("Review the analysis and add your observations and conclusions.")

	return nil
}

// findExperimentDir finds the experiment directory by name (supports partial matches).
func findExperimentDir(projectDir, name string) (string, error) {
	// Try exact match first
	expDir := experiment.ExperimentDir(projectDir, name)
	if _, err := os.Stat(expDir); err == nil {
		return expDir, nil
	}

	// Try to find by partial match
	expBaseDir := experiment.ExperimentsDir(projectDir)
	entries, err := os.ReadDir(expBaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no experiments directory found")
		}
		return "", err
	}

	var matches []string
	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(entry.Name(), name) {
			matches = append(matches, entry.Name())
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("experiment not found: %s", name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous experiment name '%s'. Matches: %s", name, strings.Join(matches, ", "))
	}

	return filepath.Join(expBaseDir, matches[0]), nil
}

// truncateString shortens a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// makeProgressBar creates an ASCII progress bar.
func makeProgressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}

	filled := (current * width) / total
	if filled > width {
		filled = width
	}

	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
