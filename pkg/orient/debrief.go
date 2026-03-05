package orient

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxDebriefItems    = 3
	truncateDebriefLen = 120
)

// DebriefSummary holds extracted data from a session debrief file.
type DebriefSummary struct {
	Date         string   `json:"date"`
	WhatHappened []string `json:"what_happened,omitempty"`
	WhatWeLearned []string `json:"what_we_learned,omitempty"`
	InFlight     []string `json:"in_flight,omitempty"`
	WhatsNext    []string `json:"whats_next,omitempty"`
}

// FindLatestDebrief returns the path to the most recent debrief file
// in the given sessions directory. Files must match *-debrief.md pattern.
func FindLatestDebrief(sessionsDir string) (string, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return "", fmt.Errorf("reading sessions dir: %w", err)
	}

	var debriefFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), "-debrief.md") {
			debriefFiles = append(debriefFiles, e.Name())
		}
	}

	if len(debriefFiles) == 0 {
		return "", fmt.Errorf("no debrief files found in %s", sessionsDir)
	}

	// Sort lexicographically — YYYY-MM-DD prefix ensures chronological order
	sort.Strings(debriefFiles)

	return filepath.Join(sessionsDir, debriefFiles[len(debriefFiles)-1]), nil
}

// ParseDebriefSummary parses a debrief markdown file into a DebriefSummary.
// Returns nil if content is empty.
func ParseDebriefSummary(content string) *DebriefSummary {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	summary := &DebriefSummary{}

	// Extract date from header: "# Session Debrief: YYYY-MM-DD"
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# Session Debrief: ") {
			summary.Date = strings.TrimPrefix(line, "# Session Debrief: ")
			break
		}
	}

	// Extract sections
	summary.WhatHappened = extractSection(content, "## What Happened", maxDebriefItems)
	summary.WhatWeLearned = extractSection(content, "## What We Learned", maxDebriefItems)
	summary.InFlight = extractSection(content, "## What's In Flight", maxDebriefItems)
	summary.WhatsNext = extractSection(content, "## What's Next", maxDebriefItems)

	return summary
}

// extractSection finds a markdown section by header and extracts list items from it.
func extractSection(content, header string, maxItems int) []string {
	idx := strings.Index(content, header)
	if idx < 0 {
		return nil
	}

	// Start after the header line
	rest := content[idx+len(header):]
	if newline := strings.Index(rest, "\n"); newline >= 0 {
		rest = rest[newline+1:]
	}

	var items []string
	for _, line := range strings.Split(rest, "\n") {
		line = strings.TrimSpace(line)

		// Stop at next section header
		if strings.HasPrefix(line, "## ") {
			break
		}

		if line == "" {
			continue
		}

		// Parse bullet items: "- item"
		if strings.HasPrefix(line, "- ") {
			item := strings.TrimPrefix(line, "- ")
			if item == "(none)" {
				continue
			}
			if len(items) < maxItems {
				items = append(items, truncateSummary(item, truncateDebriefLen))
			}
			continue
		}

		// Parse numbered items: "1. item"
		if len(line) >= 3 && line[0] >= '0' && line[0] <= '9' && strings.Contains(line[:3], ".") {
			dotIdx := strings.Index(line, ". ")
			if dotIdx >= 0 {
				item := strings.TrimSpace(line[dotIdx+2:])
				if item == "(none)" {
					continue
				}
				if len(items) < maxItems {
					items = append(items, truncateSummary(item, truncateDebriefLen))
				}
			}
			continue
		}
	}

	return items
}

// FormatPreviousSession renders a DebriefSummary as the "Previous session:" section
// for orient output. Returns empty string if summary is nil or has no content.
func FormatPreviousSession(summary *DebriefSummary) string {
	if summary == nil {
		return ""
	}

	// Check if there's any content worth showing
	if len(summary.WhatHappened) == 0 && len(summary.WhatWeLearned) == 0 &&
		len(summary.InFlight) == 0 && len(summary.WhatsNext) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("Previous session (%s):\n", summary.Date))

	if len(summary.WhatHappened) > 0 {
		b.WriteString(fmt.Sprintf("   Happened: %s\n", strings.Join(summary.WhatHappened, "; ")))
	}
	if len(summary.WhatWeLearned) > 0 {
		b.WriteString(fmt.Sprintf("   Learned: %s\n", strings.Join(summary.WhatWeLearned, "; ")))
	}
	if len(summary.InFlight) > 0 {
		b.WriteString(fmt.Sprintf("   In flight: %s\n", strings.Join(summary.InFlight, "; ")))
	}
	if len(summary.WhatsNext) > 0 {
		b.WriteString(fmt.Sprintf("   Next: %s\n", strings.Join(summary.WhatsNext, "; ")))
	}

	b.WriteString("\n")
	return b.String()
}
