// Package spawn provides probe merge functionality for the completion workflow.
// When an agent produces probes against a model, the orchestrator can merge findings
// back into the model at completion time.
package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ProjectProbe represents a probe file found in any model's probes directory.
type ProjectProbe struct {
	ModelName string // e.g., "completion-verification"
	ModelPath string // Full path to the model .md file
	Probe     probeEntry
	Impact    string // Extracted "Model Impact" section content
}

// FindProjectProbes scans all .kb/models/*/probes/*.md files in the project directory.
// Returns all probes found across all models. Returns nil if none found.
func FindProjectProbes(projectDir string) []ProjectProbe {
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	modelEntries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil
	}

	var result []ProjectProbe
	for _, entry := range modelEntries {
		if !entry.IsDir() {
			continue
		}
		modelName := entry.Name()
		probesDir := filepath.Join(modelsDir, modelName, "probes")
		probeEntries, err := os.ReadDir(probesDir)
		if err != nil {
			continue
		}

		// Find corresponding model file
		modelPath := filepath.Join(modelsDir, modelName+".md")
		if _, err := os.Stat(modelPath); err != nil {
			continue // No model file found
		}

		for _, pe := range probeEntries {
			if pe.IsDir() || !strings.HasSuffix(pe.Name(), ".md") || pe.Name() == ".gitkeep" {
				continue
			}
			info, err := pe.Info()
			if err != nil {
				continue
			}

			probePath := filepath.Join(probesDir, pe.Name())
			impact := ReadProbeModelImpact(probePath)

			result = append(result, ProjectProbe{
				ModelName: modelName,
				ModelPath: modelPath,
				Probe: probeEntry{
					Path:     probePath,
					Name:     strings.TrimSuffix(pe.Name(), ".md"),
					ModTime:  info.ModTime(),
					ModelDir: modelName,
				},
				Impact: impact,
			})
		}
	}

	return result
}

// ReadProbeModelImpact reads a probe file and extracts the "Model Impact" section.
// Returns the content between "## Model Impact" and the next "##" heading (or end of file).
// Returns empty string if section not found.
func ReadProbeModelImpact(probePath string) string {
	data, err := os.ReadFile(probePath)
	if err != nil {
		return ""
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	var impactLines []string
	inImpactSection := false

	for _, line := range lines {
		if strings.HasPrefix(line, "## Model Impact") {
			inImpactSection = true
			continue
		}
		if inImpactSection && strings.HasPrefix(line, "## ") {
			break // Hit next section
		}
		if inImpactSection {
			impactLines = append(impactLines, line)
		}
	}

	return strings.TrimSpace(strings.Join(impactLines, "\n"))
}

// FormatProbeMergeSummary formats the probes for display to the orchestrator during completion.
// Shows probe name, model name, and impact summary.
func FormatProbeMergeSummary(probes []ProjectProbe) string {
	if len(probes) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agent produced %d probe(s) against model(s):\n", len(probes)))
	for _, p := range probes {
		sb.WriteString(fmt.Sprintf("  - %s → model: %s\n", p.Probe.Name, p.ModelName))
		if p.Impact != "" {
			// Show first line of impact as summary
			firstLine := strings.SplitN(p.Impact, "\n", 2)[0]
			if len(firstLine) > 80 {
				firstLine = firstLine[:77] + "..."
			}
			sb.WriteString(fmt.Sprintf("    Impact: %s\n", firstLine))
		}
	}
	return sb.String()
}

// lastUpdatedRe matches the "**Last Updated:** YYYY-MM-DD" line in model files.
var lastUpdatedRe = regexp.MustCompile(`(?m)^\*\*Last Updated:\*\*\s+\d{4}-\d{2}-\d{2}`)

// MergeProbeIntoModel appends probe findings to the model file and updates the Last Updated date.
// It appends a "Merged Probes" section at the end of the model if one doesn't exist,
// or appends to the existing section.
func MergeProbeIntoModel(modelPath string, probe ProjectProbe) error {
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return fmt.Errorf("failed to read model %s: %w", modelPath, err)
	}

	content := string(data)
	today := time.Now().Format("2006-01-02")

	// Update Last Updated date
	if lastUpdatedRe.MatchString(content) {
		content = lastUpdatedRe.ReplaceAllString(content, fmt.Sprintf("**Last Updated:** %s", today))
	}

	// Build merge entry
	mergeEntry := fmt.Sprintf("\n### Probe: %s (%s)\n\n%s\n",
		probe.Probe.Name, today, probe.Impact)

	// Append to existing "## Merged Probes" section or create one
	mergedSectionHeader := "## Merged Probes"
	if strings.Contains(content, mergedSectionHeader) {
		// Find the end of the merged probes section (next ## heading or EOF)
		idx := strings.Index(content, mergedSectionHeader)
		afterHeader := content[idx+len(mergedSectionHeader):]

		// Find next ## heading after the merged probes section
		nextSection := regexp.MustCompile(`(?m)^## [^M]`)
		loc := nextSection.FindStringIndex(afterHeader)

		if loc != nil {
			// Insert before the next section
			insertPoint := idx + len(mergedSectionHeader) + loc[0]
			content = content[:insertPoint] + mergeEntry + "\n" + content[insertPoint:]
		} else {
			// Append at end
			content = strings.TrimRight(content, "\n") + mergeEntry + "\n"
		}
	} else {
		// Create new section at end
		content = strings.TrimRight(content, "\n") + "\n\n---\n\n" + mergedSectionHeader + "\n" + mergeEntry + "\n"
	}

	if err := os.WriteFile(modelPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write model %s: %w", modelPath, err)
	}

	return nil
}
