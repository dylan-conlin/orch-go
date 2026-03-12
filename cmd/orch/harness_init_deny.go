package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/control"
)

// ensureDenyRules adds missing deny rules to settings.json.
// Preserves existing content. Returns result with count of rules added.
func ensureDenyRules(settingsPath string) (*StepResult, error) {
	return ensureDenyRulesWithRules(settingsPath, control.DenyRules())
}

// checkDenyRules checks deny rules without modifying settings.json.
func checkDenyRules(settingsPath string) (*StepResult, error) {
	return checkDenyRulesWithRules(settingsPath, control.DenyRules())
}

// standaloneDenyRules returns deny rules for standalone mode.
// These protect Claude Code's settings files from agent modification.
// Unlike the full mode, these don't include ~/.orch/hooks/** paths.
func standaloneDenyRules() []string {
	return []string{
		"Edit(~/.claude/settings.json)",
		"Edit(~/.claude/settings.local.json)",
		"Write(~/.claude/settings.json)",
		"Write(~/.claude/settings.local.json)",
	}
}

// ensureDenyRulesWithRules adds the given deny rules to settings.json.
func ensureDenyRulesWithRules(settingsPath string, rules []string) (*StepResult, error) {
	result := &StepResult{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("reading settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	perms, ok := settings["permissions"].(map[string]any)
	if !ok {
		perms = make(map[string]any)
		settings["permissions"] = perms
	}

	var existing []string
	if denyRaw, ok := perms["deny"].([]any); ok {
		for _, r := range denyRaw {
			if s, ok := r.(string); ok {
				existing = append(existing, s)
			}
		}
	}

	existingSet := make(map[string]bool)
	for _, r := range existing {
		existingSet[r] = true
	}

	var added []string
	for _, rule := range rules {
		if !existingSet[rule] {
			existing = append(existing, rule)
			added = append(added, rule)
		}
	}

	if len(added) == 0 {
		result.AlreadyPresent = true
		return result, nil
	}

	denyAny := make([]any, len(existing))
	for i, s := range existing {
		denyAny[i] = s
	}
	perms["deny"] = denyAny

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return nil, fmt.Errorf("writing settings: %w", err)
	}

	result.RulesAdded = len(added)
	return result, nil
}

// checkDenyRulesWithRules checks deny rules without modifying settings.json.
func checkDenyRulesWithRules(settingsPath string, rules []string) (*StepResult, error) {
	result := &StepResult{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	existingSet := make(map[string]bool)
	if perms, ok := settings["permissions"].(map[string]any); ok {
		if denyRaw, ok := perms["deny"].([]any); ok {
			for _, r := range denyRaw {
				if s, ok := r.(string); ok {
					existingSet[s] = true
				}
			}
		}
	}

	missing := 0
	for _, rule := range rules {
		if !existingSet[rule] {
			missing++
		}
	}

	result.AlreadyPresent = missing == 0
	result.RulesAdded = len(rules) - missing // count of present rules (reused field for dry-run display)
	return result, nil
}
