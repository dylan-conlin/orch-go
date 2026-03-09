package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/dupdetect"
	"github.com/spf13/cobra"
)

var (
	dupdetectThreshold    float64
	dupdetectMinLines     int
	dupdetectCreateIssues bool
	dupdetectDryRun       bool
)

var dupdetectCmd = &cobra.Command{
	Use:   "dupdetect [dir]",
	Short: "Detect duplicate functions and optionally create beads issues",
	Long: `Scan Go source files for structurally similar functions using AST fingerprinting.

When --create-issues is set, auto-creates beads issues for each duplicate pair
found above the similarity threshold. Uses title-based dedup to avoid duplicates.

Examples:
  orch dupdetect                          # Scan current project, print results
  orch dupdetect --create-issues          # Scan and create beads issues
  orch dupdetect --create-issues --dry-run # Show what would be created
  orch dupdetect --threshold 0.90         # Higher similarity threshold
  orch dupdetect cmd/orch/                # Scan specific directory only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDupdetect(args)
	},
}

func init() {
	dupdetectCmd.Flags().Float64Var(&dupdetectThreshold, "threshold", 0.80, "Similarity threshold (0.0-1.0)")
	dupdetectCmd.Flags().IntVar(&dupdetectMinLines, "min-lines", 10, "Minimum function body lines")
	dupdetectCmd.Flags().BoolVar(&dupdetectCreateIssues, "create-issues", false, "Auto-create beads issues for duplicates")
	dupdetectCmd.Flags().BoolVar(&dupdetectDryRun, "dry-run", false, "Show what issues would be created (requires --create-issues)")
	rootCmd.AddCommand(dupdetectCmd)
}

func runDupdetect(args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	d := dupdetect.NewDetector()
	d.Threshold = dupdetectThreshold
	d.MinBodyLines = dupdetectMinLines

	var pairs []dupdetect.DupPair

	if len(args) > 0 {
		// Scan specific directory
		pairs, err = d.ScanDir(args[0])
	} else {
		// Scan entire project
		pairs, err = d.ScanProject(projectDir)
	}
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	if len(pairs) == 0 {
		fmt.Println("No duplicate functions found above threshold.")
		return nil
	}

	// Print results
	fmt.Printf("Found %d duplicate function pair(s):\n\n", len(pairs))
	for i, pair := range pairs {
		fmt.Printf("  %d. %.0f%% similar:\n", i+1, pair.Similarity*100)
		fmt.Printf("     %s in %s (line %d, %d lines)\n", pair.FuncA.Name, pair.FuncA.File, pair.FuncA.StartLine, pair.FuncA.Lines)
		fmt.Printf("     %s in %s (line %d, %d lines)\n", pair.FuncB.Name, pair.FuncB.File, pair.FuncB.StartLine, pair.FuncB.Lines)
		fmt.Println()
	}

	if !dupdetectCreateIssues {
		return nil
	}

	if dupdetectDryRun {
		fmt.Println("Dry run — would create the following issues:")
		for _, pair := range pairs {
			fmt.Printf("  - %s\n", dupdetect.DupPairTitle(pair))
		}
		return nil
	}

	// Create beads issues
	client := beads.NewCLIClient(beads.WithWorkDir(projectDir))

	result, err := dupdetect.ReportToBeads(client, pairs, dupdetect.ReportConfig{
		Threshold:  dupdetectThreshold,
		ProjectDir: projectDir,
	})
	if err != nil {
		return fmt.Errorf("report to beads: %w", err)
	}

	fmt.Printf("Created %d beads issue(s).\n", result.Created)
	for _, id := range result.IssueIDs {
		fmt.Printf("  - %s\n", id)
	}
	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d error(s):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
	}

	return nil
}
