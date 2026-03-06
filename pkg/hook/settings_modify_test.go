package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAddHook(t *testing.T) {
	// Create a temp settings file with existing hooks
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]interface{}{
		"env": map[string]string{"FOO": "bar"},
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "existing-hook.sh",
							"timeout": 10,
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, initial)

	// Add a new hook to PreToolUse with same matcher
	err := AddHook(path, "PreToolUse", "Bash", "new-hook.sh", 5)
	if err != nil {
		t.Fatalf("AddHook failed: %v", err)
	}

	// Verify
	settings, err := LoadSettingsFromPath(path)
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}

	groups := settings.Hooks["PreToolUse"]
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Hooks) != 2 {
		t.Fatalf("expected 2 hooks in group, got %d", len(groups[0].Hooks))
	}
	if groups[0].Hooks[1].Command != "new-hook.sh" {
		t.Errorf("expected new-hook.sh, got %s", groups[0].Hooks[1].Command)
	}
	if groups[0].Hooks[1].Timeout != 5 {
		t.Errorf("expected timeout 5, got %d", groups[0].Hooks[1].Timeout)
	}

	// Verify non-hook fields preserved
	data, _ := os.ReadFile(path)
	var full map[string]json.RawMessage
	json.Unmarshal(data, &full)
	if _, ok := full["env"]; !ok {
		t.Error("env field was lost during modification")
	}
}

func TestAddHookNewMatcher(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "bash-hook.sh",
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, initial)

	// Add hook with different matcher - should create new group
	err := AddHook(path, "PreToolUse", "Read|Edit", "read-hook.sh", 10)
	if err != nil {
		t.Fatalf("AddHook failed: %v", err)
	}

	settings, _ := LoadSettingsFromPath(path)
	groups := settings.Hooks["PreToolUse"]
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[1].Matcher != "Read|Edit" {
		t.Errorf("expected matcher Read|Edit, got %s", groups[1].Matcher)
	}
}

func TestAddHookNewEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]interface{}{
		"hooks": map[string]interface{}{},
	}
	writeJSON(t, path, initial)

	err := AddHook(path, "SessionStart", "", "startup.sh", 10)
	if err != nil {
		t.Fatalf("AddHook failed: %v", err)
	}

	settings, _ := LoadSettingsFromPath(path)
	groups := settings.Hooks["SessionStart"]
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Hooks[0].Command != "startup.sh" {
		t.Errorf("expected startup.sh, got %s", groups[0].Hooks[0].Command)
	}
}

func TestAddHookNoHooksSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]interface{}{
		"env": map[string]string{"FOO": "bar"},
	}
	writeJSON(t, path, initial)

	err := AddHook(path, "SessionStart", "", "startup.sh", 10)
	if err != nil {
		t.Fatalf("AddHook failed: %v", err)
	}

	settings, _ := LoadSettingsFromPath(path)
	if len(settings.Hooks["SessionStart"]) != 1 {
		t.Fatal("expected hook to be added")
	}
}

func TestAddHookDuplicateDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "existing-hook.sh",
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, initial)

	err := AddHook(path, "PreToolUse", "Bash", "existing-hook.sh", 10)
	if err == nil {
		t.Fatal("expected error for duplicate hook, got nil")
	}
}

func TestRemoveHook(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]interface{}{
		"env": map[string]string{"FOO": "bar"},
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "$HOME/.orch/hooks/gate-bd-close.py",
							"timeout": 10,
						},
						map[string]interface{}{
							"type":    "command",
							"command": "$HOME/.orch/hooks/other-hook.py",
							"timeout": 10,
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, initial)

	removed, err := RemoveHook(path, "PreToolUse", "gate-bd-close.py")
	if err != nil {
		t.Fatalf("RemoveHook failed: %v", err)
	}
	if !removed {
		t.Fatal("expected hook to be removed")
	}

	settings, _ := LoadSettingsFromPath(path)
	groups := settings.Hooks["PreToolUse"]
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Hooks) != 1 {
		t.Fatalf("expected 1 hook remaining, got %d", len(groups[0].Hooks))
	}
	if groups[0].Hooks[0].Command != "$HOME/.orch/hooks/other-hook.py" {
		t.Errorf("wrong hook remaining: %s", groups[0].Hooks[0].Command)
	}

	// Verify non-hook fields preserved
	data, _ := os.ReadFile(path)
	var full map[string]json.RawMessage
	json.Unmarshal(data, &full)
	if _, ok := full["env"]; !ok {
		t.Error("env field was lost during modification")
	}
}

func TestRemoveHookCleansEmptyGroup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "only-hook.sh",
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, initial)

	removed, err := RemoveHook(path, "PreToolUse", "only-hook.sh")
	if err != nil {
		t.Fatalf("RemoveHook failed: %v", err)
	}
	if !removed {
		t.Fatal("expected hook to be removed")
	}

	settings, _ := LoadSettingsFromPath(path)
	groups := settings.Hooks["PreToolUse"]
	if len(groups) != 0 {
		t.Fatalf("expected empty groups after removing last hook, got %d", len(groups))
	}
}

func TestRemoveHookNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "some-hook.sh",
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, initial)

	removed, err := RemoveHook(path, "PreToolUse", "nonexistent.sh")
	if err != nil {
		t.Fatalf("RemoveHook failed: %v", err)
	}
	if removed {
		t.Fatal("expected hook to not be found")
	}
}

func TestListHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "hook1.sh",
							"timeout": 10,
						},
					},
				},
				map[string]interface{}{
					"matcher": "Read|Edit",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "hook2.sh",
						},
					},
				},
			},
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "start.sh",
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, initial)

	hooks, err := ListHooks(path, "")
	if err != nil {
		t.Fatalf("ListHooks failed: %v", err)
	}
	if len(hooks) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(hooks))
	}

	// Filter by event
	hooks, err = ListHooks(path, "PreToolUse")
	if err != nil {
		t.Fatalf("ListHooks failed: %v", err)
	}
	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks for PreToolUse, got %d", len(hooks))
	}
}

func writeJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}
