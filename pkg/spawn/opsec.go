package spawn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// CheckOpsecProxy verifies the local OPSEC proxy is running and responsive.
// Returns nil if opsec is not required (enabled=false) or if proxy is healthy.
// Returns error if proxy is required but not running — this is a hard-blocking check.
func CheckOpsecProxy(enabled bool, port int) error {
	if !enabled {
		return nil
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		return fmt.Errorf("opsec-proxy not running on :%d — run 'orch opsec start': %w", port, err)
	}
	defer resp.Body.Close()

	// tinyproxy returns various status codes for non-proxy requests (400, 403, etc.)
	// Any response means the proxy is alive.
	return nil
}

// OpsecEnvPrefix returns the shell export prefix for OPSEC environment variables.
// When enabled, exports OPSEC_SANDBOX=1 (informational flag) and proxy env vars
// (HTTP_PROXY, HTTPS_PROXY, ALL_PROXY) as defense-in-depth.
//
// Note: OPSEC_SANDBOX=1 is no longer checked by sandbox-bash.sh (the hook is
// always active when installed globally). It remains as an informational signal
// for logging and other tools that may check it.
//
// When OPSEC is globally installed (via orch opsec install), the proxy env vars
// are defense-in-depth: they catch curl/Python/Ruby even if sandbox-exec were
// to fail or be removed by Apple.
func OpsecEnvPrefix(enabled bool, port int) string {
	if !enabled {
		return ""
	}
	proxy := fmt.Sprintf("http://127.0.0.1:%d", port)
	return fmt.Sprintf(
		"export OPSEC_SANDBOX=1; export HTTP_PROXY=%s; export HTTPS_PROXY=%s; export ALL_PROXY=%s; ",
		proxy, proxy, proxy,
	)
}

// DefaultOpsecPort is the default port for the local OPSEC proxy.
const DefaultOpsecPort = 8199

// OpsecSettingsPath returns the path to the OPSEC worker settings.json.
const OpsecSettingsPath = "~/.orch/opsec/worker-settings.json"

// OpsecHookCommand is the command path for the sandbox-bash.sh hook.
const OpsecHookCommand = "~/.orch/opsec/sandbox-bash.sh"

// OpsecLaunchAgentLabel is the launchd label for the OPSEC proxy.
const OpsecLaunchAgentLabel = "com.orch.opsec-proxy"

// OpsecDenyRules returns the permission deny rules for OPSEC enforcement.
// These block WebFetch/WebSearch to competitor domains and protect OPSEC config files.
func OpsecDenyRules() []string {
	return []string{
		// OshCut (IP ban risk — next detection = lawsuit)
		"WebFetch(domain:app.oshcut.com)",
		"WebFetch(domain:oshcut.com)",
		"WebFetch(domain:www.oshcut.com)",
		// FabWorks
		"WebFetch(domain:fabworks.com)",
		"WebFetch(domain:www.fabworks.com)",
		// Xometry
		"WebFetch(domain:xometry.com)",
		"WebFetch(domain:www.xometry.com)",
		// Protolabs
		"WebFetch(domain:protolabs.com)",
		"WebFetch(domain:www.protolabs.com)",
		// WebSearch blocks
		"WebSearch(*oshcut*)",
		"WebSearch(*fabworks*)",
		"WebSearch(*xometry*)",
		"WebSearch(*protolabs*)",
		// Self-protection: prevent agents from modifying OPSEC config
		"Edit(~/.orch/opsec/*)",
		"Write(~/.orch/opsec/*)",
	}
}

// opsecHookEntry returns the PreToolUse hook entry for the sandbox-bash.sh hook.
// Uses the nested hooks format consistent with other hooks in settings.json.
func opsecHookEntry() map[string]interface{} {
	return map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"command": OpsecHookCommand,
				"timeout": float64(5000),
				"type":    "command",
			},
		},
		"matcher": "Bash",
	}
}

// isOpsecHookEntry checks if a PreToolUse entry is the OPSEC sandbox hook.
func isOpsecHookEntry(entry map[string]interface{}) bool {
	innerHooks, ok := entry["hooks"].([]interface{})
	if !ok {
		return false
	}
	for _, h := range innerHooks {
		hm, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "sandbox-bash.sh") {
			return true
		}
	}
	return false
}

