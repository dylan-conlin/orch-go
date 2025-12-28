// lint.go - Check CLAUDE.md files against token and character limits
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	lintCmd.Flags().BoolVar(&lintSkills, "skills", false, "Validate CLI command references in skill files (not implemented)")
	lintCmd.Flags().BoolVar(&lintIssues, "issues", false, "Validate beads issues for common problems (not implemented)")
	lintCmd.Flags().BoolVar(&lintVerbose, "verbose", false, "Show detailed metrics")

	rootCmd.AddCommand(lintCmd)
}

func runLint() error {
	// Handle --skills mode (not implemented yet)
	if lintSkills {
		fmt.Println("Skill validation not yet implemented in Go. Use: orch-py lint --skills")
		return nil
	}

	// Handle --issues mode (not implemented yet)
	if lintIssues {
		fmt.Println("Issue validation not yet implemented in Go. Use: orch-py lint --issues")
		return nil
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
