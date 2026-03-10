package spawn

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// maxModelSectionChars limits each injected model section.
// Large models are truncated per section to preserve token budget.
const maxModelSectionChars = 2500

// StalenessResult holds the result of checking a model's staleness.
type StalenessResult struct {
	IsStale      bool     // Whether the model has stale references
	ChangedFiles []string // Files that changed since Last Updated
	DeletedFiles []string // Files that no longer exist
	LastUpdated  string   // The model's Last Updated date
}

type markdownHeading struct {
	line  int
	level int
	title string
}

type modelSpawnSections struct {
	summary            string
	criticalInvariants string
	whyThisFails       string
}

// hasInjectedModelContent checks whether any model matches have extractable content
// (summary, critical invariants, or why-this-fails sections). When true, spawn context
// should include probe guidance so agents produce lightweight probes instead of full investigations.
func hasInjectedModelContent(models []KBContextMatch) bool {
	for _, m := range models {
		if m.Path == "" {
			continue
		}
		sections, err := extractModelSectionsForSpawn(m.Path)
		if err != nil {
			continue
		}
		if sections.summary != "" || sections.criticalInvariants != "" || sections.whyThisFails != "" {
			return true
		}
	}
	return false
}

// DetectCrossRepoModel checks if the primary model path is in a different git repo
// than the project directory. Returns the model's repo root directory if cross-repo,
// or empty string if same-repo or if detection fails.
//
// This catches the case where `--workdir` points to repo A but the model being probed
// lives in repo B. Without this detection, probe files get created in the wrong repo.
func DetectCrossRepoModel(primaryModelPath, projectDir string) string {
	if primaryModelPath == "" || projectDir == "" {
		return ""
	}

	// Normalize paths — resolve symlinks so ~/.kb/ (symlink to .kb/global/)
	// is correctly detected as within the project directory.
	// Use evalSymlinksWithFallback to handle non-existent files by resolving
	// the deepest existing parent directory.
	absModelPath := evalSymlinksWithFallback(primaryModelPath)
	absProjectDir := evalSymlinksWithFallback(projectDir)

	// Check if model path is under project directory
	if strings.HasPrefix(absModelPath, absProjectDir+string(filepath.Separator)) {
		return "" // Same repo, no cross-repo issue
	}

	// Model is outside project dir - find the model's repo root
	// Walk up from the model path to find the nearest .git directory
	modelDir := filepath.Dir(absModelPath)
	for modelDir != "/" && modelDir != "." {
		gitDir := filepath.Join(modelDir, ".git")
		if info, err := os.Stat(gitDir); err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return modelDir
		}
		modelDir = filepath.Dir(modelDir)
	}

	// Couldn't find git root, but model is still outside project dir
	// Return the directory containing .kb/ as a reasonable guess
	modelDir = filepath.Dir(absModelPath)
	for modelDir != "/" && modelDir != "." {
		kbDir := filepath.Join(modelDir, ".kb")
		if info, err := os.Stat(kbDir); err == nil && info.IsDir() {
			return modelDir
		}
		modelDir = filepath.Dir(modelDir)
	}

	return "" // Can't determine model repo
}

func formatModelMatchForSpawn(match KBContextMatch, projectDir string, stalenessMeta *StalenessEventMeta) (string, bool) {
	var sb strings.Builder
	isStale := false

	sb.WriteString(fmt.Sprintf("- %s\n", match.Title))
	if match.Path != "" {
		sb.WriteString(fmt.Sprintf("  - See: %s\n", match.Path))
	}

	if match.Path == "" {
		return sb.String(), isStale
	}

	// Check for staleness
	modelContent, err := os.ReadFile(match.Path)
	if err == nil {
		stalenessResult, err := checkModelStaleness(string(modelContent), projectDir)
		if err == nil && stalenessResult.IsStale {
			isStale = true
			if err := RecordModelStalenessEvent(match.Path, stalenessResult, stalenessMeta); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record staleness event: %v\n", err)
			}
			// Prepend staleness warning
			sb.WriteString("  - **STALENESS WARNING:**\n")
			sb.WriteString(fmt.Sprintf("    This model was last updated %s.\n", stalenessResult.LastUpdated))
			if len(stalenessResult.ChangedFiles) > 0 {
				sb.WriteString(fmt.Sprintf("    Changed files: %s.\n", strings.Join(stalenessResult.ChangedFiles, ", ")))
			}
			if len(stalenessResult.DeletedFiles) > 0 {
				sb.WriteString(fmt.Sprintf("    Deleted files: %s.\n", strings.Join(stalenessResult.DeletedFiles, ", ")))
			}
			sb.WriteString("    Verify model claims about these files against current code.\n")
		}
	}

	sections, err := extractModelSectionsForSpawn(match.Path)
	if err != nil {
		return sb.String(), isStale
	}

	hasInjectedContent := false
	if sections.summary != "" {
		hasInjectedContent = true
		sb.WriteString("  - Summary:\n")
		sb.WriteString(indentBlock(sections.summary, "    "))
	}
	if sections.criticalInvariants != "" {
		hasInjectedContent = true
		sb.WriteString("  - Critical Invariants:\n")
		sb.WriteString(indentBlock(sections.criticalInvariants, "    "))
	}
	if sections.whyThisFails != "" {
		hasInjectedContent = true
		sb.WriteString("  - Why This Fails:\n")
		sb.WriteString(indentBlock(sections.whyThisFails, "    "))
	}

	if hasInjectedContent {
		sb.WriteString("  - Your findings should confirm, contradict, or extend the claims above.\n")
	}

	// Inject recent probes from this model's probes/ directory
	if match.Path != "" {
		probes := ListRecentProbes(match.Path, MaxRecentProbes)
		probeContent := FormatProbesForSpawn(probes)
		if probeContent != "" {
			sb.WriteString(probeContent)
		}
	}

	return sb.String(), isStale
}

