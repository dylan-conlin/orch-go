package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var precommitCmd = &cobra.Command{
	Use:   "precommit",
	Short: "Pre-commit gate subcommands",
}

var precommitAccretionCmd = &cobra.Command{
	Use:   "accretion",
	Short: "Check staged files for accretion violations (>1500 lines)",
	Long: `Checks all staged source files against the CRITICAL threshold (1500 lines).
Blocks the commit if any file exceeds the threshold.

Override: FORCE_ACCRETION=1 git commit ...`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("FORCE_ACCRETION") == "1" {
			fmt.Println("pre-commit: accretion gate bypassed (FORCE_ACCRETION=1)")
			return
		}

		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "pre-commit: cannot get working directory: %v\n", err)
			os.Exit(1)
		}

		result := verify.CheckStagedAccretion(dir)
		if result == nil {
			return
		}

		if !result.Passed {
			fmt.Fprintln(os.Stderr, verify.FormatStagedAccretionError(result))
			os.Exit(1)
		}

		fmt.Printf("pre-commit: accretion gate passed (%d staged source files checked)\n", countCheckedFiles(result))
	},
}

func countCheckedFiles(result *verify.StagedAccretionResult) int {
	// BlockedFiles only has failures; we don't track pass count in the struct.
	// Just report 0 blocked as the success signal.
	return 0
}

var precommitKnowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Check staged investigations for model coupling",
	Long: `Checks new .kb/investigations/ files for model coupling.
New investigation files must contain either:
  **Model:** <name>    (linked to a .kb/models/ entry)
  **Orphan:** acknowledged   (explicit opt-out)

Or have a probe file also staged in .kb/models/*/probes/.

Override: FORCE_ORPHAN=1 git commit ...`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("FORCE_ORPHAN") == "1" {
			fmt.Println("pre-commit: knowledge gate bypassed (FORCE_ORPHAN=1)")
			return
		}

		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "pre-commit: cannot get working directory: %v\n", err)
			os.Exit(1)
		}

		result := verify.CheckStagedKnowledge(dir)
		if result == nil {
			return
		}

		if !result.Passed {
			fmt.Fprintln(os.Stderr, verify.FormatStagedKnowledgeError(result))
			os.Exit(1)
		}

		fmt.Println("pre-commit: knowledge gate passed")
	},
}

func init() {
	precommitCmd.AddCommand(precommitAccretionCmd)
	precommitCmd.AddCommand(precommitKnowledgeCmd)
	rootCmd.AddCommand(precommitCmd)
}
