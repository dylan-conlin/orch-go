// Package main provides the registry command for managing the agent registry.
package main

import (
	"fmt"
	"strings"
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
	registryCleanUntracked bool
)

var registryCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove stale registry entries",
	Long: `Remove stale entries from the agent registry.

The registry can accumulate stale entries over time (e.g., from agents that
completed or were abandoned). This command removes entries older than a
specified age based on their spawn time.

Use --untracked to specifically remove entries for untracked agents
(spawned with --no-track). These agents have synthetic beads IDs that
don't exist in beads and can get stuck as "active" in the registry.

By default, this command runs in dry-run mode (shows what would be deleted
without actually deleting). Use --execute to actually remove entries.

Examples:
  orch registry clean --older-than 7d              # Preview: remove entries older than 7 days
  orch registry clean --older-than 7d --execute    # Actually remove entries older than 7 days
  orch registry clean --untracked                  # Preview: remove untracked agent entries
  orch registry clean --untracked --execute        # Actually remove untracked agent entries
  orch registry clean --older-than 30d --execute   # Remove entries older than 30 days`,
	RunE: runRegistryClean,
}

func init() {
	registryCmd.AddCommand(registryCleanCmd)

	registryCleanCmd.Flags().StringVar(&registryCleanOlderThan, "older-than", "", "Remove entries older than this duration (e.g., 7d, 30d, 168h)")
	registryCleanCmd.Flags().BoolVar(&registryCleanDryRun, "dry-run", true, "Show what would be deleted without making changes (default: true)")
	registryCleanCmd.Flags().BoolVar(&registryCleanExecute, "execute", false, "Actually delete entries (overrides --dry-run)")
	registryCleanCmd.Flags().BoolVar(&registryCleanUntracked, "untracked", false, "Remove untracked agent entries (--no-track spawns with synthetic beads IDs)")
}

func runRegistryClean(cmd *cobra.Command, args []string) error {
	// Require at least one filter flag
	if registryCleanOlderThan == "" && !registryCleanUntracked {
		return fmt.Errorf("specify --older-than or --untracked (or both)")
	}

	// Load registry
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	all := reg.ListAll()
	fmt.Printf("Registry: %d total entries\n", len(all))

	// Build predicate for what to remove
	var olderThanCutoff time.Time
	if registryCleanOlderThan != "" {
		duration, err := parseDuration(registryCleanOlderThan)
		if err != nil {
			return fmt.Errorf("invalid --older-than value: %w", err)
		}
		olderThanCutoff = time.Now().Add(-duration)
	}

	// Identify entries to remove
	var toRemove []*registry.Agent
	for _, agent := range all {
		if shouldRemoveRegistryEntry(agent, registryCleanUntracked, olderThanCutoff) {
			toRemove = append(toRemove, agent)
		}
	}

	if len(toRemove) == 0 {
		fmt.Println("No matching entries found.")
		return nil
	}

	// Report what would be removed
	fmt.Printf("\nEntries to remove: %d\n", len(toRemove))
	limit := len(toRemove)
	if limit > 15 {
		limit = 15
	}
	for i := 0; i < limit; i++ {
		agent := toRemove[i]
		reason := registryRemoveReason(agent, registryCleanUntracked, olderThanCutoff)
		fmt.Printf("  - %s (beads: %s, status: %s, %s)\n",
			agent.ID,
			formatBeadsIDForDisplay(agent.BeadsID),
			agent.Status,
			reason,
		)
	}
	if len(toRemove) > 15 {
		fmt.Printf("  ... and %d more\n", len(toRemove)-15)
	}

	// Dry-run check
	isDryRun := registryCleanDryRun && !registryCleanExecute
	if isDryRun {
		fmt.Printf("\n[DRY-RUN] Would remove %d entries. Use --execute to actually delete.\n", len(toRemove))
		return nil
	}

	// Build a set of IDs to remove for O(1) lookup
	removeIDs := make(map[string]bool, len(toRemove))
	for _, a := range toRemove {
		removeIDs[a.ID] = true
	}

	removed := reg.Purge(func(a *registry.Agent) bool {
		return removeIDs[a.ID]
	})

	if err := reg.SaveSkipMerge(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Printf("\nRemoved %d entries. Registry now contains %d entries.\n", removed, len(all)-removed)
	return nil
}

// shouldRemoveRegistryEntry returns true if the agent should be removed based on active filters.
func shouldRemoveRegistryEntry(agent *registry.Agent, untracked bool, olderThan time.Time) bool {
	matchesUntracked := untracked && isUntrackedRegistryEntry(agent)
	matchesAge := !olderThan.IsZero() && isOlderThan(agent, olderThan)

	// If both filters are active, match either (union)
	if untracked && !olderThan.IsZero() {
		return matchesUntracked || matchesAge
	}
	return matchesUntracked || matchesAge
}

// isUntrackedRegistryEntry returns true if the agent is an untracked spawn.
func isUntrackedRegistryEntry(agent *registry.Agent) bool {
	return agent.BeadsID != "" && strings.Contains(agent.BeadsID, "-untracked-")
}

// isOlderThan returns true if the agent was spawned before the cutoff.
func isOlderThan(agent *registry.Agent, cutoff time.Time) bool {
	spawnTime, err := time.Parse(registry.TimeFormat, agent.SpawnedAt)
	if err != nil {
		return false // Can't parse, don't remove
	}
	return spawnTime.Before(cutoff)
}

// registryRemoveReason returns a human-readable reason why the entry would be removed.
func registryRemoveReason(agent *registry.Agent, untracked bool, olderThan time.Time) string {
	reasons := make([]string, 0, 2)
	if untracked && isUntrackedRegistryEntry(agent) {
		reasons = append(reasons, "untracked")
	}
	if !olderThan.IsZero() && isOlderThan(agent, olderThan) {
		spawnTime, _ := time.Parse(registry.TimeFormat, agent.SpawnedAt)
		age := time.Since(spawnTime)
		reasons = append(reasons, fmt.Sprintf("age: %dd", int(age.Hours()/24)))
	}
	return strings.Join(reasons, ", ")
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
