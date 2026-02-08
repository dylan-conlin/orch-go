package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	reflectJSON       bool   // Output as JSON
	reflectOrphans    bool   // Show only orphan investigations
	reflectSave       bool   // Save results to ~/.orch/reflect-suggestions.json
	reflectProjectDir string // Project directory for orphan detection
)

var reflectCmd = &cobra.Command{
	Use:   "reflect",
	Short: "Run knowledge base reflection with orphan investigation detection",
	Long: `Run kb reflect analysis augmented with orphan investigation detection.

This command wraps kb reflect and adds orphan investigation detection from
orch-go's verify package. Orphan investigations are those with similar-topic
peers but no prior-work citations, indicating potential lineage gaps.

Output categories:
  - Synthesis opportunities (topic clusters)
  - Promotion candidates (kn entries worth promoting)
  - Stale decisions (low citation count)
  - Constraint drift (potentially outdated)
  - Principle refinements
  - Investigation promotions (recommend-yes)
  - Investigation authority (grouped by authority level)
  - Defect-class clusters (recurring mechanisms)
  - Orphan investigations (lineage gaps)

Examples:
  orch reflect                  # Human-readable summary
  orch reflect --json           # Machine-readable JSON output
  orch reflect --orphans-only   # Show only orphan investigations
  orch reflect --save           # Save results to disk
  orch reflect --project /path  # Specify project directory`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReflect()
	},
}

func init() {
	reflectCmd.Flags().BoolVar(&reflectJSON, "json", false, "Output as JSON")
	reflectCmd.Flags().BoolVar(&reflectOrphans, "orphans-only", false, "Show only orphan investigation results")
	reflectCmd.Flags().BoolVar(&reflectSave, "save", false, "Save results to ~/.orch/reflect-suggestions.json")
	reflectCmd.Flags().StringVar(&reflectProjectDir, "project", "", "Project directory for orphan detection (default: current directory)")
	rootCmd.AddCommand(reflectCmd)
}

func runReflect() error {
	projectDir := reflectProjectDir
	if projectDir == "" {
		var err error
		projectDir, err = currentProjectDir()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	if reflectOrphans {
		return runReflectOrphansOnly(projectDir)
	}

	return runReflectFull(projectDir)
}

// runReflectFull runs kb reflect + orphan detection together.
func runReflectFull(projectDir string) error {
	suggestions, err := daemon.RunReflectionWithOrphans(false, projectDir)
	if err != nil {
		return fmt.Errorf("reflection failed: %w", err)
	}

	if reflectSave {
		if err := daemon.SaveSuggestions(suggestions); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save suggestions: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Saved to %s\n", daemon.SuggestionsPath())
		}
	}

	if reflectJSON {
		return printJSON(suggestions)
	}

	printReflectSummary(suggestions)
	return nil
}

