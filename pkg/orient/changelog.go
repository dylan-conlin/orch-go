package orient

import (
	"fmt"
	"strings"
)

// ChangelogEntry represents a single git commit in the changelog.
type ChangelogEntry struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
}

// ParseGitLog parses output from `git log --format="%h|%s"` into ChangelogEntry slice.
// Limits results to maxCount entries.
func ParseGitLog(output string, maxCount int) []ChangelogEntry {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil
	}

	var entries []ChangelogEntry
	for _, line := range strings.Split(output, "\n") {
		if len(entries) >= maxCount {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, "|")
		if idx < 0 {
			continue
		}
		entries = append(entries, ChangelogEntry{
			Hash:    line[:idx],
			Subject: line[idx+1:],
		})
	}
	return entries
}

// FormatChangelog renders changelog entries as a section for orient output.
// Returns empty string if entries is nil or empty.
func FormatChangelog(entries []ChangelogEntry, sinceDate string) string {
	if len(entries) == 0 {
		return ""
	}

	var b strings.Builder
	if sinceDate != "" {
		b.WriteString(fmt.Sprintf("Changelog (since %s):\n", sinceDate))
	} else {
		b.WriteString("Changelog (recent):\n")
	}
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("   %s %s\n", e.Hash, e.Subject))
	}
	b.WriteString("\n")
	return b.String()
}
