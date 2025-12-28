// lint.go - Check CLAUDE.md files against token and character limits
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/claudemd"
	"github.com/spf13/cobra"
)

var (
	// Lint command flags
	lintFile      string
	lintCheckAll  bool
	lintSkills    bool
	lintIssues    bool
	lintVerbose   bool
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Check CLAUDE.md files against token and character limits",
	Long: `Check CLAUDE.md files against recommended limits.

Validates that CLAUDE.md files stay within recommended limits:
- Global (~/.claude/CLAUDE.md): 5,000 tokens, 20,000 chars
- Project (project/CLAUDE.md): 15,000 tokens, 60,000 chars

Both limits must be satisfied for a file to pass.

Examples:
  orch lint                             # Check CLAUDE.md in current project
  orch lint --file ~/.claude/CLAUDE.md  # Check specific file
  orch lint --all                       # Check all known CLAUDE.md files
  orch lint --verbose                   # Show detailed metrics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLint()
	},
}

func init() {
	lintCmd.Flags().StringVar(&lintFile, "file", "", "Specific CLAUDE.md file to check")
	lintCmd.Flags().BoolVar(&lintCheckAll, "all", false, "Check all known CLAUDE.md files")
	lintCmd.Flags().BoolVar(&lintSkills, "skills", false, "Validate CLI command references in skill files")
	lintCmd.Flags().BoolVar(&lintIssues, "issues", false, "Validate beads issues for common problems")
	lintCmd.Flags().BoolVar(&lintVerbose, "verbose", false, "Show detailed metrics")

	rootCmd.AddCommand(lintCmd)
}

func runLint() error {
	// Handle --skills mode
	if lintSkills {
		return lintSkillFiles()
	}

	// Handle --issues mode
	if lintIssues {
		return lintBeadsIssues()
	}

	// Collect files to check
	var files []string

	if lintFile != "" {
		// Expand ~ in path
		if strings.HasPrefix(lintFile, "~") {
			home, _ := os.UserHomeDir()
			lintFile = filepath.Join(home, lintFile[1:])
		}
		files = append(files, lintFile)
	} else if lintCheckAll {
		// Get all known CLAUDE.md files
		files = getAllClaudeMDFiles()
	} else {
		// Check project CLAUDE.md in current directory
		cwd, _ := os.Getwd()
		projectFile := filepath.Join(cwd, "CLAUDE.md")
		if _, err := os.Stat(projectFile); err == nil {
			files = append(files, projectFile)
		} else {
			fmt.Println("No CLAUDE.md found in current directory")
			return nil
		}
	}

	if len(files) == 0 {
		fmt.Println("No CLAUDE.md files found to check")
		return nil
	}

	// Check each file
	allPassed := true
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("⚠️  File not found: %s\n", file)
			continue
		}

		passed := checkClaudeMDFile(file, lintVerbose)
		if !passed {
			allPassed = false
		}
	}

	if allPassed {
		return nil
	}
	return fmt.Errorf("some files exceeded limits")
}

func getAllClaudeMDFiles() []string {
	var files []string
	home, _ := os.UserHomeDir()

	// Global CLAUDE.md
	globalFile := filepath.Join(home, ".claude", "CLAUDE.md")
	if _, err := os.Stat(globalFile); err == nil {
		files = append(files, globalFile)
	}

	// Project CLAUDE.md in current directory
	cwd, _ := os.Getwd()
	projectFile := filepath.Join(cwd, "CLAUDE.md")
	if _, err := os.Stat(projectFile); err == nil {
		files = append(files, projectFile)
	}

	// Check for orchestrator skill (special case)
	orchestratorSkill := filepath.Join(home, ".claude", "skills", "meta", "orchestrator", "SKILL.md")
	if _, err := os.Stat(orchestratorSkill); err == nil {
		files = append(files, orchestratorSkill)
	}

	return files
}

func checkClaudeMDFile(filePath string, verbose bool) bool {
	// Classify file type
	fileType := classifyClaudeMDFile(filePath)

	// Get limits for this file type
	limits := claudemd.GetLimits(fileType)

	// Count metrics
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("❌ Error reading %s: %v\n", filePath, err)
		return false
	}

	charCount := len(content)
	tokenCount := claudemd.CountTokens(string(content))

	// Check against limits
	charExceeded := charCount > limits.CharLimit
	tokenExceeded := tokenCount > limits.TokenLimit

	// Determine status
	var status string
	if charExceeded || tokenExceeded {
		status = "❌ FAIL"
	} else if charCount > int(float64(limits.CharLimit)*0.9) || tokenCount > int(float64(limits.TokenLimit)*0.9) {
		status = "⚠️  WARN"
	} else {
		status = "✅ PASS"
	}

	// Print result
	shortPath := shortenPath(filePath)
	fmt.Printf("%s %s (%s)\n", status, shortPath, fileType)

	if verbose || charExceeded || tokenExceeded {
		fmt.Printf("   Chars:  %d / %d (%d%%)\n", charCount, limits.CharLimit, (charCount*100)/limits.CharLimit)
		fmt.Printf("   Tokens: %d / %d (%d%%)\n", tokenCount, limits.TokenLimit, (tokenCount*100)/limits.TokenLimit)
	}

	return !charExceeded && !tokenExceeded
}

func classifyClaudeMDFile(filePath string) string {
	home, _ := os.UserHomeDir()

	// Check if it's the global CLAUDE.md
	globalPath := filepath.Join(home, ".claude", "CLAUDE.md")
	if filePath == globalPath {
		return "global"
	}

	// Check if it's the orchestrator skill
	if strings.Contains(filePath, "orchestrator") && strings.HasSuffix(filePath, "SKILL.md") {
		return "orchestrator"
	}

	// Default to project
	return "project"
}

func shortenPath(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// lintSkillFiles validates CLI command references in skill files.
func lintSkillFiles() error {
	home, _ := os.UserHomeDir()
	skillsDir := filepath.Join(home, ".claude", "skills")

	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		fmt.Println("✅ No skills directory found (~/.claude/skills/)")
		return nil
	}

	// Discover skill files
	var skillFiles []string
	err := filepath.WalkDir(skillsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "SKILL.md" {
			skillFiles = append(skillFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan skills: %w", err)
	}

	if len(skillFiles) == 0 {
		fmt.Println("✅ No skill files found (0 skills scanned)")
		return nil
	}

	// Pattern to extract orch commands (must be in backticks)
	orchPattern := regexp.MustCompile("`orch\\s+([a-z][a-z0-9-]*(?:\\s+[a-z][a-z0-9-]*)?)(?:\\s+--[a-z][a-z0-9-]*)*`")

	// Track issues
	type lintIssue struct {
		skillName string
		filePath  string
		issue     string
	}
	var issues []lintIssue
	totalCommands := 0
	validCount := 0

	// Known valid commands (simplified check)
	validCommands := map[string]bool{
		"spawn": true, "status": true, "complete": true, "send": true,
		"monitor": true, "wait": true, "tail": true, "question": true,
		"abandon": true, "clean": true, "work": true,
		"daemon": true, "daemon run": true, "daemon preview": true,
		"account": true, "account list": true, "account switch": true,
		"focus": true, "drift": true, "next": true, "review": true,
		"lint": true, "synthesis": true, "stale": true, "logs": true,
		"history": true, "transcript": true, "transcript format": true,
		"version": true, "port": true, "init": true,
		"servers": true, "servers list": true, "servers start": true,
		"servers stop": true, "servers attach": true, "servers open": true,
		"kb": true, "kb context": true, "kb create": true, "kb search": true,
		"learn": true, "patterns": true, "retries": true,
	}

	for _, skillFile := range skillFiles {
		skillName := filepath.Base(filepath.Dir(skillFile))

		content, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		matches := orchPattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			totalCommands++
			cmdPart := strings.TrimSpace(strings.ToLower(match[1]))

			// Check if command is valid
			cmdValid := validCommands[cmdPart]
			if !cmdValid {
				// Try splitting to check parent command
				parts := strings.SplitN(cmdPart, " ", 2)
				if len(parts) == 2 {
					cmdValid = validCommands[parts[0]]
				} else {
					cmdValid = validCommands[parts[0]]
				}
			}

			if !cmdValid {
				issues = append(issues, lintIssue{
					skillName: skillName,
					filePath:  skillFile,
					issue:     fmt.Sprintf("Unknown command: orch %s", cmdPart),
				})
			} else {
				validCount++
			}
		}
	}

	// Report results
	fmt.Printf("🔍 Skill CLI reference check:\n")
	fmt.Printf("   Scanned %d skill files\n", len(skillFiles))
	fmt.Printf("   Found %d orch command references\n", totalCommands)
	fmt.Println()

	if len(issues) > 0 {
		fmt.Printf("⚠️  Found %d issues:\n", len(issues))
		fmt.Println()

		// Group by skill
		bySkill := make(map[string][]string)
		for _, issue := range issues {
			bySkill[issue.skillName] = append(bySkill[issue.skillName], issue.issue)
		}

		for skillName, skillIssues := range bySkill {
			fmt.Printf("   📦 %s:\n", skillName)
			for _, issue := range skillIssues {
				fmt.Printf("      • %s\n", issue)
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("✅ All %d command references are valid!\n", validCount)
	}

	return nil
}

// lintBeadsIssues validates beads issues for common problems.
func lintBeadsIssues() error {
	// Get all open issues via CLI fallback
	issues, err := beads.FallbackList("")
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Filter to open issues
	var openIssues []beads.Issue
	for _, issue := range issues {
		if issue.Status != "closed" {
			openIssues = append(openIssues, issue)
		}
	}

	if len(openIssues) == 0 {
		fmt.Println("✅ No open issues to validate")
		return nil
	}

	fmt.Printf("🔍 Validating %d open beads issues...\n\n", len(openIssues))

	type issueWarning struct {
		issueID    string
		title      string
		warning    string
		detail     string
		suggestion string
	}
	var warnings []issueWarning
	passedCount := 0

	// Patterns for checking
	deletionKeywords := []string{"delete", "remove", "eliminate", "deprecate"}
	blockerPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)BLOCKED\s*:`),
		regexp.MustCompile(`(?i)Prerequisite\s*:`),
		regexp.MustCompile(`(?i)Depends\s+on\s*:`),
		regexp.MustCompile(`(?i)Requires\s*:`),
		regexp.MustCompile(`(?i)Must\s+wait\s+for`),
	}
	vaguePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)all\s+\w+\s+references`),
		regexp.MustCompile(`(?i)remove\s+all\s+`),
		regexp.MustCompile(`(?i)delete\s+all\s+`),
		regexp.MustCompile(`(?i)update\s+every\s+`),
		regexp.MustCompile(`(?i)across\s+the\s+codebase`),
		regexp.MustCompile(`(?i)throughout\s+the\s+project`),
	}

	for _, issue := range openIssues {
		titleLower := strings.ToLower(issue.Title)
		description := issue.Description

		var issueWarnings []issueWarning

		// Check 1: Deletion issues without migration path
		isDeletion := false
		for _, kw := range deletionKeywords {
			if strings.Contains(titleLower, kw) {
				isDeletion = true
				break
			}
		}
		if isDeletion {
			hasMigration := strings.Contains(strings.ToLower(description), "## migration") ||
				strings.Contains(strings.ToLower(description), "migration path") ||
				strings.Contains(strings.ToLower(description), "migration plan") ||
				strings.Contains(strings.ToLower(description), "migrate to")
			if !hasMigration {
				issueWarnings = append(issueWarnings, issueWarning{
					issueID:    issue.ID,
					title:      issue.Title,
					warning:    "Deletion issue without migration path",
					detail:     "Contains deletion keyword but no '## Migration' section",
					suggestion: "Add a ## Migration section explaining what replaces the deleted code",
				})
			}
		}

		// Check 2: Hidden blockers in description
		combinedText := description
		hasBlockerText := false
		for _, pattern := range blockerPatterns {
			if pattern.MatchString(combinedText) {
				hasBlockerText = true
				break
			}
		}
		if hasBlockerText && len(issue.Dependencies) == 0 {
			issueWarnings = append(issueWarnings, issueWarning{
				issueID:    issue.ID,
				title:      issue.Title,
				warning:    "Hidden blocker in description",
				detail:     "Found blocker/prerequisite text but no bd dependency tracked",
				suggestion: "Run 'bd dep <blocker-id> <this-id>' to track the blocker properly",
			})
		}

		// Check 3: Vague scope without enumeration
		isVague := false
		for _, pattern := range vaguePatterns {
			if pattern.MatchString(description) {
				isVague = true
				break
			}
		}
		if isVague {
			hasEnumeration := strings.Contains(description, "1.") ||
				strings.Contains(description, "- ") ||
				strings.Contains(description, "* ") ||
				strings.Contains(description, "files:") ||
				strings.Contains(description, "locations:")
			if !hasEnumeration {
				issueWarnings = append(issueWarnings, issueWarning{
					issueID:    issue.ID,
					title:      issue.Title,
					warning:    "Vague scope without enumeration",
					detail:     "Contains 'all X' or 'remove Y references' but no specific list",
					suggestion: "Add enumeration: list specific files, count occurrences, or scope precisely",
				})
			}
		}

		// Check 4: Stale issues (open >7 days without activity)
		if issue.UpdatedAt != "" {
			t, err := time.Parse(time.RFC3339, issue.UpdatedAt)
			if err == nil {
				ageDays := int(time.Since(t).Hours() / 24)
				if ageDays > 7 {
					issueWarnings = append(issueWarnings, issueWarning{
						issueID:    issue.ID,
						title:      issue.Title,
						warning:    fmt.Sprintf("Stale issue (no activity for %d days)", ageDays),
						detail:     fmt.Sprintf("Last updated %d days ago", ageDays),
						suggestion: "Update status, add comment, or close if no longer needed",
					})
				}
			}
		}

		// Check 5: Epic missing success criteria
		if strings.EqualFold(issue.IssueType, "epic") {
			acceptancePatterns := []string{
				"done when", "success criteria", "acceptance criteria",
				"[ ]", "complete when", "definition of done",
			}
			hasAcceptance := false
			descLower := strings.ToLower(description)
			for _, pattern := range acceptancePatterns {
				if strings.Contains(descLower, pattern) {
					hasAcceptance = true
					break
				}
			}
			if !hasAcceptance {
				issueWarnings = append(issueWarnings, issueWarning{
					issueID:    issue.ID,
					title:      issue.Title,
					warning:    "Epic missing success criteria",
					detail:     "Epic issues should have clear '## Success Criteria' section",
					suggestion: "Add success criteria or checklist to define 'done'",
				})
			}
		}

		if len(issueWarnings) > 0 {
			warnings = append(warnings, issueWarnings...)
		} else {
			passedCount++
		}
	}

	// Display results
	if len(warnings) > 0 {
		fmt.Printf("⚠️  Found %d issue(s) with problems:\n\n", len(warnings))

		// Group warnings by issue
		byIssue := make(map[string][]issueWarning)
		for _, w := range warnings {
			byIssue[w.issueID] = append(byIssue[w.issueID], w)
		}

		// Sort issue IDs for consistent output
		var issueIDs []string
		for id := range byIssue {
			issueIDs = append(issueIDs, id)
		}
		sort.Strings(issueIDs)

		for _, issueID := range issueIDs {
			issueWarnings := byIssue[issueID]
			title := issueWarnings[0].title
			displayTitle := title
			if len(displayTitle) > 50 {
				displayTitle = displayTitle[:50] + "..."
			}
			fmt.Printf("⚠️  %s: %s\n", issueID, displayTitle)

			for _, w := range issueWarnings {
				fmt.Printf("   • %s\n", w.warning)
				fmt.Printf("     %s\n", w.detail)
				fmt.Printf("     💡 %s\n", w.suggestion)
			}
			fmt.Println()
		}

		fmt.Printf("✅ %d issue(s) passed validation\n", passedCount)
	} else {
		fmt.Printf("✅ All %d issue(s) passed validation\n", len(openIssues))
	}

	return nil
}
