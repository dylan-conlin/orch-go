package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckClass2_OpenCodeOnly(t *testing.T) {
	fn := funcInfo{
		name:            "countSessions",
		startLine:       10,
		hasOpenCodeCall: true,
		hasTmuxCall:     false,
		hasDualPattern:  false,
		isAgentRelated:  true,
	}

	findings := checkClass2(fn, "cmd/orch/focus.go")
	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].Class != 2 {
		t.Errorf("Expected class 2, got %d", findings[0].Class)
	}
	if !strings.Contains(findings[0].Description, "OpenCode query without tmux") {
		t.Errorf("Unexpected description: %s", findings[0].Description)
	}
}

func TestCheckClass2_TmuxOnly(t *testing.T) {
	fn := funcInfo{
		name:            "cleanupWindows",
		startLine:       20,
		hasOpenCodeCall: false,
		hasTmuxCall:     true,
		hasDualPattern:  false,
		isAgentRelated:  true,
	}

	findings := checkClass2(fn, "cmd/orch/shared.go")
	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}
	if !strings.Contains(findings[0].Description, "Tmux check without OpenCode") {
		t.Errorf("Unexpected description: %s", findings[0].Description)
	}
}

func TestCheckClass2_BothBackends(t *testing.T) {
	fn := funcInfo{
		name:            "discoverAgents",
		startLine:       30,
		hasOpenCodeCall: true,
		hasTmuxCall:     true,
		hasDualPattern:  false,
		isAgentRelated:  true,
	}

	findings := checkClass2(fn, "cmd/orch/status.go")
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for dual-backend function, got %d", len(findings))
	}
}

func TestCheckClass2_DualPattern(t *testing.T) {
	fn := funcInfo{
		name:            "getActiveCount",
		startLine:       40,
		hasOpenCodeCall: true,
		hasTmuxCall:     false,
		hasDualPattern:  true, // calls DiscoverLiveAgents
		isAgentRelated:  true,
	}

	findings := checkClass2(fn, "cmd/orch/daemon.go")
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for function using dual pattern, got %d", len(findings))
	}
}

func TestCheckClass2_Allowlist(t *testing.T) {
	fn := funcInfo{
		name:            "DiscoverLiveAgents",
		startLine:       50,
		hasOpenCodeCall: true,
		hasTmuxCall:     false,
		hasDualPattern:  false,
		isAgentRelated:  true,
	}

	findings := checkClass2(fn, "pkg/daemon/active_count.go")
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for allowlisted function, got %d", len(findings))
	}
}

func TestCheckClass2_HighSeverityForDiscovery(t *testing.T) {
	fn := funcInfo{
		name:            "ListActiveSessions",
		startLine:       10,
		hasOpenCodeCall: true,
		hasTmuxCall:     false,
		isAgentRelated:  true,
	}

	findings := checkClass2(fn, "cmd/orch/serve.go")
	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != "high" {
		t.Errorf("Expected high severity for discovery function, got %s", findings[0].Severity)
	}
}

func TestCheckClass5_ThreeSignals(t *testing.T) {
	fn := funcInfo{
		name:             "checkCompletion",
		startLine:        100,
		hasPhaseCheck:    true,
		hasSynthesisCheck: true,
		hasSessionStatus: true,
		isAgentRelated:   true,
	}

	findings := checkClass5(fn, "cmd/orch/complete.go")
	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].Class != 5 {
		t.Errorf("Expected class 5, got %d", findings[0].Class)
	}
	if findings[0].Severity != "high" {
		t.Errorf("Expected high severity for 3+ signals, got %s", findings[0].Severity)
	}
	if !strings.Contains(findings[0].Description, "3 authority signals") {
		t.Errorf("Unexpected description: %s", findings[0].Description)
	}
}

func TestCheckClass5_TwoSignals(t *testing.T) {
	fn := funcInfo{
		name:          "isAgentDone",
		startLine:     200,
		hasPhaseCheck: true,
		hasBeadsStatus: true,
		isAgentRelated: true,
	}

	findings := checkClass5(fn, "cmd/orch/status.go")
	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != "low" {
		t.Errorf("Expected low severity for 2 signals in non-status function, got %s", findings[0].Severity)
	}
}

func TestCheckClass5_TwoSignalsStatusFunction(t *testing.T) {
	fn := funcInfo{
		name:            "DetermineStatus",
		startLine:       200,
		hasPhaseCheck:   true,
		hasSessionStatus: true,
		isAgentRelated:  true,
	}

	findings := checkClass5(fn, "cmd/orch/status.go")
	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != "medium" {
		t.Errorf("Expected medium severity for status-determining function, got %s", findings[0].Severity)
	}
}

func TestCheckClass5_SingleSignal(t *testing.T) {
	fn := funcInfo{
		name:          "checkPhase",
		startLine:     300,
		hasPhaseCheck: true,
		isAgentRelated: true,
	}

	findings := checkClass5(fn, "cmd/orch/verify.go")
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for single-signal function, got %d", len(findings))
	}
}

