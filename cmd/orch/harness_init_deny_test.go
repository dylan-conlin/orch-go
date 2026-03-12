package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessInitDenyRules(t *testing.T) {
	// Create a temp settings.json with no deny rules
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")

	initial := map[string]any{
		"permissions": map[string]any{
			"allow": []string{"Read(**)", "Glob(**)", "Grep(**)"},
		},
	}
	os.WriteFile(sp, []byte(settingsJSON(t, initial)), 0644)

	result, err := ensureDenyRules(sp)
	if err != nil {
		t.Fatal(err)
	}

	if result.AlreadyPresent {
		t.Error("expected deny rules to be added, not already present")
	}
	if result.RulesAdded == 0 {
		t.Error("expected at least one rule added")
	}

	// Verify file was written correctly
	data, _ := os.ReadFile(sp)
	var settings map[string]any
	json.Unmarshal(data, &settings)

	perms, ok := settings["permissions"].(map[string]any)
	if !ok {
		t.Fatal("permissions missing after update")
	}
	deny, ok := perms["deny"].([]any)
	if !ok {
		t.Fatal("deny rules missing after update")
	}
	if len(deny) < 6 {
		t.Errorf("expected at least 6 deny rules, got %d", len(deny))
	}

	// Verify existing allow rules preserved
	allow, ok := perms["allow"].([]any)
	if !ok || len(allow) != 3 {
		t.Error("existing allow rules were not preserved")
	}
}

func TestHarnessInitDenyRulesIdempotent(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")

	initial := map[string]any{
		"permissions": map[string]any{
			"deny": []string{
				"Edit(~/.claude/settings.json)",
				"Write(~/.claude/settings.json)",
				"Edit(~/.claude/settings.local.json)",
				"Write(~/.claude/settings.local.json)",
				"Edit(~/.orch/hooks/**)",
				"Write(~/.orch/hooks/**)",
			},
		},
	}
	os.WriteFile(sp, []byte(settingsJSON(t, initial)), 0644)

	result, err := ensureDenyRules(sp)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected all rules already present")
	}
	if result.RulesAdded != 0 {
		t.Errorf("expected 0 rules added, got %d", result.RulesAdded)
	}
}

func TestHarnessInitDenyRulesPartial(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")

	// Only some rules present
	initial := map[string]any{
		"permissions": map[string]any{
			"deny": []string{
				"Edit(~/.claude/settings.json)",
				"Write(~/.claude/settings.json)",
			},
		},
	}
	os.WriteFile(sp, []byte(settingsJSON(t, initial)), 0644)

	result, err := ensureDenyRules(sp)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("should not be already present (only partial)")
	}
	if result.RulesAdded != 4 {
		t.Errorf("expected 4 rules added, got %d", result.RulesAdded)
	}

	// Verify original rules still there (no duplicates)
	data, _ := os.ReadFile(sp)
	var settings map[string]any
	json.Unmarshal(data, &settings)
	perms := settings["permissions"].(map[string]any)
	deny := perms["deny"].([]any)
	if len(deny) != 6 {
		t.Errorf("expected 6 total deny rules, got %d", len(deny))
	}
}

func TestStandaloneDenyRulesExcludeOrchPaths(t *testing.T) {
	rules := standaloneDenyRules()
	for _, rule := range rules {
		if strings.Contains(rule, "orch") {
			t.Errorf("standalone deny rules should not reference orch paths: %s", rule)
		}
	}
	// Should still protect settings.json
	found := false
	for _, rule := range rules {
		if strings.Contains(rule, "settings.json") {
			found = true
			break
		}
	}
	if !found {
		t.Error("standalone deny rules must protect settings.json")
	}
}
