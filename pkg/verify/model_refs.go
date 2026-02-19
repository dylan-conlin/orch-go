// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// ModelReferenceMatch represents a modified file that is referenced by one or more models.
type ModelReferenceMatch struct {
	File   string   // Modified file path (repo-relative)
	Models []string // Model paths that reference the file (repo-relative)
}

// FindModelReferencesForModifiedFiles finds model references that point to files
// modified by this agent's commits. Returns empty slice if none found.
func FindModelReferencesForModifiedFiles(workspacePath, projectDir string) ([]ModelReferenceMatch, error) {
	if workspacePath == "" || projectDir == "" {
		return nil, nil
	}

	modifiedFiles, err := getModifiedFilesFromAgentCommits(workspacePath, projectDir)
	if err != nil {
		return nil, err
	}
	if len(modifiedFiles) == 0 {
		return nil, nil
	}

	refsByFile, err := loadModelCodeRefs(projectDir)
	if err != nil {
		return nil, err
	}
	if len(refsByFile) == 0 {
		return nil, nil
	}

	return matchModifiedFilesToModelRefs(modifiedFiles, refsByFile), nil
}

// FormatModelReferenceNote formats an informational note for orch complete output.
func FormatModelReferenceNote(matches []ModelReferenceMatch) string {
	if len(matches) == 0 {
		return ""
	}

	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match.Models) == 0 {
			continue
		}
		modelList := strings.Join(match.Models, ", ")
		parts = append(parts, fmt.Sprintf("%s -> %s", match.File, modelList))
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("NOTE: Modified files referenced by models: %s. Consider updating affected models.", strings.Join(parts, ", "))
}

func matchModifiedFilesToModelRefs(modifiedFiles []string, refsByFile map[string][]string) []ModelReferenceMatch {
	matches := make([]ModelReferenceMatch, 0)
	seen := make(map[string]bool)

	for _, file := range modifiedFiles {
		normalized := NormalizePath(file)
		if normalized == "" || seen[normalized] {
			continue
		}
		if models, ok := refsByFile[normalized]; ok && len(models) > 0 {
			uniqueModels := dedupeStrings(models)
			sort.Strings(uniqueModels)
			matches = append(matches, ModelReferenceMatch{File: normalized, Models: uniqueModels})
		}
		seen[normalized] = true
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].File < matches[j].File
	})

	return matches
}

func getModifiedFilesFromAgentCommits(workspacePath, projectDir string) ([]string, error) {
	spawnTime := spawn.ReadSpawnTime(workspacePath)
	var baseline string
	if manifest, err := spawn.ReadAgentManifest(workspacePath); err == nil {
		baseline = manifest.GitBaseline
	}

	if spawnTime.IsZero() && baseline == "" {
		return nil, nil
	}

	if baseline != "" {
		return gitDiffFilesBetween(projectDir, baseline, "HEAD")
	}

	return gitFilesSince(projectDir, spawnTime)
}

func gitDiffFilesBetween(projectDir, fromRef, toRef string) ([]string, error) {
	if fromRef == "" {
		return nil, nil
	}

	args := []string{"diff", "--name-only", fmt.Sprintf("%s..%s", fromRef, toRef)}
	cmd := exec.Command("git", args...)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list git diff files: %w", err)
	}

	return parseGitFileList(output), nil
}

func gitFilesSince(projectDir string, since time.Time) ([]string, error) {
	if since.IsZero() {
		return nil, nil
	}

	cmd := exec.Command("git", "log", "--name-only", "--pretty=format:", "--since="+since.Format(time.RFC3339))
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list git log files: %w", err)
	}

	return parseGitFileList(output), nil
}

func parseGitFileList(output []byte) []string {
	var files []string
	seen := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		normalized := NormalizePath(line)
		if normalized == "" || seen[normalized] {
			continue
		}
		files = append(files, normalized)
		seen[normalized] = true
	}

	return files
}

func loadModelCodeRefs(projectDir string) (map[string][]string, error) {
	modelDir := filepath.Join(projectDir, ".kb", "models")
	modelPaths, err := filepath.Glob(filepath.Join(modelDir, "*.md"))
	if err != nil {
		return nil, err
	}

	refsByFile := make(map[string][]string)
	for _, modelPath := range modelPaths {
		content, err := os.ReadFile(modelPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read model %s: %w", modelPath, err)
		}

		refs := extractCodeRefsBlock(string(content))
		if len(refs) == 0 {
			continue
		}

		relPath, err := filepath.Rel(projectDir, modelPath)
		if err != nil {
			relPath = modelPath
		}
		relPath = NormalizePath(relPath)

		for _, ref := range refs {
			normalized := NormalizePath(ref)
			if normalized == "" {
				continue
			}
			refsByFile[normalized] = append(refsByFile[normalized], relPath)
		}
	}

	return refsByFile, nil
}

func extractCodeRefsBlock(content string) []string {
	start := strings.Index(content, "<!-- code_refs")
	if start == -1 {
		return nil
	}
	end := strings.Index(content[start:], "<!-- /code_refs")
	if end == -1 {
		return nil
	}

	block := content[start : start+end]
	re := regexp.MustCompile("`([^`]+\\.[^`]+)`")
	matches := re.FindAllStringSubmatch(block, -1)
	if len(matches) == 0 {
		return nil
	}

	refs := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ref := strings.TrimSpace(match[1])
		if ref == "" {
			continue
		}
		if idx := strings.IndexAny(ref, ":#"); idx > 0 {
			ref = ref[:idx]
		}
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if !isLikelyFilePath(ref) {
			continue
		}
		if seen[ref] {
			continue
		}
		refs = append(refs, ref)
		seen[ref] = true
	}

	return refs
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}
