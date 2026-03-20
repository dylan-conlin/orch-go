// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"os"
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
// This validates that orchestrators go to "orchestrator" session, workers go to "workers-*" session,
// and exploration orchestrators go to "workers-*" session (worker lifecycle).
// Note: We can't test SpawnClaude directly without tmux, but we can verify the condition logic.
func TestSpawnClaudeSessionSelection(t *testing.T) {
	tests := []struct {
		name                 string
		isOrchestrator       bool
		isMetaOrchestrator   bool
		explore              bool
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
		{
			name:                 "explore orchestrator should use workers session",
			isOrchestrator:       true,
			explore:              true,
			wantOrchestratorSess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				IsOrchestrator:     tt.isOrchestrator,
				IsMetaOrchestrator: tt.isMetaOrchestrator,
				Explore:            tt.explore,
			}

			// This mirrors the condition used in SpawnClaude
			var useOrchestratorSession bool
			if cfg.Explore {
				useOrchestratorSession = false
			} else {
				useOrchestratorSession = cfg.IsMetaOrchestrator || cfg.IsOrchestrator
			}

			if useOrchestratorSession != tt.wantOrchestratorSess {
				t.Errorf("session selection: (Explore=%v, IsMetaOrchestrator=%v, IsOrchestrator=%v) = %v, want %v",
					cfg.Explore, cfg.IsMetaOrchestrator, cfg.IsOrchestrator, useOrchestratorSession, tt.wantOrchestratorSess)
			}
		})
	}
}

