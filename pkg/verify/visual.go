// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// Skills that require visual verification when modifying web/ files.
// Only skills that are explicitly about UI work should require visual verification.
// Non-UI skills (architects, investigations, debugging) may incidentally modify web/
// files as part of broader work - these shouldn't require visual verification.
var skillsRequiringVisualVerification = map[string]bool{
	"feature-impl": true, // UI features need visual verification
	// Note: We don't include all possible UI skills - the default is permissive.
	// If a skill is not in this list and modifies web/ files, we assume it's incidental.
}

// Skills that are explicitly excluded from visual verification requirements.
// These skills are known to work on web/ files incidentally (not as primary UI work).
var skillsExcludedFromVisualVerification = map[string]bool{
	"architect":            true, // Design work may touch web/ files
	"investigation":        true, // Research may examine/modify web/ files
	"systematic-debugging": true, // Debugging may touch web/ files
	"research":             true, // Research doesn't do UI work
	"codebase-audit":       true, // Audits may touch any files
	"reliability-testing":  true, // Testing may touch any files
	"design-session":       true, // Design sessions don't do UI implementation
	"issue-creation":       true, // Issue creation doesn't do UI work
	"writing-skills":       true, // Skill writing may touch web/ examples
}

// IsSkillRequiringVisualVerification determines if a skill requires visual verification
// for web/ file changes.
//
// The logic is:
// 1. If skill is explicitly excluded (architect, investigation, etc.) -> false
// 2. If skill is explicitly included (feature-impl) -> true
// 3. If skill is unknown -> false (permissive default to avoid false positives)
//
// This approach prevents false positives from architects/investigations that modify
// web/ files incidentally as part of broader work.
func IsSkillRequiringVisualVerification(skillName string) bool {
	// Empty skill name means we couldn't determine the skill - be permissive
	if skillName == "" {
		return false
	}

	// Normalize skill name to lowercase for comparison
	skillName = strings.ToLower(skillName)

	// Check explicit exclusions first
	if skillsExcludedFromVisualVerification[skillName] {
		return false
	}

	// Check explicit inclusions
	if skillsRequiringVisualVerification[skillName] {
		return true
	}

	// Unknown skill - be permissive to avoid false positives
	return false
}

// WebChangeRisk represents the risk level of web/ file changes.
// Risk determines whether visual verification is required:
// - LOW: Trivial changes (CSS properties, colors) - no verification required
// - MEDIUM: Component/layout changes - verification required
// - HIGH: New pages, significant UX changes - verification required
type WebChangeRisk int

const (
	// WebRiskNone means no web changes detected
	WebRiskNone WebChangeRisk = iota
	// WebRiskLow means trivial changes that don't need visual verification
	// Examples: single CSS property, color changes, adding existing class
	WebRiskLow
	// WebRiskMedium means changes that need visual verification
	// Examples: new component, layout changes, style file modifications
	WebRiskMedium
	// WebRiskHigh means significant UI changes that definitely need verification
	// Examples: new pages, major UX changes, route additions
	WebRiskHigh
)

// String returns a human-readable name for the risk level.
func (r WebChangeRisk) String() string {
	switch r {
	case WebRiskNone:
		return "NONE"
	case WebRiskLow:
		return "LOW"
	case WebRiskMedium:
		return "MEDIUM"
	case WebRiskHigh:
		return "HIGH"
	default:
		return "UNKNOWN"
	}
}

// RequiresVisualVerification returns true if this risk level requires visual verification.
func (r WebChangeRisk) RequiresVisualVerification() bool {
	return r >= WebRiskMedium
}

// VisualVerificationResult represents the result of checking for visual verification evidence.
type VisualVerificationResult struct {
	Passed           bool          // Whether verification passed
	HasWebChanges    bool          // Whether web/ files were changed
	RiskLevel        WebChangeRisk // Risk level of the web changes
	HasEvidence      bool          // Whether visual verification evidence was found
	HasHumanApproval bool          // Whether human/orchestrator explicitly approved
	NeedsApproval    bool          // Whether human approval is required but missing
	Errors           []string      // Error messages
	Warnings         []string      // Warning messages
	Evidence         []string      // Evidence found (for debugging)
}

