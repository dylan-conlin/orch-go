// Package daemon provides autonomous overnight processing capabilities.
// Architect escalation logic handles pre-spawn hotspot detection for skill routing.
// When feature-impl or systematic-debugging issues target hotspot areas,
// the daemon escalates them to architect for architectural review first.
package daemon

import (
	"strings"
)

// ArchitectEscalation contains the result of checking whether an implementation skill
// issue should be escalated to architect because it targets a hotspot area.
type ArchitectEscalation struct {
	// HotspotFile is the file path that triggered escalation.
	HotspotFile string
	// HotspotType is the type of hotspot matched (e.g., "fix-density", "bloat-size").
	HotspotType string
	// HotspotScore is the severity score of the matched hotspot.
	HotspotScore int
}

// isImplementationSkill returns true if the skill modifies code and should be
// escalated to architect when targeting hotspot areas.
func isImplementationSkill(skill string) bool {
	switch skill {
	case "feature-impl", "systematic-debugging":
		return true
	default:
		return false
	}
}

// FindMatchingHotspot checks if any inferred files match any hotspot.
// Unlike FindCriticalHotspot (which only checks bloat-size >1500),
// this matches any hotspot type and score.
// Returns the first matching hotspot, or nil if none found.
func FindMatchingHotspot(inferredFiles []string, hotspots []HotspotWarning) *HotspotWarning {
	if len(inferredFiles) == 0 || len(hotspots) == 0 {
		return nil
	}

	for _, file := range inferredFiles {
		for _, h := range hotspots {
			if matchesFilePath(file, h.Path) {
				match := h
				return &match
			}
		}
	}

	return nil
}

// PriorArchitectFinder searches for a closed architect issue that reviewed the given files.
// Returns the issue ID if found, empty string if no prior architect review exists.
// Used to skip daemon escalation when an architect has already reviewed the area.
type PriorArchitectFinder func(files []string) (string, error)

// CheckArchitectEscalation determines if an implementation skill issue should be
// escalated to architect because it targets a hotspot area.
//
// This implements Layer 2 of the hotspot enforcement system:
//   - Layer 1 (spawn gate): Blocks feature-impl/systematic-debugging for CRITICAL files (>1500 lines)
//   - Layer 2 (daemon escalation): Routes implementation skills to architect when targeting ANY hotspot
//   - Layer 3 (completion gate): Warns on additions >50 lines to files >800 lines
//
// If priorArchitectFinder is non-nil and finds a completed architect review covering the
// matched hotspot file, escalation is skipped (architect already reviewed the area).
//
// Returns nil if no escalation is needed (no target files inferred, no hotspots, skill is exempt,
// or prior architect review exists).
func CheckArchitectEscalation(issue *Issue, skill string, checker HotspotChecker, priorArchitectFinder PriorArchitectFinder) *ArchitectEscalation {
	if issue == nil || checker == nil {
		return nil
	}

	// Only escalate implementation skills
	if !isImplementationSkill(skill) {
		return nil
	}

	// Skip extraction issues (they're already handling hotspot files)
	if strings.HasPrefix(issue.Title, "Extract ") {
		return nil
	}

	// Skip issues with explicit skill:* label - user override takes precedence
	if InferSkillFromLabels(issue.Labels) != "" {
		return nil
	}

	// Infer target files from issue title/description
	files := InferTargetFilesFromIssue(issue)
	if len(files) == 0 {
		return nil
	}

	// Get hotspots for the project
	hotspots, err := checker.CheckHotspots("")
	if err != nil || len(hotspots) == 0 {
		return nil
	}

	// Check if any inferred file matches any hotspot
	match := FindMatchingHotspot(files, hotspots)
	if match == nil {
		return nil
	}

	// Check if a prior architect review already covers this hotspot area
	if priorArchitectFinder != nil {
		foundRef, findErr := priorArchitectFinder([]string{match.Path})
		if findErr == nil && foundRef != "" {
			return nil // Prior architect review exists — skip escalation
		}
	}

	return &ArchitectEscalation{
		HotspotFile:  match.Path,
		HotspotType:  match.Type,
		HotspotScore: match.Score,
	}
}
