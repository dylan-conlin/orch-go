package spawn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckOpsecProxy_NotRequired(t *testing.T) {
	err := CheckOpsecProxy(false, 8199)
	if err != nil {
		t.Errorf("expected nil error when opsec not required, got: %v", err)
	}
}

func TestCheckOpsecProxy_ProxyRunning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	var port int
	_, err := fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &port)
	if err != nil {
		t.Fatalf("failed to parse test server port: %v", err)
	}

	err = CheckOpsecProxy(true, port)
	if err != nil {
		t.Errorf("expected nil error when proxy is running, got: %v", err)
	}
}

func TestCheckOpsecProxy_ProxyNotRunning(t *testing.T) {
	err := CheckOpsecProxy(true, 59999)
	if err == nil {
		t.Error("expected error when proxy not running, got nil")
	}
}

func TestOpsecEnvPrefix(t *testing.T) {
	prefix := OpsecEnvPrefix(true, 8199)
	if prefix == "" {
		t.Error("expected non-empty opsec prefix when enabled")
	}

	expected := []string{
		"OPSEC_SANDBOX=1",
		"HTTP_PROXY=http://127.0.0.1:8199",
		"HTTPS_PROXY=http://127.0.0.1:8199",
		"ALL_PROXY=http://127.0.0.1:8199",
	}
	for _, env := range expected {
		if !strings.Contains(prefix, env) {
			t.Errorf("opsec prefix missing %q, got: %s", env, prefix)
		}
	}
}

func TestOpsecEnvPrefix_Disabled(t *testing.T) {
	prefix := OpsecEnvPrefix(false, 8199)
	if prefix != "" {
		t.Errorf("expected empty prefix when disabled, got: %s", prefix)
	}
}

// --- Settings merge/unmerge tests ---

func TestMergeOpsecIntoSettings_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := MergeOpsecIntoSettings(path); err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON after merge: %v", err)
	}

	// Should have hooks.PreToolUse with sandbox entry
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("missing hooks after merge")
	}
	preTool, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		t.Fatal("missing hooks.PreToolUse after merge")
	}

	found := false
	for _, entry := range preTool {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		// Check for the OPSEC hook — uses nested hooks format
		innerHooks, ok := m["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, h := range innerHooks {
			hm, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "sandbox-bash.sh") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("sandbox-bash.sh hook not found in PreToolUse after merge: %s", string(data))
	}

	// Should have deny rules
	perms, ok := settings["permissions"].(map[string]interface{})
	if !ok {
		t.Fatal("missing permissions after merge")
	}
	deny, ok := perms["deny"].([]interface{})
	if !ok {
		t.Fatal("missing permissions.deny after merge")
	}

	hasOshcut := false
	for _, rule := range deny {
		s, ok := rule.(string)
		if !ok {
			continue
		}
		if strings.Contains(s, "oshcut") {
			hasOshcut = true
			break
		}
	}
	if !hasOshcut {
		t.Errorf("OshCut deny rules not found after merge: %v", deny)
	}
}