// visualEvidencePatterns defines patterns that indicate visual verification was performed.
// These patterns are checked against beads comments.
var visualEvidencePatterns = []*regexp.Regexp{
	// Screenshot mentions
	regexp.MustCompile(`(?i)screenshot`),
	regexp.MustCompile(`(?i)screen\s*shot`),
	regexp.MustCompile(`(?i)captured.*image`),
	regexp.MustCompile(`(?i)image.*captured`),
	// Visual verification mentions
	regexp.MustCompile(`(?i)visual\s*verif`),
	regexp.MustCompile(`(?i)visually\s*verif`),
	regexp.MustCompile(`(?i)browser\s*verif`),
	regexp.MustCompile(`(?i)ui\s*verif`),
	// Playwright/browser tool mentions
	regexp.MustCompile(`(?i)playwright`),
	regexp.MustCompile(`(?i)browser_take_screenshot`),
	regexp.MustCompile(`(?i)browser_navigate`),
	// Glass browser automation tool mentions
	regexp.MustCompile(`(?i)glass_page_state`),
	regexp.MustCompile(`(?i)glass_elements`),
	regexp.MustCompile(`(?i)glass_click`),
	regexp.MustCompile(`(?i)glass_type`),
	regexp.MustCompile(`(?i)glass_navigate`),
	regexp.MustCompile(`(?i)glass_screenshot`),
	regexp.MustCompile(`(?i)glass_scroll`),
	regexp.MustCompile(`(?i)glass_hover`),
	regexp.MustCompile(`(?i)glass_tabs`),
	regexp.MustCompile(`(?i)glass_focus`),
	regexp.MustCompile(`(?i)glass_enable_user_tracking`),
	regexp.MustCompile(`(?i)glass_recent_actions`),
	regexp.MustCompile(`(?i)glass assert`),
	regexp.MustCompile(`(?i)glass\s+tool`),
	// Smoke test with UI context
	regexp.MustCompile(`(?i)smoke\s*test.*ui`),
	regexp.MustCompile(`(?i)ui.*smoke\s*test`),
	// "Verified in browser" style comments
	regexp.MustCompile(`(?i)verified.*browser`),
	regexp.MustCompile(`(?i)browser.*verified`),
	regexp.MustCompile(`(?i)checked.*browser`),
	regexp.MustCompile(`(?i)tested.*browser`),
}

// humanApprovalPatterns defines patterns that indicate explicit human/orchestrator approval.
// These patterns must come from a human orchestrator, not from the agent itself.
// The patterns are designed to be unlikely to be accidentally used by agents.
var humanApprovalPatterns = []*regexp.Regexp{
	// Explicit approval markers (orchestrator uses these)
	regexp.MustCompile(`(?i)✅\s*APPROVED`),
	regexp.MustCompile(`(?i)UI\s*APPROVED`),
	regexp.MustCompile(`(?i)VISUAL\s*APPROVED`),
	regexp.MustCompile(`(?i)human_approved:\s*true`),
	regexp.MustCompile(`(?i)orchestrator_approved:\s*true`),
	// "I approve" style (first person indicates human)
	regexp.MustCompile(`(?i)I\s+approve\s+(the\s+)?(UI|visual|changes)`),
	regexp.MustCompile(`(?i)LGTM.*UI`),
	regexp.MustCompile(`(?i)UI.*LGTM`),
}

// HasWebChangesInRecentCommits checks if any of the last 5 commits contain changes
// to web/ files (Svelte, TypeScript, CSS, etc.).
//
// DEPRECATED: This function checks the last 5 project commits, which may include
// commits from other agents or prior work. Use HasWebChangesForAgent instead,
// which scopes to commits made since the agent was spawned.
func HasWebChangesInRecentCommits(projectDir string) bool {
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

	return hasWebChangesInFiles(string(output))
}

