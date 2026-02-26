package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// HotspotAdvisoryMatch pairs a modified file with its matching hotspot.
type HotspotAdvisoryMatch struct {
	FilePath string  // File the agent modified
	Hotspot  Hotspot // Matching hotspot
}

// matchModifiedFilesToHotspots cross-references a list of modified files against
// the hotspot report, returning matches. Each file-hotspot pair appears at most once.
func matchModifiedFilesToHotspots(modifiedFiles []string, hotspots []Hotspot) []HotspotAdvisoryMatch {
	var matches []HotspotAdvisoryMatch
	seen := make(map[string]bool) // "file|hotspot.Path|hotspot.Type" dedup key

	for _, file := range modifiedFiles {
		for _, h := range hotspots {
			matched := false

			switch h.Type {
			case "fix-density", "bloat-size":
				matched = pathOrSuffixMatch(file, h.Path)
			case "investigation-cluster":
				// Topic match: check if the hotspot topic appears in the file path
				matched = strings.Contains(strings.ToLower(file), strings.ToLower(h.Path))
			case "coupling-cluster":
				// Check topic match in file path
				if strings.Contains(strings.ToLower(file), strings.ToLower(h.Path)) {
					matched = true
				}
				// Also check related files
				if !matched {
					for _, rf := range h.RelatedFiles {
						if pathOrSuffixMatch(file, rf) {
							matched = true
							break
						}
					}
				}
			}

			if matched {
				key := fmt.Sprintf("%s|%s|%s", file, h.Path, h.Type)
				if !seen[key] {
					seen[key] = true
					matches = append(matches, HotspotAdvisoryMatch{
						FilePath: file,
						Hotspot:  h,
					})
				}
			}
		}
	}

	return matches
}

// formatHotspotAdvisory formats matched hotspots as a readable advisory block.
// Returns empty string if no matches.
func formatHotspotAdvisory(matches []HotspotAdvisoryMatch) string {
	if len(matches) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("┌─────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  HOTSPOT ADVISORY: Agent modified files in hotspot areas    │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")

	for _, m := range matches {
		typeIcon := hotspotTypeIcon(m.Hotspot.Type)
		displayPath := m.FilePath
		if len(displayPath) > 40 {
			displayPath = "..." + displayPath[len(displayPath)-37:]
		}
		line := fmt.Sprintf("│  %s %-40s [%s:%d]", typeIcon, displayPath, m.Hotspot.Type, m.Hotspot.Score)
		// Pad to box width
		for len(line) < 62 {
			line += " "
		}
		sb.WriteString(line + "│\n")
	}

	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
	sb.WriteString("│  Run `orch hotspot` for full analysis                       │\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}

// hotspotTypeIcon returns an icon for the hotspot type.
func hotspotTypeIcon(hotspotType string) string {
	switch hotspotType {
	case "fix-density":
		return "FIX"
	case "investigation-cluster":
		return "INV"
	case "bloat-size":
		return "BIG"
	case "coupling-cluster":
		return "CPL"
	default:
		return "???"
	}
}

// getModifiedFilesFromRecentCommits returns a list of files modified in the last N commits.
func getModifiedFilesFromRecentCommits(projectDir string, commitCount int) ([]string, error) {
	ref := fmt.Sprintf("HEAD~%d..HEAD", commitCount)
	cmd := exec.Command("git", "diff", "--name-only", ref)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Fewer commits than requested - try last commit
		cmd = exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git diff failed: %w", err)
		}
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// RunHotspotAdvisoryForCompletion checks if the agent modified files in hotspot areas.
// Runs the full hotspot analysis and cross-references against recently modified files.
// Returns formatted advisory text or empty string if no matches.
//
// This is informational only - it does not block completion.
func RunHotspotAdvisoryForCompletion(projectDir string) string {
	if projectDir == "" {
		return ""
	}

	// Get files modified in recent commits (agent's work)
	modifiedFiles, err := getModifiedFilesFromRecentCommits(projectDir, 5)
	if err != nil || len(modifiedFiles) == 0 {
		return ""
	}

	// Run hotspot analysis (reuses existing hotspot infrastructure)
	var allHotspots []Hotspot

	// Fix-density hotspots
	fixHotspots, _, err := analyzeFixCommits(projectDir, 28, 5)
	if err == nil {
		allHotspots = append(allHotspots, fixHotspots...)
	}

	// Investigation clusters
	invHotspots, _, _ := analyzeInvestigationClusters(projectDir, 3)
	allHotspots = append(allHotspots, invHotspots...)

	// Coupling clusters
	couplingHotspots, _, _ := analyzeCouplingClusters(projectDir, 28)
	allHotspots = append(allHotspots, couplingHotspots...)

	// Bloat-size hotspots
	bloatHotspots, _, _ := analyzeBloatFiles(projectDir, 800)
	allHotspots = append(allHotspots, bloatHotspots...)

	if len(allHotspots) == 0 {
		return ""
	}

	// Cross-reference modified files against hotspots
	matches := matchModifiedFilesToHotspots(modifiedFiles, allHotspots)

	return formatHotspotAdvisory(matches)
}