func TestMergeOpsecIntoSettings_PreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	existing := `{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [{"command": "~/.orch/hooks/gate-bd-close.py", "timeout": 10000, "type": "command"}],
        "matcher": "Bash"
      }
    ]
  },
  "permissions": {
    "deny": ["Edit(~/.claude/settings.json)"]
  },
  "skipDangerousModePermissionPrompt": true
}`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := MergeOpsecIntoSettings(path); err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	// Verify existing hook preserved
	hooks := settings["hooks"].(map[string]interface{})
	preTool := hooks["PreToolUse"].([]interface{})

	hasGate := false
	hasSandbox := false
	for _, entry := range preTool {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		innerHooks, ok := m["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, h := range innerHooks {
			hm, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			cmd, _ := hm["command"].(string)
			if strings.Contains(cmd, "gate-bd-close.py") {
				hasGate = true
			}
			if strings.Contains(cmd, "sandbox-bash.sh") {
				hasSandbox = true
			}
		}
	}
	if !hasGate {
		t.Error("existing gate-bd-close.py hook was lost during merge")
	}
	if !hasSandbox {
		t.Error("sandbox-bash.sh hook not added during merge")
	}

	// Verify existing deny rule preserved
	perms := settings["permissions"].(map[string]interface{})
	deny := perms["deny"].([]interface{})
	hasSettingsDeny := false
	for _, rule := range deny {
		if s, ok := rule.(string); ok && s == "Edit(~/.claude/settings.json)" {
			hasSettingsDeny = true
		}
	}
	if !hasSettingsDeny {
		t.Error("existing deny rule Edit(~/.claude/settings.json) was lost during merge")
	}

	// Verify other fields preserved
	if _, ok := settings["skipDangerousModePermissionPrompt"]; !ok {
		t.Error("skipDangerousModePermissionPrompt field was lost during merge")
	}
}

func TestMergeOpsecIntoSettings_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte("{}"), 0644)

	// Merge twice
	MergeOpsecIntoSettings(path)
	MergeOpsecIntoSettings(path)

	data, _ := os.ReadFile(path)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	// Count sandbox hooks — should be exactly 1
	hooks := settings["hooks"].(map[string]interface{})
	preTool := hooks["PreToolUse"].([]interface{})
	count := 0
	for _, entry := range preTool {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		innerHooks, ok := m["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, h := range innerHooks {
			hm, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "sandbox-bash.sh") {
				count++
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 sandbox hook after double merge, got %d", count)
	}

	// Count oshcut deny rules — should have no duplicates
	perms := settings["permissions"].(map[string]interface{})
	deny := perms["deny"].([]interface{})
	oshcutCount := 0
	for _, rule := range deny {
		if s, ok := rule.(string); ok && s == "WebFetch(domain:app.oshcut.com)" {
			oshcutCount++
		}
	}
	if oshcutCount != 1 {
		t.Errorf("expected exactly 1 WebFetch oshcut deny rule after double merge, got %d", oshcutCount)
	}
}

func TestUnmergeOpsecFromSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte("{}"), 0644)

	// Merge then unmerge
	MergeOpsecIntoSettings(path)
	if err := UnmergeOpsecFromSettings(path); err != nil {
		t.Fatalf("unmerge failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	// Hooks should not contain sandbox-bash.sh
	if hooks, ok := settings["hooks"].(map[string]interface{}); ok {
		if preTool, ok := hooks["PreToolUse"].([]interface{}); ok {
			for _, entry := range preTool {
				m, ok := entry.(map[string]interface{})
				if !ok {
					continue
				}
				innerHooks, ok := m["hooks"].([]interface{})
				if !ok {
					continue
				}
				for _, h := range innerHooks {
					hm, ok := h.(map[string]interface{})
					if !ok {
						continue
					}
					if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "sandbox-bash.sh") {
						t.Error("sandbox-bash.sh hook still present after unmerge")
					}
				}
			}
		}
	}

	// Deny rules should not contain opsec domains
	if perms, ok := settings["permissions"].(map[string]interface{}); ok {
		if deny, ok := perms["deny"].([]interface{}); ok {
			for _, rule := range deny {
				if s, ok := rule.(string); ok && strings.Contains(s, "oshcut") {
					t.Errorf("oshcut deny rule still present after unmerge: %s", s)
				}
			}
		}
	}
}

