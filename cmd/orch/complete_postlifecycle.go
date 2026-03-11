// Package main provides post-lifecycle helper functions used after the completion
// pipeline's lifecycle transition. These handle cache invalidation, auto-rebuild,
// telemetry collection, transcript export, and accretion delta analysis.
// Extracted from complete_actions.go to keep each file focused.
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
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
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
	return beads.FallbackAddComment(beadsID, comment, "")
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

	return hasOrchGoChangesInOutput(string(output))
}

// hasAgentGoChanges checks if the agent modified cmd/orch/*.go or pkg/*.go files
// using the agent's spawn baseline. Falls back to hasGoChangesInRecentCommits
// if no baseline is available.
func hasAgentGoChanges(workspacePath, projectDir string) bool {
	if workspacePath == "" {
		return hasGoChangesInRecentCommits(projectDir)
	}

	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	baseline := manifest.GitBaseline
	if baseline == "" {
		return hasGoChangesInRecentCommits(projectDir)
	}

	cmd := exec.Command("git", "diff", "--name-only", baseline+"..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Baseline may be gc'd — fall back to global check
		return hasGoChangesInRecentCommits(projectDir)
	}

	return hasOrchGoChangesInOutput(string(output))
}

// hasOrchGoChangesInOutput checks if git diff output contains cmd/orch/*.go or pkg/*.go files.
func hasOrchGoChangesInOutput(output string) bool {
	lines := strings.Split(output, "\n")
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

// findWorkspaceByBeadsIDAcrossProjects searches all known projects for a workspace
// matching the given beads ID. This handles cross-repo spawns where the workspace
// is created in the target project (via --workdir) but the beads issue belongs to
// the source project. Without this fallback, orch complete fails to find the workspace
// when the beads ID prefix matches the CWD project.
func findWorkspaceByBeadsIDAcrossProjects(beadsID string) (workspacePath, agentName string) {
	for _, project := range getKBProjectsWithNames() {
		if wsPath, name := findWorkspaceByBeadsID(project.Path, beadsID); wsPath != "" {
			return wsPath, name
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", ""
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
			if wsPath, name := findWorkspaceByBeadsID(projectDir, beadsID); wsPath != "" {
				return wsPath, name
			}
		}
	}

	return "", ""
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

// collectAccretionDelta collects file growth/shrinkage metrics from git diff.
// Uses the agent's git baseline (commit SHA at spawn time) to compute the net
// diff between spawn and HEAD. Falls back to --since=<spawnTime> when no baseline.
func collectAccretionDelta(projectDir, workspacePath string) *events.AccretionDeltaData {
	if workspacePath == "" {
		fmt.Fprintf(os.Stderr, "accretion.delta skipped: no workspace path\n")
		return nil
	}

	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)

	// Primary: use git baseline for precise diff
	if manifest.GitBaseline != "" {
		return collectAccretionFromBaseline(projectDir, manifest.GitBaseline)
	}

	// Fallback: use spawn time with git log (no path filter)
	spawnTime := manifest.ParseSpawnTime()
	if spawnTime.IsZero() {
		fmt.Fprintf(os.Stderr, "accretion.delta skipped: no git baseline or spawn time\n")
		return nil
	}

	return collectAccretionFromSpawnTime(projectDir, spawnTime)
}

// collectAccretionFromBaseline uses git diff --numstat <baseline>..HEAD to get
// the net file changes since the agent was spawned.
func collectAccretionFromBaseline(projectDir, baseline string) *events.AccretionDeltaData {
	cmd := exec.Command("git", "diff", "--numstat", baseline+"..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "accretion.delta skipped: git diff baseline failed: %v\n", err)
		return nil
	}

	return parseNumstatOutput(projectDir, string(output))
}

// collectAccretionFromSpawnTime uses git log --since to find commits and aggregate
// their numstat. This is the fallback when no git baseline is available.
func collectAccretionFromSpawnTime(projectDir string, spawnTime time.Time) *events.AccretionDeltaData {
	sinceStr := spawnTime.UTC().Format("2006-01-02T15:04:05Z")

	// Find all commits since spawn time (no path filter — captures all agent work)
	cmd := exec.Command("git", "log", "--since="+sinceStr, "--format=%H")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		fmt.Fprintf(os.Stderr, "accretion.delta skipped: no commits since %s\n", sinceStr)
		return nil
	}

	commitHashes := strings.Split(strings.TrimSpace(string(output)), "\n")

	fileDeltas := make(map[string]*events.FileDelta)

	for _, hash := range commitHashes {
		if hash == "" {
			continue
		}

		cmd := exec.Command("git", "show", "--numstat", "--format=", hash)
		cmd.Dir = projectDir
		showOutput, err := cmd.Output()
		if err != nil {
			continue
		}

		parseNumstatLines(string(showOutput), fileDeltas)
	}

	return buildAccretionResult(projectDir, fileDeltas)
}

// parseNumstatOutput parses git diff --numstat output into accretion data.
func parseNumstatOutput(projectDir, output string) *events.AccretionDeltaData {
	fileDeltas := make(map[string]*events.FileDelta)
	parseNumstatLines(output, fileDeltas)
	return buildAccretionResult(projectDir, fileDeltas)
}

// parseNumstatLines parses numstat-formatted lines into a file delta map.
func parseNumstatLines(output string, fileDeltas map[string]*events.FileDelta) {
	lines := strings.Split(output, "\n")
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

// buildAccretionResult converts a file delta map into AccretionDeltaData.
func buildAccretionResult(projectDir string, fileDeltas map[string]*events.FileDelta) *events.AccretionDeltaData {
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
