// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MaxRecentProbes is the maximum number of recent probes to show in spawn context.
const MaxRecentProbes = 5

// ProbeFilenameFormat is the expected filename format for probe files.
// Example: 2026-02-08-check-session-lifecycle.md
const ProbeFilenameFormat = "YYYY-MM-DD-{slug}.md"

// probeEntry represents a probe file found in a model's probes/ directory.
type probeEntry struct {
	Path     string    // Full path to probe file
	Name     string    // Filename without extension
	ModTime  time.Time // Last modification time
	ModelDir string    // Parent model directory name
}

// ModelNameFromPath extracts the model name from a model file path.
// Given ".kb/models/spawn-architecture.md", returns "spawn-architecture".
// Given ".kb/models/spawn-architecture/probes/foo.md", returns "spawn-architecture".
func ModelNameFromPath(modelPath string) string {
	dir := filepath.Dir(modelPath)
	base := filepath.Base(modelPath)

	// If this is a .md file directly in models/
	if filepath.Base(dir) == "models" {
		return strings.TrimSuffix(base, ".md")
	}

	// If this is inside a model subdirectory (e.g., models/spawn-architecture/probes/)
	// Walk up to find the model name
	parts := strings.Split(filepath.Clean(modelPath), string(filepath.Separator))
	for i, part := range parts {
		if part == "models" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return strings.TrimSuffix(base, ".md")
}

// ProbesDirForModel returns the probes directory path for a given model file.
// Given "/path/.kb/models/spawn-architecture.md", returns "/path/.kb/models/spawn-architecture/probes".
func ProbesDirForModel(modelPath string) string {
	modelName := ModelNameFromPath(modelPath)
	modelsDir := filepath.Dir(modelPath)

	// If modelPath is a file directly in models/, the dir is already models/
	if filepath.Base(modelsDir) == "models" {
		return filepath.Join(modelsDir, modelName, "probes")
	}

	// If already inside a subdirectory, find models/ and build from there
	parts := strings.Split(filepath.Clean(modelPath), string(filepath.Separator))
	for i, part := range parts {
		if part == "models" && i+1 < len(parts) {
			modelsBase := string(filepath.Separator) + filepath.Join(parts[:i+1]...)
			return filepath.Join(modelsBase, parts[i+1], "probes")
		}
	}

	return filepath.Join(modelsDir, modelName, "probes")
}

// ProbeFilePath generates the full path for a new probe file.
// Format: .kb/models/{model-name}/probes/YYYY-MM-DD-{slug}.md
func ProbeFilePath(modelPath, slug string) string {
	probesDir := ProbesDirForModel(modelPath)
	datestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.md", datestamp, slug)
	return filepath.Join(probesDir, filename)
}

// ListRecentProbes returns the most recent probes for a model, sorted by modification time (newest first).
// Returns at most maxProbes entries. Returns nil if the probes directory doesn't exist or is empty.
func ListRecentProbes(modelPath string, maxProbes int) []probeEntry {
	probesDir := ProbesDirForModel(modelPath)

	entries, err := os.ReadDir(probesDir)
	if err != nil {
		return nil
	}

	var probes []probeEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		// Skip .gitkeep
		if name == ".gitkeep" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		probes = append(probes, probeEntry{
			Path:     filepath.Join(probesDir, name),
			Name:     strings.TrimSuffix(name, ".md"),
			ModTime:  info.ModTime(),
			ModelDir: ModelNameFromPath(modelPath),
		})
	}

	// Sort by modification time, newest first
	sort.Slice(probes, func(i, j int) bool {
		return probes[i].ModTime.After(probes[j].ModTime)
	})

	if maxProbes > 0 && len(probes) > maxProbes {
		probes = probes[:maxProbes]
	}

	return probes
}

// FormatProbesForSpawn formats a list of probes for inclusion in spawn context.
// Returns empty string if no probes exist.
func FormatProbesForSpawn(probes []probeEntry) string {
	if len(probes) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("  - Recent Probes:\n")
	for _, p := range probes {
		// Format: "    - YYYY-MM-DD-slug (See: /path/to/probe.md)"
		sb.WriteString(fmt.Sprintf("    - %s\n", p.Name))
		sb.WriteString(fmt.Sprintf("      See: %s\n", p.Path))
	}
	return sb.String()
}

// EnsureProbesDir creates the probes directory for a model if it doesn't exist.
// Returns the probes directory path.
func EnsureProbesDir(modelPath string) (string, error) {
	probesDir := ProbesDirForModel(modelPath)
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create probes directory: %w", err)
	}
	return probesDir, nil
}

// DefaultProbeTemplate is a minimal placeholder probe template.
// The full template is expected to come from kb-cli (orch-go-xxa6e).
const DefaultProbeTemplate = `# Probe: {title}

**Model:** {model-name}
**Date:** {date}
**Status:** Active

---

## Question

[What specific claim or invariant from the model are you testing?]

---

## What I Tested

[Command run, code examined, or experiment performed — not just code review]

` + "```bash" + `
# Actual command(s) run
` + "```" + `

---

## What I Observed

[Actual output, behavior, or evidence gathered]

---

## Model Impact

- [ ] **Confirms** invariant: [which one]
- [ ] **Contradicts** invariant: [which one] — [what's actually true]
- [ ] **Extends** model with: [new finding not covered by existing model]

---

## Notes

[Any additional context, caveats, or follow-up questions]
`
