package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessInitHookRegistration(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")

	// Create hook files that will be referenced
	hooksDir := filepath.Join(dir, "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "gate-bd-close.py"), []byte("# hook"), 0644)
	os.WriteFile(filepath.Join(hooksDir, "gate-worker-git-add-all.py"), []byte("# hook"), 0644)

	// Empty settings
	os.WriteFile(sp, []byte("{}"), 0644)

	result, err := ensureHookRegistration(sp, hooksDir)
	if err != nil {
		t.Fatal(err)
	}

	if result.AlreadyPresent {
		t.Error("expected hooks to be added")
	}
	if result.HooksRegistered == 0 {
		t.Error("expected at least one hook registered")
	}

	// Verify settings.json has hooks
	data, _ := os.ReadFile(sp)
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks section missing")
	}
	ptu, ok := hooks["PreToolUse"].([]any)
	if !ok {
		t.Fatal("PreToolUse section missing")
	}
	if len(ptu) == 0 {
		t.Error("expected at least one PreToolUse entry")
	}
}

func TestHarnessInitHookRegistrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "gate-bd-close.py"), []byte("# hook"), 0644)
	os.WriteFile(filepath.Join(hooksDir, "gate-worker-git-add-all.py"), []byte("# hook"), 0644)

	os.WriteFile(sp, []byte("{}"), 0644)

	// Run twice
	ensureHookRegistration(sp, hooksDir)
	result, err := ensureHookRegistration(sp, hooksDir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected hooks already registered on second run")
	}
}

func TestEnsureStandaloneHookScripts(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".claude", "hooks")

	result, err := ensureStandaloneHookScripts(hooksDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected scripts to be created")
	}
	if result.Created != true {
		t.Error("expected Created=true")
	}

	// Verify gate-git-add-all.py was created
	gatePath := filepath.Join(hooksDir, "gate-git-add-all.py")
	info, err := os.Stat(gatePath)
	if err != nil {
		t.Fatalf("gate-git-add-all.py not created: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("gate-git-add-all.py should be executable")
	}

	// Verify it has explanatory comments
	data, _ := os.ReadFile(gatePath)
	content := string(data)
	if !strings.Contains(content, "WHY THIS GATE EXISTS") {
		t.Error("generated hook should contain explanatory comments")
	}
	// Should NOT reference CLAUDE_CONTEXT (orch-specific)
	if strings.Contains(content, "CLAUDE_CONTEXT") {
		t.Error("standalone hook should not reference CLAUDE_CONTEXT")
	}
}

func TestEnsureStandaloneHookScriptsIdempotent(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".claude", "hooks")

	ensureStandaloneHookScripts(hooksDir)
	result, err := ensureStandaloneHookScripts(hooksDir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected scripts already present on second run")
	}
}

func TestEnsureStandaloneHookRegistration(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "project", ".claude", "hooks")

	// Create the hook script first
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "gate-git-add-all.py"), []byte("# hook"), 0755)

	os.WriteFile(sp, []byte("{}"), 0644)

	// Pass empty user settings path to avoid detecting real user hooks
	result, err := ensureStandaloneHookRegistration(sp, hooksDir, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected hook to be registered")
	}

	// Verify settings.json has the hook
	data, _ := os.ReadFile(sp)
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks section missing")
	}
	ptu, ok := hooks["PreToolUse"].([]any)
	if !ok {
		t.Fatal("PreToolUse section missing")
	}
	if len(ptu) == 0 {
		t.Error("expected at least one PreToolUse entry")
	}

	// Verify the registered command uses a relative path for portability
	group := ptu[0].(map[string]any)
	hookList := group["hooks"].([]any)
	hookMap := hookList[0].(map[string]any)
	cmd := hookMap["command"].(string)
	if !strings.Contains(cmd, "gate-git-add-all.py") {
		t.Errorf("expected command to reference gate-git-add-all.py, got %s", cmd)
	}
	if !strings.HasPrefix(cmd, "python3 .claude/hooks/") {
		t.Errorf("expected relative path for portability, got %s", cmd)
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
			name:       "full-mode equivalent with python3 prefix",
			commands:   map[string]bool{"python3 ~/.orch/hooks/gate-worker-git-add-all.py": true},
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
		{
			name:       "unknown script name",
			commands:   map[string]bool{"~/.orch/hooks/gate-worker-git-add-all.py": true},
			scriptName: "gate-unknown.py",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEquivalentHookRegistered(tt.commands, tt.scriptName)
			if got != tt.want {
				t.Errorf("isEquivalentHookRegistered() = %v, want %v", got, tt.want)
			}
		})
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

	commands := collectRegisteredCommands(sp)
	if !commands["~/.orch/hooks/gate-worker-git-add-all.py"] {
		t.Error("expected gate-worker-git-add-all.py command to be collected")
	}
}

func TestCollectRegisteredCommandsMissingFile(t *testing.T) {
	commands := collectRegisteredCommands("/nonexistent/settings.json")
	if len(commands) != 0 {
		t.Errorf("expected empty map for missing file, got %d entries", len(commands))
	}
}

func TestStandaloneHookSkipsWhenEquivalentInUserSettings(t *testing.T) {
	dir := t.TempDir()

	// Simulate user-level settings with an equivalent hook already registered
	userSettingsDir := filepath.Join(dir, "user-home", ".claude")
	os.MkdirAll(userSettingsDir, 0755)
	userSP := filepath.Join(userSettingsDir, "settings.json")
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

	// Project settings (empty)
	projectSP := filepath.Join(dir, "project", ".claude", "settings.json")
	hooksDir := filepath.Join(dir, "project", ".claude", "hooks")
	os.MkdirAll(filepath.Dir(projectSP), 0755)
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(projectSP, []byte("{}"), 0644)
	os.WriteFile(filepath.Join(hooksDir, "gate-git-add-all.py"), []byte("# hook"), 0755)

	// Direct test: verify isEquivalentHookRegistered catches it
	userCommands := collectRegisteredCommands(userSP)
	if !isEquivalentHookRegistered(userCommands, "gate-git-add-all.py") {
		t.Error("expected equivalent hook to be detected in user settings")
	}
}

func TestStandaloneHookRegistrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "project", ".claude", "hooks")

	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "gate-git-add-all.py"), []byte("# hook"), 0755)
	os.WriteFile(sp, []byte("{}"), 0644)

	// Run twice — pass empty user settings to avoid detecting real user hooks
	ensureStandaloneHookRegistration(sp, hooksDir, "")
	result, err := ensureStandaloneHookRegistration(sp, hooksDir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected hooks already registered on second run")
	}
}

func TestStandaloneHookSkipsWhenEquivalentInUserSettingsIntegration(t *testing.T) {
	dir := t.TempDir()

	// Simulate user-level settings with an equivalent hook
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

	// Project settings (empty)
	projectSP := filepath.Join(dir, "project-settings.json")
	hooksDir := filepath.Join(dir, "project", ".claude", "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(projectSP, []byte("{}"), 0644)
	os.WriteFile(filepath.Join(hooksDir, "gate-git-add-all.py"), []byte("# hook"), 0755)

	// Should skip registration because equivalent hook exists in user settings
	result, err := ensureStandaloneHookRegistration(projectSP, hooksDir, userSP)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected hook to be detected as already present (equivalent in user settings)")
	}

	// Verify project settings was NOT modified
	data, _ := os.ReadFile(projectSP)
	if string(data) != "{}" {
		t.Error("project settings should not be modified when equivalent exists")
	}
}
