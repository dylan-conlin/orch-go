package main

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// TestBuildBatchSynthesis verifies that batch synthesis aggregates agent findings.
func TestBuildBatchSynthesis(t *testing.T) {
	completions := []CompletionInfo{
		{
			WorkspaceID:   "og-feat-add-auth-10mar",
			BeadsID:       "orch-go-abc1",
			Project:       "orch-go",
			Skill:         "feature-impl",
			Phase:         "Complete",
			Summary:       "Added JWT auth middleware",
			VerifyOK:      true,
			WorkspacePath: "/tmp/test-ws1",
			Synthesis: &verify.Synthesis{
				TLDR:      "Added JWT authentication middleware to API server",
				Outcome:   "success",
				Knowledge: "Discovered that the existing session handler already validates tokens but doesn't refresh them.",
				Delta:     "### Files Created\n\n- pkg/auth/jwt.go\n- pkg/auth/jwt_test.go\n\n### Files Modified\n\n- cmd/orch/serve.go\n\n### Commits\n\n- abc123 feat: add JWT auth",
				NextActions: []string{
					"Add token refresh endpoint",
					"Update dashboard to pass auth headers",
				},
				UnexploredQuestions: "How does the OAuth flow interact with the new JWT middleware?",
				Recommendation:     "close",
			},
		},
		{
			WorkspaceID:   "og-inv-auth-flow-10mar",
			BeadsID:       "orch-go-abc2",
			Project:       "orch-go",
			Skill:         "investigation",
			Phase:         "Complete",
			Summary:       "Investigated auth flow end-to-end",
			VerifyOK:      true,
			WorkspacePath: "/tmp/test-ws2",
			Synthesis: &verify.Synthesis{
				TLDR:      "OAuth flow has 3 token sources with no unified refresh path",
				Outcome:   "success",
				Knowledge: "Token sources: OpenCode auth.json, macOS Keychain, accounts.yaml. No single refresh mechanism covers all three.",
				NextActions: []string{
					"Unify token refresh across all 3 sources",
					"Add token refresh endpoint",
				},
				UnexploredQuestions: "What happens when keychain token expires during an active spawn?",
				Recommendation:     "close",
			},
		},
		{
			WorkspaceID: "og-feat-dashboard-10mar",
			BeadsID:     "orch-go-abc3",
			Project:     "orch-go",
			Skill:       "feature-impl",
			Phase:       "Complete",
			Summary:     "Fixed dashboard SSE proxy",
			VerifyOK:    true,
			IsLightTier: true,
			// No synthesis (light tier)
		},
	}

	batch := buildBatchSynthesis(completions, "orch-go")

	// Should have correct agent count
	if batch.AgentCount != 3 {
		t.Errorf("Expected 3 agents, got %d", batch.AgentCount)
	}

	// Should have 2 agents with synthesis
	if batch.SynthesisCount != 2 {
		t.Errorf("Expected 2 agents with synthesis, got %d", batch.SynthesisCount)
	}

	// Should have 1 light tier
	if batch.LightTierCount != 1 {
		t.Errorf("Expected 1 light tier agent, got %d", batch.LightTierCount)
	}

	// Should collect TLDRs
	if len(batch.Findings) != 2 {
		t.Errorf("Expected 2 findings, got %d", len(batch.Findings))
	}

	// Should deduplicate next actions
	if len(batch.NextActions) == 0 {
		t.Error("Expected non-empty next actions")
	}
	// "Add token refresh endpoint" appears in both agents — should be deduplicated
	refreshCount := 0
	for _, a := range batch.NextActions {
		if strings.Contains(strings.ToLower(a.Action), "token refresh endpoint") {
			refreshCount++
		}
	}
	if refreshCount != 1 {
		t.Errorf("Expected 'token refresh endpoint' deduplicated to 1, got %d", refreshCount)
	}

	// Should collect open questions
	if len(batch.OpenQuestions) == 0 {
		t.Error("Expected non-empty open questions")
	}

	// Should detect cross-references (both agents mention auth/token topics)
	if len(batch.Connections) == 0 {
		t.Error("Expected non-empty connections (shared next actions)")
	}
}

// TestBuildBatchSynthesisEmpty handles no completions.
func TestBuildBatchSynthesisEmpty(t *testing.T) {
	batch := buildBatchSynthesis(nil, "orch-go")

	if batch.AgentCount != 0 {
		t.Errorf("Expected 0 agents, got %d", batch.AgentCount)
	}
	if len(batch.Findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(batch.Findings))
	}
}

