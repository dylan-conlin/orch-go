package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/skill"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Skill management commands",
}

var skillLintCmd = &cobra.Command{
	Use:   "lint [SKILL.md or skill-name]",
	Short: "Static analysis for skill markdown",
	Long: `Run 5 static analysis rules against a skill markdown file:

  1. must-density        — MUST/NEVER/CRITICAL/ALWAYS per 100 words (>3 = warning)
  2. cosmetic-redundancy — Same constraint phrase >2 times (warning)
  3. section-sprawl      — Total constraints >30 (warning)
  4. signal-imbalance    — Same behavior reinforced >3 times (warning)
  5. dead-constraint     — Constraints with no test coverage (info)

Examples:
  orch skill lint orchestrator
  orch skill lint ~/.claude/skills/meta/orchestrator/SKILL.md
  orch skill lint feature-impl`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSkillLint(args[0])
	},
}

func init() {
	skillCmd.AddCommand(skillLintCmd)
	rootCmd.AddCommand(skillCmd)
}

func runSkillLint(target string) error {
	content, sourcePath, err := resolveSkillContent(target)
	if err != nil {
		return err
	}

	fmt.Printf("Linting: %s\n", sourcePath)
	fmt.Println(strings.Repeat("-", 60))

	results := skill.LintContent(content, nil)

	if len(results) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	warnings := 0
	infos := 0
	for _, r := range results {
		icon := "⚠️"
		if r.Severity == skill.SeverityInfo {
			icon = "ℹ️"
			infos++
		} else {
			warnings++
		}
		fmt.Printf("%s  [%s] %s\n", icon, r.Rule, r.Message)
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Summary: %d warning(s), %d info(s)\n", warnings, infos)

	return nil
}

// resolveSkillContent loads skill content from a file path or skill name.
func resolveSkillContent(target string) (content string, path string, err error) {
	// If target looks like a file path (contains / or ends in .md), read directly
	if strings.Contains(target, "/") || strings.HasSuffix(target, ".md") {
		expanded := target
		if strings.HasPrefix(expanded, "~") {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				return "", "", fmt.Errorf("cannot expand ~: %w", homeErr)
			}
			expanded = filepath.Join(home, expanded[1:])
		}
		data, readErr := os.ReadFile(expanded)
		if readErr != nil {
			return "", "", fmt.Errorf("cannot read %s: %w", expanded, readErr)
		}
		return string(data), expanded, nil
	}

	// Otherwise, resolve as a skill name via the loader
	loader := skills.DefaultLoader()
	skillPath, findErr := loader.FindSkillPath(target)
	if findErr != nil {
		return "", "", fmt.Errorf("skill %q not found: %w", target, findErr)
	}

	data, readErr := os.ReadFile(skillPath)
	if readErr != nil {
		return "", "", fmt.Errorf("cannot read %s: %w", skillPath, readErr)
	}

	return string(data), skillPath, nil
}
