// Package verify provides verification helpers for agent completion.
package verify

import "strings"

// SkillClass groups skills by whether they are expected to produce code.
type SkillClass string

const (
	// SkillClassCode is for skills that primarily produce code changes.
	SkillClassCode SkillClass = "code"
	// SkillClassKnowledge is for skills that primarily produce knowledge artifacts.
	SkillClassKnowledge SkillClass = "knowledge"
)

var skillClassCodeSkills = map[string]struct{}{
	"feature-impl":         {},
	"systematic-debugging": {},
	"reliability-testing":  {},
}

var skillClassKnowledgeSkills = map[string]struct{}{
	"investigation":  {},
	"architect":      {},
	"design-session": {},
	"research":       {},
	"codebase-audit": {},
	"issue-creation": {},
	"writing-skills": {},
}

// SkillClassForName classifies a skill as code-producing or knowledge-producing.
// Unknown skills default to code-producing as a conservative fallback.
func SkillClassForName(skillName string) SkillClass {
	normalized := strings.ToLower(strings.TrimSpace(skillName))

	if _, ok := skillClassKnowledgeSkills[normalized]; ok {
		return SkillClassKnowledge
	}

	if _, ok := skillClassCodeSkills[normalized]; ok {
		return SkillClassCode
	}

	return SkillClassCode
}
