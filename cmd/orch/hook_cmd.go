package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/hook"
	"github.com/spf13/cobra"
)

// ============================================================================
// orch hook - Hook testing, tracing, and validation
// ============================================================================

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Test, validate, and trace Claude Code hooks",
	Long: `Hook development toolkit for Claude Code hooks.

Provides tools to test hooks outside of Claude Code sessions,
validate hook configuration, and trace hook execution.

Commands:
  test       Simulate hook invocations with controlled input
  validate   Lint hook configuration for common errors
  trace      View runtime hook execution traces`,
}

// ============================================================================
// orch hook test
// ============================================================================

var (
	hookTestTool     string
	hookTestInput    string
	hookTestEnv      []string
	hookTestHook     string
	hookTestDryRun   bool
	hookTestVerbose  bool
	hookTestAllHooks bool
	hookTestSettings string
)

var hookTestCmd = &cobra.Command{
	Use:   "test [event]",
	Short: "Simulate a hook invocation outside of Claude Code",
	Long: `Test hooks by simulating Claude Code's hook invocation pipeline.

Reads settings.json, resolves matchers, constructs JSON input, runs hooks,
and validates output format against the expected schema for the event type.

Examples:
  orch hook test PreToolUse --tool Bash
  orch hook test PreToolUse --tool Task --env CLAUDE_CONTEXT=orchestrator
  orch hook test PreToolUse --tool Bash --input '{"command": "bd close orch-go-1234"}'
  orch hook test PreToolUse --hook ~/.orch/hooks/gate-bd-close.py --tool Bash
  orch hook test SessionStart
  orch hook test PreToolUse --tool Bash --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHookTest(args[0])
	},
}

var hookValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Lint hook configuration for common errors",
	Long: `Validate the hook configuration in settings.json.

Checks for:
  - Command files exist and are executable
  - Matchers are valid regex
  - Timeouts are set and reasonable
  - Script files have shebang lines

Examples:
  orch hook validate
  orch hook validate --settings ~/.claude/settings.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHookValidate()
	},
}

// ============================================================================
// orch hook trace
// ============================================================================

var (
	hookTraceLimit   int
	hookTraceSession string
	hookTraceHook    string
	hookTraceEvent   string
	hookTracePath    string
)

var hookTraceCmd = &cobra.Command{
	Use:   "trace",
	Short: "View hook execution traces",
	Long: `View runtime hook execution traces from ~/.orch/hooks/trace.jsonl.

Hooks must opt-in to tracing by checking the HOOK_TRACE environment variable.
Enable tracing in your SessionStart hook by setting HOOK_TRACE=1.

Examples:
  orch hook trace                          # Show last 50 entries
  orch hook trace --limit 100              # Show last 100 entries
  orch hook trace --session abc123         # Filter by session ID
  orch hook trace --hook gate-bd-close     # Filter by hook name
  orch hook trace --event PreToolUse       # Filter by event type`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHookTrace()
	},
}

func init() {
	// hook test flags
	hookTestCmd.Flags().StringVar(&hookTestTool, "tool", "", "Tool name for matcher resolution (Bash, Read, Edit, Task, etc.)")
	hookTestCmd.Flags().StringVar(&hookTestInput, "input", "", "JSON string for event-specific input fields")
	hookTestCmd.Flags().StringSliceVar(&hookTestEnv, "env", nil, "Override env vars (KEY=VALUE, repeatable)")
	hookTestCmd.Flags().StringVar(&hookTestHook, "hook", "", "Test a specific hook file directly (skip settings.json)")
	hookTestCmd.Flags().BoolVar(&hookTestDryRun, "dry-run", false, "Show which hooks would fire without executing")
	hookTestCmd.Flags().BoolVar(&hookTestVerbose, "verbose", false, "Show full JSON input/output")
	hookTestCmd.Flags().BoolVar(&hookTestAllHooks, "all-hooks", false, "Run all hooks for this event, not just matching")
	hookTestCmd.Flags().StringVar(&hookTestSettings, "settings", "", "Path to settings.json (default: ~/.claude/settings.json)")

	// hook validate flags
	hookValidateCmd.Flags().StringVar(&hookTestSettings, "settings", "", "Path to settings.json (default: ~/.claude/settings.json)")

	// hook trace flags
	hookTraceCmd.Flags().IntVar(&hookTraceLimit, "limit", 50, "Maximum entries to show")
	hookTraceCmd.Flags().StringVar(&hookTraceSession, "session", "", "Filter by session ID")
	hookTraceCmd.Flags().StringVar(&hookTraceHook, "hook", "", "Filter by hook name (substring)")
	hookTraceCmd.Flags().StringVar(&hookTraceEvent, "event", "", "Filter by event type")
	hookTraceCmd.Flags().StringVar(&hookTracePath, "path", "", "Path to trace file (default: ~/.orch/hooks/trace.jsonl)")

	hookCmd.AddCommand(hookTestCmd)
	hookCmd.AddCommand(hookValidateCmd)
	hookCmd.AddCommand(hookTraceCmd)
}

