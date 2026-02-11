// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/skills"
)

// TestOrchestratorSkillDetection tests that the orchestrator skill is correctly
// detected as an orchestrator-type skill (skill-type: policy).
func TestOrchestratorSkillDetection(t *testing.T) {
	loader := skills.DefaultLoader()

	// Load the orchestrator skill content
	content, err := loader.LoadSkillContent("orchestrator")
	if err != nil {
		t.Skip("Skipping: orchestrator skill file not available in this environment")
	}

	// Parse the metadata
	metadata, err := skills.ParseSkillMetadata(content)
	if err != nil {
		t.Fatalf("Failed to parse orchestrator skill metadata: %v", err)
	}

	// Check skill type
	if metadata.SkillType != "policy" {
		t.Errorf("orchestrator skill-type = %q, want %q", metadata.SkillType, "policy")
	}

	// Check that it would be detected as orchestrator
	isOrchestrator := metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"
	if !isOrchestrator {
		t.Errorf("orchestrator skill should be detected as orchestrator type, but isOrchestrator = %v", isOrchestrator)
	}
}

// TestSpawnClaudeContextFilePathSelection tests that SpawnClaude would use
// the correct context file path based on IsOrchestrator and IsMetaOrchestrator flags.
// Note: We can't directly test SpawnClaude without tmux, so we test the underlying
// ContextFilePath() method which is what SpawnClaude uses.
func TestSpawnClaudeContextFilePathSelection(t *testing.T) {
	tests := []struct {
		name               string
		isOrchestrator     bool
		isMetaOrchestrator bool
		wantFilename       string
	}{
		{
			name:               "worker spawn uses SPAWN_CONTEXT.md",
			isOrchestrator:     false,
			isMetaOrchestrator: false,
			wantFilename:       "SPAWN_CONTEXT.md",
		},
		{
			name:               "orchestrator spawn uses ORCHESTRATOR_CONTEXT.md",
			isOrchestrator:     true,
			isMetaOrchestrator: false,
			wantFilename:       "ORCHESTRATOR_CONTEXT.md",
		},
		{
			name:               "meta-orchestrator spawn uses META_ORCHESTRATOR_CONTEXT.md",
			isOrchestrator:     true,
			isMetaOrchestrator: true,
			wantFilename:       "META_ORCHESTRATOR_CONTEXT.md",
		},
		{
			name:               "meta-orchestrator flag takes priority even if IsOrchestrator is false",
			isOrchestrator:     false,
			isMetaOrchestrator: true,
			wantFilename:       "META_ORCHESTRATOR_CONTEXT.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ProjectDir:         "/Users/test/project",
				WorkspaceName:      "og-test-workspace-15jan",
				IsOrchestrator:     tt.isOrchestrator,
				IsMetaOrchestrator: tt.isMetaOrchestrator,
			}

			path := cfg.ContextFilePath()

			if !strings.HasSuffix(path, tt.wantFilename) {
				t.Errorf("ContextFilePath() = %s, want suffix %s", path, tt.wantFilename)
			}
		})
	}
}

// TestSpawnClaudeSessionSelection tests the session selection logic that SpawnClaude uses.
// This validates that orchestrators go to "orchestrator" session and workers go to "workers-*" session.
// Note: We can't test SpawnClaude directly without tmux, but we can verify the condition logic.
func TestSpawnClaudeSessionSelection(t *testing.T) {
	tests := []struct {
		name                 string
		isOrchestrator       bool
		isMetaOrchestrator   bool
		wantOrchestratorSess bool // true = should use orchestrator session, false = workers session
	}{
		{
			name:                 "worker should use workers session",
			isOrchestrator:       false,
			isMetaOrchestrator:   false,
			wantOrchestratorSess: false,
		},
		{
			name:                 "orchestrator should use orchestrator session",
			isOrchestrator:       true,
			isMetaOrchestrator:   false,
			wantOrchestratorSess: true,
		},
		{
			name:                 "meta-orchestrator should use orchestrator session",
			isOrchestrator:       false,
			isMetaOrchestrator:   true,
			wantOrchestratorSess: true,
		},
		{
			name:                 "both orchestrator flags set should use orchestrator session",
			isOrchestrator:       true,
			isMetaOrchestrator:   true,
			wantOrchestratorSess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				IsOrchestrator:     tt.isOrchestrator,
				IsMetaOrchestrator: tt.isMetaOrchestrator,
			}

			// This is the same condition used in SpawnClaude at line 18
			useOrchestratorSession := cfg.IsMetaOrchestrator || cfg.IsOrchestrator

			if useOrchestratorSession != tt.wantOrchestratorSess {
				t.Errorf("session selection: (IsMetaOrchestrator=%v || IsOrchestrator=%v) = %v, want %v",
					cfg.IsMetaOrchestrator, cfg.IsOrchestrator, useOrchestratorSession, tt.wantOrchestratorSess)
			}
		})
	}
}