// HasWebChangesForAgent checks if any commits since the agent's spawn time
// contain changes to web/ files (Svelte, TypeScript, CSS, etc.).
//
// This function scopes to agent-specific changes by:
// 1. Reading the spawn time from the workspace's .spawn_time file
// 2. Finding commits that touch the workspace directory (to filter out other agents' commits)
// 3. Checking if any of those commits modified web/ files
//
// This workspace-scoped approach prevents false positives when multiple agents run
// concurrently - each agent only sees its own commits, not commits from other agents
// that happened to occur around the same spawn time.
//
// If the workspace has no spawn time file (legacy workspace), falls back to
// checking the last 5 commits for backward compatibility.
func HasWebChangesForAgent(projectDir, workspacePath string) bool {
	// Read spawn time from workspace
	spawnTime := spawn.ReadSpawnTime(workspacePath)

	// If no spawn time, fall back to the old behavior for backward compatibility
	if spawnTime.IsZero() {
		return HasWebChangesInRecentCommits(projectDir)
	}

	// Use workspace-scoped check to filter out concurrent agents' commits
	return hasWebChangesSinceTimeForWorkspace(projectDir, spawnTime, workspacePath)
}

// hasWebChangesSinceTime checks if any commits since the given time modified web/ files.
//
// DEPRECATED: This function checks ALL commits since the given time, which may include
// commits from other concurrent agents. Use hasWebChangesSinceTimeForWorkspace instead,
// which scopes to commits that touch the workspace directory.
func hasWebChangesSinceTime(projectDir string, since time.Time) bool {
	// Format time for git --since flag (ISO 8601 format works well)
	sinceStr := since.UTC().Format("2006-01-02T15:04:05Z")

	// Get all files changed in commits since spawn time
	// Using git log with --name-only to get file paths
	cmd := exec.Command("git", "log", "--since="+sinceStr, "--name-only", "--format=")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails, return false (no web changes detectable)
		return false
	}

	return hasWebChangesInFiles(string(output))
}

// hasWebChangesSinceTimeForWorkspace checks if any commits since the given time
// that touch the workspace directory contain web/ file changes.
//
// This is more accurate than hasWebChangesSinceTime because it only considers
// commits that modified files in the workspace directory. This prevents false positives
// where concurrent agents (spawned around the same time) make commits that would
// incorrectly trigger visual verification requirements for this agent.
//
// If workspacePath is empty, falls back to hasWebChangesSinceTime for backward compatibility.
func hasWebChangesSinceTimeForWorkspace(projectDir string, since time.Time, workspacePath string) bool {
	// If no workspace path provided, fall back to the unscoped check
	if workspacePath == "" {
		return hasWebChangesSinceTime(projectDir, since)
	}

	sinceStr := since.UTC().Format("2006-01-02T15:04:05Z")

	// Convert workspace path to relative path from project dir for git matching
	relWorkspace := workspacePath
	if filepath.IsAbs(workspacePath) && filepath.IsAbs(projectDir) {
		rel, err := filepath.Rel(projectDir, workspacePath)
		if err == nil {
			relWorkspace = rel
		}
	}

	// Get commit hashes since spawn time that touch the workspace
	cmd := exec.Command("git", "log", "--since="+sinceStr, "--format=%H", "--", relWorkspace)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		// No commits touching workspace, or error - no web changes
		return false
	}

	// Get the commit hashes
	commitHashes := strings.Split(strings.TrimSpace(string(output)), "\n")

	// For each commit that touched the workspace, get all changed files
	var allChangedFiles []string
	for _, hash := range commitHashes {
		if hash == "" {
			continue
		}
		cmd := exec.Command("git", "show", "--name-only", "--format=", hash)
		cmd.Dir = projectDir
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		files := strings.Split(string(output), "\n")
		allChangedFiles = append(allChangedFiles, files...)
	}

	return hasWebChangesInFiles(strings.Join(allChangedFiles, "\n"))
}

