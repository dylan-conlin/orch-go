package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettingsFromPath(t *testing.T) {
	// Create temp settings file
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	content := `{
		"hooks": {
			"PreToolUse": [
				{"matcher": "Bash", "hooks": [{"type": "command", "command": "$HOME/.orch/hooks/gate.py", "timeout": 10}]},
				{"matcher": "Read|Edit", "hooks": [{"type": "command", "command": "$HOME/.orch/hooks/access.py", "timeout": 10}]},
				{"matcher": "Task", "hooks": [{"type": "command", "command": "$HOME/.orch/hooks/task-gate.py"}]}
			],
			"SessionStart": [
				{"hooks": [{"type": "command", "command": "$HOME/.claude/hooks/session-start.sh", "timeout": 10}]}
			],
			"Stop": []
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	settings, err := LoadSettingsFromPath(path)
	if err != nil {
		t.Fatalf("LoadSettingsFromPath failed: %v", err)
	}

	// Check events parsed
	if len(settings.Hooks) != 3 {
		t.Errorf("expected 3 event types, got %d", len(settings.Hooks))
	}

	// Check PreToolUse has 3 groups
	groups := settings.Hooks["PreToolUse"]
	if len(groups) != 3 {
		t.Errorf("expected 3 PreToolUse groups, got %d", len(groups))
	}

	// Check first group
	if groups[0].Matcher != "Bash" {
		t.Errorf("expected matcher 'Bash', got '%s'", groups[0].Matcher)
	}
	if len(groups[0].Hooks) != 1 {
		t.Errorf("expected 1 hook in first group, got %d", len(groups[0].Hooks))
	}
	if groups[0].Hooks[0].Timeout != 10 {
		t.Errorf("expected timeout 10, got %d", groups[0].Hooks[0].Timeout)
	}

	// Check SessionStart (no matcher)
	ssGroups := settings.Hooks["SessionStart"]
	if len(ssGroups) != 1 {
		t.Errorf("expected 1 SessionStart group, got %d", len(ssGroups))
	}
	if ssGroups[0].Matcher != "" {
		t.Errorf("expected empty matcher for SessionStart, got '%s'", ssGroups[0].Matcher)
	}

	// Check Stop is empty
	stopGroups := settings.Hooks["Stop"]
	if len(stopGroups) != 0 {
		t.Errorf("expected 0 Stop groups, got %d", len(stopGroups))
	}
}

func TestLoadSettingsFromPath_NoHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	content := `{"model": "opus"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	settings, err := LoadSettingsFromPath(path)
	if err != nil {
		t.Fatalf("LoadSettingsFromPath failed: %v", err)
	}
	if len(settings.Hooks) != 0 {
		t.Errorf("expected 0 hooks, got %d", len(settings.Hooks))
	}
}

func TestLoadSettingsFromPath_FileNotFound(t *testing.T) {
	_, err := LoadSettingsFromPath("/nonexistent/settings.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestResolveHooks_ExactMatch(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Bash", Hooks: []HookConfig{{Type: "command", Command: "gate.py", Timeout: 10}}},
				{Matcher: "Task", Hooks: []HookConfig{{Type: "command", Command: "task-gate.py"}}},
			},
		},
	}

	resolved := s.ResolveHooks("PreToolUse", "Bash")
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved hook, got %d", len(resolved))
	}
	if resolved[0].Command != "gate.py" {
		t.Errorf("expected command 'gate.py', got '%s'", resolved[0].Command)
	}
	if resolved[0].Event != "PreToolUse" {
		t.Errorf("expected event 'PreToolUse', got '%s'", resolved[0].Event)
	}
}

