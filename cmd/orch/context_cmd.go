package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/hook"
	"github.com/spf13/cobra"
)

var (
	contextVerbose bool
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Show the SessionStart hook context that would be injected",
	Long: `Show the combined context output from all SessionStart hooks.

This displays the same content that Claude Code receives when a session starts,
including orchestrator skill, beads workflow, frontier state, and orient metrics.

Useful for debugging what context agents receive at session start.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runContext()
	},
}

func init() {
	contextCmd.Flags().BoolVar(&contextVerbose, "verbose", false, "Show per-hook details (name, duration, exit code)")
	rootCmd.AddCommand(contextCmd)
}

func runContext() error {
	settingsPath := hook.DefaultSettingsPath()
	settings, err := hook.LoadSettingsFromPath(settingsPath)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Build SessionStart input with source=startup
	input := hook.BuildInput("SessionStart", "", map[string]interface{}{
		"source": "startup",
	})

	// Get all SessionStart hooks
	groups, ok := settings.Hooks["SessionStart"]
	if !ok {
		fmt.Println("No SessionStart hooks configured")
		return nil
	}

	var hooks []hook.ResolvedHook
	for _, group := range groups {
		for _, h := range group.Hooks {
			hooks = append(hooks, hook.ResolvedHook{
				Event:       "SessionStart",
				Matcher:     group.Matcher,
				Command:     h.Command,
				Timeout:     h.Timeout,
				ExpandedCmd: hook.ExpandCommand(h.Command),
			})
		}
	}

	if len(hooks) == 0 {
		fmt.Println("No SessionStart hooks configured")
		return nil
	}

	if contextVerbose {
		fmt.Printf("Running %d SessionStart hooks...\n\n", len(hooks))
	}

	// Run each hook and collect additionalContext
	var contextParts []string
	for _, h := range hooks {
		result := hook.RunHook(h, hook.RunOptions{
			Input: input,
		})

		if contextVerbose {
			name := hook.CommandBasename(h.Command)
			fmt.Printf("--- %s (exit=%d, %v) ---\n", name, result.ExitCode, result.Duration.Round(1000000))
			if result.Error != nil {
				fmt.Printf("  error: %v\n", result.Error)
			}
		}

		if result.Error != nil || result.ExitCode != 0 {
			continue
		}

		// Extract additionalContext from hook output
		ctx := extractAdditionalContext(result.Stdout)
		if ctx != "" {
			contextParts = append(contextParts, ctx)
		}
	}

	if len(contextParts) == 0 {
		fmt.Println("No context produced by SessionStart hooks")
		return nil
	}

	// Print combined context
	fmt.Println(strings.Join(contextParts, "\n"))
	return nil
}

// extractAdditionalContext parses hook JSON output and returns the additionalContext field.
// Handles both hookSpecificOutput.additionalContext and root-level additionalContext.
func extractAdditionalContext(stdout string) string {
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		return ""
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		// Non-JSON output — return as-is (some hooks output plain text)
		return trimmed
	}

	// Check hookSpecificOutput.additionalContext (SessionStart format)
	if hso, ok := raw["hookSpecificOutput"].(map[string]interface{}); ok {
		if ctx, ok := hso["additionalContext"].(string); ok && ctx != "" {
			return ctx
		}
	}

	// Check root-level additionalContext
	if ctx, ok := raw["additionalContext"].(string); ok && ctx != "" {
		return ctx
	}

	return ""
}