// TestBuildBatchSynthesisLightTierOnly handles all light tier agents.
func TestBuildBatchSynthesisLightTierOnly(t *testing.T) {
	completions := []CompletionInfo{
		{
			WorkspaceID: "og-feat-a",
			Project:     "orch-go",
			IsLightTier: true,
			Phase:       "Complete",
			Summary:     "Did thing A",
		},
		{
			WorkspaceID: "og-feat-b",
			Project:     "orch-go",
			IsLightTier: true,
			Phase:       "Complete",
			Summary:     "Did thing B",
		},
	}

	batch := buildBatchSynthesis(completions, "orch-go")

	if batch.AgentCount != 2 {
		t.Errorf("Expected 2 agents, got %d", batch.AgentCount)
	}
	if batch.SynthesisCount != 0 {
		t.Errorf("Expected 0 synthesis, got %d", batch.SynthesisCount)
	}
	if batch.LightTierCount != 2 {
		t.Errorf("Expected 2 light tier, got %d", batch.LightTierCount)
	}
}

// TestFormatBatchSynthesis verifies the output format.
func TestFormatBatchSynthesis(t *testing.T) {
	batch := BatchSynthesis{
		Project:        "orch-go",
		AgentCount:     2,
		SynthesisCount: 2,
		Findings: []AgentFinding{
			{
				WorkspaceID: "og-feat-auth",
				BeadsID:     "orch-go-abc1",
				Skill:       "feature-impl",
				TLDR:        "Added JWT auth",
				Outcome:     "success",
				Knowledge:   "Session handler validates but doesn't refresh.",
			},
			{
				WorkspaceID: "og-inv-auth",
				BeadsID:     "orch-go-abc2",
				Skill:       "investigation",
				TLDR:        "3 token sources, no unified refresh",
				Outcome:     "success",
				Knowledge:   "Token sources: auth.json, keychain, accounts.yaml.",
			},
		},
		NextActions: []BatchNextAction{
			{Action: "Add token refresh endpoint", Sources: []string{"og-feat-auth", "og-inv-auth"}},
			{Action: "Update dashboard auth headers", Sources: []string{"og-feat-auth"}},
		},
		OpenQuestions: []BatchQuestion{
			{Question: "How does OAuth interact with JWT?", Source: "og-feat-auth"},
		},
		Connections: []Connection{
			{Description: "Shared next action: Add token refresh endpoint", Agents: []string{"og-feat-auth", "og-inv-auth"}},
		},
	}

	output := formatBatchSynthesis(batch)

	// Should contain section headers
	if !strings.Contains(output, "BATCH SYNTHESIS") {
		t.Error("Expected 'BATCH SYNTHESIS' header")
	}
	if !strings.Contains(output, "WHAT WE NOW KNOW") {
		t.Error("Expected 'WHAT WE NOW KNOW' section")
	}
	if !strings.Contains(output, "NEXT ACTIONS") {
		t.Error("Expected 'NEXT ACTIONS' section")
	}
	if !strings.Contains(output, "OPEN QUESTIONS") {
		t.Error("Expected 'OPEN QUESTIONS' section")
	}
	if !strings.Contains(output, "CONNECTIONS") {
		t.Error("Expected 'CONNECTIONS' section")
	}

	// Should contain agent findings
	if !strings.Contains(output, "Added JWT auth") {
		t.Error("Expected finding TLDR in output")
	}

	// Should show multi-source next actions
	if !strings.Contains(output, "og-feat-auth, og-inv-auth") {
		t.Error("Expected shared sources in next actions")
	}
}

// TestDeduplicateNextActions verifies deduplication logic.
func TestDeduplicateNextActions(t *testing.T) {
	actions := []rawNextAction{
		{action: "Add token refresh endpoint", source: "ws-1"},
		{action: "add token refresh endpoint", source: "ws-2"}, // case-insensitive dup
		{action: "Update dashboard", source: "ws-1"},
		{action: "Add token refresh endpoint", source: "ws-3"}, // exact dup
	}

	deduped := deduplicateNextActions(actions)

	if len(deduped) != 2 {
		t.Errorf("Expected 2 deduplicated actions, got %d", len(deduped))
	}

	// First action should have 3 sources (deduplicated)
	for _, a := range deduped {
		if strings.Contains(strings.ToLower(a.Action), "token refresh") {
			if len(a.Sources) != 3 {
				t.Errorf("Expected 3 sources for 'token refresh', got %d", len(a.Sources))
			}
		}
	}
}

// TestReviewSynthesizeCommandExists verifies the subcommand is registered.
func TestReviewSynthesizeCommandExists(t *testing.T) {
	cmd, _, err := reviewCmd.Find([]string{"synthesize"})
	if err != nil || cmd == nil {
		t.Fatal("Expected 'synthesize' subcommand on review")
	}

	// Check aliases
	aliases := cmd.Aliases
	found := false
	for _, a := range aliases {
		if a == "synth" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'synth' alias for synthesize subcommand")
	}
}
