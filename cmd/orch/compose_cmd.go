package main

import (
	"fmt"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/compose"
	"github.com/dylan-conlin/orch-go/pkg/identity"
	"github.com/spf13/cobra"
)

var composeWorkdir string

var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Compose briefs into a digest — cluster by content similarity",
	Long: `Scan .kb/briefs/ for all briefs, cluster them by keyword overlap,
match clusters against .kb/threads/, and write a digest to .kb/digests/.

The digest surfaces cross-cutting patterns, harvests tension sections,
and proposes thread connections — all as draft proposals for human review.

Composition does NOT drain the comprehension queue. Clustering is a
navigation aid, not a comprehension act.

Examples:
  orch compose                    # Compose all briefs into today's digest
  orch compose --workdir /path    # Compose from a specific project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _, err := identity.ResolveProjectDirectory(composeWorkdir)
		if err != nil {
			return fmt.Errorf("resolving project directory: %w", err)
		}

		briefsDir := filepath.Join(projectDir, ".kb", "briefs")
		threadsDir := filepath.Join(projectDir, ".kb", "threads")
		digestsDir := filepath.Join(projectDir, ".kb", "digests")

		fmt.Printf("Composing briefs from %s...\n", briefsDir)

		digest, err := compose.Compose(briefsDir, threadsDir)
		if err != nil {
			return fmt.Errorf("composing briefs: %w", err)
		}

		path, err := compose.WriteDigest(digest, digestsDir)
		if err != nil {
			return fmt.Errorf("writing digest: %w", err)
		}

		// Summary output
		fmt.Printf("\nDigest written: %s\n", path)
		fmt.Printf("  Briefs composed: %d\n", digest.BriefsComposed)
		fmt.Printf("  Clusters found:  %d\n", digest.ClustersFound)
		fmt.Printf("  Unclustered:     %d\n", len(digest.Unclustered))
		fmt.Printf("  Tension orphans: %d\n", len(digest.TensionOrphans))

		if digest.ClustersFound > 0 {
			fmt.Println("\nClusters:")
			for i, dc := range digest.Clusters {
				fmt.Printf("  %d. %s (%d briefs)\n", i+1, dc.Name, len(dc.Briefs))
				if len(dc.ThreadMatches) > 0 {
					fmt.Printf("     → thread: %s\n", dc.ThreadMatches[0].Thread.Title)
				}
			}
		}

		return nil
	},
}

func init() {
	composeCmd.Flags().StringVar(&composeWorkdir, "workdir", "", "Project directory (default: current)")
}
