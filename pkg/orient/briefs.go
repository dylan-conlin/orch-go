package orient

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// RecentBrief represents a brief surfaced on the thinking surface.
type RecentBrief struct {
	BeadsID    string `json:"beads_id"`
	Title      string `json:"title"`
	HasTension bool   `json:"has_tension"`
	IsUnread   bool   `json:"is_unread"`
}

var briefIDRe = regexp.MustCompile(`#\s+Brief:\s+([\w-]+)`)

// ScanRecentBriefs reads .kb/briefs/ and returns the most recent briefs with read state.
// readState maps beadsID → true for briefs that have been read.
// Returns the brief list (limited to maxCount) and total unread count.
func ScanRecentBriefs(briefsDir string, readState map[string]bool, maxCount int) ([]RecentBrief, int) {
	entries, err := os.ReadDir(briefsDir)
	if err != nil {
		return nil, 0
	}

	// Collect file info for sorting by modification time (newest first)
	type briefFile struct {
		path    string
		modTime int64
	}
	var files []briefFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, briefFile{
			path:    filepath.Join(briefsDir, e.Name()),
			modTime: info.ModTime().Unix(),
		})
	}

	// Sort newest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	var briefs []RecentBrief
	totalUnread := 0

	for _, f := range files {
		b, err := parseBriefForOrient(f.path)
		if err != nil || b == nil {
			continue
		}

		// Check read state
		if readState != nil && readState[b.BeadsID] {
			b.IsUnread = false
		} else {
			b.IsUnread = true
			totalUnread++
		}

		briefs = append(briefs, *b)
	}

	// Count all unread before truncating
	if len(briefs) > maxCount {
		// Recount unread from all briefs
		briefs = briefs[:maxCount]
	}

	return briefs, totalUnread
}

// parseBriefForOrient extracts minimal metadata from a brief file.
func parseBriefForOrient(path string) (*RecentBrief, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	// Extract ID
	beadsID := ""
	if m := briefIDRe.FindStringSubmatch(content); len(m) >= 2 {
		beadsID = m[1]
	} else {
		base := filepath.Base(path)
		beadsID = strings.TrimSuffix(base, ".md")
	}

	// Extract Frame section for title
	frame := extractBriefSection(content, "## Frame")
	if frame == "" {
		return nil, nil // Skip briefs without a Frame
	}

	// Title is first sentence of Frame
	title := extractFirstSentence(frame)
	if title == "" {
		title = truncateSummary(frame, 80)
	}

	// Check for tension
	tension := extractBriefSection(content, "## Tension")
	hasTension := strings.TrimSpace(tension) != ""

	return &RecentBrief{
		BeadsID:    beadsID,
		Title:      title,
		HasTension: hasTension,
	}, nil
}

// extractBriefSection pulls text between a heading and the next ## heading or EOF.
func extractBriefSection(content, heading string) string {
	idx := strings.Index(content, heading)
	if idx < 0 {
		return ""
	}

	start := idx + len(heading)
	if nl := strings.Index(content[start:], "\n"); nl >= 0 {
		start += nl + 1
	}

	rest := content[start:]
	if nextH := strings.Index(rest, "\n## "); nextH >= 0 {
		rest = rest[:nextH]
	}

	return strings.TrimSpace(rest)
}

// extractFirstSentence returns the first sentence (ending in . or newline) from text.
func extractFirstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// Find first period followed by space or end of string
	for i := 0; i < len(text)-1; i++ {
		if text[i] == '.' && (i+1 >= len(text) || text[i+1] == ' ' || text[i+1] == '\n') {
			return text[:i+1]
		}
	}

	// No period found — use first line
	if nl := strings.Index(text, "\n"); nl >= 0 {
		return text[:nl]
	}

	return truncateSummary(text, 100)
}

// FormatRecentBriefs renders the briefs section for the thinking surface.
func FormatRecentBriefs(briefs []RecentBrief, unreadCount int) string {
	if len(briefs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Recent briefs (%d unread):\n", unreadCount))
	for _, brief := range briefs {
		marker := " "
		if brief.IsUnread {
			marker = "*"
		}
		tensionMark := ""
		if brief.HasTension {
			tensionMark = " [tension]"
		}
		b.WriteString(fmt.Sprintf("  %s %s%s (%s)\n", marker, brief.Title, tensionMark, brief.BeadsID))
	}
	b.WriteString("\n")
	return b.String()
}
