package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestParseOvermindStatus verifies that we can parse the text output of overmind status
func TestParseOvermindStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ServiceState
		wantErr  bool
	}{
		{
			name: "normal running services",
			input: `PROCESS   PID       STATUS
api       82423     running
web       82424     running
opencode  82425     running`,
			expected: []ServiceState{
				{Name: "api", PID: 82423, Status: "running"},
				{Name: "web", PID: 82424, Status: "running"},
				{Name: "opencode", PID: 82425, Status: "running"},
			},
			wantErr: false,
		},
		{
			name: "mixed statuses",
			input: `PROCESS   PID       STATUS
api       82423     running
web       0         stopped
opencode  82425     running`,
			expected: []ServiceState{
				{Name: "api", PID: 82423, Status: "running"},
				{Name: "web", PID: 0, Status: "stopped"},
				{Name: "opencode", PID: 82425, Status: "running"},
			},
			wantErr: false,
		},
		{
			name:     "empty output",
			input:    "",
			expected: []ServiceState{},
			wantErr:  false,
		},
		{
			name:     "header only",
			input:    `PROCESS   PID       STATUS`,
			expected: []ServiceState{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOvermindStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOvermindStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.expected) {
				t.Errorf("parseOvermindStatus() got %d services, want %d", len(got), len(tt.expected))
				return
			}
			for i, s := range got {
				if s.Name != tt.expected[i].Name || s.PID != tt.expected[i].PID || s.Status != tt.expected[i].Status {
					t.Errorf("parseOvermindStatus() service[%d] = %+v, want %+v", i, s, tt.expected[i])
				}
			}
		})
	}
}

// TestDetectCrashes verifies that we correctly identify crashed services
func TestDetectCrashes(t *testing.T) {
	tests := []struct {
		name        string
		lastState   map[string]ServiceState
		currentList []ServiceState
		expectCrash []string // service names that should be detected as crashed
	}{
		{
			name: "service crashed (PID changed to 0)",
			lastState: map[string]ServiceState{
				"api": {Name: "api", PID: 12345, Status: "running", LastSeen: time.Now()},
			},
			currentList: []ServiceState{
				{Name: "api", PID: 0, Status: "stopped"},
			},
			expectCrash: []string{"api"},
		},
		{
			name: "service restarted (PID changed, still running)",
			lastState: map[string]ServiceState{
				"api": {Name: "api", PID: 12345, Status: "running", LastSeen: time.Now()},
			},
			currentList: []ServiceState{
				{Name: "api", PID: 99999, Status: "running"},
			},
			expectCrash: []string{"api"}, // PID change = crash + restart
		},
		{
			name: "no change",
			lastState: map[string]ServiceState{
				"api": {Name: "api", PID: 12345, Status: "running", LastSeen: time.Now()},
			},
			currentList: []ServiceState{
				{Name: "api", PID: 12345, Status: "running"},
			},
			expectCrash: []string{},
		},
		{
			name:      "first run (no previous state)",
			lastState: map[string]ServiceState{},
			currentList: []ServiceState{
				{Name: "api", PID: 12345, Status: "running"},
			},
			expectCrash: []string{}, // No crashes on first run
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crashes := detectCrashes(tt.lastState, tt.currentList)
			if len(crashes) != len(tt.expectCrash) {
				t.Errorf("detectCrashes() got %d crashes, want %d", len(crashes), len(tt.expectCrash))
				t.Logf("Got crashes: %v", crashes)
				t.Logf("Want crashes: %v", tt.expectCrash)
				return
			}
			// Check that all expected crashes are present
			crashMap := make(map[string]bool)
			for _, name := range crashes {
				crashMap[name] = true
			}
			for _, expected := range tt.expectCrash {
				if !crashMap[expected] {
					t.Errorf("detectCrashes() missing expected crash: %s", expected)
				}
			}
		})
	}
}

// MockNotifier for testing (doesn't send actual notifications)
type MockNotifier struct {
	notifications []struct {
		title   string
		message string
	}
}

func (m *MockNotifier) ServiceCrashed(serviceName string, projectPath string) error {
	m.notifications = append(m.notifications, struct {
		title   string
		message string
	}{
		title:   "Service Crashed: " + serviceName,
		message: "Project: " + projectPath,
	})
	return nil
}

