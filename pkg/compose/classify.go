// Package compose implements brief composition — clustering briefs by content
// similarity and producing digest artifacts for orchestrator review.
//
// classify.go provides brief category classification. Briefs are categorized as
// "maintenance" or "knowledge" to control comprehension queue behavior:
// maintenance items bypass the queue, knowledge items enter it.
package compose

import "strings"

const (
	// CategoryMaintenance marks briefs that don't require human comprehension.
	// Bug fixes, test fixes, infra, config, cleanup — the fix is in the code.
	CategoryMaintenance = "maintenance"

	// CategoryKnowledge marks briefs that produce knowledge requiring human engagement.
	// Investigations, research, architectural decisions, new features with design choices.
	CategoryKnowledge = "knowledge"
)

// maintenanceTitleKeywords are title substrings that indicate maintenance work
// when the skill is "feature-impl" (which handles both maintenance and knowledge work).
var maintenanceTitleKeywords = []string{
	"fix", "test", "config", "infra", "cleanup", "lint", "wire", "plumb",
	"migrate", "update dep", "bump", "rename",
}

// maintenanceTaskRefactorKeywords indicate maintenance when skill is "feature-impl"
// and issue type is "task".
var maintenanceTaskRefactorKeywords = []string{
	"refactor", "extract", "move", "reorganize",
}

// ClassifyBriefCategory determines whether a brief represents maintenance or knowledge work.
//
// Classification rules (from design spec):
//   - MAINTENANCE when: issue type is "bug", skill is "systematic-debugging",
//     or skill is "feature-impl" with maintenance-indicating title keywords.
//   - KNOWLEDGE when: skill is "investigation", "research", or "architect",
//     issue type is "investigation", "question", or "experiment".
//   - Default: knowledge (false knowledge is safer than false maintenance).
func ClassifyBriefCategory(issueType, skill, title string) string {
	// All bugs are maintenance
	if issueType == "bug" {
		return CategoryMaintenance
	}

	// Debugging is always maintenance
	if skill == "systematic-debugging" {
		return CategoryMaintenance
	}

	// Knowledge-producing skills are always knowledge
	switch skill {
	case "investigation", "research", "architect":
		return CategoryKnowledge
	}

	// Knowledge-producing issue types are always knowledge
	switch issueType {
	case "investigation", "question", "experiment":
		return CategoryKnowledge
	}

	// feature-impl with maintenance title keywords
	if skill == "feature-impl" {
		lower := strings.ToLower(title)

		for _, kw := range maintenanceTitleKeywords {
			if strings.Contains(lower, kw) {
				return CategoryMaintenance
			}
		}

		// Task + refactor keywords
		if issueType == "task" {
			for _, kw := range maintenanceTaskRefactorKeywords {
				if strings.Contains(lower, kw) {
					return CategoryMaintenance
				}
			}
		}
	}

	// Default: knowledge (false knowledge is safer than false maintenance)
	return CategoryKnowledge
}
