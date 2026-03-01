package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/notify"
)

// DoctorDaemonConfig holds configuration for the self-healing daemon.
type DoctorDaemonConfig struct {
	PollInterval        time.Duration
	OrphanedViteMaxAge  time.Duration
	LongRunningBdMaxAge time.Duration
	LogPath             string
	Verbose             bool
}

// DefaultDoctorDaemonConfig returns sensible defaults for the daemon.
func DefaultDoctorDaemonConfig() DoctorDaemonConfig {
	home, _ := os.UserHomeDir()
	return DoctorDaemonConfig{
		PollInterval:        30 * time.Second,
		OrphanedViteMaxAge:  5 * time.Minute,
		LongRunningBdMaxAge: 10 * time.Minute,
		LogPath:             filepath.Join(home, ".orch", "doctor.log"),
		Verbose:             doctorVerbose,
	}
}

// DoctorDaemonIntervention represents a self-healing action taken by the daemon.
type DoctorDaemonIntervention struct {
	Timestamp time.Time
	Type      string
	Target    string
	Reason    string
	Success   bool
	Error     string
}

// DoctorDaemonLogger handles logging for the self-healing daemon.
type DoctorDaemonLogger struct {
	logPath string
	verbose bool
}

// NewDoctorDaemonLogger creates a new logger for the daemon.
func NewDoctorDaemonLogger(logPath string, verbose bool) *DoctorDaemonLogger {
	dir := filepath.Dir(logPath)
	os.MkdirAll(dir, 0755)
	return &DoctorDaemonLogger{logPath: logPath, verbose: verbose}
}

// Log writes an intervention to the log file.
func (l *DoctorDaemonLogger) Log(intervention DoctorDaemonIntervention) {
	f, err := os.OpenFile(l.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		line := fmt.Sprintf("[%s] %s: %s (%s) success=%v",
			intervention.Timestamp.Format("2006-01-02 15:04:05"),
			intervention.Type, intervention.Target, intervention.Reason, intervention.Success)
		if intervention.Error != "" {
			line += fmt.Sprintf(" error=%s", intervention.Error)
		}
		f.WriteString(line + "\n")
	}
	if l.verbose {
		status := "✓"
		if !intervention.Success {
			status = "✗"
		}
		fmt.Printf("[%s] %s %s: %s (%s)\n",
			intervention.Timestamp.Format("15:04:05"), status, intervention.Type,
			intervention.Target, intervention.Reason)
	}
}

// runDoctorDaemon runs the self-healing background daemon.
func runDoctorDaemon() error {
	config := DefaultDoctorDaemonConfig()
	logger := NewDoctorDaemonLogger(config.LogPath, config.Verbose)
	notifier := notify.Default()

	fmt.Println("orch doctor --daemon")
	fmt.Println("Self-Healing Background Daemon")
	fmt.Println("==============================")
	fmt.Printf("Poll interval: %s\n", config.PollInterval)
	fmt.Printf("Log file:      %s\n", config.LogPath)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	previousHealth := make(map[string]bool)
	totalInterventions := 0

	interventions := runDaemonHealthCycle(config, logger, notifier, previousHealth)
	totalInterventions += interventions

	for {
		select {
		case <-ticker.C:
			interventions := runDaemonHealthCycle(config, logger, notifier, previousHealth)
			totalInterventions += interventions
		case <-sigChan:
			fmt.Printf("\nDaemon stopped. Total interventions: %d\n", totalInterventions)
			return nil
		}
	}
}

// runDaemonHealthCycle runs one cycle of health checks and self-healing.
func runDaemonHealthCycle(config DoctorDaemonConfig, logger *DoctorDaemonLogger, notifier *notify.Notifier, previousHealth map[string]bool) int {
	interventions := 0
	timestamp := time.Now()
	timeStr := timestamp.Format("15:04:05")

	killed := killOrphanedViteProcesses(config, logger)
	interventions += killed

	killed = killLongRunningBdProcesses(config, logger)
	interventions += killed

	restarted := restartCrashedServices(config, logger)
	interventions += restarted

	report := &DoctorReport{Healthy: true, Services: make([]ServiceStatus, 0)}
	for _, status := range []ServiceStatus{checkOpenCode(), checkOrchServe(), checkWebUI(), checkOvermindServices()} {
		report.Services = append(report.Services, status)
		if !status.Running {
			report.Healthy = false
		}
	}

	if config.Verbose {
		if report.Healthy && interventions == 0 {
			fmt.Printf("[%s] ✓ All healthy, no interventions\n", timeStr)
		} else if interventions > 0 {
			fmt.Printf("[%s] Interventions: %d\n", timeStr, interventions)
		}
	}

	for _, svc := range report.Services {
		wasHealthy, exists := previousHealth[svc.Name]
		isHealthy := svc.Running
		previousHealth[svc.Name] = isHealthy
		if exists && wasHealthy && !isHealthy {
			notifier.ServiceCrashed(svc.Name, "orch-go")
			logger.Log(DoctorDaemonIntervention{
				Timestamp: timestamp, Type: "service_down", Target: svc.Name, Reason: svc.Details, Success: true,
			})
		}
	}

	return interventions
}

