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

	kbAuditModelsJSON bool

	kbAuditDecisionsJSON bool
	kbAuditDecisionsType string
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

var kbAuditModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Flag oversized models (>30KB) that need synthesis or pruning",
	Long: `Scan .kb/models/ and .kb/global/models/ for model.md files exceeding 30KB.

Models over 30KB that haven't had a consolidation pass (Last Updated) in 2+ weeks
are flagged for architect review. The gate triggers review, not automated pruning.

Examples:
  orch kb audit models          # Human-readable report
  orch kb audit models --json   # Machine-readable output`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKBAuditModels()
	},
}

func runKBAuditModels() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	kbDir := filepath.Join(projectDir, ".kb")
	reports, err := kbmetrics.AuditModelSize(kbDir, 30*1024, 14)
	if err != nil {
		return fmt.Errorf("audit models: %w", err)
	}

	if kbAuditModelsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(reports)
	}

	fmt.Print(kbmetrics.FormatModelSizeText(reports))
	return nil
}

var kbAuditDecisionsCmd = &cobra.Command{
	Use:   "decisions",
	Short: "Audit decisions for structural anchoring (gates, hooks, tests, file existence)",
	Long: `Audit .kb/decisions/ by splitting into architectural-principle vs implementation
decisions and applying type-appropriate validation.

Architectural decisions (principles, patterns, models): checked for reflection
in gates, hooks, tests, CLAUDE.md, and daemon/skill config.

Implementation decisions (specific code changes): checked for file existence
of referenced paths and frontmatter block patterns.

Each decision is scored: enforced (>= 50% checks pass), partial (some pass),
or unanchored (no checks pass).

Examples:
  orch kb audit decisions                        # Full report
  orch kb audit decisions --json                 # Machine-readable
  orch kb audit decisions --type architectural   # Only architectural
  orch kb audit decisions --type implementation  # Only implementation`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKBAuditDecisions()
	},
}

func runKBAuditDecisions() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	kbDir := filepath.Join(projectDir, ".kb")
	reports, err := kbmetrics.AuditDecisions(kbDir, projectDir)
	if err != nil {
		return fmt.Errorf("audit decisions: %w", err)
	}

	// Filter by type if requested
	if kbAuditDecisionsType != "" {
		dt := kbmetrics.DecisionType(kbAuditDecisionsType)
		var filtered []kbmetrics.DecisionReport
		for _, r := range reports {
			if r.Type == dt {
				filtered = append(filtered, r)
			}
		}
		reports = filtered
	}

	if kbAuditDecisionsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(reports)
	}

	fmt.Print(kbmetrics.FormatDecisionAuditText(reports))
	return nil
}

func init() {
	kbAuditProvenanceCmd.Flags().BoolVar(&kbAuditProvenanceJSON, "json", false, "Output as JSON")
	kbAuditProvenanceCmd.Flags().BoolVar(&kbAuditProvenanceVerbose, "verbose", false, "Show individual unannotated claims")
	kbAuditProvenanceCmd.Flags().StringVar(&kbAuditProvenanceModel, "model", "", "Audit a specific model by name")
	kbAuditModelsCmd.Flags().BoolVar(&kbAuditModelsJSON, "json", false, "Output as JSON")
	kbAuditDecisionsCmd.Flags().BoolVar(&kbAuditDecisionsJSON, "json", false, "Output as JSON")
	kbAuditDecisionsCmd.Flags().StringVar(&kbAuditDecisionsType, "type", "", "Filter by type: architectural or implementation")
	kbAuditCmd.AddCommand(kbAuditProvenanceCmd)
	kbAuditCmd.AddCommand(kbAuditModelsCmd)
	kbAuditCmd.AddCommand(kbAuditDecisionsCmd)
	kbCmd.AddCommand(kbAuditCmd)
}
