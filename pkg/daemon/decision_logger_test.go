package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestLogDecision(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")
	logger := events.NewLogger(logPath)

	err := LogDecision(logger, DecisionLogEntry{
		Class:      daemonconfig.DecisionSelectIssue,
		Compliance: daemonconfig.ComplianceStandard,
		Target:     "orch-go-abc12",
		Reason:     "highest priority in queue",
	})
	if err != nil {
		t.Fatalf("LogDecision() error = %v", err)
	}

	// Read and verify the event
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.Type != events.EventTypeDecisionMade {
		t.Errorf("Type = %q, want %q", event.Type, events.EventTypeDecisionMade)
	}
	if got := event.Data["class"]; got != "select_issue" {
		t.Errorf("class = %v, want %q", got, "select_issue")
	}
	if got := event.Data["category"]; got != "spawn" {
		t.Errorf("category = %v, want %q", got, "spawn")
	}
	if got := event.Data["tier"]; got != "autonomous" {
		t.Errorf("tier = %v, want %q", got, "autonomous")
	}
	if got := event.Data["base_tier"]; got != "autonomous" {
		t.Errorf("base_tier = %v, want %q", got, "autonomous")
	}
	if got := event.Data["compliance_level"]; got != "standard" {
		t.Errorf("compliance_level = %v, want %q", got, "standard")
	}
	if got := event.Data["target"]; got != "orch-go-abc12" {
		t.Errorf("target = %v, want %q", got, "orch-go-abc12")
	}
	if got := event.Data["reason"]; got != "highest priority in queue" {
		t.Errorf("reason = %v, want %q", got, "highest priority in queue")
	}
	// decision_id should be present and non-empty
	if got, ok := event.Data["decision_id"].(string); !ok || got == "" {
		t.Errorf("decision_id should be a non-empty string, got %v", event.Data["decision_id"])
	}
}

func TestLogDecision_ComplianceModulatesTier(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")
	logger := events.NewLogger(logPath)

	// AutoCompleteLight is base Tier 2. Under strict, should be Tier 3.
	err := LogDecision(logger, DecisionLogEntry{
		Class:      daemonconfig.DecisionAutoCompleteLight,
		Compliance: daemonconfig.ComplianceStrict,
		Target:     "orch-go-xyz99",
		Reason:     "effort:small agent",
	})
	if err != nil {
		t.Fatalf("LogDecision() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if got := event.Data["tier"]; got != "genuine-decision" {
		t.Errorf("tier = %v, want %q (strict promotes T2 to T3)", got, "genuine-decision")
	}
	if got := event.Data["base_tier"]; got != "propose-and-act" {
		t.Errorf("base_tier = %v, want %q", got, "propose-and-act")
	}
}

func TestLogDecision_MinimalFields(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")
	logger := events.NewLogger(logPath)

	// No target or reason
	err := LogDecision(logger, DecisionLogEntry{
		Class:      daemonconfig.DecisionDetectDuplicate,
		Compliance: daemonconfig.ComplianceStandard,
	})
	if err != nil {
		t.Fatalf("LogDecision() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	// Target and reason should be absent (not empty strings)
	if _, ok := event.Data["target"]; ok {
		t.Errorf("target should be absent when empty, got %v", event.Data["target"])
	}
	if _, ok := event.Data["reason"]; ok {
		t.Errorf("reason should be absent when empty, got %v", event.Data["reason"])
	}
}

func TestLogDecision_NilLogger(t *testing.T) {
	// Should not panic with nil logger — just no-op
	err := LogDecision(nil, DecisionLogEntry{
		Class:      daemonconfig.DecisionSelectIssue,
		Compliance: daemonconfig.ComplianceStandard,
	})
	if err != nil {
		t.Errorf("LogDecision(nil) should not return error, got %v", err)
	}
}