func TestUnmergeOpsecFromSettings_PreservesNonOpsec(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	existing := `{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [{"command": "~/.orch/hooks/gate-bd-close.py", "timeout": 10000, "type": "command"}],
        "matcher": "Bash"
      }
    ]
  },
  "permissions": {
    "deny": ["Edit(~/.claude/settings.json)"]
  },
  "skipDangerousModePermissionPrompt": true
}`
	os.WriteFile(path, []byte(existing), 0644)

	MergeOpsecIntoSettings(path)
	UnmergeOpsecFromSettings(path)

	data, _ := os.ReadFile(path)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	// Existing hook should survive
	hooks := settings["hooks"].(map[string]interface{})
	preTool := hooks["PreToolUse"].([]interface{})
	hasGate := false
	for _, entry := range preTool {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		innerHooks, ok := m["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, h := range innerHooks {
			hm, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if cmd, _ := hm["command"].(string); strings.Contains(cmd, "gate-bd-close.py") {
				hasGate = true
			}
		}
	}
	if !hasGate {
		t.Error("existing gate-bd-close.py hook was removed during unmerge")
	}

	// Existing deny rule should survive
	perms := settings["permissions"].(map[string]interface{})
	deny := perms["deny"].([]interface{})
	hasSettingsDeny := false
	for _, rule := range deny {
		if s, ok := rule.(string); ok && s == "Edit(~/.claude/settings.json)" {
			hasSettingsDeny = true
		}
	}
	if !hasSettingsDeny {
		t.Error("existing deny rule was removed during unmerge")
	}

	// Other fields preserved
	if _, ok := settings["skipDangerousModePermissionPrompt"]; !ok {
		t.Error("skipDangerousModePermissionPrompt lost during unmerge")
	}
}

func TestOpsecDenyRules(t *testing.T) {
	rules := OpsecDenyRules()
	if len(rules) == 0 {
		t.Fatal("expected non-empty deny rules")
	}

	// Must have OshCut rules (highest priority — lawsuit risk)
	hasOshcut := false
	for _, rule := range rules {
		if strings.Contains(rule, "oshcut") {
			hasOshcut = true
			break
		}
	}
	if !hasOshcut {
		t.Error("OshCut deny rules missing")
	}

	// Must have both WebFetch and WebSearch rules
	hasWebFetch := false
	hasWebSearch := false
	for _, rule := range rules {
		if strings.HasPrefix(rule, "WebFetch(") {
			hasWebFetch = true
		}
		if strings.HasPrefix(rule, "WebSearch(") {
			hasWebSearch = true
		}
	}
	if !hasWebFetch {
		t.Error("WebFetch deny rules missing")
	}
	if !hasWebSearch {
		t.Error("WebSearch deny rules missing")
	}

	// Must have OPSEC config self-protection rules
	hasOpsecProtect := false
	for _, rule := range rules {
		if strings.Contains(rule, "~/.orch/opsec/") {
			hasOpsecProtect = true
			break
		}
	}
	if !hasOpsecProtect {
		t.Error("OPSEC config self-protection deny rules missing")
	}
}

func TestGenerateLaunchAgentPlist(t *testing.T) {
	plist := GenerateLaunchAgentPlist("/Users/testuser/.orch/opsec/tinyproxy.conf")
	if !strings.Contains(plist, "com.orch.opsec-proxy") {
		t.Error("plist missing label")
	}
	if !strings.Contains(plist, "tinyproxy") {
		t.Error("plist missing tinyproxy program")
	}
	if !strings.Contains(plist, "RunAtLoad") {
		t.Error("plist missing RunAtLoad")
	}
	if !strings.Contains(plist, "KeepAlive") {
		t.Error("plist missing KeepAlive")
	}
	if !strings.Contains(plist, "tinyproxy.conf") {
		t.Error("plist missing config path")
	}
}

func TestIsOpsecInstalled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	// Empty settings — not installed
	os.WriteFile(path, []byte("{}"), 0644)
	if IsOpsecInstalled(path) {
		t.Error("expected not installed for empty settings")
	}

	// After merge — installed
	MergeOpsecIntoSettings(path)
	if !IsOpsecInstalled(path) {
		t.Error("expected installed after merge")
	}

	// After unmerge — not installed
	UnmergeOpsecFromSettings(path)
	if IsOpsecInstalled(path) {
		t.Error("expected not installed after unmerge")
	}
}
