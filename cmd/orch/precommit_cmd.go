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
	Short: "Check staged files for accretion violations",
	Long: `Checks all staged source files against accretion thresholds.

Hard block (exit 1):  >1500 lines
Warning (non-blocking): >800 lines with ≥30 net lines added
Warning (non-blocking): >600 lines with ≥50 net lines added

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

		// Print warnings (non-blocking) for 800/600 thresholds
		if warnings := verify.FormatStagedAccretionWarnings(result); warnings != "" {
			fmt.Fprintln(os.Stderr, warnings)
		}

		fmt.Println("pre-commit: accretion gate passed")
	},
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