func extractModelSectionsForSpawn(path string) (*modelSpawnSections, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	headings := collectMarkdownHeadings(lines)

	sections := &modelSpawnSections{
		summary: truncateModelSection(extractSectionByHeading(lines, headings, func(title string) bool {
			return strings.HasPrefix(title, "summary") || title == "what this is" || strings.HasPrefix(title, "executive summary")
		})),
		criticalInvariants: truncateModelSection(extractSectionByHeading(lines, headings, func(title string) bool { return strings.HasPrefix(title, "critical invariants") })),
		whyThisFails:       truncateModelSection(extractSectionByHeading(lines, headings, func(title string) bool { return strings.HasPrefix(title, "why this fails") })),
	}

	return sections, nil
}

func collectMarkdownHeadings(lines []string) []markdownHeading {
	headings := make([]markdownHeading, 0)
	inCodeFence := false

	for i, raw := range lines {
		line := strings.TrimSpace(raw)

		if strings.HasPrefix(line, "```") {
			inCodeFence = !inCodeFence
			continue
		}
		if inCodeFence {
			continue
		}

		level, title, ok := parseMarkdownHeading(line)
		if !ok {
			continue
		}

		headings = append(headings, markdownHeading{
			line:  i,
			level: level,
			title: normalizeHeading(title),
		})
	}

	return headings
}

func parseMarkdownHeading(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}

	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}

	if level < 2 || level > 6 {
		return 0, "", false
	}

	if len(line) == level || line[level] != ' ' {
		return 0, "", false
	}

	title := strings.TrimSpace(line[level:])
	if title == "" {
		return 0, "", false
	}

	return level, title, true
}

func normalizeHeading(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

func extractSectionByHeading(lines []string, headings []markdownHeading, matcher func(string) bool) string {
	for idx, heading := range headings {
		if !matcher(heading.title) {
			continue
		}

		startLine := heading.line + 1
		endLine := len(lines)

		for next := idx + 1; next < len(headings); next++ {
			if headings[next].level <= heading.level {
				endLine = headings[next].line
				break
			}
		}

		if startLine >= len(lines) || startLine >= endLine {
			return ""
		}

		content := strings.TrimSpace(strings.Join(lines[startLine:endLine], "\n"))
		return content
	}

	return ""
}

func truncateModelSection(content string) string {
	content = strings.TrimSpace(content)
	if content == "" || len(content) <= maxModelSectionChars {
		return content
	}

	truncated := strings.TrimSpace(content[:maxModelSectionChars])
	if lastBreak := strings.LastIndexAny(truncated, "\n "); lastBreak > maxModelSectionChars/2 {
		truncated = strings.TrimSpace(truncated[:lastBreak])
	}

	return truncated + "\n... [truncated]"
}

func indentBlock(content, indent string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = indent
			continue
		}
		lines[i] = indent + line
	}

	return strings.Join(lines, "\n") + "\n"
}

// evalSymlinksWithFallback resolves symlinks in a path, falling back to resolving
// the deepest existing parent directory when the full path doesn't exist.
// On macOS, /var is a symlink to /private/var, so EvalSymlinks on temp dirs
// resolves to /private/var/... while Abs keeps /var/... — this mismatch breaks
// path prefix comparison. By resolving the existing parent, both paths use
// the same canonical form.
func evalSymlinksWithFallback(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved
	}

	// File doesn't exist — resolve the deepest existing parent directory
	// and append the remaining path components
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}

	// Walk up until we find an existing directory
	remaining := ""
	dir := absPath
	for {
		resolved, err := filepath.EvalSymlinks(dir)
		if err == nil {
			if remaining == "" {
				return resolved
			}
			return filepath.Join(resolved, remaining)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, can't resolve
			return absPath
		}
		remaining = filepath.Join(filepath.Base(dir), remaining)
		dir = parent
	}
}

