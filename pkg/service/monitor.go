// Package service provides service monitoring for overmind-managed processes.
package service

import (
	"bufio"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HealthProbe defines an HTTP health check for a specific service.
// When configured, the monitor makes HTTP GET requests to the URL and
// considers the service unresponsive if it fails consecutively.
type HealthProbe struct {
	URL     string        // Health check endpoint (e.g., "http://127.0.0.1:4096/session")
	Timeout time.Duration // HTTP request timeout
}

// ServiceState represents the state of a single service at a point in time.
type ServiceState struct {
	Name         string
	PID          int
	Status       string // "running", "stopped", etc.
	LastSeen     time.Time
	RestartCount      int
	UnresponsiveCount int // Consecutive health check failures while PID unchanged
}

// ServiceNotifier defines the interface for sending service crash notifications.
// This abstraction allows for dependency injection and testing.
type ServiceNotifier interface {
	ServiceCrashed(serviceName string, projectPath string) error
}

// EventLogger defines the interface for logging service lifecycle events.
type EventLogger interface {
	LogServiceCrashed(serviceName, projectPath string, oldPID, newPID int) error
	LogServiceRestarted(serviceName, projectPath string, newPID, restartCount int, autoRestart bool) error
	LogServiceStarted(serviceName, projectPath string, pid int) error
	LogServiceUnresponsive(serviceName, projectPath string, pid, consecutiveFailures int) error
}

// DefaultUnresponsiveThreshold is the number of consecutive health check failures
// before a service is considered unresponsive and force-restarted.
// At 10s polling interval, 3 failures = 30 seconds of unresponsiveness.
const DefaultUnresponsiveThreshold = 3

// ServiceMonitor monitors overmind-managed services and detects crashes.
type ServiceMonitor struct {
	projectPath  string
	lastState    map[string]ServiceState
	notifier     ServiceNotifier
	eventLogger  EventLogger
	mu           sync.RWMutex
	interval     time.Duration
	autoRestart           bool // Whether to auto-restart crashed services
	sessionStart          time.Time
	healthProbes          map[string]HealthProbe // service name → health check config
	unresponsiveThreshold int                    // consecutive failures before restart
}

// NewMonitor creates a new ServiceMonitor for the given project path.
// The monitor polls overmind status at the specified interval.
// If autoRestart is true, crashed services will be restarted automatically.
func NewMonitor(projectPath string, notifier ServiceNotifier, eventLogger EventLogger, interval time.Duration, autoRestart bool) *ServiceMonitor {
	return &ServiceMonitor{
		projectPath:  projectPath,
		lastState:    make(map[string]ServiceState),
		notifier:     notifier,
		eventLogger:  eventLogger,
		interval:     interval,
		autoRestart:           autoRestart,
		sessionStart:          time.Now(),
		healthProbes:          make(map[string]HealthProbe),
		unresponsiveThreshold: DefaultUnresponsiveThreshold,
	}
}

// AddHealthProbe registers an HTTP health check for a specific service.
// When the service's PID is unchanged but the health check fails for
// unresponsiveThreshold consecutive polls, the service is force-restarted.
func (m *ServiceMonitor) AddHealthProbe(serviceName string, probe HealthProbe) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthProbes[serviceName] = probe
}

// Start begins the monitoring loop in a background goroutine.
// It polls overmind status at regular intervals and detects crashes.
// The goroutine continues until the context is cancelled.
func (m *ServiceMonitor) Start() {
	ticker := time.NewTicker(m.interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if err := m.Poll(); err != nil {
				// Log error but continue monitoring
				// In production, we'd use a proper logger
				fmt.Printf("Service monitor poll error: %v\n", err)
			}
		}
	}()
}

