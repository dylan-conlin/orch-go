// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/notify"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/question"
	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/usage"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	serverURL string

	// Version information (set at build time)
	version = "dev"
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
}

var (
	// Spawn command flags
	spawnSkill             string
	spawnIssue             string
	spawnPhases            string
	spawnMode              string
	spawnValidation        string
	spawnInline            bool   // Run inline (blocking) instead of in tmux
	spawnModel             string // Model to use for standalone spawns
	spawnNoTrack           bool   // Opt-out of beads tracking
	spawnMCP               string // MCP server config (e.g., "playwright")
	spawnSkipArtifactCheck bool   // Bypass pre-spawn kb context check
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [skill] [task]",
	Short: "Spawn a new OpenCode session with skill context",
	Long: `Spawn a new OpenCode session with skill context.

By default, spawns the agent in a tmux window and returns immediately.
Use --inline to run in the current terminal (blocking).

Model aliases: opus, sonnet, haiku (Anthropic), flash, pro (Google)
Full format: provider/model (e.g., anthropic/claude-opus-4-5-20251101)

Examples:
  orch-go spawn investigation "explore the codebase"
  orch-go spawn feature-impl "add new spawn command" --phases implementation,validation
  orch-go spawn --issue proj-123 feature-impl "implement the feature"
  orch-go spawn --inline investigation "explore codebase"  # Run inline (blocking)
  orch-go spawn --model opus investigation "explore the codebase"  # Use Claude Opus
  orch-go spawn --model flash investigation "explore the codebase"  # Use Gemini Flash
  orch-go spawn --no-track investigation "exploratory work"  # Skip beads tracking
  orch-go spawn --mcp playwright feature-impl "add UI feature"  # With Playwright MCP
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
	spawnCmd.Flags().BoolVar(&spawnInline, "inline", false, "Run inline (blocking) instead of in tmux")
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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "List active OpenCode sessions",
	Long: `List all active OpenCode sessions with their status.

Shows session ID, workspace/title, directory, and last update time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(serverURL)
	},
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
	workInline bool // Run inline (blocking) instead of in tmux
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

Examples:
  orch-go work proj-123           # Start work on issue proj-123 in tmux
  orch-go work proj-123 --inline  # Start work inline (blocking)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runWork(serverURL, beadsID, workInline)
	},
}

func init() {
	workCmd.Flags().BoolVar(&workInline, "inline", false, "Run inline (blocking) instead of in tmux")
}

var (
	// Tail command flags
	tailLines int
)

var tailCmd = &cobra.Command{
	Use:   "tail [beads-id]",
	Short: "Capture recent output from an agent's tmux window",
	Long: `Capture recent output from an agent's tmux window for debugging stuck agents.

Finds the tmux window associated with the beads issue ID and captures
the last N lines of output.

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
	// Get current directory to determine project name
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)

	// Get workers session name for this project
	sessionName := tmux.GetWorkersSessionName(projectName)

	// Check if session exists
	if !tmux.SessionExists(sessionName) {
		return fmt.Errorf("no workers session found for project %s (expected: %s)", projectName, sessionName)
	}

	// Find window by beads ID
	window, err := tmux.FindWindowByBeadsID(sessionName, beadsID)
	if err != nil {
		return fmt.Errorf("failed to find window: %w", err)
	}
	if window == nil {
		return fmt.Errorf("no window found for beads ID: %s", beadsID)
	}

	// Capture lines from the window
	output, err := tmux.CaptureLines(window.Target, lines)
	if err != nil {
		return fmt.Errorf("failed to capture output: %w", err)
	}

	// Print the captured output
	fmt.Printf("=== Output from %s (last %d lines) ===\n", window.Name, lines)
	for _, line := range output {
		fmt.Println(line)
	}
	fmt.Printf("=== End of output ===\n")

	return nil
}

var questionCmd = &cobra.Command{
	Use:   "question [beads-id]",
	Short: "Extract pending question from an agent's tmux window",
	Long: `Extract pending question from an agent's tmux window.

Finds the tmux window associated with the beads issue ID and extracts
any pending question the agent is asking. Useful for monitoring agents
that are blocked waiting for user input.

Examples:
  orch-go question proj-123  # Extract question from agent's window`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runQuestion(beadsID)
	},
}

