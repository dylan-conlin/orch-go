package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// SpawnHotspotResult contains the result of checking hotspots for a spawn task.
type SpawnHotspotResult struct {
	HasHotspots        bool      `json:"has_hotspots"`
	HasCriticalHotspot bool      `json:"has_critical_hotspot"` // True when any matched bloat-size file >1500 lines
	MatchedHotspots    []Hotspot `json:"matched_hotspots,omitempty"`
	CriticalFiles      []string  `json:"critical_files,omitempty"` // File paths of CRITICAL hotspots (>1500 lines)
	MaxScore           int       `json:"max_score"`
	Warning            string    `json:"warning,omitempty"`
}

// extractPathsFromTask extracts file/directory paths from a task description.
// Returns a list of paths found in the task text.
func extractPathsFromTask(task string) []string {
	var paths []string

	// Pattern matches file paths like:
	// - cmd/orch/spawn.go
	// - pkg/daemon/daemon.go
	// - web/src/components/Dashboard.tsx
	// - "pkg/auth/token.go" (quoted)
	// - pkg/daemon/ (directories)
	pathPattern := regexp.MustCompile(`(?:^|[\s"'])([a-zA-Z0-9_\-./]+(?:\.[a-zA-Z0-9]+|/))(?:[\s"']|$)`)

	matches := pathPattern.FindAllStringSubmatch(task, -1)
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			path := strings.Trim(match[1], `"'`)
			// Validate it looks like a real path (has at least one directory separator or extension)
			if (strings.Contains(path, "/") || strings.Contains(path, ".")) && !seen[path] {
				// Filter out common non-path patterns
				if !isLikelyNotAPath(path) {
					paths = append(paths, path)
					seen[path] = true
				}
			}
		}
	}

	return paths
}

// isLikelyNotAPath returns true if the string is unlikely to be a file path.
func isLikelyNotAPath(s string) bool {
	// URLs
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return true
	}
	// Very short paths are probably not real
	if len(s) < 4 {
		return true
	}
	// Common words that might match the pattern but aren't paths
	nonPaths := []string{"e.g.", "i.e.", "etc.", "vs.", "no."}
	for _, np := range nonPaths {
		if s == np {
			return true
		}
	}
	return false
}

// pathOrSuffixMatch returns true if path matches target via exact match,
// suffix match (e.g., "complete_cmd.go" matches "cmd/orch/complete_cmd.go"),
// or directory containment.
func pathOrSuffixMatch(path, target string) bool {
	if path == target {
		return true
	}
	// Suffix match: "complete_cmd.go" matches "cmd/orch/complete_cmd.go"
	// Also handles partial paths: "orch/complete_cmd.go" matches "cmd/orch/complete_cmd.go"
	if strings.HasSuffix(target, "/"+path) {
		return true
	}
	// Directory containment: "cmd/orch/" contains "cmd/orch/complete_cmd.go"
	if strings.HasSuffix(path, "/") && strings.HasPrefix(target, path) {
		return true
	}
	// Reverse: hotspot is a directory containing the path
	if strings.HasSuffix(target, "/") && strings.HasPrefix(path, target) {
		return true
	}
	return false
}

// matchPathToHotspots checks if a path matches any hotspot.
// Returns true and the highest matching score if a match is found.
func matchPathToHotspots(path string, hotspots []Hotspot) (bool, int) {
	maxScore := 0
	matched := false

	for _, h := range hotspots {
		switch h.Type {
		case "fix-density":
			// For fix-density, check for exact, suffix, or directory containment match
			if pathOrSuffixMatch(path, h.Path) {
				matched = true
			}
			if matched && h.Score > maxScore {
				maxScore = h.Score
			}
		case "investigation-cluster":
			// For investigation clusters, check if the topic appears in the path
			if strings.Contains(strings.ToLower(path), strings.ToLower(h.Path)) {
				matched = true
				if h.Score > maxScore {
					maxScore = h.Score
				}
			}
		case "bloat-size":
			// For bloat-size, check for exact, suffix, or directory containment match
			if pathOrSuffixMatch(path, h.Path) {
				matched = true
			}
			if matched && h.Score > maxScore {
				maxScore = h.Score
			}
		case "coupling-cluster":
			// For coupling clusters, check if the concept appears in the path
			// or if the path matches any of the related files
			if strings.Contains(strings.ToLower(path), strings.ToLower(h.Path)) {
				matched = true
			}
			// Also check related files for exact, suffix, or directory matches
			for _, rf := range h.RelatedFiles {
				if pathOrSuffixMatch(path, rf) {
					matched = true
					break
				}
			}
			if matched && h.Score > maxScore {
				maxScore = h.Score
			}
		}
	}

	return matched, maxScore
}

