// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"encoding/json"
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
		t.Fatalf("Failed to load orchestrator skill: %v", err)
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

// TestMCPConfigJSON tests that known MCP presets produce valid JSON configs.
func TestMCPConfigJSON(t *testing.T) {
	t.Run("playwright preset returns valid JSON", func(t *testing.T) {
		configJSON, ok := MCPConfigJSON("playwright")
		if !ok {
			t.Fatal("MCPConfigJSON('playwright') returned false, want true")
		}
		if configJSON == "" {
			t.Fatal("MCPConfigJSON('playwright') returned empty string")
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &parsed); err != nil {
			t.Fatalf("MCPConfigJSON('playwright') returned invalid JSON: %v\nGot: %s", err, configJSON)
		}

		// Verify structure: {"mcpServers": {"playwright": {"command": "npx", "args": [...]}}}
		servers, ok := parsed["mcpServers"].(map[string]interface{})
		if !ok {
			t.Fatalf("missing or invalid mcpServers key in: %s", configJSON)
		}
		pw, ok := servers["playwright"].(map[string]interface{})
		if !ok {
			t.Fatalf("missing or invalid playwright server in: %s", configJSON)
		}
		if pw["command"] != "npx" {
			t.Errorf("playwright command = %v, want 'npx'", pw["command"])
		}
		args, ok := pw["args"].([]interface{})
		if !ok || len(args) < 2 {
			t.Fatalf("playwright args missing or too short: %v", pw["args"])
		}
		if args[0] != "-y" {
			t.Errorf("playwright args[0] = %v, want '-y'", args[0])
		}
		argsStr, _ := args[1].(string)
		if !strings.HasPrefix(argsStr, "@playwright/mcp") {
			t.Errorf("playwright args[1] = %v, want prefix '@playwright/mcp'", args[1])
		}
	})

	t.Run("unknown preset returns false", func(t *testing.T) {
		_, ok := MCPConfigJSON("nonexistent")
		if ok {
			t.Error("MCPConfigJSON('nonexistent') returned true, want false")
		}
	})
}

// TestBuildClaudeLaunchCommand tests command construction with and without MCP.
func TestBuildClaudeLaunchCommand(t *testing.T) {
	tests := []struct {
		name         string
		contextPath  string
		claudeCtx    string
		mcp          string
		configDir    string
		wantContains []string
		wantExcludes []string
	}{
		{
			name:        "no MCP - basic command",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "",
			wantContains: []string{
				"export CLAUDE_CONTEXT=worker",
				"cat \"/tmp/workspace/SPAWN_CONTEXT.md\"",
				"claude --dangerously-skip-permissions",
			},
			wantExcludes: []string{
				"--mcp-config",
				"CLAUDE_CONFIG_DIR",
			},
		},
		{
			name:        "playwright MCP preset",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "playwright",
			configDir:   "",
			wantContains: []string{
				"export CLAUDE_CONTEXT=worker",
				"claude --dangerously-skip-permissions",
				"--mcp-config",
				"mcpServers",
				"playwright",
				"@playwright/mcp",
			},
		},
		{
			name:        "unknown MCP passed as raw value",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "/path/to/custom-mcp.json",
			configDir:   "",
			wantContains: []string{
				"--mcp-config '/path/to/custom-mcp.json'",
			},
		},
		{
			name:        "orchestrator context includes disallowedTools",
			contextPath: "/tmp/workspace/ORCHESTRATOR_CONTEXT.md",
			claudeCtx:   "orchestrator",
			mcp:         "",
			configDir:   "",
			wantContains: []string{
				"export CLAUDE_CONTEXT=orchestrator",
				"ORCHESTRATOR_CONTEXT.md",
				"--disallowedTools",
				"Task",
				"Edit",
				"Write",
				"NotebookEdit",
			},
		},
		{
			name:        "meta-orchestrator context includes disallowedTools",
			contextPath: "/tmp/workspace/META_ORCHESTRATOR_CONTEXT.md",
			claudeCtx:   "meta-orchestrator",
			mcp:         "",
			configDir:   "",
			wantContains: []string{
				"export CLAUDE_CONTEXT=meta-orchestrator",
				"--disallowedTools",
				"Task",
			},
		},
		{
			name:        "worker context does NOT include disallowedTools",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "",
			wantContains: []string{
				"export CLAUDE_CONTEXT=worker",
			},
			wantExcludes: []string{
				"--disallowedTools",
				"--mcp-config",
				"CLAUDE_CONFIG_DIR",
			},
		},
		{
			name:        "orchestrator with MCP includes both flags",
			contextPath: "/tmp/workspace/ORCHESTRATOR_CONTEXT.md",
			claudeCtx:   "orchestrator",
			mcp:         "playwright",
			configDir:   "",
			wantContains: []string{
				"--disallowedTools",
				"--mcp-config",
			},
		},
		{
			name:        "non-default configDir injects CLAUDE_CONFIG_DIR and unsets token",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "~/.claude-personal",
			wantContains: []string{
				"unset CLAUDE_CODE_OAUTH_TOKEN",
				"export CLAUDE_CONFIG_DIR=~/.claude-personal",
				"export CLAUDE_CONTEXT=worker",
			},
		},
		{
			name:        "default configDir does NOT inject CLAUDE_CONFIG_DIR",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "~/.claude",
			wantExcludes: []string{
				"CLAUDE_CONFIG_DIR",
				"unset CLAUDE_CODE_OAUTH_TOKEN",
			},
		},
		{
			name:        "empty configDir does NOT inject CLAUDE_CONFIG_DIR",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "",
			wantExcludes: []string{
				"CLAUDE_CONFIG_DIR",
				"unset CLAUDE_CODE_OAUTH_TOKEN",
			},
		},
		{
			name:        "configDir with MCP - both injected",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "playwright",
			configDir:   "~/.claude-personal",
			wantContains: []string{
				"unset CLAUDE_CODE_OAUTH_TOKEN",
				"export CLAUDE_CONFIG_DIR=~/.claude-personal",
				"--mcp-config",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildClaudeLaunchCommand(tt.contextPath, tt.claudeCtx, tt.mcp, tt.configDir)

			for _, want := range tt.wantContains {
				if !strings.Contains(cmd, want) {
					t.Errorf("command missing %q\nGot: %s", want, cmd)
				}
			}
			for _, exclude := range tt.wantExcludes {
				if strings.Contains(cmd, exclude) {
					t.Errorf("command should not contain %q\nGot: %s", exclude, cmd)
				}
			}
		})
	}
}
