package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAuditVerdict_FailHighConfidence(t *testing.T) {
	content := `verdict: FAIL
original_issue: orch-go-abc12
category: quality
confidence: high
reason: Tests don't cover the primary edge case in token refresh
evidence: pkg/auth/refresh.go:45 — no test for expired+revoked combo
`
	verdict, err := ParseAuditVerdict([]byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != "FAIL" {
		t.Errorf("verdict = %q, want FAIL", verdict.Verdict)
	}
	if verdict.OriginalIssue != "orch-go-abc12" {
		t.Errorf("original_issue = %q, want orch-go-abc12", verdict.OriginalIssue)
	}
	if verdict.Category != "quality" {
		t.Errorf("category = %q, want quality", verdict.Category)
	}
	if verdict.Confidence != "high" {
		t.Errorf("confidence = %q, want high", verdict.Confidence)
	}
	if verdict.Reason == "" {
		t.Error("reason should not be empty")
	}
}

func TestParseAuditVerdict_Pass(t *testing.T) {
	content := `verdict: PASS
original_issue: orch-go-xyz99
confidence: high
reason: Implementation matches intent, tests cover main paths
`
	verdict, err := ParseAuditVerdict([]byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != "PASS" {
		t.Errorf("verdict = %q, want PASS", verdict.Verdict)
	}
	if verdict.OriginalIssue != "orch-go-xyz99" {
		t.Errorf("original_issue = %q, want orch-go-xyz99", verdict.OriginalIssue)
	}
}

func TestParseAuditVerdict_MissingVerdict(t *testing.T) {
	content := `original_issue: orch-go-abc12
confidence: high
reason: no verdict field
`
	_, err := ParseAuditVerdict([]byte(content))
	if err == nil {
		t.Error("expected error for missing verdict field")
	}
}

func TestParseAuditVerdict_MissingOriginalIssue(t *testing.T) {
	content := `verdict: FAIL
confidence: high
reason: missing original issue
`
	_, err := ParseAuditVerdict([]byte(content))
	if err == nil {
		t.Error("expected error for missing original_issue field")
	}
}

func TestParseAuditVerdict_InvalidVerdict(t *testing.T) {
	content := `verdict: MAYBE
original_issue: orch-go-abc12
confidence: high
reason: invalid verdict value
`
	_, err := ParseAuditVerdict([]byte(content))
	if err == nil {
		t.Error("expected error for invalid verdict value")
	}
}

func TestReadAuditVerdictFromWorkspace(t *testing.T) {
	dir := t.TempDir()
	content := `verdict: FAIL
original_issue: orch-go-test1
category: scope
confidence: medium
reason: Work addresses wrong scope
`
	if err := os.WriteFile(filepath.Join(dir, "AUDIT_VERDICT.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	verdict, err := ReadAuditVerdictFromWorkspace(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != "FAIL" {
		t.Errorf("verdict = %q, want FAIL", verdict.Verdict)
	}
	if verdict.OriginalIssue != "orch-go-test1" {
		t.Errorf("original_issue = %q, want orch-go-test1", verdict.OriginalIssue)
	}
}

func TestReadAuditVerdictFromWorkspace_NoFile(t *testing.T) {
	dir := t.TempDir()
	verdict, err := ReadAuditVerdictFromWorkspace(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != nil {
		t.Errorf("expected nil verdict when no AUDIT_VERDICT.md exists")
	}
}

func TestProcessAuditVerdict_FailHighConfidence(t *testing.T) {
	rejected := false
	rejectedID := ""
	rejectedReason := ""
	rejectedCategory := ""

	d := &Daemon{
		Rejector: &mockRejector{
			RejectFunc: func(beadsID, reason, category, workdir string) error {
				rejected = true
				rejectedID = beadsID
				rejectedReason = reason
				rejectedCategory = category
				return nil
			},
		},
	}

	verdict := &AuditVerdict{
		Verdict:       "FAIL",
		OriginalIssue: "orch-go-abc12",
		Category:      "quality",
		Confidence:    "high",
		Reason:        "Tests missing edge case",
	}

	result := d.ProcessAuditVerdict(verdict, "orch-go-audit1", "")
	if !result.Rejected {
		t.Error("expected Rejected=true for FAIL+high confidence")
	}
	if !rejected {
		t.Error("Rejector.Reject should have been called")
	}
	if rejectedID != "orch-go-abc12" {
		t.Errorf("rejected ID = %q, want orch-go-abc12", rejectedID)
	}
	if rejectedReason == "" {
		t.Error("rejected reason should not be empty")
	}
	if rejectedCategory != "quality" {
		t.Errorf("rejected category = %q, want quality", rejectedCategory)
	}
}

func TestProcessAuditVerdict_FailLowConfidence(t *testing.T) {
	rejected := false
	labeled := false
	labeledID := ""
	labeledLabel := ""

	d := &Daemon{
		Rejector: &mockRejector{
			RejectFunc: func(beadsID, reason, category, workdir string) error {
				rejected = true
				return nil
			},
		},
		AuditLabeler: &mockAuditLabeler{
			AddLabelFunc: func(beadsID, label, workdir string) error {
				labeled = true
				labeledID = beadsID
				labeledLabel = label
				return nil
			},
		},
	}

	verdict := &AuditVerdict{
		Verdict:       "FAIL",
		OriginalIssue: "orch-go-abc12",
		Category:      "quality",
		Confidence:    "low",
		Reason:        "Might be missing tests, unclear",
	}

	result := d.ProcessAuditVerdict(verdict, "orch-go-audit1", "")
	if result.Rejected {
		t.Error("expected Rejected=false for FAIL+low confidence")
	}
	if rejected {
		t.Error("Rejector.Reject should NOT be called for low confidence")
	}
	if !labeled {
		t.Error("AuditLabeler should have been called for low confidence")
	}
	if labeledID != "orch-go-abc12" {
		t.Errorf("labeled ID = %q, want orch-go-abc12", labeledID)
	}
	if labeledLabel != "audit:needs-review" {
		t.Errorf("label = %q, want audit:needs-review", labeledLabel)
	}
	if !result.NeedsReview {
		t.Error("expected NeedsReview=true for low confidence")
	}
}

func TestProcessAuditVerdict_Pass(t *testing.T) {
	rejected := false
	removedLabel := ""

	d := &Daemon{
		Rejector: &mockRejector{
			RejectFunc: func(beadsID, reason, category, workdir string) error {
				rejected = true
				return nil
			},
		},
		AuditLabeler: &mockAuditLabeler{
			RemoveLabelFunc: func(beadsID, label, workdir string) error {
				removedLabel = label
				return nil
			},
		},
	}

	verdict := &AuditVerdict{
		Verdict:       "PASS",
		OriginalIssue: "orch-go-abc12",
		Confidence:    "high",
		Reason:        "All good",
	}

	result := d.ProcessAuditVerdict(verdict, "orch-go-audit1", "")
	if result.Rejected {
		t.Error("expected Rejected=false for PASS verdict")
	}
	if rejected {
		t.Error("Rejector.Reject should NOT be called for PASS")
	}
	if removedLabel != "audit:deep-review" {
		t.Errorf("removed label = %q, want audit:deep-review", removedLabel)
	}
	if !result.Passed {
		t.Error("expected Passed=true for PASS verdict")
	}
}

func TestProcessAuditVerdict_NoRejector(t *testing.T) {
	d := &Daemon{}

	verdict := &AuditVerdict{
		Verdict:       "FAIL",
		OriginalIssue: "orch-go-abc12",
		Category:      "quality",
		Confidence:    "high",
		Reason:        "Bad work",
	}

	result := d.ProcessAuditVerdict(verdict, "orch-go-audit1", "")
	if result.Rejected {
		t.Error("expected Rejected=false when no Rejector is set")
	}
	if result.Error == nil {
		t.Error("expected error when no Rejector is set")
	}
}

func TestProcessAuditVerdictIfPresent_WithVerdictFile(t *testing.T) {
	// Set up a workspace with AUDIT_VERDICT.md
	dir := t.TempDir()
	content := `verdict: FAIL
original_issue: orch-go-original1
category: scope
confidence: high
reason: Work addresses wrong requirements
`
	if err := os.WriteFile(filepath.Join(dir, "AUDIT_VERDICT.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rejected := false
	rejectedID := ""

	d := &Daemon{
		Rejector: &mockRejector{
			RejectFunc: func(beadsID, reason, category, workdir string) error {
				rejected = true
				rejectedID = beadsID
				return nil
			},
		},
	}

	agent := CompletedAgent{
		BeadsID:       "orch-go-audit99",
		WorkspacePath: dir,
	}
	config := CompletionConfig{ProjectDir: "/tmp/test"}

	d.processAuditVerdictIfPresent(agent, config, nil)

	if !rejected {
		t.Error("expected Rejector.Reject to be called for FAIL verdict")
	}
	if rejectedID != "orch-go-original1" {
		t.Errorf("rejected ID = %q, want orch-go-original1", rejectedID)
	}
}

func TestProcessAuditVerdictIfPresent_NoVerdictFile(t *testing.T) {
	dir := t.TempDir()
	rejected := false

	d := &Daemon{
		Rejector: &mockRejector{
			RejectFunc: func(beadsID, reason, category, workdir string) error {
				rejected = true
				return nil
			},
		},
	}

	agent := CompletedAgent{
		BeadsID:       "orch-go-normal1",
		WorkspacePath: dir,
	}
	config := CompletionConfig{ProjectDir: "/tmp/test"}

	d.processAuditVerdictIfPresent(agent, config, nil)

	if rejected {
		t.Error("Rejector should NOT be called when no AUDIT_VERDICT.md exists")
	}
}

func TestProcessAuditVerdictIfPresent_NoWorkspace(t *testing.T) {
	rejected := false

	d := &Daemon{
		Rejector: &mockRejector{
			RejectFunc: func(beadsID, reason, category, workdir string) error {
				rejected = true
				return nil
			},
		},
	}

	agent := CompletedAgent{
		BeadsID:       "orch-go-noworkspace",
		WorkspacePath: "", // no workspace
	}
	config := CompletionConfig{ProjectDir: "/tmp/test"}

	d.processAuditVerdictIfPresent(agent, config, nil)

	if rejected {
		t.Error("Rejector should NOT be called when no workspace path")
	}
}

// --- Mock types ---

type mockRejector struct {
	RejectFunc func(beadsID, reason, category, workdir string) error
}

func (m *mockRejector) Reject(beadsID, reason, category, workdir string) error {
	return m.RejectFunc(beadsID, reason, category, workdir)
}

type mockAuditLabeler struct {
	AddLabelFunc    func(beadsID, label, workdir string) error
	RemoveLabelFunc func(beadsID, label, workdir string) error
}

func (m *mockAuditLabeler) AddLabel(beadsID, label, workdir string) error {
	if m.AddLabelFunc != nil {
		return m.AddLabelFunc(beadsID, label, workdir)
	}
	return nil
}

func (m *mockAuditLabeler) RemoveLabel(beadsID, label, workdir string) error {
	if m.RemoveLabelFunc != nil {
		return m.RemoveLabelFunc(beadsID, label, workdir)
	}
	return nil
}
