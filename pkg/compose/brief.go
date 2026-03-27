// Package compose implements brief composition — clustering briefs by content
// similarity and producing digest artifacts for orchestrator review.
package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Brief represents a parsed brief with its sections extracted.
type Brief struct {
	ID         string // e.g., "orch-go-f8y50"
	FilePath   string
	Frame      string
	Resolution string
	Tension    string
	Keywords   []string // Extracted significant words
}

// ParseBrief reads a brief markdown file and extracts Frame, Resolution, and Tension sections.
func ParseBrief(path string) (*Brief, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading brief %s: %w", path, err)
	}

	content := string(data)
	b := &Brief{
		FilePath: path,
	}

	// Extract ID from the "# Brief: orch-go-XXXXX" header
	if id := extractBriefID(content); id != "" {
		b.ID = id
	} else {
		// Fall back to filename
		base := filepath.Base(path)
		b.ID = strings.TrimSuffix(base, ".md")
	}

	b.Frame = extractSection(content, "## Frame")
	b.Resolution = extractSection(content, "## Resolution")
	b.Tension = extractSection(content, "## Tension")
	b.Keywords = ExtractKeywords(b.Frame + " " + b.Resolution + " " + b.Tension)

	return b, nil
}

var briefIDRe = regexp.MustCompile(`#\s+Brief:\s+([\w-]+)`)

func extractBriefID(content string) string {
	m := briefIDRe.FindStringSubmatch(content)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// extractSection pulls the text between a heading and the next ## heading (or EOF).
func extractSection(content, heading string) string {
	idx := strings.Index(content, heading)
	if idx < 0 {
		return ""
	}

	// Skip past the heading line
	start := idx + len(heading)
	if nl := strings.Index(content[start:], "\n"); nl >= 0 {
		start += nl + 1
	}

	// Find next ## heading or end of content
	rest := content[start:]
	if nextH := strings.Index(rest, "\n## "); nextH >= 0 {
		rest = rest[:nextH]
	}

	return strings.TrimSpace(rest)
}

// LoadBriefs reads all .md files from a briefs directory.
func LoadBriefs(briefsDir string) ([]*Brief, error) {
	entries, err := os.ReadDir(briefsDir)
	if err != nil {
		return nil, fmt.Errorf("reading briefs directory %s: %w", briefsDir, err)
	}

	var briefs []*Brief
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(briefsDir, e.Name())
		b, err := ParseBrief(path)
		if err != nil {
			// Skip unparseable briefs rather than failing entirely
			continue
		}
		if b.Frame == "" && b.Resolution == "" && b.Tension == "" {
			continue // Skip empty/non-brief markdown files
		}
		briefs = append(briefs, b)
	}

	return briefs, nil
}
