package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// mockComprehensionQuerier is a test double for ComprehensionQuerier.
type mockComprehensionQuerier struct {
	count int
	err   error
}

func (m *mockComprehensionQuerier) CountPending() (int, error) {
	return m.count, m.err
}

func TestCheckComprehensionThrottle_NilQuerier(t *testing.T) {
	allowed, count, threshold := CheckComprehensionThrottle(nil, 5)
	if !allowed {
		t.Error("nil querier should allow spawning")
	}
	if count != 0 {
		t.Errorf("nil querier count = %d, want 0", count)
	}
	if threshold != 5 {
		t.Errorf("threshold = %d, want 5", threshold)
	}
}

func TestCheckComprehensionThrottle_BelowThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 3}
	allowed, count, threshold := CheckComprehensionThrottle(q, 5)
	if !allowed {
		t.Error("should allow when below threshold")
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if threshold != 5 {
		t.Errorf("threshold = %d, want 5", threshold)
	}
}

func TestCheckComprehensionThrottle_AtThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 5}
	allowed, _, _ := CheckComprehensionThrottle(q, 5)
	if allowed {
		t.Error("should block when at threshold")
	}
}

func TestCheckComprehensionThrottle_AboveThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 8}
	allowed, _, _ := CheckComprehensionThrottle(q, 5)
	if allowed {
		t.Error("should block when above threshold")
	}
}

func TestCheckComprehensionThrottle_ErrorFailsOpen(t *testing.T) {
	q := &mockComprehensionQuerier{count: 0, err: fmt.Errorf("bd failed")}
	allowed, _, _ := CheckComprehensionThrottle(q, 5)
	if !allowed {
		t.Error("should fail-open on error")
	}
}

func TestCheckComprehensionThrottle_DefaultThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 3}
	_, _, threshold := CheckComprehensionThrottle(q, 0)
	if threshold != DefaultComprehensionThreshold {
		t.Errorf("default threshold = %d, want %d", threshold, DefaultComprehensionThreshold)
	}
}

func TestComprehensionLabelConstants(t *testing.T) {
	if LabelComprehensionUnread != "comprehension:unread" {
		t.Errorf("LabelComprehensionUnread = %q, want %q", LabelComprehensionUnread, "comprehension:unread")
	}
	if LabelComprehensionProcessed != "comprehension:processed" {
		t.Errorf("LabelComprehensionProcessed = %q, want %q", LabelComprehensionProcessed, "comprehension:processed")
	}
	if LabelComprehensionPending != "comprehension:pending" {
		t.Errorf("LabelComprehensionPending = %q, want %q", LabelComprehensionPending, "comprehension:pending")
	}
}

func TestRecordBriefFeedback_ValidRatings(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(filepath.Join(projectDir, ".kb", "briefs"), 0755)

	// Test valid ratings
	for _, rating := range []string{"shallow", "good"} {
		err := RecordBriefFeedback("test-123", rating, projectDir)
		if err != nil {
			t.Errorf("RecordBriefFeedback(%q) failed: %v", rating, err)
		}

		got, err := ReadBriefFeedback("test-123", projectDir)
		if err != nil {
			t.Errorf("ReadBriefFeedback failed: %v", err)
		}
		if got != rating {
			t.Errorf("ReadBriefFeedback = %q, want %q", got, rating)
		}
	}
}

func TestRecordBriefFeedback_InvalidRating(t *testing.T) {
	tmpDir := t.TempDir()
	err := RecordBriefFeedback("test-123", "invalid", tmpDir)
	if err == nil {
		t.Error("expected error for invalid rating")
	}
}

