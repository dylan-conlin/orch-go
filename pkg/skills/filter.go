package skills

import "strings"

// SectionFilter configures which sections to keep when filtering skill content.
// Used for progressive skill disclosure: only include the phases/modes relevant
// to a specific spawn, reducing prompt token count.
type SectionFilter struct {
	Phases    []string // Include only these phases (empty = all)
	Mode      string   // Include only this mode (empty = all)
	SpawnMode string   // "interactive" or "autonomous" (empty = all)
}

// isEmpty returns true if the filter has no active constraints.
func (f *SectionFilter) IsEmpty() bool {
	return len(f.Phases) == 0 && f.Mode == "" && f.SpawnMode == ""
}

// FilterSkillSections removes @section-annotated sections that don't match the filter.
// Sections without annotations are always preserved.
// If filter is nil, returns content unchanged (backward compatible).
//
// Marker format: <!-- @section: key=value, key=value -->
// Close marker:  <!-- @/section -->
//
// Supported keys:
//   - phase: matches against filter.Phases (e.g., phase=investigation)
//   - mode: matches against filter.Mode (e.g., mode=tdd)
//   - spawn-mode: matches against filter.SpawnMode (e.g., spawn-mode=autonomous)
func FilterSkillSections(content string, filter *SectionFilter) string {
	if filter == nil || filter.IsEmpty() {
		return content
	}

	lines := strings.Split(content, "\n")
	var result []string
	skipping := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if skipping {
			if trimmed == "<!-- @/section -->" {
				skipping = false
			}
			continue
		}

		// Check for section open marker
		if strings.HasPrefix(trimmed, "<!-- @section:") && strings.HasSuffix(trimmed, "-->") {
			attrs := parseSectionAttrs(trimmed)
			if !sectionMatches(attrs, filter) {
				skipping = true
				continue
			}
			// Matching section — skip the marker line but include content
			continue
		}

		// Close marker for an included section — skip the marker itself
		if trimmed == "<!-- @/section -->" {
			continue
		}

		result = append(result, line)
	}

	// Collapse runs of 3+ consecutive blank lines to 2
	output := strings.Join(result, "\n")
	for strings.Contains(output, "\n\n\n\n") {
		output = strings.ReplaceAll(output, "\n\n\n\n", "\n\n\n")
	}

	return output
}

// parseSectionAttrs extracts key=value pairs from a section marker line.
// Example: "<!-- @section: phase=investigation, mode=tdd -->" returns
// map[string]string{"phase": "investigation", "mode": "tdd"}.
func parseSectionAttrs(marker string) map[string]string {
	attrs := make(map[string]string)

	start := strings.Index(marker, "<!-- @section:")
	if start == -1 {
		return attrs
	}
	end := strings.Index(marker, "-->")
	if end == -1 || end <= start {
		return attrs
	}

	inner := marker[start+len("<!-- @section:") : end]
	inner = strings.TrimSpace(inner)

	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			attrs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return attrs
}

// sectionMatches returns true if a section's attributes match the filter.
// A section matches when every attribute present in the section is accepted by
// the corresponding filter field. Attributes not mentioned in the section are
// not checked (i.e., a section with only phase=X is included regardless of Mode
// if the phase matches).
func sectionMatches(attrs map[string]string, filter *SectionFilter) bool {
	if phase, ok := attrs["phase"]; ok && len(filter.Phases) > 0 {
		found := false
		for _, p := range filter.Phases {
			if p == phase {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if mode, ok := attrs["mode"]; ok && filter.Mode != "" {
		if mode != filter.Mode {
			return false
		}
	}

	if spawnMode, ok := attrs["spawn-mode"]; ok && filter.SpawnMode != "" {
		if spawnMode != filter.SpawnMode {
			return false
		}
	}

	return true
}
