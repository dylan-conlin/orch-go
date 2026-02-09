package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const batchSkipFilename = "batch-skip-issues.json"

type batchSkipFile struct {
	Issues []string `json:"issues"`
}

var skipClearAll bool

var skipSetCmd = &cobra.Command{
	Use:   "skip-set [beads-id...]",
	Short: "Add issues to batch-complete skip list",
	Long: `Add one or more issues to the project-level batch completion skip list.

Issues in this list are skipped by orch batch-complete (including --all).`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, err := currentProjectDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		skipSet, err := readBatchSkipSet(projectDir)
		if err != nil {
			return err
		}

		resolvedIDs, err := resolveBatchSkipIDs(args)
		if err != nil {
			return err
		}

		added := 0
		already := 0
		for _, id := range resolvedIDs {
			if _, exists := skipSet[id]; exists {
				already++
				continue
			}
			skipSet[id] = struct{}{}
			added++
		}

		if err := writeBatchSkipSet(projectDir, skipSet); err != nil {
			return err
		}

		fmt.Printf("Skip list updated: %d added, %d already present (total: %d)\n", added, already, len(skipSet))
		return nil
	},
}

var skipListCmd = &cobra.Command{
	Use:   "skip-list",
	Short: "List issues skipped by batch-complete",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, err := currentProjectDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		skipSet, err := readBatchSkipSet(projectDir)
		if err != nil {
			return err
		}

		ids := sortedBatchSkipIDs(skipSet)
		if len(ids) == 0 {
			fmt.Println("No skipped issues configured")
			return nil
		}

		fmt.Printf("Skipped issues (%d):\n", len(ids))
		for _, id := range ids {
			fmt.Printf("  %s\n", id)
		}

		return nil
	},
}

var skipClearCmd = &cobra.Command{
	Use:   "skip-clear [beads-id...]",
	Short: "Remove issues from batch-complete skip list",
	Long: `Remove one or more issues from the project-level batch completion skip list.
Use --all to clear the list entirely.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if skipClearAll {
			return nil
		}
		return cobra.MinimumNArgs(1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, err := currentProjectDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		skipSet, err := readBatchSkipSet(projectDir)
		if err != nil {
			return err
		}

		if len(skipSet) == 0 {
			fmt.Println("Skip list is already empty")
			return nil
		}

		if skipClearAll {
			cleared := len(skipSet)
			if err := writeBatchSkipSet(projectDir, map[string]struct{}{}); err != nil {
				return err
			}
			fmt.Printf("Cleared %d skipped issue(s)\n", cleared)
			return nil
		}

		resolvedIDs, err := resolveBatchSkipIDs(args)
		if err != nil {
			return err
		}

		removed := 0
		missing := 0
		for _, id := range resolvedIDs {
			if _, exists := skipSet[id]; !exists {
				missing++
				continue
			}
			delete(skipSet, id)
			removed++
		}

		if err := writeBatchSkipSet(projectDir, skipSet); err != nil {
			return err
		}

		fmt.Printf("Skip list updated: %d removed, %d not found (remaining: %d)\n", removed, missing, len(skipSet))
		return nil
	},
}

func init() {
	skipClearCmd.Flags().BoolVar(&skipClearAll, "all", false, "Clear all skipped issues")
}

func batchSkipPath(projectDir string) string {
	return filepath.Join(projectDir, ".orch", batchSkipFilename)
}

func readBatchSkipSet(projectDir string) (map[string]struct{}, error) {
	path := batchSkipPath(projectDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]struct{}{}, nil
		}
		return nil, fmt.Errorf("failed to read batch skip list: %w", err)
	}

	if strings.TrimSpace(string(data)) == "" {
		return map[string]struct{}{}, nil
	}

	var file batchSkipFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse batch skip list %s: %w", path, err)
	}

	skipSet := make(map[string]struct{}, len(file.Issues))
	for _, id := range file.Issues {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		skipSet[id] = struct{}{}
	}

	return skipSet, nil
}

func writeBatchSkipSet(projectDir string, skipSet map[string]struct{}) error {
	path := batchSkipPath(projectDir)

	if len(skipSet) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to clear batch skip list: %w", err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create .orch directory: %w", err)
	}

	file := batchSkipFile{Issues: sortedBatchSkipIDs(skipSet)}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal batch skip list: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write batch skip list: %w", err)
	}

	return nil
}

func sortedBatchSkipIDs(skipSet map[string]struct{}) []string {
	ids := make([]string, 0, len(skipSet))
	for id := range skipSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func resolveBatchSkipIDs(args []string) ([]string, error) {
	ids := make([]string, 0, len(args))
	for _, rawID := range args {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}

		resolvedID, err := resolveShortBeadsID(id)
		if err == nil {
			ids = append(ids, resolvedID)
			continue
		}

		// If this already looks like a full beads ID, allow it as-is.
		if strings.Contains(id, "-") {
			ids = append(ids, id)
			continue
		}

		return nil, fmt.Errorf("failed to resolve beads ID %q: %w", id, err)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no valid beads IDs provided")
	}

	return ids, nil
}
