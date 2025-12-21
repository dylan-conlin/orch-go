// Package main provides the CLI entry point for orch-go.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/question"
	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/usage"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	serverURL string

	// Version information (set at build time)
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "orch-go",
	Short: "OpenCode orchestration CLI",
	Long: `orch-go is a CLI tool for orchestrating OpenCode sessions.

It provides commands for spawning new sessions, sending messages to existing
sessions, and monitoring session events via SSE.`,
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://127.0.0.1:4096", "OpenCode server URL")

	rootCmd.AddCommand(spawnCmd)
	rootCmd.AddCommand(askCmd)
	sendCmd.Flags().BoolVar(&sendAsync, "async", true, "Send message asynchronously (non-blocking)")
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(completeCmd)
	rootCmd.AddCommand(workCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(tailCmd)
	rootCmd.AddCommand(questionCmd)
	rootCmd.AddCommand(abandonCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(focusCmd)
	rootCmd.AddCommand(driftCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("orch version %s\n", version)
		fmt.Printf("build time: %s\n", buildTime)
	},
}

var (
	// Spawn command flags
	spawnSkill             string
	spawnIssue             string
	spawnPhases            string
	spawnMode              string
	spawnValidation        string
	spawnInline            bool   // Run inline (blocking) instead of headless
	spawnModel             string // Model to use for standalone spawns
	spawnNoTrack           bool   // Opt-out of beads tracking
	spawnMCP               string // MCP server config (e.g., "playwright")
	spawnSkipArtifactCheck bool   // Bypass pre-spawn kb context check
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [skill] [task]",
	Short: "Spawn a new OpenCode session with skill context",
	Long: `Spawn a new OpenCode session with skill context.

By default, spawns the agent headlessly via HTTP API (no TUI) and returns immediately.
Use --inline to run in the current terminal (blocking with TUI).

Model aliases: opus, sonnet, haiku (Anthropic), flash, pro (Google)
Full format: provider/model (e.g., anthropic/claude-opus-4-5-20251101)

Examples:
  orch-go spawn investigation "explore the codebase"           # Default: headless
  orch-go spawn feature-impl "add new spawn command" --phases implementation,validation
  orch-go spawn --issue proj-123 feature-impl "implement the feature"
  orch-go spawn --inline investigation "explore codebase"      # Run inline (blocking TUI)
  orch-go spawn --model opus investigation "explore the codebase"  # Use Claude Opus
  orch-go spawn --model flash investigation "explore the codebase"  # Use Gemini Flash
  orch-go spawn --no-track investigation "exploratory work"    # Skip beads tracking
  orch-go spawn --mcp playwright feature-impl "add UI feature" # With Playwright MCP
  orch-go spawn --skip-artifact-check investigation "fresh start"  # Skip kb context check`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		task := strings.Join(args[1:], " ")

		return runSpawnWithSkill(serverURL, skillName, task, spawnInline)
	},
}

func init() {
	spawnCmd.Flags().StringVar(&spawnIssue, "issue", "", "Beads issue ID for tracking")
	spawnCmd.Flags().StringVar(&spawnPhases, "phases", "", "Feature-impl phases (e.g., implementation,validation)")
	spawnCmd.Flags().StringVar(&spawnMode, "mode", "tdd", "Implementation mode: tdd or direct")
	spawnCmd.Flags().StringVar(&spawnValidation, "validation", "tests", "Validation level: none, tests, smoke-test")
	spawnCmd.Flags().BoolVar(&spawnInline, "inline", false, "Run inline (blocking) with TUI")
	spawnCmd.Flags().StringVar(&spawnModel, "model", "", "Model alias (opus, sonnet, haiku, flash, pro) or provider/model format")
	spawnCmd.Flags().BoolVar(&spawnNoTrack, "no-track", false, "Opt-out of beads issue tracking (ad-hoc work)")
	spawnCmd.Flags().StringVar(&spawnMCP, "mcp", "", "MCP server config (e.g., 'playwright' for browser automation)")
	spawnCmd.Flags().BoolVar(&spawnSkipArtifactCheck, "skip-artifact-check", false, "Bypass pre-spawn kb context check")
}

var askCmd = &cobra.Command{
	Use:   "ask [session-id] [prompt]",
	Short: "Send a message to an existing session (alias for send)",
	Long:  "Send a message to an existing OpenCode session. This is an alias for the 'send' command.",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		prompt := strings.Join(args[1:], " ")
		return runSend(serverURL, sessionID, prompt)
	},
}

