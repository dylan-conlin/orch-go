package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/findingdedup"
	"github.com/spf13/cobra"
)

var (
	kbFindingsJSON      bool
	kbFindingsThreshold float64
	kbFindingsMinSize   int
)

var kbFindingsCmd = &cobra.Command{
	Use:   "findings",
	Short: "Detect duplicate findings across investigations — anti-coherence check",
	Long: `Scan investigation and synthesis files for duplicate findings.

When the same insight appears in 3+ investigations with different wording,
the system is narrating rather than learning — regenerating conclusions
instead of building on prior work.

Scans:
  .kb/investigations/    (structured Finding N: format)
  .orch/workspace/       (SYNTHESIS.md Evidence/Knowledge sections)

Examples:
  orch kb findings                     # Human-readable report
  orch kb findings --json              # Machine-readable output
  orch kb findings --threshold 0.25    # Adjust similarity threshold
  orch kb findings --min-size 2        # Show pairs (not just 3+)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKBFindings()
	},
}

func runKBFindings() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	d := findingdedup.NewDetector()
	if kbFindingsThreshold > 0 {
		d.Threshold = kbFindingsThreshold
	}
	if kbFindingsMinSize > 0 {
		d.MinClusterSize = kbFindingsMinSize
	}

	// Scan investigation directories
	dirs := []string{
		filepath.Join(projectDir, ".kb", "investigations"),
	}

	// Also scan archived investigations
	archivedDir := filepath.Join(projectDir, ".kb", "investigations", "archived")
	if info, err := os.Stat(archivedDir); err == nil && info.IsDir() {
		dirs = append(dirs, archivedDir)
	}

	// Scan synthesized investigations
	synthDir := filepath.Join(projectDir, ".kb", "investigations", "synthesized")
	if info, err := os.Stat(synthDir); err == nil && info.IsDir() {
		dirs = append(dirs, synthDir)
		// Also check synthesis-meta subdirectory
		synthMetaDir := filepath.Join(synthDir, "synthesis-meta")
		if info, err := os.Stat(synthMetaDir); err == nil && info.IsDir() {
			dirs = append(dirs, synthMetaDir)
		}
	}

	// Scan workspace SYNTHESIS.md files
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	if info, err := os.Stat(workspaceDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(workspaceDir)
		for _, entry := range entries {
			if entry.IsDir() {
				synthPath := filepath.Join(workspaceDir, entry.Name(), "SYNTHESIS.md")
				if _, err := os.Stat(synthPath); err == nil {
					// Add the workspace subdirectory
					dirs = append(dirs, filepath.Join(workspaceDir, entry.Name()))
				}
			}
		}
	}

	clusters, err := d.ScanDirs(dirs...)
	if err != nil {
		return fmt.Errorf("scan findings: %w", err)
	}

	if kbFindingsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(clusters)
	}

	fmt.Print(findingdedup.FormatReport(clusters))
	return nil
}

func init() {
	kbFindingsCmd.Flags().BoolVar(&kbFindingsJSON, "json", false, "Output as JSON")
	kbFindingsCmd.Flags().Float64Var(&kbFindingsThreshold, "threshold", 0, "Similarity threshold (default 0.20)")
	kbFindingsCmd.Flags().IntVar(&kbFindingsMinSize, "min-size", 0, "Minimum cluster size (default 3)")

	kbCmd.AddCommand(kbFindingsCmd)
}
