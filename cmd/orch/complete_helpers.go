// Package main provides helper functions for the complete command.
// Includes changelog detection, auto-rebuild, new CLI command detection, and display helpers.
// Extracted from complete_cmd.go for maintainability.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// DefaultMakeInstallTimeout is the maximum time to wait for 'make install' to complete.
// 120 seconds is generous for a Go build while preventing indefinite hangs.
const DefaultMakeInstallTimeout = 120 * time.Second

// hasGoChangesInRecentCommits checks if any of the last 5 commits contain changes
// to cmd/orch/*.go or pkg/*.go files.
func hasGoChangesInRecentCommits(projectDir string) bool {
	files, err := verify.GetChangedFiles(projectDir, "")
	if err != nil {
		return false
	}

	for _, line := range files {
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

	// Get files added (not modified) in last 5 commits.
	// The 'A' status means added.
	lines, err := verify.GetChangedNameStatus(projectDir)
	if err != nil {
		return nil
	}

	for _, line := range lines {
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

// trackDocDebt adds new commands to the doc debt tracker.
// Returns the number of newly tracked commands.
func trackDocDebt(commands []string) int {
	debt, err := userconfig.LoadDocDebt()
	if err != nil {
		// Silent failure - don't break completion for doc tracking issues
		return 0
	}

	newlyTracked := 0
	for _, cmd := range commands {
		if debt.AddCommand(cmd) {
			newlyTracked++
		}
	}

	if newlyTracked > 0 {
		if err := userconfig.SaveDocDebt(debt); err != nil {
			// Silent failure
			return 0
		}
	}

	return newlyTracked
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

// rebuildGoProjectsIfNeeded checks for Go changes and rebuilds affected projects.
// This is called BEFORE verification to ensure verification runs against fresh binaries.
// It handles both the beads project directory and cross-project scenarios.
func rebuildGoProjectsIfNeeded(beadsProjectDir, workspacePath string) {
	// Collect unique project directories that might have Go changes
	projectDirs := make(map[string]bool)

	// Always check the beads project directory
	if beadsProjectDir != "" {
		projectDirs[beadsProjectDir] = true
	}

	// Check if workspace points to a different project (cross-project agent)
	if workspacePath != "" {
		projectDirFromWorkspace := extractProjectDirFromWorkspace(workspacePath)
		if projectDirFromWorkspace != "" && projectDirFromWorkspace != beadsProjectDir {
			projectDirs[projectDirFromWorkspace] = true
		}
	}

	// Check each project for Go changes and rebuild if needed
	var rebuiltOrchGo bool
	var orchGoDir string

	for projectDir := range projectDirs {
		if !hasGoChangesInRecentCommits(projectDir) {
			continue
		}

		// Check if this is a Go project (has go.mod)
		goModPath := filepath.Join(projectDir, "go.mod")
		if _, err := os.Stat(goModPath); os.IsNotExist(err) {
			continue
		}

		// Check if binary is already up-to-date (skip redundant rebuilds)
		upToDate, err := isBinaryUpToDate(projectDir)
		if err != nil {
			// Log warning but continue with rebuild on check failure
			fmt.Fprintf(os.Stderr, "Warning: failed to check binary freshness: %v\n", err)
		}
		if upToDate {
			// Binary is current, no rebuild needed
			continue
		}

		projectName := filepath.Base(projectDir)
		fmt.Printf("Detected Go changes in %s, auto-rebuilding...\n", projectName)

		if err := runAutoRebuild(projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-rebuild failed for %s: %v\n", projectName, err)
			continue
		}

		fmt.Printf("Auto-rebuild completed: %s/make install\n", projectName)

		// Track if we rebuilt orch-go (for service restart)
		if projectName == "orch-go" {
			rebuiltOrchGo = true
			orchGoDir = projectDir
		}
	}

	// Restart orch serve if orch-go was rebuilt
	if rebuiltOrchGo {
		if restarted, err := restartOrchServe(orchGoDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restart orch serve: %v\n", err)
		} else if restarted {
			fmt.Println("Restarted orch serve")
		}
	}
}

// runAutoRebuild runs make install in the project directory with a timeout.
func runAutoRebuild(projectDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultMakeInstallTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", "install")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("make install timed out after %v", DefaultMakeInstallTimeout)
	}
	return err
}

// isBinaryUpToDate checks if the binary is newer than the most recent Go source change.
// It compares the binary's modification time against the Git commit timestamp of the
// most recent commit that modified Go files. Returns true if the binary is current
// (no rebuild needed), false if a rebuild is needed, and an error if the check fails.
func isBinaryUpToDate(projectDir string) (bool, error) {
	// Find the binary path (check build/ directory first, then ~/bin symlink target)
	binaryPath := filepath.Join(projectDir, "build", "orch")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Binary doesn't exist, definitely needs rebuild
		return false, nil
	}

	// Get binary modification time
	binaryInfo, err := os.Stat(binaryPath)
	if err != nil {
		return false, fmt.Errorf("failed to stat binary: %w", err)
	}
	binaryMtime := binaryInfo.ModTime()

	// Get the timestamp of the most recent commit that modified Go files
	// Using: git log -1 --format=%ct -- "*.go" "**/*.go"
	timestampStr, err := verify.GetLatestCommitUnixTimestamp(projectDir, "*.go", "**/*.go")
	if err != nil {
		// If git command fails, assume rebuild is needed
		return false, nil
	}

	// Parse the Unix timestamp
	if timestampStr == "" {
		// No Go files in history, no rebuild needed
		return true, nil
	}

	unixTimestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false, fmt.Errorf("failed to parse commit timestamp: %w", err)
	}

	lastGoCommitTime := time.Unix(unixTimestamp, 0)

	// Binary is up-to-date if its mtime is after the last Go commit time
	// Add a small buffer (1 second) to handle timing edge cases
	return binaryMtime.After(lastGoCommitTime.Add(-time.Second)), nil
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

// printBehavioralValidationInfo outputs structured behavioral validation information.
// This is informational output for orchestrators, not a blocking gate.
func printBehavioralValidationInfo(result *verify.BehavioralValidationResult) {
	if result == nil || !result.BehavioralValidationSuggested {
		return
	}

	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│  🔍 BEHAVIORAL VALIDATION SUGGESTED                         │")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")

	// Validation type
	if result.ValidationType != "" {
		fmt.Printf("│  Type: %s\n", result.ValidationType)
	}

	// Trigger reason
	if result.TriggerReason != "" {
		fmt.Printf("│  Reason: %s\n", truncateString(result.TriggerReason, 50))
	}

	// Show evidence status
	if result.HasBehavioralEvidence {
		fmt.Println("│  ✅ Behavioral evidence found in beads comments")
		for _, e := range result.Evidence {
			if len(e) > 50 {
				e = e[:47] + "..."
			}
			fmt.Printf("│     • %s\n", e)
		}
	} else {
		fmt.Println("│  ⚠️  No behavioral evidence found in beads comments")
	}

	// Suggested URL for UI changes
	if result.SuggestedURL != "" {
		fmt.Printf("│  URL: %s\n", result.SuggestedURL)
	}

	// Suggested validation steps
	if len(result.SuggestedSteps) > 0 {
		fmt.Println("├─────────────────────────────────────────────────────────────┤")
		fmt.Println("│  Suggested validation steps:                                │")
		for i, step := range result.SuggestedSteps {
			if i >= 4 {
				fmt.Printf("│  ... and %d more steps\n", len(result.SuggestedSteps)-4)
				break
			}
			stepTrunc := step
			if len(step) > 50 {
				stepTrunc = step[:47] + "..."
			}
			fmt.Printf("│  %d. %s\n", i+1, stepTrunc)
		}
	}

	// Changed files that triggered this
	if len(result.ChangedFiles) > 0 && len(result.ChangedFiles) <= 3 {
		fmt.Println("├─────────────────────────────────────────────────────────────┤")
		fmt.Println("│  Changed files:                                             │")
		for _, f := range result.ChangedFiles {
			if len(f) > 50 {
				f = f[:47] + "..."
			}
			fmt.Printf("│    %s\n", f)
		}
	} else if len(result.ChangedFiles) > 3 {
		fmt.Println("├─────────────────────────────────────────────────────────────┤")
		fmt.Printf("│  Changed files: %d files (showing first 3)                 │\n", len(result.ChangedFiles))
		for _, f := range result.ChangedFiles[:3] {
			if len(f) > 50 {
				f = f[:47] + "..."
			}
			fmt.Printf("│    %s\n", f)
		}
	}

	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println()
}
