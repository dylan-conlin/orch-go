// Package main provides the opsec command for managing OPSEC proxy infrastructure.
package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

var opsecCmd = &cobra.Command{
	Use:   "opsec",
	Short: "Manage OPSEC proxy infrastructure",
	Long: `Manage the local OPSEC proxy and sandbox enforcement that prevents all Claude
sessions from reaching competitor domains. The proxy runs on localhost:8199
and enforces a domain blocklist via tinyproxy.

Commands:
  install   - Install OPSEC as environmental enforcement (global settings + LaunchAgent)
  uninstall - Remove OPSEC from global settings and unload LaunchAgent
  start     - Start tinyproxy on localhost:8199
  stop      - Stop tinyproxy
  status    - Check proxy health, enforcement scope, and config
  test      - Run sandbox-exec + proxy verification tests`,
}

var opsecStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the OPSEC proxy (tinyproxy)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOpsecStart()
	},
}

var opsecStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the OPSEC proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOpsecStop()
	},
}

var opsecStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check OPSEC proxy health and config",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOpsecStatus()
	},
}

var opsecTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Run OPSEC verification tests (sandbox-exec + proxy)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOpsecTest()
	},
}

var opsecInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install OPSEC as environmental enforcement (all Claude sessions)",
	Long: `Installs OPSEC enforcement at the harness level:
1. Merges sandbox-bash.sh hook into ~/.claude/settings.json (global)
2. Merges WebFetch/WebSearch competitor deny rules into global settings
3. Creates and loads a LaunchAgent plist for auto-starting tinyproxy
4. Starts the proxy immediately

After install, ALL Claude sessions on this machine are protected —
not just spawned agents.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOpsecInstall()
	},
}

var opsecUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove OPSEC from global settings and unload LaunchAgent",
	Long: `Removes OPSEC enforcement from the harness level:
1. Removes sandbox-bash.sh hook from ~/.claude/settings.json
2. Removes WebFetch/WebSearch competitor deny rules from global settings
3. Unloads and removes the LaunchAgent plist

After uninstall, OPSEC reverts to spawn-only enforcement
(active only when OPSEC_SANDBOX=1 is set by orch spawn).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOpsecUninstall()
	},
}

func init() {
	opsecCmd.AddCommand(opsecInstallCmd)
	opsecCmd.AddCommand(opsecUninstallCmd)
	opsecCmd.AddCommand(opsecStartCmd)
	opsecCmd.AddCommand(opsecStopCmd)
	opsecCmd.AddCommand(opsecStatusCmd)
	opsecCmd.AddCommand(opsecTestCmd)
	rootCmd.AddCommand(opsecCmd)
}

func opsecConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "opsec")
}

func opsecPidFile() string {
	return filepath.Join(opsecConfigDir(), "tinyproxy.pid")
}

func globalSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func launchAgentDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents")
}

func launchAgentPlistPath() string {
	return filepath.Join(launchAgentDir(), spawn.OpsecLaunchAgentLabel+".plist")
}

func runOpsecInstall() error {
	configDir := opsecConfigDir()
	confPath := filepath.Join(configDir, "tinyproxy.conf")
	settingsPath := globalSettingsPath()

	// Verify prerequisites
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		return fmt.Errorf("tinyproxy.conf not found at %s — OPSEC config files must exist first", confPath)
	}
	if _, err := os.Stat(filepath.Join(configDir, "sandbox-bash.sh")); os.IsNotExist(err) {
		return fmt.Errorf("sandbox-bash.sh not found at %s/sandbox-bash.sh", configDir)
	}
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return fmt.Errorf("global settings not found at %s", settingsPath)
	}

	// Step 1: Merge OPSEC into global settings
	fmt.Print("Merging OPSEC into global settings... ")
	if err := spawn.MergeOpsecIntoSettings(settingsPath); err != nil {
		return fmt.Errorf("failed to merge settings: %w", err)
	}
	fmt.Println("OK")

	// Step 2: Create and load LaunchAgent
	fmt.Print("Installing LaunchAgent... ")
	plistContent := spawn.GenerateLaunchAgentPlist(confPath)
	plistPath := launchAgentPlistPath()

	if err := os.MkdirAll(launchAgentDir(), 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents dir: %w", err)
	}
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	// Unload first in case it's already loaded (ignore errors)
	exec.Command("launchctl", "unload", plistPath).Run()

	if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
		fmt.Printf("WARNING: launchctl load failed: %v\n", err)
		fmt.Println("  You may need to load manually: launchctl load " + plistPath)
	} else {
		fmt.Println("OK")
	}

	// Step 3: Verify proxy is running (LaunchAgent should have started it)
	time.Sleep(1 * time.Second)
	if err := spawn.CheckOpsecProxy(true, spawn.DefaultOpsecPort); err != nil {
		fmt.Println("Proxy not yet running — starting manually...")
		if startErr := runOpsecStart(); startErr != nil {
			return fmt.Errorf("proxy start failed: %w", startErr)
		}
	} else {
		fmt.Printf("Proxy running on localhost:%d\n", spawn.DefaultOpsecPort)
	}

	fmt.Println()
	fmt.Println("OPSEC installed as environmental enforcement.")
	fmt.Println("All Claude sessions on this machine are now protected.")
	return nil
}

