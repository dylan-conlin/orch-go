// Package main provides helper functions used by the completion pipeline phases.
// These are extracted from complete_cmd.go to keep the command file focused on
// CLI definition and the thin pipeline orchestrator.
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
	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// addApprovalComment adds an approval comment to a beads issue.
// This is used by --approve flag to mark visual changes as human-reviewed.
func addApprovalComment(beadsID, comment string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		err := client.AddComment(beadsID, "orchestrator", comment)
		if err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackAddComment(beadsID, comment)
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
		return
	}
	defer resp.Body.Close()
}

// hasGoChangesInRecentCommits checks if any of the last 5 commits contain changes
// to cmd/orch/*.go or pkg/*.go files.
func hasGoChangesInRecentCommits(projectDir string) bool {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
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

	cmd := exec.Command("git", "diff", "--name-status", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
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
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		filePath := parts[1]

		if status != "A" {
			continue
		}

		if !strings.HasPrefix(filePath, "cmd/orch/") || !strings.HasSuffix(filePath, ".go") {
			continue
		}
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}

		fullPath := filepath.Join(projectDir, filePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		if strings.Contains(string(content), "cobra.Command{") &&
			strings.Contains(string(content), "rootCmd.AddCommand(") {
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
// notable changes that the orchestrator should be aware of.
func detectNotableChangelogEntries(projectDir string, agentSkill string) []string {
	var entries []string

	result, err := GetChangelog(3, "all")
	if err != nil {
		return nil
	}

	for _, dateCommits := range result.CommitsByDate {
		for _, commit := range dateCommits {
			var reasons []string

			if commit.SemanticInfo.IsBreaking {
				reasons = append(reasons, "BREAKING")
			}

			if commit.SemanticInfo.ChangeType == ChangeTypeBehavioral {
				if commit.Category == "skills" || commit.Category == "skill-behavioral" ||
					commit.Category == "cmd" || commit.Category == "pkg" {
					reasons = append(reasons, "behavioral")
				}
			}

			if agentSkill != "" && isSkillRelevantChange(commit, agentSkill) {
				reasons = append(reasons, fmt.Sprintf("relevant to %s", agentSkill))
			}

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

	if len(entries) > 5 {
		entries = entries[:5]
	}

	return entries
}

// isSkillRelevantChange checks if a commit affects files related to a specific skill.
func isSkillRelevantChange(commit CommitInfo, skillName string) bool {
	for _, file := range commit.Files {
		if strings.Contains(file, "skills/") {
			if strings.Contains(file, "/"+skillName+"/") ||
				strings.Contains(file, "/"+skillName+".") ||
				strings.HasPrefix(file, "skills/"+skillName+"/") ||
				strings.Contains(file, "/skills/"+skillName+"/") {
				return true
			}
		}

		if strings.Contains(file, "SPAWN_CONTEXT") ||
			strings.Contains(file, "pkg/spawn/") {
			return true
		}

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
	overmindSock := filepath.Join(projectDir, ".overmind.sock")
	if _, err := os.Stat(overmindSock); err == nil {
		cmd := exec.Command("overmind", "restart", "api")
		cmd.Dir = projectDir
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("overmind restart api failed: %w", err)
		}
		return true, nil
	}

	cmd := exec.Command("pgrep", "-f", "orch.*serve")
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(pids) == 0 || pids[0] == "" {
		return false, nil
	}

	currentPID := os.Getpid()

	var killedAny bool
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
		if err != nil {
			continue
		}
		if pid == currentPID {
			continue
		}
		killCmd := exec.Command("kill", "-TERM", pidStr)
		if err := killCmd.Run(); err == nil {
			killedAny = true
		}
	}

	if !killedAny {
		return false, nil
	}

	time.Sleep(500 * time.Millisecond)

	serveCmd := exec.Command("nohup", "orch", "serve")
	serveCmd.Dir = projectDir
	devNull, _ := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	serveCmd.Stdout = devNull
	serveCmd.Stderr = devNull
	if err := serveCmd.Start(); err != nil {
		return true, fmt.Errorf("killed old serve but failed to start new: %w", err)
	}

	return true, nil
}

func looksLikeWorkspaceName(identifier string) bool {
	return strings.HasPrefix(identifier, "og-") ||
		strings.HasPrefix(identifier, "meta-") ||
		strings.HasPrefix(identifier, "orch-")
}

func findWorkspaceByNameAcrossProjects(workspaceName string) string {
	for _, project := range getKBProjectsWithNames() {
		if wsPath := findWorkspaceByName(project.Path, workspaceName); wsPath != "" {
			return wsPath
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	rootCandidates := []string{
		filepath.Join(homeDir, "Documents", "personal"),
		filepath.Join(homeDir, "projects"),
		filepath.Join(homeDir, "src"),
	}

	for _, root := range rootCandidates {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			projectDir := filepath.Join(root, entry.Name())
			if wsPath := findWorkspaceByName(projectDir, workspaceName); wsPath != "" {
				return wsPath
			}
		}
	}

	return ""
}

// exportOrchestratorTranscript exports the session transcript for orchestrator sessions.
func exportOrchestratorTranscript(workspacePath, projectDir, beadsID string) error {
	orchestratorMarker := filepath.Join(workspacePath, ".orchestrator")
	metaOrchestratorMarker := filepath.Join(workspacePath, ".meta-orchestrator")

	isOrchestrator := false
	if _, err := os.Stat(orchestratorMarker); err == nil {
		isOrchestrator = true
	} else if _, err := os.Stat(metaOrchestratorMarker); err == nil {
		isOrchestrator = true
	}

	if !isOrchestrator {
		return nil
	}

	window, _, err := tmux.FindWindowByBeadsIDAllSessions(beadsID)
	if err != nil || window == nil {
		return fmt.Errorf("could not find tmux window for orchestrator")
	}

	existingExports := make(map[string]bool)
	pattern := filepath.Join(projectDir, "session-ses_*.md")
	matches, _ := filepath.Glob(pattern)
	for _, m := range matches {
		existingExports[m] = true
	}

	if err := tmux.SendKeys(window.Target, "/export"); err != nil {
		return fmt.Errorf("failed to send /export: %w", err)
	}
	if err := tmux.SendEnter(window.Target); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	fmt.Println("Exporting orchestrator transcript...")

	var newExportPath string
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if !existingExports[m] {
				newExportPath = m
				break
			}
		}
		if newExportPath != "" {
			break
		}
	}

	if newExportPath == "" {
		return fmt.Errorf("timeout waiting for export file")
	}

	destPath := filepath.Join(workspacePath, "TRANSCRIPT.md")
	if err := os.Rename(newExportPath, destPath); err != nil {
		input, err := os.ReadFile(newExportPath)
		if err != nil {
			return fmt.Errorf("failed to read export: %w", err)
		}
		if err := os.WriteFile(destPath, input, 0644); err != nil {
			return fmt.Errorf("failed to write transcript: %w", err)
		}
		os.Remove(newExportPath)
	}

	fmt.Printf("Saved transcript: %s\n", destPath)
	return nil
}

// archiveWorkspace moves a completed workspace to the archived directory.
// Returns the new archived path on success, or an error if archival fails.
// The function handles name collisions by adding a timestamp suffix.
//
// Note: This function is superseded by LifecycleManager.Archive() but retained
// for backward compatibility with tests.
func archiveWorkspace(workspacePath, projectDir string) (string, error) {
	if workspacePath == "" {
		return "", fmt.Errorf("workspace path is empty")
	}

	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return "", fmt.Errorf("workspace does not exist: %s", workspacePath)
	}

	workspaceName := filepath.Base(workspacePath)
	archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")

	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create archived directory: %w", err)
	}

	destPath := filepath.Join(archivedDir, workspaceName)

	if _, err := os.Stat(destPath); err == nil {
		suffix := time.Now().Format("150405")
		destPath = filepath.Join(archivedDir, workspaceName+"-"+suffix)
		fmt.Printf("Note: Archive destination exists, using: %s-%s\n", workspaceName, suffix)
	}

	if err := os.Rename(workspacePath, destPath); err != nil {
		return "", fmt.Errorf("failed to archive workspace: %w", err)
	}

	return destPath, nil
}

// collectCompletionTelemetry collects duration and token usage for telemetry.
// Returns (durationSeconds, tokensInput, tokensOutput, outcome).
func collectCompletionTelemetry(workspacePath string, forced bool, verificationPassed bool) (int, int, int, string) {
	var durationSeconds int
	var tokensInput int
	var tokensOutput int
	var outcome string

	if forced {
		outcome = "forced"
	} else if verificationPassed {
		outcome = "success"
	} else {
		outcome = "failed"
	}

	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	if spawnTime := manifest.ParseSpawnTime(); !spawnTime.IsZero() {
		durationSeconds = int(time.Since(spawnTime).Seconds())
	}

	sessionID := spawn.ReadSessionID(workspacePath)
	if sessionID != "" {
		client := opencode.NewClient("http://127.0.0.1:4096")
		if tokenStats, err := client.GetSessionTokens(sessionID); err == nil && tokenStats != nil {
			tokensInput = tokenStats.InputTokens
			tokensOutput = tokenStats.OutputTokens
		}
	}

	return durationSeconds, tokensInput, tokensOutput, outcome
}

type verificationChecklistItem struct {
	Label  string
	Status string // passed, pending, skipped
}

func buildVerificationChecklist(
	result verify.VerificationResult,
	issueType string,
	tier string,
	isOrchestrator bool,
	skipConfig verify.SkipConfig,
	gate1Complete bool,
	gate2Complete bool,
) []verificationChecklistItem {
	items := []verificationChecklistItem{}
	appendItem := func(label, status string) {
		if status == "n/a" {
			return
		}
		items = append(items, verificationChecklistItem{Label: label, Status: status})
	}

	gateStatus := func(gate string) string {
		if skipConfig.ShouldSkipGate(gate) {
			return "skipped"
		}
		for _, failed := range result.GatesFailed {
			if failed == gate {
				return "pending"
			}
		}
		return "passed"
	}

	if isOrchestrator {
		appendItem("session handoff", gateStatus(verify.GateSessionHandoff))
		appendItem("handoff content", gateStatus(verify.GateHandoffContent))
		return items
	}

	if issueType != "" && checkpoint.RequiresCheckpoint(issueType) {
		explainStatus := "pending"
		if skipConfig.ExplainBack {
			explainStatus = "skipped"
		} else if gate1Complete {
			explainStatus = "passed"
		}
		appendItem("explain-back (gate1)", explainStatus)
	} else {
		appendItem("explain-back (gate1)", "n/a")
	}

	if issueType != "" && checkpoint.RequiresGate2(issueType) {
		behaviorStatus := "pending"
		if gate2Complete {
			behaviorStatus = "passed"
		}
		appendItem("behavioral verification (gate2)", behaviorStatus)
	} else {
		appendItem("behavioral verification (gate2)", "n/a")
	}

	appendItem("phase complete", gateStatus(verify.GatePhaseComplete))

	if tier == "light" || verify.IsKnowledgeProducingSkill(result.Skill) {
		appendItem("synthesis", "n/a")
	} else {
		appendItem("synthesis", gateStatus(verify.GateSynthesis))
	}

	appendItem("test evidence", gateStatus(verify.GateTestEvidence))
	appendItem("visual verification", gateStatus(verify.GateVisualVerify))
	appendItem("git diff", gateStatus(verify.GateGitDiff))
	appendItem("build", gateStatus(verify.GateBuild))
	appendItem("constraint", gateStatus(verify.GateConstraint))
	appendItem("phase gate", gateStatus(verify.GatePhaseGate))
	appendItem("skill output", gateStatus(verify.GateSkillOutput))
	appendItem("decision patch limit", gateStatus(verify.GateDecisionPatchLimit))
	appendItem("accretion", gateStatus(verify.GateAccretion))
	appendItem("architectural choices", gateStatus(verify.GateArchitecturalChoices))

	return items
}

func printVerificationChecklist(items []verificationChecklistItem) {
	if len(items) == 0 {
		return
	}

	fmt.Println("\n--- Verification Checklist ---")
	for _, item := range items {
		fmt.Printf("  [%s] %s\n", formatChecklistStatus(item.Status), item.Label)
	}
	fmt.Println("--------------------------------")
}

func formatChecklistStatus(status string) string {
	switch status {
	case "passed":
		return "PASS"
	case "pending":
		return "PEND"
	case "skipped":
		return "SKIP"
	default:
		return "N/A"
	}
}

// collectAccretionDelta collects file growth/shrinkage metrics from git diff.
func collectAccretionDelta(projectDir, workspacePath string) *events.AccretionDeltaData {
	if workspacePath == "" {
		return nil
	}

	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	spawnTime := manifest.ParseSpawnTime()
	if spawnTime.IsZero() {
		return nil
	}

	sinceStr := spawnTime.UTC().Format("2006-01-02T15:04:05Z")

	relWorkspace := workspacePath
	if filepath.IsAbs(workspacePath) && filepath.IsAbs(projectDir) {
		rel, err := filepath.Rel(projectDir, workspacePath)
		if err == nil {
			relWorkspace = rel
		}
	}

	cmd := exec.Command("git", "log", "--since="+sinceStr, "--format=%H", "--", relWorkspace)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		return nil
	}

	commitHashes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(commitHashes) == 0 {
		return nil
	}

	fileDeltas := make(map[string]*events.FileDelta)

	for _, hash := range commitHashes {
		if hash == "" {
			continue
		}

		cmd := exec.Command("git", "show", "--numstat", "--format=", hash)
		cmd.Dir = projectDir
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := strings.Split(line, "\t")
			if len(parts) < 3 {
				continue
			}

			filePath := parts[2]

			added := 0
			removed := 0
			if parts[0] != "-" {
				var n int
				_, err := fmt.Sscanf(parts[0], "%d", &n)
				if err == nil {
					added = n
				}
			}
			if parts[1] != "-" {
				var n int
				_, err := fmt.Sscanf(parts[1], "%d", &n)
				if err == nil {
					removed = n
				}
			}

			if existing, ok := fileDeltas[filePath]; ok {
				existing.LinesAdded += added
				existing.LinesRemoved += removed
				existing.NetDelta = existing.LinesAdded - existing.LinesRemoved
			} else {
				fileDeltas[filePath] = &events.FileDelta{
					Path:         filePath,
					LinesAdded:   added,
					LinesRemoved: removed,
					NetDelta:     added - removed,
				}
			}
		}
	}

	var totalAdded, totalRemoved, riskFiles int
	var deltas []events.FileDelta

	for _, delta := range fileDeltas {
		fullPath := filepath.Join(projectDir, delta.Path)
		if lineCount, err := countFileLines(fullPath); err == nil {
			delta.TotalLines = lineCount
			delta.IsAccretionRisk = lineCount > 800

			if delta.IsAccretionRisk && delta.NetDelta > 0 {
				riskFiles++
			}
		}

		totalAdded += delta.LinesAdded
		totalRemoved += delta.LinesRemoved
		deltas = append(deltas, *delta)
	}

	if len(deltas) == 0 {
		return nil
	}

	return &events.AccretionDeltaData{
		FileDeltas:   deltas,
		TotalFiles:   len(deltas),
		TotalAdded:   totalAdded,
		TotalRemoved: totalRemoved,
		NetDelta:     totalAdded - totalRemoved,
		RiskFiles:    riskFiles,
	}
}

// countFileLines counts the number of lines in a file.
func countFileLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return lineCount, nil
}
