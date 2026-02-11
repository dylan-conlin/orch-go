// Package main provides infrastructure health checking for the status command.
// Extracted from status_cmd.go as part of the status_cmd.go refactoring.
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// checkInfrastructureHealth checks the health of infrastructure services.
// Performs TCP connect tests for dashboard (port 3348) and OpenCode (port 4096),
// and reads daemon status from ~/.orch/daemon-status.json.
func checkInfrastructureHealth() *InfrastructureHealth {
	health := &InfrastructureHealth{
		AllHealthy: true,
		Services:   make([]InfraServiceStatus, 0, 2),
	}

	// Check Dashboard server (orch serve) on port 3348
	dashboardStatus := checkTCPPort("Dashboard", DefaultServePort)
	health.Services = append(health.Services, dashboardStatus)
	if !dashboardStatus.Running {
		health.AllHealthy = false
	}

	// Check OpenCode HTTP API server on port 4096
	opencodeStatus := checkTCPPort("OpenCode HTTP API", 4096)
	health.Services = append(health.Services, opencodeStatus)
	if !opencodeStatus.Running {
		health.AllHealthy = false
	}

	// Check daemon status from file
	daemonStatus := readDaemonStatus()
	health.Daemon = daemonStatus
	if daemonStatus == nil || daemonStatus.Status != "running" {
		health.AllHealthy = false
	}

	return health
}

// checkTCPPort performs a TCP connect test to verify a service is listening.
func checkTCPPort(name string, port int) InfraServiceStatus {
	status := InfraServiceStatus{
		Name: name,
		Port: port,
	}

	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := tcpDialTimeout(addr, 1*time.Second)
	if err != nil {
		status.Running = false
		status.Details = "not responding"
		return status
	}
	conn.Close()

	status.Running = true
	status.Details = "listening"
	return status
}

// tcpDialTimeout dials a TCP address with a timeout.
// This is a wrapper to allow for testing.
var tcpDialTimeout = tcpDialTimeoutImpl

// tcpDialTimeoutImpl is the actual implementation of TCP dial using net.DialTimeout.
func tcpDialTimeoutImpl(addr string, timeout time.Duration) (interface{ Close() error }, error) {
	return net.DialTimeout("tcp", addr, timeout)
}

// readDaemonStatus reads the daemon status from ~/.orch/daemon-status.json.
func readDaemonStatus() *DaemonStatus {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	statusPath := filepath.Join(homeDir, ".orch", "daemon-status.json")
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil
	}

	var status DaemonStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil
	}

	return &status
}

// printInfrastructureHealth prints the infrastructure health section.
func printInfrastructureHealth(health *InfrastructureHealth) {
	if health == nil {
		return
	}

	fmt.Println("SYSTEM HEALTH")
	for _, svc := range health.Services {
		emoji := "✅"
		if !svc.Running {
			emoji = "❌"
		}
		fmt.Printf("  %s %s (port %d) - %s\n", emoji, svc.Name, svc.Port, svc.Details)
	}

	// Print daemon status
	if health.Daemon != nil {
		emoji := "✅"
		if health.Daemon.Status != "running" {
			emoji = "❌"
		}
		daemonDetails := health.Daemon.Status
		if health.Daemon.Status == "running" && health.Daemon.ReadyCount > 0 {
			daemonDetails = fmt.Sprintf("%s (%d ready)", health.Daemon.Status, health.Daemon.ReadyCount)
		}
		fmt.Printf("  %s Daemon - %s\n", emoji, daemonDetails)
	} else {
		fmt.Println("  ❌ Daemon - not running")
	}
	fmt.Println()
}