func runOpsecUninstall() error {
	settingsPath := globalSettingsPath()

	// Step 1: Remove OPSEC from global settings
	fmt.Print("Removing OPSEC from global settings... ")
	if err := spawn.UnmergeOpsecFromSettings(settingsPath); err != nil {
		fmt.Printf("WARNING: %v\n", err)
	} else {
		fmt.Println("OK")
	}

	// Step 2: Unload and remove LaunchAgent
	plistPath := launchAgentPlistPath()
	fmt.Print("Unloading LaunchAgent... ")
	if _, err := os.Stat(plistPath); err == nil {
		exec.Command("launchctl", "unload", plistPath).Run()
		os.Remove(plistPath)
		fmt.Println("OK")
	} else {
		fmt.Println("not installed")
	}

	// Step 3: Stop proxy
	fmt.Print("Stopping proxy... ")
	if err := spawn.CheckOpsecProxy(true, spawn.DefaultOpsecPort); err == nil {
		runOpsecStop()
	} else {
		fmt.Println("not running")
	}

	fmt.Println()
	fmt.Println("OPSEC uninstalled from environmental enforcement.")
	fmt.Println("Spawn-only enforcement remains available via orch spawn --opsec.")
	return nil
}

func runOpsecStart() error {
	configDir := opsecConfigDir()
	confPath := filepath.Join(configDir, "tinyproxy.conf")

	// Check config exists
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		return fmt.Errorf("tinyproxy.conf not found at %s — run setup first", confPath)
	}

	// Check if already running
	if err := spawn.CheckOpsecProxy(true, spawn.DefaultOpsecPort); err == nil {
		fmt.Println("OPSEC proxy already running on :8199")
		return nil
	}

	// Start tinyproxy
	cmd := exec.Command("tinyproxy", "-c", confPath, "-d")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tinyproxy: %w", err)
	}

	// Write PID file
	pid := cmd.Process.Pid
	if err := os.WriteFile(opsecPidFile(), []byte(strconv.Itoa(pid)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write PID file: %v\n", err)
	}

	// Detach — don't wait for it
	if err := cmd.Process.Release(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to release process: %v\n", err)
	}

	// Wait briefly and verify
	time.Sleep(500 * time.Millisecond)
	if err := spawn.CheckOpsecProxy(true, spawn.DefaultOpsecPort); err != nil {
		return fmt.Errorf("tinyproxy started (pid %d) but health check failed: %w", pid, err)
	}

	fmt.Printf("OPSEC proxy started (pid %d) on localhost:%d\n", pid, spawn.DefaultOpsecPort)
	return nil
}

func runOpsecStop() error {
	// Try PID file first
	pidData, err := os.ReadFile(opsecPidFile())
	if err == nil {
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if err == nil {
			proc, err := os.FindProcess(pid)
			if err == nil {
				if err := proc.Signal(os.Interrupt); err == nil {
					_ = os.Remove(opsecPidFile())
					fmt.Printf("OPSEC proxy stopped (pid %d)\n", pid)
					return nil
				}
			}
		}
	}

	// Fallback: kill by name
	cmd := exec.Command("pkill", "-f", "tinyproxy.*opsec")
	if err := cmd.Run(); err != nil {
		// Check if proxy is actually running
		if checkErr := spawn.CheckOpsecProxy(true, spawn.DefaultOpsecPort); checkErr != nil {
			fmt.Println("OPSEC proxy is not running")
			return nil
		}
		return fmt.Errorf("failed to stop tinyproxy: %w", err)
	}

	_ = os.Remove(opsecPidFile())
	fmt.Println("OPSEC proxy stopped")
	return nil
}

