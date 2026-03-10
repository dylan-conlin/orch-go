package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/kbgate"
	"github.com/spf13/cobra"
)

var kbGateJSON bool

var kbGateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Adversarial gates for the knowledge pipeline",
}

var kbGatePublishCmd = &cobra.Command{
	Use:   "publish <publication-path>",
	Short: "Check if a publication passes adversarial gate requirements",
	Long: `Run Phase 1 adversarial gate checks on a publication file.

Checks:
  1. Publication contract: challenge_refs and claim_refs must exist in frontmatter
  2. Challenge artifact: referenced challenge files must exist on disk
  3. Lineage: generalization/novel claims must have exogenous evidence
     (not just model/probe self-references)
  4. Banned language: novelty terms (physics, new framework, general law,
     substrate-independent, proves, validated theory) block publication

Exit code 1 if any check fails.

Examples:
  orch kb gate publish docs/blog/my-post.md
  orch kb gate publish .kb/publications/knowledge-physics.md --json`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		result := kbgate.CheckPublish(args[0])

		if kbGateJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(result); err != nil {
				return err
			}
		} else {
			fmt.Print(kbgate.FormatResult(result))
		}

		if !result.Pass {
			return fmt.Errorf("publication gate failed")
		}
		return nil
	},
}

func init() {
	kbGatePublishCmd.Flags().BoolVar(&kbGateJSON, "json", false, "Output as JSON")
	kbGateCmd.AddCommand(kbGatePublishCmd)
	kbCmd.AddCommand(kbGateCmd)
}
