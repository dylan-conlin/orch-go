package orch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestFormatGapContext_TimeoutWithoutMatches(t *testing.T) {
	analysis := &spawn.GapAnalysis{
		HasGaps:        true,
		ContextQuality: 0,
		Gaps: []spawn.Gap{{
			Type:     spawn.GapTypeTimeout,
			Severity: spawn.GapSeverityWarning,
		}},
	}

	got := formatGapContext(analysis, nil)
	want := "⚠️ KB context check timed out — agent may be missing historical context"
	if got != want {
		t.Fatalf("formatGapContext() = %q, want %q", got, want)
	}
}

func TestFormatGapContext_PrependsSummaryToFormattedContext(t *testing.T) {
	analysis := &spawn.GapAnalysis{
		HasGaps:        true,
		ContextQuality: 10,
		Gaps: []spawn.Gap{{
			Type:     spawn.GapTypeSparseContext,
			Severity: spawn.GapSeverityWarning,
		}},
	}
	formatResult := &spawn.KBContextFormatResult{Content: "## PRIOR KNOWLEDGE\n\nbody"}

	got := formatGapContext(analysis, formatResult)
	want := "⚠️ Limited context (10/100) - agent may need to discover patterns during work\n\n## PRIOR KNOWLEDGE\n\nbody"
	if got != want {
		t.Fatalf("formatGapContext() = %q, want %q", got, want)
	}
}

func TestLogKBContextTimeoutIfNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	t.Setenv("ORCH_EVENTS_PATH", logPath)

	analysis := &spawn.GapAnalysis{
		Query:   "wire context timeout detection",
		HasGaps: true,
		Gaps: []spawn.Gap{{
			Type:     spawn.GapTypeTimeout,
			Severity: spawn.GapSeverityWarning,
		}},
	}

	logKBContextTimeoutIfNeeded(analysis, "og-feat-wire-kb-context-27mar-7ee3", "orch-go-r0pu5", "/tmp/orch-go", "feature-impl")

	data, err := os.ReadFile(events.NewLogger(logPath).CurrentPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if event.Type != events.EventTypeKBContextTimeout {
		t.Fatalf("Type = %q, want %q", event.Type, events.EventTypeKBContextTimeout)
	}
	if event.SessionID != "og-feat-wire-kb-context-27mar-7ee3" {
		t.Fatalf("SessionID = %q, want %q", event.SessionID, "og-feat-wire-kb-context-27mar-7ee3")
	}
	if event.Data["query"] != "wire context timeout detection" {
		t.Fatalf("query = %v, want %q", event.Data["query"], "wire context timeout detection")
	}
}

func TestLogKBContextTimeoutIfNeeded_IgnoresNonTimeoutGaps(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	t.Setenv("ORCH_EVENTS_PATH", logPath)

	analysis := &spawn.GapAnalysis{
		Query:   "wire context timeout detection",
		HasGaps: true,
		Gaps: []spawn.Gap{{
			Type:     spawn.GapTypeNoContext,
			Severity: spawn.GapSeverityCritical,
		}},
	}

	logKBContextTimeoutIfNeeded(analysis, "workspace", "orch-go-r0pu5", "/tmp/orch-go", "feature-impl")

	if _, err := os.Stat(events.NewLogger(logPath).CurrentPath()); !os.IsNotExist(err) {
		t.Fatalf("expected no timeout event log, got err=%v", err)
	}
}