func TestResolveHooks_RegexMatch(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Read|Edit", Hooks: []HookConfig{{Type: "command", Command: "access.py"}}},
			},
		},
	}

	// Should match Read
	resolved := s.ResolveHooks("PreToolUse", "Read")
	if len(resolved) != 1 {
		t.Fatalf("expected 1 hook for Read, got %d", len(resolved))
	}

	// Should match Edit
	resolved = s.ResolveHooks("PreToolUse", "Edit")
	if len(resolved) != 1 {
		t.Fatalf("expected 1 hook for Edit, got %d", len(resolved))
	}

	// Should NOT match Write
	resolved = s.ResolveHooks("PreToolUse", "Write")
	if len(resolved) != 0 {
		t.Errorf("expected 0 hooks for Write, got %d", len(resolved))
	}
}

func TestResolveHooks_EmptyMatcher(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"SessionStart": {
				{Matcher: "", Hooks: []HookConfig{{Type: "command", Command: "session-start.sh"}}},
			},
		},
	}

	// Empty matcher should match any tool (or empty tool)
	resolved := s.ResolveHooks("SessionStart", "")
	if len(resolved) != 1 {
		t.Fatalf("expected 1 hook for empty tool, got %d", len(resolved))
	}

	resolved = s.ResolveHooks("SessionStart", "Bash")
	if len(resolved) != 1 {
		t.Fatalf("expected 1 hook for Bash with empty matcher, got %d", len(resolved))
	}
}

func TestResolveHooks_MultipleHooksInGroup(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Bash", Hooks: []HookConfig{
					{Type: "command", Command: "hook1.py", Timeout: 10},
					{Type: "command", Command: "hook2.py", Timeout: 10},
				}},
			},
		},
	}

	resolved := s.ResolveHooks("PreToolUse", "Bash")
	if len(resolved) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(resolved))
	}
	if resolved[0].Command != "hook1.py" {
		t.Errorf("expected hook1.py, got %s", resolved[0].Command)
	}
	if resolved[1].Command != "hook2.py" {
		t.Errorf("expected hook2.py, got %s", resolved[1].Command)
	}
}

func TestResolveHooks_UnknownEvent(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Bash", Hooks: []HookConfig{{Type: "command", Command: "gate.py"}}},
			},
		},
	}

	resolved := s.ResolveHooks("UnknownEvent", "Bash")
	if len(resolved) != 0 {
		t.Errorf("expected 0 hooks for unknown event, got %d", len(resolved))
	}
}

func TestMatchesTool(t *testing.T) {
	tests := []struct {
		matcher string
		tool    string
		want    bool
	}{
		{"", "", true},
		{"", "Bash", true},
		{"Bash", "", true},
		{"Bash", "Bash", true},
		{"Bash", "Read", false},
		{"Read|Edit", "Read", true},
		{"Read|Edit", "Edit", true},
		{"Read|Edit", "Write", false},
		{"Read|Glob|Grep", "Glob", true},
		{"Read|Glob|Grep", "Task", false},
		{"Task", "Task", true},
		{"Edit|Write", "Edit", true},
	}

	for _, tt := range tests {
		got := matchesTool(tt.matcher, tt.tool)
		if got != tt.want {
			t.Errorf("matchesTool(%q, %q) = %v, want %v", tt.matcher, tt.tool, got, tt.want)
		}
	}
}

func TestExpandCommand(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"$HOME/.orch/hooks/gate.py", home + "/.orch/hooks/gate.py"},
		{"${HOME}/.orch/hooks/gate.py", home + "/.orch/hooks/gate.py"},
		{"bd prime", "bd prime"},
		{"/usr/bin/test.sh", "/usr/bin/test.sh"},
	}

	for _, tt := range tests {
		got := expandCommand(tt.input)
		if got != tt.want {
			t.Errorf("expandCommand(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAllHooks(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Bash", Hooks: []HookConfig{{Type: "command", Command: "gate.py"}}},
				{Matcher: "Task", Hooks: []HookConfig{{Type: "command", Command: "task-gate.py"}}},
			},
			"SessionStart": {
				{Hooks: []HookConfig{{Type: "command", Command: "session-start.sh"}}},
			},
		},
	}

	all := s.AllHooks()
	if len(all) != 3 {
		t.Errorf("expected 3 total hooks, got %d", len(all))
	}
}