// isOpsecDenyRule checks if a deny rule was added by OPSEC.
func isOpsecDenyRule(rule string) bool {
	opsecRules := OpsecDenyRules()
	for _, r := range opsecRules {
		if rule == r {
			return true
		}
	}
	return false
}

// MergeOpsecIntoSettings merges OPSEC hooks and deny rules into a Claude settings.json file.
// This is an additive merge — existing entries are preserved. The merge is idempotent:
// running it twice produces the same result as running it once.
func MergeOpsecIntoSettings(settingsPath string) error {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("reading settings: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parsing settings: %w", err)
	}

	// Ensure hooks.PreToolUse exists
	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	preTool, _ := hooks["PreToolUse"].([]interface{})

	// Check if OPSEC hook already present (idempotency)
	hasOpsecHook := false
	for _, entry := range preTool {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if isOpsecHookEntry(m) {
			hasOpsecHook = true
			break
		}
	}
	if !hasOpsecHook {
		preTool = append(preTool, opsecHookEntry())
		hooks["PreToolUse"] = preTool
	}

	// Ensure permissions.deny exists
	perms, _ := settings["permissions"].(map[string]interface{})
	if perms == nil {
		perms = make(map[string]interface{})
		settings["permissions"] = perms
	}

	deny, _ := perms["deny"].([]interface{})

	// Build set of existing deny rules for dedup
	existingRules := make(map[string]bool)
	for _, rule := range deny {
		if s, ok := rule.(string); ok {
			existingRules[s] = true
		}
	}

	// Add OPSEC deny rules (skip duplicates)
	for _, rule := range OpsecDenyRules() {
		if !existingRules[rule] {
			deny = append(deny, rule)
		}
	}
	perms["deny"] = deny

	// Write back with indentation
	output, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	return os.WriteFile(settingsPath, append(output, '\n'), 0644)
}

// UnmergeOpsecFromSettings removes OPSEC hooks and deny rules from a Claude settings.json file.
// Non-OPSEC entries are preserved.
func UnmergeOpsecFromSettings(settingsPath string) error {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("reading settings: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parsing settings: %w", err)
	}

	// Remove OPSEC hook from PreToolUse
	if hooks, ok := settings["hooks"].(map[string]interface{}); ok {
		if preTool, ok := hooks["PreToolUse"].([]interface{}); ok {
			var filtered []interface{}
			for _, entry := range preTool {
				m, ok := entry.(map[string]interface{})
				if !ok {
					filtered = append(filtered, entry)
					continue
				}
				if !isOpsecHookEntry(m) {
					filtered = append(filtered, entry)
				}
			}
			hooks["PreToolUse"] = filtered
		}
	}

	// Remove OPSEC deny rules
	if perms, ok := settings["permissions"].(map[string]interface{}); ok {
		if deny, ok := perms["deny"].([]interface{}); ok {
			var filtered []interface{}
			for _, rule := range deny {
				s, ok := rule.(string)
				if !ok {
					filtered = append(filtered, rule)
					continue
				}
				if !isOpsecDenyRule(s) {
					filtered = append(filtered, rule)
				}
			}
			perms["deny"] = filtered
		}
	}

	output, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	return os.WriteFile(settingsPath, append(output, '\n'), 0644)
}

// IsOpsecInstalled checks if OPSEC is installed in the global settings.
// Returns true if the sandbox-bash.sh hook is present in PreToolUse.
func IsOpsecInstalled(settingsPath string) bool {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return false
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return false
	}

	preTool, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		return false
	}

	for _, entry := range preTool {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if isOpsecHookEntry(m) {
			return true
		}
	}
	return false
}

// GenerateLaunchAgentPlist generates a macOS LaunchAgent plist for tinyproxy.
// The plist starts tinyproxy on login and restarts it if it crashes.
func GenerateLaunchAgentPlist(confPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>/opt/homebrew/bin/tinyproxy</string>
        <string>-d</string>
        <string>-c</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/tmp/opsec-proxy.err</string>
</dict>
</plist>
`, OpsecLaunchAgentLabel, confPath)
}
