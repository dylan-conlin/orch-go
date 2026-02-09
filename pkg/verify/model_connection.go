// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

var regexModelCandidateField = regexp.MustCompile(`(?i)^\s*(?:\*\*)?model\s+candidate:(?:\*\*)?\s*(.+?)\s*$`)

// ModelConnectionResult represents verification of model linkage for knowledge-producing skills.
type ModelConnectionResult struct {
	Passed             bool
	SkillName          string
	HasProbeConnection bool
	ProbeFiles         []string
	HasModelCandidate  bool
	ModelCandidate     string
	Errors             []string
	Warnings           []string
}

var skillsRequiringModelConnection = map[string]bool{
	"investigation": true,
	"research":      true,
	"architect":     true,
}

// IsSkillRequiringModelConnection returns true for knowledge-producing skills that
// must connect findings to an existing model.
func IsSkillRequiringModelConnection(skillName string) bool {
	skill := strings.ToLower(strings.TrimSpace(skillName))
	if skill == "" {
		return false
	}
	return skillsRequiringModelConnection[skill]
}

// VerifyModelConnection checks that knowledge-producing skills provide either:
// 1) A probe file changed under .kb/models/*/probes/*.md, or
// 2) A "Model candidate:" field in SYNTHESIS.md.
func VerifyModelConnection(skillName, workspacePath, projectDir string) ModelConnectionResult {
	result := ModelConnectionResult{
		Passed:    true,
		SkillName: strings.ToLower(strings.TrimSpace(skillName)),
	}

	if !IsSkillRequiringModelConnection(result.SkillName) {
		return result
	}

	result.ModelCandidate = extractModelCandidate(workspacePath)
	result.HasModelCandidate = result.ModelCandidate != ""

	probeFiles, warnings := detectProbeConnections(workspacePath, projectDir)
	result.ProbeFiles = probeFiles
	result.HasProbeConnection = len(probeFiles) > 0
	result.Warnings = append(result.Warnings, warnings...)

	if result.HasProbeConnection || result.HasModelCandidate {
		return result
	}

	result.Passed = false
	result.Errors = append(result.Errors,
		fmt.Sprintf("skill '%s' requires model connection evidence", result.SkillName),
		"add a probe file under .kb/models/*/probes/*.md OR set 'Model candidate:' in SYNTHESIS.md",
	)
	return result
}

// VerifyModelConnectionForCompletion returns nil when the gate is not applicable.
func VerifyModelConnectionForCompletion(skillName, workspacePath, projectDir string) *ModelConnectionResult {
	if !IsSkillRequiringModelConnection(skillName) {
		return nil
	}
	result := VerifyModelConnection(skillName, workspacePath, projectDir)
	return &result
}

func detectProbeConnections(workspacePath, projectDir string) ([]string, []string) {
	var warnings []string
	if workspacePath == "" || projectDir == "" {
		return nil, warnings
	}

	spawnTime := spawn.ReadSpawnTime(workspacePath)
	baseline := ""
	if manifest, err := spawn.ReadAgentManifest(workspacePath); err == nil {
		baseline = strings.TrimSpace(manifest.GitBaseline)
	}

	files, err := GetGitDiffFiles(projectDir, spawnTime, baseline)
	if err == nil {
		matched := filterProbePaths(files)
		if len(matched) > 0 {
			return matched, warnings
		}
	} else {
		warnings = append(warnings, fmt.Sprintf("failed to detect probe files from git diff: %v", err))
	}

	matched := scanProbeFilesSince(projectDir, spawnTime)
	return matched, warnings
}

func filterProbePaths(files []string) []string {
	var out []string
	seen := make(map[string]bool)
	for _, file := range files {
		normalized := NormalizePath(file)
		if !isModelProbePath(normalized) || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	return out
}

func scanProbeFilesSince(projectDir string, spawnTime time.Time) []string {
	pattern := filepath.Join(projectDir, ".kb", "models", "*", "probes", "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}

	var out []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if filepath.Base(match) == ".gitkeep" {
			continue
		}

		rel, relErr := filepath.Rel(projectDir, match)
		if relErr != nil {
			continue
		}

		normalized := NormalizePath(rel)
		if !isModelProbePath(normalized) || seen[normalized] {
			continue
		}

		if !spawnTime.IsZero() {
			info, statErr := os.Stat(match)
			if statErr != nil || info.ModTime().Before(spawnTime) {
				continue
			}
		}

		seen[normalized] = true
		out = append(out, normalized)
	}

	return out
}

func extractModelCandidate(workspacePath string) string {
	if workspacePath == "" {
		return ""
	}

	data, err := os.ReadFile(filepath.Join(workspacePath, "SYNTHESIS.md"))
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		matches := regexModelCandidateField.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}
		value := strings.TrimSpace(matches[1])
		if isPlaceholderCandidate(value) {
			continue
		}
		return value
	}

	return ""
}

func isPlaceholderCandidate(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		return true
	}
	return false
}

func isModelProbePath(path string) bool {
	if !strings.HasPrefix(path, ".kb/models/") {
		return false
	}
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return false
	}
	if parts[3] != "probes" {
		return false
	}
	if strings.HasSuffix(parts[len(parts)-1], ".md") {
		return true
	}
	return false
}