func TestCheckClass5_Allowlist(t *testing.T) {
	fn := funcInfo{
		name:              "determineAgentStatus",
		startLine:         50,
		hasPhaseCheck:     true,
		hasSynthesisCheck: true,
		hasSessionStatus:  true,
		hasBeadsStatus:    true,
		isAgentRelated:    true,
	}

	findings := checkClass5(fn, "cmd/orch/serve_agents_status.go")
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for allowlisted function, got %d", len(findings))
	}
}

func TestCheckClass2_NotAgentRelated(t *testing.T) {
	fn := funcInfo{
		name:            "buildConfig",
		startLine:       10,
		hasOpenCodeCall: true,
		hasTmuxCall:     false,
		isAgentRelated:  false, // not agent related
	}

	// scanFile skips non-agent-related functions, so checkClass2 would be called
	// but the caller filters. Let's verify the function still reports if called directly.
	findings := checkClass2(fn, "cmd/orch/config.go")
	if len(findings) != 1 {
		t.Fatalf("checkClass2 should still report (filtering is done by caller), got %d", len(findings))
	}
}

func TestParseFunctions(t *testing.T) {
	content := `package main

func simpleFunc() {
	client.ListSessions("")
}

func dualFunc() {
	client.ListSessions("")
	tmux.ListWindows("workers")
}

func statusFunc() {
	IsPhaseComplete()
	checkWorkspaceSynthesis()
	sessionStatus = "idle"
}
`
	// Write to temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	funcs := parseFunctions(f)

	if len(funcs) != 3 {
		t.Fatalf("Expected 3 functions, got %d", len(funcs))
	}

	// simpleFunc: OpenCode only
	if funcs[0].name != "simpleFunc" {
		t.Errorf("Expected simpleFunc, got %s", funcs[0].name)
	}
	if !funcs[0].hasOpenCodeCall {
		t.Error("simpleFunc should have OpenCode call")
	}
	if funcs[0].hasTmuxCall {
		t.Error("simpleFunc should not have tmux call")
	}

	// dualFunc: Both backends
	if funcs[1].name != "dualFunc" {
		t.Errorf("Expected dualFunc, got %s", funcs[1].name)
	}
	if !funcs[1].hasOpenCodeCall {
		t.Error("dualFunc should have OpenCode call")
	}
	if !funcs[1].hasTmuxCall {
		t.Error("dualFunc should have tmux call")
	}

	// statusFunc: Multiple authority signals
	if funcs[2].name != "statusFunc" {
		t.Errorf("Expected statusFunc, got %s", funcs[2].name)
	}
	if !funcs[2].hasPhaseCheck {
		t.Error("statusFunc should have phase check")
	}
	if !funcs[2].hasSynthesisCheck {
		t.Error("statusFunc should have synthesis check")
	}
	if !funcs[2].hasSessionStatus {
		t.Error("statusFunc should have session status")
	}
}

func TestScanForDefects_Integration(t *testing.T) {
	// Create a temp directory with test Go files
	tmpDir := t.TempDir()
	cmdDir := filepath.Join(tmpDir, "cmd", "orch")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a file with Class 2 and Class 5 patterns
	testFile := `package main

func countActiveSessions() {
	client.ListSessions("")
	// No tmux check - Class 2 blind spot
}

func checkAgentStatus() {
	IsPhaseComplete()
	checkWorkspaceSynthesis()
	sessionStatus := getSessionStatus()
	_ = sessionStatus
	// 3 authority signals - Class 5
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "test_scan.go"), []byte(testFile), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := scanForDefects(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if report.FilesScanned != 1 {
		t.Errorf("Expected 1 file scanned, got %d", report.FilesScanned)
	}

	// Should find at least one Class 2 and one Class 5
	if report.Class2Count == 0 {
		t.Error("Expected at least 1 Class 2 finding")
	}
	if report.Class5Count == 0 {
		t.Error("Expected at least 1 Class 5 finding")
	}
}

func TestDefectFindingFields(t *testing.T) {
	f := DefectFinding{
		Class:       2,
		ClassName:   "Multi-Backend Blindness",
		File:        "cmd/orch/focus.go",
		Function:    "countSessions",
		Line:        42,
		Description: "OpenCode query without tmux check",
		Severity:    "high",
	}

	if f.Class != 2 {
		t.Error("Class field")
	}
	if f.ClassName != "Multi-Backend Blindness" {
		t.Error("ClassName field")
	}
	if f.Severity != "high" {
		t.Error("Severity field")
	}
}

func TestSeverityIcon(t *testing.T) {
	if severityIcon("high") != "🔴" {
		t.Error("high severity icon")
	}
	if severityIcon("medium") != "🟡" {
		t.Error("medium severity icon")
	}
	if severityIcon("low") != "🔵" {
		t.Error("low severity icon")
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny("ListActiveSessions", "List", "Count") {
		t.Error("should match List")
	}
	if containsAny("buildConfig", "List", "Count") {
		t.Error("should not match")
	}
}
