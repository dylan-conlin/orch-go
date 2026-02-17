// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// InferTargetFilesFromIssue extracts file paths from issue title and description using keyword matching.
// Returns a list of potential target files mentioned in the issue.
func InferTargetFilesFromIssue(issue *Issue) []string {
	if issue == nil {
		return nil
	}

	// Combine title and description for searching
	text := issue.Title + " " + issue.Description

	var files []string
	seen := make(map[string]bool)

	// Pattern 1: Explicit file paths (e.g., pkg/spawn/spawn.go, cmd/orch/spawn_cmd.go)
	// Matches: word characters, forward slashes, dots, underscores, hyphens
	// Examples: pkg/daemon/daemon.go, cmd/orch/spawn_cmd.go, web/components/Header.tsx
	filePathRegex := regexp.MustCompile(`\b([a-zA-Z0-9_-]+/[a-zA-Z0-9_/-]+\.[a-zA-Z0-9]+)\b`)
	matches := filePathRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			file := match[1]
			if !seen[file] {
				files = append(files, file)
				seen[file] = true
			}
		}
	}

	// Pattern 2: Bare filename mentions (e.g., "spawn_cmd.go" without a path)
	// Only matches words ending in .go that weren't already captured by Pattern 1
	for _, word := range strings.Fields(strings.ToLower(text)) {
		if strings.HasSuffix(word, ".go") && !strings.Contains(word, "/") {
			if !seen[word] {
				files = append(files, word)
				seen[word] = true
			}
		}
	}

	return files
}

// FindCriticalHotspot checks if any of the inferred files matches a CRITICAL hotspot (>1500 lines).
// Returns the first matching critical hotspot, or nil if none found.
func FindCriticalHotspot(inferredFiles []string, hotspots []HotspotWarning) *HotspotWarning {
	if len(inferredFiles) == 0 || len(hotspots) == 0 {
		return nil
	}

	// Check each inferred file against hotspots
	for _, file := range inferredFiles {
		for _, h := range hotspots {
			// Only consider bloat-size hotspots (file size)
			if h.Type != "bloat-size" {
				continue
			}

			// Check if hotspot is CRITICAL (>1500 lines)
			if h.Score <= 1500 {
				continue
			}

			// Check if inferred file matches hotspot path
			if matchesFilePath(file, h.Path) {
				// Return a copy of the hotspot
				critical := h
				return &critical
			}
		}
	}

	return nil
}

// matchesFilePath checks if an inferred file matches a hotspot path.
// Handles partial matches (e.g., "spawn_cmd.go" matches "cmd/orch/spawn_cmd.go").
func matchesFilePath(inferredFile, hotspotPath string) bool {
	// Normalize paths
	inferred := strings.TrimSpace(strings.ToLower(inferredFile))
	hotspot := strings.TrimSpace(strings.ToLower(hotspotPath))

	// Exact match
	if inferred == hotspot {
		return true
	}

	// Check if hotspot path ends with inferred filename
	// Example: "spawn_cmd.go" matches "cmd/orch/spawn_cmd.go"
	if strings.HasSuffix(hotspot, inferred) {
		return true
	}

	// Check if hotspot path contains inferred as basename
	// Example: "spawn_cmd.go" matches "cmd/orch/spawn_cmd.go"
	hotspotBase := hotspot[strings.LastIndex(hotspot, "/")+1:]
	if hotspotBase == inferred {
		return true
	}

	return false
}

// GenerateExtractionTask creates a task description for the extraction agent.
// Format: "Extract [inferred concern] from [file] into [pkg/appropriate/package/]. Pure structural extraction — no behavior changes."
func GenerateExtractionTask(issue *Issue, criticalFile string) string {
	// Infer concern from issue title/type
	concern := inferConcernFromIssue(issue)

	// Infer target package from file path
	targetPkg := inferTargetPackage(criticalFile)

	return fmt.Sprintf(
		"Extract %s from %s into %s. Pure structural extraction — no behavior changes.",
		concern,
		criticalFile,
		targetPkg,
	)
}

// inferConcernFromIssue attempts to extract the main concern from the issue.
// Falls back to generic "related functionality" if unclear.
func inferConcernFromIssue(issue *Issue) string {
	if issue == nil {
		return "related functionality"
	}

	// Try to extract verb + noun pattern from title
	// Examples: "Add hotspot detection" → "hotspot detection"
	//           "Fix daemon spawn" → "daemon spawn logic"
	title := strings.ToLower(issue.Title)

	// Remove common action verbs from the start
	actionVerbs := []string{"add ", "fix ", "implement ", "update ", "refactor ", "extract ", "move "}
	for _, verb := range actionVerbs {
		if strings.HasPrefix(title, verb) {
			concern := strings.TrimSpace(title[len(verb):])
			if concern != "" {
				return concern
			}
		}
	}

	// If no verb pattern, use first 3-5 words of title
	words := strings.Fields(title)
	if len(words) <= 5 {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:5], " ")
}

