// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// TestEvidenceResult represents the result of checking for test execution evidence.
type TestEvidenceResult struct {
	Passed                 bool     // Whether verification passed
	HasCodeChanges         bool     // Whether code files were changed (requires test evidence)
	HasTestEvidence        bool     // Whether test execution evidence was found
	RequiresIntegration    bool     // Whether behavioral acceptance criteria require integration/E2E evidence
	HasIntegrationEvidence bool     // Whether integration/E2E evidence was found
	MarkdownOnlyExempt     bool     // Whether exempted due to markdown-only changes
	OutsideProjectExempt   bool     // Whether exempted due to files outside project
	Errors                 []string // Error messages (blocking)
	Warnings               []string // Warning messages (non-blocking)
	Evidence               []string // Evidence found (for debugging)
	IntegrationEvidence    []string // Integration/E2E-specific evidence found
	BehavioralCriteria     []string // Behavioral criteria that triggered integration requirement
	SkillName              string   // Skill that was used
}

// Skills that require test execution evidence before completion.
// Only implementation-focused skills need test verification.
// Investigation/research skills produce artifacts, not code changes.
var skillsRequiringTestEvidence = map[string]bool{
	"feature-impl":         true, // Primary implementation skill
	"systematic-debugging": true, // Debug fixes should be tested
	"reliability-testing":  true, // Testing skill should document tests
}

// Skills explicitly excluded from test evidence requirements.
// These skills may modify code incidentally but don't require test evidence.
var skillsExcludedFromTestEvidence = map[string]bool{
	"investigation":  true, // Research skill, produces investigations
	"architect":      true, // Design skill, produces decisions
	"research":       true, // External research, no code changes
	"design-session": true, // Scoping skill, produces epics
	"codebase-audit": true, // Audit skill, produces reports
	"issue-creation": true, // Triage skill, creates issues
	"writing-skills": true, // Meta skill, modifies skills
}

// IsSkillRequiringTestEvidence determines if a skill requires test execution evidence.
//
// The logic is:
// 1. If skill is explicitly excluded (investigation, architect, etc.) -> false
// 2. If skill is explicitly included (feature-impl, debugging) -> true
// 3. If skill is unknown -> false (permissive default)
func IsSkillRequiringTestEvidence(skillName string) bool {
	if skillName == "" {
		return false
	}

	skillName = strings.ToLower(skillName)

	// Check explicit exclusions first
	if skillsExcludedFromTestEvidence[skillName] {
		return false
	}

	// Check explicit inclusions
	if skillsRequiringTestEvidence[skillName] {
		return true
	}

	// Unknown skill - be permissive
	return false
}

