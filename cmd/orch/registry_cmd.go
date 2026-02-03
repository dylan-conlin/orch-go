// Package main provides the registry command for managing the agent registry.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage the agent registry",
	Long: `Manage the agent registry for orch-go.

The registry is a spawn-time metadata cache that stores agent information.
It can accumulate stale entries over time and needs periodic cleanup.`,
}

var (
	registryCleanOlderThan string
	registryCleanDryRun    bool
	registryCleanExecute   bool
)

var registryCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove stale registry entries",
	Long: `Remove stale entries from the agent registry.

The registry can accumulate stale entries over time (e.g., from agents that
completed or were abandoned). This command removes entries older than a
specified age based on their spawn time.

By default, this command runs in dry-run mode (shows what would be deleted
without actually deleting). Use --execute to actually remove entries.

Examples:
  orch registry clean --older-than 7d          # Preview: remove entries older than 7 days
  orch registry clean --older-than 7d --execute   # Actually remove entries older than 7 days
  orch registry clean --older-than 30d --execute  # Remove entries older than 30 days
  orch registry clean --older-than 168h --execute # Remove entries older than 1 week (in hours)`,
	RunE: runRegistryClean,
}

func init() {
	registryCmd.AddCommand(registryCleanCmd)

	registryCleanCmd.Flags().StringVar(&registryCleanOlderThan, "older-than", "7d", "Remove entries older than this duration (e.g., 7d, 30d, 168h)")
	registryCleanCmd.Flags().BoolVar(&registryCleanDryRun, "dry-run", true, "Show what would be deleted without making changes (default: true)")
	registryCleanCmd.Flags().BoolVar(&registryCleanExecute, "execute", false, "Actually delete entries (overrides --dry-run)")
}

func runRegistryClean(cmd *cobra.Command, args []string) error {
	// Parse duration
	duration, err := parseDuration(registryCleanOlderThan)
	if err != nil {
		return fmt.Errorf("invalid --older-than value: %w", err)
	}

	// Load registry
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Get all agents (including non-active ones)
	agents := reg.ListAgents()
	initialCount := len(agents)

	// Calculate cutoff time
	cutoff := time.Now().Add(-duration)

	// Filter agents older than cutoff
	var toRemove []*registry.Agent
	var toKeep []*registry.Agent
	for _, agent := range agents {
		spawnTime, err := time.Parse(registry.TimeFormat, agent.SpawnedAt)
		if err != nil {
			// If we can't parse the spawn time, keep the agent (safer)
			toKeep = append(toKeep, agent)
			continue
		}

		if spawnTime.Before(cutoff) {
			toRemove = append(toRemove, agent)
		} else {
			toKeep = append(toKeep, agent)
		}
	}

	// Report findings
	fmt.Printf("Registry cleanup report:\n")
	fmt.Printf("  Total entries: %d\n", initialCount)
	fmt.Printf("  Entries older than %s: %d\n", registryCleanOlderThan, len(toRemove))
	fmt.Printf("  Entries to keep: %d\n", len(toKeep))
	fmt.Println()

	if len(toRemove) == 0 {
		fmt.Println("No stale entries found.")
		return nil
	}

	// Show sample of what will be removed
	fmt.Printf("Sample of entries to remove (showing up to 10):\n")
	sampleCount := len(toRemove)
	if sampleCount > 10 {
		sampleCount = 10
	}
	for i := 0; i < sampleCount; i++ {
		agent := toRemove[i]
		spawnTime, _ := time.Parse(registry.TimeFormat, agent.SpawnedAt)
		age := time.Since(spawnTime)
		fmt.Printf("  - %s (age: %dd, beads: %s, status: %s)\n",
			agent.ID,
			int(age.Hours()/24),
			agent.BeadsID,
			agent.Status,
		)
	}

	if len(toRemove) > 10 {
		fmt.Printf("  ... and %d more\n", len(toRemove)-10)
	}
	fmt.Println()

	// Check if we should actually delete
	isDryRun := registryCleanDryRun && !registryCleanExecute
	if isDryRun {
		fmt.Printf("[DRY-RUN] Would remove %d entries. Use --execute to actually delete.\n", len(toRemove))
		return nil
	}

	// Actually remove entries by rebuilding the registry with only entries to keep
	fmt.Printf("Removing %d entries...\n", len(toRemove))

	// Since the registry package doesn't provide bulk delete, we'll rebuild it directly
	// by creating a new registry data structure with only the entries to keep
	registryPath := registry.DefaultPath()

	// Create registryData structure for JSON marshaling
	type registryData struct {
		Agents []*registry.Agent `json:"agents"`
	}

	newData := registryData{
		Agents: toKeep,
	}

	// Marshal to JSON with indentation (matching the original format)
	jsonData, err := json.MarshalIndent(newData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write the new registry file
	if err := os.WriteFile(registryPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	fmt.Printf("Successfully removed %d entries.\n", len(toRemove))
	fmt.Printf("Registry now contains %d entries.\n", len(toKeep))

	return nil
}

// parseDuration parses a duration string (e.g., "7d", "30d", "168h").
// Supports common suffixes: d (days), h (hours), m (minutes), s (seconds).
func parseDuration(s string) (time.Duration, error) {
	// Handle days suffix specially since time.ParseDuration doesn't support it
	if len(s) > 1 && s[len(s)-1] == 'd' {
		days := s[:len(s)-1]
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		return time.Duration(d) * 24 * time.Hour, nil
	}

	// For other suffixes, use standard parsing
	return time.ParseDuration(s)
}