// hasWebChangesInFiles checks if any files in the output are web/ files.
// This is extracted for testing.
func hasWebChangesInFiles(gitOutput string) bool {
	lines := strings.Split(gitOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if IsWebFile(line) {
			return true
		}
	}
	return false
}

// IsWebFile returns true if the file path is a web-related file.
// Matches files in web/ directory with web file extensions.
func IsWebFile(filePath string) bool {
	// Must be in web/ directory
	if !strings.HasPrefix(filePath, "web/") {
		return false
	}

	// Check for web file extensions
	webExtensions := []string{
		".svelte", ".ts", ".tsx", ".js", ".jsx",
		".css", ".scss", ".html", ".vue",
	}

	for _, ext := range webExtensions {
		if strings.HasSuffix(filePath, ext) {
			return true
		}
	}

	return false
}

// WebFileChange represents a changed web file with its diff stats.
type WebFileChange struct {
	Path         string // File path (e.g., "web/src/routes/page.svelte")
	LinesAdded   int    // Number of lines added
	LinesRemoved int    // Number of lines removed (renamed to avoid Go reserved word)
	IsNew        bool   // Whether this is a newly created file
}

// IsCSSOnlyFile returns true if the file is a pure CSS/SCSS file.
func (w WebFileChange) IsCSSOnlyFile() bool {
	return strings.HasSuffix(w.Path, ".css") || strings.HasSuffix(w.Path, ".scss")
}

// IsRouteFile returns true if the file is in a routes directory (Svelte/Next.js pattern).
func (w WebFileChange) IsRouteFile() bool {
	return strings.Contains(w.Path, "/routes/") || strings.Contains(w.Path, "/pages/")
}

// IsComponentFile returns true if the file is a component file.
func (w WebFileChange) IsComponentFile() bool {
	return strings.Contains(w.Path, "/components/") ||
		strings.Contains(w.Path, "/lib/") ||
		strings.HasSuffix(w.Path, ".svelte") ||
		strings.HasSuffix(w.Path, ".tsx") ||
		strings.HasSuffix(w.Path, ".jsx") ||
		strings.HasSuffix(w.Path, ".vue")
}

// IsLayoutFile returns true if the file is a layout file.
func (w WebFileChange) IsLayoutFile() bool {
	base := filepath.Base(w.Path)
	return strings.HasPrefix(base, "+layout") ||
		strings.HasPrefix(base, "_layout") ||
		base == "layout.svelte" ||
		base == "layout.tsx" ||
		base == "Layout.tsx" ||
		base == "Layout.svelte"
}

// TotalChanges returns the total number of line changes.
func (w WebFileChange) TotalChanges() int {
	return w.LinesAdded + w.LinesRemoved
}

// lowRiskCSSPatterns are patterns that indicate trivial CSS changes.
// These don't require visual verification.
var lowRiskCSSPatterns = []string{
	"color:", "background-color:", "border-color:", "fill:", "stroke:",
	"opacity:", "visibility:",
	"font-size:", "font-weight:", "font-family:",
	"padding:", "margin:", "gap:",
	"z-index:", "cursor:",
}

// AssessWebChangeRisk evaluates the risk level of web file changes.
// Returns the highest risk level among all changed files.
//
// Risk heuristics:
// - LOW: CSS-only files with small changes (≤10 lines), or trivial modifications
// - MEDIUM: Component changes, style files with significant changes, layout modifications
// - HIGH: New route files, new pages, major UX restructuring
func AssessWebChangeRisk(changes []WebFileChange) WebChangeRisk {
	if len(changes) == 0 {
		return WebRiskNone
	}

	maxRisk := WebRiskLow

	for _, change := range changes {
		risk := assessSingleFileRisk(change)
		if risk > maxRisk {
			maxRisk = risk
		}
	}

	return maxRisk
}

