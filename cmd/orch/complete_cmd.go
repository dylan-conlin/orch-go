// Package main provides the complete command for completing agents and closing beads issues.
// Extracted from main.go as part of the main.go refactoring (Phase 3).
package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Complete command flags
	completeForce            bool
	completeReason           string
	completeApprove          bool
	completeWorkdir          string
	completeNoChangelogCheck bool
	completeSkipReproCheck   bool
	completeSkipReproReason  string
)

var completeCmd = &cobra.Command{
	Use:   "complete [beads-id]",
	Short: "Complete an agent and close the beads issue",
	Long: `Complete an agent's work by verifying Phase: Complete and closing the beads issue.

Checks that the agent has reported "Phase: Complete" via beads comments before
closing the issue. Use --force to skip phase and liveness verification.

For agents that modified web/ files (UI tasks), --approve is required to explicitly
confirm human review of the visual changes. This prevents agents from self-certifying
UI correctness.

For cross-project completion (agents spawned with --workdir in another project),
the command auto-detects the project from the workspace's SPAWN_CONTEXT.md.
Use --workdir as explicit override when auto-detection fails.

For bug-type issues, prompts the orchestrator to verify that the original
reproduction no longer occurs. This repro verification runs even with --force.
Use --skip-repro-check with --skip-repro-reason to bypass.

Examples:
  orch-go complete proj-123
  orch-go complete proj-123 --reason "All tests passing"
  orch-go complete proj-123 --approve       # Approve UI changes after visual review
  orch-go complete proj-123 --force         # Skip phase/liveness verification (not repro)
  orch-go complete kb-cli-123 --workdir ~/projects/kb-cli  # Cross-project completion
  orch-go complete proj-123 --skip-repro-check --skip-repro-reason "Repro verified via automated test"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runComplete(beadsID, completeWorkdir)
	},
}

func init() {
	completeCmd.Flags().BoolVarP(&completeForce, "force", "f", false, "Skip phase and liveness verification (repro still runs)")
	completeCmd.Flags().StringVarP(&completeReason, "reason", "r", "", "Reason for closing (default: uses phase summary)")
	completeCmd.Flags().BoolVar(&completeApprove, "approve", false, "Approve visual changes for UI tasks (adds approval comment)")
	completeCmd.Flags().StringVar(&completeWorkdir, "workdir", "", "Target project directory (for cross-project completion)")
	completeCmd.Flags().BoolVar(&completeNoChangelogCheck, "no-changelog-check", false, "Skip changelog detection for notable changes")
	completeCmd.Flags().BoolVar(&completeSkipReproCheck, "skip-repro-check", false, "Skip reproduction verification for bug issues (requires --reason)")
	completeCmd.Flags().StringVar(&completeSkipReproReason, "skip-repro-reason", "", "Reason for skipping reproduction verification")
}

func runComplete(beadsID, workdir string) error {
	// Resolve short beads ID to full ID (e.g., "qdaa" -> "orch-go-qdaa")
	// This ensures all downstream operations use the full ID consistently
	resolvedID, err := resolveShortBeadsID(beadsID)
	if err != nil {
		return fmt.Errorf("failed to resolve beads ID: %w", err)
	}
	beadsID = resolvedID

	// Get current directory as base project dir
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine beads project directory:
	// 1. If --workdir provided, use that
	// 2. Otherwise, try to auto-detect from workspace SPAWN_CONTEXT.md
	// 3. Fall back to current directory
	var beadsProjectDir string
	var workspacePath, agentName string

	// First, find workspace in current project (even for cross-project agents, workspace is in orchestrator's project)
	workspacePath, agentName = findWorkspaceByBeadsID(currentDir, beadsID)

	if workdir != "" {
		// Explicit --workdir flag provided
		beadsProjectDir, err = filepath.Abs(workdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		// Verify directory exists
		if stat, err := os.Stat(beadsProjectDir); err != nil {
			return fmt.Errorf("workdir does not exist: %s", beadsProjectDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", beadsProjectDir)
		}
		fmt.Printf("Using explicit workdir: %s\n", beadsProjectDir)
	} else if workspacePath != "" {
		// Try to extract PROJECT_DIR from workspace SPAWN_CONTEXT.md
		projectDirFromWorkspace := extractProjectDirFromWorkspace(workspacePath)
		if projectDirFromWorkspace != "" && projectDirFromWorkspace != currentDir {
			// Cross-project agent detected
			beadsProjectDir = projectDirFromWorkspace
			fmt.Printf("Auto-detected cross-project: %s\n", filepath.Base(beadsProjectDir))
		} else {
			beadsProjectDir = currentDir
		}
	} else {
		beadsProjectDir = currentDir
	}

	// Set beads.DefaultDir for cross-project operations BEFORE any beads operations
	if beadsProjectDir != currentDir {
		beads.DefaultDir = beadsProjectDir
	}

	// Get issue to verify it exists
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		// Provide helpful error message for cross-project issues
		projectName := filepath.Base(beadsProjectDir)
		issuePrefix := strings.Split(beadsID, "-")[0]
		if len(strings.Split(beadsID, "-")) > 1 {
			issuePrefix = strings.Join(strings.Split(beadsID, "-")[:len(strings.Split(beadsID, "-"))-1], "-")
		}
		if issuePrefix != projectName {
			return fmt.Errorf("failed to get beads issue %s: %w\n\nHint: The issue ID suggests it belongs to project '%s', but you're in '%s'.\nTry: orch complete %s --workdir ~/path/to/%s", beadsID, err, issuePrefix, projectName, beadsID, issuePrefix)
		}
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Check if already closed
	isClosed := issue.Status == "closed"
	if isClosed {
		fmt.Printf("Issue %s is already closed in beads\n", beadsID)
	}

	// If --approve flag is set, add approval comment BEFORE verification
	// This ensures the visual verification gate sees the approval
	if completeApprove {
		approvalComment := "✅ APPROVED - Visual changes reviewed and approved by orchestrator"
		if err := addApprovalComment(beadsID, approvalComment); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add approval comment: %v\n", err)
			// Continue anyway - the approval might already exist or we can fallback
		} else {
			fmt.Printf("Added approval: %s\n", approvalComment)
		}
	}

	// Verify phase status unless force flag is set
	if !completeForce {
		// Workspace already found at top of function
		if workspacePath != "" {
			fmt.Printf("Workspace: %s\n", agentName)
		}

		// Use beadsProjectDir for verification (where the beads issue lives)
		result, err := verify.VerifyCompletionFull(beadsID, workspacePath, beadsProjectDir, "")
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

		// Print constraint warnings
		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "⚠️  %s\n", w)
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

	// Check liveness before closing - warn if agent appears still running
	// BUT: Skip this check if Phase: Complete was reported - agent said it's done,
	// so whether its session is still open is irrelevant.
	// This prevents false positives from OpenCode sessions that persist to disk.
	if !completeForce {
		// Check if Phase: Complete was reported
		phaseComplete, _ := verify.IsPhaseComplete(beadsID)

		// Only check liveness if agent hasn't reported completion
		if !phaseComplete {
			liveness := state.GetLiveness(beadsID, serverURL, beadsProjectDir)
			if liveness.IsAlive() {
				// Build warning message with details about what's still running
				var runningDetails []string
				if liveness.TmuxLive {
					detail := "tmux window"
					if liveness.WindowID != "" {
						detail += " (" + liveness.WindowID + ")"
					}
					runningDetails = append(runningDetails, detail)
				}
				if liveness.OpencodeLive {
					detail := "OpenCode session"
					if liveness.SessionID != "" {
						detail += " (" + liveness.SessionID[:12] + ")"
					}
					runningDetails = append(runningDetails, detail)
				}

				fmt.Fprintf(os.Stderr, "⚠️  Agent appears still running: %s\n", strings.Join(runningDetails, ", "))

				// Check if stdin is a terminal for interactive prompting
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return fmt.Errorf("agent still running and stdin is not a terminal; use --force to complete anyway")
				}

				// Prompt user for confirmation
				fmt.Fprint(os.Stderr, "Proceed anyway? [y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read response: %w", err)
				}

				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					return fmt.Errorf("aborted: agent still running")
				}

				fmt.Println("Proceeding with completion despite liveness warning...")
			}
		}
	}

	// DISABLED: Reproduction verification gate (Jan 4, 2026)
	// This was added to ensure bugs are actually fixed before closing, but it created
	// too much friction - agents couldn't complete without manual intervention.
	// Keeping the code commented for potential future re-enablement with better UX.
	// See: .kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md
	/*
		if !completeSkipReproCheck {
			reproResult, err := verify.GetReproForCompletion(beadsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check reproduction: %v\n", err)
			} else if reproResult != nil && reproResult.IsBug {
				// ... gate logic disabled ...
			}
		}
	*/
	_ = completeSkipReproCheck  // silence unused variable warning
	_ = completeSkipReproReason // silence unused variable warning

	// Check synthesis for follow-up recommendations (workspace already found at top)
	if workspacePath != "" {
		synthesis, err := verify.ParseSynthesis(workspacePath)
		if err == nil && synthesis != nil {
			// Check if there are follow-up recommendations to surface
			hasFollowUp := false
			if synthesis.Recommendation == "spawn-follow-up" || synthesis.Recommendation == "escalate" || synthesis.Recommendation == "resume" {
				hasFollowUp = true
			}
			if len(synthesis.NextActions) > 0 {
				hasFollowUp = true
			}

			if hasFollowUp {
				fmt.Println("\n--- Follow-up Recommendations ---")

				if synthesis.Recommendation != "" && synthesis.Recommendation != "close" {
					fmt.Printf("Recommendation: %s\n", synthesis.Recommendation)
				}

				// Collect all actionable items
				var actionableItems []string
				actionableItems = append(actionableItems, synthesis.NextActions...)
				actionableItems = append(actionableItems, synthesis.AreasToExplore...)
				actionableItems = append(actionableItems, synthesis.Uncertainties...)

				if len(actionableItems) > 0 {
					fmt.Printf("\n%d actionable items found:\n", len(actionableItems))
					for i, action := range actionableItems {
						fmt.Printf("  %d. %s\n", i+1, action)
					}
				}

				fmt.Println("\n---------------------------------")

				// Prompt for each actionable item (only if stdin is a terminal)
				if len(actionableItems) > 0 {
					if !term.IsTerminal(int(os.Stdin.Fd())) {
						fmt.Println("(Skipping interactive prompts - stdin is not a terminal)")
					} else {
						reader := bufio.NewReader(os.Stdin)
						createdCount := 0

						for i, action := range actionableItems {
							fmt.Printf("\n[%d/%d] %s\n", i+1, len(actionableItems), action)
							fmt.Print("Create issue? [y/N/q to quit]: ")
							response, err := reader.ReadString('\n')
							if err != nil {
								break
							}
							response = strings.TrimSpace(strings.ToLower(response))

							if response == "q" || response == "quit" {
								fmt.Println("Skipping remaining items.")
								break
							}

							if response == "y" || response == "yes" {
								// Create the issue using beads
								issue, err := beads.FallbackCreate(action, "", "task", 2, []string{"triage:review"})
								if err != nil {
									fmt.Fprintf(os.Stderr, "  Failed to create issue: %v\n", err)
								} else {
									fmt.Printf("  Created: %s\n", issue.ID)
									createdCount++
								}
							}
						}

						if createdCount > 0 {
							fmt.Printf("\n✓ Created %d follow-up issue(s)\n", createdCount)
						}
					}
				}
			}
		}
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

		// Remove triage:ready label on successful completion
		// This ensures failed/abandoned agents leave issues in ready queue for daemon retry
		if err := verify.RemoveTriageReadyLabel(beadsID); err != nil {
			// Non-critical - the issue may not have had this label
			// or it was already removed
		}
	}
	fmt.Printf("Reason: %s\n", reason)

	// Clean up tmux window if it exists (prevents phantom accumulation)
	if window, sessionName, err := tmux.FindWindowByBeadsIDAllSessions(beadsID); err == nil && window != nil {
		if err := tmux.KillWindow(window.Target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close tmux window %s: %v\n", window.Target, err)
		} else {
			fmt.Printf("Closed tmux window: %s:%s\n", sessionName, window.Name)
		}
	}

	// Auto-rebuild if agent committed Go changes (in the beads project)
	if hasGoChangesInRecentCommits(beadsProjectDir) {
		fmt.Println("Detected Go file changes in recent commits")
		if err := runAutoRebuild(beadsProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-rebuild failed: %v\n", err)
		} else {
			fmt.Println("Auto-rebuild completed: make install")
			// Restart orch serve if running
			if restarted, err := restartOrchServe(beadsProjectDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restart orch serve: %v\n", err)
			} else if restarted {
				fmt.Println("Restarted orch serve")
			}
		}

		// Check for new CLI commands that may need skill documentation
		newCommands := detectNewCLICommands(beadsProjectDir)
		if len(newCommands) > 0 {
			fmt.Println()
			fmt.Println("┌─────────────────────────────────────────────────────────────┐")
			fmt.Println("│  📚 NEW CLI COMMANDS DETECTED                               │")
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			for _, cmd := range newCommands {
				fmt.Printf("│  • %s\n", cmd)
			}
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			fmt.Println("│  Consider updating skill documentation:                     │")
			fmt.Println("│  - ~/.claude/skills/meta/orchestrator/SKILL.md              │")
			fmt.Println("│  - docs/orch-commands-reference.md                          │")
			fmt.Println("└─────────────────────────────────────────────────────────────┘")
		}
	}

	// Check for notable changelog entries (BREAKING/behavioral changes, especially skill changes)
	if !completeNoChangelogCheck {
		// Extract agent's skill from workspace if available
		var agentSkill string
		if workspacePath != "" {
			agentSkill, _ = verify.ExtractSkillNameFromSpawnContext(workspacePath)
		}

		notableEntries := detectNotableChangelogEntries(beadsProjectDir, agentSkill)
		if len(notableEntries) > 0 {
			fmt.Println()
			fmt.Println("┌─────────────────────────────────────────────────────────────┐")
			fmt.Println("│  ⚠️  NOTABLE ECOSYSTEM CHANGES DETECTED                      │")
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			for _, entry := range notableEntries {
				// Wrap long entries
				if len(entry) > 55 {
					fmt.Printf("│  %s\n", entry[:55])
					fmt.Printf("│    %s\n", entry[55:])
				} else {
					fmt.Printf("│  %s\n", entry)
				}
			}
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			fmt.Println("│  Review recent changes that may affect agent behavior       │")
			fmt.Println("│  Run: orch changelog --days 3                               │")
			fmt.Println("└─────────────────────────────────────────────────────────────┘")
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

	// Invalidate orch serve cache to ensure dashboard shows updated status immediately.
	// Without this, the TTL cache holds stale "active" status after completion.
	invalidateServeCache()

	return nil
}

// invalidateServeCache sends a request to orch serve to invalidate its caches.
// This ensures the dashboard shows updated agent status immediately after completion.
// Silently fails if orch serve is not running (cache will refresh via TTL).
func invalidateServeCache() {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Post(
		fmt.Sprintf("http://localhost:%d/api/cache/invalidate", DefaultServePort),
		"application/json",
		nil,
	)
	if err != nil {
		// Silent failure - orch serve might not be running
		return
	}
	defer resp.Body.Close()
	// We don't care about the response - if it worked, great; if not, TTL will eventually refresh
}

// addApprovalComment adds an approval comment to a beads issue.
// This is used by --approve flag to mark visual changes as human-reviewed.
func addApprovalComment(beadsID, comment string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		// Use "orchestrator" as the author for approval comments
		err := client.AddComment(beadsID, "orchestrator", comment)
		if err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackAddComment(beadsID, comment)
}

// hasGoChangesInRecentCommits checks if any of the last 5 commits contain changes
// to cmd/orch/*.go or pkg/*.go files.
func hasGoChangesInRecentCommits(projectDir string) bool {
	// Get changed files from last 5 commits
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails (e.g., not enough commits), try last 1 commit
		cmd = exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return false
		}
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if file matches cmd/orch/*.go or pkg/*.go or pkg/**/*.go
		if strings.HasPrefix(line, "cmd/orch/") && strings.HasSuffix(line, ".go") {
			return true
		}
		if strings.HasPrefix(line, "pkg/") && strings.HasSuffix(line, ".go") {
			return true
		}
	}
	return false
}

// detectNewCLICommands checks if any of the last 5 commits added new CLI command files
// to cmd/orch/. A file is considered a new command if:
// 1. It's in cmd/orch/*.go (not a test file)
// 2. It was added (not modified) in recent commits
// 3. It contains cobra.Command definitions
// Returns the list of new command file names (without path prefix).
func detectNewCLICommands(projectDir string) []string {
	var newCommands []string

	// Get files added (not modified) in last 5 commits
	// The 'A' status means added
	cmd := exec.Command("git", "diff", "--name-status", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails (e.g., not enough commits), try last 1 commit
		cmd = exec.Command("git", "diff", "--name-status", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return nil
		}
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse status line: "A\tcmd/orch/newcmd.go" or "M\tcmd/orch/main.go"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		filePath := parts[1]

		// Only care about added files (not modified)
		if status != "A" {
			continue
		}

		// Only check cmd/orch/*.go files (not test files)
		if !strings.HasPrefix(filePath, "cmd/orch/") || !strings.HasSuffix(filePath, ".go") {
			continue
		}
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}

		// Read the file to check if it contains cobra command definitions
		fullPath := filepath.Join(projectDir, filePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		// Look for cobra command pattern: "var xxxCmd = &cobra.Command{"
		if strings.Contains(string(content), "cobra.Command{") &&
			strings.Contains(string(content), "rootCmd.AddCommand(") {
			// Extract just the filename
			fileName := filepath.Base(filePath)
			newCommands = append(newCommands, fileName)
		}
	}

	return newCommands
}

// NotableChangelogEntry represents a notable change from the changelog.
type NotableChangelogEntry struct {
	Commit CommitInfo
	Reason string // Why this is notable (e.g., "BREAKING", "skill-relevant", "behavioral")
}

// detectNotableChangelogEntries checks recent commits across ecosystem repos for
// notable changes that the orchestrator should be aware of:
// - BREAKING changes
// - Behavioral changes (feat/fix commits)
// - Skill changes relevant to the agent's skill
// Returns formatted strings for display.
func detectNotableChangelogEntries(projectDir string, agentSkill string) []string {
	var entries []string

	// Get changelog data for last 3 days (recent enough to be relevant)
	result, err := GetChangelog(3, "all")
	if err != nil {
		return nil
	}

	// Iterate through commits looking for notable entries
	for _, dateCommits := range result.CommitsByDate {
		for _, commit := range dateCommits {
			var reasons []string

			// Check for BREAKING changes
			if commit.SemanticInfo.IsBreaking {
				reasons = append(reasons, "BREAKING")
			}

			// Check for behavioral changes (feat/fix)
			if commit.SemanticInfo.ChangeType == ChangeTypeBehavioral {
				// Only surface if it's in a category that could affect agents
				if commit.Category == "skills" || commit.Category == "skill-behavioral" ||
					commit.Category == "cmd" || commit.Category == "pkg" {
					reasons = append(reasons, "behavioral")
				}
			}

			// Check for skill-relevant changes
			if agentSkill != "" && isSkillRelevantChange(commit, agentSkill) {
				reasons = append(reasons, fmt.Sprintf("relevant to %s", agentSkill))
			}

			// If we have reasons, add to the list
			if len(reasons) > 0 {
				icon := "📌"
				if commit.SemanticInfo.IsBreaking {
					icon = "🚨"
				} else if strings.Contains(strings.Join(reasons, ","), "relevant to") {
					icon = "🎯"
				}

				entry := fmt.Sprintf("%s [%s] %s (%s)",
					icon,
					commit.Repo,
					truncateString(commit.Subject, 40),
					strings.Join(reasons, ", "))
				entries = append(entries, entry)
			}
		}
	}

	// Limit to top 5 most notable entries to avoid noise
	if len(entries) > 5 {
		entries = entries[:5]
	}

	return entries
}

// isSkillRelevantChange checks if a commit affects files related to a specific skill.
func isSkillRelevantChange(commit CommitInfo, skillName string) bool {
	for _, file := range commit.Files {
		// Check for skill-specific paths (handles both "skills/" prefix and "/skills/")
		if strings.Contains(file, "skills/") {
			// Check if this skill is mentioned in the path
			if strings.Contains(file, "/"+skillName+"/") ||
				strings.Contains(file, "/"+skillName+".") ||
				strings.HasPrefix(file, "skills/"+skillName+"/") ||
				strings.Contains(file, "/skills/"+skillName+"/") {
				return true
			}
		}

		// Check for SPAWN_CONTEXT or spawn package changes (affects all skills)
		if strings.Contains(file, "SPAWN_CONTEXT") ||
			strings.Contains(file, "pkg/spawn/") {
			return true
		}

		// Check for skill verification changes
		if strings.Contains(file, "pkg/verify/skill") {
			return true
		}
	}
	return false
}

// runAutoRebuild runs make install in the project directory.
func runAutoRebuild(projectDir string) error {
	cmd := exec.Command("make", "install")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// restartOrchServe checks if orch serve is running and restarts it.
// Returns true if it was restarted, false if it wasn't running.
func restartOrchServe(projectDir string) (bool, error) {
	// Find the orch serve process
	// We look for processes matching "orch serve" or "orch-go serve"
	cmd := exec.Command("pgrep", "-f", "orch.*serve")
	output, err := cmd.Output()
	if err != nil {
		// No process found - that's fine, just means serve isn't running
		return false, nil
	}

	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(pids) == 0 || pids[0] == "" {
		return false, nil
	}

	// Get the current PID to avoid killing ourselves
	currentPID := os.Getpid()

	// Kill the serve process(es)
	var killedAny bool
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
		if err != nil {
			continue
		}
		// Don't kill ourselves
		if pid == currentPID {
			continue
		}
		// Send SIGTERM for graceful shutdown
		killCmd := exec.Command("kill", "-TERM", pidStr)
		if err := killCmd.Run(); err == nil {
			killedAny = true
		}
	}

	if !killedAny {
		return false, nil
	}

	// Wait a moment for the process to stop
	time.Sleep(500 * time.Millisecond)

	// Start orch serve in the background
	// We use nohup to ensure it survives after we exit
	serveCmd := exec.Command("nohup", "orch", "serve")
	serveCmd.Dir = projectDir
	// Redirect output to files to avoid blocking
	devNull, _ := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	serveCmd.Stdout = devNull
	serveCmd.Stderr = devNull
	if err := serveCmd.Start(); err != nil {
		return true, fmt.Errorf("killed old serve but failed to start new: %w", err)
	}

	return true, nil
}
