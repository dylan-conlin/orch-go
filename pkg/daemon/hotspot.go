// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"strings"
)

// HotspotWarning represents a detected hotspot that affects an issue.
type HotspotWarning struct {
	// Path is the file path or topic name that is a hotspot.
	Path string `json:"path"`
	// Type is the hotspot type: "fix-density" or "investigation-cluster".
	Type string `json:"type"`
	// Score is the hotspot score (higher = more severe).
	Score int `json:"score"`
	// Recommendation is the suggested action for this hotspot.
	Recommendation string `json:"recommendation,omitempty"`
}

// IsCritical returns true if this hotspot has a critical score (10+).
func (h *HotspotWarning) IsCritical() bool {
	return h.Score >= 10
}

// HotspotChecker is an interface for checking hotspots in a project.
type HotspotChecker interface {
	// CheckHotspots returns all current hotspots for the project.
	CheckHotspots(projectDir string) ([]HotspotWarning, error)
}

// CheckHotspotsForIssue checks if an issue touches any hotspot areas.
// Returns warnings for any matching hotspots.
func CheckHotspotsForIssue(issue *Issue, checker HotspotChecker) []HotspotWarning {
	if issue == nil || checker == nil {
		return nil
	}

	// Get all hotspots for the project
	// Note: In a real implementation, this would filter based on issue content/files
	// For now, we return all hotspots as potential warnings since the daemon
	// doesn't have detailed file information about what an issue will touch.
	hotspots, err := checker.CheckHotspots("")
	if err != nil {
		// Graceful degradation - return no warnings on error
		return nil
	}

	return hotspots
}

// FormatHotspotWarnings formats hotspot warnings for display.
func FormatHotspotWarnings(warnings []HotspotWarning) string {
	if len(warnings) == 0 {
		return ""
	}

	var sb strings.Builder

	// Header
	sb.WriteString("\n⚠️  HOTSPOT WARNING\n")
	sb.WriteString("────────────────────────────────────────\n")

	// List each warning
	for _, w := range warnings {
		// Icon based on severity
		icon := "🔸"
		if w.IsCritical() {
			icon = "🔴"
		} else if w.Score >= 7 {
			icon = "🟡"
		}

		sb.WriteString(fmt.Sprintf("%s [%d] %s (%s)\n", icon, w.Score, w.Path, w.Type))
		if w.Recommendation != "" {
			sb.WriteString(fmt.Sprintf("   └─ %s\n", w.Recommendation))
		}
	}

	sb.WriteString("────────────────────────────────────────\n")

	// Summary recommendation
	hasCritical := false
	for _, w := range warnings {
		if w.IsCritical() {
			hasCritical = true
			break
		}
	}
	sb.WriteString(GenerateHotspotRecommendation(hasCritical))

	return sb.String()
}

// GenerateHotspotRecommendation generates a recommendation based on hotspot severity.
func GenerateHotspotRecommendation(hasCritical bool) string {
	if hasCritical {
		return "CRITICAL: Consider spawning architect instead to address structural issues\n"
	}
	return "Consider review before auto-spawning to verify approach\n"
}