var sendCmd = &cobra.Command{
	Use:   "send [session-id] [message]",
	Short: "Send a message to an existing session",
	Long: `Send a message to an existing OpenCode session.

The session can be running or completed. Response text is streamed to stdout
as it's received from the agent.

Examples:
  orch-go send ses_abc123 "what files did you modify?"
  orch-go send ses_xyz789 "can you explain the changes?"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		message := strings.Join(args[1:], " ")
		return runSend(serverURL, sessionID, message)
	},
}

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor SSE events for session completion",
	Long:  "Monitor the OpenCode server for session events and send notifications on completion.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMonitor(serverURL)
	},
}

var (
	// Status command flags
	statusJSON bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show swarm status and active agents",
	Long: `Show swarm status including active/queued/completed agent counts,
per-account usage percentages, and individual agent details.

Examples:
  orch-go status              # Show swarm status with agent details
  orch-go status --json       # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(serverURL)
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON for scripting")
}

var (
	// Complete command flags
	completeForce  bool
	completeReason string
)

var completeCmd = &cobra.Command{
	Use:   "complete [beads-id]",
	Short: "Complete an agent and close the beads issue",
	Long: `Complete an agent's work by verifying Phase: Complete and closing the beads issue.

Checks that the agent has reported "Phase: Complete" via beads comments before
closing the issue. Use --force to skip phase verification.

Examples:
  orch-go complete proj-123
  orch-go complete proj-123 --reason "All tests passing"
  orch-go complete proj-123 --force  # Skip phase verification`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runComplete(beadsID)
	},
}

func init() {
	completeCmd.Flags().BoolVarP(&completeForce, "force", "f", false, "Skip phase verification")
	completeCmd.Flags().StringVarP(&completeReason, "reason", "r", "", "Reason for closing (default: uses phase summary)")
}

var (
	// Work command flags
	workInline bool // Run inline (blocking) with TUI
)

var workCmd = &cobra.Command{
	Use:   "work [beads-id]",
	Short: "Start work on a beads issue with skill inference",
	Long: `Start work on a beads issue by inferring the skill from the issue type.

The skill is automatically determined from the issue type:
  - bug         → systematic-debugging
  - feature     → feature-impl
  - task        → feature-impl
  - investigation → investigation

The issue description becomes the task prompt for the spawned agent.

By default, spawns headlessly via HTTP API (no TUI).
Use --inline to run in the current terminal (blocking with TUI).

Examples:
  orch-go work proj-123           # Start work headlessly (default)
  orch-go work proj-123 --inline  # Start work inline (blocking TUI)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runWork(serverURL, beadsID, workInline)
	},
}

func init() {
	workCmd.Flags().BoolVar(&workInline, "inline", false, "Run inline (blocking) with TUI")
}

var (
	// Tail command flags
	tailLines int

	// Send command flags
	sendAsync bool
)

var tailCmd = &cobra.Command{
	Use:   "tail [beads-id]",
	Short: "Capture recent output from an agent",
	Long: `Capture recent output from an agent for debugging.

Fetches messages from the OpenCode API for the agent's session.

Examples:
  orch-go tail proj-123              # Capture last 50 lines (default)
  orch-go tail proj-123 --lines 100  # Capture last 100 lines
  orch-go tail proj-123 -n 20        # Capture last 20 lines`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runTail(beadsID, tailLines)
	},
}

func init() {
	tailCmd.Flags().IntVarP(&tailLines, "lines", "n", 50, "Number of lines to capture")
}

func runTail(beadsID string, lines int) error {
	// Find the agent in the registry
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}

	agent := reg.Find(beadsID)
	if agent == nil {
		return fmt.Errorf("no agent found for beads ID: %s", beadsID)
	}

	if agent.SessionID == "" {
		return fmt.Errorf("agent %s has no session ID - cannot fetch via API", agent.ID)
	}

	client := opencode.NewClient(serverURL)

	// Fetch messages from the session
	messages, err := client.GetMessages(agent.SessionID)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	if len(messages) == 0 {
		fmt.Println("No messages found in session")
		return nil
	}

	// Extract recent text from messages
	textLines := opencode.ExtractRecentText(messages, lines)

	// Print the captured output
	fmt.Printf("=== Output from %s (last %d lines) ===\n", agent.ID, lines)
	for _, line := range textLines {
		fmt.Println(line)
	}
	fmt.Printf("=== End of output ===\n")

	return nil
}

