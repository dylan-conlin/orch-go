package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/notify"
)

// runDoctorWatch runs continuous health monitoring with desktop notifications.
// Polls every 30 seconds and notifies on state transitions (healthy → unhealthy).
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
		binaryServiceStatus.Details = fmt.Sprintf("STALE (binary=%s, HEAD=%s)", shortID(binaryStatus.BinaryHash), shortID(binaryStatus.CurrentHash))
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
