// Package main provides commands for managing CLI documentation debt.
package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Manage CLI documentation debt tracking",
	Long: `Manage CLI documentation debt tracking.

Doc debt is tracked automatically when new CLI commands are detected during
'orch complete'. Use these commands to view and manage the debt.

Commands:
  list                List all tracked commands and their documentation status
  mark <command-file> Mark a command as documented
  unmark <command-file> Mark a command as undocumented (revert a mark)

Examples:
  orch docs list                # Show all tracked commands
  orch docs mark reconcile.go   # Mark reconcile command as documented
  orch docs unmark reconcile.go # Revert documentation mark`,
}

var docsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracked commands and their documentation status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDocsList()
	},
}

var docsMarkCmd = &cobra.Command{
	Use:   "mark <command-file>",
	Short: "Mark a command as documented",
	Long: `Mark a CLI command file as documented.

The command file should match the file name used during tracking (e.g., "reconcile.go").
Use 'orch docs list' to see available command files.

Examples:
  orch docs mark reconcile.go
  orch docs mark focus.go`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDocsMark(args[0])
	},
}

var docsUnmarkCmd = &cobra.Command{
	Use:   "unmark <command-file>",
	Short: "Mark a command as undocumented (revert a mark)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDocsUnmark(args[0])
	},
}

func init() {
	docsCmd.AddCommand(docsListCmd)
	docsCmd.AddCommand(docsMarkCmd)
	docsCmd.AddCommand(docsUnmarkCmd)
	rootCmd.AddCommand(docsCmd)
}

func runDocsList() error {
	debt, err := userconfig.LoadDocDebt()
	if err != nil {
		return fmt.Errorf("failed to load doc debt: %w", err)
	}

	if len(debt.Commands) == 0 {
		fmt.Println("No CLI commands tracked yet.")
		fmt.Println("Doc debt tracking starts automatically when new commands are detected during 'orch complete'.")
		return nil
	}

	// Collect and sort commands
	var entries []userconfig.DocDebtEntry
	for _, entry := range debt.Commands {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		// Sort undocumented first, then by date added
		if entries[i].Documented != entries[j].Documented {
			return !entries[i].Documented // undocumented first
		}
		return entries[i].DateAdded < entries[j].DateAdded
	})

	// Count stats
	undocumented := 0
	for _, e := range entries {
		if !e.Documented {
			undocumented++
		}
	}
	documented := len(entries) - undocumented

	fmt.Printf("CLI Documentation Debt (%d tracked, %d documented, %d undocumented)\n", len(entries), documented, undocumented)
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	// Print undocumented first
	if undocumented > 0 {
		fmt.Println("❌ UNDOCUMENTED:")
		for _, entry := range entries {
			if !entry.Documented {
				fmt.Printf("   • %s (added %s)\n", entry.CommandFile, entry.DateAdded)
			}
		}
		fmt.Println()
	}

	// Print documented
	if documented > 0 {
		fmt.Println("✓ DOCUMENTED:")
		for _, entry := range entries {
			if entry.Documented {
				fmt.Printf("   • %s (documented %s)\n", entry.CommandFile, entry.DateDocumented)
			}
		}
		fmt.Println()
	}

	if undocumented > 0 {
		fmt.Println("To mark a command as documented:")
		fmt.Println("  orch docs mark <command-file>")
	}

	return nil
}

func runDocsMark(commandFile string) error {
	debt, err := userconfig.LoadDocDebt()
	if err != nil {
		return fmt.Errorf("failed to load doc debt: %w", err)
	}

	if !debt.MarkDocumented(commandFile) {
		// Check if it exists but is already documented
		if entry, exists := debt.Commands[commandFile]; exists && entry.Documented {
			fmt.Printf("Command '%s' is already marked as documented.\n", commandFile)
			return nil
		}
		return fmt.Errorf("command '%s' not found in doc debt tracker. Use 'orch docs list' to see available commands", commandFile)
	}

	if err := userconfig.SaveDocDebt(debt); err != nil {
		return fmt.Errorf("failed to save doc debt: %w", err)
	}

	fmt.Printf("✓ Marked '%s' as documented.\n", commandFile)
	fmt.Println()
	fmt.Println("Remember to update:")
	fmt.Println("  - ~/.claude/skills/meta/orchestrator/SKILL.md")
	fmt.Println("  - docs/orch-commands-reference.md")

	return nil
}

func runDocsUnmark(commandFile string) error {
	debt, err := userconfig.LoadDocDebt()
	if err != nil {
		return fmt.Errorf("failed to load doc debt: %w", err)
	}

	entry, exists := debt.Commands[commandFile]
	if !exists {
		return fmt.Errorf("command '%s' not found in doc debt tracker", commandFile)
	}

	if !entry.Documented {
		fmt.Printf("Command '%s' is already marked as undocumented.\n", commandFile)
		return nil
	}

	// Revert the documented status
	entry.Documented = false
	entry.DateDocumented = ""
	debt.Commands[commandFile] = entry

	if err := userconfig.SaveDocDebt(debt); err != nil {
		return fmt.Errorf("failed to save doc debt: %w", err)
	}

	fmt.Printf("✓ Marked '%s' as undocumented.\n", commandFile)

	return nil
}
