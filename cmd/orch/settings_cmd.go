package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/hook"
	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Modify ~/.claude/settings.json programmatically",
	Long: `Modify Claude Code settings.json from agents or scripts.

Agents run in a sandbox that blocks direct file writes to ~/.claude/settings.json.
This command runs unsandboxed via orch, enabling agents to add/remove hooks
without leaving manual instructions that get forgotten.

Commands:
  add-hook      Add a hook to an event type
  remove-hook   Remove a hook by command match
  list-hooks    List configured hooks`,
}

var (
	settingsCmdPath    string
	addHookMatcher     string
	addHookTimeout     int
	listHooksEvent     string
)

var settingsAddHookCmd = &cobra.Command{
	Use:   "add-hook <event> <command>",
	Short: "Add a hook to an event type",
	Long: `Add a hook command to a Claude Code event type.

Events: SessionStart, SessionEnd, PreToolUse, PostToolUse, PreCompact, Stop

Examples:
  orch settings add-hook SessionStart '$HOME/.orch/hooks/my-hook.sh'
  orch settings add-hook PreToolUse '$HOME/.orch/hooks/gate.py' --matcher Bash --timeout 10
  orch settings add-hook PostToolUse '$HOME/.orch/hooks/log.py' --matcher 'Read|Edit'`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		event, command := args[0], args[1]
		path := resolveSettingsPath()

		if err := hook.AddHook(path, event, addHookMatcher, command, addHookTimeout); err != nil {
			return err
		}

		matcherDisplay := addHookMatcher
		if matcherDisplay == "" {
			matcherDisplay = "*"
		}
		fmt.Printf("Added hook to %s (matcher: %s): %s\n", event, matcherDisplay, command)
		return nil
	},
}

var settingsRemoveHookCmd = &cobra.Command{
	Use:   "remove-hook <event> <command-match>",
	Short: "Remove a hook by command substring match",
	Long: `Remove a hook from a Claude Code event type by matching the command string.

The command-match is a substring match against the hook's command field.

Examples:
  orch settings remove-hook PreToolUse gate-bd-close.py
  orch settings remove-hook SessionStart my-hook.sh`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		event, match := args[0], args[1]
		path := resolveSettingsPath()

		removed, err := hook.RemoveHook(path, event, match)
		if err != nil {
			return err
		}
		if !removed {
			fmt.Printf("No hook matching %q found in %s\n", match, event)
			return nil
		}

		fmt.Printf("Removed hook matching %q from %s\n", match, event)
		return nil
	},
}

var settingsListHooksCmd = &cobra.Command{
	Use:   "list-hooks",
	Short: "List configured hooks",
	Long: `List all hooks configured in settings.json.

Examples:
  orch settings list-hooks
  orch settings list-hooks --event PreToolUse`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := resolveSettingsPath()

		entries, err := hook.ListHooks(path, listHooksEvent)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			if listHooksEvent != "" {
				fmt.Printf("No hooks configured for %s\n", listHooksEvent)
			} else {
				fmt.Println("No hooks configured")
			}
			return nil
		}

		// Group by event for display
		byEvent := make(map[string][]hook.HookEntry)
		for _, e := range entries {
			byEvent[e.Event] = append(byEvent[e.Event], e)
		}

		events := make([]string, 0, len(byEvent))
		for e := range byEvent {
			events = append(events, e)
		}
		sort.Strings(events)

		for _, event := range events {
			fmt.Printf("%s:\n", event)
			for _, e := range byEvent[event] {
				matcher := e.Matcher
				if matcher == "" {
					matcher = "*"
				}
				parts := []string{fmt.Sprintf("  %s", e.Command)}
				parts = append(parts, fmt.Sprintf("(matcher: %s", matcher))
				if e.Timeout > 0 {
					parts[len(parts)-1] += fmt.Sprintf(", timeout: %ds)", e.Timeout)
				} else {
					parts[len(parts)-1] += ")"
				}
				fmt.Println(strings.Join(parts, " "))
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	settingsCmd.PersistentFlags().StringVar(&settingsCmdPath, "settings", "", "Path to settings.json (default: ~/.claude/settings.json)")

	settingsAddHookCmd.Flags().StringVar(&addHookMatcher, "matcher", "", "Tool matcher pattern (e.g., 'Bash', 'Read|Edit')")
	settingsAddHookCmd.Flags().IntVar(&addHookTimeout, "timeout", 10, "Hook timeout in seconds")

	settingsListHooksCmd.Flags().StringVar(&listHooksEvent, "event", "", "Filter by event type")

	settingsCmd.AddCommand(settingsAddHookCmd)
	settingsCmd.AddCommand(settingsRemoveHookCmd)
	settingsCmd.AddCommand(settingsListHooksCmd)
}

func resolveSettingsPath() string {
	if settingsCmdPath != "" {
		return settingsCmdPath
	}
	return hook.DefaultSettingsPath()
}
