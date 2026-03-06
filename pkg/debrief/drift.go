package debrief

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// DriftItem represents aggregated model drift for one model domain.
type DriftItem struct {
	Domain       string // Normalized domain name (e.g., "agent-lifecycle-state-model")
	ModelPath    string // Path to the model file
	SpawnCount   int    // Number of stale spawns today
	ChangedFiles int    // Total changed files across events
	DeletedFiles int    // Total deleted files across events
}

// CollectDriftSummary aggregates today's staleness events into drift items.
// Groups by model path, counts stale spawns, and returns sorted by spawn count descending.
func CollectDriftSummary(events []spawn.StalenessEvent, today time.Time) []DriftItem {
	if len(events) == 0 {
		return nil
	}

	todayStr := today.Format("2006-01-02")

	// Filter to today and aggregate by model
	type agg struct {
		modelPath  string
		count      int
		changedSet map[string]struct{}
		deletedSet map[string]struct{}
	}

	aggregates := map[string]*agg{}
	for _, e := range events {
		// Filter to today
		ts := strings.TrimSpace(e.Timestamp)
		if ts == "" || !strings.HasPrefix(ts, todayStr) {
			continue
		}

		model := strings.TrimSpace(e.Model)
		if model == "" {
			continue
		}

		a := aggregates[model]
		if a == nil {
			a = &agg{
				modelPath:  model,
				changedSet: make(map[string]struct{}),
				deletedSet: make(map[string]struct{}),
			}
			aggregates[model] = a
		}
		a.count++
		for _, f := range e.ChangedFiles {
			if f != "" {
				a.changedSet[f] = struct{}{}
			}
		}
		for _, f := range e.DeletedFiles {
			if f != "" {
				a.deletedSet[f] = struct{}{}
			}
		}
	}

	if len(aggregates) == 0 {
		return nil
	}

	var items []DriftItem
	for _, a := range aggregates {
		domain := domainFromModelPath(a.modelPath)
		items = append(items, DriftItem{
			Domain:       domain,
			ModelPath:    a.modelPath,
			SpawnCount:   a.count,
			ChangedFiles: len(a.changedSet),
			DeletedFiles: len(a.deletedSet),
		})
	}

	// Sort by spawn count descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].SpawnCount > items[j].SpawnCount
	})

	return items
}

// FormatDriftSummary formats drift items into debrief bullet list lines.
func FormatDriftSummary(items []DriftItem) []string {
	if len(items) == 0 {
		return nil
	}

	var lines []string
	for _, item := range items {
		var parts []string
		parts = append(parts, fmt.Sprintf("%d stale spawn(s)", item.SpawnCount))
		if item.ChangedFiles > 0 {
			parts = append(parts, fmt.Sprintf("%d changed", item.ChangedFiles))
		}
		if item.DeletedFiles > 0 {
			parts = append(parts, fmt.Sprintf("%d deleted", item.DeletedFiles))
		}
		line := fmt.Sprintf("**%s:** %s", item.Domain, strings.Join(parts, ", "))
		lines = append(lines, line)
	}
	return lines
}

// domainFromModelPath extracts a readable domain name from a model path.
// e.g., ".kb/models/agent-lifecycle-state-model/MODEL.md" -> "agent-lifecycle-state-model"
func domainFromModelPath(modelPath string) string {
	// Look for .kb/models/<domain>/ pattern
	parts := strings.Split(modelPath, "/")
	for i, p := range parts {
		if p == "models" && i+1 < len(parts) {
			// Next part is the domain directory
			domain := parts[i+1]
			// If it looks like a file, use parent
			if strings.HasSuffix(domain, ".md") && i+2 <= len(parts) {
				return strings.TrimSuffix(domain, ".md")
			}
			return domain
		}
	}
	// Fallback: use basename without extension
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return strings.TrimSuffix(parts[i], ".md")
		}
	}
	return modelPath
}
