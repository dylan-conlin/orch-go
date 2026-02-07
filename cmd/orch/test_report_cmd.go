// Package main provides the test-report command for running tests and reporting results to beads.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/spf13/cobra"
)

var (
	testReportDryRun  bool
	testReportVerbose bool
	testReportCommand string // Custom test command override
)

var testReportCmd = &cobra.Command{
	Use:    "test-report <beads-id>",
	Short:  "Run tests and report results to beads in verification-gate-compatible format",
	Hidden: true,
	Long: `Run tests and automatically report results to beads comments.

This command detects the project type, runs the appropriate test command,
parses the output, and submits a properly formatted beads comment that
satisfies the test evidence verification gate.

DETECTION ORDER:
  1. Go (go.mod exists) - runs: go test ./...
  2. Node (package.json exists) - runs: npm test
  3. Python (pyproject.toml or setup.py exists) - runs: pytest
  4. Rust (Cargo.toml exists) - runs: cargo test

OUTPUT FORMAT:
The command formats the beads comment to match the verification patterns:
  "Tests: go test ./... - PASS (47 tests in 2.3s)"
  "Tests: npm test - 23 passed, 0 failed"
  "Tests: pytest - 15 passed, 0 failed"

USE --command TO OVERRIDE:
If automatic detection doesn't work for your project, specify a custom test command:
  orch test-report proj-123 --command "make test"

Examples:
  orch test-report proj-123                    # Auto-detect and run tests
  orch test-report proj-123 --dry-run          # Show what would be reported without submitting
  orch test-report proj-123 --verbose          # Show full test output
  orch test-report proj-123 --command "npm run test:unit"  # Use custom test command`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runTestReport(beadsID)
	},
}

func init() {
	testReportCmd.Flags().BoolVar(&testReportDryRun, "dry-run", false, "Show what would be reported without submitting to beads")
	testReportCmd.Flags().BoolVar(&testReportVerbose, "verbose", false, "Show full test output")
	testReportCmd.Flags().StringVar(&testReportCommand, "command", "", "Custom test command to run (overrides auto-detection)")
	rootCmd.AddCommand(testReportCmd)
}

// ProjectType represents the detected project type.
type ProjectType string

const (
	ProjectTypeGo      ProjectType = "go"
	ProjectTypeNode    ProjectType = "node"
	ProjectTypePython  ProjectType = "python"
	ProjectTypeRust    ProjectType = "rust"
	ProjectTypeUnknown ProjectType = "unknown"
)

// TestResult holds the parsed test results.
type TestResult struct {
	Command     string
	Passed      int
	Failed      int
	Skipped     int
	Duration    time.Duration
	RawOutput   string
	ExitCode    int
	Summary     string // Human-readable summary
	EvidenceStr string // Verification-gate-compatible string
}