// TestServiceMonitorPoll verifies the full poll cycle
func TestServiceMonitorPoll(t *testing.T) {
	mockNotifier := &MockNotifier{}
	monitor := &ServiceMonitor{
		projectPath:           "/test/project",
		lastState:             make(map[string]ServiceState),
		notifier:              mockNotifier,
		healthProbes:          make(map[string]HealthProbe),
		unresponsiveThreshold: DefaultUnresponsiveThreshold,
	}

	// Simulate first poll (services running)
	firstStatus := `PROCESS   PID       STATUS
api       12345     running
web       12346     running`

	states, err := parseOvermindStatus(firstStatus)
	if err != nil {
		t.Fatalf("Failed to parse first status: %v", err)
	}
	monitor.updateState(states)

	// No crashes expected on first poll
	if len(mockNotifier.notifications) != 0 {
		t.Errorf("Expected no notifications on first poll, got %d", len(mockNotifier.notifications))
	}

	// Simulate second poll (api crashed)
	secondStatus := `PROCESS   PID       STATUS
api       0         stopped
web       12346     running`

	states, err = parseOvermindStatus(secondStatus)
	if err != nil {
		t.Fatalf("Failed to parse second status: %v", err)
	}

	crashes := detectCrashes(monitor.lastState, states)
	if len(crashes) != 1 || crashes[0] != "api" {
		t.Errorf("Expected crash detection for 'api', got: %v", crashes)
	}
}

// Benchmark for parsing performance (should be fast for 3-10 services)
func BenchmarkParseOvermindStatus(b *testing.B) {
	input := `PROCESS   PID       STATUS
api       82423     running
web       82424     running
opencode  82425     running
db        82426     running
cache     82427     running
worker    82428     running
scheduler 82429     running
monitor   82430     running
logger    82431     running
proxy     82432     running`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseOvermindStatus(input)
	}
}

// Helper function to check if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, val) {
			return true
		}
	}
	return false
}

// MockEventLogger satisfies the EventLogger interface for testing.
type MockEventLogger struct {
	crashed      []string
	restarted    []string
	started      []string
	unresponsive []string
}

func (m *MockEventLogger) LogServiceCrashed(serviceName, projectPath string, oldPID, newPID int) error {
	m.crashed = append(m.crashed, serviceName)
	return nil
}

func (m *MockEventLogger) LogServiceRestarted(serviceName, projectPath string, newPID, restartCount int, autoRestart bool) error {
	m.restarted = append(m.restarted, serviceName)
	return nil
}

func (m *MockEventLogger) LogServiceStarted(serviceName, projectPath string, pid int) error {
	m.started = append(m.started, serviceName)
	return nil
}

func (m *MockEventLogger) LogServiceUnresponsive(serviceName, projectPath string, pid, consecutiveFailures int) error {
	m.unresponsive = append(m.unresponsive, fmt.Sprintf("%s:%d", serviceName, consecutiveFailures))
	return nil
}

// TestCheckHealthProbes_HealthyService verifies that healthy services reset unresponsive count.
func TestCheckHealthProbes_HealthyService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	monitor := &ServiceMonitor{
		projectPath: "/test",
		lastState: map[string]ServiceState{
			"opencode": {Name: "opencode", PID: 1234, Status: "running", UnresponsiveCount: 2},
		},
		healthProbes: map[string]HealthProbe{
			"opencode": {URL: server.URL, Timeout: 2 * time.Second},
		},
		unresponsiveThreshold: 3,
	}

	currentStates := []ServiceState{
		{Name: "opencode", PID: 1234, Status: "running"},
	}

	unresponsive := monitor.checkHealthProbes(currentStates)
	if len(unresponsive) != 0 {
		t.Errorf("Expected no unresponsive services, got %d", len(unresponsive))
	}
	if monitor.lastState["opencode"].UnresponsiveCount != 0 {
		t.Errorf("Expected UnresponsiveCount reset to 0, got %d", monitor.lastState["opencode"].UnresponsiveCount)
	}
}

