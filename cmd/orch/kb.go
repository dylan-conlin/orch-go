// Package main provides kb subcommands for knowledge base management.
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
	kbAskSave   bool   // Save result as investigation artifact
	kbAskModel  string // Model to use for synthesis
	kbAskLimit  int    // Maximum artifacts to read
	kbAskGlobal bool   // Search across all projects

	// kb extract flags
	kbExtractTo           string // Target project name
	kbExtractUpdateSource bool   // Add extracted-to reference in original

	// kb orphans flags
	kbOrphansJSON       bool
	kbOrphansStratified bool

	// kb claims flags
	kbClaimsJSON    bool
	kbClaimsVerbose bool
)

var kbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create knowledge base artifacts",
}

var kbCreateModelCmd = &cobra.Command{
	Use:   "model <name>",
	Short: "Create a new model with directory structure and template",
	Long: `Create a new model in .kb/models/ with proper directory structure.

Creates:
  .kb/models/<name>/model.md    (from TEMPLATE.md)
  .kb/models/<name>/probes/     (empty directory for future probes)

Model names must be lowercase kebab-case (e.g., "spawn-architecture").

Examples:
  orch kb create model agent-lifecycle
  orch kb create model dashboard-architecture`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, err := os.Getwd()
		if err != nil {
			return err
		}
		return runKBCreateModel(args[0], projectDir)
	},
}

var kbCmd = &cobra.Command{
	Use:   "kb",
	Short: "Knowledge base commands for inline queries and artifact management",
	Long: `Knowledge base commands for quick inline queries and artifact management.

The kb subcommand provides fast access to knowledge synthesis without
the overhead of spawning full investigation agents.

Examples:
  orch kb ask "how should we sort the swarm map?"
  orch kb ask "what's our auth pattern?" --save
  orch kb ask "rate limiting approach" --global
  orch kb extract .kb/decisions/2025-01-01-auth-pattern.md --to skillc`,
}

var kbExtractCmd = &cobra.Command{
	Use:   "extract <artifact-path>",
	Short: "Extract artifact to another project with lineage tracking",
	Long: `Extract a knowledge artifact to another project with lineage metadata.

This command copies an artifact (investigation, decision, etc.) to another
project's .kb/ directory while preserving lineage information. The copy
includes an 'extracted-from' header, and optionally updates the source
with an 'extracted-to' reference.

The artifact is COPIED, not moved - the original remains for historical reference.

Examples:
  # Extract a decision to skillc project
  orch kb extract .kb/decisions/2025-01-01-skill-template.md --to skillc

  # Extract and update source with back-reference
  orch kb extract .kb/investigations/2025-01-01-auth-flow.md --to auth-service --update-source

  # Use absolute path
  orch kb extract /path/to/project/.kb/decisions/foo.md --to other-project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if kbExtractTo == "" {
			return fmt.Errorf("--to flag is required: specify target project name")
		}
		return runKBExtract(args[0], kbExtractTo, kbExtractUpdateSource)
	},
}

var kbAskCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Get inline answers from knowledge base (~5-10s)",
	Long: `Get quick inline answers by synthesizing knowledge base context.

This command:
1. Runs kb context with your question keywords
2. Reads top matching artifacts (investigations, decisions, kn entries)
3. Sends to LLM with synthesis prompt
4. Returns answer inline (~5-10 seconds)

Use this for quick questions. For questions worth preserving as artifacts,
use --save or spawn a full investigation.

Examples:
  orch kb ask "how should we handle rate limiting?"
  orch kb ask "what's our auth pattern?"
  orch kb ask "spawning best practices" --save  # Save as investigation
  orch kb ask "config patterns" --global         # Search all projects
  orch kb ask "db migrations" --limit 5          # Limit artifacts read`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		question := args[0]
		return runKBAsk(question)
	},
}

var kbClaimsCmd = &cobra.Command{
	Use:   "claims",
	Short: "Analyze claims per model — knowledge equivalent of lines-per-file",
	Long: `Extract and count claims from .kb/models/*/model.md files.

Claims are the knowledge equivalent of lines of code. Models with too many
claims become unfocused and hard to probe, similar to bloated source files.

Thresholds:
  healthy:  < 30 claims
  warning:  30-49 claims (may need splitting)
  critical: >= 50 claims (needs splitting)

Claim types extracted:
  core:       Core claim section assertions
  invariant:  Numbered items (Critical Invariants)
  assertion:  Bold-prefixed bullet points
  data:       Table data rows
  constraint: Constraint/Implication pairs
  failure:    Failure mode root causes

Examples:
  orch kb claims                    # Human-readable report
  orch kb claims --json             # Machine-readable output
  orch kb claims --verbose          # Show individual claims`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKBClaims()
	},
}

var kbOrphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Show investigation orphan rate — percentage unconnected to models/decisions/guides",
	Long: `Compute the orphan rate for .kb/investigations/ files.

An investigation is "orphaned" if no other .kb/ file (model, decision, guide,
probe, or other investigation) references it. High orphan rates signal
under-synthesis — investigations producing findings that never get integrated.

The orphan rate was first measured at 85.5% during the knowledge-accretion probe
(Mar 2026). The model-era rate (after probe system existed) was 52.0%.

Examples:
  orch kb orphans          # Human-readable report
  orch kb orphans --json   # Machine-readable output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKBOrphans()
	},
}

func runKBOrphans() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	kbDir := filepath.Join(projectDir, ".kb")

	if kbOrphansStratified {
		return runKBOrphansStratified(kbDir)
	}

	report, err := kbmetrics.ComputeOrphanRate(kbDir)
	if err != nil {
		return fmt.Errorf("compute orphan rate: %w", err)
	}

	if kbOrphansJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	if report.Total == 0 {
		fmt.Println("No investigations found in .kb/investigations/")
		return nil
	}

	fmt.Printf("Investigation Orphan Rate\n")
	fmt.Printf("=========================\n\n")
	fmt.Printf("Total investigations:  %d\n", report.Total)
	fmt.Printf("Connected:             %d\n", report.Connected)
	fmt.Printf("Orphaned:              %d\n", report.Orphaned)
	fmt.Printf("Orphan rate:           %.1f%%\n", report.OrphanRate)

	return nil
}

func runKBOrphansStratified(kbDir string) error {
	report, err := kbmetrics.ComputeStratifiedOrphanRate(kbDir)
	if err != nil {
		return fmt.Errorf("compute stratified orphan rate: %w", err)
	}

	if kbOrphansJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	if report.Total == 0 {
		fmt.Println("No investigations found in .kb/investigations/")
		return nil
	}

	fmt.Print(report.StratifiedSummary())
	return nil
}

func runKBClaims() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	modelsDir := filepath.Join(projectDir, ".kb", "models")
	results, err := kbmetrics.AnalyzeModels(modelsDir)
	if err != nil {
		return fmt.Errorf("analyze models: %w", err)
	}

	if kbClaimsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	fmt.Print(kbmetrics.FormatText(results))

	if kbClaimsVerbose {
		fmt.Println()
		for _, r := range results {
			if r.ClaimCount == 0 {
				continue
			}
			fmt.Printf("--- %s (%d claims) ---\n", r.Name, r.ClaimCount)
			for _, c := range r.Claims {
				text := c.Text
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				fmt.Printf("  L%-4d [%-10s] %s\n", c.Line, c.Type, text)
			}
			fmt.Println()
		}
	}

	return nil
}

func init() {
	kbAskCmd.Flags().BoolVar(&kbAskSave, "save", false, "Save result as investigation artifact")
	kbAskCmd.Flags().StringVar(&kbAskModel, "model", "", "Model to use (default: sonnet for speed)")
	kbAskCmd.Flags().IntVar(&kbAskLimit, "limit", 3, "Maximum artifacts to read for context")
	kbAskCmd.Flags().BoolVarP(&kbAskGlobal, "global", "g", false, "Search across all known projects")

	kbExtractCmd.Flags().StringVar(&kbExtractTo, "to", "", "Target project name (required)")
	kbExtractCmd.Flags().BoolVar(&kbExtractUpdateSource, "update-source", false, "Add extracted-to reference in original file")

	kbClaimsCmd.Flags().BoolVar(&kbClaimsJSON, "json", false, "Output as JSON")
	kbClaimsCmd.Flags().BoolVar(&kbClaimsVerbose, "verbose", false, "Show individual claims")

	kbOrphansCmd.Flags().BoolVar(&kbOrphansJSON, "json", false, "Output as JSON")
	kbOrphansCmd.Flags().BoolVar(&kbOrphansStratified, "stratified", false, "Break orphans into categories: empty, negative-result, superseded, positive-unlinked")

	kbCreateCmd.AddCommand(kbCreateModelCmd)

	kbCmd.AddCommand(kbInitCmd)
	kbCmd.AddCommand(kbAskCmd)
	kbCmd.AddCommand(kbExtractCmd)
	kbCmd.AddCommand(kbClaimsCmd)
	kbCmd.AddCommand(kbOrphansCmd)
	kbCmd.AddCommand(kbCreateCmd)
	rootCmd.AddCommand(kbCmd)
}