func runTestReport(beadsID string) error {
	// Resolve short beads ID
	resolvedID, err := resolveShortBeadsID(beadsID)
	if err != nil {
		return fmt.Errorf("failed to resolve beads ID: %w", err)
	}
	beadsID = resolvedID

	// Get current directory as project dir
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine test command
	var testCmd string
	var projectType ProjectType

	// Always detect project type for parsing purposes
	projectType, autoCmd := detectProjectType(projectDir)

	if testReportCommand != "" {
		// Use custom command but keep detected project type for parsing
		testCmd = testReportCommand
		// If project type unknown, try to infer from command
		if projectType == ProjectTypeUnknown {
			projectType = inferProjectTypeFromCommand(testReportCommand)
		}
		fmt.Printf("Using custom test command: %s\n", testCmd)
		if projectType != ProjectTypeUnknown {
			fmt.Printf("Parsing as %s project\n", projectType)
		}
	} else {
		// Use auto-detected command
		if projectType == ProjectTypeUnknown {
			return fmt.Errorf("could not detect project type. Use --command to specify a test command")
		}
		testCmd = autoCmd
		fmt.Printf("Detected %s project, running: %s\n", projectType, testCmd)
	}

	// Run tests
	fmt.Println("Running tests...")
	result, err := runTests(testCmd, projectDir, projectType)
	if err != nil {
		// Even if tests fail, we want to report the results
		fmt.Fprintf(os.Stderr, "Warning: test execution error: %v\n", err)
	}

	// Show verbose output if requested
	if testReportVerbose && result.RawOutput != "" {
		fmt.Println("\n--- Test Output ---")
		fmt.Println(result.RawOutput)
		fmt.Println("-------------------")
	}

	// Format the evidence string
	evidenceStr := formatTestEvidence(result)
	result.EvidenceStr = evidenceStr

	// Show results
	fmt.Printf("\nTest Results:\n")
	fmt.Printf("  Passed:  %d\n", result.Passed)
	fmt.Printf("  Failed:  %d\n", result.Failed)
	if result.Skipped > 0 {
		fmt.Printf("  Skipped: %d\n", result.Skipped)
	}
	if result.Duration > 0 {
		fmt.Printf("  Duration: %v\n", result.Duration)
	}
	fmt.Printf("  Exit code: %d\n", result.ExitCode)
	fmt.Printf("\nEvidence string:\n  %s\n", evidenceStr)

	// Submit to beads (or dry-run)
	if testReportDryRun {
		fmt.Println("\n--dry-run: Would submit the following comment:")
		fmt.Printf("  bd comment %s \"%s\"\n", beadsID, evidenceStr)
		return nil
	}

	// Submit comment
	fmt.Printf("\nSubmitting to beads issue %s...\n", beadsID)
	if err := submitTestEvidence(beadsID, evidenceStr); err != nil {
		return fmt.Errorf("failed to submit beads comment: %w", err)
	}

	fmt.Println("Test evidence reported successfully")

	// Return error if tests failed (exit code != 0)
	if result.ExitCode != 0 {
		return fmt.Errorf("tests failed with exit code %d", result.ExitCode)
	}

	return nil
}

// detectProjectType detects the project type from the project directory.
func detectProjectType(projectDir string) (ProjectType, string) {
	// Go
	if _, err := os.Stat(filepath.Join(projectDir, "go.mod")); err == nil {
		return ProjectTypeGo, "go test ./..."
	}

	// Node
	if _, err := os.Stat(filepath.Join(projectDir, "package.json")); err == nil {
		// Check for common test scripts
		return ProjectTypeNode, "npm test"
	}

	// Python
	if _, err := os.Stat(filepath.Join(projectDir, "pyproject.toml")); err == nil {
		return ProjectTypePython, "pytest"
	}
	if _, err := os.Stat(filepath.Join(projectDir, "setup.py")); err == nil {
		return ProjectTypePython, "pytest"
	}

	// Rust
	if _, err := os.Stat(filepath.Join(projectDir, "Cargo.toml")); err == nil {
		return ProjectTypeRust, "cargo test"
	}

	return ProjectTypeUnknown, ""
}

// inferProjectTypeFromCommand tries to infer the project type from the test command.
// This is used when --command is provided to ensure proper output parsing.
func inferProjectTypeFromCommand(cmd string) ProjectType {
	cmd = strings.ToLower(cmd)

	// Go
	if strings.Contains(cmd, "go test") {
		return ProjectTypeGo
	}

	// Node
	if strings.Contains(cmd, "npm ") || strings.Contains(cmd, "yarn ") ||
		strings.Contains(cmd, "bun ") || strings.Contains(cmd, "jest") ||
		strings.Contains(cmd, "mocha") || strings.Contains(cmd, "vitest") {
		return ProjectTypeNode
	}

	// Python
	if strings.Contains(cmd, "pytest") || strings.Contains(cmd, "python -m pytest") ||
		strings.Contains(cmd, "python -m unittest") || strings.Contains(cmd, "unittest") {
		return ProjectTypePython
	}

	// Rust
	if strings.Contains(cmd, "cargo test") {
		return ProjectTypeRust
	}

	return ProjectTypeUnknown
}

