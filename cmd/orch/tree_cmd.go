package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/tree"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Display knowledge lineage tree visualization",
	Long: `Display knowledge lineage tree visualization.

Shows relationships between investigations, decisions, models, and issues.

Two views:
  - Knowledge view (default): investigations/decisions/models as primary nodes
  - Work view (--work): issues grouped by state as primary nodes

Examples:
  orch tree                           # Show full knowledge tree
  orch tree --work                    # Show work view (issues grouped by state)
  orch tree --cluster entropy-spiral  # Filter to specific cluster
  orch tree --depth 2                 # Limit depth to 2 levels
  orch tree --format json             # Output as JSON`,
	RunE: runTreeCmd,
}

var (
	treeCluster string
	treeDepth   int
	treeFormat  string
	treeWork    bool
)

func init() {
	treeCmd.Flags().StringVar(&treeCluster, "cluster", "", "Filter to specific cluster")
	treeCmd.Flags().IntVar(&treeDepth, "depth", 2, "Maximum depth to render (0 = unlimited)")
	treeCmd.Flags().StringVar(&treeFormat, "format", "text", "Output format: text or json")
	treeCmd.Flags().BoolVar(&treeWork, "work", false, "Show work view (issues as primary nodes)")

	rootCmd.AddCommand(treeCmd)
}

func runTreeCmd(cmd *cobra.Command, args []string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find .kb/ directory
	kbDir := filepath.Join(cwd, ".kb")
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		return fmt.Errorf(".kb/ directory not found in current directory")
	}

	opts := tree.TreeOptions{
		ClusterFilter: treeCluster,
		Depth:         treeDepth,
		Format:        treeFormat,
		WorkView:      treeWork,
	}

	// Validate format
	if opts.Format != "text" && opts.Format != "json" {
		return fmt.Errorf("invalid format %q, must be 'text' or 'json'", opts.Format)
	}

	if treeWork {
		// Work view: issues as primary nodes
		issues, err := tree.BuildWorkTree(kbDir, cwd, opts)
		if err != nil {
			return fmt.Errorf("failed to build work tree: %w", err)
		}

		output, err := tree.RenderWorkView(issues, opts)
		if err != nil {
			return fmt.Errorf("failed to render work view: %w", err)
		}

		fmt.Print(output)
	} else {
		// Knowledge view: investigations/decisions/models as primary nodes
		root, err := tree.BuildKnowledgeTree(kbDir, opts)
		if err != nil {
			return fmt.Errorf("failed to build knowledge tree: %w", err)
		}

		output, err := tree.RenderTree(root, opts)
		if err != nil {
			return fmt.Errorf("failed to render tree: %w", err)
		}

		fmt.Print(output)
	}

	return nil
}