// assessSingleFileRisk determines the risk level for a single file change.
func assessSingleFileRisk(change WebFileChange) WebChangeRisk {
	// New route files are always HIGH risk - new pages need visual verification
	if change.IsNew && change.IsRouteFile() {
		return WebRiskHigh
	}

	// New layout files are HIGH risk - they affect multiple pages
	if change.IsNew && change.IsLayoutFile() {
		return WebRiskHigh
	}

	// New component files are MEDIUM risk
	if change.IsNew && change.IsComponentFile() {
		return WebRiskMedium
	}

	// Large changes to route files are HIGH risk
	if change.IsRouteFile() && change.TotalChanges() > 50 {
		return WebRiskHigh
	}

	// Large layout changes are HIGH risk
	if change.IsLayoutFile() && change.TotalChanges() > 20 {
		return WebRiskHigh
	}

	// CSS-only files with small changes are LOW risk
	if change.IsCSSOnlyFile() {
		if change.TotalChanges() <= 10 {
			return WebRiskLow
		}
		// Larger CSS changes are MEDIUM risk
		return WebRiskMedium
	}

	// Component modifications
	if change.IsComponentFile() {
		// Small component changes (e.g., adding a class, tweaking styles)
		if change.TotalChanges() <= 5 {
			return WebRiskLow
		}
		// Medium-sized component changes
		if change.TotalChanges() <= 30 {
			return WebRiskMedium
		}
		// Large component changes
		return WebRiskHigh
	}

	// Default: MEDIUM risk for unclassified web files
	return WebRiskMedium
}

// GetWebChangesWithStats returns web file changes with their diff stats.
// Uses git diff --numstat to get line counts for each changed file.
func GetWebChangesWithStats(projectDir, workspacePath string) ([]WebFileChange, error) {
	spawnTime := spawn.ReadSpawnTime(workspacePath)

	if spawnTime.IsZero() {
		// Fallback to last 5 commits if no spawn time
		return getWebChangesFromRecentCommits(projectDir)
	}

	return getWebChangesSinceTimeForWorkspace(projectDir, spawnTime, workspacePath)
}

// getWebChangesFromRecentCommits gets web file changes from the last 5 commits.
func getWebChangesFromRecentCommits(projectDir string) ([]WebFileChange, error) {
	// Get files changed in last 5 commits with stats
	cmd := exec.Command("git", "diff", "--numstat", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Try with fewer commits
		cmd = exec.Command("git", "diff", "--numstat", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return nil, err
		}
	}

	return parseNumstatOutput(string(output), projectDir)
}

// getWebChangesSinceTimeForWorkspace gets web file changes since spawn time for this workspace.
func getWebChangesSinceTimeForWorkspace(projectDir string, since time.Time, workspacePath string) ([]WebFileChange, error) {
	// If no workspace path, use unscoped check
	if workspacePath == "" {
		return getWebChangesSinceTime(projectDir, since)
	}

	sinceStr := since.UTC().Format("2006-01-02T15:04:05Z")

	// Convert workspace path to relative path
	relWorkspace := workspacePath
	if filepath.IsAbs(workspacePath) && filepath.IsAbs(projectDir) {
		rel, err := filepath.Rel(projectDir, workspacePath)
		if err == nil {
			relWorkspace = rel
		}
	}

	// Get commit hashes that touch the workspace
	cmd := exec.Command("git", "log", "--since="+sinceStr, "--format=%H", "--", relWorkspace)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		return nil, nil // No commits touching workspace
	}

	commitHashes := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Collect all changes from commits that touched workspace
	var allChanges []WebFileChange
	seen := make(map[string]bool)

	for _, hash := range commitHashes {
		if hash == "" {
			continue
		}

		// Get numstat for this commit
		cmd := exec.Command("git", "show", "--numstat", "--format=", hash)
		cmd.Dir = projectDir
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		changes, err := parseNumstatOutput(string(output), projectDir)
		if err != nil {
			continue
		}

		// Merge changes, taking max values for duplicates
		for _, change := range changes {
			if seen[change.Path] {
				// Find and update existing
				for i := range allChanges {
					if allChanges[i].Path == change.Path {
						allChanges[i].LinesAdded += change.LinesAdded
						allChanges[i].LinesRemoved += change.LinesRemoved
						break
					}
				}
			} else {
				seen[change.Path] = true
				allChanges = append(allChanges, change)
			}
		}
	}

	return allChanges, nil
}

