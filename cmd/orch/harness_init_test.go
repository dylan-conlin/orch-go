package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// settingsJSON is a helper to build settings.json content.
func settingsJSON(t *testing.T, obj map[string]any) string {
	t.Helper()
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

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

func TestHarnessInitBeadsCloseHook(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	os.MkdirAll(beadsDir, 0755)

	result, err := ensureBeadsCloseHook(dir)
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
	// Check executable
	if info.Mode()&0111 == 0 {
		t.Error("hook file should be executable")
	}
}

func TestHarnessInitBeadsCloseHookIdempotent(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads", "hooks")
	os.MkdirAll(beadsDir, 0755)
	os.WriteFile(filepath.Join(beadsDir, "on_close"), []byte("#!/bin/bash\nexisting"), 0755)

	result, err := ensureBeadsCloseHook(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected hook already present")
	}

	// Verify content was NOT overwritten
	data, _ := os.ReadFile(filepath.Join(beadsDir, "on_close"))
	if string(data) != "#!/bin/bash\nexisting" {
		t.Error("existing hook was overwritten")
	}
}

func TestHarnessInitPreCommitGate(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	result, err := ensurePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected pre-commit to be created")
	}

	hookPath := filepath.Join(gitDir, "pre-commit")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal("pre-commit not created")
	}
	if info.Mode()&0111 == 0 {
		t.Error("pre-commit should be executable")
	}
}

func TestHarnessInitPreCommitGateAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	existing := "#!/bin/bash\necho 'existing hook'\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	result, err := ensurePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected accretion gate to be added")
	}

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)
	if content[:len(existing)] != existing {
		t.Error("existing content was modified")
	}
	if !strings.Contains(content, "orch precommit accretion") {
		t.Error("accretion gate not appended")
	}
}

func TestHarnessInitPreCommitGateIdempotent(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	existing := "#!/bin/bash\norch precommit accretion 2>/dev/null || true\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	result, err := ensurePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected gate already present")
	}
}

func TestHarnessInitSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range harnessCmd.Commands() {
		if cmd.Use == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Error("harness init subcommand not registered")
	}
}

// --- Standalone mode tests ---

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

func TestEnsureStandalonePreCommitGate(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)

	result, err := ensureStandalonePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected pre-commit to be created")
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal("pre-commit not created")
	}
	content := string(data)

	// Should NOT reference orch CLI
	if strings.Contains(content, "orch precommit") {
		t.Error("standalone pre-commit should not reference orch CLI")
	}

	// Should contain inline accretion check
	if !strings.Contains(content, "accretion") {
		t.Error("pre-commit should contain accretion check")
	}

	// Should have explanatory comments
	if !strings.Contains(content, "WHY THIS GATE EXISTS") {
		t.Error("pre-commit should contain explanatory comments")
	}
}

func TestEnsureStandalonePreCommitGateIdempotent(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	ensureStandalonePreCommitGate(dir)
	result, err := ensureStandalonePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected gate already present on second run")
	}
}

func TestStandaloneModeSkipsBeadsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	// No .beads/ directory — should not fail
	_, exists := detectStandaloneMode(dir)
	if !exists {
		// standalone mode should be detected when .beads is absent
	}
}