func runQuestion(beadsID string) error {
	// Get current directory to determine project name
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)

	// Get workers session name for this project
	sessionName := tmux.GetWorkersSessionName(projectName)

	// Check if session exists
	if !tmux.SessionExists(sessionName) {
		return fmt.Errorf("no workers session found for project %s (expected: %s)", projectName, sessionName)
	}

	// Find window by beads ID
	window, err := tmux.FindWindowByBeadsID(sessionName, beadsID)
	if err != nil {
		return fmt.Errorf("failed to find window: %w", err)
	}
	if window == nil {
		return fmt.Errorf("no window found for beads ID: %s", beadsID)
	}

	// Capture pane content (full visible content)
	content, err := tmux.GetPaneContent(window.Target)
	if err != nil {
		return fmt.Errorf("failed to capture pane content: %w", err)
	}

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
	Long: `Abandon an agent by killing its tmux window and marking it abandoned in the registry.

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

	// Kill the tmux window if it has one
	if agent.WindowID != "" {
		if err := tmux.KillWindowByID(agent.WindowID); err != nil {
			// Window might already be gone, just warn
			fmt.Fprintf(os.Stderr, "Warning: could not kill window %s: %v\n", agent.WindowID, err)
		} else {
			fmt.Printf("Killed tmux window: %s\n", agent.Window)
		}
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

	// Decide spawn mode: tmux (default) or inline
	useTmux := !inline && tmux.IsAvailable()

	if useTmux {
		return runSpawnInTmux(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Inline mode (blocking) - original behavior
	return runSpawnInline(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}

// runSpawnInTmux spawns the agent in a tmux window and returns immediately.
func runSpawnInTmux(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	// Ensure workers session exists
	sessionName, err := tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to ensure workers session: %w", err)
	}

	// Build window name
	windowName := tmux.BuildWindowName(cfg.WorkspaceName, skillName, beadsID)

	// Create window
	windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}

	// Build opencode command using 'run' mode
	// This is more robust as it handles model selection and initial prompt directly
	runCfg := &tmux.RunConfig{
		ProjectDir: cfg.ProjectDir,
		Model:      cfg.Model,
		Title:      cfg.WorkspaceName,
		Prompt:     minimalPrompt,
	}

	cmd := tmux.BuildRunCommand(runCfg)

	// Ensure the command is properly quoted for tmux send-keys
	var args []string
	for _, arg := range cmd.Args {
		if strings.Contains(arg, " ") {
			args = append(args, fmt.Sprintf("\"%s\"", arg))
		} else {
			args = append(args, arg)
		}
	}
	opencodeCmd := strings.Join(args, " ")

	// If using Gemini, ensure we don't have a stale Anthropic key in the window
	if strings.Contains(cfg.Model, "gemini") {
		opencodeCmd = "unset ANTHROPIC_API_KEY && " + opencodeCmd
	}

	// Send the command to the tmux window
	if err := tmux.SendKeysLiteral(windowTarget, opencodeCmd); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	// Send Enter to execute
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	// Select the window to focus it
	_ = tmux.SelectWindow(windowTarget)

	// Register agent in persistent registry

	reg, err := registry.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open registry: %v\n", err)
	} else {
		agent := &registry.Agent{
			ID:         cfg.WorkspaceName,
			BeadsID:    beadsID,
			WindowID:   windowID,
			Window:     windowTarget,
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
		"window":              windowTarget,
		"window_id":           windowID,
		"spawn_mode":          "tmux",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary
	fmt.Printf("Spawned agent:\n")
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Window:     %s\n", windowTarget)
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
	cmd := client.BuildAskCommand(sessionID, message)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	// Stream text content to stdout as it arrives
	result, err := opencode.ProcessOutputWithStreaming(stdout, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to process output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("opencode exited with error: %w", err)
	}

	// Log the send
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.send",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message":     message,
			"event_count": len(result.Events),
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print newline after streamed content for cleaner output
	fmt.Println()
	return nil
}

func runStatus(serverURL string) error {
	client := opencode.NewClient(serverURL)
	sessions, err := client.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions")
		return nil
	}

	// Print table header
	fmt.Printf("%-35s %-30s %-40s %s\n", "SESSION ID", "TITLE", "DIRECTORY", "UPDATED")
	fmt.Printf("%s\n", strings.Repeat("-", 130))

	// Print each session
	for _, s := range sessions {
		// Format updated time
		updated := time.Unix(s.Time.Updated/1000, 0).Format("2006-01-02 15:04:05")

		// Truncate long fields
		title := truncate(s.Title, 28)
		dir := truncate(s.Directory, 38)

		fmt.Printf("%-35s %-30s %-40s %s\n", s.ID, title, dir, updated)
	}

	fmt.Printf("\nTotal: %d sessions\n", len(sessions))
	return nil
}

func runComplete(beadsID string) error {
	// Get issue to verify it exists
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Check if already closed
	if issue.Status == "closed" {
		fmt.Printf("Issue %s is already closed\n", beadsID)
		return nil
	}

	// Verify phase status unless force flag is set
	if !completeForce {
		result, err := verify.VerifyCompletion(beadsID)
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

	// Close the beads issue
	if err := verify.CloseIssue(beadsID, reason); err != nil {
		return fmt.Errorf("failed to close issue: %w", err)
	}

	fmt.Printf("Closed beads issue: %s\n", beadsID)
	fmt.Printf("Reason: %s\n", reason)

	// Clean up tmux window and registry
	reg, err := registry.New("")
	if err == nil {
		agent := reg.Find(beadsID)
		if agent != nil {
			// Kill the tmux window if it has one
			if agent.WindowID != "" {
				if err := tmux.KillWindowByID(agent.WindowID); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not kill window %s: %v\n", agent.WindowID, err)
				} else {
					fmt.Printf("Closed tmux window: %s\n", agent.Window)
				}
			}

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
	sseURL := serverURL + "/event"
	sseClient := opencode.NewSSEClient(sseURL)
	apiClient := opencode.NewClient(serverURL)
	notifier := notify.Default()

	fmt.Printf("Monitoring SSE events at %s...\n", sseURL)

	sseEvents := make(chan opencode.SSEEvent, 100)
	errChan := make(chan error, 1)

	go func() {
		if err := sseClient.Connect(sseEvents); err != nil {
			errChan <- err
		}
		close(sseEvents)
	}()

	logger := events.NewLogger(events.DefaultLogPath())
	var sessionEvents []opencode.SSEEvent

	for {
		select {
		case event, ok := <-sseEvents:
			if !ok {
				return nil
			}

			// Log every event
			logEvent := events.Event{
				Type:      event.Event,
				Timestamp: time.Now().Unix(),
				Data:      map[string]interface{}{"raw_data": event.Data},
			}

			// Parse session info if available
			if event.Event == "session.status" || event.Event == "session.created" {
				status, sid := opencode.ParseSessionStatus(event.Data)
				if sid != "" {
					logEvent.SessionID = sid
				}
				if status != "" {
					logEvent.Data["status"] = status
				}
			}

			if err := logger.Log(logEvent); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
			}

			fmt.Printf("[%s] %s\n", event.Event, event.Data)
			sessionEvents = append(sessionEvents, event)

			// Check for completion
			sessionID, completed := opencode.DetectCompletion(sessionEvents)
			if completed {
				// Try to get session title for notification
				workspace := ""
				if session, err := apiClient.GetSession(sessionID); err == nil && session != nil {
					workspace = session.Title
				}

				if err := notifier.SessionComplete(sessionID, workspace); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to send notification: %v\n", err)
				}
				fmt.Printf("\nSession %s completed!\n", sessionID)

				// Log completion
				completionEvent := events.Event{
					Type:      "session.completed",
					SessionID: sessionID,
					Timestamp: time.Now().Unix(),
				}
				if err := logger.Log(completionEvent); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log completion: %v\n", err)
				}

				// Reset for next session
				sessionEvents = nil
			}

		case err := <-errChan:
			return fmt.Errorf("SSE connection error: %w", err)
		}
	}
}

var (
	// Clean command flags
	cleanDryRun bool
	cleanAll    bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove completed agents and close their tmux windows",
	Long: `Remove completed and abandoned agents from the registry and close their tmux windows.

