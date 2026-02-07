// Package attention provides types and interfaces for the composable attention architecture.
package attention

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// VerifyFailedEntry represents a single verification failure stored in JSONL.
type VerifyFailedEntry struct {
	BeadsID      string   `json:"beads_id"`
	Title        string   `json:"title,omitempty"`
	FailedGates  []string `json:"failed_gates"`
	Errors       []string `json:"errors"`
	PhaseSummary string   `json:"phase_summary,omitempty"`
	Escalation   string   `json:"escalation,omitempty"`
	Timestamp    int64    `json:"timestamp"`
}

// VerifyFailedCollector implements the Collector interface for verification failures.
// It reads from the verify-failed.jsonl file and surfaces issues that failed auto-completion.
type VerifyFailedCollector struct {
	// StoragePath is the path to the JSONL file storing verification failures.
	StoragePath string
	// MaxAgeHours is the maximum age of failures to surface (default: 72 hours).
	MaxAgeHours int
}

// DefaultVerifyFailedStoragePath returns the default path for verify-failed.jsonl.
func DefaultVerifyFailedStoragePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".orch", "verify-failed.jsonl")
	}
	return filepath.Join(home, ".orch", "verify-failed.jsonl")
}

// NewVerifyFailedCollector creates a new VerifyFailedCollector.
// If storagePath is empty, uses the default path.
// If maxAgeHours is <= 0, defaults to 72 hours.
func NewVerifyFailedCollector(storagePath string, maxAgeHours int) *VerifyFailedCollector {
	if storagePath == "" {
		storagePath = DefaultVerifyFailedStoragePath()
	}
	if maxAgeHours <= 0 {
		maxAgeHours = 72 // Default: 72 hours (3 days)
	}
	return &VerifyFailedCollector{
		StoragePath: storagePath,
		MaxAgeHours: maxAgeHours,
	}
}

// Collect gathers attention items for verification failures.
// These are Authority signals that require human review to resolve.
func (c *VerifyFailedCollector) Collect(role string) ([]AttentionItem, error) {
	// Load verification failures from JSONL
	entries, err := c.loadEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to load verification failures: %w", err)
	}

	if len(entries) == 0 {
		return []AttentionItem{}, nil
	}

	// Filter by age and deduplicate (keep latest per beads ID)
	cutoff := time.Now().Add(-time.Duration(c.MaxAgeHours) * time.Hour)
	latestByID := make(map[string]VerifyFailedEntry)

	for _, entry := range entries {
		entryTime := time.Unix(entry.Timestamp, 0)
		if entryTime.Before(cutoff) {
			continue // Too old
		}

		existing, exists := latestByID[entry.BeadsID]
		if !exists || entry.Timestamp > existing.Timestamp {
			latestByID[entry.BeadsID] = entry
		}
	}

	// Transform to attention items
	items := make([]AttentionItem, 0, len(latestByID))
	now := time.Now()

	for _, entry := range latestByID {
		// Calculate priority based on age and number of failures
		priority := calculateVerifyFailedPriority(entry, role)

		// Build summary
		gateStr := strings.Join(entry.FailedGates, ", ")
		summary := fmt.Sprintf("Verification failed: %s", gateStr)
		if entry.Title != "" {
			summary = fmt.Sprintf("%s: %s", entry.Title, summary)
		}

		// Build action hint
		actionHint := fmt.Sprintf("orch complete %s # Re-run verification", entry.BeadsID)

		item := AttentionItem{
			ID:          fmt.Sprintf("verify-failed-%s", entry.BeadsID),
			Source:      "daemon",
			Concern:     Authority, // Requires human intervention
			Signal:      "verify-failed",
			Subject:     entry.BeadsID,
			Summary:     summary,
			Priority:    priority,
			Role:        role,
			ActionHint:  actionHint,
			CollectedAt: now,
			Metadata: map[string]any{
				"failed_gates":  entry.FailedGates,
				"errors":        entry.Errors,
				"phase_summary": entry.PhaseSummary,
				"escalation":    entry.Escalation,
				"failed_at":     time.Unix(entry.Timestamp, 0).Format(time.RFC3339),
			},
		}
		items = append(items, item)
	}

	// Sort by priority (lower = higher priority)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Priority < items[j].Priority
	})

	return items, nil
}

// loadEntries reads all verification failure entries from the JSONL file.
func (c *VerifyFailedCollector) loadEntries() ([]VerifyFailedEntry, error) {
	data, err := os.ReadFile(c.StoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []VerifyFailedEntry{}, nil // File doesn't exist yet
		}
		return nil, err
	}

	var entries []VerifyFailedEntry
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry VerifyFailedEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// calculateVerifyFailedPriority determines priority based on role and failure severity.
// Lower numbers = higher priority.
func calculateVerifyFailedPriority(entry VerifyFailedEntry, role string) int {
	// Base priority for verification failures - high because work is stuck
	basePriority := 30

	// Adjust based on number of failed gates (more = higher priority)
	if len(entry.FailedGates) > 3 {
		basePriority -= 10
	} else if len(entry.FailedGates) > 1 {
		basePriority -= 5
	}

	// Adjust based on age (older = higher priority, stuck longer)
	age := time.Since(time.Unix(entry.Timestamp, 0))
	if age > 24*time.Hour {
		basePriority -= 15 // Very stuck
	} else if age > 6*time.Hour {
		basePriority -= 10
	} else if age > 2*time.Hour {
		basePriority -= 5
	}

	// Role-aware adjustments
	switch role {
	case "human":
		// Humans are the primary audience for verification failures
		return basePriority

	case "orchestrator":
		// Orchestrators care about stuck work
		return basePriority - 5

	case "daemon":
		// Daemons already tried, need human intervention
		return basePriority + 50

	default:
		return basePriority
	}
}

// StoreVerifyFailed persists a verification failure to the JSONL file.
// This is called by the daemon when auto-completion fails.
func StoreVerifyFailed(entry VerifyFailedEntry) error {
	storagePath := DefaultVerifyFailedStoragePath()

	// Ensure directory exists
	dir := filepath.Dir(storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Set timestamp if not set
	if entry.Timestamp == 0 {
		entry.Timestamp = time.Now().Unix()
	}

	// Open file for appending
	f, err := os.OpenFile(storagePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open storage file: %w", err)
	}
	defer f.Close()

	// Encode and write
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	return nil
}

// ClearVerifyFailed removes a verification failure entry for a beads ID.
// This is called when an issue is re-verified successfully or manually cleared.
func ClearVerifyFailed(beadsID string) error {
	storagePath := DefaultVerifyFailedStoragePath()

	// Load existing entries
	data, err := os.ReadFile(storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to clear
		}
		return err
	}

	// Filter out entries for this beads ID
	var remaining []string
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry VerifyFailedEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Keep malformed lines (don't lose data)
			remaining = append(remaining, line)
			continue
		}

		if entry.BeadsID != beadsID {
			remaining = append(remaining, line)
		}
	}

	// Rewrite file with filtered entries
	newData := strings.Join(remaining, "\n")
	if len(remaining) > 0 {
		newData += "\n"
	}

	if err := os.WriteFile(storagePath, []byte(newData), 0644); err != nil {
		return fmt.Errorf("failed to write filtered storage: %w", err)
	}

	return nil
}