// checkSpawnHotspots checks if a task description references any hotspot areas.
// Returns a SpawnHotspotResult with details about matched hotspots.
func checkSpawnHotspots(task string, hotspots []Hotspot) *SpawnHotspotResult {
	result := &SpawnHotspotResult{}

	// Extract paths from task
	paths := extractPathsFromTask(task)

	// Check each hotspot against extracted file paths only.
	// We do NOT match investigation-cluster or coupling-cluster topic keywords
	// against the raw task text — this causes false positives when tasks merely
	// cite code evidence (e.g., "daemon.go", "model", "gate") without targeting
	// those files for modification.
	for _, h := range hotspots {
		matched := false

		// Check extracted paths against this hotspot
		for _, path := range paths {
			if pathMatches, _ := matchPathToHotspots(path, []Hotspot{h}); pathMatches {
				matched = true
				break
			}
		}

		if matched {
			result.HasHotspots = true
			result.MatchedHotspots = append(result.MatchedHotspots, h)
			if h.Score > result.MaxScore {
				result.MaxScore = h.Score
			}
			// Track CRITICAL hotspots: bloat-size files >1500 lines
			if h.Type == "bloat-size" && h.Score > 1500 {
				result.HasCriticalHotspot = true
				result.CriticalFiles = append(result.CriticalFiles, h.Path)
			}
		}
	}

	if result.HasHotspots {
		result.Warning = formatHotspotWarning(result)
	}

	return result
}

