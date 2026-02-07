package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/spf13/cobra"
)

var doctorInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the doctor daemon as a launchd service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctorInstall()
	},
}

var doctorUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the doctor daemon launchd service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctorUninstall()
	},
}

func startOpenCode() error {
	// Pre-flight check: Is something already listening on the port?
	addr := fmt.Sprintf("localhost:%d", 4096)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err == nil {
		conn.Close()
		// Something is listening, but the API check failed earlier
		// Try to find and kill zombie opencode processes
		if err := killZombieOpenCodeProcesses(); err != nil {
			return fmt.Errorf("port 4096 in use but API not responding, and failed to clean up: %w", err)
		}
		// Wait for port to be released
		time.Sleep(2 * time.Second)
	}

	// Find opencode binary - prefer the known location from Procfile
	homeDir, _ := os.UserHomeDir()
	opencodePath := filepath.Join(homeDir, ".bun", "bin", "opencode")
	if _, err := os.Stat(opencodePath); os.IsNotExist(err) {
		// Fallback to PATH
		opencodePath, err = exec.LookPath("opencode")
		if err != nil {
			return fmt.Errorf("opencode binary not found at ~/.bun/bin/opencode or in PATH")
		}
	}

	// Create a log file for startup diagnostics
	logDir := filepath.Join(homeDir, ".local", "share", "opencode")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Continue anyway, just won't have logs
	}
	startupLogPath := filepath.Join(logDir, "startup.log")

	// Start OpenCode server in background, fully detached via shell
	// This ensures the process survives even if the parent is killed
	// - ORCH_WORKER=1: so spawned agents know they are orch-managed workers
	// - env -u ANTHROPIC_API_KEY: use OAuth stealth mode (matches Procfile)
	// - Capture stdout/stderr to startup.log for debugging
	cmdStr := fmt.Sprintf(
		"ORCH_WORKER=1 env -u ANTHROPIC_API_KEY %s serve --port 4096 >> %s 2>&1 &",
		opencodePath, startupLogPath,
	)
	cmd := exec.Command("sh", "-c", cmdStr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start OpenCode: %w", err)
	}

	// Wait for it to be ready (poll for up to 15 seconds)
	client := opencode.NewClient(serverURL) // entry-point: startOpenCode is infrastructure startup
	var lastErr error
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := client.ListSessions("")
		if err == nil {
			return nil
		}
		lastErr = err
	}

	// Failed to start - read the startup log for diagnostics
	logContent, _ := os.ReadFile(startupLogPath)
	if len(logContent) > 0 {
		// Truncate to last 500 bytes
		if len(logContent) > 500 {
			logContent = logContent[len(logContent)-500:]
		}
		return fmt.Errorf("OpenCode not responding after 15s (last error: %v)\nStartup log tail:\n%s", lastErr, string(logContent))
	}
	return fmt.Errorf("OpenCode not responding after 15s (last error: %v, no startup log)", lastErr)
}

// killZombieOpenCodeProcesses finds and kills unresponsive opencode serve processes.
func killZombieOpenCodeProcesses() error {
	// Find opencode serve processes
	cmd := exec.Command("pgrep", "-f", "opencode serve.*4096")
	output, err := cmd.Output()
	if err != nil {
		// No processes found, which is fine
		return nil
	}

	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, pidStr := range pids {
		if pidStr == "" {
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		// Send SIGTERM first
		proc, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			continue
		}
		if doctorVerbose {
			fmt.Printf("  Sent SIGTERM to zombie opencode process (PID %d)\n", pid)
		}
	}
	return nil
}