var questionCmd = &cobra.Command{
	Use:   "question [beads-id]",
	Short: "Extract pending question from an agent's session",
	Long: `Extract pending question from an agent's session.

Finds the OpenCode session associated with the beads issue ID and extracts
any pending question the agent is asking. Useful for monitoring agents
that are blocked waiting for user input.

Examples:
  orch-go question proj-123  # Extract question from agent's session`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runQuestion(beadsID)
	},
}

func runQuestion(beadsID string) error {
	// Find agent in registry
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}

	agent := reg.Find(beadsID)
	if agent == nil {
		return fmt.Errorf("no agent found for beads ID: %s", beadsID)
	}

	if agent.SessionID == "" {
		return fmt.Errorf("agent %s has no session ID - cannot extract question", agent.ID)
	}

	// Fetch recent messages from the session
	client := opencode.NewClient(serverURL)
	messages, err := client.GetMessages(agent.SessionID)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	if len(messages) == 0 {
		fmt.Println("No messages found in session")
		return nil
	}

	// Extract recent text content from messages to search for questions
	textLines := opencode.ExtractRecentText(messages, 100)
	content := strings.Join(textLines, "\n")

	// Extract question from content
	q := question.Extract(content)
	if q == "" {
		fmt.Println("No pending question found")
		return nil
	}

	fmt.Printf("Pending question:\n%s\n", q)
	return nil
}

