package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStandaloneDenyRulesExcludeOrchPaths(t *testing.T) {
	rules := StandaloneDenyRules()
	for _, rule := range rules {
		if strings.Contains(rule, "orch") {
			t.Errorf("standalone deny rules should not reference orch paths: %s", rule)
		}
	}
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

func TestEnsureDenyRules(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")

	initial := map[string]any{
		"permissions": map[string]any{
			"allow": []string{"Read(**)", "Glob(**)", "Grep(**)"},
		},
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	os.WriteFile(sp, data, 0644)

	result, err := EnsureDenyRules(sp, StandaloneDenyRules())
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
	data, _ = os.ReadFile(sp)
	var settings map[string]any
	json.Unmarshal(data, &settings)

	perms := settings["permissions"].(map[string]any)
	deny := perms["deny"].([]any)
	if len(deny) < 4 {
		t.Errorf("expected at least 4 deny rules, got %d", len(deny))
	}

	// Verify existing allow rules preserved
	allow := perms["allow"].([]any)
	if len(allow) != 3 {
		t.Error("existing allow rules were not preserved")
	}
}

func TestEnsureDenyRulesIdempotent(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")

	initial := map[string]any{
		"permissions": map[string]any{
			"deny": []string{
				"Edit(~/.claude/settings.json)",
				"Edit(~/.claude/settings.local.json)",
				"Write(~/.claude/settings.json)",
				"Write(~/.claude/settings.local.json)",
			},
		},
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	os.WriteFile(sp, data, 0644)

	result, err := EnsureDenyRules(sp, StandaloneDenyRules())
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected all rules already present")
	}
}

func TestCheckDenyRules(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")

	initial := map[string]any{
		"permissions": map[string]any{
			"deny": []string{
				"Edit(~/.claude/settings.json)",
			},
		},
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	os.WriteFile(sp, data, 0644)

	result, err := CheckDenyRules(sp, StandaloneDenyRules())
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected missing rules")
	}
}

func TestEnsureHookScripts(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".claude", "hooks")

	result, err := EnsureHookScripts(hooksDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected scripts to be created")
	}
	if !result.Created {
		t.Error("expected Created=true")
	}

	gatePath := filepath.Join(hooksDir, "gate-git-add-all.py")
	info, err := os.Stat(gatePath)
	if err != nil {
		t.Fatalf("gate-git-add-all.py not created: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("gate-git-add-all.py should be executable")
	}

	data, _ := os.ReadFile(gatePath)
	content := string(data)
	if !strings.Contains(content, "WHY THIS GATE EXISTS") {
		t.Error("generated hook should contain explanatory comments")
	}
}

func TestEnsureHookScriptsIdempotent(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".claude", "hooks")

	EnsureHookScripts(hooksDir)
	result, err := EnsureHookScripts(hooksDir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected scripts already present on second run")
	}
}

func TestEnsureHookRegistration(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "project", ".claude", "hooks")

	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "gate-git-add-all.py"), []byte("# hook"), 0755)
	os.WriteFile(sp, []byte("{}"), 0644)

	result, err := EnsureHookRegistration(sp, hooksDir, StandaloneHookSpecs, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected hook to be registered")
	}

	data, _ := os.ReadFile(sp)
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]any)
	ptu := hooks["PreToolUse"].([]any)
	if len(ptu) == 0 {
		t.Error("expected at least one PreToolUse entry")
	}

	// Verify relative path used
	group := ptu[0].(map[string]any)
	hookList := group["hooks"].([]any)
	hookMap := hookList[0].(map[string]any)
	cmd := hookMap["command"].(string)
	if !strings.HasPrefix(cmd, "python3 .claude/hooks/") {
		t.Errorf("expected relative path for portability, got %s", cmd)
	}
}

func TestEnsureHookRegistrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "project", ".claude", "hooks")

	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "gate-git-add-all.py"), []byte("# hook"), 0755)
	os.WriteFile(sp, []byte("{}"), 0644)

	EnsureHookRegistration(sp, hooksDir, StandaloneHookSpecs, true, "")
	result, err := EnsureHookRegistration(sp, hooksDir, StandaloneHookSpecs, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected hooks already registered on second run")
	}
}

func TestEnsureHookRegistrationSkipsWhenEquivalentInUserSettings(t *testing.T) {
	dir := t.TempDir()

	userSP := filepath.Join(dir, "user-settings.json")
	userSettings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "~/.orch/hooks/gate-worker-git-add-all.py",
						},
					},
				},
			},
		},
	}
	userData, _ := json.Marshal(userSettings)
	os.WriteFile(userSP, userData, 0644)

	projectSP := filepath.Join(dir, "project-settings.json")
	hooksDir := filepath.Join(dir, "project", ".claude", "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(projectSP, []byte("{}"), 0644)
	os.WriteFile(filepath.Join(hooksDir, "gate-git-add-all.py"), []byte("# hook"), 0755)

	result, err := EnsureHookRegistration(projectSP, hooksDir, StandaloneHookSpecs, true, userSP)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected hook to be detected as already present (equivalent in user settings)")
	}

	// Project settings should NOT be modified
	data, _ := os.ReadFile(projectSP)
	if string(data) != "{}" {
		t.Error("project settings should not be modified when equivalent exists")
	}
}

