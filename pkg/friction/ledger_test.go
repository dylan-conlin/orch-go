package friction

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "friction-ledger.jsonl")
	entry := Entry{
		Symptom:      "Daemon spawned duplicate worker",
		Impact:       "Consumed one extra slot for 4 minutes",
		EvidencePath: ".kb/investigations/2026-02-08-dup-spawn.md",
		LinkedIssue:  "orch-go-21409",
	}

	written, err := Append(path, entry)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if written.ID == "" {
		t.Fatal("expected generated ID")
	}
	if written.Timestamp.IsZero() {
		t.Fatal("expected generated timestamp")
	}

	entries, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Symptom != entry.Symptom {
		t.Fatalf("unexpected symptom: %q", entries[0].Symptom)
	}
	if entries[0].LinkedIssue != entry.LinkedIssue {
		t.Fatalf("unexpected linked issue: %q", entries[0].LinkedIssue)
	}
}

func TestAppendValidation(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "friction-ledger.jsonl")
	_, err := Append(path, Entry{Symptom: "", Impact: "impact", EvidencePath: "path", LinkedIssue: "id"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "does-not-exist.jsonl")
	entries, err := Load(path)
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestLoadParseErrorIncludesLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "friction-ledger.jsonl")
	if err := os.WriteFile(path, []byte("{\"symptom\":\"ok\"}\n{bad json}\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "line 2") {
		t.Fatalf("expected error to mention line 2, got: %v", err)
	}
}

func TestSummarizeGroupsByNormalizedSymptom(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 2, 8, 19, 0, 0, 0, time.UTC)
	entries := []Entry{
		{
			Timestamp:    base,
			Symptom:      "Duplicate spawn race",
			Impact:       "One wasted slot",
			EvidencePath: "a.md",
			LinkedIssue:  "orch-go-1",
		},
		{
			Timestamp:    base.Add(1 * time.Hour),
			Symptom:      " duplicate   spawn race ",
			Impact:       "Two wasted slots",
			EvidencePath: "b.md",
			LinkedIssue:  "orch-go-2",
		},
		{
			Timestamp:    base.Add(2 * time.Hour),
			Symptom:      "Missing phase complete",
			Impact:       "Completion queue blocked",
			EvidencePath: "c.md",
			LinkedIssue:  "orch-go-3",
		},
	}

	summaries := Summarize(entries)
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	if summaries[0].Count != 2 {
		t.Fatalf("expected top summary count=2, got %d", summaries[0].Count)
	}
	if summaries[0].LatestEvidence != "b.md" {
		t.Fatalf("expected latest evidence b.md, got %s", summaries[0].LatestEvidence)
	}
	if len(summaries[0].LinkedIssues) != 2 {
		t.Fatalf("expected 2 linked issues, got %d", len(summaries[0].LinkedIssues))
	}
}