// TestCheckHealthProbes_UnresponsiveService verifies unresponsive detection after threshold.
func TestCheckHealthProbes_UnresponsiveService(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	monitor := &ServiceMonitor{
		projectPath: "/test",
		lastState: map[string]ServiceState{
			"opencode": {Name: "opencode", PID: 1234, Status: "running", UnresponsiveCount: 0},
		},
		healthProbes: map[string]HealthProbe{
			"opencode": {URL: server.URL, Timeout: 2 * time.Second},
		},
		unresponsiveThreshold: 3,
	}

	currentStates := []ServiceState{
		{Name: "opencode", PID: 1234, Status: "running"},
	}

	// First two checks — not yet at threshold
	for i := 1; i <= 2; i++ {
		unresponsive := monitor.checkHealthProbes(currentStates)
		if len(unresponsive) != 0 {
			t.Errorf("Poll %d: Expected no unresponsive services, got %d", i, len(unresponsive))
		}
	}

	// Third check — reaches threshold
	unresponsive := monitor.checkHealthProbes(currentStates)
	if len(unresponsive) != 1 {
		t.Fatalf("Poll 3: Expected 1 unresponsive service, got %d", len(unresponsive))
	}
	if unresponsive[0].Name != "opencode" {
		t.Errorf("Expected 'opencode', got '%s'", unresponsive[0].Name)
	}
	if monitor.lastState["opencode"].UnresponsiveCount != 0 {
		t.Errorf("Expected counter reset, got %d", monitor.lastState["opencode"].UnresponsiveCount)
	}
}

// TestCheckHealthProbes_PIDChange skips count when PID changes.
func TestCheckHealthProbes_PIDChange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	monitor := &ServiceMonitor{
		projectPath: "/test",
		lastState: map[string]ServiceState{
			"opencode": {Name: "opencode", PID: 1234, Status: "running", UnresponsiveCount: 2},
		},
		healthProbes: map[string]HealthProbe{
			"opencode": {URL: server.URL, Timeout: 2 * time.Second},
		},
		unresponsiveThreshold: 3,
	}

	currentStates := []ServiceState{
		{Name: "opencode", PID: 5678, Status: "running"},
	}

	unresponsive := monitor.checkHealthProbes(currentStates)
	if len(unresponsive) != 0 {
		t.Errorf("Expected no unresponsive after PID change, got %d", len(unresponsive))
	}
}

// TestCheckHealthProbes_NoProbeConfigured verifies services without probes are skipped.
func TestCheckHealthProbes_NoProbeConfigured(t *testing.T) {
	monitor := &ServiceMonitor{
		projectPath:           "/test",
		lastState:             map[string]ServiceState{"web": {Name: "web", PID: 1234, Status: "running"}},
		healthProbes:          map[string]HealthProbe{},
		unresponsiveThreshold: 3,
	}

	unresponsive := monitor.checkHealthProbes([]ServiceState{{Name: "web", PID: 1234, Status: "running"}})
	if len(unresponsive) != 0 {
		t.Errorf("Expected no checks, got %d", len(unresponsive))
	}
}

// TestCheckHealthProbes_ConnectionRefused verifies connection refused behavior.
func TestCheckHealthProbes_ConnectionRefused(t *testing.T) {
	monitor := &ServiceMonitor{
		projectPath: "/test",
		lastState: map[string]ServiceState{
			"opencode": {Name: "opencode", PID: 1234, Status: "running", UnresponsiveCount: 0},
		},
		healthProbes: map[string]HealthProbe{
			"opencode": {URL: "http://127.0.0.1:19999/session", Timeout: 500 * time.Millisecond},
		},
		unresponsiveThreshold: 2,
	}

	currentStates := []ServiceState{{Name: "opencode", PID: 1234, Status: "running"}}

	monitor.checkHealthProbes(currentStates)
	unresponsive := monitor.checkHealthProbes(currentStates)
	if len(unresponsive) != 1 {
		t.Errorf("Expected 1 unresponsive, got %d", len(unresponsive))
	}
}

// TestAddHealthProbe verifies the AddHealthProbe method.
func TestAddHealthProbe(t *testing.T) {
	monitor := NewMonitor("/test", &MockNotifier{}, &MockEventLogger{}, 10*time.Second, true)
	monitor.AddHealthProbe("opencode", HealthProbe{
		URL:     "http://127.0.0.1:4096/session",
		Timeout: 5 * time.Second,
	})

	if _, ok := monitor.healthProbes["opencode"]; !ok {
		t.Error("Expected health probe registered")
	}
}