// inferTargetPackage suggests an appropriate target package based on the file path.
// Returns a generic package path that can be refined by the extraction agent.
func inferTargetPackage(filePath string) string {
	// Get directory from file path
	lastSlash := strings.LastIndex(filePath, "/")
	if lastSlash == -1 {
		return "pkg/appropriate/package/"
	}

	dir := filePath[:lastSlash]

	// Suggest a parallel package structure
	// Example: "cmd/orch/spawn_cmd.go" → "pkg/spawn/"
	//          "pkg/daemon/daemon.go" → "pkg/daemon/extracted/"
	if strings.HasPrefix(dir, "cmd/") {
		// Extract the command name and suggest pkg/ equivalent
		parts := strings.Split(dir, "/")
		if len(parts) >= 2 {
			return "pkg/" + parts[1] + "/"
		}
	}

	if strings.HasPrefix(dir, "pkg/") {
		// Suggest a sub-package within the same package
		return dir + "/extracted/"
	}

	// Default fallback
	return "pkg/appropriate/package/"
}

// ExtractionResult contains the result of checking if extraction is needed before feature work.
type ExtractionResult struct {
	// Needed indicates that extraction should happen before the feature agent.
	Needed bool
	// CriticalFile is the hotspot file path that triggered extraction.
	CriticalFile string
	// ExtractionTask is the generated task description for the extraction agent.
	ExtractionTask string
	// Hotspot is the matched critical hotspot warning.
	Hotspot *HotspotWarning
}

// CheckExtractionNeeded determines if an issue targets a CRITICAL hotspot file (>1500 lines)
// that requires extraction before feature work can begin.
// Returns nil if no extraction is needed (no target files inferred, no hotspots, or no critical match).
func CheckExtractionNeeded(issue *Issue, checker HotspotChecker) *ExtractionResult {
	if issue == nil || checker == nil {
		return nil
	}

	// Skip extraction checks for extraction issues themselves (prevents recursion).
	// Extraction issues have titles like: "Extract X from file.go into pkg/..."
	// Without this guard, InferTargetFilesFromIssue() would parse the file path from
	// the extraction issue's title, triggering another extraction and creating
	// cascading chains of duplicate extraction issues.
	if strings.HasPrefix(issue.Title, "Extract ") {
		return nil
	}

	// Step 1: Infer target files from issue title/description
	files := InferTargetFilesFromIssue(issue)
	if len(files) == 0 {
		return nil
	}

	// Step 2: Get hotspots for the project
	hotspots, err := checker.CheckHotspots("")
	if err != nil || len(hotspots) == 0 {
		return nil
	}

	// Step 3: Check if any inferred file matches a CRITICAL hotspot (>1500 lines, bloat-size type)
	critical := FindCriticalHotspot(files, hotspots)
	if critical == nil {
		return nil
	}

	// Step 4: Generate extraction task description
	task := GenerateExtractionTask(issue, critical.Path)

	return &ExtractionResult{
		Needed:         true,
		CriticalFile:   critical.Path,
		ExtractionTask: task,
		Hotspot:        critical,
	}
}

// DefaultCreateExtractionIssue creates a beads issue for extraction work and adds a dependency
// so the parent issue is blocked until extraction completes.
// Returns the new extraction issue ID.
func DefaultCreateExtractionIssue(task string, parentIssueID string) (string, error) {
	// Create extraction issue via bd create
	cmd := exec.Command("bd", "create", task, "--type", "task", "--priority", "1", "-l", "triage:ready")
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create extraction issue: %w: %s", err, string(output))
	}

	// Parse the issue ID from bd create output
	extractionID := parseBeadsIDFromOutput(string(output))
	if extractionID == "" {
		return "", fmt.Errorf("could not parse extraction issue ID from: %s", strings.TrimSpace(string(output)))
	}

	// Add dependency: parentIssueID depends on extractionID
	// (extraction must complete before parent can be spawned)
	depCmd := exec.Command("bd", "dep", "add", parentIssueID, extractionID)
	depCmd.Env = os.Environ()
	depOutput, err := depCmd.CombinedOutput()
	if err != nil {
		return extractionID, fmt.Errorf("created %s but failed to add dependency: %w: %s",
			extractionID, err, string(depOutput))
	}

	return extractionID, nil
}

// parseBeadsIDFromOutput extracts a beads issue ID from command output.
// Beads IDs follow the pattern: project-name-shortid (e.g., orch-go-b8c, orch-go-a1b2).
func parseBeadsIDFromOutput(output string) string {
	// Match pattern: lowercase word segments joined by hyphens, ending with 3-4 char short ID
	re := regexp.MustCompile(`[a-z][a-z0-9]*(?:-[a-z0-9]+)+-[a-z0-9]{3,4}`)
	return re.FindString(strings.TrimSpace(output))
}
