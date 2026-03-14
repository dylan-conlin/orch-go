package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/kbmetrics"
	"github.com/spf13/cobra"
)

var (
	kbAuditProvenanceJSON    bool
	kbAuditProvenanceVerbose bool
	kbAuditProvenanceModel   string
)

var kbAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit knowledge base for quality gaps",
}

var kbAuditProvenanceCmd = &cobra.Command{
	Use:   "provenance",
	Short: "Scan models for evidence quality gaps, orphan contradictions, and unannotated claims",
	Long: `Audit model evidence provenance across .kb/models/.

Scans all model.md files (or a specific model) for:
  1. Unannotated claims — claims without **Evidence quality:** annotation
  2. Low-confidence claims — claims annotated as single-source or assumed
  3. Orphan contradictions — probes with [x] **Contradicts** but model
     not updated since probe date
  4. Coverage metric — percentage of claims with evidence annotations

Examples:
  orch kb audit provenance                      # All models
  orch kb audit provenance --model orchestrator-skill  # Single model
  orch kb audit provenance --json               # Machine-readable
  orch kb audit provenance --verbose            # Show individual claims`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKBAuditProvenance()
	},
}

func runKBAuditProvenance() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	kbDir := filepath.Join(projectDir, ".kb")
	reports, err := kbmetrics.AuditProvenance(kbDir)
	if err != nil {
		return fmt.Errorf("audit provenance: %w", err)
	}

	// Filter to specific model if requested
	if kbAuditProvenanceModel != "" {
		var filtered []kbmetrics.ProvenanceReport
		for _, r := range reports {
			if r.Name == kbAuditProvenanceModel {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("model not found: %s", kbAuditProvenanceModel)
		}
		reports = filtered
	}

	if kbAuditProvenanceJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(reports)
	}

	output := kbmetrics.FormatProvenanceText(reports)
	fmt.Print(output)

	if kbAuditProvenanceVerbose {
		fmt.Println()
		for _, r := range reports {
			if len(r.UnannotatedClaims) == 0 {
				continue
			}
			fmt.Printf("--- %s: unannotated claims ---\n", r.Name)
			for _, uc := range r.UnannotatedClaims {
				text := uc.Text
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				fmt.Printf("  L%-4d %s\n", uc.Line, text)
			}
			fmt.Println()
		}
	}

	return nil
}

func init() {
	kbAuditProvenanceCmd.Flags().BoolVar(&kbAuditProvenanceJSON, "json", false, "Output as JSON")
	kbAuditProvenanceCmd.Flags().BoolVar(&kbAuditProvenanceVerbose, "verbose", false, "Show individual unannotated claims")
	kbAuditProvenanceCmd.Flags().StringVar(&kbAuditProvenanceModel, "model", "", "Audit a specific model by name")
	kbAuditCmd.AddCommand(kbAuditProvenanceCmd)
	kbCmd.AddCommand(kbAuditCmd)
}
