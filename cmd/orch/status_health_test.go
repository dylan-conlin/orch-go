package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// mockConn implements the Close() interface for testing.
type mockConn struct{}

func (m *mockConn) Close() error { return nil }

// statusMockError is a simple error type for testing TCP checks.
type statusMockError struct {
	msg string
}

func (e *statusMockError) Error() string { return e.msg }

// TestCheckTCPPort tests the TCP port check functionality.
func TestCheckTCPPort(t *testing.T) {
	// Save original dial function and restore after test
	originalDial := tcpDialTimeout
	defer func() { tcpDialTimeout = originalDial }()

	tests := []struct {
		name        string
		serviceName string
		port        int
		dialError   error
		wantRunning bool
		wantDetails string
	}{
		{
			name:        "service is listening",
			serviceName: "TestService",
			port:        8080,
			dialError:   nil,
			wantRunning: true,
			wantDetails: "listening",
		},
		{
			name:        "service is not responding",
			serviceName: "TestService",
			port:        8080,
			dialError:   &statusMockError{"connection refused"},
			wantRunning: false,
			wantDetails: "not responding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the dial function
			tcpDialTimeout = func(addr string, timeout time.Duration) (interface{ Close() error }, error) {
				if tt.dialError != nil {
					return nil, tt.dialError
				}
				return &mockConn{}, nil
			}

			status := checkTCPPort(tt.serviceName, tt.port)

			if status.Running != tt.wantRunning {
				t.Errorf("checkTCPPort().Running = %v, want %v", status.Running, tt.wantRunning)
			}
			if status.Details != tt.wantDetails {
				t.Errorf("checkTCPPort().Details = %q, want %q", status.Details, tt.wantDetails)
			}
			if status.Name != tt.serviceName {
				t.Errorf("checkTCPPort().Name = %q, want %q", status.Name, tt.serviceName)
			}
			if status.Port != tt.port {
				t.Errorf("checkTCPPort().Port = %d, want %d", status.Port, tt.port)
			}
		})
	}
}

// TestCheckInfrastructureHealth tests the overall infrastructure health check.
func TestCheckInfrastructureHealth(t *testing.T) {
	// Save original dial function and restore after test
	originalDial := tcpDialTimeout
	defer func() { tcpDialTimeout = originalDial }()

	tests := []struct {
		name           string
		dashboardUp    bool
		opencodeUp     bool
		wantAllHealthy bool
	}{
		{
			name:           "all services up",
			dashboardUp:    true,
			opencodeUp:     true,
			wantAllHealthy: true, // Will be false if daemon file not found, but services will be up
		},
		{
			name:           "dashboard down",
			dashboardUp:    false,
			opencodeUp:     true,
			wantAllHealthy: false,
		},
		{
			name:           "opencode down",
			dashboardUp:    true,
			opencodeUp:     false,
			wantAllHealthy: false,
		},
		{
			name:           "both services down",
			dashboardUp:    false,
			opencodeUp:     false,
			wantAllHealthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the dial function based on port
			tcpDialTimeout = func(addr string, timeout time.Duration) (interface{ Close() error }, error) {
				// Check which port is being tested
				if strings.Contains(addr, ":3348") {
					if tt.dashboardUp {
						return &mockConn{}, nil
					}
					return nil, &statusMockError{"connection refused"}
				}
				if strings.Contains(addr, ":4096") {
					if tt.opencodeUp {
						return &mockConn{}, nil
					}
					return nil, &statusMockError{"connection refused"}
				}
				return nil, &statusMockError{"unknown port"}
			}

			health := checkInfrastructureHealth()

			// Check services count
			if len(health.Services) != 2 {
				t.Errorf("checkInfrastructureHealth() returned %d services, want 2", len(health.Services))
			}

			// Check Dashboard status
			var dashboardStatus, opencodeStatus *InfraServiceStatus
			for i := range health.Services {
				if health.Services[i].Name == "Dashboard" {
					dashboardStatus = &health.Services[i]
				}
				if health.Services[i].Name == "OpenCode" {
					opencodeStatus = &health.Services[i]
				}
			}

			if dashboardStatus == nil {
				t.Error("Dashboard service not found in health check")
			} else if dashboardStatus.Running != tt.dashboardUp {
				t.Errorf("Dashboard.Running = %v, want %v", dashboardStatus.Running, tt.dashboardUp)
			}

			if opencodeStatus == nil {
				t.Error("OpenCode service not found in health check")
			} else if opencodeStatus.Running != tt.opencodeUp {
				t.Errorf("OpenCode.Running = %v, want %v", opencodeStatus.Running, tt.opencodeUp)
			}

			// Note: AllHealthy also depends on daemon status, which we're not mocking
			// So we just verify that when services are down, AllHealthy is false
			if !tt.dashboardUp || !tt.opencodeUp {
				if health.AllHealthy {
					t.Error("AllHealthy should be false when a service is down")
				}
			}
		})
	}
}

// TestPrintInfrastructureHealth tests the infrastructure health output.
func TestPrintInfrastructureHealth(t *testing.T) {
	tests := []struct {
		name           string
		health         *InfrastructureHealth
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "nil health",
			health:         nil,
			wantContains:   []string{},
			wantNotContain: []string{"SYSTEM HEALTH"},
		},
		{
			name: "all services running",
			health: &InfrastructureHealth{
				AllHealthy: true,
				Services: []InfraServiceStatus{
					{Name: "Dashboard", Running: true, Port: 3348, Details: "listening"},
					{Name: "OpenCode", Running: true, Port: 4096, Details: "listening"},
				},
				Daemon: &DaemonStatus{Status: "running", ReadyCount: 5},
			},
			wantContains:   []string{"SYSTEM HEALTH", "✅ Dashboard", "✅ OpenCode", "✅ Daemon", "listening"},
			wantNotContain: []string{"❌"},
		},
		{
			name: "service not running",
			health: &InfrastructureHealth{
				AllHealthy: false,
				Services: []InfraServiceStatus{
					{Name: "Dashboard", Running: false, Port: 3348, Details: "not responding"},
					{Name: "OpenCode", Running: true, Port: 4096, Details: "listening"},
				},
				Daemon: nil,
			},
			wantContains: []string{"SYSTEM HEALTH", "❌ Dashboard", "✅ OpenCode", "❌ Daemon", "not responding"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printInfrastructureHealth(tt.health)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Output should contain %q, got:\n%s", want, output)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("Output should NOT contain %q, got:\n%s", notWant, output)
				}
			}
		})
	}
}