// getWebChangesSinceTime gets web file changes since a specific time.
func getWebChangesSinceTime(projectDir string, since time.Time) ([]WebFileChange, error) {
	sinceStr := since.UTC().Format("2006-01-02T15:04:05Z")

	// Get commit hashes since spawn time
	cmd := exec.Command("git", "log", "--since="+sinceStr, "--format=%H")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		return nil, nil
	}

	commitHashes := strings.Split(strings.TrimSpace(string(output)), "\n")

	var allChanges []WebFileChange
	seen := make(map[string]bool)

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

		changes, err := parseNumstatOutput(string(output), projectDir)
		if err != nil {
			continue
		}

		for _, change := range changes {
			if !seen[change.Path] {
				seen[change.Path] = true
				allChanges = append(allChanges, change)
			}
		}
	}

	return allChanges, nil
}

// parseNumstatOutput parses git diff --numstat output into WebFileChange structs.
// Format: added<TAB>removed<TAB>filepath
// Binary files show "-" for counts.
func parseNumstatOutput(output string, projectDir string) ([]WebFileChange, error) {
	var changes []WebFileChange

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
		if !IsWebFile(filePath) {
			continue
		}

		// Parse line counts (may be "-" for binary files)
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

		// Check if file is new by looking for it in the first commit
		isNew := isNewFile(projectDir, filePath)

		changes = append(changes, WebFileChange{
			Path:         filePath,
			LinesAdded:   added,
			LinesRemoved: removed,
			IsNew:        isNew,
		})
	}

	return changes, nil
}

// isNewFile checks if a file was created (not modified) by checking git status.
func isNewFile(projectDir, filePath string) bool {
	// Check if file exists in HEAD~1
	cmd := exec.Command("git", "cat-file", "-e", "HEAD~1:"+filePath)
	cmd.Dir = projectDir
	err := cmd.Run()
	// If error, file didn't exist before → it's new
	return err != nil
}

// HasVisualVerificationEvidence checks beads comments for evidence of visual verification.
// Returns true if any comment mentions screenshots, visual verification, or browser testing.
func HasVisualVerificationEvidence(comments []Comment) (bool, []string) {
	var evidence []string

	for _, comment := range comments {
		for _, pattern := range visualEvidencePatterns {
			if pattern.MatchString(comment.Text) {
				// Extract a snippet around the match for evidence
				matches := pattern.FindString(comment.Text)
				if matches != "" {
					evidence = append(evidence, matches)
				}
			}
		}
	}

	return len(evidence) > 0, evidence
}

// HasHumanApproval checks beads comments for explicit human/orchestrator approval.
// Returns true if any comment contains an explicit approval marker.
// These markers are designed to be used by human orchestrators, not agents.
func HasHumanApproval(comments []Comment) (bool, []string) {
	var approvals []string

	for _, comment := range comments {
		for _, pattern := range humanApprovalPatterns {
			if pattern.MatchString(comment.Text) {
				matches := pattern.FindString(comment.Text)
				if matches != "" {
					approvals = append(approvals, matches)
				}
			}
		}
	}

	return len(approvals) > 0, approvals
}

// HasVisualVerificationInSynthesis checks SYNTHESIS.md for visual verification evidence.
// Looks in the Evidence section for screenshot/visual verification mentions.
func HasVisualVerificationInSynthesis(workspacePath string) (bool, []string) {
	if workspacePath == "" {
		return false, nil
	}

	synthesis, err := ParseSynthesis(workspacePath)
	if err != nil {
		return false, nil
	}

	var evidence []string

	// Check Evidence section
	for _, pattern := range visualEvidencePatterns {
		if pattern.MatchString(synthesis.Evidence) {
			matches := pattern.FindString(synthesis.Evidence)
			if matches != "" {
				evidence = append(evidence, "Evidence: "+matches)
			}
		}
	}

	// Also check TLDR
	for _, pattern := range visualEvidencePatterns {
		if pattern.MatchString(synthesis.TLDR) {
			matches := pattern.FindString(synthesis.TLDR)
			if matches != "" {
				evidence = append(evidence, "TLDR: "+matches)
			}
		}
	}

	return len(evidence) > 0, evidence
}