// TestClaudeContextExploreOverride tests that Explore=true overrides IsOrchestrator
// to return "worker" context, ensuring explore orchestrators get worker hooks.
func TestClaudeContextExploreOverride(t *testing.T) {
	tests := []struct {
		name           string
		isOrchestrator bool
		explore        bool
		want           string
	}{
		{"worker", false, false, "worker"},
		{"orchestrator", true, false, "orchestrator"},
		{"explore overrides orchestrator", true, true, "worker"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				IsOrchestrator: tt.isOrchestrator,
				Explore:        tt.explore,
			}
			if got := cfg.ClaudeContext(); got != tt.want {
				t.Errorf("ClaudeContext() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestMCPConfigJSON tests that MCP presets produce valid JSON configs.
func TestMCPConfigJSON(t *testing.T) {
	t.Run("playwright MCP preset returns valid config", func(t *testing.T) {
		// --mcp playwright should produce a valid MCP server config.
		config, ok := MCPConfigJSON("playwright")
		if !ok {
			t.Fatal("MCPConfigJSON('playwright') returned false, want true (playwright is an MCP server preset)")
		}
		if !strings.Contains(config, "mcpServers") {
			t.Errorf("MCPConfigJSON('playwright') missing 'mcpServers' key: %s", config)
		}
		if !strings.Contains(config, "@anthropic-ai/mcp-server-playwright") {
			t.Errorf("MCPConfigJSON('playwright') missing server package: %s", config)
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
		name          string
		contextPath   string
		claudeCtx     string
		mcp           string
		configDir     string
		beadsDir      string
		beadsID       string
		disallowTools string
		wantContains  []string
		wantExcludes  []string
	}{
		{
			name:        "no MCP - basic command",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "",
			wantContains: []string{
				"export ORCH_SPAWNED=1",
				"export CLAUDE_CONTEXT=worker",
				"cat \"/tmp/workspace/SPAWN_CONTEXT.md\"",
				"claude --dangerously-skip-permissions",
			},
			wantExcludes: []string{
				"--mcp-config",
				"CLAUDE_CONFIG_DIR",
				"ORCH_BEADS_ID",
			},
		},
		{
			name:        "playwright MCP preset adds mcp-config flag",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "playwright",
			configDir:   "",
			wantContains: []string{
				"export CLAUDE_CONTEXT=worker",
				"claude --dangerously-skip-permissions",
				"--mcp-config",
				"@anthropic-ai/mcp-server-playwright",
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
			name:          "orchestrator context includes disallowedTools",
			contextPath:   "/tmp/workspace/ORCHESTRATOR_CONTEXT.md",
			claudeCtx:     "orchestrator",
			mcp:           "",
			configDir:     "",
			disallowTools: "Agent,Edit,Write,NotebookEdit",
			wantContains: []string{
				"export CLAUDE_CONTEXT=orchestrator",
				"ORCHESTRATOR_CONTEXT.md",
				"--disallowedTools",
				"Agent",
				"Edit",
				"Write",
				"NotebookEdit",
			},
		},
		{
			name:          "meta-orchestrator context includes disallowedTools",
			contextPath:   "/tmp/workspace/META_ORCHESTRATOR_CONTEXT.md",
			claudeCtx:     "meta-orchestrator",
			mcp:           "",
			configDir:     "",
			disallowTools: "Agent,Edit,Write,NotebookEdit",
			wantContains: []string{
				"export CLAUDE_CONTEXT=meta-orchestrator",
				"--disallowedTools",
				"Agent",
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
			name:          "orchestrator with playwright MCP gets both flags",
			contextPath:   "/tmp/workspace/ORCHESTRATOR_CONTEXT.md",
			claudeCtx:     "orchestrator",
			mcp:           "playwright",
			configDir:     "",
			disallowTools: "Agent,Edit,Write,NotebookEdit",
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
			name:        "configDir with playwright MCP - both configDir and mcp-config injected",
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
		{
			name:        "cross-repo beadsDir injects BEADS_DIR",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "",
			beadsDir:    "/Users/test/orch-go/.beads",
			wantContains: []string{
				"export BEADS_DIR=/Users/test/orch-go/.beads",
				"export CLAUDE_CONTEXT=worker",
			},
		},
		{
			name:        "empty beadsDir does NOT inject BEADS_DIR",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "",
			beadsDir:    "",
			wantExcludes: []string{
				"BEADS_DIR",
			},
		},
		{
			name:        "cross-repo beadsDir with configDir - both injected",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "~/.claude-personal",
			beadsDir:    "/Users/test/orch-go/.beads",
			wantContains: []string{
				"export CLAUDE_CONFIG_DIR=~/.claude-personal",
				"export BEADS_DIR=/Users/test/orch-go/.beads",
			},
		},
		{
			name:        "beadsID injects ORCH_BEADS_ID",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "",
			beadsID:     "orch-go-puat",
			wantContains: []string{
				"export ORCH_BEADS_ID=orch-go-puat",
				"export ORCH_SPAWNED=1",
			},
		},
		{
			name:        "empty beadsID does NOT inject ORCH_BEADS_ID",
			contextPath: "/tmp/workspace/SPAWN_CONTEXT.md",
			claudeCtx:   "worker",
			mcp:         "",
			configDir:   "",
			beadsID:     "",
			wantExcludes: []string{
				"ORCH_BEADS_ID",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildClaudeLaunchCommand(tt.contextPath, tt.claudeCtx, tt.mcp, tt.configDir, tt.beadsDir, tt.beadsID, "", 0, "", "", tt.disallowTools)

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

// TestBuildClaudeLaunchCommandMaxTurns tests --max-turns flag injection.
func TestBuildClaudeLaunchCommandMaxTurns(t *testing.T) {
	tests := []struct {
		name         string
		maxTurns     int
		wantContains []string
		wantExcludes []string
	}{
		{
			name:     "zero maxTurns omits flag",
			maxTurns: 0,
			wantExcludes: []string{
				"--max-turns",
			},
		},
		{
			name:     "positive maxTurns adds flag",
			maxTurns: 150,
			wantContains: []string{
				"--max-turns 150",
			},
		},
		{
			name:     "small maxTurns for light tier",
			maxTurns: 30,
			wantContains: []string{
				"--max-turns 30",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildClaudeLaunchCommand("/tmp/SPAWN_CONTEXT.md", "worker", "", "", "", "", "", tt.maxTurns, "", "", "")

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

// TestBuildClaudeLaunchCommandEffort tests --effort flag injection.
func TestBuildClaudeLaunchCommandEffort(t *testing.T) {
	tests := []struct {
		name         string
		effort       string
		wantContains []string
		wantExcludes []string
	}{
		{
			name:   "empty effort omits flag",
			effort: "",
			wantExcludes: []string{
				"--effort",
			},
		},
		{
			name:   "high effort adds flag",
			effort: "high",
			wantContains: []string{
				"--effort high",
			},
		},
		{
			name:   "medium effort adds flag",
			effort: "medium",
			wantContains: []string{
				"--effort medium",
			},
		},
		{
			name:   "low effort adds flag",
			effort: "low",
			wantContains: []string{
				"--effort low",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildClaudeLaunchCommand("/tmp/SPAWN_CONTEXT.md", "worker", "", "", "", "", tt.effort, 0, "", "", "")

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

// TestBuildClaudeLaunchCommandSystemPromptFile tests --append-system-prompt injection.
func TestBuildClaudeLaunchCommandSystemPromptFile(t *testing.T) {
	tests := []struct {
		name             string
		systemPromptFile string
		wantContains     []string
		wantExcludes     []string
	}{
		{
			name:             "empty systemPromptFile omits flags",
			systemPromptFile: "",
			wantExcludes: []string{
				"--append-system-prompt",
				"--disable-slash-commands",
			},
		},
		{
			name:             "systemPromptFile adds append-system-prompt with cat substitution",
			systemPromptFile: "/tmp/workspace/SKILL_PROMPT.md",
			wantContains: []string{
				`--append-system-prompt "$(cat`,
				"/tmp/workspace/SKILL_PROMPT.md",
				"--disable-slash-commands",
			},
		},
		{
			name:             "systemPromptFile with spaces in path is quoted",
			systemPromptFile: "/tmp/my workspace/SKILL_PROMPT.md",
			wantContains: []string{
				`--append-system-prompt "$(cat`,
				"/tmp/my workspace/SKILL_PROMPT.md",
				"--disable-slash-commands",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildClaudeLaunchCommand("/tmp/SPAWN_CONTEXT.md", "worker", "", "", "", "", "", 0, "", tt.systemPromptFile, "")

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

// TestWriteSkillPromptFile tests writing skill content to SKILL_PROMPT.md.
func TestWriteSkillPromptFile(t *testing.T) {
	t.Run("empty skill content is a no-op", func(t *testing.T) {
		cfg := &Config{
			ProjectDir:    t.TempDir(),
			WorkspaceName: "test-workspace",
			SkillContent:  "",
		}
		// Create workspace dir
		os.MkdirAll(cfg.WorkspacePath(), 0755)

		err := WriteSkillPromptFile(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.SystemPromptFile != "" {
			t.Errorf("SystemPromptFile should be empty for empty SkillContent, got %q", cfg.SystemPromptFile)
		}
	})

	t.Run("writes skill content to SKILL_PROMPT.md", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{
			ProjectDir:    tmpDir,
			WorkspaceName: "test-workspace",
			SkillContent:  "# Test Skill\n\nSome skill content here.",
			BeadsID:       "test-123",
			Tier:          "light",
		}
		os.MkdirAll(cfg.WorkspacePath(), 0755)

		err := WriteSkillPromptFile(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify SystemPromptFile is set
		expectedPath := cfg.WorkspacePath() + "/SKILL_PROMPT.md"
		if cfg.SystemPromptFile != expectedPath {
			t.Errorf("SystemPromptFile = %q, want %q", cfg.SystemPromptFile, expectedPath)
		}

		// Verify file content
		content, err := os.ReadFile(cfg.SystemPromptFile)
		if err != nil {
			t.Fatalf("failed to read SKILL_PROMPT.md: %v", err)
		}
		if !strings.Contains(string(content), "Test Skill") {
			t.Errorf("SKILL_PROMPT.md missing skill content, got: %s", content)
		}
	})

	t.Run("processes template variables", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{
			ProjectDir:    tmpDir,
			WorkspaceName: "test-workspace",
			SkillContent:  "Report to {{.BeadsID}} at tier {{.Tier}}",
			BeadsID:       "orch-go-abc1",
			Tier:          "full",
		}
		os.MkdirAll(cfg.WorkspacePath(), 0755)

		err := WriteSkillPromptFile(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(cfg.SystemPromptFile)
		if err != nil {
			t.Fatalf("failed to read SKILL_PROMPT.md: %v", err)
		}
		got := string(content)
		if !strings.Contains(got, "orch-go-abc1") {
			t.Errorf("template variable {{.BeadsID}} not processed, got: %s", got)
		}
		if !strings.Contains(got, "full") {
			t.Errorf("template variable {{.Tier}} not processed, got: %s", got)
		}
	})
}

// TestBuildClaudeLaunchCommandSettings tests --settings flag injection.
func TestBuildClaudeLaunchCommandSettings(t *testing.T) {
	tests := []struct {
		name         string
		settings     string
		wantContains []string
		wantExcludes []string
	}{
		{
			name:     "empty settings omits flag",
			settings: "",
			wantExcludes: []string{
				"--settings",
			},
		},
		{
			name:     "settings path adds flag",
			settings: "/Users/test/.claude/worker-settings.json",
			wantContains: []string{
				`--settings "/Users/test/.claude/worker-settings.json"`,
			},
		},
		{
			name:     "settings path with spaces is quoted",
			settings: "/Users/test/my settings/hooks.json",
			wantContains: []string{
				`--settings "/Users/test/my settings/hooks.json"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildClaudeLaunchCommand("/tmp/SPAWN_CONTEXT.md", "worker", "", "", "", "", "", 0, tt.settings, "", "")

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
