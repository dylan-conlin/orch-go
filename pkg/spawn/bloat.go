// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// BloatThreshold is the line count above which a file is considered bloated.
const BloatThreshold = 800

// BloatWarning represents a warning about a bloated file.
type BloatWarning struct {
	// Path is the file path (relative to project root)
	Path string
	// LineCount is the number of lines in the file
	LineCount int
	// Recommendation is the suggested action
	Recommendation string
}

// Pre-compiled regex for extracting file paths from task text.
// Matches patterns like:
// - pkg/spawn/context.go
// - src/components/Button.tsx
// - ./internal/handler.go
// Does not match URLs or common non-file patterns.
var regexFilePath = regexp.MustCompile(`(?:^|[\s\`+"``"+`'"(])([a-zA-Z0-9_./\-]+\.[a-zA-Z0-9]+)(?:[\s\`+"``"+`'"):,]|$)`)

// CheckBloatedFiles extracts file paths from the task string and checks if any exceed
// the bloat threshold. Returns warnings for any bloated files found.
// Test files (*_test.go, *.test.ts, etc.) are exempt as they are expected to be longer.
func CheckBloatedFiles(task, projectDir string) []BloatWarning {
	// Extract file paths from task
	paths := extractFilePaths(task)
	if len(paths) == 0 {
		return nil
	}

	var warnings []BloatWarning
	for _, relPath := range paths {
		// Skip test files - they are expected to be longer
		if isTestFile(relPath) {
			continue
		}

		// Construct full path and check if file exists
		fullPath := filepath.Join(projectDir, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue // File doesn't exist, skip
		}

		// Count lines
		lineCount, err := countFileLines(fullPath)
		if err != nil {
			continue // Can't count lines, skip
		}

		// Check against threshold
		if lineCount > BloatThreshold {
			warnings = append(warnings, BloatWarning{
				Path:           relPath,
				LineCount:      lineCount,
				Recommendation: generateBloatRecommendation(relPath, lineCount),
			})
		}
	}

	return warnings
}

// extractFilePaths extracts potential file paths from text.
// Looks for patterns like pkg/foo/bar.go, src/component.tsx, etc.
func extractFilePaths(text string) []string {
	matches := regexFilePath.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	// Use map to deduplicate
	seen := make(map[string]bool)
	var paths []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		path := strings.TrimPrefix(match[1], "./")

		// Filter out non-file patterns
		if !looksLikeFilePath(path) {
			continue
		}

		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}

	return paths
}

// looksLikeFilePath returns true if the string looks like a valid file path.
// Filters out URLs, version numbers, and other common false positives.
func looksLikeFilePath(s string) bool {
	// Must contain at least one directory separator or be a direct file reference
	if !strings.Contains(s, "/") && !strings.Contains(s, ".") {
		return false
	}

	// Filter out URLs
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return false
	}

	// Filter out version strings like v1.0.0
	if matched, _ := regexp.MatchString(`^v?\d+\.\d+`, s); matched {
		return false
	}

	// Must have a recognizable source file extension
	ext := filepath.Ext(s)
	sourceExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
		".svelte": true, ".py": true, ".rb": true, ".java": true,
		".c": true, ".cpp": true, ".h": true, ".hpp": true, ".rs": true,
		".swift": true, ".kt": true, ".scala": true, ".sh": true,
		".css": true, ".scss": true, ".html": true, ".vue": true,
		".yaml": true, ".yml": true, ".json": true, ".md": true,
	}

	return sourceExts[ext]
}

// isTestFile returns true if the file is a test file.
// Test files are exempt from bloat warnings as they are expected to be longer.
func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go") ||
		strings.HasSuffix(path, ".test.ts") ||
		strings.HasSuffix(path, ".test.js") ||
		strings.HasSuffix(path, ".test.tsx") ||
		strings.HasSuffix(path, ".test.jsx") ||
		strings.HasSuffix(path, ".spec.ts") ||
		strings.HasSuffix(path, ".spec.js") ||
		strings.Contains(path, "__tests__/")
}

// countFileLines counts the number of lines in a file.
// Uses buffered reading for efficiency.
func countFileLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	buf := make([]byte, 32*1024) // 32KB buffer
	lineSep := []byte{'\n'}

	for {
		c, err := file.Read(buf)
		count += strings.Count(string(buf[:c]), string(lineSep))

		if err != nil {
			break // EOF or error
		}
	}

	return count, nil
}

// generateBloatRecommendation creates a recommendation based on file size.
func generateBloatRecommendation(file string, lines int) string {
	if lines > 1500 {
		return fmt.Sprintf("CRITICAL (%d lines): Consider extracting components before making changes. See .kb/guides/code-extraction-patterns.md", lines)
	}
	return fmt.Sprintf("WARNING (%d lines): File exceeds 800-line threshold. Consider extraction if adding significant code.", lines)
}

// GenerateBloatWarningSection creates the bloat warning section for SPAWN_CONTEXT.md.
// Returns empty string if no warnings.
func GenerateBloatWarningSection(warnings []BloatWarning) string {
	if len(warnings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## ⚠️ BLOAT WARNING\n\n")
	sb.WriteString("The following files mentioned in your task exceed the 800-line bloat threshold:\n\n")

	for _, w := range warnings {
		sb.WriteString(fmt.Sprintf("- **`%s`** - %s\n", w.Path, w.Recommendation))
	}

	sb.WriteString("\n**Extraction Recommendations:**\n")
	sb.WriteString("1. Before adding code, check if functionality can be extracted to a new file\n")
	sb.WriteString("2. Look for natural boundaries: related functions, types, or concerns\n")
	sb.WriteString("3. Reference: `.kb/guides/code-extraction-patterns.md` for extraction workflow\n\n")
	sb.WriteString("**Why this matters:** Files over 800 lines degrade LLM comprehension (context noise).\n\n")

	return sb.String()
}
