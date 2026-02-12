package verify

import "testing"

func TestSkillClassForName(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		want      SkillClass
	}{
		{"feature impl is code", "feature-impl", SkillClassCode},
		{"systematic debugging is code", "systematic-debugging", SkillClassCode},
		{"reliability testing is code", "reliability-testing", SkillClassCode},
		{"investigation is knowledge", "investigation", SkillClassKnowledge},
		{"architect is knowledge", "architect", SkillClassKnowledge},
		{"design session is knowledge", "design-session", SkillClassKnowledge},
		{"research is knowledge", "research", SkillClassKnowledge},
		{"codebase audit is knowledge", "codebase-audit", SkillClassKnowledge},
		{"issue creation is knowledge", "issue-creation", SkillClassKnowledge},
		{"writing skills is knowledge", "writing-skills", SkillClassKnowledge},
		{"unknown defaults to code", "unknown-skill", SkillClassCode},
		{"empty defaults to code", "", SkillClassCode},
		{"case insensitive", "FEATURE-IMPL", SkillClassCode},
		{"trimmed case insensitive", "  Investigation  ", SkillClassKnowledge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SkillClassForName(tt.skillName)
			if got != tt.want {
				t.Errorf("SkillClassForName(%q) = %q, want %q", tt.skillName, got, tt.want)
			}
		})
	}
}