// ============================================================================
// Implementation
// ============================================================================

func runHookTest(event string) error {
	// Parse env overrides
	envOverrides := parseEnvOverrides(hookTestEnv)

	// Parse user input
	var userInput map[string]interface{}
	if hookTestInput != "" {
		if err := json.Unmarshal([]byte(hookTestInput), &userInput); err != nil {
			return fmt.Errorf("invalid --input JSON: %w", err)
		}
	}

	// Build the full input
	input := hook.BuildInput(event, hookTestTool, userInput)

	if hookTestVerbose {
		inputJSON, _ := json.MarshalIndent(input, "", "  ")
		fmt.Printf("Input JSON:\n%s\n\n", inputJSON)
	}

	// If --hook is specified, test that specific hook directly
	if hookTestHook != "" {
		return runSingleHookTest(event, hookTestHook, input, envOverrides)
	}

	// Load settings and resolve hooks
	settingsPath := hookTestSettings
	if settingsPath == "" {
		settingsPath = hook.DefaultSettingsPath()
	}

	settings, err := hook.LoadSettingsFromPath(settingsPath)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	var hooks []hook.ResolvedHook
	if hookTestAllHooks {
		// Get all hooks for this event type
		groups, ok := settings.Hooks[event]
		if !ok {
			fmt.Printf("No hooks configured for %s\n", event)
			return nil
		}
		for _, group := range groups {
			for _, h := range group.Hooks {
				hooks = append(hooks, hook.ResolvedHook{
					Event:       event,
					Matcher:     group.Matcher,
					Command:     h.Command,
					Timeout:     h.Timeout,
					ExpandedCmd: hook.ExpandCommand(h.Command),
				})
			}
		}
	} else {
		hooks = settings.ResolveHooks(event, hookTestTool)
	}

	if len(hooks) == 0 {
		fmt.Printf("No matching hooks for %s", event)
		if hookTestTool != "" {
			fmt.Printf(" (tool: %s)", hookTestTool)
		}
		fmt.Println()
		return nil
	}

	// Print header
	fmt.Printf("Matching hooks for %s", event)
	if hookTestTool != "" {
		fmt.Printf(" (tool: %s)", hookTestTool)
	}
	fmt.Printf(":\n")
	for i, h := range hooks {
		matcherStr := h.Matcher
		if matcherStr == "" {
			matcherStr = "*"
		}
		fmt.Printf("  %d. %s (matcher: %s)\n", i+1, hook.CommandBasename(h.Command), matcherStr)
	}
	fmt.Println()

	if hookTestDryRun {
		fmt.Println("(dry-run: hooks not executed)")
		return nil
	}

	// Execute each hook
	for i, h := range hooks {
		fmt.Printf("Running hook %d: %s\n", i+1, hook.CommandBasename(h.Command))
		result := hook.RunHook(h, hook.RunOptions{
			EnvOverrides: envOverrides,
			Input:        input,
			Verbose:      hookTestVerbose,
		})

		printHookResult(result)
		fmt.Println()
	}

	return nil
}

func runSingleHookTest(event, hookPath string, input map[string]interface{}, envOverrides map[string]string) error {
	expanded := hook.ExpandCommand(hookPath)

	h := hook.ResolvedHook{
		Event:       event,
		Command:     hookPath,
		ExpandedCmd: expanded,
		Timeout:     10,
	}

	fmt.Printf("Running hook: %s\n", hook.CommandBasename(hookPath))

	if hookTestDryRun {
		fmt.Println("(dry-run: hook not executed)")
		return nil
	}

	result := hook.RunHook(h, hook.RunOptions{
		EnvOverrides: envOverrides,
		Input:        input,
		Verbose:      hookTestVerbose,
	})

	printHookResult(result)
	return nil
}