// extractCodeRefs parses file paths from Primary Evidence sections in model content.
// Looks for backtick-quoted file paths (with optional :line or :function suffixes).
// Returns relative file paths without line numbers or function names.
func extractCodeRefs(content string) []string {
	var refs []string
	seen := make(map[string]bool)

	// Look for Primary Evidence section
	lines := strings.Split(content, "\n")
	inPrimaryEvidence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of Primary Evidence section
		if strings.Contains(trimmed, "Primary Evidence") {
			inPrimaryEvidence = true
			continue
		}

		// End of section (next ## heading, or explicit code_refs close marker)
		if inPrimaryEvidence {
			if strings.HasPrefix(trimmed, "##") || strings.HasPrefix(trimmed, "# ") {
				break
			}
			if strings.Contains(trimmed, "<!-- /code_refs") {
				break
			}
		}

		// Extract file paths from backticks
		if inPrimaryEvidence && strings.Contains(trimmed, "`") {
			// Find all backtick-quoted content
			parts := strings.Split(trimmed, "`")
			for i := 1; i < len(parts); i += 2 {
				path := strings.TrimSpace(parts[i])

				// Check if this looks like a file path (has .go, .md, etc.)
				if !strings.Contains(path, ".") {
					continue
				}

				// Remove :line number or :function() suffix
				if idx := strings.Index(path, ":"); idx > 0 {
					path = path[:idx]
				}

				// Skip if already seen
				if seen[path] {
					continue
				}

				// Only include if it looks like a file path (contains / or has valid extension)
				if strings.Contains(path, "/") || hasValidFileExtension(path) {
					refs = append(refs, path)
					seen[path] = true
				}
			}
		}
	}

	return refs
}

// hasValidFileExtension checks if a string ends with a common file extension.
func hasValidFileExtension(path string) bool {
	validExts := []string{".go", ".md", ".yaml", ".yml", ".json", ".toml", ".sh", ".js", ".ts", ".tsx", ".jsx"}
	for _, ext := range validExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// extractLastUpdated parses the "Last Updated:" date from model content.
// Returns the date string (YYYY-MM-DD format) or empty string if not found.
func extractLastUpdated(content string) string {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for **Last Updated:** or **last updated:**
		if strings.HasPrefix(trimmed, "**Last Updated:") || strings.HasPrefix(trimmed, "**last updated:") {
			// Extract the date after the colon
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				date := strings.TrimSpace(parts[1])
				// Remove leading and trailing ** (markdown bold markers)
				date = strings.TrimPrefix(date, "**")
				date = strings.TrimSuffix(date, "**")
				return strings.TrimSpace(date)
			}
		}
	}

	return ""
}

// checkModelStaleness checks if a model's referenced files have changed since its Last Updated date.
// Returns a StalenessResult with changed/deleted files, or an empty result if the model can't be checked.
func checkModelStaleness(modelContent string, projectDir string) (*StalenessResult, error) {
	result := &StalenessResult{
		IsStale:      false,
		ChangedFiles: []string{},
		DeletedFiles: []string{},
	}

	// Extract Last Updated date
	lastUpdated := extractLastUpdated(modelContent)
	if lastUpdated == "" {
		// No Last Updated field - can't check staleness
		return result, nil
	}
	result.LastUpdated = lastUpdated

	// Parse date and add 1 day to avoid same-day boundary false positives.
	// git log --since=YYYY-MM-DD includes all commits from midnight of that day,
	// but the model was updated at some point during that day, so earlier same-day
	// commits would appear as "changed" even though the model already accounts for them.
	sinceDate := lastUpdated
	if t, err := time.Parse("2006-01-02", lastUpdated); err == nil {
		sinceDate = t.AddDate(0, 0, 1).Format("2006-01-02")
	}

	// Extract code references
	codeRefs := extractCodeRefs(modelContent)
	if len(codeRefs) == 0 {
		// No code references - can't check staleness
		return result, nil
	}

	// Check each referenced file
	for _, ref := range codeRefs {
		filePath := ref
		if strings.HasPrefix(filePath, "~/") {
			// Expand tilde to home directory
			home, err := os.UserHomeDir()
			if err == nil {
				filePath = filepath.Join(home, filePath[2:])
			}
		} else if !strings.HasPrefix(filePath, "/") {
			// Make relative paths absolute
			filePath = fmt.Sprintf("%s/%s", projectDir, ref)
		}

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			result.DeletedFiles = append(result.DeletedFiles, ref)
			result.IsStale = true
			continue
		}

		// Check if file changed since Last Updated using git log
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "git", "log", "--since="+sinceDate, "--oneline", "--", ref)
		cmd.Dir = projectDir
		output, err := cmd.Output()
		if err != nil {
			// Git command failed - might not be in a git repo, or file not tracked
			// This is not an error condition - just skip this file
			continue
		}

		// If output is non-empty, file has commits since Last Updated
		if len(strings.TrimSpace(string(output))) > 0 {
			result.ChangedFiles = append(result.ChangedFiles, ref)
			result.IsStale = true
		}
	}

	return result, nil
}