// Poll runs a single monitoring cycle: fetch current state, detect crashes,
// run health probes, and update state.
func (m *ServiceMonitor) Poll() error {
	// Run overmind status to get current service states
	currentStates, err := m.fetchOvermindStatus()
	if err != nil {
		return fmt.Errorf("failed to fetch overmind status: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Detect crashes by comparing current state to last known state
	crashedServices := m.detectAndHandleCrashes(currentStates)

	// Auto-restart crashed services if enabled
	if m.autoRestart && len(crashedServices) > 0 {
		for _, crash := range crashedServices {
			if err := m.restartService(crash.Name); err != nil {
				fmt.Printf("Failed to restart service %s: %v\n", crash.Name, err)
			} else {
				// Wait a moment for service to start, then re-fetch status
				time.Sleep(1 * time.Second)
				newStates, _ := m.fetchOvermindStatus()
				// Find the new PID for this service
				for _, s := range newStates {
					if s.Name == crash.Name {
						// Log restart event
						if m.eventLogger != nil {
							m.eventLogger.LogServiceRestarted(crash.Name, m.projectPath, s.PID, crash.RestartCount+1, true)
						}
						break
					}
				}
			}
		}
		// Re-fetch status after restarts
		currentStates, _ = m.fetchOvermindStatus()
	}

	// Run HTTP health probes for services that have them configured.
	// This detects unresponsiveness (process alive but not serving) which
	// PID-based crash detection cannot catch.
	unresponsiveServices := m.checkHealthProbes(currentStates)
	if m.autoRestart && len(unresponsiveServices) > 0 {
		for _, svc := range unresponsiveServices {
			fmt.Printf("Service %s is unresponsive (PID %d, %d consecutive failures), restarting\n",
				svc.Name, svc.PID, svc.UnresponsiveCount)

			if m.eventLogger != nil {
				m.eventLogger.LogServiceUnresponsive(svc.Name, m.projectPath, svc.PID, svc.UnresponsiveCount)
			}

			if err := m.notifier.ServiceCrashed(
				fmt.Sprintf("⚠️ %s unresponsive (PID %d alive but not serving, %d failures), restarting",
					svc.Name, svc.PID, svc.UnresponsiveCount),
				m.projectPath,
			); err != nil {
				fmt.Printf("Failed to send unresponsive notification for %s: %v\n", svc.Name, err)
			}

			if err := m.restartService(svc.Name); err != nil {
				fmt.Printf("Failed to restart unresponsive service %s: %v\n", svc.Name, err)
			} else {
				time.Sleep(1 * time.Second)
				newStates, _ := m.fetchOvermindStatus()
				for _, s := range newStates {
					if s.Name == svc.Name {
						if m.eventLogger != nil {
							m.eventLogger.LogServiceRestarted(svc.Name, m.projectPath, s.PID, svc.RestartCount+1, true)
						}
						break
					}
				}
			}
		}
		// Re-fetch status after restarts
		currentStates, _ = m.fetchOvermindStatus()
	}

	// Update state for next poll
	m.updateState(currentStates)

	return nil
}

// checkHealthProbes runs HTTP health checks for services that have probes configured.
// Returns services that have exceeded the unresponsive threshold.
func (m *ServiceMonitor) checkHealthProbes(currentStates []ServiceState) []ServiceState {
	var unresponsive []ServiceState

	for _, current := range currentStates {
		probe, hasProbe := m.healthProbes[current.Name]
		if !hasProbe || current.Status != "running" {
			continue
		}

		healthy := m.probeHealth(probe)
		last, exists := m.lastState[current.Name]

		if healthy {
			// Reset unresponsive counter on success
			if exists {
				last.UnresponsiveCount = 0
				m.lastState[current.Name] = last
			}
			continue
		}

		// Health check failed — only count if PID unchanged (not a crash)
		if !exists || last.PID != current.PID {
			continue
		}

		last.UnresponsiveCount++
		m.lastState[current.Name] = last

		if last.UnresponsiveCount >= m.unresponsiveThreshold {
			unresponsive = append(unresponsive, ServiceState{
				Name:              current.Name,
				PID:               current.PID,
				Status:            current.Status,
				LastSeen:          time.Now(),
				RestartCount:      last.RestartCount,
				UnresponsiveCount: last.UnresponsiveCount,
			})
			// Reset counter after triggering restart
			last.UnresponsiveCount = 0
			m.lastState[current.Name] = last
		}
	}

	return unresponsive
}

// probeHealth makes an HTTP GET request to the health probe URL.
// Returns true if the service responds with any 2xx status code.
func (m *ServiceMonitor) probeHealth(probe HealthProbe) bool {
	client := &http.Client{Timeout: probe.Timeout}
	resp, err := client.Get(probe.URL)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// detectAndHandleCrashes detects crashed services and handles notifications/logging.
// Returns list of crashed services with their metadata.
func (m *ServiceMonitor) detectAndHandleCrashes(currentList []ServiceState) []ServiceState {
	var crashedServices []ServiceState

	for _, current := range currentList {
		last, exists := m.lastState[current.Name]
		if !exists {
			// New service (first time seeing it)
			if m.eventLogger != nil {
				m.eventLogger.LogServiceStarted(current.Name, m.projectPath, current.PID)
			}
			continue
		}

		// Detect crash: PID changed
		if last.PID != 0 && last.PID != current.PID {
			crashedServices = append(crashedServices, ServiceState{
				Name:         current.Name,
				PID:          current.PID,
				Status:       current.Status,
				LastSeen:     time.Now(),
				RestartCount: last.RestartCount,
			})

			// Log crash event
			if m.eventLogger != nil {
				m.eventLogger.LogServiceCrashed(current.Name, m.projectPath, last.PID, current.PID)
			}

			// Send notification with restart count
			restartCount := last.RestartCount + 1
			notificationMsg := fmt.Sprintf("🔄 %s crashed and will be restarted (restart #%d)", current.Name, restartCount)
			if err := m.notifier.ServiceCrashed(notificationMsg, m.projectPath); err != nil {
				fmt.Printf("Failed to send crash notification for %s: %v\n", current.Name, err)
			}
		}
	}

	return crashedServices
}

// restartService restarts a service using overmind restart.
func (m *ServiceMonitor) restartService(serviceName string) error {
	cmd := exec.Command("overmind", "restart", serviceName)
	cmd.Dir = m.projectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart failed: %w (output: %s)", err, string(output))
	}
	fmt.Printf("Service %s restarted successfully\n", serviceName)
	return nil
}

// fetchOvermindStatus runs overmind status and returns parsed service states.
func (m *ServiceMonitor) fetchOvermindStatus() ([]ServiceState, error) {
	// Run overmind status from the project directory
	cmd := exec.Command("overmind", "status")
	cmd.Dir = m.projectPath

	output, err := cmd.Output()
	if err != nil {
		// If overmind isn't running or not found, return empty list (no services)
		return []ServiceState{}, nil
	}

	return parseOvermindStatus(string(output))
}

// parseOvermindStatus parses the text output from overmind status.
// Expected format:
//
//	PROCESS   PID       STATUS
//	api       82423     running
//	web       82424     running
func parseOvermindStatus(output string) ([]ServiceState, error) {
	var states []ServiceState
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Skip header line
	if scanner.Scan() {
		// First line is "PROCESS   PID       STATUS"
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Split by whitespace
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue // Skip malformed lines
		}

		name := fields[0]
		pidStr := fields[1]
		status := fields[2]

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			// If PID parsing fails, treat as 0 (stopped)
			pid = 0
		}

		states = append(states, ServiceState{
			Name:     name,
			PID:      pid,
			Status:   status,
			LastSeen: time.Now(),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning overmind output: %w", err)
	}

	return states, nil
}

// detectCrashes compares last known state to current state and returns crashed service names.
// A crash is detected when:
//  1. A service's PID changed from non-zero to zero (stopped)
//  2. A service's PID changed to a different non-zero value (crashed and restarted)
func detectCrashes(lastState map[string]ServiceState, currentList []ServiceState) []string {
	var crashes []string

	for _, current := range currentList {
		last, exists := lastState[current.Name]
		if !exists {
			// New service (first time seeing it), not a crash
			continue
		}

		// Detect crash: PID changed
		if last.PID != 0 && last.PID != current.PID {
			crashes = append(crashes, current.Name)
		}
	}

	return crashes
}

// updateState updates the internal state map with new service states.
func (m *ServiceMonitor) updateState(states []ServiceState) {
	for _, s := range states {
		// If service already exists, preserve restart count and unresponsive count
		if existing, ok := m.lastState[s.Name]; ok {
			// If PID changed, increment restart count
			if existing.PID != s.PID && s.PID != 0 {
				s.RestartCount = existing.RestartCount + 1
			} else {
				s.RestartCount = existing.RestartCount
			}
			// Preserve unresponsive count across state updates
			s.UnresponsiveCount = existing.UnresponsiveCount
		}
		m.lastState[s.Name] = s
	}
}

// GetState returns the current known state of all services (for debugging/testing).
func (m *ServiceMonitor) GetState() map[string]ServiceState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	stateCopy := make(map[string]ServiceState)
	for k, v := range m.lastState {
		stateCopy[k] = v
	}
	return stateCopy
}
