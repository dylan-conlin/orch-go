package claims

import (
	"testing"
)

func TestParseArchitectOutput_Valid(t *testing.T) {
	yaml := `
cluster_id: tc-displacement-governance
resolution_type: restructure
summary: "Add redirect hints and displacement tracking to deny hooks"

issues:
  - title: "Add redirect hints to deny hooks for governance-protected files"
    skill: feature-impl
    priority: 2
    claim_provenance:
      - AE-09
      - KA-10
    depends_on: []
    description: |
      Deny hooks currently block edits without suggesting where code should go.
      Add redirect hints to hook error messages.

  - title: "Implement displacement tracking metric"
    skill: feature-impl
    priority: 3
    claim_provenance:
      - MH-05
      - MH-07
    depends_on: [0]
    description: |
      Track whether denied edits land in architecturally correct locations.
`

	out, err := ParseArchitectOutput([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.ClusterID != "tc-displacement-governance" {
		t.Errorf("cluster_id = %s, want tc-displacement-governance", out.ClusterID)
	}
	if out.ResolutionType != "restructure" {
		t.Errorf("resolution_type = %s, want restructure", out.ResolutionType)
	}
	if out.Summary != "Add redirect hints and displacement tracking to deny hooks" {
		t.Errorf("summary = %s", out.Summary)
	}
	if len(out.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(out.Issues))
	}

	issue0 := out.Issues[0]
	if issue0.Title != "Add redirect hints to deny hooks for governance-protected files" {
		t.Errorf("issue[0].title = %s", issue0.Title)
	}
	if issue0.Skill != "feature-impl" {
		t.Errorf("issue[0].skill = %s", issue0.Skill)
	}
	if issue0.Priority != 2 {
		t.Errorf("issue[0].priority = %d", issue0.Priority)
	}
	if len(issue0.ClaimProvenance) != 2 || issue0.ClaimProvenance[0] != "AE-09" || issue0.ClaimProvenance[1] != "KA-10" {
		t.Errorf("issue[0].claim_provenance = %v", issue0.ClaimProvenance)
	}
	if len(issue0.DependsOn) != 0 {
		t.Errorf("issue[0].depends_on = %v, want empty", issue0.DependsOn)
	}

	issue1 := out.Issues[1]
	if len(issue1.DependsOn) != 1 || issue1.DependsOn[0] != 0 {
		t.Errorf("issue[1].depends_on = %v, want [0]", issue1.DependsOn)
	}
	if len(issue1.ClaimProvenance) != 2 {
		t.Errorf("issue[1].claim_provenance = %v", issue1.ClaimProvenance)
	}
}

func TestParseArchitectOutput_EmptyIssues(t *testing.T) {
	yaml := `
cluster_id: tc-test
resolution_type: defer
summary: "Not actionable now"
issues: []
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for empty issues, got nil")
	}
}

func TestParseArchitectOutput_InvalidYAML(t *testing.T) {
	_, err := ParseArchitectOutput([]byte(":::not yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseArchitectOutput_MissingClusterID(t *testing.T) {
	yaml := `
resolution_type: restructure
summary: "test"
issues:
  - title: "test issue"
    skill: feature-impl
    priority: 2
    claim_provenance: [A-01]
    depends_on: []
    description: "test"
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing cluster_id")
	}
}

func TestParseArchitectOutput_MissingResolutionType(t *testing.T) {
	yaml := `
cluster_id: tc-test
summary: "test"
issues:
  - title: "test issue"
    skill: feature-impl
    priority: 2
    claim_provenance: [A-01]
    depends_on: []
    description: "test"
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing resolution_type")
	}
}

func TestParseArchitectOutput_InvalidResolutionType(t *testing.T) {
	yaml := `
cluster_id: tc-test
resolution_type: unknown
summary: "test"
issues:
  - title: "test issue"
    skill: feature-impl
    priority: 2
    claim_provenance: [A-01]
    depends_on: []
    description: "test"
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid resolution_type")
	}
}