// runReflectOrphansOnly runs only orphan investigation detection (no kb reflect).
func runReflectOrphansOnly(projectDir string) error {
	orphans, err := verify.DetectOrphanInvestigations(projectDir)
	if err != nil {
		return fmt.Errorf("orphan detection failed: %w", err)
	}

	if reflectJSON {
		return printJSON(orphans)
	}

	printOrphanSummary(orphans)
	return nil
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func printReflectSummary(s *daemon.ReflectSuggestions) {
	if !s.HasSuggestions() {
		fmt.Println("No reflection suggestions found.")
		return
	}

	fmt.Printf("Reflection: %s\n\n", s.Summary())

	if len(s.Synthesis) > 0 {
		fmt.Printf("Synthesis Opportunities (%d):\n", len(s.Synthesis))
		for _, item := range s.Synthesis {
			fmt.Printf("  %s (%d investigations)\n", item.Topic, item.Count)
			if item.Suggestion != "" {
				fmt.Printf("    %s\n", item.Suggestion)
			}
		}
		fmt.Println()
	}

	if len(s.Promote) > 0 {
		fmt.Printf("Promotion Candidates (%d):\n", len(s.Promote))
		for _, item := range s.Promote {
			fmt.Printf("  [%s] %s\n", item.ID, truncate(item.Content, 80))
			if item.Suggestion != "" {
				fmt.Printf("    %s\n", item.Suggestion)
			}
		}
		fmt.Println()
	}

	if len(s.Stale) > 0 {
		fmt.Printf("Stale Decisions (%d):\n", len(s.Stale))
		for _, item := range s.Stale {
			fmt.Printf("  %s (%d days old)\n", filepath.Base(item.Path), item.Age)
			if item.Suggestion != "" {
				fmt.Printf("    %s\n", item.Suggestion)
			}
		}
		fmt.Println()
	}

	if len(s.Drift) > 0 {
		fmt.Printf("Potential Drift (%d):\n", len(s.Drift))
		for _, item := range s.Drift {
			fmt.Printf("  [%s] %s\n", item.ID, truncate(item.Content, 80))
			if item.Suggestion != "" {
				fmt.Printf("    %s\n", item.Suggestion)
			}
		}
		fmt.Println()
	}

	if len(s.Refine) > 0 {
		fmt.Printf("Principle Refinements (%d):\n", len(s.Refine))
		for _, item := range s.Refine {
			fmt.Printf("  [%s] %s -> %s\n", item.ID, truncate(item.Content, 60), item.Principle)
			if item.Suggestion != "" {
				fmt.Printf("    %s\n", item.Suggestion)
			}
		}
		fmt.Println()
	}

	if len(s.InvestigationPromotion) > 0 {
		fmt.Printf("Investigation Promotions (%d):\n", len(s.InvestigationPromotion))
		for _, item := range s.InvestigationPromotion {
			fmt.Printf("  %s (%d days old)\n", item.Title, item.AgeDays)
			if item.Suggestion != "" {
				fmt.Printf("    %s\n", item.Suggestion)
			}
		}
		fmt.Println()
	}

	if len(s.InvestigationAuthority) > 0 {
		fmt.Printf("Recommendations by Authority (%d):\n", len(s.InvestigationAuthority))
		for _, item := range s.InvestigationAuthority {
			fmt.Printf("  [%s] %s (%d days)\n", item.Authority, item.Title, item.AgeDays)
			if item.NextAction != "" {
				fmt.Printf("    Next: %s\n", item.NextAction)
			}
		}
		fmt.Println()
	}

	if len(s.DefectClass) > 0 {
		fmt.Printf("Recurring Defect Classes (%d):\n", len(s.DefectClass))
		for _, item := range s.DefectClass {
			fmt.Printf("  %s (%d investigations in %d days)\n", item.DefectClass, item.Count, item.WindowDays)
			if item.Suggestion != "" {
				fmt.Printf("    %s\n", item.Suggestion)
			}
		}
		fmt.Println()
	}

	if len(s.OrphanInvestigations) > 0 {
		fmt.Printf("Orphan Investigations (%d):\n", len(s.OrphanInvestigations))
		for _, item := range s.OrphanInvestigations {
			fmt.Printf("  %s (topic: %s)\n", filepath.Base(item.Path), item.Topic)
			fmt.Printf("    %s\n", item.Suggestion)
			if len(item.SimilarInvestigations) > 0 {
				fmt.Printf("    Similar: %s\n", formatPaths(item.SimilarInvestigations))
			}
		}
		fmt.Println()
	}
}

func printOrphanSummary(o *verify.OrphanInvestigations) {
	fmt.Printf("Scanned %d investigations\n\n", o.TotalScanned)

	if !o.HasOrphans() {
		fmt.Println("No orphan investigations found.")
		return
	}

	fmt.Printf("Found %d orphan investigation(s) with potential lineage gaps:\n\n", len(o.Orphans))
	for _, item := range o.Orphans {
		fmt.Printf("  %s (topic: %s)\n", filepath.Base(item.Path), item.Topic)
		fmt.Printf("    %s\n", item.Suggestion)
		if len(item.SimilarInvestigations) > 0 {
			fmt.Printf("    Similar: %s\n", formatPaths(item.SimilarInvestigations))
		}
		fmt.Println()
	}
}

// formatPaths returns a comma-separated list of basenames from full paths.
func formatPaths(paths []string) string {
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = filepath.Base(p)
	}
	return strings.Join(names, ", ")
}
