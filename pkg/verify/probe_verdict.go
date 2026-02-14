// Package verify provides verification helpers for agent completion.
// This file provides probe verdict parsing for the orch complete reverse path.
// Probes in .kb/models/{name}/probes/ contain a Model Impact section with
// verdicts (confirms/contradicts/extends) that need to be surfaced during
// completion so the orchestrator can merge model updates.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ProbeVerdict represents a parsed probe verdict from a probe file's Model Impact section.
type ProbeVerdict struct {
	ModelName string // Model name (e.g., "completion-verification")
	ProbePath string // Full path to probe file
	Title     string // Title from "# Probe: {title}"
	Verdict   string // confirms, contradicts, or extends
	Details   string // Explanation text after the verdict
	Question  string // From "## Question" section
}

// Pre-compiled regex patterns for probe parsing
var (
	regexProbeTitle    = regexp.MustCompile(`(?m)^# Probe:\s*(.+)$`)
	regexProbeModel    = regexp.MustCompile(`(?m)\*\*Model:\*\*\s*(.+)$`)
	regexStructVerdict = regexp.MustCompile(`(?m)\*\*Verdict:\*\*\s*(\w+)\s*[—–-]\s*(.+)$`)
	regexCheckConfirms = regexp.MustCompile(`(?m)^-\s*\[x\]\s*\*\*Confirms\*\*\s*(?:invariant:\s*)?(.+)$`)
	regexCheckContra   = regexp.MustCompile(`(?m)^-\s*\[x\]\s*\*\*Contradicts\*\*\s*(?:invariant:\s*)?(.+)$`)
	regexCheckExtends  = regexp.MustCompile(`(?m)^-\s*\[x\]\s*\*\*Extends\*\*\s*(?:model with:\s*)?(.+)$`)
)

// ParseProbeVerdict extracts the verdict from a probe file's content.
// Supports two formats:
//  1. Structured: "**Verdict:** extends — description"
//  2. Checkbox: "- [x] **Confirms** invariant: description"
func ParseProbeVerdict(content []byte) ProbeVerdict {
	text := string(content)
	v := ProbeVerdict{}

	// Parse title
	if m := regexProbeTitle.FindStringSubmatch(text); len(m) >= 2 {
		v.Title = strings.TrimSpace(m[1])
	}

	// Parse model name
	if m := regexProbeModel.FindStringSubmatch(text); len(m) >= 2 {
		modelStr := strings.TrimSpace(m[1])
		// Strip backticks and path components: "`models/foo.md`" → "foo"
		modelStr = strings.Trim(modelStr, "`")
		modelStr = strings.TrimSuffix(modelStr, ".md")
		if idx := strings.LastIndex(modelStr, "/"); idx != -1 {
			modelStr = modelStr[idx+1:]
		}
		v.ModelName = modelStr
	}

	// Parse question section
	v.Question = extractProbeSection(text, "Question")

	// Parse Model Impact section
	impactSection := extractProbeSection(text, "Model Impact")
	if impactSection == "" {
		return v
	}

	// Try structured format first: "**Verdict:** extends — description"
	if m := regexStructVerdict.FindStringSubmatch(impactSection); len(m) >= 3 {
		v.Verdict = strings.ToLower(strings.TrimSpace(m[1]))
		v.Details = strings.TrimSpace(m[2])
		return v
	}

	// Try checkbox format: "- [x] **Confirms** invariant: ..."
	if m := regexCheckConfirms.FindStringSubmatch(impactSection); len(m) >= 2 {
		v.Verdict = "confirms"
		v.Details = strings.TrimSpace(m[1])
		return v
	}
	if m := regexCheckContra.FindStringSubmatch(impactSection); len(m) >= 2 {
		v.Verdict = "contradicts"
		v.Details = strings.TrimSpace(m[1])
		return v
	}
	if m := regexCheckExtends.FindStringSubmatch(impactSection); len(m) >= 2 {
		v.Verdict = "extends"
		v.Details = strings.TrimSpace(m[1])
		return v
	}

	return v
}

// extractProbeSection extracts the content of a markdown section from probe content.
func extractProbeSection(content, sectionName string) string {
	pattern := regexp.MustCompile(`(?s)## ` + regexp.QuoteMeta(sectionName) + `\s*\n(.*?)(?:\n---\n|\n## |\z)`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// FindProbesForWorkspace scans .kb/models/*/probes/ for probe files
// modified after the workspace's spawn time. This identifies probes
// that were produced by the agent during its session.
func FindProbesForWorkspace(workspacePath, projectDir string) []ProbeVerdict {
	// Read spawn time to determine which probes are relevant
	spawnTimeFile := filepath.Join(workspacePath, ".spawn_time")
	spawnTimeBytes, err := os.ReadFile(spawnTimeFile)
	if err != nil {
		return nil // Can't determine relevance without spawn time
	}

	spawnTime, err := time.Parse(time.RFC3339, strings.TrimSpace(string(spawnTimeBytes)))
	if err != nil {
		return nil
	}

	// Scan all model probe directories
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	modelEntries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil
	}

	var verdicts []ProbeVerdict

	for _, modelEntry := range modelEntries {
		if !modelEntry.IsDir() {
			continue
		}

		probesDir := filepath.Join(modelsDir, modelEntry.Name(), "probes")
		probeEntries, err := os.ReadDir(probesDir)
		if err != nil {
			continue
		}

		for _, probeEntry := range probeEntries {
			if probeEntry.IsDir() || !strings.HasSuffix(probeEntry.Name(), ".md") {
				continue
			}

			info, err := probeEntry.Info()
			if err != nil {
				continue
			}

			// Only include probes modified after spawn time
			if !info.ModTime().After(spawnTime) {
				continue
			}

			probePath := filepath.Join(probesDir, probeEntry.Name())
			content, err := os.ReadFile(probePath)
			if err != nil {
				continue
			}

			verdict := ParseProbeVerdict(content)
			verdict.ProbePath = probePath

			// Use directory name as model name if not parsed from content
			if verdict.ModelName == "" {
				verdict.ModelName = modelEntry.Name()
			}

			// Only include probes that have a verdict
			if verdict.Verdict != "" {
				verdicts = append(verdicts, verdict)
			}
		}
	}

	return verdicts
}

// FormatProbeVerdicts formats probe verdicts for display in orch complete output.
// Returns empty string if no verdicts.
func FormatProbeVerdicts(verdicts []ProbeVerdict) string {
	if len(verdicts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n--- Probe Verdicts ---\n")

	for _, v := range verdicts {
		icon := verdictIcon(v.Verdict)
		sb.WriteString(fmt.Sprintf("%s  Model: %s\n", icon, v.ModelName))
		if v.Question != "" {
			q := v.Question
			if len(q) > 80 {
				q = q[:77] + "..."
			}
			sb.WriteString(fmt.Sprintf("   Claim tested: %s\n", q))
		}
		sb.WriteString(fmt.Sprintf("   Verdict: %s — %s\n", v.Verdict, v.Details))
		sb.WriteString(fmt.Sprintf("   Path: %s\n", v.ProbePath))
		sb.WriteString("\n")
	}

	sb.WriteString("Action required: Review verdicts and merge model updates if appropriate.\n")
	sb.WriteString("----------------------\n")

	return sb.String()
}

// verdictIcon returns an icon for the verdict type.
func verdictIcon(verdict string) string {
	switch verdict {
	case "confirms":
		return "\u2705" // check mark
	case "contradicts":
		return "\u274c" // cross mark
	case "extends":
		return "\U0001f4a1" // light bulb
	default:
		return "\u2753" // question mark
	}
}