func runOpsecStatus() error {
	configDir := opsecConfigDir()
	port := spawn.DefaultOpsecPort
	settingsPath := globalSettingsPath()

	fmt.Println("OPSEC Status")
	fmt.Println(strings.Repeat("-", 50))

	// Enforcement scope
	isGlobal := spawn.IsOpsecInstalled(settingsPath)
	if isGlobal {
		fmt.Println("Scope:    ENVIRONMENTAL (all Claude sessions)")
	} else {
		fmt.Println("Scope:    SPAWN-ONLY (only orch spawn --opsec)")
	}

	// Check proxy health
	if err := spawn.CheckOpsecProxy(true, port); err != nil {
		fmt.Printf("Proxy:    NOT RUNNING (port %d)\n", port)
	} else {
		fmt.Printf("Proxy:    RUNNING on localhost:%d\n", port)
	}

	// Check LaunchAgent
	plistPath := launchAgentPlistPath()
	if _, err := os.Stat(plistPath); err == nil {
		fmt.Println("AutoStart: LaunchAgent installed")
	} else {
		fmt.Println("AutoStart: not configured (manual start required)")
	}

	// Check config files
	files := []struct {
		name string
		path string
	}{
		{"Config", filepath.Join(configDir, "tinyproxy.conf")},
		{"Blocklist", filepath.Join(configDir, "blocked-domains.txt")},
		{"Sandbox hook", filepath.Join(configDir, "sandbox-bash.sh")},
		{"Worker settings", filepath.Join(configDir, "worker-settings.json")},
	}

	fmt.Println()
	for _, f := range files {
		if _, err := os.Stat(f.path); os.IsNotExist(err) {
			fmt.Printf("%-16s MISSING  %s\n", f.name+":", f.path)
		} else {
			fmt.Printf("%-16s OK       %s\n", f.name+":", f.path)
		}
	}

	// Show blocked domains
	blocklist := filepath.Join(configDir, "blocked-domains.txt")
	if data, err := os.ReadFile(blocklist); err == nil {
		fmt.Println()
		fmt.Println("Blocked domains:")
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	// Check sandbox-exec availability
	fmt.Println()
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		fmt.Println("sandbox-exec:  NOT FOUND")
	} else {
		fmt.Println("sandbox-exec:  available")
	}

	if !isGlobal {
		fmt.Println()
		fmt.Println("Run 'orch opsec install' to enable environmental enforcement.")
	}

	return nil
}

func runOpsecTest() error {
	port := spawn.DefaultOpsecPort
	passed := 0
	failed := 0

	testCase := func(name string, fn func() error) {
		fmt.Printf("Test: %s... ", name)
		if err := fn(); err != nil {
			fmt.Printf("FAIL (%v)\n", err)
			failed++
		} else {
			fmt.Println("PASS")
			passed++
		}
	}

	// Test 1: sandbox-exec blocks direct outbound
	testCase("sandbox-exec blocks direct curl to oshcut.com", func() error {
		profile := `(version 1)(allow default)(deny network-outbound)(allow network-outbound (remote ip "localhost:*"))`
		cmd := exec.Command("sandbox-exec", "-p", profile, "curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "--connect-timeout", "3", "https://app.oshcut.com")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil // Expected: should fail
		}
		// If it somehow succeeded, that's a failure
		if strings.TrimSpace(string(output)) == "200" {
			return fmt.Errorf("curl succeeded (got 200), should have been blocked")
		}
		return nil
	})

	// Test 2: sandbox-exec + proxy returns 403 for blocked domains
	testCase("proxy returns 403 for oshcut.com", func() error {
		if err := spawn.CheckOpsecProxy(true, port); err != nil {
			return fmt.Errorf("proxy not running, skipping: %w", err)
		}
		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(mustParseURL(fmt.Sprintf("http://127.0.0.1:%d", port))),
			},
			Timeout: 5 * time.Second,
		}
		resp, err := client.Get("http://app.oshcut.com")
		if err != nil {
			// Connection refused or similar is also acceptable (domain blocked)
			return nil
		}
		defer resp.Body.Close()
		if resp.StatusCode == 403 {
			return nil
		}
		return fmt.Errorf("expected 403, got %d", resp.StatusCode)
	})

	// Test 3: proxy allows non-blocked domains
	testCase("proxy allows httpbin.org", func() error {
		if err := spawn.CheckOpsecProxy(true, port); err != nil {
			return fmt.Errorf("proxy not running, skipping: %w", err)
		}
		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(mustParseURL(fmt.Sprintf("http://127.0.0.1:%d", port))),
			},
			Timeout: 10 * time.Second,
		}
		resp, err := client.Get("http://httpbin.org/ip")
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			return nil
		}
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	})

	// Test 4: sandbox-exec allows localhost
	testCase("sandbox-exec allows localhost connections", func() error {
		profile := `(version 1)(allow default)(deny network-outbound)(allow network-outbound (remote ip "localhost:*"))`
		cmd := exec.Command("sandbox-exec", "-p", profile, "curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "--connect-timeout", "3", fmt.Sprintf("http://127.0.0.1:%d/", port))
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Non-zero exit is OK if proxy responded (e.g., 400)
			outputStr := strings.TrimSpace(string(output))
			if outputStr == "400" || outputStr == "403" {
				return nil
			}
			return fmt.Errorf("failed: %v (output: %s)", err, outputStr)
		}
		return nil
	})

	// Test 5: git works (would be allowlisted by hook)
	testCase("git commands work normally", func() error {
		cmd := exec.Command("git", "status")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git status failed: %w", err)
		}
		return nil
	})

	fmt.Printf("\nResults: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		return fmt.Errorf("%d test(s) failed", failed)
	}
	return nil
}

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
