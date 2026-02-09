package dialogue

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeriveArtifactsExtractsDecisionAndFollowUps(t *testing.T) {
	result := &RelayResult{
		Approved:  true,
		Verdict:   "APPROVED",
		EndReason: "ghost_approved",
		Turns: []Turn{
			{Number: 1, Phase: PhaseExplore, GhostMessage: "What currently causes retry storms?", ExpertResponse: "Three services run retries independently."},
			{Number: 2, Phase: PhaseTerminate, GhostMessage: `[VERDICT: APPROVED]

## Decision
Adopt a centralized retry coordinator with shared policy.

## Follow-Up Issues
1. Build coordinator package and wire service callers.
2. Remove local retry wrappers and update telemetry dashboards.`},
		},
	}

	decisions, followUps := DeriveArtifacts(result)
	if len(decisions) != 1 {
		t.Fatalf("len(decisions) = %d, want 1", len(decisions))
	}
	if got := decisions[0].Summary; got != "Adopt a centralized retry coordinator with shared policy." {
		t.Fatalf("decision summary = %q", got)
	}

	if len(followUps) != 2 {
		t.Fatalf("len(followUps) = %d, want 2", len(followUps))
	}
	if followUps[0].IssueType != "feature" {
		t.Fatalf("followUps[0].IssueType = %q, want %q", followUps[0].IssueType, "feature")
	}
	if followUps[1].IssueType != "task" {
		t.Fatalf("followUps[1].IssueType = %q, want %q", followUps[1].IssueType, "task")
	}
}

func TestFormatTranscriptIncludesStructuredMetadata(t *testing.T) {
	started := time.Date(2026, 2, 8, 10, 0, 0, 0, time.UTC)
	completed := started.Add(4 * time.Minute)

	content := FormatTranscript(TranscriptMetadata{
		SessionID:       "ses_test",
		Topic:           "retry architecture",
		QuestionerModel: "claude-sonnet-4-5-20250929",
		ExpertModel:     "anthropic/claude-opus-4-1-20250805",
		StartedAt:       started,
		CompletedAt:     completed,
	}, RelayConfig{ExploreTurns: 2, ConvergeTurns: 4, MaxTurns: 6}, &RelayResult{
		Approved:  true,
		Verdict:   "APPROVED",
		EndReason: "ghost_approved",
		Turns: []Turn{{
			Number:         1,
			Phase:          PhaseExplore,
			GhostMessage:   "What changed this week?",
			ExpertResponse: "Two services shipped independent retries.",
			Usage:          Usage{InputTokens: 10, OutputTokens: 7},
		}},
	})

	mustContain := []string{
		"type: dialogue_transcript",
		"session_id: \"ses_test\"",
		"# Dialogue Transcript",
		"## Turn 1 (explore)",
		"_Ghost usage: input=10 output=7_",
	}
	for _, token := range mustContain {
		if !strings.Contains(content, token) {
			t.Fatalf("transcript missing %q", token)
		}
	}
}

func TestWriteArtifactsWritesAllFiles(t *testing.T) {
	workspace := t.TempDir()
	result := &RelayResult{
		Approved:  true,
		Verdict:   "APPROVED",
		EndReason: "ghost_approved",
		Turns: []Turn{{
			Number:       1,
			Phase:        PhaseTerminate,
			GhostMessage: "[VERDICT: APPROVED]\n\n## Decision\nUse a single queue.\n\n## Follow-Up Issues\n- Build queue adapter",
		}},
	}

	files, err := WriteArtifacts(workspace, TranscriptMetadata{SessionID: "ses_test", Topic: "queues", QuestionerModel: "sonnet"}, RelayConfig{}, result)
	if err != nil {
		t.Fatalf("WriteArtifacts() error = %v", err)
	}

	paths := []string{files.TranscriptPath, files.ArtifactMDPath, files.ArtifactJSONPath}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file at %s: %v", path, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(workspace, ArtifactReportFileName))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(data), "## Follow-Up Issue Drafts") {
		t.Fatalf("artifact report missing follow-up section")
	}
}