var abandonCmd = &cobra.Command{
	Use:   "abandon [beads-id]",
	Short: "Abandon a stuck or frozen agent",
	Long: `Abandon an agent and mark it abandoned in the registry.

Use this command for stuck or frozen agents that are not responding.
The agent's beads issue is NOT closed - you can restart work with 'orch work'.

Examples:
  orch-go abandon proj-123           # Abandon agent for issue proj-123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runAbandon(beadsID)
	},
}

func runAbandon(beadsID string) error {
	// Open the registry
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}

	// Find agent by beads ID
	agent := reg.Find(beadsID)
	if agent == nil {
		return fmt.Errorf("no agent found for beads ID: %s", beadsID)
	}

	// Check if already abandoned or completed
	if agent.Status != registry.StateActive {
		return fmt.Errorf("agent %s is not active (status: %s)", agent.ID, agent.Status)
	}

	// Mark agent as abandoned in registry
	if !reg.Abandon(agent.ID) {
		return fmt.Errorf("failed to mark agent as abandoned")
	}

	// Save the registry
	if err := reg.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	// Log the abandonment
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agent.abandoned",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id":  beadsID,
			"agent_id":  agent.ID,
			"window_id": agent.WindowID,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Abandoned agent: %s\n", agent.ID)
	fmt.Printf("  Beads ID: %s\n", beadsID)
	fmt.Printf("  Use 'orch work %s' to restart work on this issue\n", beadsID)

	return nil
}

// InferSkillFromIssueType maps issue types to appropriate skills.
// Returns an error for types that cannot be spawned (e.g., epic) or unknown types.
func InferSkillFromIssueType(issueType string) (string, error) {
	switch issueType {
	case "bug":
		return "systematic-debugging", nil
	case "feature":
		return "feature-impl", nil
	case "task":
		return "feature-impl", nil
	case "investigation":
		return "investigation", nil
	case "epic":
		return "", fmt.Errorf("cannot spawn work on epic issues - epics are decomposed into sub-issues")
	case "":
		return "", fmt.Errorf("issue type is empty")
	default:
		return "", fmt.Errorf("unknown issue type: %s", issueType)
	}
}

func runWork(serverURL, beadsID string, inline bool) error {
	// Get issue details
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Infer skill from issue type
	skillName, err := InferSkillFromIssueType(issue.IssueType)
	if err != nil {
		return fmt.Errorf("cannot work on issue %s: %w", beadsID, err)
	}

	// Use issue title as the task (description is often longer form context)
	task := issue.Title

	// Set the spawnIssue flag so runSpawnWithSkill uses the existing issue
	spawnIssue = beadsID

	fmt.Printf("Starting work on: %s\n", beadsID)
	fmt.Printf("  Title:  %s\n", issue.Title)
	fmt.Printf("  Type:   %s\n", issue.IssueType)
	fmt.Printf("  Skill:  %s\n", skillName)

	return runSpawnWithSkill(serverURL, skillName, task, inline)
}

func runSpawnWithSkill(serverURL, skillName, task string, inline bool) error {
	// Get current directory as project dir
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get project name from directory
	projectName := filepath.Base(projectDir)

	// Generate workspace name
	workspaceName := spawn.GenerateWorkspaceName(skillName, task)

	// Load skill content
	loader := skills.DefaultLoader()
	skillContent, err := loader.LoadSkillContent(skillName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load skill '%s': %v\n", skillName, err)
		skillContent = "" // Continue without skill content
	}

	// Determine beads ID - either from flag, create new issue, or skip if --no-track
	beadsID := spawnIssue
	if beadsID == "" && !spawnNoTrack {
		// Create a new beads issue (default behavior)
		beadsID, err = createBeadsIssue(projectName, skillName, task)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create beads issue: %v\n", err)
			beadsID = fmt.Sprintf("%s-%d", projectName, time.Now().Unix()) // Fallback ID
		}
	} else if spawnNoTrack {
		// Generate a local-only ID for untracked work
		beadsID = fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix())
		fmt.Println("Skipping beads tracking (--no-track)")
	}

	// Update beads issue status to in_progress (only if tracking a real issue)
	if !spawnNoTrack && spawnIssue != "" {
		if err := verify.UpdateIssueStatus(beadsID, "in_progress"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
			// Continue anyway
		}
	}

	// Resolve model - convert aliases to full format
	resolvedModel := model.Resolve(spawnModel)

	// Build spawn config
	cfg := &spawn.Config{
		Task:              task,
		SkillName:         skillName,
		Project:           projectName,
		ProjectDir:        projectDir,
		WorkspaceName:     workspaceName,
		SkillContent:      skillContent,
		BeadsID:           beadsID,
		Phases:            spawnPhases,
		Mode:              spawnMode,
		Validation:        spawnValidation,
		Model:             resolvedModel.Format(),
		MCP:               spawnMCP,
		NoTrack:           spawnNoTrack,
		SkipArtifactCheck: spawnSkipArtifactCheck,
	}

	// Write SPAWN_CONTEXT.md
	if err := spawn.WriteContext(cfg); err != nil {
		return fmt.Errorf("failed to write spawn context: %w", err)
	}

	// Generate minimal prompt
	minimalPrompt := spawn.MinimalPrompt(cfg)

	// Spawn mode: inline (blocking with TUI) or headless (HTTP API, no TUI)
	if inline {
		// Inline mode (blocking) - run in current terminal with TUI
		return runSpawnInline(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Default: Headless mode - spawn via HTTP API (no TUI)
	return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}

// runSpawnInline spawns the agent inline (blocking) - original behavior.
func runSpawnInline(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	// Spawn opencode session
	client := opencode.NewClient(serverURL)
	cmd := client.BuildSpawnCommand(minimalPrompt, cfg.WorkspaceName)
	cmd.Stderr = os.Stderr
	cmd.Dir = cfg.ProjectDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	result, err := opencode.ProcessOutput(stdout)
	if err != nil {
		return fmt.Errorf("failed to process output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("opencode exited with error: %w", err)
	}

	// Register agent in persistent registry (inline agents have no window_id)
	reg, err := registry.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open registry: %v\n", err)
	} else {
		agent := &registry.Agent{
			ID:         cfg.WorkspaceName,
			BeadsID:    beadsID,
			ProjectDir: cfg.ProjectDir,
			Skill:      skillName,
		}
		if err := reg.Register(agent); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to register agent: %v\n", err)
		} else if err := reg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save registry: %v\n", err)
		}
	}

	// Log the session creation
	inlineLogger := events.NewLogger(events.DefaultLogPath())
	inlineEventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"spawn_mode":          "inline",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		inlineEventData["mcp"] = cfg.MCP
	}
	inlineEvent := events.Event{
		Type:      "session.spawned",
		SessionID: result.SessionID,
		Timestamp: time.Now().Unix(),
		Data:      inlineEventData,
	}
	if err := inlineLogger.Log(inlineEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary
	fmt.Printf("Spawned agent:\n")
	fmt.Printf("  Session ID: %s\n", result.SessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	fmt.Printf("  Context:    %s\n", cfg.ContextFilePath())

	return nil
}

// runSpawnHeadless spawns the agent via HTTP API without a TUI.
// This is useful for automation and daemon-driven spawns.
// The agent is registered with window_id='headless' for tracking.
func runSpawnHeadless(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	client := opencode.NewClient(serverURL)

	// Create session via HTTP API
	sessionResp, err := client.CreateSession(cfg.WorkspaceName, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Send the prompt to start the agent
	if err := client.SendPrompt(sessionResp.ID, minimalPrompt); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}

	// Register agent in persistent registry with window_id='headless' and session_id
	reg, err := registry.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open registry: %v\n", err)
	} else {
		agent := &registry.Agent{
			ID:         cfg.WorkspaceName,
			BeadsID:    beadsID,
			SessionID:  sessionResp.ID,            // Track OpenCode session ID for headless agents
			WindowID:   registry.HeadlessWindowID, // Special marker for headless spawns
			ProjectDir: cfg.ProjectDir,
			Skill:      skillName,
		}
		if err := reg.Register(agent); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to register agent: %v\n", err)
		} else if err := reg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save registry: %v\n", err)
		}
	}

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"session_id":          sessionResp.ID,
		"spawn_mode":          "headless",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionResp.ID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary
	fmt.Printf("Spawned agent (headless):\n")
	fmt.Printf("  Session ID: %s\n", sessionResp.ID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	fmt.Printf("  Context:    %s\n", cfg.ContextFilePath())

	return nil
}

// createBeadsIssue creates a new beads issue for tracking the agent.
func createBeadsIssue(projectName, skillName, task string) (string, error) {
	// Build issue title
	title := fmt.Sprintf("[%s] %s: %s", projectName, skillName, truncate(task, 50))

	// Run bd create command
	cmd := exec.Command("bd", "create", title)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bd create failed: %w", err)
	}

	// Parse issue ID from output (search all lines for "issue: <id>")
	outputStr := strings.TrimSpace(string(output))
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for "issue:" in the line and extract the ID after it
		parts := strings.Fields(line)
		for i, part := range parts {
			if strings.Contains(part, "issue:") {
				// Issue ID should be the next word after "issue:"
				if i+1 < len(parts) {
					return parts[i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not parse issue ID from: %s", outputStr)
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func runSend(serverURL, sessionID, message string) error {
	client := opencode.NewClient(serverURL)

	// Log the send event first (before streaming starts)
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.send",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message": message,
			"async":   sendAsync,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	if sendAsync {
		// Send message asynchronously (non-blocking)
		if err := client.SendMessageAsync(sessionID, message); err != nil {
			return fmt.Errorf("failed to send message asynchronously: %w", err)
		}
		fmt.Printf("✓ Message sent to session %s\n", sessionID)
		return nil
	}

	// Send message and stream the response to stdout (blocking)
	if err := client.SendMessageWithStreaming(sessionID, message, os.Stdout); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Add newline at end for clean output
	fmt.Println()

	return nil
}

// SwarmStatus represents aggregate swarm information.
type SwarmStatus struct {
	Active    int `json:"active"`
	Queued    int `json:"queued"`
	Completed int `json:"completed_today"`
}

// AccountUsage represents usage info for a single account.
type AccountUsage struct {
	Name        string  `json:"name"`
	Email       string  `json:"email,omitempty"`
	UsedPercent float64 `json:"used_percent"`
	ResetTime   string  `json:"reset_time,omitempty"`
	IsActive    bool    `json:"is_active"`
}

// AgentInfo represents information about an active agent.
type AgentInfo struct {
	SessionID string `json:"session_id"`
	BeadsID   string `json:"beads_id,omitempty"`
	Skill     string `json:"skill,omitempty"`
	Account   string `json:"account,omitempty"`
	Runtime   string `json:"runtime"`
	Title     string `json:"title,omitempty"`
}

// StatusOutput represents the full status output for JSON serialization.
type StatusOutput struct {
	Swarm    SwarmStatus    `json:"swarm"`
	Accounts []AccountUsage `json:"accounts"`
	Agents   []AgentInfo    `json:"agents"`
}

func runStatus(serverURL string) error {
	client := opencode.NewClient(serverURL)

	// Fetch sessions from OpenCode
	sessions, err := client.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	// Open registry to get agent metadata
	reg, regErr := registry.New("")
	if regErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open registry: %v\n", regErr)
	}

	// Get active agents from registry for metadata
	var regAgents []*registry.Agent
	if reg != nil {
		regAgents = reg.ListActive()
	}

	// Build agent lookup by session ID
	agentBySession := make(map[string]*registry.Agent)
	for _, a := range regAgents {
		if a.SessionID != "" {
			agentBySession[a.SessionID] = a
		}
	}

	// Build agents list
	now := time.Now()
	agents := make([]AgentInfo, 0, len(sessions))
	for _, s := range sessions {
		agent := AgentInfo{
			SessionID: s.ID,
			Title:     s.Title,
		}

		// Calculate runtime from session creation
		createdAt := time.Unix(s.Time.Created/1000, 0)
		runtime := now.Sub(createdAt)
		agent.Runtime = formatDuration(runtime)

		// Enrich with registry metadata if available
		if regAgent, ok := agentBySession[s.ID]; ok {
			agent.BeadsID = regAgent.BeadsID
			agent.Skill = regAgent.Skill
		}

		agents = append(agents, agent)
	}

	// Count completed today from registry
	completedToday := 0
	if reg != nil {
		today := time.Now().Truncate(24 * time.Hour)
		for _, a := range reg.ListCompleted() {
			if a.CompletedAt != "" {
				completedTime, err := time.Parse(registry.TimeFormat, a.CompletedAt)
				if err == nil && completedTime.After(today) {
					completedToday++
				}
			}
		}
	}

	// Build swarm status
	swarm := SwarmStatus{
		Active:    len(sessions),
		Queued:    0, // TODO: implement queuing system
		Completed: completedToday,
	}

	// Fetch account usage information
	accounts := getAccountUsage()

	// Build output
	output := StatusOutput{
		Swarm:    swarm,
		Accounts: accounts,
		Agents:   agents,
	}

	// Output as JSON if flag is set
	if statusJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Print human-readable output
	printSwarmStatus(output)
	return nil
}

// getAccountUsage fetches usage info for all configured accounts.
func getAccountUsage() []AccountUsage {
	var accounts []AccountUsage

	// Get current account usage
	currentUsage := usage.FetchUsage()
	if currentUsage.Error == "" && currentUsage.SevenDay != nil {
		current := AccountUsage{
			Name:        "current",
			Email:       currentUsage.Email,
			UsedPercent: currentUsage.SevenDay.Utilization,
			IsActive:    true,
		}
		if currentUsage.SevenDay.ResetsAt != nil {
			current.ResetTime = currentUsage.SevenDay.TimeUntilReset()
		}
		accounts = append(accounts, current)
	}

	// Try to get saved accounts info (without switching)
	cfg, err := account.LoadConfig()
	if err == nil {
		for name, acc := range cfg.Accounts {
			if acc.Source == "saved" {
				// Check if this is the current account (by email match)
				isCurrentAccount := false
				for i := range accounts {
					if accounts[i].Email == acc.Email {
						accounts[i].Name = name // Update name to the saved account name
						isCurrentAccount = true
						break
					}
				}
				if !isCurrentAccount {
					// Add as a saved account (no live usage data without switching)
					accounts = append(accounts, AccountUsage{
						Name:     name,
						Email:    acc.Email,
						IsActive: false,
					})
				}
			}
		}
	}

	return accounts
}

// printSwarmStatus prints the swarm status in human-readable format.
func printSwarmStatus(output StatusOutput) {
	// Print swarm summary
	fmt.Println("SWARM STATUS")
	fmt.Printf("  Active:    %d\n", output.Swarm.Active)
	if output.Swarm.Queued > 0 {
		fmt.Printf("  Queued:    %d\n", output.Swarm.Queued)
	}
	fmt.Printf("  Completed: %d (today)\n", output.Swarm.Completed)
	fmt.Println()

	// Print account usage
	if len(output.Accounts) > 0 {
		fmt.Println("ACCOUNTS")
		for _, acc := range output.Accounts {
			activeMarker := ""
			if acc.IsActive {
				activeMarker = " *"
			}
			usageStr := "N/A"
			if acc.UsedPercent > 0 || acc.IsActive {
				usageStr = fmt.Sprintf("%.0f%% used", acc.UsedPercent)
				if acc.ResetTime != "" {
					usageStr += fmt.Sprintf(" (resets in %s)", acc.ResetTime)
				}
			}
			name := acc.Name
			if acc.Email != "" && acc.Name == "current" {
				name = acc.Email
			}
			fmt.Printf("  %-20s %s%s\n", name+":", usageStr, activeMarker)
		}
		fmt.Println()
	}

	// Print active agents table
	if len(output.Agents) > 0 {
		fmt.Println("ACTIVE AGENTS")
		fmt.Printf("  %-35s %-15s %-20s %-10s %s\n", "SESSION ID", "BEADS ID", "SKILL", "ACCOUNT", "RUNTIME")
		fmt.Printf("  %s\n", strings.Repeat("-", 90))

		for _, agent := range output.Agents {
			beadsID := agent.BeadsID
			if beadsID == "" {
				beadsID = "-"
			}
			skill := agent.Skill
			if skill == "" {
				skill = "-"
			}
			accountName := agent.Account
			if accountName == "" {
				accountName = "-"
			}

			fmt.Printf("  %-35s %-15s %-20s %-10s %s\n",
				truncate(agent.SessionID, 33),
				truncate(beadsID, 13),
				truncate(skill, 18),
				truncate(accountName, 8),
				agent.Runtime)
		}
	} else {
		fmt.Println("No active agents")
	}
}

func runComplete(beadsID string) error {
	// Get issue to verify it exists
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Check if already closed
	isClosed := issue.Status == "closed"
	if isClosed {
		fmt.Printf("Issue %s is already closed in beads\n", beadsID)
	}

	// Verify phase status unless force flag is set
	if !completeForce {
		// Find agent in registry to get workspace path for SYNTHESIS.md verification
		var workspacePath string
		reg, err := registry.New("")
		if err == nil {
			agent := reg.Find(beadsID)
			if agent != nil && agent.ProjectDir != "" {
				workspacePath = filepath.Join(agent.ProjectDir, ".orch", "workspace", agent.ID)
			}
		}

		result, err := verify.VerifyCompletion(beadsID, workspacePath)
		if err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}

		if !result.Passed {
			fmt.Fprintf(os.Stderr, "Cannot complete agent - verification failed:\n")
			for _, e := range result.Errors {
				fmt.Fprintf(os.Stderr, "  - %s\n", e)
			}
			fmt.Fprintf(os.Stderr, "\nAgent must run: bd comment %s \"Phase: Complete - <summary>\"\n", beadsID)
			fmt.Fprintf(os.Stderr, "Or use --force to skip verification\n")
			return fmt.Errorf("verification failed")
		}

		// Print phase info
		if result.Phase.Found {
			fmt.Printf("Phase: %s\n", result.Phase.Phase)
			if result.Phase.Summary != "" {
				fmt.Printf("Summary: %s\n", result.Phase.Summary)
			}
		}
	} else {
		fmt.Println("Skipping phase verification (--force)")
	}

	// Determine close reason
	reason := completeReason
	if reason == "" {
		// Try to get summary from phase status
		status, _ := verify.GetPhaseStatus(beadsID)
		if status.Summary != "" {
			reason = status.Summary
		} else {
			reason = "Completed via orch complete"
		}
	}

	// Close the beads issue if not already closed
	if !isClosed {
		if err := verify.CloseIssue(beadsID, reason); err != nil {
			return fmt.Errorf("failed to close issue: %w", err)
		}
		fmt.Printf("Closed beads issue: %s\n", beadsID)
	}
	fmt.Printf("Reason: %s\n", reason)

	// Update registry to mark agent as completed
	reg, err := registry.New("")
	if err == nil {
		agent := reg.Find(beadsID)
		if agent != nil {
			// Mark agent as completed in registry
			if reg.Complete(agent.ID) {
				if err := reg.Save(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to save registry: %v\n", err)
				}
			}
		}
	}

	// Log the completion
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agent.completed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id": beadsID,
			"reason":   reason,
			"forced":   completeForce,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	return nil
}

func runMonitor(serverURL string) error {
	// Use the new CompletionService which handles:
	// - SSE monitoring with automatic reconnection
	// - Desktop notifications
	// - Registry updates
	// - Beads phase updates
	service, err := opencode.NewCompletionService(serverURL)
	if err != nil {
		return fmt.Errorf("failed to create completion service: %w", err)
	}

	fmt.Printf("Monitoring SSE events at %s/event...\n", serverURL)
	fmt.Println("On session completion:")
	fmt.Println("  - Desktop notification sent")
	fmt.Println("  - Registry updated")
	fmt.Println("  - Beads phase updated (if applicable)")
	fmt.Println("Press Ctrl+C to stop")

	service.Start()

	// Block forever - the user will Ctrl+C to stop
	select {}
}

var (
	// Clean command flags
	cleanDryRun bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove completed agents from the registry",
	Long: `Remove completed and abandoned agents from the registry.

By default, only cleans agents that are marked as completed or abandoned in the registry.

Examples:
  orch-go clean              # Clean completed/abandoned agents
  orch-go clean --dry-run    # Show what would be cleaned`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClean(cleanDryRun)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
}

