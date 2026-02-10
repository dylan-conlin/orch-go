package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestApplyAutoModelSkipProfileForGPT(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".tier"), []byte("light\n"), 0644); err != nil {
		t.Fatalf("failed to write .tier: %v", err)
	}

	if err := spawn.WriteAgentManifest(workspace, spawn.AgentManifest{
		WorkspaceName: "og-feat-gpt",
		Tier:          "light",
		Model:         "openai/gpt-5.3-codex",
	}); err != nil {
		t.Fatalf("failed to write AGENT_MANIFEST.json: %v", err)
	}

	target := &CompletionTarget{WorkspacePath: workspace, AgentName: "og-feat-gpt"}
	skip := applyAutoModelSkipProfile(target, SkipConfig{})

	if !skip.ModelConnection {
		t.Fatalf("expected %s skip to be enabled", verify.GateModelConnection)
	}
	if !skip.GitDiff {
		t.Fatalf("expected %s skip to be enabled", verify.GateGitDiff)
	}
	if !skip.VerificationSpec {
		t.Fatalf("expected %s skip to be enabled", verify.GateVerificationSpec)
	}
	if skip.Reason != autoGPTSkipReason {
		t.Fatalf("skip reason = %q, want %q", skip.Reason, autoGPTSkipReason)
	}
}

func TestApplyAutoModelSkipProfilePreservesExplicitReason(t *testing.T) {
	workspace := t.TempDir()
	if err := spawn.WriteAgentManifest(workspace, spawn.AgentManifest{
		WorkspaceName: "og-feat-gpt",
		Tier:          "light",
		Model:         "gpt",
	}); err != nil {
		t.Fatalf("failed to write AGENT_MANIFEST.json: %v", err)
	}

	target := &CompletionTarget{WorkspacePath: workspace, AgentName: "og-feat-gpt"}
	skip := applyAutoModelSkipProfile(target, SkipConfig{
		BatchMode: true,
		Reason:    "batch mode - core gates only",
	})

	if skip.Reason != "batch mode - core gates only" {
		t.Fatalf("skip reason should be preserved, got %q", skip.Reason)
	}
	if !skip.ModelConnection || !skip.GitDiff || !skip.VerificationSpec {
		t.Fatalf("expected GPT profile skips to be enabled, got %+v", skip)
	}
}

func TestApplyAutoModelSkipProfileNonGPTNoop(t *testing.T) {
	workspace := t.TempDir()
	if err := spawn.WriteAgentManifest(workspace, spawn.AgentManifest{
		WorkspaceName: "og-feat-claude",
		Tier:          "light",
		Model:         "anthropic/claude-opus-4-6",
	}); err != nil {
		t.Fatalf("failed to write AGENT_MANIFEST.json: %v", err)
	}

	original := SkipConfig{BatchMode: true, Reason: "batch mode - core gates only"}
	target := &CompletionTarget{WorkspacePath: workspace, AgentName: "og-feat-claude"}
	got := applyAutoModelSkipProfile(target, original)

	if got != original {
		t.Fatalf("expected non-GPT profile to be unchanged, got %+v", got)
	}
}
