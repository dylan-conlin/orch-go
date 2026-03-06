package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// HookEntry represents a hook with its event and matcher context for listing.
type HookEntry struct {
	Event   string
	Matcher string
	Command string
	Timeout int
}

// AddHook adds a hook command to a specific event type and matcher in settings.json.
// If a group with the same matcher already exists, the hook is appended to it.
// If no matching group exists, a new group is created.
// Returns an error if the exact command already exists in the matching group.
func AddHook(path, event, matcher, command string, timeout int) error {
	full, err := readFullSettings(path)
	if err != nil {
		return err
	}

	hooks := getOrCreateHooks(full)

	// Parse existing groups for this event
	var groups []HookGroup
	if raw, ok := hooks[event]; ok {
		data, _ := json.Marshal(raw)
		if err := json.Unmarshal(data, &groups); err != nil {
			return fmt.Errorf("failed to parse %s hooks: %w", event, err)
		}
	}

	// Check for duplicate and find matching group
	matchIdx := -1
	for i, g := range groups {
		if g.Matcher == matcher {
			matchIdx = i
			for _, h := range g.Hooks {
				if h.Command == command {
					return fmt.Errorf("hook %q already exists in %s (matcher: %q)", command, event, matcher)
				}
			}
		}
	}

	newHook := HookConfig{
		Type:    "command",
		Command: command,
		Timeout: timeout,
	}

	if matchIdx >= 0 {
		groups[matchIdx].Hooks = append(groups[matchIdx].Hooks, newHook)
	} else {
		groups = append(groups, HookGroup{
			Matcher: matcher,
			Hooks:   []HookConfig{newHook},
		})
	}

	hooks[event] = groups
	full["hooks"], _ = marshalRaw(hooks)

	return writeFullSettings(path, full)
}

// RemoveHook removes a hook matching the given command substring from the specified event.
// Returns true if a hook was removed, false if not found.
// Empty groups are cleaned up after removal.
func RemoveHook(path, event, commandSubstr string) (bool, error) {
	full, err := readFullSettings(path)
	if err != nil {
		return false, err
	}

	hooksRaw, ok := full["hooks"]
	if !ok {
		return false, nil
	}

	var allHooks map[string]json.RawMessage
	if err := json.Unmarshal(hooksRaw, &allHooks); err != nil {
		return false, fmt.Errorf("failed to parse hooks: %w", err)
	}

	eventRaw, ok := allHooks[event]
	if !ok {
		return false, nil
	}

	var groups []HookGroup
	if err := json.Unmarshal(eventRaw, &groups); err != nil {
		return false, fmt.Errorf("failed to parse %s hooks: %w", event, err)
	}

	removed := false
	var newGroups []HookGroup
	for _, g := range groups {
		var remaining []HookConfig
		for _, h := range g.Hooks {
			if strings.Contains(h.Command, commandSubstr) {
				removed = true
			} else {
				remaining = append(remaining, h)
			}
		}
		if len(remaining) > 0 {
			g.Hooks = remaining
			newGroups = append(newGroups, g)
		}
	}

	if !removed {
		return false, nil
	}

	allHooks[event], _ = marshalRaw(newGroups)
	full["hooks"], _ = marshalRaw(allHooks)

	return true, writeFullSettings(path, full)
}

// ListHooks returns all hooks, optionally filtered by event type.
func ListHooks(path, eventFilter string) ([]HookEntry, error) {
	settings, err := LoadSettingsFromPath(path)
	if err != nil {
		return nil, err
	}

	var entries []HookEntry
	for event, groups := range settings.Hooks {
		if eventFilter != "" && event != eventFilter {
			continue
		}
		for _, g := range groups {
			for _, h := range g.Hooks {
				entries = append(entries, HookEntry{
					Event:   event,
					Matcher: g.Matcher,
					Command: h.Command,
					Timeout: h.Timeout,
				})
			}
		}
	}
	return entries, nil
}

// readFullSettings reads settings.json preserving all fields as raw JSON.
func readFullSettings(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]json.RawMessage), nil
		}
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	var full map[string]json.RawMessage
	if err := json.Unmarshal(data, &full); err != nil {
		return nil, fmt.Errorf("failed to parse settings: %w", err)
	}
	return full, nil
}

// writeFullSettings writes the full settings map back to disk with indentation.
func writeFullSettings(path string, full map[string]json.RawMessage) error {
	data, err := json.MarshalIndent(full, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

// getOrCreateHooks extracts or creates the hooks map from full settings.
func getOrCreateHooks(full map[string]json.RawMessage) map[string]interface{} {
	hooks := make(map[string]interface{})
	if raw, ok := full["hooks"]; ok {
		json.Unmarshal(raw, &hooks)
	}
	return hooks
}

// marshalRaw converts a value to json.RawMessage.
func marshalRaw(v interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	return json.RawMessage(data), err
}