// testEvidencePatterns defines regex patterns that indicate test execution was performed.
// These patterns match actual test output, not just claims like "tests pass".
var testEvidencePatterns = []*regexp.Regexp{
	// Go test output patterns
	regexp.MustCompile(`(?i)go\s+test\s+.*\s*[-–—]?\s*PASS`),    // "go test ./... - PASS"
	regexp.MustCompile(`(?i)ok\s+\S+\s+\d+\.\d+s`),              // "ok  package/name  0.123s"
	regexp.MustCompile(`(?i)PASS:\s*\d+`),                       // "PASS: 15" (test count)
	regexp.MustCompile(`(?i)---\s*PASS:\s*\w+`),                 // "--- PASS: TestName"
	regexp.MustCompile(`(?i)FAIL:\s*\d+`),                       // "FAIL: 2" (captures failures too)
	regexp.MustCompile(`(?i)\(\d+\s+tests?\s+in\s+\d+\.\d+s\)`), // "(12 tests in 0.8s)"
	regexp.MustCompile(`(?i)\d+\s+tests?\s+passed`),             // "15 tests passed" (requires count)
	regexp.MustCompile(`(?i)all\s+\d+\s+tests?\s+pass`),         // "all 15 tests pass" (requires count)

	// npm/yarn/bun test output patterns
	regexp.MustCompile(`(?i)npm\s+test\s*[-–—]?\s*(passed|success)`),  // "npm test - passed"
	regexp.MustCompile(`(?i)yarn\s+test\s*[-–—]?\s*(passed|success)`), // "yarn test - passed"
	regexp.MustCompile(`(?i)bun\s+test\s*[-–—]?\s*(passed|success)`),  // "bun test - passed"
	regexp.MustCompile(`(?i)\d+\s+pass(ed|ing)?[,\s]+\d+\s+fail`),     // "15 passing, 0 failing"
	regexp.MustCompile(`(?i)Tests:\s+\d+\s+passed`),                   // "Tests: 15 passed"
	regexp.MustCompile(`(?i)Test\s+Suites?:\s+\d+\s+passed`),          // "Test Suites: 5 passed"

	// pytest output patterns
	regexp.MustCompile(`(?i)pytest\s*[-–—]?\s*\d+\s+passed`),                       // "pytest - 15 passed"
	regexp.MustCompile(`(?i)==+\s+\d+\s+passed`),                                   // "======= 15 passed"
	regexp.MustCompile(`(?i)\d+\s+passed,?\s*\d*\s*(?:warnings?|errors?|failed)?`), // "15 passed, 0 failed"

	// cargo test output patterns
	regexp.MustCompile(`(?i)cargo\s+test\s*[-–—]?\s*(ok|passed)`), // "cargo test - ok"
	regexp.MustCompile(`(?i)test\s+result:\s+ok`),                 // "test result: ok"
	regexp.MustCompile(`(?i)\d+\s+passed;\s+\d+\s+failed`),        // "15 passed; 0 failed"

	// Generic test execution evidence
	regexp.MustCompile(`(?i)Tests?:\s*(?:go\s+test|npm\s+test|pytest|cargo\s+test|yarn\s+test|bun\s+test)`), // "Tests: go test ..."
	regexp.MustCompile(`(?i)ran\s+\d+\s+tests?\s+in\s+\d+`),                                                 // "ran 15 tests in 2.3s"
	regexp.MustCompile(`(?i)test\s+suite\s+(?:passed|completed)`),                                           // "test suite passed"
	regexp.MustCompile(`(?i)all\s+\d+\s+tests?\s+(?:passed|succeeded)`),                                     // "all 15 tests passed"

	// Playwright/e2e test patterns
	regexp.MustCompile(`(?i)playwright\s+test.*\d+\s+passed`), // "playwright test - 5 passed"
	regexp.MustCompile(`(?i)\d+\s+passed\s+\(\d+[smh]\)`),     // "5 passed (2s)"
}

// falsePositivePatterns defines patterns that indicate a claim without evidence.
// These should NOT count as test evidence.
// The key insight: vague claims lack quantifiable output (counts, timing, specific output).
var falsePositivePatterns = []*regexp.Regexp{
	// Simple vague claims without counts or details
	regexp.MustCompile(`(?i)^tests?\s+pass(ed)?\s*$`),            // Just "tests pass" or "tests passed"
	regexp.MustCompile(`(?i)^all\s+tests?\s+pass(ed)?\s*$`),      // "all tests pass" without count
	regexp.MustCompile(`(?i)verified\s+tests?\s+pass`),           // "verified tests pass" (claim)
	regexp.MustCompile(`(?i)tests?\s+should\s+pass`),             // "tests should pass" (expectation)
	regexp.MustCompile(`(?i)assuming\s+tests?\s+pass`),           // "assuming tests pass" (assumption)
	regexp.MustCompile(`(?i)tests?\s+will\s+pass`),               // "tests will pass" (prediction)
	regexp.MustCompile(`(?i)tests?\s+(?:are\s+)?passing`),        // "tests passing" or "tests are passing" (state claim)
	regexp.MustCompile(`(?i)^the\s+tests?\s+pass(ed)?\s*$`),      // "the tests pass"
	regexp.MustCompile(`(?i)confirmed?\s+tests?\s+pass`),         // "confirmed tests pass" (claim)
	regexp.MustCompile(`(?i)tests?\s+(?:have\s+)?succeeded`),     // "tests succeeded" without details
	regexp.MustCompile(`(?i)tests?\s+completed?\s+successfully`), // "tests completed successfully"
	regexp.MustCompile(`(?i)^all\s+tests?\s+pass(ed|ing)?\b`),    // "all tests pass" at start of string (without count)
}

