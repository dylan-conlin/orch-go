package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/decisions"
	"github.com/spf13/cobra"
)

var decisionsCmd = &cobra.Command{
	Use:   "decisions",
	Short: "Decision lifecycle management",
	Long:  "Manage decision lifecycle: staleness detection, budget cap, enforcement audit.",
}

var decisionsStaleCmd = &cobra.Command{
	Use:   "stale",
	Short: "Find stale context-only decisions (>30d, uncited)",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := os.Getwd()
		result, err := decisions.FindStale(projectDir)
		if err != nil {
			return err
		}

		fmt.Printf("Active decisions: %d / %d budget cap\n", result.Active, result.Budget)
		if result.OverBy > 0 {
			fmt.Printf("⚠️  Over budget by %d — archive before adding new decisions\n", result.OverBy)
		}

		if len(result.Stale) == 0 {
			fmt.Println("No stale context-only decisions found.")
			return nil
		}

		fmt.Printf("\n%d stale context-only decisions (>%dd, 0 citations):\n",
			len(result.Stale), decisions.StaleThresholdDays)
		for _, d := range result.Stale {
			fmt.Printf("  %s  %s\n", d.Name, d.Title)
		}

		archiveFlag, _ := cmd.Flags().GetBool("archive")
		if archiveFlag {
			return archiveDecisions(projectDir, result.Stale)
		}

		fmt.Printf("\nRun with --archive to move these to .kb/decisions/archived/\n")
		return nil
	},
}

var decisionsBudgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Show decision budget status",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := os.Getwd()
		status, err := decisions.CheckBudget(projectDir)
		if err != nil {
			return err
		}

		fmt.Printf("Decision Budget: %d / %d\n", status.Active, status.Cap)
		if status.OverBy > 0 {
			fmt.Printf("⚠️  Over budget by %d\n", status.OverBy)
		} else {
			fmt.Printf("✓ Under budget (room for %d more)\n", status.Cap-status.Active)
		}

		fmt.Printf("\nBy enforcement type:\n")
		for _, et := range []decisions.EnforcementType{
			decisions.EnforcementGate,
			decisions.EnforcementHook,
			decisions.EnforcementConvention,
			decisions.EnforcementContextOnly,
		} {
			if count, ok := status.ByType[et]; ok && count > 0 {
				fmt.Printf("  %-15s %d\n", et, count)
			}
		}
		if status.Unclassified > 0 {
			fmt.Printf("  %-15s %d  (need **Enforcement:** field)\n", "unclassified", status.Unclassified)
		}

		return nil
	},
}

var decisionsAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit decisions missing enforcement type",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := os.Getwd()
		decs, err := decisions.ListActiveDecisions(projectDir)
		if err != nil {
			return err
		}

		var missing []decisions.Decision
		for _, d := range decs {
			if d.Enforcement == decisions.EnforcementUnknown {
				missing = append(missing, d)
			}
		}

		if len(missing) == 0 {
			fmt.Println("All active decisions have enforcement type declared.")
			return nil
		}

		fmt.Printf("%d decisions missing **Enforcement:** field:\n\n", len(missing))
		for _, d := range missing {
			fmt.Printf("  %s\n", d.Name)
		}
		fmt.Printf("\nValid types: gate, hook, convention, context-only\n")
		return nil
	},
}

func init() {
	decisionsStaleCmd.Flags().Bool("archive", false, "Auto-archive stale decisions")
	decisionsCmd.AddCommand(decisionsStaleCmd)
	decisionsCmd.AddCommand(decisionsBudgetCmd)
	decisionsCmd.AddCommand(decisionsAuditCmd)
}

// archiveDecisions moves decisions to .kb/decisions/archived/.
func archiveDecisions(projectDir string, decs []decisions.Decision) error {
	archiveDir := filepath.Join(projectDir, ".kb", "decisions", "archived")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return fmt.Errorf("creating archive dir: %w", err)
	}

	for _, d := range decs {
		src := d.Path
		dst := filepath.Join(archiveDir, filepath.Base(src))
		if err := os.Rename(src, dst); err != nil {
			// Try copy+delete if rename fails (cross-device)
			data, readErr := os.ReadFile(src)
			if readErr != nil {
				fmt.Fprintf(os.Stderr, "failed to archive %s: %v\n", d.Name, err)
				continue
			}
			if writeErr := os.WriteFile(dst, data, 0o644); writeErr != nil {
				fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", dst, writeErr)
				continue
			}
			os.Remove(src)
		}
		fmt.Printf("  archived: %s\n", d.Name)
	}

	remaining := strings.TrimSpace(fmt.Sprintf("%d decisions archived", len(decs)))
	fmt.Println(remaining)
	return nil
}