func runClean(dryRun bool) error {
	// Open registry
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}

	// Get cleanable agents (completed or abandoned)
	agents := reg.ListCleanable()

	if len(agents) == 0 {
		fmt.Println("No agents to clean")
		return nil
	}

	// Track cleanup stats
	agentsCleaned := 0

	fmt.Printf("Found %d agents to clean:\n", len(agents))

	for _, agent := range agents {
		status := string(agent.Status)

		if dryRun {
			fmt.Printf("  [DRY-RUN] Would clean: %s [%s]\n", agent.ID, status)
			continue
		}

		fmt.Printf("  Cleaning: %s [%s]\n", agent.ID, status)

		// Mark agent as deleted in registry
		if reg.Remove(agent.ID) {
			agentsCleaned++
		}
	}

	if dryRun {
		fmt.Printf("\nDry run complete. Would clean %d agents.\n", len(agents))
		return nil
	}

	// Save registry
	if err := reg.SaveSkipMerge(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	// Get current directory for logging
	projectDir, _ := os.Getwd()
	projectName := filepath.Base(projectDir)

	// Log the cleanup
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agents.cleaned",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"agents_cleaned": agentsCleaned,
			"project":        projectName,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("\nCleaned %d agents\n", agentsCleaned)
	return nil
}

// ============================================================================
// Account Management
// ============================================================================

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage Claude Max accounts",
	Long: `Manage multiple Claude Max accounts for usage tracking and rate limit arbitrage.

Accounts are stored in ~/.orch/accounts.yaml with refresh tokens for switching.

Examples:
  orch-go account list              # List all saved accounts
  orch-go account switch personal   # Switch to 'personal' account
  orch-go account remove work       # Remove 'work' account`,
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved accounts",
	Long:  "List all saved accounts with their email and default status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountList()
	},
}

var accountSwitchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Switch to a saved account",
	Long: `Switch to a saved account by refreshing its OAuth tokens.

This updates the OpenCode auth file with new tokens from the saved refresh token.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountSwitch(args[0])
	},
}

var accountRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a saved account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountRemove(args[0])
	},
}

func init() {
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountSwitchCmd)
	accountCmd.AddCommand(accountRemoveCmd)
}

func runAccountList() error {
	accounts, err := account.ListAccountInfo()
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("No saved accounts")
		fmt.Println("\nTo save the current account:")
		fmt.Println("  1. Login via OpenCode first")
		fmt.Println("  2. Use Python orch: orch account save <name>")
		return nil
	}

	fmt.Printf("%-15s %-35s %-10s\n", "NAME", "EMAIL", "DEFAULT")
	fmt.Printf("%s\n", strings.Repeat("-", 65))

	for _, acc := range accounts {
		def := ""
		if acc.IsDefault {
			def = "✓"
		}
		fmt.Printf("%-15s %-35s %-10s\n", acc.Name, acc.Email, def)
	}

	return nil
}

func runAccountSwitch(name string) error {
	email, err := account.SwitchAccount(name)
	if err != nil {
		return err
	}

	fmt.Printf("Switched to account: %s (%s)\n", name, email)
	return nil
}

func runAccountRemove(name string) error {
	cfg, err := account.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}

	if err := cfg.Remove(name); err != nil {
		return fmt.Errorf("failed to remove account: %w", err)
	}

	if err := account.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed account: %s\n", name)
	return nil
}

// ============================================================================
// Usage Tracking (Placeholder - defers to Python for now)
// ============================================================================

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show Claude Max usage limits",
	Long: `Show Claude Max weekly usage limits.

Reads OAuth token from OpenCode's auth.json and fetches usage data
from the Anthropic API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUsage()
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}

func runUsage() error {
	info := usage.FetchUsage()
	fmt.Println(usage.FormatDisplay(info))
	return nil
}
