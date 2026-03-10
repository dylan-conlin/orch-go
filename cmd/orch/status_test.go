package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// TestFormatDuration tests the formatDuration function.
// Note: formatDuration is defined in wait.go
func TestFormatDurationForStatus(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes and seconds", 5*time.Minute + 23*time.Second, "5m 23s"},
		{"hours and minutes", 1*time.Hour + 2*time.Minute, "1h 2m"},
		{"zero", 0, "0s"},
		{"just minutes", 10 * time.Minute, "10m"},
		{"just hours", 3 * time.Hour, "3h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

// TestStatusUsesOpenCodeAPI verifies that status command uses OpenCode API.
// This is a design test - the actual implementation uses ListSessions() from the API.
func TestStatusUsesOpenCodeAPI(t *testing.T) {
	// The status command uses OpenCode API (ListSessions) for session data.
	// There is no agent registry file - all state comes from OpenCode + beads.
	//
	// The runStatus function:
	// 1. Creates an OpenCode client
	// 2. Calls client.ListSessions()
	// 3. Filters for active sessions
	// 4. Enriches with tmux window info if available
	// 5. Displays results
	//
	// Integration testing requires a running OpenCode server.
}

// TestExtractSkillFromTitle_StatusContext tests skill extraction for status display.
func TestExtractSkillFromTitle_StatusContext(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		wantSkill string
	}{
		{
			name:      "feature-impl from -feat-",
			title:     "og-feat-add-feature-19dec",
			wantSkill: "feature-impl",
		},
		{
			name:      "investigation from -inv-",
			title:     "og-inv-explore-codebase-19dec",
			wantSkill: "investigation",
		},
		{
			name:      "systematic-debugging from -debug-",
			title:     "og-debug-fix-bug-19dec",
			wantSkill: "systematic-debugging",
		},
		{
			name:      "architect from -arch-",
			title:     "og-arch-design-system-19dec",
			wantSkill: "architect",
		},
		{
			name:      "no matching pattern",
			title:     "random-session-name",
			wantSkill: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSkillFromTitle(tt.title)
			if got != tt.wantSkill {
				t.Errorf("extractSkillFromTitle(%q) = %q, want %q", tt.title, got, tt.wantSkill)
			}
		})
	}
}

// TestAbbreviateSkill tests skill name abbreviation for narrow displays.
func TestAbbreviateSkill(t *testing.T) {
	tests := []struct {
		skill    string
		expected string
	}{
		{"feature-impl", "feat"},
		{"investigation", "inv"},
		{"systematic-debugging", "debug"},
		{"architect", "arch"},
		{"codebase-audit", "audit"},
		{"reliability-testing", "rel-test"},
		{"issue-creation", "issue"},
		{"design-session", "design"},
		{"research", "research"},
		{"unknown-skill", "unknown-skill"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			got := abbreviateSkill(tt.skill)
			if got != tt.expected {
				t.Errorf("abbreviateSkill(%q) = %q, want %q", tt.skill, got, tt.expected)
			}
		})
	}
}