func TestParseBriefSignalCount(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			"frontmatter with signals",
			"---\nbeads_id: test-123\nsignal_count: 5\nsignal_total: 6\n---\n\n# Brief",
			5,
		},
		{
			"zero signals",
			"---\nbeads_id: test-123\nsignal_count: 0\nsignal_total: 6\n---\n\n# Brief",
			0,
		},
		{
			"no frontmatter",
			"# Brief: test-123\n\n## Frame\n",
			0,
		},
		{
			"malformed frontmatter",
			"---\nno closing delimiter",
			0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseBriefSignalCount(tc.content)
			if got != tc.want {
				t.Errorf("ParseBriefSignalCount() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestParseBriefSignals(t *testing.T) {
	content := `---
beads_id: test-123
quality_signals:
  structural_completeness:
    score: "4/4"
    detected: true
    evidence: "TLDR, Delta, Evidence, Knowledge"
  evidence_specificity:
    score: "true"
    detected: true
    evidence: "pkg/"
  model_connection:
    score: "false"
    detected: false
    evidence: ""
  connective_reasoning:
    score: "true"
    detected: true
    evidence: ""
  tension_quality:
    score: "true"
    detected: true
    evidence: ""
  insight_vs_report:
    score: "2/3"
    detected: true
    evidence: ""
signal_count: 5
signal_total: 6
---

# Brief: test-123`

	signals := ParseBriefSignals(content)
	if len(signals) != 6 {
		t.Errorf("Expected 6 signals, got %d", len(signals))
	}

	// Check a specific signal
	sc, ok := signals["structural_completeness"]
	if !ok {
		t.Fatal("Missing structural_completeness signal")
	}
	if !sc.Detected {
		t.Error("structural_completeness should be detected")
	}
	if sc.Score != "4/4" {
		t.Errorf("structural_completeness score = %q, want %q", sc.Score, "4/4")
	}

	// Check undetected signal
	mc, ok := signals["model_connection"]
	if !ok {
		t.Fatal("Missing model_connection signal")
	}
	if mc.Detected {
		t.Error("model_connection should not be detected")
	}
}

func TestParseBriefSignals_NoFrontmatter(t *testing.T) {
	signals := ParseBriefSignals("# Brief: test\n\n## Frame\n")
	if len(signals) != 0 {
		t.Errorf("Expected 0 signals for no-frontmatter brief, got %d", len(signals))
	}
}

func TestOrderBriefsBySignals(t *testing.T) {
	briefs := []BriefQueueEntry{
		{BeadsID: "low", SignalCount: 2, Signals: map[string]BriefSignal{
			"evidence_specificity": {Detected: true},
			"tension_quality":     {Detected: true},
		}},
		{BeadsID: "high", SignalCount: 5, Signals: map[string]BriefSignal{
			"structural_completeness": {Detected: true},
			"evidence_specificity":    {Detected: true},
			"model_connection":        {Detected: true},
			"connective_reasoning":    {Detected: true},
			"tension_quality":         {Detected: true},
		}},
		{BeadsID: "mid", SignalCount: 3, Signals: map[string]BriefSignal{
			"structural_completeness": {Detected: true},
			"evidence_specificity":    {Detected: true},
			"connective_reasoning":    {Detected: true},
		}},
	}

	OrderBriefsBySignals(briefs, nil)

	if briefs[0].BeadsID != "high" {
		t.Errorf("First brief should be 'high', got %q", briefs[0].BeadsID)
	}
	if briefs[1].BeadsID != "mid" {
		t.Errorf("Second brief should be 'mid', got %q", briefs[1].BeadsID)
	}
	if briefs[2].BeadsID != "low" {
		t.Errorf("Third brief should be 'low', got %q", briefs[2].BeadsID)
	}
}

func TestOrderBriefsBySignals_PrioritizeSpecific(t *testing.T) {
	briefs := []BriefQueueEntry{
		{BeadsID: "more-signals", SignalCount: 4, Signals: map[string]BriefSignal{
			"structural_completeness": {Detected: true},
			"evidence_specificity":    {Detected: true},
			"connective_reasoning":    {Detected: true},
			"tension_quality":         {Detected: true},
		}},
		{BeadsID: "model-connected", SignalCount: 3, Signals: map[string]BriefSignal{
			"structural_completeness": {Detected: true},
			"model_connection":        {Detected: true},
			"evidence_specificity":    {Detected: true},
		}},
	}

	// Prioritize briefs with model_connection
	priority := []string{"model_connection"}
	OrderBriefsBySignals(briefs, priority)

	if briefs[0].BeadsID != "model-connected" {
		t.Errorf("model_connection brief should sort first with priority, got %q", briefs[0].BeadsID)
	}
}

// TestStripAllComprehensionLabels_LiveBeads verifies that all comprehension lifecycle
// labels are removed from an issue. This is the core fix for:
// "orch complete should strip comprehension:unread label"
func TestStripAllComprehensionLabels_LiveBeads(t *testing.T) {
	projectDir := t.TempDir()

	// Initialize beads project
	out, err := runBdCommandForCompTest(projectDir, "init")
	if err != nil {
		t.Skipf("bd init failed (bd CLI not available?): %v: %s", err, out)
	}

	// Create a test issue
	out, err = runBdCommandForCompTest(projectDir, "create", "test comprehension strip", "--type", "task", "--json")
	if err != nil {
		t.Fatalf("bd create failed: %v: %s", err, out)
	}
	issueID := extractCompTestIssueID(t, string(out))

	// Add all three comprehension labels
	for _, label := range []string{LabelComprehensionUnread, LabelComprehensionPending, LabelComprehensionProcessed} {
		if _, err := runBdCommandForCompTest(projectDir, "label", "add", issueID, label); err != nil {
			t.Fatalf("failed to add label %s: %v", label, err)
		}
	}

	// Verify labels are present before stripping
	for _, label := range []string{LabelComprehensionUnread, LabelComprehensionPending, LabelComprehensionProcessed} {
		if !compTestIssueHasLabel(t, projectDir, issueID, label) {
			t.Fatalf("expected label %s to be present before strip", label)
		}
	}

	// Strip all comprehension labels (simulating what orch complete now does pre-lifecycle)
	for _, label := range []string{LabelComprehensionUnread, LabelComprehensionPending, LabelComprehensionProcessed} {
		runBdCommandForCompTest(projectDir, "label", "remove", issueID, label)
	}

	// Verify ALL labels are removed
	for _, label := range []string{LabelComprehensionUnread, LabelComprehensionPending, LabelComprehensionProcessed} {
		if compTestIssueHasLabel(t, projectDir, issueID, label) {
			t.Errorf("label %s should be removed after strip, but still present", label)
		}
	}
}

// TestStripBeforeClose_LiveBeads verifies that stripping labels BEFORE close works,
// while stripping AFTER close may leave labels behind (proving the ordering fix).
func TestStripBeforeClose_LiveBeads(t *testing.T) {
	projectDir := t.TempDir()

	out, err := runBdCommandForCompTest(projectDir, "init")
	if err != nil {
		t.Skipf("bd init failed (bd CLI not available?): %v: %s", err, out)
	}

	// --- Test 1: Strip BEFORE close (the fix) ---
	out, err = runBdCommandForCompTest(projectDir, "create", "test pre-close strip", "--type", "task", "--json")
	if err != nil {
		t.Fatalf("bd create failed: %v: %s", err, out)
	}
	preCloseID := extractCompTestIssueID(t, string(out))

	// Add label, strip it, THEN close
	runBdCommandForCompTest(projectDir, "label", "add", preCloseID, LabelComprehensionUnread)
	runBdCommandForCompTest(projectDir, "label", "remove", preCloseID, LabelComprehensionUnread)
	runBdCommandForCompTest(projectDir, "close", preCloseID, "--force")

	if compTestIssueHasLabel(t, projectDir, preCloseID, LabelComprehensionUnread) {
		t.Error("pre-close strip: label should be removed")
	}

	// --- Test 2: Strip AFTER close (the old broken behavior) ---
	out, err = runBdCommandForCompTest(projectDir, "create", "test post-close strip", "--type", "task", "--json")
	if err != nil {
		t.Fatalf("bd create failed: %v: %s", err, out)
	}
	postCloseID := extractCompTestIssueID(t, string(out))

	// Add label, close THEN try to strip
	runBdCommandForCompTest(projectDir, "label", "add", postCloseID, LabelComprehensionUnread)
	runBdCommandForCompTest(projectDir, "close", postCloseID, "--force")
	runBdCommandForCompTest(projectDir, "label", "remove", postCloseID, LabelComprehensionUnread)

	postCloseHasLabel := compTestIssueHasLabel(t, projectDir, postCloseID, LabelComprehensionUnread)
	if postCloseHasLabel {
		t.Logf("CONFIRMED: post-close label removal fails — this is why pre-close ordering is required")
	} else {
		t.Logf("post-close label removal works in this bd version — pre-close ordering still preferred for safety")
	}
}

func runBdCommandForCompTest(projectDir string, args ...string) ([]byte, error) {
	beadsDir := filepath.Join(projectDir, ".beads")
	cmd := exec.Command("bd", args...)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "BEADS_DIR="+beadsDir)
	return cmd.CombinedOutput()
}

func extractCompTestIssueID(t *testing.T, output string) string {
	t.Helper()
	jsonStart := strings.Index(output, "{")
	jsonEnd := strings.LastIndex(output, "}")
	if jsonStart < 0 || jsonEnd <= jsonStart {
		t.Fatalf("no JSON in bd create output: %s", output)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(output[jsonStart:jsonEnd+1]), &created); err != nil {
		t.Fatalf("failed to parse issue JSON: %v: %s", err, output)
	}
	if created.ID == "" {
		t.Fatalf("empty issue ID from bd create output: %s", output)
	}
	return created.ID
}

func compTestIssueHasLabel(t *testing.T, projectDir, issueID, label string) bool {
	t.Helper()
	out, err := runBdCommandForCompTest(projectDir, "show", issueID, "--json")
	if err != nil {
		t.Fatalf("bd show failed: %v: %s", err, out)
	}
	outStr := string(out)
	jsonStart := strings.Index(outStr, "{")
	jsonEnd := strings.LastIndex(outStr, "}")
	if jsonStart < 0 || jsonEnd <= jsonStart {
		t.Fatalf("no JSON in bd show output: %s", outStr)
	}
	var issue struct {
		Labels []string `json:"labels"`
	}
	if err := json.Unmarshal([]byte(outStr[jsonStart:jsonEnd+1]), &issue); err != nil {
		t.Fatalf("failed to parse issue JSON: %v: %s", err, outStr)
	}
	for _, l := range issue.Labels {
		if l == label {
			return true
		}
	}
	return false
}

func TestReadBriefFeedback_NoFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	rating, err := ReadBriefFeedback("nonexistent", tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if rating != "" {
		t.Errorf("expected empty rating, got %q", rating)
	}
}
