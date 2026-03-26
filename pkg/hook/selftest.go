package hook

import (
	"fmt"
	"os"
	"time"
)

// SelfTestOptions configures how self-test runs.
type SelfTestOptions struct {
	// Timeout in seconds for each hook (default: 10).
	Timeout int
	// Verbose includes full output details.
	Verbose bool
}

// SelfTestResult contains the result of testing a single hook.
type SelfTestResult struct {
	HookName string
	Event    string
	Matcher  string
	Command  string
	Passed   bool
	Summary  string
	Warnings []string
	Duration time.Duration
	Result   *RunResult // Full result for verbose output
}

// SelfTestSummary aggregates self-test results.
type SelfTestSummary struct {
	Total     int
	Passed    int
	Failed    int
	Warnings  int
	AllPassed bool
}

// SelfTest runs all configured hooks with synthetic inputs and validates
// they execute correctly: script exists, exits cleanly, output format is valid.
func SelfTest(settings *Settings, opts SelfTestOptions) []SelfTestResult {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10
	}

	var results []SelfTestResult

	for event, groups := range settings.Hooks {
		for _, group := range groups {
			for _, h := range group.Hooks {
				result := selfTestHook(event, group.Matcher, h, timeout)
				results = append(results, result)
			}
		}
	}

	return results
}

func selfTestHook(event, matcher string, h HookConfig, timeout int) SelfTestResult {
	result := SelfTestResult{
		HookName: CommandBasename(h.Command),
		Event:    event,
		Matcher:  matcher,
		Command:  h.Command,
	}

	// Check if command file exists (for path-based commands)
	expanded := expandCommand(h.Command)
	if _, err := os.Stat(expanded); os.IsNotExist(err) {
		result.Passed = false
		result.Summary = fmt.Sprintf("script not found: %s", expanded)
		return result
	}

	// Build synthetic input for the event type
	tool := "Bash" // Default tool for matcher-based events
	if matcher != "" {
		tool = matcher
	}
	input := BuildInput(event, tool, nil)

	// Run the hook
	resolved := ResolvedHook{
		Event:       event,
		Matcher:     matcher,
		Command:     h.Command,
		Timeout:     timeout,
		ExpandedCmd: expanded,
	}

	runResult := RunHook(resolved, RunOptions{
		Input:   input,
		Timeout: time.Duration(timeout) * time.Second,
	})
	result.Result = runResult
	result.Duration = runResult.Duration

	// Check for execution errors
	if runResult.Error != nil {
		result.Passed = false
		result.Summary = fmt.Sprintf("execution error: %v", runResult.Error)
		return result
	}

	// Non-zero exit = fail for selftest (hooks shouldn't deny synthetic input)
	if runResult.ExitCode != 0 {
		result.Passed = false
		result.Summary = fmt.Sprintf("exit code %d (expected 0 for synthetic input)", runResult.ExitCode)
		return result
	}

	// Collect format warnings
	if runResult.Validation != nil {
		result.Warnings = runResult.Validation.Warnings
	}

	result.Passed = true
	result.Summary = fmt.Sprintf("OK (%v)", runResult.Duration.Round(time.Millisecond))
	return result
}

// FormatSelfTestSummary builds an aggregate summary from results.
func FormatSelfTestSummary(results []SelfTestResult) SelfTestSummary {
	summary := SelfTestSummary{Total: len(results)}

	for _, r := range results {
		if r.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
		summary.Warnings += len(r.Warnings)
	}

	summary.AllPassed = summary.Failed == 0
	return summary
}