// startOrchServe starts the orch serve API server in the background.
func startOrchServe() error {
	// Find the orch binary path
	orchPath, err := exec.LookPath("orch")
	if err != nil {
		// Try with full path from home directory
		homeDir, _ := os.UserHomeDir()
		orchPath = homeDir + "/bin/orch"
		if _, err := os.Stat(orchPath); os.IsNotExist(err) {
			return fmt.Errorf("orch binary not found in PATH or ~/bin/orch")
		}
	}

	// Start orch serve in background
	cmd := exec.Command("sh", "-c", fmt.Sprintf("nohup %s serve --port %d </dev/null >/dev/null 2>&1 &", orchPath, DefaultServePort))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start orch serve: %w", err)
	}

	// Wait for it to be ready (poll for up to 5 seconds)
	// First check TCP, then HTTPS health endpoint
	addr := fmt.Sprintf("localhost:%d", DefaultServePort)

	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)

		// Quick TCP check first
		conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
		if err != nil {
			continue
		}
		conn.Close()

		// TCP succeeded, now verify HTTPS health
		healthURL := fmt.Sprintf("https://localhost:%d/health", DefaultServePort)
		httpClient := &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec // Self-signed localhost cert
				},
			},
		}

		resp, err := httpClient.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}

	return fmt.Errorf("orch serve started but not responding after 5s")
}

func getDoctorPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.orch.doctor.plist")
}

// runDoctorInstall generates and installs the doctor daemon plist.
func runDoctorInstall() error {
	plistPath := getDoctorPlistPath()
	fmt.Println("Installing orch doctor daemon...")
	fmt.Printf("  Plist path: %s\n", plistPath)
	fmt.Println()

	orchPath, err := exec.LookPath("orch")
	if err != nil {
		home, _ := os.UserHomeDir()
		orchPath = filepath.Join(home, "bin", "orch")
		if _, err := os.Stat(orchPath); os.IsNotExist(err) {
			return fmt.Errorf("orch binary not found in PATH or ~/bin/orch")
		}
	}

	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".orch", "doctor.log")
	errLogPath := filepath.Join(home, ".orch", "doctor-error.log")

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.orch.doctor</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>doctor</string>
        <string>--daemon</string>
        <string>--verbose</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>%s/bin:%s/.bun/bin:%s/go/bin:/opt/homebrew/bin:%s/.local/bin:/usr/local/bin:/usr/bin:/bin</string>
    </dict>
</dict>
</plist>
`, orchPath, logPath, errLogPath, home, home, home, home)

	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}
	fmt.Println("  ✓ Plist created")

	uidCmd := exec.Command("id", "-u")
	uidOutput, err := uidCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}
	uid := strings.TrimSpace(string(uidOutput))

	loadCmd := exec.Command("launchctl", "load", plistPath)
	if err := loadCmd.Run(); err != nil {
		exec.Command("launchctl", "unload", plistPath).Run()
		if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
			return fmt.Errorf("failed to load plist: %w", err)
		}
	}
	fmt.Println("  ✓ Daemon loaded")

	if err := exec.Command("launchctl", "kickstart", "-k", fmt.Sprintf("gui/%s/com.orch.doctor", uid)).Run(); err != nil {
		fmt.Printf("  ⚠️  Failed to kickstart (may already be running): %v\n", err)
	} else {
		fmt.Println("  ✓ Daemon started")
	}

	fmt.Println()
	fmt.Println("Doctor daemon installed successfully!")
	fmt.Println("  To check status:   launchctl list | grep com.orch.doctor")
	fmt.Println("  To view logs:      tail -f ~/.orch/doctor.log")
	fmt.Println("  To uninstall:      orch doctor uninstall")

	return nil
}

// runDoctorUninstall removes the doctor daemon plist.
func runDoctorUninstall() error {
	plistPath := getDoctorPlistPath()
	fmt.Println("Uninstalling orch doctor daemon...")
	fmt.Println()

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Println("  Doctor daemon is not installed.")
		return nil
	}

	if err := exec.Command("launchctl", "unload", plistPath).Run(); err != nil {
		fmt.Printf("  ⚠️  Failed to unload (may not be running): %v\n", err)
	} else {
		fmt.Println("  ✓ Daemon stopped")
	}

	if err := os.Remove(plistPath); err != nil {
		return fmt.Errorf("failed to remove plist: %w", err)
	}
	fmt.Println("  ✓ Plist removed")
	fmt.Println()
	fmt.Println("Doctor daemon uninstalled successfully!")

	return nil
}