// TestGetAgentStatus tests agent status determination.
func TestGetAgentStatus(t *testing.T) {
	tests := []struct {
		name     string
		agent    AgentInfo
		expected string
	}{
		{
			name:     "completed takes precedence",
			agent:    AgentInfo{IsCompleted: true, IsPhantom: true, IsProcessing: true},
			expected: "completed",
		},
		{
			name:     "phantom takes precedence over processing",
			agent:    AgentInfo{IsPhantom: true, IsProcessing: true},
			expected: "phantom",
		},
		{
			name:     "processing/running",
			agent:    AgentInfo{IsProcessing: true},
			expected: "running",
		},
		{
			name:     "default is idle",
			agent:    AgentInfo{},
			expected: "idle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAgentStatus(tt.agent)
			if got != tt.expected {
				t.Errorf("getAgentStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestTerminalWidthConstants verifies the threshold values are sensible.
func TestTerminalWidthConstants(t *testing.T) {
	// Verify the constants are ordered correctly
	if termWidthMin >= termWidthNarrow {
		t.Errorf("termWidthMin (%d) should be less than termWidthNarrow (%d)", termWidthMin, termWidthNarrow)
	}
	if termWidthNarrow >= termWidthWide {
		t.Errorf("termWidthNarrow (%d) should be less than termWidthWide (%d)", termWidthNarrow, termWidthWide)
	}
}

// TestPrintSwarmStatusWithWidth tests output format selection based on width.
// We capture stdout and verify the output format by checking for specific patterns.
func TestPrintSwarmStatusWithWidth(t *testing.T) {
	// Create test data
	testOutput := StatusOutput{
		Swarm: SwarmStatus{
			Active:     2,
			Processing: 1,
			Idle:       1,
			Phantom:    0,
			Completed:  0,
		},
		Agents: []AgentInfo{
			{
				BeadsID:      "orch-go-abcd",
				Skill:        "feature-impl",
				Phase:        "Implementing",
				Task:         "Add terminal width detection to orch status",
				Runtime:      "15m",
				IsProcessing: true,
			},
			{
				BeadsID: "orch-go-efgh",
				Skill:   "investigation",
				Phase:   "Complete",
				Task:    "Investigate API design patterns",
				Runtime: "30m",
			},
		},
	}

	tests := []struct {
		name         string
		width        int
		expectWide   bool // Expect full table with TASK column
		expectNarrow bool // Expect table without TASK column
		expectCard   bool // Expect vertical card format
		expectAbbrev bool // Expect abbreviated skill names
	}{
		{
			name:       "very wide terminal (150 chars)",
			width:      150,
			expectWide: true,
		},
		{
			name:       "wide terminal (exactly 120 chars)",
			width:      120,
			expectWide: true,
		},
		{
			name:         "narrow terminal (90 chars)",
			width:        90,
			expectNarrow: true,
			expectAbbrev: true,
		},
		{
			name:         "minimum narrow (80 chars)",
			width:        80,
			expectNarrow: true,
			expectAbbrev: true,
		},
		{
			name:       "very narrow terminal (70 chars)",
			width:      70,
			expectCard: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printSwarmStatusWithWidth(testOutput, false, tt.width)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check for wide format markers
			hasTaskColumn := strings.Contains(output, "TASK")
			hasTokensColumn := strings.Contains(output, "TOKENS")
			hasSeparator115 := strings.Contains(output, strings.Repeat("-", 115)) // Wide format with TOKENS column
			hasSeparator75 := strings.Contains(output, strings.Repeat("-", 75))   // Narrow format with TOKENS column
			hasAbbreviatedSkill := strings.Contains(output, "feat") && !strings.Contains(output, "feature-impl")
			hasCardFormat := strings.Contains(output, "Phase:") && strings.Contains(output, "Skill:") &&
				strings.Contains(output, "Task:") && strings.Contains(output, "Runtime:")

			if tt.expectWide {
				if !hasTaskColumn {
					t.Errorf("Wide format should have TASK column, got:\n%s", output)
				}
				if !hasTokensColumn {
					t.Errorf("Wide format should have TOKENS column, got:\n%s", output)
				}
				if !hasSeparator115 {
					t.Errorf("Wide format should have 115-char separator, got:\n%s", output)
				}
			}

			if tt.expectNarrow {
				if hasTaskColumn {
					t.Errorf("Narrow format should NOT have TASK column, got:\n%s", output)
				}
				if !hasTokensColumn {
					t.Errorf("Narrow format should have TOKENS column, got:\n%s", output)
				}
				if !hasSeparator75 {
					t.Errorf("Narrow format should have 75-char separator, got:\n%s", output)
				}
			}

			if tt.expectAbbrev {
				if !hasAbbreviatedSkill {
					t.Errorf("Narrow format should use abbreviated skills, got:\n%s", output)
				}
			}

			if tt.expectCard {
				if !hasCardFormat {
					t.Errorf("Card format should have labeled lines (Phase:, Skill:, etc.), got:\n%s", output)
				}
			}
		})
	}
}

// TestFormatTokenCount tests the token count formatter.
func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{12500, "12.5K"},
		{100000, "100.0K"},
		{999999, "1000.0K"}, // Still shows K suffix
		{1000000, "1.0M"},
		{2500000, "2.5M"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatTokenCount(tt.count)
			if got != tt.expected {
				t.Errorf("formatTokenCount(%d) = %q, want %q", tt.count, got, tt.expected)
			}
		})
	}
}

// TestFormatTokenStats tests the full token stats formatter.
func TestFormatTokenStats(t *testing.T) {
	tests := []struct {
		name     string
		tokens   *opencode.TokenStats
		expected string
	}{
		{
			name:     "nil tokens",
			tokens:   nil,
			expected: "-",
		},
		{
			name: "basic tokens",
			tokens: &opencode.TokenStats{
				InputTokens:  8000,
				OutputTokens: 4000,
			},
			expected: "in:8.0K out:4.0K",
		},
		{
			name: "with cache",
			tokens: &opencode.TokenStats{
				InputTokens:     8000,
				OutputTokens:    4000,
				CacheReadTokens: 2000,
			},
			expected: "in:8.0K out:4.0K (cache:2.0K)",
		},
		{
			name: "small counts",
			tokens: &opencode.TokenStats{
				InputTokens:  500,
				OutputTokens: 250,
			},
			expected: "in:500 out:250",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokenStats(tt.tokens)
			if got != tt.expected {
				t.Errorf("formatTokenStats() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestFormatTokenStatsCompact tests the compact token stats formatter.
func TestFormatTokenStatsCompact(t *testing.T) {
	tests := []struct {
		name     string
		tokens   *opencode.TokenStats
		expected string
	}{
		{
			name:     "nil tokens",
			tokens:   nil,
			expected: "-",
		},
		{
			name: "basic tokens with total",
			tokens: &opencode.TokenStats{
				InputTokens:  8000,
				OutputTokens: 4000,
				TotalTokens:  12000,
			},
			expected: "12.0K (8.0K/4.0K)",
		},
		{
			name: "tokens without total field",
			tokens: &opencode.TokenStats{
				InputTokens:  5000,
				OutputTokens: 2500,
			},
			expected: "7.5K (5.0K/2.5K)",
		},
		{
			name: "zero tokens",
			tokens: &opencode.TokenStats{
				InputTokens:  0,
				OutputTokens: 0,
			},
			expected: "-",
		},
		{
			name: "small counts",
			tokens: &opencode.TokenStats{
				InputTokens:  500,
				OutputTokens: 250,
			},
			expected: "750 (500/250)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokenStatsCompact(tt.tokens)
			if got != tt.expected {
				t.Errorf("formatTokenStatsCompact() = %q, want %q", got, tt.expected)
			}
		})
	}
}

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

// mockConn implements the Close() interface for testing.
type mockConn struct{}

func (m *mockConn) Close() error { return nil }

// statusMockError is a simple error type for testing TCP checks.
type statusMockError struct {
	msg string
}

func (e *statusMockError) Error() string { return e.msg }

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

func TestPhaseName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"QUESTION", "QUESTION"},
		{"QUESTION - Should we use JWT?", "QUESTION"},
		{"BLOCKED - Need API key", "BLOCKED"},
		{"Complete - All tests pass", "Complete"},
		{"Implementing - Adding auth middleware", "Implementing"},
		{"Planning", "Planning"},
		{"", ""},
		{"  QUESTION  ", "QUESTION"},
		{"QUESTION - ", "QUESTION"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := phaseName(tt.input)
			if got != tt.want {
				t.Errorf("phaseName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCompactModeShowsQuestionAgents(t *testing.T) {
	// Verify that agents with "QUESTION - detail text" pass the needsAttention filter
	// This tests the fix for the EqualFold bug where detail text prevented matching

	agent := AgentInfo{
		Phase:   "QUESTION - Should we use JWT or session-based auth?",
		BeadsID: "test-123",
	}

	// The needsAttention logic from runStatus
	isComplete := strings.HasPrefix(agent.Phase, "Complete")
	isRecent := true
	needsAttention := (isComplete && isRecent) ||
		strings.EqualFold(phaseName(agent.Phase), "BLOCKED") ||
		strings.EqualFold(phaseName(agent.Phase), "QUESTION")

	if !needsAttention {
		t.Error("Agent with 'QUESTION - detail' phase should pass needsAttention filter")
	}
}

func TestCompactModeShowsBlockedAgents(t *testing.T) {
	agent := AgentInfo{
		Phase:   "BLOCKED - Need clarification on API contract",
		BeadsID: "test-456",
	}

	needsAttention := strings.EqualFold(phaseName(agent.Phase), "BLOCKED") ||
		strings.EqualFold(phaseName(agent.Phase), "QUESTION")

	if !needsAttention {
		t.Error("Agent with 'BLOCKED - detail' phase should pass needsAttention filter")
	}
}

// TestReviewQueueDisplayed tests that review queue count appears between SWARM STATUS and ACCOUNTS.
func TestReviewQueueDisplayed(t *testing.T) {
	testOutput := StatusOutput{
		Swarm: SwarmStatus{
			Active:     1,
			Processing: 1,
		},
		ReviewQueue: &ReviewQueueStatus{Ready: 4},
		Accounts: []AccountUsage{
			{Name: "personal", UsedPercent: 50, IsActive: true},
		},
		Agents: []AgentInfo{
			{
				BeadsID:      "orch-go-abcd",
				Skill:        "feature-impl",
				Phase:        "Implementing",
				Runtime:      "15m",
				IsProcessing: true,
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printSwarmStatusWithWidth(testOutput, false, 150)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify REVIEW QUEUE appears
	if !strings.Contains(output, "REVIEW QUEUE: 4 ready") {
		t.Errorf("Expected 'REVIEW QUEUE: 4 ready' in output, got:\n%s", output)
	}

	// Verify ordering: SWARM STATUS before REVIEW QUEUE before ACCOUNTS
	swarmIdx := strings.Index(output, "SWARM STATUS")
	reviewIdx := strings.Index(output, "REVIEW QUEUE")
	accountsIdx := strings.Index(output, "ACCOUNTS")

	if swarmIdx == -1 || reviewIdx == -1 || accountsIdx == -1 {
		t.Fatalf("Missing expected sections. swarm=%d, review=%d, accounts=%d\nOutput:\n%s",
			swarmIdx, reviewIdx, accountsIdx, output)
	}

	if swarmIdx >= reviewIdx {
		t.Errorf("SWARM STATUS (idx %d) should appear before REVIEW QUEUE (idx %d)", swarmIdx, reviewIdx)
	}
	if reviewIdx >= accountsIdx {
		t.Errorf("REVIEW QUEUE (idx %d) should appear before ACCOUNTS (idx %d)", reviewIdx, accountsIdx)
	}
}

// TestReviewQueueHiddenWhenZero tests that review queue section is omitted when count is zero.
func TestReviewQueueHiddenWhenZero(t *testing.T) {
	testOutput := StatusOutput{
		Swarm: SwarmStatus{Active: 0},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printSwarmStatusWithWidth(testOutput, false, 150)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if strings.Contains(output, "REVIEW QUEUE") {
		t.Errorf("REVIEW QUEUE should not appear when nil, got:\n%s", output)
	}
}