// screenshotExtensions defines file extensions that are considered screenshot files.
var screenshotExtensions = []string{".png", ".jpg", ".jpeg", ".webp", ".gif"}

// HasScreenshotFilesInWorkspace checks if the workspace's screenshots/ directory
// contains any image files (screenshots captured by the agent).
// Returns true if any screenshot files exist, along with the list of file names.
func HasScreenshotFilesInWorkspace(workspacePath string) (bool, []string) {
	if workspacePath == "" {
		return false, nil
	}

	screenshotsDir := filepath.Join(workspacePath, "screenshots")

	// Check if screenshots directory exists
	stat, err := os.Stat(screenshotsDir)
	if err != nil || !stat.IsDir() {
		return false, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(screenshotsDir)
	if err != nil {
		return false, nil
	}

	var screenshotFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		lower := strings.ToLower(name)
		for _, ext := range screenshotExtensions {
			if strings.HasSuffix(lower, ext) {
				screenshotFiles = append(screenshotFiles, name)
				break
			}
		}
	}

	return len(screenshotFiles) > 0, screenshotFiles
}

// VerifyVisualVerification checks if visual verification was performed for web/ changes.
// This is a gate that blocks completion if web/ files were modified without visual verification evidence
// AND explicit human approval.
//
// The verification passes if:
// 1. No web/ files were modified in recent commits, OR
// 2. The skill is not a UI-focused skill (architect, investigation, debugging, etc.), OR
// 3. Visual verification evidence is found AND human approval is present
//
// This skill-aware approach prevents false positives from non-UI skills that incidentally
// modify web/ files as part of broader work. Only feature-impl (and similar UI-focused skills)
// require visual verification for web/ changes.
//
// Evidence includes:
// - Screenshots mentioned (screenshot, captured image)
// - Visual verification mentioned (visually verified, UI verified)
// - Browser testing mentioned (playwright, browser_take_screenshot, tested in browser)
//
// Human Approval includes:
// - ✅ APPROVED marker
// - UI APPROVED / VISUAL APPROVED
// - human_approved: true
// - orchestrator_approved: true
// - "I approve the UI/visual/changes"
func VerifyVisualVerification(beadsID, workspacePath, projectDir string) VisualVerificationResult {
	return VerifyVisualVerificationWithComments(beadsID, workspacePath, projectDir, nil)
}

