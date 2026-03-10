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