// behavioralAcceptancePatterns detect criteria that describe runtime behavior.
// These patterns are used to separate behavioral acceptance criteria from structural tasks.
var behavioralAcceptancePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(user|admin|agent|orchestrator|system|workflow|pipeline|etl|feature)\b.{0,50}\b(can|cannot|can't|does|doesn't|should|must|will|won't|skips?|triggers?|survives?|persists?|works?|fails?)\b`),
	regexp.MustCompile(`(?i)\b(when|if)\b.+\b(then|should|must|will)\b`),
	regexp.MustCompile(`(?i)\b\w+\b\s+(survives?|triggers?|skips?)\b\s+\w+`),
	regexp.MustCompile(`(?i)\b(end[- ]to[- ]end|e2e|full\s+workflow|real[- ]world|production\s+path)\b`),
}

// integrationEvidencePatterns detect explicit integration/E2E validation output.
// This is stricter than generic test evidence and is required for behavioral acceptance work.
var integrationEvidencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(integration|e2e|end[- ]to[- ]end|smoke)\s+(test|tests|verification)\b.*\b(pass|passed|passing|working|works|verified|successful|succeeded)\b`),
	regexp.MustCompile(`(?i)\bTests?:\s*(go\s+test|npm\s+test|pytest|playwright\s+test|cargo\s+test).*(integration|e2e|end[- ]to[- ]end|smoke)`),
	regexp.MustCompile(`(?i)\bplaywright\s+test\b.*\b(pass|passed|passing)\b`),
	regexp.MustCompile(`(?i)\b(full\s+workflow|real\s+flow|production\s+path)\b.*\b(pass|passed|verified|working|works)\b`),
}

var numberedListPattern = regexp.MustCompile(`^\d+[\.)]\s+`)

// HasTestExecutionEvidence checks beads comments for evidence of test execution.
// Returns true if any comment contains actual test output patterns.
// Returns false for vague claims like "tests pass" without evidence.
func HasTestExecutionEvidence(comments []Comment) (bool, []string) {
	var evidence []string

	for _, comment := range comments {
		// Skip false positives
		isFalsePositive := false
		for _, fp := range falsePositivePatterns {
			if fp.MatchString(comment.Text) {
				isFalsePositive = true
				break
			}
		}
		if isFalsePositive {
			continue
		}

		// Check for valid test evidence patterns
		for _, pattern := range testEvidencePatterns {
			if pattern.MatchString(comment.Text) {
				matches := pattern.FindString(comment.Text)
				if matches != "" {
					evidence = append(evidence, matches)
				}
			}
		}
	}

	return len(evidence) > 0, evidence
}

// HasIntegrationTestEvidence checks beads comments for integration/E2E evidence.
func HasIntegrationTestEvidence(comments []Comment) (bool, []string) {
	var evidence []string

	for _, comment := range comments {
		for _, pattern := range integrationEvidencePatterns {
			if pattern.MatchString(comment.Text) {
				matches := pattern.FindString(comment.Text)
				if matches != "" {
					evidence = append(evidence, matches)
				}
			}
		}
	}

	return len(evidence) > 0, evidence
}

// extractTaskFromSpawnContext extracts the top TASK section from SPAWN_CONTEXT.md.
func extractTaskFromSpawnContext(workspacePath string) string {
	if workspacePath == "" {
		return ""
	}

	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	content, err := os.ReadFile(spawnContextPath)
	if err != nil {
		return ""
	}

	text := string(content)
	taskIdx := strings.Index(text, "TASK:")
	if taskIdx == -1 {
		return ""
	}

	taskSection := text[taskIdx+len("TASK:"):]
	markers := []string{
		"\n## DESIGN REFERENCE",
		"\nSPAWN TIER:",
		"\n## REPRODUCTION (BUG FIX)",
		"\n## ORCHESTRATOR NOTES",
		"\nCONTEXT: [See task description]",
		"\nPROJECT_DIR:",
		"\nAUTHORITY:",
	}

	end := len(taskSection)
	for _, marker := range markers {
		if idx := strings.Index(taskSection, marker); idx != -1 && idx < end {
			end = idx
		}
	}

	return strings.TrimSpace(taskSection[:end])
}

// extractAcceptanceCriteria returns criterion-like lines from an acceptance criteria section.
func extractAcceptanceCriteria(taskText string) []string {
	lower := strings.ToLower(taskText)
	idx := strings.Index(lower, "acceptance criteria")
	if idx == -1 {
		return nil
	}

	section := taskText[idx:]
	if headingIdx := strings.Index(section, "\n## "); headingIdx != -1 {
		section = section[:headingIdx]
	}

	lines := strings.Split(section, "\n")
	var criteria []string
	for i, line := range lines {
		if i == 0 {
			continue // heading line
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		trimmed = numberedListPattern.ReplaceAllString(trimmed, "")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			criteria = append(criteria, trimmed)
		}
	}

	return criteria
}

// DetectBehavioralAcceptanceCriteria identifies behavioral acceptance criteria from task text.
func DetectBehavioralAcceptanceCriteria(taskText string) (bool, []string) {
	if strings.TrimSpace(taskText) == "" {
		return false, nil
	}

	candidates := extractAcceptanceCriteria(taskText)
	if len(candidates) == 0 {
		for _, line := range strings.Split(taskText, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				candidates = append(candidates, trimmed)
			}
		}
	}

	matched := make([]string, 0)
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		for _, pattern := range behavioralAcceptancePatterns {
			if pattern.MatchString(candidate) {
				if !seen[candidate] {
					matched = append(matched, candidate)
					seen[candidate] = true
				}
				break
			}
		}
	}

	return len(matched) > 0, matched
}

// requiresIntegrationEvidenceForBehavioralAcceptance determines if integration evidence is required.
func requiresIntegrationEvidenceForBehavioralAcceptance(skillName, workspacePath string) (bool, []string) {
	if strings.ToLower(skillName) != "feature-impl" {
		return false, nil
	}

	taskText := extractTaskFromSpawnContext(workspacePath)
	if taskText == "" {
		return false, nil
	}

	return DetectBehavioralAcceptanceCriteria(taskText)
}

// codeFileExtensions defines file extensions that are considered "code files"
// that typically require test verification when modified.
var codeFileExtensions = []string{
	".go", ".py", ".js", ".ts", ".jsx", ".tsx",
	".rs", ".rb", ".java", ".kt", ".swift",
	".c", ".cpp", ".h", ".hpp", ".cs",
	".svelte", ".vue", // UI components
}

// HasCodeChangesInRecentCommits checks if any code files were modified
// in recent commits that would require test verification.
//
// DEPRECATED: Use HasCodeChangesSinceSpawn for accurate per-agent change detection.
// This function uses HEAD~5 which includes changes from OTHER agents' commits.
func HasCodeChangesInRecentCommits(projectDir string) bool {
	files, err := getChangedFiles(projectDir, "")
	if err != nil {
		return false
	}

	return hasCodeChangesInFiles(strings.Join(files, "\n"))
}

// HasCodeChangesSinceSpawn checks if any code files were modified
// in commits since the given spawn time.
//
// DEPRECATED: Use HasCodeChangesSinceSpawnForWorkspace for accurate per-agent change detection.
// This function considers ALL commits since spawn time, including concurrent agents' commits.
func HasCodeChangesSinceSpawn(projectDir string, spawnTime time.Time) bool {
	return HasCodeChangesSinceSpawnForWorkspace(projectDir, spawnTime, "")
}

// HasCodeChangesSinceSpawnForWorkspace checks if any code files were modified
// in commits since the given spawn time that are associated with the given workspace.
//
// This is more accurate than HasCodeChangesSinceSpawn because it only considers
// commits that modified files in the workspace directory. This prevents false positives
// where markdown-only changes trigger the test evidence gate because concurrent agents
// (spawned around the same time) made commits with code changes.
//
// If workspacePath is empty, it falls back to checking all commits since spawn time.
func HasCodeChangesSinceSpawnForWorkspace(projectDir string, spawnTime time.Time, workspacePath string) bool {
	if spawnTime.IsZero() {
		// Fall back to recent commits if spawn time is unavailable
		return HasCodeChangesInRecentCommits(projectDir)
	}

	// Use git log with --since to get commits since spawn time
	sinceStr := spawnTime.Format(time.RFC3339)

	// If workspacePath is provided, filter to commits that touch the workspace
	if workspacePath != "" {
		return hasCodeChangesInWorkspaceCommits(projectDir, sinceStr, workspacePath)
	}

	files, err := getChangedFiles(projectDir, sinceStr)
	if err != nil {
		// Fall back to recent commits if git log fails
		return HasCodeChangesInRecentCommits(projectDir)
	}

	return hasCodeChangesInFiles(strings.Join(files, "\n"))
}

// hasCodeChangesInWorkspaceCommits checks for code changes in commits that modified
// files within the given workspace directory. This filters out commits from concurrent
// agents that happened to occur after the spawn time but weren't made by this agent.
func hasCodeChangesInWorkspaceCommits(projectDir, sinceStr, workspacePath string) bool {
	commitHashes, err := getCommitHashes(projectDir, workspacePath, sinceStr)
	if err != nil || len(commitHashes) == 0 {
		// No commits touching workspace, or error - no code changes
		return false
	}

	// For each commit that touched the workspace, get all changed files
	var allChangedFiles []string
	for _, hash := range commitHashes {
		files, err := getFileChangesForCommit(projectDir, hash)
		if err != nil {
			continue
		}
		allChangedFiles = append(allChangedFiles, files...)
	}

	return hasCodeChangesInFiles(strings.Join(allChangedFiles, "\n"))
}

// hasCodeChangesInFiles checks if any files in the output are code files.
func hasCodeChangesInFiles(gitOutput string) bool {
	lines := strings.Split(gitOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isCodeFile(line) {
			return true
		}
	}
	return false
}

// isCodeFile checks if a file path is a code file based on extension.
func isCodeFile(filePath string) bool {
	// Get base filename for test file checks
	baseName := filePath
	if idx := strings.LastIndex(filePath, "/"); idx != -1 {
		baseName = filePath[idx+1:]
	}

	// Skip test files themselves (they don't require tests of tests)
	if strings.Contains(filePath, "_test.go") ||
		strings.Contains(filePath, ".test.") ||
		strings.Contains(filePath, ".spec.") ||
		strings.HasSuffix(filePath, "_test.py") ||
		strings.HasPrefix(baseName, "test_") {
		return false
	}

	for _, ext := range codeFileExtensions {
		if strings.HasSuffix(filePath, ext) {
			return true
		}
	}
	return false
}

// isMarkdownFile checks if a file path is a markdown file.
func isMarkdownFile(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".md")
}

// isFileOutsideProject checks if a file path is outside the project directory.
// Returns true for absolute paths not under projectDir, or relative paths starting with "../".
func isFileOutsideProject(filePath, projectDir string) bool {
	if projectDir == "" {
		return false
	}

	// Handle absolute paths
	if filepath.IsAbs(filePath) {
		// Check if file is under project dir
		rel, err := filepath.Rel(projectDir, filePath)
		if err != nil {
			return true // Can't determine relationship, treat as outside
		}
		// If relative path starts with "..", it's outside the project
		return strings.HasPrefix(rel, "..")
	}

	// For relative paths, check if they start with ".."
	return strings.HasPrefix(filePath, "..")
}

// getChangedFilesSinceSpawn returns all files changed since spawn time that are
// associated with the given workspace. Returns empty slice on error.
func getChangedFilesSinceSpawn(projectDir string, spawnTime time.Time, workspacePath string) []string {
	if spawnTime.IsZero() || projectDir == "" {
		return nil
	}

	sinceStr := spawnTime.Format(time.RFC3339)

	// If workspacePath provided, filter to commits that touch the workspace
	if workspacePath != "" {
		return getChangedFilesInWorkspaceCommits(projectDir, sinceStr, workspacePath)
	}

	files, err := getChangedFiles(projectDir, sinceStr)
	if err != nil {
		return nil
	}

	return files
}

// getChangedFilesInWorkspaceCommits returns files changed in commits that modified
// files within the given workspace directory.
func getChangedFilesInWorkspaceCommits(projectDir, sinceStr, workspacePath string) []string {
	commitHashes, err := getCommitHashes(projectDir, workspacePath, sinceStr)
	if err != nil || len(commitHashes) == 0 {
		return nil
	}

	// For each commit that touched the workspace, get all changed files
	var allChangedFiles []string
	for _, hash := range commitHashes {
		files, err := getFileChangesForCommit(projectDir, hash)
		if err != nil {
			continue
		}
		allChangedFiles = append(allChangedFiles, files...)
	}

	return allChangedFiles
}

// parseFileList parses git output into a list of file paths.
func parseFileList(gitOutput string) []string {
	var files []string
	lines := strings.Split(gitOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

// areAllFilesMarkdown checks if all files in the list are markdown files.
// Returns (true, count) if all files are .md, (false, count) otherwise.
// Returns (true, 0) for empty list.
func areAllFilesMarkdown(files []string) (bool, int) {
	if len(files) == 0 {
		return true, 0
	}

	for _, f := range files {
		if !isMarkdownFile(f) {
			return false, len(files)
		}
	}
	return true, len(files)
}

// areAllFilesOutsideProject checks if all files are outside the project directory.
// Returns (true, count) if all files are outside, (false, count) otherwise.
// Returns (true, 0) for empty list.
func areAllFilesOutsideProject(files []string, projectDir string) (bool, int) {
	if len(files) == 0 || projectDir == "" {
		return true, 0
	}

	for _, f := range files {
		if !isFileOutsideProject(f, projectDir) {
			return false, len(files)
		}
	}
	return true, len(files)
}

// VerifyTestEvidence checks if test execution evidence exists for code changes.
// This is a gate that blocks completion if code files were modified without
// test execution evidence in beads comments.
//
// The verification passes if:
// 1. No code files were modified in recent commits, OR
// 2. The skill is not an implementation-focused skill, OR
// 3. Test execution evidence is found in beads comments
//
// Evidence must show actual test output (pass counts, timing, framework output)
// not just claims like "tests pass".
func VerifyTestEvidence(beadsID, workspacePath, projectDir string) TestEvidenceResult {
	return VerifyTestEvidenceWithComments(beadsID, workspacePath, projectDir, nil)
}

// VerifyTestEvidenceWithComments is like VerifyTestEvidence but accepts pre-fetched comments.
// If comments is nil, comments will be fetched from beads API.
func VerifyTestEvidenceWithComments(beadsID, workspacePath, projectDir string, comments []Comment) TestEvidenceResult {
	return verifyTestEvidenceInternal(beadsID, workspacePath, projectDir, "", comments)
}

// verifyTestEvidenceInternal is the shared implementation for test evidence verification.
// If skillOverride is non-empty, it's used instead of extracting from SPAWN_CONTEXT.md.
func verifyTestEvidenceInternal(beadsID, workspacePath, projectDir, skillOverride string, comments []Comment) TestEvidenceResult {
	result := TestEvidenceResult{Passed: true}

	// Extract skill name for skill-based gating
	var skillName string
	if skillOverride != "" {
		skillName = skillOverride
	} else {
		skillName = ResolveSkillName(workspacePath, projectDir, beadsID)
	}
	result.SkillName = skillName
	requiresIntegration, behavioralCriteria := requiresIntegrationEvidenceForBehavioralAcceptance(skillName, workspacePath)
	result.RequiresIntegration = requiresIntegration
	result.BehavioralCriteria = behavioralCriteria

	// Check if skill requires test evidence
	if !IsSkillRequiringTestEvidence(skillName) {
		result.Warnings = append(result.Warnings,
			"skill '"+skillName+"' does not require test evidence")
		return result
	}

	// Get spawn time for change detection
	spawnTime := spawn.ReadSpawnTime(workspacePath)

	// Get all changed files for exemption checks
	changedFiles := getChangedFilesSinceSpawn(projectDir, spawnTime, workspacePath)

	// Exemption 1: Markdown-only changes
	// If ALL modified files are .md files, no test harness applies
	if allMd, count := areAllFilesMarkdown(changedFiles); allMd && count > 0 {
		result.MarkdownOnlyExempt = true
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("markdown-only changes (%d .md files) - test evidence not required", count))
		return result
	}

	// Exemption 2: Files outside project directory
	// If ALL modified files are outside projectDir (e.g., ~/.claude/skills/...),
	// there's no test harness to run
	if allOutside, count := areAllFilesOutsideProject(changedFiles, projectDir); allOutside && count > 0 {
		result.OutsideProjectExempt = true
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("all changes outside project dir (%d files) - no test harness available", count))
		return result
	}

	// Check if code files were modified since this agent was spawned
	// Using workspace-filtered commits ensures we only look at THIS agent's commits,
	// not concurrent agents. This prevents false positives where markdown-only
	// changes trigger the gate because concurrent agents' commits had code changes.
	result.HasCodeChanges = HasCodeChangesSinceSpawnForWorkspace(projectDir, spawnTime, workspacePath)

	// No code changes = no test evidence needed
	if !result.HasCodeChanges {
		result.Warnings = append(result.Warnings,
			"no code files modified - test evidence not required")
		return result
	}

	// Code changes exist - check for test evidence in beads comments
	// Use pre-fetched comments if available
	if comments == nil {
		var err error
		comments, err = GetComments(beadsID)
		if err != nil {
			result.Warnings = append(result.Warnings,
				"failed to get beads comments: "+err.Error())
			// Don't fail verification if we can't fetch comments
			return result
		}
	}

	hasEvidence, evidence := HasTestExecutionEvidence(comments)
	hasIntegrationEvidence, integrationEvidence := HasIntegrationTestEvidence(comments)
	result.HasTestEvidence = hasEvidence
	result.Evidence = evidence
	result.HasIntegrationEvidence = hasIntegrationEvidence
	result.IntegrationEvidence = integrationEvidence

	if hasIntegrationEvidence && !result.HasTestEvidence {
		result.HasTestEvidence = true
		result.Evidence = append(result.Evidence, integrationEvidence...)
	}

	if result.RequiresIntegration && !result.HasIntegrationEvidence {
		result.Passed = false
		result.Errors = append(result.Errors,
			"behavioral acceptance criteria detected but no integration/E2E evidence found in beads comments",
			"Behavioral work must include end-to-end proof, not only isolated unit tests",
			"Example: bd comment <id> 'Integration test: go test ./integration/... - PASS (3 tests in 1.4s)'",
			"Example: bd comment <id> 'E2E verification: playwright test e2e/workflow.spec.ts - 4 passed'",
		)
		if len(result.BehavioralCriteria) > 0 {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Behavioral criteria matched: %s", strings.Join(result.BehavioralCriteria, "; ")))
		}
	}

	if !result.HasTestEvidence {
		result.Passed = false
		result.Errors = append(result.Errors,
			"code files modified but no test execution evidence found in beads comments",
			"Agent must run tests and report actual output (not just 'tests pass')",
			"Example: bd comment <id> 'Tests: go test ./pkg/... - PASS (12 tests in 0.8s)'",
			"Example: bd comment <id> 'Tests: npm test - 15 passing, 0 failing'",
		)
	}

	return result
}

// VerifyTestEvidenceForCompletion is a convenience function for use in VerifyCompletionFull.
// Returns nil if no verification is needed (no code changes or non-implementation skill).
// Returns EscalationBlock level result if test evidence is missing.
func VerifyTestEvidenceForCompletion(beadsID, workspacePath, projectDir string) *TestEvidenceResult {
	return VerifyTestEvidenceForCompletionWithComments(beadsID, workspacePath, projectDir, nil)
}

// VerifyTestEvidenceForCompletionWithComments is like VerifyTestEvidenceForCompletion but accepts pre-fetched comments.
// If comments is nil, comments will be fetched from beads API.
func VerifyTestEvidenceForCompletionWithComments(beadsID, workspacePath, projectDir string, comments []Comment) *TestEvidenceResult {
	result := VerifyTestEvidenceWithComments(beadsID, workspacePath, projectDir, comments)

	return filterTestEvidenceResult(&result)
}

// VerifyTestEvidenceForCompletionWithSkill is like VerifyTestEvidenceForCompletionWithComments
// but accepts a pre-extracted skill name, avoiding re-extraction from a potentially wrong path.
// This is used by checkTestEvidence when the skill has already been resolved via ResolveSkillName.
func VerifyTestEvidenceForCompletionWithSkill(beadsID, workspacePath, projectDir, skillName string, comments []Comment) *TestEvidenceResult {
	result := verifyTestEvidenceInternal(beadsID, workspacePath, projectDir, skillName, comments)

	return filterTestEvidenceResult(&result)
}

// filterTestEvidenceResult applies common filtering logic for completion convenience functions.
func filterTestEvidenceResult(result *TestEvidenceResult) *TestEvidenceResult {

	// Return nil if no code changes - no action needed
	if !result.HasCodeChanges {
		return nil
	}

	// Return nil if skill doesn't require test evidence
	if !IsSkillRequiringTestEvidence(result.SkillName) {
		return nil
	}

	// Return nil if exempted due to markdown-only changes
	if result.MarkdownOnlyExempt {
		return nil
	}

	// Return nil if exempted due to files outside project
	if result.OutsideProjectExempt {
		return nil
	}

	return result
}