// VerifyVisualVerificationWithComments is like VerifyVisualVerification but accepts pre-fetched comments.
// If comments is nil, comments will be fetched from beads API.
func VerifyVisualVerificationWithComments(beadsID, workspacePath, projectDir string, comments []Comment) VisualVerificationResult {
	result := VisualVerificationResult{Passed: true, RiskLevel: WebRiskNone}

	// Check if web/ files were modified by this agent (scoped by spawn time)
	result.HasWebChanges = HasWebChangesForAgent(projectDir, workspacePath)

	// No web changes = no verification needed
	if !result.HasWebChanges {
		return result
	}

	// Check skill type - only UI-focused skills require visual verification
	skillName, _ := ExtractSkillNameFromSpawnContext(workspacePath)
	if !IsSkillRequiringVisualVerification(skillName) {
		// Non-UI skill modifying web/ files - this is incidental, not UI work
		// Skip visual verification requirement
		result.Warnings = append(result.Warnings,
			"web/ files modified by non-UI skill ("+skillName+") - visual verification not required")
		return result
	}

	// UI-focused skill (feature-impl) - assess risk level of web changes
	webChanges, err := GetWebChangesWithStats(projectDir, workspacePath)
	if err != nil {
		result.Warnings = append(result.Warnings, "failed to get web change stats: "+err.Error())
		// Fall back to requiring verification if we can't assess risk
		result.RiskLevel = WebRiskMedium
	} else {
		result.RiskLevel = AssessWebChangeRisk(webChanges)
	}

	// LOW risk changes don't require visual verification
	if !result.RiskLevel.RequiresVisualVerification() {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("web/ files modified with %s risk - visual verification not required", result.RiskLevel))
		// Add details about what was changed
		for _, change := range webChanges {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("  %s: +%d/-%d lines", change.Path, change.LinesAdded, change.LinesRemoved))
		}
		return result
	}

	// MEDIUM/HIGH risk - need visual verification evidence AND human approval

	// Check beads comments for evidence and approval (use pre-fetched if available)
	if comments == nil {
		var err error
		comments, err = GetComments(beadsID)
		if err != nil {
			result.Warnings = append(result.Warnings, "failed to get beads comments: "+err.Error())
			comments = nil // Reset to indicate we couldn't fetch
		}
	}

	if comments != nil {
		// Check for visual verification evidence
		hasEvidence, evidence := HasVisualVerificationEvidence(comments)
		if hasEvidence {
			result.HasEvidence = true
			result.Evidence = append(result.Evidence, evidence...)
		}

		// Check for human approval
		hasApproval, approvals := HasHumanApproval(comments)
		if hasApproval {
			result.HasHumanApproval = true
			result.Evidence = append(result.Evidence, approvals...)
		}
	}

	// Check SYNTHESIS.md for evidence
	if workspacePath != "" {
		hasEvidence, evidence := HasVisualVerificationInSynthesis(workspacePath)
		if hasEvidence {
			result.HasEvidence = true
			result.Evidence = append(result.Evidence, evidence...)
		}
	}

	// Check for actual screenshot files in workspace
	if workspacePath != "" {
		hasScreenshots, screenshotFiles := HasScreenshotFilesInWorkspace(workspacePath)
		if hasScreenshots {
			result.HasEvidence = true
			for _, file := range screenshotFiles {
				result.Evidence = append(result.Evidence, "Screenshot file: "+file)
			}
		}
	}

	// Determine what's missing
	if !result.HasEvidence {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("web/ files modified with %s risk - visual verification required", result.RiskLevel),
			"Agent must capture screenshot or mention visual verification in beads comment",
			"Example: bd comment <id> \"Visual verification: screenshot captured showing [description]\"",
		)
	} else if !result.HasHumanApproval {
		// Evidence exists but needs human approval
		result.Passed = false
		result.NeedsApproval = true
		result.Errors = append(result.Errors,
			fmt.Sprintf("web/ files modified with %s risk - visual evidence found but requires human approval", result.RiskLevel),
			"Use: orch complete <id> --approve   OR",
			"Add approval comment: bd comment <id> \"✅ APPROVED\"",
		)
	}

	return result
}

// VerifyVisualVerificationForCompletion is a convenience function for use in orch complete.
// Returns nil if no verification is needed (no web changes) or if verification passes.
func VerifyVisualVerificationForCompletion(beadsID, workspacePath, projectDir string) *VisualVerificationResult {
	return VerifyVisualVerificationForCompletionWithComments(beadsID, workspacePath, projectDir, nil)
}

// VerifyVisualVerificationForCompletionWithComments is like VerifyVisualVerificationForCompletion but accepts pre-fetched comments.
// If comments is nil, comments will be fetched from beads API.
func VerifyVisualVerificationForCompletionWithComments(beadsID, workspacePath, projectDir string, comments []Comment) *VisualVerificationResult {
	result := VerifyVisualVerificationWithComments(beadsID, workspacePath, projectDir, comments)

	// Return nil if no web changes - no action needed
	if !result.HasWebChanges {
		return nil
	}

	return &result
}