func TestIsEquivalentHookRegistered(t *testing.T) {
	tests := []struct {
		name       string
		commands   map[string]bool
		scriptName string
		want       bool
	}{
		{
			name:       "full-mode equivalent exists",
			commands:   map[string]bool{"~/.orch/hooks/gate-worker-git-add-all.py": true},
			scriptName: "gate-git-add-all.py",
			want:       true,
		},
		{
			name:       "no equivalent",
			commands:   map[string]bool{"~/.orch/hooks/gate-bd-close.py": true},
			scriptName: "gate-git-add-all.py",
			want:       false,
		},
		{
			name:       "empty commands",
			commands:   map[string]bool{},
			scriptName: "gate-git-add-all.py",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEquivalentHookRegistered(tt.commands, tt.scriptName)
			if got != tt.want {
				t.Errorf("IsEquivalentHookRegistered() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveTrailingExit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trailing exit 0",
			input:    "#!/bin/bash\necho hello\nexit 0\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "no trailing exit",
			input:    "#!/bin/bash\necho hello\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "exit 1 not removed",
			input:    "#!/bin/bash\necho hello\nexit 1\n",
			expected: "#!/bin/bash\necho hello\nexit 1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveTrailingExit(tt.input)
			if got != tt.expected {
				t.Errorf("RemoveTrailingExit(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEnsureBashShebang(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sh to bash",
			input:    "#!/bin/sh\necho hello\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "already bash",
			input:    "#!/bin/bash\necho hello\n",
			expected: "#!/bin/bash\necho hello\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureBashShebang(tt.input)
			if got != tt.expected {
				t.Errorf("EnsureBashShebang(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEnsureStandalonePreCommitGate(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)

	result, err := EnsureStandalonePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected pre-commit to be created")
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	data, _ := os.ReadFile(hookPath)
	content := string(data)

	if !strings.Contains(content, "Accretion Gate") {
		t.Error("pre-commit should contain accretion gate")
	}
	if strings.Contains(content, "orch precommit") {
		t.Error("standalone pre-commit should not reference orch CLI")
	}
}

func TestEnsureStandalonePreCommitGateIdempotent(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)

	EnsureStandalonePreCommitGate(dir)
	result, err := EnsureStandalonePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected gate already present on second run")
	}
}

func TestEnsureStandalonePreCommitGateAppendsBeforeExitZero(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	existing := "#!/bin/bash\necho 'existing hook'\nexit 0\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	result, err := EnsureStandalonePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected accretion gate to be added")
	}

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)

	if !strings.Contains(content, "Accretion Gate") {
		t.Error("accretion gate not found in hook")
	}
	if !strings.Contains(content, "existing hook") {
		t.Error("original hook content not preserved")
	}
}

func TestEnsureBeadsCloseHook(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	os.MkdirAll(beadsDir, 0755)

	result, err := EnsureBeadsCloseHook(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected hook to be created")
	}

	hookPath := filepath.Join(beadsDir, "hooks", "on_close")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("hook file not created: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("hook file should be executable")
	}
}

func TestEnsureBeadsCloseHookIdempotent(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads", "hooks")
	os.MkdirAll(beadsDir, 0755)
	os.WriteFile(filepath.Join(beadsDir, "on_close"), []byte("#!/bin/bash\nexisting"), 0755)

	result, err := EnsureBeadsCloseHook(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected hook already present")
	}

	data, _ := os.ReadFile(filepath.Join(beadsDir, "on_close"))
	if string(data) != "#!/bin/bash\nexisting" {
		t.Error("existing hook was overwritten")
	}
}

func TestShortPath(t *testing.T) {
	home := "/Users/test"
	tests := []struct {
		path     string
		expected string
	}{
		{"/Users/test/.claude/settings.json", "~/.claude/settings.json"},
		{"/other/path/file.go", "/other/path/file.go"},
	}
	for _, tt := range tests {
		got := ShortPath(tt.path, home)
		if got != tt.expected {
			t.Errorf("ShortPath(%q, %q) = %q, want %q", tt.path, home, got, tt.expected)
		}
	}
}

func TestCollectRegisteredCommands(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "~/.orch/hooks/gate-worker-git-add-all.py",
						},
					},
				},
			},
		},
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(sp, data, 0644)

	commands := CollectRegisteredCommands(sp)
	if !commands["~/.orch/hooks/gate-worker-git-add-all.py"] {
		t.Error("expected command to be collected")
	}
}

func TestCollectRegisteredCommandsMissingFile(t *testing.T) {
	commands := CollectRegisteredCommands("/nonexistent/settings.json")
	if len(commands) != 0 {
		t.Errorf("expected empty map for missing file, got %d entries", len(commands))
	}
}

func TestSettingsPath(t *testing.T) {
	standalone := SettingsPath(ModeStandalone, "/project")
	if standalone != "/project/.claude/settings.json" {
		t.Errorf("standalone settings path wrong: %s", standalone)
	}

	// Full mode should use default (user-level)
	os.Unsetenv("ORCH_SETTINGS_PATH")
	full := SettingsPath(ModeFull, "/project")
	if strings.Contains(full, "/project/") {
		t.Error("full mode should not use project-level path")
	}
}
