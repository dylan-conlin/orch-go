package main

import (
	"testing"
)

func TestInferSkillFromIssueType(t *testing.T) {
	tests := []struct {
		name      string
		issueType string
		wantSkill string
	}{
		{
			name:      "bug maps to systematic-debugging (direct action)",
			issueType: "bug",
			wantSkill: "systematic-debugging",
		},
		{
			name:      "feature maps to feature-impl",
			issueType: "feature",
			wantSkill: "feature-impl",
		},
		{
			name:      "task maps to feature-impl",
			issueType: "task",
			wantSkill: "feature-impl",
		},
		{
			name:      "investigation maps to investigation",
			issueType: "investigation",
			wantSkill: "investigation",
		},
		{
			name:      "question maps to investigation",
			issueType: "question",
			wantSkill: "investigation",
		},
		{
			name:      "epic returns error - not spawnable",
			issueType: "epic",
			wantSkill: "",
		},
		{
			name:      "unknown type returns error",
			issueType: "unknown",
			wantSkill: "",
		},
		{
			name:      "empty type returns error",
			issueType: "",
			wantSkill: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSkill, err := InferSkillFromIssueType(tt.issueType)
			if tt.wantSkill == "" {
				if err == nil {
					t.Errorf("InferSkillFromIssueType(%q) expected error, got skill %q", tt.issueType, gotSkill)
				}
				return
			}
			if err != nil {
				t.Errorf("InferSkillFromIssueType(%q) unexpected error: %v", tt.issueType, err)
				return
			}
			if gotSkill != tt.wantSkill {
				t.Errorf("InferSkillFromIssueType(%q) = %q, want %q", tt.issueType, gotSkill, tt.wantSkill)
			}
		})
	}
}