func printHookResult(result *hook.RunResult) {
	if result.Error != nil {
		fmt.Printf("  Error: %v\n", result.Error)
		return
	}

	fmt.Printf("  Exit code: %d\n", result.ExitCode)
	fmt.Printf("  Duration: %v\n", result.Duration.Round(100*1000)) // microsecond precision

	if result.Validation != nil {
		v := result.Validation

		// Decision
		fmt.Printf("  Decision: %s", v.Decision)
		if v.Reason != "" {
			fmt.Printf(" — %s", v.Reason)
		}
		fmt.Println()

		// Context
		if v.Context != "" {
			contextPreview := v.Context
			if len(contextPreview) > 200 {
				contextPreview = contextPreview[:200] + "..."
			}
			fmt.Printf("  Context: %s\n", contextPreview)
		}

		// Format validation
		if v.Valid && len(v.Warnings) == 0 {
			fmt.Printf("  Format: ✅ Valid")
			if v.Raw != nil {
				fmt.Printf(" (%s)", describeFormat(result.Hook.Event, v))
			}
			fmt.Println()
		} else {
			for _, w := range v.Warnings {
				fmt.Printf("  Format: ⚠️  WARNING: %s\n", w)
			}
			if !v.Valid {
				fmt.Printf("  %s\n", hook.FormatExpectedSchema(result.Hook.Event))
			}
		}
	}

	// Verbose: show raw output
	if hookTestVerbose && result.Stdout != "" {
		fmt.Printf("  Raw stdout:\n    %s\n", strings.ReplaceAll(strings.TrimSpace(result.Stdout), "\n", "\n    "))
	}
	if result.Stderr != "" {
		fmt.Printf("  Stderr:\n    %s\n", strings.ReplaceAll(strings.TrimSpace(result.Stderr), "\n", "\n    "))
	}
}

func describeFormat(event string, v *hook.ValidationResult) string {
	switch event {
	case "PreToolUse":
		if v.Raw != nil {
			if _, ok := v.Raw["hookSpecificOutput"]; ok {
				return fmt.Sprintf("hookSpecificOutput.permissionDecision = %q", strings.ToLower(string(v.Decision)))
			}
		}
		return "exit 0, no JSON = allow"
	case "PostToolUse":
		if v.Decision == hook.DecisionBlock {
			return "decision = \"block\""
		}
		return "no block decision"
	default:
		return "generic output"
	}
}

func runHookValidate() error {
	settingsPath := hookTestSettings
	if settingsPath == "" {
		settingsPath = hook.DefaultSettingsPath()
	}

	settings, err := hook.LoadSettingsFromPath(settingsPath)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	fmt.Printf("Checking hook configuration in %s...\n\n", settingsPath)

	// Display hooks grouped by event
	events := settings.ListEvents()
	sort.Strings(events)

	totalHooks := 0
	for _, event := range events {
		groups := settings.Hooks[event]
		if len(groups) == 0 {
			continue
		}

		fmt.Printf("%s:\n", event)
		for _, group := range groups {
			for _, h := range group.Hooks {
				totalHooks++
				basename := hook.CommandBasename(h.Command)
				matcherStr := group.Matcher
				if matcherStr == "" {
					matcherStr = "*"
				}
				timeoutStr := "default(600)"
				if h.Timeout > 0 {
					timeoutStr = fmt.Sprintf("%ds", h.Timeout)
				}

				// Quick check if file exists
				expanded := hook.ExpandCommand(h.Command)
				status := "✅"
				if strings.Contains(h.Command, "/") || strings.Contains(h.Command, "$HOME") {
					if _, err := os.Stat(expanded); os.IsNotExist(err) {
						status = "❌"
					}
				}

				fmt.Printf("  %s %s (matcher: %s, timeout: %s)\n", status, basename, matcherStr, timeoutStr)
			}
		}
		fmt.Println()
	}

	// Run full validation
	issues := hook.ValidateConfig(settings)
	errors := 0
	warnings := 0
	for _, issue := range issues {
		switch issue.Severity {
		case hook.SeverityError:
			errors++
		case hook.SeverityWarning:
			warnings++
		}
	}

	if len(issues) > 0 {
		fmt.Println("Issues found:")
		fmt.Println(hook.FormatIssues(issues))
	}

	fmt.Printf("\nValidation summary: %d hooks, %d errors, %d warnings\n", totalHooks, errors, warnings)
	return nil
}

func runHookTrace() error {
	path := hookTracePath
	if path == "" {
		path = hook.DefaultTracePath()
	}

	entries, err := hook.ReadTrace(path, hook.TraceOptions{
		Limit:         hookTraceLimit,
		SessionFilter: hookTraceSession,
		HookFilter:    hookTraceHook,
		EventFilter:   hookTraceEvent,
	})
	if err != nil {
		return err
	}

	fmt.Print(hook.FormatTraceEntries(entries))
	return nil
}

// parseEnvOverrides converts "KEY=VALUE" strings to a map.
func parseEnvOverrides(envSlice []string) map[string]string {
	overrides := make(map[string]string)
	for _, env := range envSlice {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			overrides[parts[0]] = parts[1]
		}
	}
	return overrides
}