// formatHotspotWarning formats a warning message for hotspot matches.
func formatHotspotWarning(result *SpawnHotspotResult) string {
	if !result.HasHotspots || len(result.MatchedHotspots) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("┌─────────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  🔥 HOTSPOT WARNING: Task targets high-churn area                          │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────────────────────┤\n")

	for _, h := range result.MatchedHotspots {
		typeIcon := "🔧"
		if h.Type == "investigation-cluster" {
			typeIcon = "📚"
		} else if h.Type == "bloat-size" {
			typeIcon = "📏"
		} else if h.Type == "coupling-cluster" {
			typeIcon = "🔗"
		}
		line := fmt.Sprintf("│  %s [%d] %s", typeIcon, h.Score, h.Path)
		// Pad to box width
		if len(line) < 78 {
			line += strings.Repeat(" ", 78-len(line))
		}
		sb.WriteString(line + "│\n")
	}

	// Add defect class information if available
	var matchedFiles []string
	for _, h := range result.MatchedHotspots {
		matchedFiles = append(matchedFiles, h.Path)
	}
	defectClasses := DefectClassesForHotspots(matchedFiles)
	if len(defectClasses) > 0 {
		sb.WriteString("├─────────────────────────────────────────────────────────────────────────────┤\n")
		sb.WriteString("│  🎯 LIKELY DEFECT CLASSES:                                                │\n")
		for _, class := range defectClasses {
			line := fmt.Sprintf("│     %s", class)
			if len(line) < 78 {
				line += strings.Repeat(" ", 78-len(line))
			}
			sb.WriteString(line + "│\n")
		}
	}
	sb.WriteString("├─────────────────────────────────────────────────────────────────────────────┤\n")
	sb.WriteString("│  💡 RECOMMENDATION: Consider spawning architect first to review design     │\n")
	sb.WriteString("│     orch spawn architect \"Review design for [area]\"                        │\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}

// defectClassMapping maps file path keywords to likely defect classes.
// Based on the defect class taxonomy in .kb/models/defect-class-taxonomy/model.md.
// Key = keyword that appears in hotspot file/topic path.
// Value = defect class names likely to manifest in that area.
var defectClassMapping = map[string][]string{
	"spawn":     {"Class 2: Multi-Backend Blindness", "Class 4: Cross-Project Boundary Bleed", "Class 6: Duplicate Action"},
	"daemon":    {"Class 2: Multi-Backend Blindness", "Class 3: Stale Artifact Accumulation", "Class 6: Duplicate Action"},
	"complete":  {"Class 1: Filter Amnesia", "Class 5: Contradictory Authority Signals", "Class 7: Premature Destruction"},
	"verify":    {"Class 1: Filter Amnesia", "Class 5: Contradictory Authority Signals"},
	"status":    {"Class 2: Multi-Backend Blindness", "Class 5: Contradictory Authority Signals"},
	"hotspot":   {"Class 0: Scope Expansion", "Class 1: Filter Amnesia"},
	"serve":     {"Class 1: Filter Amnesia", "Class 4: Cross-Project Boundary Bleed"},
	"clean":     {"Class 3: Stale Artifact Accumulation", "Class 7: Premature Destruction"},
	"tmux":      {"Class 2: Multi-Backend Blindness", "Class 7: Premature Destruction"},
	"workspace": {"Class 3: Stale Artifact Accumulation", "Class 4: Cross-Project Boundary Bleed"},
	"session":   {"Class 2: Multi-Backend Blindness", "Class 3: Stale Artifact Accumulation"},
	"account":   {"Class 4: Cross-Project Boundary Bleed"},
}

// DefectClassesForHotspots returns the unique defect class names likely to manifest
// for the given hotspot file/topic paths. Matches keywords in the path against
// the defect class mapping table.
func DefectClassesForHotspots(files []string) []string {
	seen := make(map[string]bool)
	var classes []string

	for _, file := range files {
		fileLower := strings.ToLower(file)
		for keyword, classList := range defectClassMapping {
			if strings.Contains(fileLower, keyword) {
				for _, class := range classList {
					if !seen[class] {
						seen[class] = true
						classes = append(classes, class)
					}
				}
			}
		}
	}

	// Sort for deterministic output (by class number)
	sort.Strings(classes)
	return classes
}

// RunHotspotCheckForSpawn runs hotspot analysis and checks task against results.
// This is the main entry point for spawn integration.
// Returns nil if no hotspots detected, otherwise returns the result with warning.
func RunHotspotCheckForSpawn(projectDir, task string) (*SpawnHotspotResult, error) {
	// Run hotspot analysis (reuse existing logic)
	report := HotspotReport{
		GeneratedAt:    time.Now().Format(time.RFC3339),
		AnalysisPeriod: fmt.Sprintf("Last %d days", 28), // Default to 28 days
		FixThreshold:   5,                               // Default threshold
		InvThreshold:   3,                               // Default threshold
		BloatThreshold: 800,                             // Default bloat threshold
		Hotspots:       []Hotspot{},
	}

	// Analyze git history for fix commit density
	fixHotspots, _, err := analyzeFixCommits(projectDir, 28, 5)
	if err == nil {
		report.Hotspots = append(report.Hotspots, fixHotspots...)
	}

	// Analyze investigation clusters (silent failure if kb not available)
	invHotspots, _, _ := analyzeInvestigationClusters(projectDir, 3)
	report.Hotspots = append(report.Hotspots, invHotspots...)

	// Analyze coupling clusters
	couplingHotspots, _, _ := analyzeCouplingClusters(projectDir, 28)
	report.Hotspots = append(report.Hotspots, couplingHotspots...)

	// Analyze file sizes for bloat detection (CRITICAL files >1500 lines trigger spawn blocking)
	bloatHotspots, _, _ := analyzeBloatFiles(projectDir, 800)
	report.Hotspots = append(report.Hotspots, bloatHotspots...)

	if len(report.Hotspots) == 0 {
		return nil, nil
	}

	// Check task against hotspots
	result := checkSpawnHotspots(task, report.Hotspots)

	if !result.HasHotspots {
		return nil, nil
	}

	return result, nil
}