// runTests executes the test command and captures output.
func runTests(testCmd, projectDir string, projectType ProjectType) (*TestResult, error) {
	result := &TestResult{
		Command: testCmd,
	}

	// Parse command into parts
	parts := strings.Fields(testCmd)
	if len(parts) == 0 {
		return result, fmt.Errorf("empty test command")
	}

	// Create command
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = projectDir

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Track timing
	startTime := time.Now()
	err := cmd.Run()
	result.Duration = time.Since(startTime)

	// Combine output
	result.RawOutput = stdout.String()
	if stderr.Len() > 0 {
		result.RawOutput += "\n" + stderr.String()
	}

	// Get exit code
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else if err != nil {
		result.ExitCode = 1
	}

	// Parse results based on project type
	parseTestOutput(result, projectType)

	return result, err
}

// parseTestOutput extracts pass/fail counts from test output.
func parseTestOutput(result *TestResult, projectType ProjectType) {
	output := result.RawOutput

	switch projectType {
	case ProjectTypeGo:
		parseGoTestOutput(result, output)
	case ProjectTypeNode:
		parseNodeTestOutput(result, output)
	case ProjectTypePython:
		parsePythonTestOutput(result, output)
	case ProjectTypeRust:
		parseRustTestOutput(result, output)
	default:
		// Try to detect patterns from any framework
		parseGenericTestOutput(result, output)
	}
}

// Go test output patterns
var (
	goOkPattern   = regexp.MustCompile(`(?m)^ok\s+\S+\s+(\d+\.?\d*)s`)
	goFailPattern = regexp.MustCompile(`(?m)^FAIL\s+\S+`)
	// Match --- PASS/FAIL/SKIP with any leading whitespace (for subtests)
	goPassCount = regexp.MustCompile(`(?m)^\s*---\s*PASS:\s*`)
	goFailCount = regexp.MustCompile(`(?m)^\s*---\s*FAIL:\s*`)
	goSkipCount = regexp.MustCompile(`(?m)^\s*---\s*SKIP:\s*`)
)

func parseGoTestOutput(result *TestResult, output string) {
	// Count test results from verbose output
	result.Passed = len(goPassCount.FindAllString(output, -1))
	result.Failed = len(goFailCount.FindAllString(output, -1))
	result.Skipped = len(goSkipCount.FindAllString(output, -1))

	// Count package results
	okCount := len(goOkPattern.FindAllString(output, -1))
	failCount := len(goFailPattern.FindAllString(output, -1))

	// If no individual test counts, use package counts
	if result.Passed == 0 && result.Failed == 0 {
		// Check for overall pass/fail
		if result.ExitCode == 0 {
			// Tests passed but we don't have counts - use ok packages
			if okCount > 0 {
				result.Passed = okCount
				result.Summary = fmt.Sprintf("%d packages passed", okCount)
			}
		} else {
			result.Failed = failCount
			if okCount > 0 {
				result.Passed = okCount
			}
		}
	}
}

// Node test output patterns (Jest, Mocha, etc.)
var (
	jestPassPattern  = regexp.MustCompile(`Tests:\s+(\d+)\s+passed`)
	jestFailPattern  = regexp.MustCompile(`Tests:\s+\d+\s+failed,?\s*(\d+)?\s+passed`)
	mochaPassPattern = regexp.MustCompile(`(\d+)\s+passing`)
	mochaFailPattern = regexp.MustCompile(`(\d+)\s+failing`)
)

func parseNodeTestOutput(result *TestResult, output string) {
	// Try Jest patterns
	if matches := jestPassPattern.FindStringSubmatch(output); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.Passed)
	}

	// Try Mocha patterns
	if matches := mochaPassPattern.FindStringSubmatch(output); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.Passed)
	}
	if matches := mochaFailPattern.FindStringSubmatch(output); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.Failed)
	}

	// If exit code is 0 but no counts found, assume passed
	if result.ExitCode == 0 && result.Passed == 0 {
		result.Passed = 1 // At least indicate tests ran
	}
}

