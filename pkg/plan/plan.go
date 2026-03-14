// Package plan provides plan parsing and types shared between cmd/orch and pkg/daemon.
package plan

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// File represents a parsed .kb/plans/ artifact.
type File struct {
	Title        string
	Date         string
	Status       string // active, completed, superseded, draft
	Owner        string
	Filename     string
	Projects     []string
	SupersededBy string
	Phases       []Phase
}

// Phase represents a phase within a plan.
type Phase struct {
	Name      string
	Goal      string
	DependsOn string
	BeadsIDs  []string
}

// ScanDir reads all .md files from the plans directory and parses them.
func ScanDir(dir string) ([]File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var plans []File
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		plan := ParseContent(string(content), entry.Name())
		plans = append(plans, plan)
	}

	return plans, nil
}

// ParseContent extracts metadata and phases from a plan markdown file.
func ParseContent(content, filename string) File {
	plan := File{
		Filename: filename,
	}

	lines := strings.Split(content, "\n")

	var currentPhase *Phase
	inPhases := false
	statusFound := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Title: "# Plan: Title Here" or "# Coordination Plan: Title Here"
		if strings.HasPrefix(trimmed, "# Plan: ") {
			plan.Title = strings.TrimPrefix(trimmed, "# Plan: ")
			continue
		}
		if strings.HasPrefix(trimmed, "# Coordination Plan: ") {
			plan.Title = strings.TrimPrefix(trimmed, "# Coordination Plan: ")
			continue
		}

		// Metadata fields (only parse plan-level Status before Phases section)
		if strings.HasPrefix(trimmed, "**Date:**") && !inPhases {
			plan.Date = extractMetaValue(trimmed, "**Date:**")
			continue
		}
		if strings.HasPrefix(trimmed, "**Status:**") && !inPhases && !statusFound {
			plan.Status = strings.ToLower(extractMetaValue(trimmed, "**Status:**"))
			statusFound = true
			continue
		}
		if strings.HasPrefix(trimmed, "**Owner:**") {
			plan.Owner = extractMetaValue(trimmed, "**Owner:**")
			continue
		}
		if strings.HasPrefix(trimmed, "**Projects:**") {
			val := extractMetaValue(trimmed, "**Projects:**")
			if val != "" {
				for _, p := range strings.Split(val, ",") {
					p = strings.TrimSpace(p)
					if p != "" {
						plan.Projects = append(plan.Projects, p)
					}
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "**Superseded-By:**") {
			plan.SupersededBy = extractMetaValue(trimmed, "**Superseded-By:**")
			continue
		}

		// Phases section
		if trimmed == "## Phases" {
			inPhases = true
			continue
		}
		// End of phases section (next ## heading)
		if inPhases && strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			if currentPhase != nil {
				plan.Phases = append(plan.Phases, *currentPhase)
				currentPhase = nil
			}
			inPhases = false
			continue
		}

		if !inPhases {
			continue
		}

		// Phase heading: "### Phase N: Name"
		if strings.HasPrefix(trimmed, "### Phase ") {
			if currentPhase != nil {
				plan.Phases = append(plan.Phases, *currentPhase)
			}
			// Extract name after "### Phase N: "
			name := trimmed
			if idx := strings.Index(trimmed, ": "); idx >= 0 {
				name = trimmed[idx+2:]
			}
			currentPhase = &Phase{Name: name}
			continue
		}

		if currentPhase == nil {
			continue
		}

		// Phase metadata
		if strings.HasPrefix(trimmed, "**Goal:**") {
			currentPhase.Goal = extractMetaValue(trimmed, "**Goal:**")
		}
		if strings.HasPrefix(trimmed, "**Depends on:**") {
			currentPhase.DependsOn = extractMetaValue(trimmed, "**Depends on:**")
		}
		if strings.HasPrefix(trimmed, "**Beads:**") {
			currentPhase.BeadsIDs = ParseBeadsLine(trimmed)
		}
	}

	// Don't forget the last phase
	if currentPhase != nil {
		plan.Phases = append(plan.Phases, *currentPhase)
	}

	return plan
}

// extractMetaValue extracts the value from a "**Key:** value" line.
func extractMetaValue(line, prefix string) string {
	val := strings.TrimPrefix(line, prefix)
	return strings.TrimSpace(val)
}

// ParseBeadsLine extracts beads IDs from a "**Beads:** id1, id2" line.
func ParseBeadsLine(line string) []string {
	val := extractMetaValue(line, "**Beads:**")
	if val == "" || val == "none" {
		return nil
	}

	var ids []string
	for _, id := range strings.Split(val, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// FilterByStatus returns plans matching the given status.
func FilterByStatus(plans []File, status string) []File {
	var result []File
	for _, p := range plans {
		if p.Status == status {
			result = append(result, p)
		}
	}
	return result
}

// FindBySlug finds a plan whose filename contains the slug.
func FindBySlug(plans []File, slug string) *File {
	for i, p := range plans {
		if strings.Contains(p.Filename, slug) {
			return &plans[i]
		}
	}
	return nil
}

// CollectAllBeadsIDs gathers all beads IDs from all phases of a plan.
func CollectAllBeadsIDs(p *File) []string {
	var ids []string
	for _, phase := range p.Phases {
		ids = append(ids, phase.BeadsIDs...)
	}
	return ids
}

// IsHydrated returns true if at least one phase has beads IDs.
func (p *File) IsHydrated() bool {
	for _, phase := range p.Phases {
		if len(phase.BeadsIDs) > 0 {
			return true
		}
	}
	return false
}

// ExtractSlugFromFilename extracts the slug from a plan filename.
// "2026-03-11-gate-signal-vs-noise.md" → "gate-signal-vs-noise"
func ExtractSlugFromFilename(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	dateRe := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
	name = dateRe.ReplaceAllString(name, "")
	return name
}

// ParseDependsOn extracts 0-indexed phase numbers from a "Depends on" field.
// Handles: "Nothing", "none", "Phase 1", "Phase 1 (extra text)", "Phases 1-3", "Phase 1, Phase 3"
func ParseDependsOn(dep string) []int {
	dep = strings.TrimSpace(dep)
	lower := strings.ToLower(dep)

	if lower == "" || lower == "nothing" || lower == "none" || strings.HasPrefix(lower, "nothing") {
		return nil
	}

	var result []int

	// Match "Phases N-M" range pattern (only first range)
	rangeRe := regexp.MustCompile(`(?i)phases?\s+(\d+)\s*-\s*(\d+)`)
	if m := rangeRe.FindStringSubmatch(dep); len(m) == 3 {
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		for i := start; i <= end; i++ {
			result = append(result, i-1) // convert to 0-indexed
		}
		return result
	}

	// Match individual "Phase N" references
	phaseRe := regexp.MustCompile(`(?i)phase\s+(\d+)`)
	matches := phaseRe.FindAllStringSubmatch(dep, -1)
	for _, m := range matches {
		n, _ := strconv.Atoi(m[1])
		result = append(result, n-1) // convert to 0-indexed
	}

	return result
}
