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

func runDoctorWatch() error {
	fmt.Println("orch doctor --watch")
	fmt.Println("Continuous Health Monitoring")
	fmt.Println("============================")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Create notifier
	notifier := notify.Default()

	// Track previous health state to detect transitions
	previousHealth := make(map[string]bool)

	// Set up signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create ticker for 30-second polling
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Run initial check immediately
	runHealthCheckWithNotifications(notifier, previousHealth)

	// Main watch loop
	for {
		select {
		case <-ticker.C:
			runHealthCheckWithNotifications(notifier, previousHealth)
		case <-sigChan:
			fmt.Println("\nStopping health monitoring...")
			return nil
		}
	}
}

// runHealthCheckWithNotifications runs a health check and sends notifications on state changes.
func runHealthCheckWithNotifications(notifier *notify.Notifier, previousHealth map[string]bool) {
	// Run all health checks
	report := &DoctorReport{
		Healthy:  true,
		Services: make([]ServiceStatus, 0),
	}

	// Check all services (same as regular doctor command)
	binaryStatus := checkStaleBinary()
	binaryServiceStatus := ServiceStatus{
		Name:   "orch binary",
		CanFix: false,
	}
	if binaryStatus.Error != "" {
		binaryServiceStatus.Running = true
		binaryServiceStatus.Details = binaryStatus.Error
	} else if binaryStatus.Stale {
		binaryServiceStatus.Running = false
		binaryServiceStatus.Details = fmt.Sprintf("STALE (binary=%s, HEAD=%s)", binaryStatus.BinaryHash[:12], binaryStatus.CurrentHash[:12])
		report.Healthy = false
	} else {
		binaryServiceStatus.Running = true
		binaryServiceStatus.Details = "UP TO DATE"
	}
	report.Services = append(report.Services, binaryServiceStatus)

	openCodeStatus := checkOpenCode()
	report.Services = append(report.Services, openCodeStatus)
	if !openCodeStatus.Running {
		report.Healthy = false
	}

	orchServeStatus := checkOrchServe()
	report.Services = append(report.Services, orchServeStatus)
	if !orchServeStatus.Running {
		report.Healthy = false
	}

	webUIStatus := checkWebUI()
	report.Services = append(report.Services, webUIStatus)
	if !webUIStatus.Running {
		report.Healthy = false
	}

	overmindStatus := checkOvermindServices()
	report.Services = append(report.Services, overmindStatus)
	if !overmindStatus.Running {
		report.Healthy = false
	}

	beadsDaemonStatus := checkBeadsDaemon()
	report.Services = append(report.Services, beadsDaemonStatus)
	// Beads daemon is optional

	stalledStatus := checkStalledSessions()
	report.Services = append(report.Services, stalledStatus)
	if !stalledStatus.Running {
		report.Healthy = false
	}

	// Correctness checks (also in watch mode)
	beadsIntegrityStatus := checkBeadsIntegrity()
	report.Services = append(report.Services, beadsIntegrityStatus)
	if !beadsIntegrityStatus.Running {
		report.Healthy = false
	}

	dockerStatus := checkDockerBackend()
	report.Services = append(report.Services, dockerStatus)
	if !dockerStatus.Running {
		report.Healthy = false
	}

	// Print current status with timestamp
	fmt.Printf("[%s] ", time.Now().Format("15:04:05"))
	if report.Healthy {
		fmt.Println("✓ All services healthy")
	} else {
		fmt.Printf("✗ %d service(s) unhealthy\n", countUnhealthy(report.Services))
	}

	// Check for state transitions and send notifications
	for _, svc := range report.Services {
		wasHealthy, exists := previousHealth[svc.Name]
		isHealthy := svc.Running

		// Update current state
		previousHealth[svc.Name] = isHealthy

		// Notify only on transition from healthy to unhealthy
		if exists && wasHealthy && !isHealthy {
			message := fmt.Sprintf("%s: %s", svc.Name, svc.Details)
			if err := notifier.ServiceCrashed(svc.Name, "orch-go"); err != nil {
				fmt.Printf("  ⚠️  Failed to send notification: %v\n", err)
			} else {
				fmt.Printf("  📬 Notification sent: %s\n", message)
			}
		}

		// Print current unhealthy services
		if !isHealthy {
			fmt.Printf("  ✗ %s: %s\n", svc.Name, svc.Details)
		}
	}
}

// countUnhealthy counts the number of unhealthy services in a report.
func countUnhealthy(services []ServiceStatus) int {
	count := 0
	for _, svc := range services {
		if !svc.Running {
			count++
		}
	}
	return count
}

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
	// Liveness checks
	for _, status := range []ServiceStatus{checkOpenCode(), checkOrchServe(), checkWebUI(), checkOvermindServices()} {
		report.Services = append(report.Services, status)
		if !status.Running {
			report.Healthy = false
		}
	}
	// Correctness checks (also in daemon mode)
	for _, status := range []ServiceStatus{checkBeadsIntegrity(), checkDockerBackend()} {
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
