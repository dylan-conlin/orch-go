// Package main provides the lint command for validating skill CLI references.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	lintSkills bool
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate skill files and CLAUDE.md configuration",
	Long: `Validate skill files and CLAUDE.md configuration.

With --skills, validates that skill files reference valid orch CLI commands
and flags. This catches stale references to commands that have been renamed,
removed, or never existed.

Examples:
  orch lint --skills    # Validate skill CLI references`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if lintSkills {
			return runLintSkills()
		}
		return cmd.Help()
	},
}

func init() {
	lintCmd.Flags().BoolVar(&lintSkills, "skills", false, "Validate CLI command references in skill files")
	rootCmd.AddCommand(lintCmd)
}

// commandInfo stores information about a valid CLI command.
type commandInfo struct {
	Name        string
	Flags       map[string]bool // flag name (without --) -> exists
	Subcommands map[string]bool // subcommand name -> exists
}

// collectCommands walks the cobra command tree and collects all valid commands.
func collectCommands(root *cobra.Command) map[string]*commandInfo {
	commands := make(map[string]*commandInfo)

	var walk func(cmd *cobra.Command, prefix string)
	walk = func(cmd *cobra.Command, prefix string) {
		name := prefix
		if cmd != root {
			if prefix == "" {
				name = cmd.Name()
			} else {
				name = prefix + " " + cmd.Name()
			}
		}

		if name != "" {
			info := &commandInfo{
				Name:        name,
				Flags:       make(map[string]bool),
				Subcommands: make(map[string]bool),
			}

			// Collect flags (both local and inherited)
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				info.Flags[f.Name] = true
			})
			// Also include persistent flags from parent
			cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
				info.Flags[f.Name] = true
			})

			commands[name] = info
		}

		// Process subcommands
		for _, sub := range cmd.Commands() {
			if sub.Hidden {
				continue
			}
			// Record subcommand in parent
			if name != "" {
				if info, ok := commands[name]; ok {
					info.Subcommands[sub.Name()] = true
				}
			}
			walk(sub, name)
		}
	}

	walk(root, "")
	return commands
}

// lintIssue represents a single lint issue found in a skill file.
type lintIssue struct {
	SkillName   string
	FilePath    string
	Description string
}

// runLintSkills validates CLI command references in skill files.
func runLintSkills() error {
	// Collect valid commands from our own CLI
	validCommands := collectCommands(rootCmd)

	// Find skill files
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	skillsDir := filepath.Join(homeDir, ".claude", "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		fmt.Println("✅ No skills directory found (~/.claude/skills/)")
		return nil
	}

	// Discover skill files (hierarchical structure: category/skill/SKILL.md)
	var skillFiles []string
	err = filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.Name() == "SKILL.md" && !info.IsDir() {
			skillFiles = append(skillFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan skills directory: %w", err)
	}

	if len(skillFiles) == 0 {
		fmt.Println("✅ No skill files found (0 skills scanned)")
		return nil
	}

	// Pattern to extract orch commands and flags from backtick-delimited references
	// Matches: `orch <command> [subcommand] [--flag] [--flag2]`
	orchPattern := regexp.MustCompile(
		"`orch\\s+([a-z][a-z0-9-]*(?:\\s+[a-z][a-z0-9-]*)?)" + // command + optional subcommand
			"((?:\\s+--[a-z][a-z0-9-]*)*)`", // optional flags, ends with backtick
	)
	flagPattern := regexp.MustCompile(`--([a-z][a-z0-9-]*)`)

	var issues []lintIssue
	totalCommands := 0
	validCount := 0

	for _, skillFile := range skillFiles {
		// Extract skill name from path (parent directory name)
		skillName := filepath.Base(filepath.Dir(skillFile))

		content, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		matches := orchPattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			totalCommands++
			cmdPart := strings.TrimSpace(strings.ToLower(match[1]))
			flagsPart := ""
			if len(match) > 2 {
				flagsPart = strings.TrimSpace(match[2])
			}

			// Check if command is valid
			cmdValid := false
			var matchedCmd string

			if _, ok := validCommands[cmdPart]; ok {
				cmdValid = true
				matchedCmd = cmdPart
			} else {
				// Try splitting into command + subcommand
				parts := strings.Fields(cmdPart)
				if len(parts) == 2 {
					fullCmd := parts[0] + " " + parts[1]
					if _, ok := validCommands[fullCmd]; ok {
						cmdValid = true
						matchedCmd = fullCmd
					} else if parentInfo, ok := validCommands[parts[0]]; ok {
						// Command exists but subcommand may be invalid
						cmdValid = true
						matchedCmd = parts[0]
						if len(parentInfo.Subcommands) > 0 {
							if !parentInfo.Subcommands[parts[1]] {
								validSubs := make([]string, 0, len(parentInfo.Subcommands))
								for s := range parentInfo.Subcommands {
									validSubs = append(validSubs, s)
								}
								sort.Strings(validSubs)
								issues = append(issues, lintIssue{
									SkillName:   skillName,
									FilePath:    skillFile,
									Description: fmt.Sprintf("Unknown subcommand: orch %s %s (valid: %s)", parts[0], parts[1], strings.Join(validSubs, ", ")),
								})
								continue
							}
						}
					}
				} else if len(parts) == 1 {
					if _, ok := validCommands[parts[0]]; ok {
						cmdValid = true
						matchedCmd = parts[0]
					}
				}
			}

			if !cmdValid {
				issues = append(issues, lintIssue{
					SkillName:   skillName,
					FilePath:    skillFile,
					Description: fmt.Sprintf("Unknown command: orch %s", cmdPart),
				})
				continue
			}

			// Check flags if command is valid
			if flagsPart != "" && matchedCmd != "" {
				flagMatches := flagPattern.FindAllStringSubmatch(flagsPart, -1)
				for _, fm := range flagMatches {
					flagName := fm[1]
					if cmdInfo, ok := validCommands[matchedCmd]; ok {
						if !cmdInfo.Flags[flagName] {
							issues = append(issues, lintIssue{
								SkillName:   skillName,
								FilePath:    skillFile,
								Description: fmt.Sprintf("Unknown flag: --%s on 'orch %s'", flagName, matchedCmd),
							})
							continue
						}
					}
				}
			}

			validCount++
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
		var skillOrder []string
		for _, issue := range issues {
			if _, seen := bySkill[issue.SkillName]; !seen {
				skillOrder = append(skillOrder, issue.SkillName)
			}
			bySkill[issue.SkillName] = append(bySkill[issue.SkillName], issue.Description)
		}

		sort.Strings(skillOrder)
		for _, skillName := range skillOrder {
			fmt.Printf("   📦 %s:\n", skillName)
			for _, desc := range bySkill[skillName] {
				fmt.Printf("      • %s\n", desc)
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("✅ All %d command references are valid!\n", validCount)
	}

	return nil
}