func TestParseArchitectOutput_CyclicDependency(t *testing.T) {
	yaml := `
cluster_id: tc-test
resolution_type: restructure
summary: "test"
issues:
  - title: "issue A"
    skill: feature-impl
    priority: 2
    claim_provenance: [A-01]
    depends_on: [1]
    description: "a"
  - title: "issue B"
    skill: feature-impl
    priority: 2
    claim_provenance: [B-01]
    depends_on: [0]
    description: "b"
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for cyclic dependency")
	}
}

func TestParseArchitectOutput_OutOfBoundsDependency(t *testing.T) {
	yaml := `
cluster_id: tc-test
resolution_type: strengthen
summary: "test"
issues:
  - title: "issue A"
    skill: feature-impl
    priority: 2
    claim_provenance: [A-01]
    depends_on: [5]
    description: "a"
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for out-of-bounds dependency")
	}
}

func TestParseArchitectOutput_IssueMissingTitle(t *testing.T) {
	yaml := `
cluster_id: tc-test
resolution_type: accept
summary: "test"
issues:
  - skill: feature-impl
    priority: 2
    claim_provenance: [A-01]
    depends_on: []
    description: "no title"
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestParseArchitectOutput_IssueMissingSkill(t *testing.T) {
	yaml := `
cluster_id: tc-test
resolution_type: accept
summary: "test"
issues:
  - title: "test"
    priority: 2
    claim_provenance: [A-01]
    depends_on: []
    description: "no skill"
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing skill")
	}
}

func TestParseArchitectOutput_IssueMissingProvenance(t *testing.T) {
	yaml := `
cluster_id: tc-test
resolution_type: accept
summary: "test"
issues:
  - title: "test"
    skill: feature-impl
    priority: 2
    claim_provenance: []
    depends_on: []
    description: "no provenance"
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for empty claim_provenance")
	}
}

func TestParseArchitectOutput_AllResolutionTypes(t *testing.T) {
	for _, rt := range []string{"restructure", "strengthen", "accept", "defer"} {
		yaml := `
cluster_id: tc-test
resolution_type: ` + rt + `
summary: "test"
issues:
  - title: "test"
    skill: feature-impl
    priority: 2
    claim_provenance: [A-01]
    depends_on: []
    description: "test"
`
		out, err := ParseArchitectOutput([]byte(yaml))
		if err != nil {
			t.Errorf("resolution_type %s: unexpected error: %v", rt, err)
		}
		if out.ResolutionType != rt {
			t.Errorf("resolution_type = %s, want %s", out.ResolutionType, rt)
		}
	}
}

func TestParseArchitectOutput_LoadFile(t *testing.T) {
	// LoadArchitectOutput delegates to ParseArchitectOutput after reading file.
	// Test with non-existent file.
	_, err := LoadArchitectOutput("/nonexistent/ARCHITECT_OUTPUT.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestParseArchitectOutput_DAGValid(t *testing.T) {
	// Valid DAG: 0 <- 1 <- 2 (chain)
	yaml := `
cluster_id: tc-test
resolution_type: restructure
summary: "test"
issues:
  - title: "issue 0"
    skill: feature-impl
    priority: 2
    claim_provenance: [A-01]
    depends_on: []
    description: "first"
  - title: "issue 1"
    skill: feature-impl
    priority: 3
    claim_provenance: [A-02]
    depends_on: [0]
    description: "second"
  - title: "issue 2"
    skill: investigation
    priority: 3
    claim_provenance: [A-03]
    depends_on: [1]
    description: "third"
`
	out, err := ParseArchitectOutput([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(out.Issues))
	}
}

func TestParseArchitectOutput_SelfDependency(t *testing.T) {
	yaml := `
cluster_id: tc-test
resolution_type: restructure
summary: "test"
issues:
  - title: "issue 0"
    skill: feature-impl
    priority: 2
    claim_provenance: [A-01]
    depends_on: [0]
    description: "self-dep"
`
	_, err := ParseArchitectOutput([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for self-dependency")
	}
}
