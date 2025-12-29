// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// UIDetectionResult contains the result of UI task detection.
type UIDetectionResult struct {
	IsUITask      bool     // Whether UI task was detected
	Reasons       []string // Why UI task was detected
	Confidence    string   // "high", "medium", "low"
	ShouldAutoMCP bool     // Whether to auto-add MCP
}

// uiPathPatterns are directory/file patterns that indicate UI work.
var uiPathPatterns = []string{
	"web/",
	"frontend/",
	"src/routes/",
	"src/components/",
	"pages/",
	"app/",     // Next.js app router
	"views/",   // Traditional MVC views
	"layouts/", // Layout components
}

// uiFileExtensions are file extensions that indicate UI work.
var uiFileExtensions = []string{
	".svelte",
	".tsx",
	".jsx",
	".vue",
	".astro",
	".html",
	".css",
	".scss",
	".sass",
}

// uiKeywordsHigh are keywords that strongly indicate UI work.
var uiKeywordsHigh = []string{
	"ui",
	"component",
	"visual",
	"browser",
	"frontend",
	"dashboard",
	"page",
	"view",
	"layout",
	"svelte",
	"react",
	"vue",
}

// uiKeywordsMedium are keywords that moderately indicate UI work.
var uiKeywordsMedium = []string{
	"button",
	"form",
	"modal",
	"dialog",
	"menu",
	"navigation",
	"sidebar",
	"header",
	"footer",
	"style",
	"css",
	"theme",
	"responsive",
	"animation",
	"hover",
	"click",
	"scroll",
	"drag",
	"drop",
	"tooltip",
	"popup",
	"dropdown",
	"tab",
	"card",
	"list",
	"table",
	"grid",
	"chart",
	"graph",
	"icon",
	"image",
	"avatar",
	"badge",
	"toast",
	"notification",
	"alert",
	"progress",
	"spinner",
	"loading",
	"skeleton",
}

// DetectUITask analyzes task description and project directory to detect UI work.
// Returns detection result with confidence level and reasons.
func DetectUITask(task string, projectDir string) *UIDetectionResult {
	result := &UIDetectionResult{
		Reasons: []string{},
	}

	taskLower := strings.ToLower(task)
	var highConfidenceMatches int
	var mediumConfidenceMatches int

	// Check for high-confidence keywords in task
	for _, keyword := range uiKeywordsHigh {
		// Use word boundary matching to avoid false positives
		pattern := `\b` + regexp.QuoteMeta(keyword) + `\b`
		if matched, _ := regexp.MatchString(pattern, taskLower); matched {
			result.Reasons = append(result.Reasons, "task mentions '"+keyword+"'")
			highConfidenceMatches++
		}
	}

	// Check for medium-confidence keywords in task
	for _, keyword := range uiKeywordsMedium {
		pattern := `\b` + regexp.QuoteMeta(keyword) + `\b`
		if matched, _ := regexp.MatchString(pattern, taskLower); matched {
			result.Reasons = append(result.Reasons, "task mentions '"+keyword+"'")
			mediumConfidenceMatches++
		}
	}

	// Check for file extensions mentioned in task
	for _, ext := range uiFileExtensions {
		if strings.Contains(taskLower, ext) {
			result.Reasons = append(result.Reasons, "task references "+ext+" files")
			highConfidenceMatches++
		}
	}

	// Check for path patterns mentioned in task
	for _, pathPattern := range uiPathPatterns {
		if strings.Contains(taskLower, strings.TrimSuffix(pathPattern, "/")) {
			result.Reasons = append(result.Reasons, "task references "+pathPattern+" directory")
			highConfidenceMatches++
		}
	}

	// Check if project has UI directories
	if projectDir != "" {
		for _, pathPattern := range uiPathPatterns {
			checkPath := filepath.Join(projectDir, pathPattern)
			if info, err := os.Stat(checkPath); err == nil && info.IsDir() {
				result.Reasons = append(result.Reasons, "project has "+pathPattern+" directory")
				// Only count as medium confidence - project having UI doesn't mean task is UI
				mediumConfidenceMatches++
				break // Only count once
			}
		}
	}

	// Determine confidence level and whether to auto-add MCP
	switch {
	case highConfidenceMatches >= 2:
		result.Confidence = "high"
		result.IsUITask = true
		result.ShouldAutoMCP = true
	case highConfidenceMatches >= 1:
		result.Confidence = "high"
		result.IsUITask = true
		result.ShouldAutoMCP = true
	case mediumConfidenceMatches >= 3:
		result.Confidence = "medium"
		result.IsUITask = true
		result.ShouldAutoMCP = true
	case mediumConfidenceMatches >= 2:
		result.Confidence = "medium"
		result.IsUITask = true
		result.ShouldAutoMCP = true
	case mediumConfidenceMatches >= 1 && highConfidenceMatches >= 0:
		result.Confidence = "low"
		result.IsUITask = true
		result.ShouldAutoMCP = false // Don't auto-add for low confidence
	default:
		result.Confidence = "none"
		result.IsUITask = false
		result.ShouldAutoMCP = false
	}

	return result
}

// FormatUIDetectionMessage formats a message for when UI detection auto-adds MCP.
func FormatUIDetectionMessage(result *UIDetectionResult) string {
	if !result.ShouldAutoMCP || len(result.Reasons) == 0 {
		return ""
	}

	var msg strings.Builder
	msg.WriteString("🎨 UI task detected - auto-adding --mcp playwright\n")
	msg.WriteString("   Reasons: ")

	// Show up to 3 reasons
	maxReasons := 3
	if len(result.Reasons) < maxReasons {
		maxReasons = len(result.Reasons)
	}
	reasons := result.Reasons[:maxReasons]
	msg.WriteString(strings.Join(reasons, ", "))

	if len(result.Reasons) > 3 {
		msg.WriteString(" (and more)")
	}
	msg.WriteString("\n")
	msg.WriteString("   Use --no-mcp to disable auto-detection\n")

	return msg.String()
}