// Python test output patterns (pytest)
var (
	pytestSummary = regexp.MustCompile(`(\d+)\s+passed`)
	pytestFailed  = regexp.MustCompile(`(\d+)\s+failed`)
	pytestSkipped = regexp.MustCompile(`(\d+)\s+skipped`)
)

func parsePythonTestOutput(result *TestResult, output string) {
	if matches := pytestSummary.FindStringSubmatch(output); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.Passed)
	}
	if matches := pytestFailed.FindStringSubmatch(output); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.Failed)
	}
	if matches := pytestSkipped.FindStringSubmatch(output); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.Skipped)
	}
}

// Rust test output patterns
var (
	rustTestResult = regexp.MustCompile(`test result: (ok|FAILED)\.\s+(\d+)\s+passed;\s+(\d+)\s+failed`)
)

func parseRustTestOutput(result *TestResult, output string) {
	if matches := rustTestResult.FindStringSubmatch(output); len(matches) > 3 {
		fmt.Sscanf(matches[2], "%d", &result.Passed)
		fmt.Sscanf(matches[3], "%d", &result.Failed)
	}
}

// Generic patterns for unknown frameworks
func parseGenericTestOutput(result *TestResult, output string) {
	// Try common patterns
	patterns := []struct {
		passed *regexp.Regexp
		failed *regexp.Regexp
	}{
		{
			passed: regexp.MustCompile(`(\d+)\s+(?:tests?\s+)?pass(?:ed|ing)?`),
			failed: regexp.MustCompile(`(\d+)\s+(?:tests?\s+)?fail(?:ed|ing)?`),
		},
		{
			passed: regexp.MustCompile(`PASS(?:ED)?:\s*(\d+)`),
			failed: regexp.MustCompile(`FAIL(?:ED)?:\s*(\d+)`),
		},
	}

	for _, p := range patterns {
		if matches := p.passed.FindStringSubmatch(output); len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &result.Passed)
		}
		if matches := p.failed.FindStringSubmatch(output); len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &result.Failed)
		}
		if result.Passed > 0 || result.Failed > 0 {
			break
		}
	}

	// If still no counts but exit code is 0, indicate tests ran
	if result.ExitCode == 0 && result.Passed == 0 {
		result.Passed = 1
		result.Summary = "tests passed (count unknown)"
	}
}

// formatTestEvidence creates a verification-gate-compatible evidence string.
func formatTestEvidence(result *TestResult) string {
	// Format duration
	var durationStr string
	if result.Duration > 0 {
		if result.Duration < time.Second {
			durationStr = fmt.Sprintf("%.0fms", float64(result.Duration.Milliseconds()))
		} else {
			durationStr = fmt.Sprintf("%.1fs", result.Duration.Seconds())
		}
	}

	// Build evidence string
	var parts []string

	// Pass/fail counts
	if result.Passed > 0 || result.Failed > 0 {
		if result.Failed > 0 {
			parts = append(parts, fmt.Sprintf("%d passed, %d failed", result.Passed, result.Failed))
		} else {
			parts = append(parts, fmt.Sprintf("%d passed", result.Passed))
		}
	} else if result.Summary != "" {
		parts = append(parts, result.Summary)
	} else if result.ExitCode == 0 {
		parts = append(parts, "PASS")
	} else {
		parts = append(parts, "FAIL")
	}

	// Add duration
	if durationStr != "" {
		parts = append(parts, fmt.Sprintf("in %s", durationStr))
	}

	// Format: "Tests: <command> - <results>"
	return fmt.Sprintf("Tests: %s - %s", result.Command, strings.Join(parts, " "))
}

// submitTestEvidence submits the test evidence as a beads comment.
func submitTestEvidence(beadsID, evidence string) error {
	err := beads.Do("", func(client *beads.Client) error {
		return client.AddComment(beadsID, "agent", evidence)
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return nil
	}

	// Fallback to CLI
	return beads.FallbackAddComment(beadsID, evidence)
}
