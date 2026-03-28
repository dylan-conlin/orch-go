package orient

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	digestBriefsLineRe      = regexp.MustCompile(`\*\*Briefs:\*\*\s+(.+)$`)
	digestUnclusteredLineRe = regexp.MustCompile(`^- \*\*([\w-]+)\*\*`)
)

// DigestSummary holds aggregated digest data for orient display.
type DigestSummary struct {
	DigestCount      int `json:"digest_count"`
	BriefsComposed   int `json:"briefs_composed"`
	ClustersFound    int `json:"clusters_found"`
	MaintenanceCount int `json:"maintenance_count,omitempty"`
}

// ScanRecentDigests reads .kb/digests/ for files newer than prevSessionDate.
// Returns an aggregated summary across all matching digests, or nil if none found.
// If prevSessionDate is zero, all digests are included.
func ScanRecentDigests(digestsDir string, prevSessionDate time.Time) *DigestSummary {
	entries, err := os.ReadDir(digestsDir)
	if err != nil {
		return nil
	}

	var summary DigestSummary

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		path := filepath.Join(digestsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		fm := parseDigestFrontmatter(string(data))
		if fm.date.IsZero() {
			continue
		}

		// Skip digests older than previous session (same-day digests are included)
		if !prevSessionDate.IsZero() && fm.date.Before(prevSessionDate) {
			continue
		}

		summary.DigestCount++
		summary.BriefsComposed += fm.briefsComposed
		summary.ClustersFound += fm.clustersFound
	}

	if summary.DigestCount == 0 {
		return nil
	}

	return &summary
}

// DigestedBriefIDs returns the set of brief IDs already included in digest artifacts.
func DigestedBriefIDs(digestsDir string) map[string]bool {
	entries, err := os.ReadDir(digestsDir)
	if err != nil {
		return nil
	}

	ids := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		path := filepath.Join(digestsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if m := digestBriefsLineRe.FindStringSubmatch(line); len(m) == 2 {
				for _, id := range strings.Split(m[1], ",") {
					id = strings.TrimSpace(id)
					if id != "" {
						ids[id] = true
					}
				}
				continue
			}
			if m := digestUnclusteredLineRe.FindStringSubmatch(line); len(m) == 2 {
				ids[m[1]] = true
			}
		}
	}

	if len(ids) == 0 {
		return nil
	}
	return ids
}

// CountMaintenanceBriefs counts briefs with `category: maintenance` in their frontmatter.
// Returns 0 if no briefs have category metadata (Phase 1 not yet implemented).
func CountMaintenanceBriefs(briefsDir string, since time.Time) int {
	entries, err := os.ReadDir(briefsDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		if !since.IsZero() && !info.ModTime().After(since) {
			continue
		}

		path := filepath.Join(briefsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if hasFrontmatterField(string(data), "category", "maintenance") {
			count++
		}
	}

	return count
}

// FormatDigestSummary renders the digest summary for the thinking surface.
func FormatDigestSummary(summary *DigestSummary) string {
	if summary == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Between sessions: %d briefs cluster into %d themes",
		summary.BriefsComposed, summary.ClustersFound))
	b.WriteString("\n")

	if summary.MaintenanceCount > 0 {
		b.WriteString(fmt.Sprintf("   Also: %d maintenance completions\n", summary.MaintenanceCount))
	}
	b.WriteString("\n")

	return b.String()
}

type digestFrontmatter struct {
	date           time.Time
	briefsComposed int
	clustersFound  int
}

// parseDigestFrontmatter extracts YAML frontmatter fields from a digest file.
func parseDigestFrontmatter(content string) digestFrontmatter {
	var fm digestFrontmatter

	// Find frontmatter block between --- markers
	if !strings.HasPrefix(content, "---") {
		return fm
	}

	end := strings.Index(content[3:], "---")
	if end < 0 {
		return fm
	}

	block := content[3 : end+3]
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "date":
			fm.date, _ = time.Parse("2006-01-02", val)
		case "briefs_composed":
			fm.briefsComposed, _ = strconv.Atoi(val)
		case "clusters_found":
			fm.clustersFound, _ = strconv.Atoi(val)
		}
	}

	return fm
}

// hasFrontmatterField checks if a markdown file's YAML frontmatter contains
// a specific key-value pair.
func hasFrontmatterField(content, key, value string) bool {
	if !strings.HasPrefix(content, "---") {
		return false
	}
	end := strings.Index(content[3:], "---")
	if end < 0 {
		return false
	}
	block := content[3 : end+3]
	target := key + ": " + value
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}