// killOrphanedViteProcesses finds and kills vite processes with PPID=1 (orphaned).
func killOrphanedViteProcesses(config DoctorDaemonConfig, logger *DoctorDaemonLogger) int {
	cmd := exec.Command("ps", "-eo", "pid,ppid,etime,comm")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	killed := 0
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "vite") && !strings.Contains(line, "node") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pid, ppid, etime := fields[0], fields[1], fields[2]
		if ppid != "1" {
			continue
		}
		elapsed := parseElapsedTime(etime)
		if elapsed < config.OrphanedViteMaxAge {
			continue
		}
		killCmd := exec.Command("kill", "-9", pid)
		err := killCmd.Run()
		intervention := DoctorDaemonIntervention{
			Timestamp: time.Now(), Type: "kill_orphan_vite", Target: fmt.Sprintf("PID %s", pid),
			Reason: fmt.Sprintf("orphaned vite (PPID=1, age=%s)", etime), Success: err == nil,
		}
		if err != nil {
			intervention.Error = err.Error()
		} else {
			killed++
		}
		logger.Log(intervention)
	}
	return killed
}

// killLongRunningBdProcesses finds and kills bd processes running too long.
func killLongRunningBdProcesses(config DoctorDaemonConfig, logger *DoctorDaemonLogger) int {
	cmd := exec.Command("ps", "-eo", "pid,etime,comm")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	killed := 0
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasSuffix(line, "bd") && !strings.Contains(line, "/bd ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, etime := fields[0], fields[1]
		elapsed := parseElapsedTime(etime)
		if elapsed < config.LongRunningBdMaxAge {
			continue
		}
		killCmd := exec.Command("kill", "-9", pid)
		err := killCmd.Run()
		intervention := DoctorDaemonIntervention{
			Timestamp: time.Now(), Type: "kill_long_bd", Target: fmt.Sprintf("PID %s", pid),
			Reason: fmt.Sprintf("long-running bd (age=%s > %s)", etime, config.LongRunningBdMaxAge), Success: err == nil,
		}
		if err != nil {
			intervention.Error = err.Error()
		} else {
			killed++
		}
		logger.Log(intervention)
	}
	return killed
}

// restartCrashedServices checks launchd services and restarts any that are crashed.
func restartCrashedServices(config DoctorDaemonConfig, logger *DoctorDaemonLogger) int {
	services := []struct {
		Label     string
		CheckPort int
		CheckFunc func() ServiceStatus
	}{
		{Label: "com.opencode.serve", CheckPort: 4096, CheckFunc: checkOpenCode},
	}

	restarted := 0
	for _, svc := range services {
		status := svc.CheckFunc()
		if status.Running {
			continue
		}
		uidCmd := exec.Command("id", "-u")
		uidOutput, err := uidCmd.Output()
		if err != nil {
			continue
		}
		uid := strings.TrimSpace(string(uidOutput))
		kickstartCmd := exec.Command("launchctl", "kickstart", "-k", fmt.Sprintf("gui/%s/%s", uid, svc.Label))
		err = kickstartCmd.Run()
		intervention := DoctorDaemonIntervention{
			Timestamp: time.Now(), Type: "restart_service", Target: svc.Label,
			Reason: fmt.Sprintf("not responding on port %d", svc.CheckPort), Success: err == nil,
		}
		if err != nil {
			intervention.Error = err.Error()
		} else {
			restarted++
		}
		logger.Log(intervention)
	}
	return restarted
}

// parseElapsedTime parses ps elapsed time format.
func parseElapsedTime(etime string) time.Duration {
	etime = strings.TrimSpace(etime)
	if strings.Contains(etime, "-") {
		parts := strings.SplitN(etime, "-", 2)
		if len(parts) == 2 {
			days := 0
			fmt.Sscanf(parts[0], "%d", &days)
			return time.Duration(days)*24*time.Hour + parseElapsedTime(parts[1])
		}
	}
	parts := strings.Split(etime, ":")
	switch len(parts) {
	case 2:
		var mins, secs int
		fmt.Sscanf(parts[0], "%d", &mins)
		fmt.Sscanf(parts[1], "%d", &secs)
		return time.Duration(mins)*time.Minute + time.Duration(secs)*time.Second
	case 3:
		var hours, mins, secs int
		fmt.Sscanf(parts[0], "%d", &hours)
		fmt.Sscanf(parts[1], "%d", &mins)
		fmt.Sscanf(parts[2], "%d", &secs)
		return time.Duration(hours)*time.Hour + time.Duration(mins)*time.Minute + time.Duration(secs)*time.Second
	}
	return 0
}

// getDoctorPlistPath returns the path to the doctor daemon's launchd plist.
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