By default, only cleans agents that are marked as completed or abandoned in the registry.
Use --all to also reconcile the registry with active tmux windows first (marking agents
whose windows have closed as completed).

Examples:
  orch-go clean              # Clean completed/abandoned agents
  orch-go clean --dry-run    # Show what would be cleaned
  orch-go clean --all        # Reconcile with tmux first, then clean`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClean(cleanDryRun, cleanAll)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Reconcile with active tmux windows first")
}

func runClean(dryRun, reconcileFirst bool) error {
	// Get current directory to determine project name
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)

	// Open registry
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}

	// Optionally reconcile with tmux first
	if reconcileFirst {
		sessionName := tmux.GetWorkersSessionName(projectName)
		if tmux.SessionExists(sessionName) {
			activeIDs, err := tmux.ListWindowIDs(sessionName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to list tmux windows: %v\n", err)
			} else {
				completedCount := reg.Reconcile(activeIDs)
				if completedCount > 0 {
					fmt.Printf("Reconciled: %d agents marked as completed (windows closed)\n", completedCount)
					if err := reg.Save(); err != nil {
						return fmt.Errorf("failed to save registry after reconcile: %w", err)
					}
				}
			}
		}
	}

	// Get cleanable agents (completed or abandoned)
	agents := reg.ListCleanable()

	if len(agents) == 0 {
		fmt.Println("No agents to clean")
		return nil
	}

	// Get workers session name for this project
	sessionName := tmux.GetWorkersSessionName(projectName)
	sessionExists := tmux.SessionExists(sessionName)

	// Track cleanup stats
	windowsClosed := 0
	agentsCleaned := 0

	fmt.Printf("Found %d agents to clean:\n", len(agents))

	for _, agent := range agents {
		status := string(agent.Status)
		windowInfo := ""
		if agent.WindowID != "" {
			windowInfo = fmt.Sprintf(" (window: %s)", agent.WindowID)
		}

		if dryRun {
			fmt.Printf("  [DRY-RUN] Would clean: %s [%s]%s\n", agent.ID, status, windowInfo)
			continue
		}

		fmt.Printf("  Cleaning: %s [%s]%s\n", agent.ID, status, windowInfo)

		// Try to close tmux window if it exists
		if agent.WindowID != "" && sessionExists {
			if err := tmux.KillWindowByID(agent.WindowID); err != nil {
				// Window may already be closed, just log warning
				fmt.Fprintf(os.Stderr, "    Warning: failed to close window %s: %v\n", agent.WindowID, err)
			} else {
				windowsClosed++
			}
		}

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

	// Log the cleanup
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agents.cleaned",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"agents_cleaned":  agentsCleaned,
			"windows_closed":  windowsClosed,
			"project":         projectName,
			"reconcile_first": reconcileFirst,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("\nCleaned %d agents, closed %d windows\n", agentsCleaned, windowsClosed)
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
