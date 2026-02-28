// Package hook provides hook testing, validation, and tracing for Claude Code hooks.
// It reads hook configuration from ~/.claude/settings.json, resolves matchers,
// executes hooks with simulated input, and validates output schemas.
package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Settings represents the hook-related portion of ~/.claude/settings.json.
type Settings struct {
	Hooks map[string][]HookGroup `json:"hooks"`
}

// HookGroup represents a matcher + list of hooks for a given event type.
type HookGroup struct {
	Matcher string       `json:"matcher"`
	Hooks   []HookConfig `json:"hooks"`
}

// HookConfig represents a single hook command configuration.
type HookConfig struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// ResolvedHook is a hook that matched for a specific event+tool combination.
type ResolvedHook struct {
	Event       string
	Matcher     string
	Command     string
	Timeout     int
	ExpandedCmd string // Command with $HOME expanded
}

// LoadSettings reads and parses the hook configuration from settings.json.
func LoadSettings() (*Settings, error) {
	return LoadSettingsFromPath(DefaultSettingsPath())
}

// LoadSettingsFromPath reads and parses hook configuration from a specific path.
func LoadSettingsFromPath(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse settings JSON: %w", err)
	}

	hooksRaw, ok := raw["hooks"]
	if !ok {
		return &Settings{Hooks: map[string][]HookGroup{}}, nil
	}

	var hooks map[string][]HookGroup
	if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
		return nil, fmt.Errorf("failed to parse hooks section: %w", err)
	}

	return &Settings{Hooks: hooks}, nil
}

// DefaultSettingsPath returns the default path to settings.json.
func DefaultSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".claude", "settings.json")
	}
	return filepath.Join(home, ".claude", "settings.json")
}

// ResolveHooks finds all hooks that would fire for a given event and tool.
// For events without matchers (SessionStart, SessionEnd, etc.), tool can be empty.
func (s *Settings) ResolveHooks(event, tool string) []ResolvedHook {
	groups, ok := s.Hooks[event]
	if !ok {
		return nil
	}

	var resolved []ResolvedHook
	for _, group := range groups {
		if matchesTool(group.Matcher, tool) {
			for _, h := range group.Hooks {
				resolved = append(resolved, ResolvedHook{
					Event:       event,
					Matcher:     group.Matcher,
					Command:     h.Command,
					Timeout:     h.Timeout,
					ExpandedCmd: expandCommand(h.Command),
				})
			}
		}
	}
	return resolved
}

// ListEvents returns all event types that have hooks configured.
func (s *Settings) ListEvents() []string {
	var events []string
	for event := range s.Hooks {
		events = append(events, event)
	}
	return events
}

// AllHooks returns all hooks across all events, flattened.
func (s *Settings) AllHooks() []ResolvedHook {
	var all []ResolvedHook
	for event, groups := range s.Hooks {
		for _, group := range groups {
			for _, h := range group.Hooks {
				all = append(all, ResolvedHook{
					Event:       event,
					Matcher:     group.Matcher,
					Command:     h.Command,
					Timeout:     h.Timeout,
					ExpandedCmd: expandCommand(h.Command),
				})
			}
		}
	}
	return all
}

// matchesTool checks if a tool name matches a hook group's matcher pattern.
// Empty matcher matches everything (used for events like SessionStart).
// Matchers use regex (e.g., "Bash", "Read|Edit", "Task").
func matchesTool(matcher, tool string) bool {
	if matcher == "" {
		return true
	}
	if tool == "" {
		return true
	}
	// Try exact match first (most common case)
	if matcher == tool {
		return true
	}
	// Try regex match for patterns like "Read|Edit"
	re, err := regexp.Compile("^(" + matcher + ")$")
	if err != nil {
		return false
	}
	return re.MatchString(tool)
}

// ExpandCommand expands environment variables in hook commands.
// Exported for use by CLI commands that need to expand paths.
func ExpandCommand(cmd string) string {
	return expandCommand(cmd)
}

// expandCommand expands environment variables in hook commands.
func expandCommand(cmd string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return cmd
	}
	expanded := strings.ReplaceAll(cmd, "$HOME", home)
	expanded = strings.ReplaceAll(expanded, "${HOME}", home)
	return expanded
}
