// Package orient provides session orientation data for orchestrator consumption.
//
// The orient package aggregates data from multiple sources (beads, events, models)
// to produce a structured session start briefing. It surfaces throughput baselines,
// ready work, relevant models, and freshness warnings.
package orient

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	// StaleThresholdDays is the number of days after which a model is considered stale.
	StaleThresholdDays = 14

	// ProbeRecencyDays is the window for "recent" probes.
	ProbeRecencyDays = 30
)

// ModelFreshness holds freshness metadata for a single knowledge model.
type ModelFreshness struct {
	Name            string
	Summary         string
	LastUpdated     time.Time
	AgeDays         int
	HasRecentProbes bool
	LatestProbeDate time.Time
}

// IsStale returns true if the model exceeds the staleness threshold
// and has no recent probes.
func (m ModelFreshness) IsStale() bool {
	return m.AgeDays > StaleThresholdDays && !m.HasRecentProbes
}

// ScanModelFreshness reads .kb/models/ and returns freshness data for each model.
// Returns empty slice (not error) if directory doesn't exist.
func ScanModelFreshness(modelsDir string) ([]ModelFreshness, error) {
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []ModelFreshness
	now := time.Now()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Skip non-model directories
		if name == "archived" || strings.HasPrefix(name, ".") {
			continue
		}

		modelPath := filepath.Join(modelsDir, name, "model.md")
		content, err := os.ReadFile(modelPath)
		if err != nil {
			continue // Skip models without model.md
		}

		contentStr := string(content)

		lastUpdated, ok := extractLastUpdated(contentStr)
		if !ok {
			continue // Skip models without parseable dates
		}

		ageDays := int(now.Sub(lastUpdated).Hours() / 24)
		if ageDays < 0 {
			ageDays = 0
		}

		summary := extractSummary(contentStr)

		hasRecentProbes, latestProbe := scanProbes(filepath.Join(modelsDir, name, "probes"), now)

		results = append(results, ModelFreshness{
			Name:            name,
			Summary:         summary,
			LastUpdated:     lastUpdated,
			AgeDays:         ageDays,
			HasRecentProbes: hasRecentProbes,
			LatestProbeDate: latestProbe,
		})
	}

	return results, nil
}

// FilterStaleModels returns up to maxCount stale models, sorted by age descending (stalest first).
func FilterStaleModels(models []ModelFreshness, maxCount int) []ModelFreshness {
	var stale []ModelFreshness
	for _, m := range models {
		if m.IsStale() {
			stale = append(stale, m)
		}
	}

	sort.Slice(stale, func(i, j int) bool {
		return stale[i].AgeDays > stale[j].AgeDays
	})

	if len(stale) > maxCount {
		stale = stale[:maxCount]
	}

	return stale
}

// HumanAge formats an age in days as a human-readable string.
func HumanAge(days int) string {
	if days == 0 {
		return "today"
	}
	return strings.Replace(strings.TrimSpace(
		strings.Replace(string(rune(days+'0'))+"d ago", string(rune(days+'0')), intToStr(days), 1),
	), " ", " ", 1)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

var lastUpdatedRe = regexp.MustCompile(`\*\*Last Updated:\*\*\s+(\d{4}-\d{2}-\d{2})`)

// extractLastUpdated parses the **Last Updated:** field from model.md content.
func extractLastUpdated(content string) (time.Time, bool) {
	matches := lastUpdatedRe.FindStringSubmatch(content)
	if len(matches) < 2 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

var summaryHeaderRe = regexp.MustCompile(`(?m)^## Summary`)

// extractSummary extracts the summary section from model.md content.
// Returns the first paragraph after the ## Summary header, up to the next --- or ## heading.
func extractSummary(content string) string {
	loc := summaryHeaderRe.FindStringIndex(content)
	if loc == nil {
		return ""
	}

	// Skip past the header line
	rest := content[loc[1]:]
	// Skip header suffix (e.g., " (30 seconds)") and newlines
	if idx := strings.Index(rest, "\n"); idx >= 0 {
		rest = rest[idx+1:]
	}

	// Trim leading whitespace/newlines
	rest = strings.TrimLeft(rest, "\n\r ")

	// Find end of summary paragraph (next ---, ## heading, or double newline after content)
	var summary strings.Builder
	for _, line := range strings.Split(rest, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" || strings.HasPrefix(trimmed, "## ") {
			break
		}
		if trimmed == "" && summary.Len() > 0 {
			break
		}
		if trimmed == "" {
			continue
		}
		if summary.Len() > 0 {
			summary.WriteString(" ")
		}
		summary.WriteString(trimmed)
	}

	return summary.String()
}

var probeDateRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)

// scanProbes checks if a model has recent probes.
func scanProbes(probesDir string, now time.Time) (hasRecent bool, latest time.Time) {
	entries, err := os.ReadDir(probesDir)
	if err != nil {
		return false, time.Time{}
	}

	threshold := now.AddDate(0, 0, -ProbeRecencyDays)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		matches := probeDateRe.FindStringSubmatch(entry.Name())
		if len(matches) < 2 {
			continue
		}
		probeDate, err := time.Parse("2006-01-02", matches[1])
		if err != nil {
			continue
		}
		if probeDate.After(latest) {
			latest = probeDate
		}
		if probeDate.After(threshold) {
			hasRecent = true
		}
	}

	return hasRecent, latest
}
