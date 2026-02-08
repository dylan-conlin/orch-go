package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"gopkg.in/yaml.v3"
)

var regexNumberedItem = regexp.MustCompile(`^\d+\.\s+`)

// DesignActionItem represents one actionable item extracted from a design artifact.
type DesignActionItem struct {
	Section string
	Text    string
}

// DesignDecompositionDoc is a design artifact with actionable items that still need decomposition.
type DesignDecompositionDoc struct {
	Path             string
	RelativePath     string
	ActionItems      []DesignActionItem
	Decomposed       bool
	DecompositionIDs []string
}

// FindDesignDocsRequiringDecomposition scans design artifacts changed by the workspace and
// returns those that contain actionable items but are not marked as decomposed.
func FindDesignDocsRequiringDecomposition(workspacePath, projectDir string) ([]DesignDecompositionDoc, []string, error) {
	files, warnings, err := changedFilesForWorkspace(workspacePath, projectDir)
	if err != nil {
		return nil, warnings, err
	}

	if len(files) == 0 {
		return nil, warnings, nil
	}

	var pending []DesignDecompositionDoc
	for _, relPath := range files {
		normalized := NormalizePath(relPath)
		if !isDesignArtifactPath(normalized) {
			continue
		}

		absPath := relPath
		if !filepath.IsAbs(relPath) {
			absPath = filepath.Join(projectDir, relPath)
		}

		content, readErr := os.ReadFile(absPath)
		if readErr != nil {
			warnings = append(warnings, fmt.Sprintf("failed to read design artifact %s: %v", normalized, readErr))
			continue
		}

		items := ExtractDesignActionItems(string(content))
		if len(items) == 0 {
			continue
		}

		meta := ParseDesignDecompositionMetadata(string(content))
		if meta.Decomposed && len(meta.DecompositionIssues) >= len(items) {
			continue
		}

		pending = append(pending, DesignDecompositionDoc{
			Path:             absPath,
			RelativePath:     normalized,
			ActionItems:      items,
			Decomposed:       meta.Decomposed,
			DecompositionIDs: meta.DecompositionIssues,
		})
	}

	return pending, warnings, nil
}

func changedFilesForWorkspace(workspacePath, projectDir string) ([]string, []string, error) {
	var warnings []string

	spawnTime := spawn.ReadSpawnTime(workspacePath)
	var baseline string
	if manifest, err := spawn.ReadAgentManifest(workspacePath); err == nil {
		baseline = strings.TrimSpace(manifest.GitBaseline)
	}

	if baseline == "" && spawnTime.IsZero() {
		warnings = append(warnings, "spawn metadata unavailable (baseline and spawn time missing), skipping design decomposition scan")
		return nil, warnings, nil
	}

	files, err := GetGitDiffFiles(projectDir, spawnTime, baseline)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to get changed files for decomposition scan: %w", err)
	}

	return files, warnings, nil
}

func isDesignArtifactPath(path string) bool {
	if !strings.HasSuffix(path, ".md") {
		return false
	}

	return strings.HasPrefix(path, ".kb/investigations/") ||
		strings.HasPrefix(path, "docs/designs/")
}

// ExtractDesignActionItems extracts actionable list items from implementation-oriented sections.
func ExtractDesignActionItems(content string) []DesignActionItem {
	lines := strings.Split(content, "\n")
	var items []DesignActionItem
	seen := map[string]bool{}

	inTargetSection := false
	currentSection := ""
	targetHeadingLevel := 0

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		level, heading, isHeading := parseMarkdownHeading(line)
		if isHeading {
			if inTargetSection && level <= targetHeadingLevel {
				inTargetSection = false
				currentSection = ""
				targetHeadingLevel = 0
			}

			normalizedHeading := normalizeHeading(heading)
			if normalizedHeading == "implementation notes" ||
				normalizedHeading == "components to build" ||
				normalizedHeading == "api changes" {
				inTargetSection = true
				currentSection = heading
				targetHeadingLevel = level
			}
			continue
		}

		if !inTargetSection {
			continue
		}

		txt, ok := parseActionItem(line)
		if !ok {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(txt))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true

		items = append(items, DesignActionItem{
			Section: currentSection,
			Text:    txt,
		})
	}

	return items
}

func parseMarkdownHeading(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}

	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}

	if level == 0 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}

	heading := strings.TrimSpace(line[level:])
	if heading == "" {
		return 0, "", false
	}

	return level, heading, true
}

func normalizeHeading(heading string) string {
	heading = strings.TrimSpace(strings.ToLower(heading))
	heading = strings.TrimSuffix(heading, ":")
	return heading
}

func parseActionItem(line string) (string, bool) {
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		return strings.TrimSpace(line[2:]), true
	}

	if regexNumberedItem.MatchString(line) {
		parts := regexNumberedItem.Split(line, 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1]), true
		}
	}

	return "", false
}

// DesignDecompositionMetadata contains decomposition fields parsed from YAML frontmatter.
type DesignDecompositionMetadata struct {
	Decomposed          bool     `yaml:"decomposed"`
	DecompositionParent string   `yaml:"decomposition_parent,omitempty"`
	DecompositionIssues []string `yaml:"decomposition_issues,omitempty"`
}

// ParseDesignDecompositionMetadata parses decomposition metadata from frontmatter.
func ParseDesignDecompositionMetadata(content string) DesignDecompositionMetadata {
	frontmatter, _, ok := splitFrontmatter(content)
	if !ok {
		return DesignDecompositionMetadata{}
	}

	var meta DesignDecompositionMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return DesignDecompositionMetadata{}
	}

	meta.DecompositionIssues = dedupeAndSortIDs(meta.DecompositionIssues)
	return meta
}

// MarkDesignDocumentDecomposed updates the document frontmatter with decomposition state.
func MarkDesignDocumentDecomposed(filePath, parentID string, issueIDs []string) error {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read design document: %w", err)
	}
	content := string(contentBytes)

	frontmatter, body, hasFrontmatter := splitFrontmatter(content)

	frontmatterMap := map[string]interface{}{}
	if hasFrontmatter && strings.TrimSpace(frontmatter) != "" {
		if err := yaml.Unmarshal([]byte(frontmatter), &frontmatterMap); err != nil {
			return fmt.Errorf("failed to parse frontmatter in %s: %w", filePath, err)
		}
	}

	frontmatterMap["decomposed"] = true
	if parentID != "" {
		frontmatterMap["decomposition_parent"] = parentID
	}
	frontmatterMap["decomposition_issues"] = dedupeAndSortIDs(issueIDs)
	frontmatterMap["decomposed_at"] = time.Now().Format("2006-01-02")

	yamlBytes, err := yaml.Marshal(frontmatterMap)
	if err != nil {
		return fmt.Errorf("failed to marshal decomposition frontmatter: %w", err)
	}

	cleanBody := strings.TrimLeft(body, "\r\n")
	updated := "---\n" + string(yamlBytes) + "---\n\n" + cleanBody

	if err := os.WriteFile(filePath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write updated design document: %w", err)
	}

	return nil
}

func splitFrontmatter(content string) (string, string, bool) {
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return "", content, false
	}

	lines := strings.Split(content, "\n")
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return "", content, false
	}

	frontmatter := strings.Join(lines[1:endIndex], "\n")
	body := ""
	if endIndex+1 < len(lines) {
		body = strings.Join(lines[endIndex+1:], "\n")
	}

	return frontmatter, body, true
}

func dedupeAndSortIDs(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
