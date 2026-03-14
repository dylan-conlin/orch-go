package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/kbgate"
	"github.com/spf13/cobra"
)

var (
	kbGateJSON            bool
	kbGateModelJSON       bool
	kbAcknowledgeClaims   bool
	kbScanClaimsJSON      bool
)

var kbGateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Adversarial gates for the knowledge pipeline",
}

var kbGatePublishCmd = &cobra.Command{
	Use:   "publish <publication-path>",
	Short: "Check if a publication passes adversarial gate requirements",
	Long: `Run Phase 1 adversarial gate checks on a publication file.

Checks:
  1. Publication contract: challenge_refs, claim_refs, and ledger_ref must exist in frontmatter
  2. Challenge artifact: referenced challenge files must exist on disk
  3. Lineage: generalization/novel claims must have exogenous evidence
     (not just model/probe self-references)
  4. Banned language: novelty terms (physics, new framework, general law,
     substrate-independent, proves, validated theory) block publication
  5. Claim-upgrade signals: novelty language, self-validating probes,
     and causal language in model summaries
  6. Claim ledger: ledger_ref must point to a valid YAML file with structured
     claims (id, text, type, scope, evidence, strength)

Exit code 1 if any check fails.

Examples:
  orch kb gate publish docs/blog/my-post.md
  orch kb gate publish .kb/publications/knowledge-accretion.md --json
  orch kb gate publish .kb/publications/draft.md --acknowledge-claims`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := kbgate.CheckPublishOpts{
			AcknowledgeClaims: kbAcknowledgeClaims,
		}
		result := kbgate.CheckPublishWithOpts(args[0], opts)

		if kbGateJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(result); err != nil {
				return err
			}
		} else {
			fmt.Print(kbgate.FormatResult(result))

			// If claim-upgrade signals were found, show details for the target file
			if !kbAcknowledgeClaims {
				pubPath := args[0]
				scanResult := kbgate.ScanFile(pubPath)
				if scanResult.Total() > 0 {
					fmt.Println()
					fmt.Print(kbgate.FormatClaimScanResult(scanResult))
				}
			}
		}

		if !result.Pass {
			return fmt.Errorf("publication gate failed")
		}
		return nil
	},
}

var kbScanClaimsCmd = &cobra.Command{
	Use:   "scan-claims [kb-dir]",
	Short: "Scan knowledge base for claim-upgrade signals",
	Long: `Run claim-upgrade boundary detection on the knowledge base without
the full publish gate. Detects three signal types:

  1. Novelty language: novel, first (as novelty), new framework,
     substrate-independent, physics, discovered, absent from, new discipline
  2. Self-validating probes: confirms/extends in Model Impact sections
     without external citations
  3. Causal language: predict, cause, produce, determine, guarantee,
     ensure, always, never in model Summary sections

Examples:
  orch kb scan-claims
  orch kb scan-claims .kb/
  orch kb scan-claims --json`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		kbDir := ".kb"
		if len(args) > 0 {
			kbDir = args[0]
		}

		if _, err := os.Stat(kbDir); os.IsNotExist(err) {
			return fmt.Errorf("knowledge base directory not found: %s", kbDir)
		}

		result := kbgate.ScanAllClaims(kbDir)

		if kbScanClaimsJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		fmt.Print(kbgate.FormatClaimScanResult(result))

		if result.Total() > 0 {
			return fmt.Errorf("%d claim-upgrade signal(s) detected", result.Total())
		}
		return nil
	},
}

var kbGateModelCmd = &cobra.Command{
	Use:   "model <model-path>",
	Short: "Check if a model has required claim ledger and vocabulary canonicalization",
	Long: `Run Phase 2 adversarial gate checks on a model file.

Checks:
  1. Claim ledger table: model must contain a markdown table with columns
     claim_id, claim_text, claim_type, scope, novelty_level, evidence_refs
  2. Vocabulary canonicalization table: model must contain a markdown table with
     columns term, plain_language, nearest_existing_concepts, claimed_delta, verdict
  3. Claim entry validation: each claim must have valid type and novelty level
  4. Evidence warnings: non-observation claims need evidence_refs;
     generalization/novel claims with only model/probe refs get warned
  5. Vocabulary inflation: empty claimed_delta with prior-art mapped = warning

Exit code 1 if any required check fails. Warnings do not block.

Examples:
  orch kb gate model .kb/models/harness-engineering/model.md
  orch kb gate model .kb/models/knowledge-accretion/model.md --json`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		result := kbgate.CheckModel(args[0])

		if kbGateModelJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(result); err != nil {
				return err
			}
		} else {
			fmt.Print(kbgate.FormatModelResult(result))
		}

		if !result.Pass {
			return fmt.Errorf("model gate failed")
		}
		return nil
	},
}

func init() {
	kbGatePublishCmd.Flags().BoolVar(&kbGateJSON, "json", false, "Output as JSON")
	kbGatePublishCmd.Flags().BoolVar(&kbAcknowledgeClaims, "acknowledge-claims", false, "Acknowledge claim-upgrade signals (downgrades to warnings)")
	kbGateModelCmd.Flags().BoolVar(&kbGateModelJSON, "json", false, "Output as JSON")
	kbScanClaimsCmd.Flags().BoolVar(&kbScanClaimsJSON, "json", false, "Output as JSON")
	kbGateCmd.AddCommand(kbGatePublishCmd)
	kbGateCmd.AddCommand(kbGateModelCmd)
	kbCmd.AddCommand(kbGateCmd)
	kbCmd.AddCommand(kbScanClaimsCmd)
}
