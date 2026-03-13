package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/events"
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

Hard block (exit 1):  >1500 lines (only when agent's changes caused the threshold crossing)
Warning (non-blocking): >1500 lines (pre-existing bloat — file was already over threshold)
Warning (non-blocking): >800 lines with ≥30 net lines added
Warning (non-blocking): >600 lines with ≥50 net lines added

Override: FORCE_ACCRETION=1 git commit ...`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("FORCE_ACCRETION") == "1" {
			fmt.Println("pre-commit: accretion gate bypassed (FORCE_ACCRETION=1)")
			logPrecommitGateDecision("accretion_precommit", "bypass", "FORCE_ACCRETION=1", nil)
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
			// Log gate block with target files
			var targetFiles []string
			for _, bf := range result.BlockedFiles {
				targetFiles = append(targetFiles, bf.Path)
			}
			logPrecommitGateDecision("accretion_precommit", "block", "file exceeds accretion threshold", targetFiles)
			fmt.Fprintln(os.Stderr, verify.FormatStagedAccretionError(result))
			os.Exit(1)
		}

		// Print warnings (non-blocking) for 800/600 thresholds
		if warnings := verify.FormatStagedAccretionWarnings(result); warnings != "" {
			fmt.Fprintln(os.Stderr, warnings)
		}

		logPrecommitGateDecision("accretion_precommit", "allow", "staged files within accretion threshold", nil)
		fmt.Println("pre-commit: accretion gate passed")
	},
}

var precommitModelStubCmd = &cobra.Command{
	Use:   "model-stub",
	Short: "Check staged model files for unfilled template placeholders",
	Long: `Checks staged .kb/models/*/model.md files for template placeholder text.

Models created with 'kb create model' without filling in the content will be blocked.
Detects bracket-enclosed placeholder patterns like [Concise claim statement].

Override: FORCE_MODEL_STUB=1 git commit ...`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("FORCE_MODEL_STUB") == "1" {
			fmt.Println("pre-commit: model-stub gate bypassed (FORCE_MODEL_STUB=1)")
			return
		}

		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "pre-commit: cannot get working directory: %v\n", err)
			os.Exit(1)
		}

		result := verify.CheckStagedModelStubs(dir)
		if result == nil {
			return
		}

		if !result.Passed {
			fmt.Fprintln(os.Stderr, verify.FormatStagedModelStubError(result))
			os.Exit(1)
		}

		fmt.Println("pre-commit: model-stub gate passed")
	},
}

var precommitDuplicationCmd = &cobra.Command{
	Use:   "duplication",
	Short: "Check staged files for function duplication (advisory)",
	Long: `Scans staged Go files for functions that are near-clones of existing functions.

This is advisory only — it warns but does not block the commit.
Functions with ≥85% structural similarity (AST fingerprinting) are reported.

Override: SKIP_DUPDETECT=1 git commit ...`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("SKIP_DUPDETECT") == "1" {
			fmt.Println("pre-commit: duplication check skipped (SKIP_DUPDETECT=1)")
			return
		}

		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "pre-commit: cannot get working directory: %v\n", err)
			return
		}

		result := verify.CheckStagedDuplication(dir)
		if result == nil {
			return
		}

		if warnings := verify.FormatStagedDuplicationWarning(result); warnings != "" {
			fmt.Fprintln(os.Stderr, warnings)
		} else {
			fmt.Println("pre-commit: duplication check passed")
		}
	},
}

func init() {
	precommitCmd.AddCommand(precommitAccretionCmd)
	precommitCmd.AddCommand(precommitModelStubCmd)
	precommitCmd.AddCommand(precommitDuplicationCmd)
	rootCmd.AddCommand(precommitCmd)
}

// logPrecommitGateDecision logs a spawn.gate_decision event for pre-commit gate evaluations.
func logPrecommitGateDecision(gateName, decision, reason string, targetFiles []string) {
	logger := events.NewLogger(events.DefaultLogPath())
	_ = logger.LogGateDecision(events.GateDecisionData{
		GateName:    gateName,
		Decision:    decision,
		TargetFiles: targetFiles,
		Reason:      reason,
	})
}