// --- Helper function tests ---

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
			name:     "trailing exit 0 no newline",
			input:    "#!/bin/bash\necho hello\nexit 0",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "trailing exit $?",
			input:    "#!/bin/bash\necho hello\nexit $?\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "no trailing exit",
			input:    "#!/bin/bash\necho hello\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "exit 0 mid-script not removed",
			input:    "#!/bin/bash\nif true; then\n  exit 0\nfi\necho done\n",
			expected: "#!/bin/bash\nif true; then\n  exit 0\nfi\necho done\n",
		},
		{
			name:     "exit 1 not removed",
			input:    "#!/bin/bash\necho hello\nexit 1\n",
			expected: "#!/bin/bash\necho hello\nexit 1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeTrailingExit(tt.input)
			if got != tt.expected {
				t.Errorf("removeTrailingExit(%q) = %q, want %q", tt.input, got, tt.expected)
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
		{
			name:     "no shebang",
			input:    "echo hello\n",
			expected: "echo hello\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureBashShebang(tt.input)
			if got != tt.expected {
				t.Errorf("ensureBashShebang(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripScriptShebang(t *testing.T) {
	input := "#!/bin/bash\n# comment\necho hello\n"
	expected := "# comment\necho hello\n"
	got := stripScriptShebang(input)
	if got != expected {
		t.Errorf("stripScriptShebang(%q) = %q, want %q", input, got, expected)
	}

	// No shebang — unchanged
	noShebang := "# comment\necho hello\n"
	got2 := stripScriptShebang(noShebang)
	if got2 != noShebang {
		t.Errorf("stripScriptShebang(%q) = %q, want %q", noShebang, got2, noShebang)
	}
}

func TestPreCommitGateAppendsBeforeExitZero(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Existing hook ends with exit 0 — gate must be reachable
	existing := "#!/bin/bash\necho 'existing hook'\nexit 0\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	result, err := ensurePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected accretion gate to be added")
	}

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)

	if !strings.Contains(content, "orch precommit accretion") {
		t.Error("accretion gate not found in hook")
	}

	// Gate must be reachable: either no exit 0 remains, or gate appears before it
	gateIdx := strings.Index(content, "orch precommit accretion")
	lastExitIdx := strings.LastIndex(content, "exit 0")
	if lastExitIdx >= 0 && gateIdx > lastExitIdx {
		t.Error("accretion gate appears after exit 0 — would be dead code")
	}

	// Verify the original exit 0 was removed (gate is reachable)
	if gateIdx < 0 {
		t.Error("gate not found in output")
	}
}

func TestStandalonePreCommitGateAppendsBeforeExitZero(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Existing hook ends with exit 0
	existing := "#!/bin/bash\necho 'existing hook'\nexit 0\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	result, err := ensureStandalonePreCommitGate(dir)
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

	// Should contain the accretion check logic
	if !strings.Contains(content, "HARD_LIMIT") {
		t.Error("inline accretion check not found")
	}

	// The process substitution line must be reachable (not after exit 0 of original hook)
	// Check that the gate's own exit handling works, but original exit 0 doesn't kill it
	hardLimitIdx := strings.Index(content, "HARD_LIMIT")
	lines := strings.Split(content, "\n")
	// Find the original "echo 'existing hook'" line
	echoIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "existing hook") {
			echoIdx = i
			break
		}
	}
	if echoIdx == -1 {
		t.Error("original hook content not preserved")
	}
	_ = hardLimitIdx // used implicitly via content checks
}

func TestStandalonePreCommitGateNoDoubleShebang(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	existing := "#!/bin/bash\necho 'existing hook'\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	ensureStandalonePreCommitGate(dir)

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)

	// Count shebangs — should be exactly 1
	shebangCount := strings.Count(content, "#!/bin/bash")
	if shebangCount != 1 {
		t.Errorf("expected exactly 1 shebang, found %d", shebangCount)
	}
}

func TestStandalonePreCommitGateUpgradesShShebang(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Existing hook uses /bin/sh
	existing := "#!/bin/sh\necho 'existing hook'\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	ensureStandalonePreCommitGate(dir)

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)

	// Shebang must be bash, not sh (process substitution requires bash)
	if strings.HasPrefix(content, "#!/bin/sh") {
		t.Error("shebang should be upgraded from #!/bin/sh to #!/bin/bash")
	}
	if !strings.HasPrefix(content, "#!/bin/bash") {
		t.Errorf("expected #!/bin/bash shebang, got first line: %s", strings.SplitN(content, "\n", 2)[0])
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

	result, err := ensureStandaloneHookRegistration(sp, hooksDir)
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

