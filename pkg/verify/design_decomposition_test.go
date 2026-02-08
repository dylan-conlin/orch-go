package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractDesignActionItems(t *testing.T) {
	content := `# Design: Example

## Implementation Notes

### Data Requirements
- Capture attempt history

### Components to Build
1. WorkInProgressSection
2. IssueSidePanel

### API Changes
- Add /api/work-graph endpoint

## Out of Scope
- Ignore this item
`

	items := ExtractDesignActionItems(content)
	if len(items) != 4 {
		t.Fatalf("expected 4 action items, got %d", len(items))
	}

	texts := []string{
		"Capture attempt history",
		"WorkInProgressSection",
		"IssueSidePanel",
		"Add /api/work-graph endpoint",
	}

	for _, want := range texts {
		found := false
		for _, item := range items {
			if item.Text == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected action item %q not found", want)
		}
	}
}

func TestParseDesignDecompositionMetadata(t *testing.T) {
	content := `---
decomposed: true
decomposition_parent: orch-go-123
decomposition_issues:
  - orch-go-300
  - orch-go-200
---

# Design`

	meta := ParseDesignDecompositionMetadata(content)
	if !meta.Decomposed {
		t.Fatal("expected decomposed=true")
	}
	if meta.DecompositionParent != "orch-go-123" {
		t.Fatalf("expected decomposition_parent orch-go-123, got %q", meta.DecompositionParent)
	}
	if len(meta.DecompositionIssues) != 2 {
		t.Fatalf("expected 2 decomposition issues, got %d", len(meta.DecompositionIssues))
	}
	if meta.DecompositionIssues[0] != "orch-go-200" || meta.DecompositionIssues[1] != "orch-go-300" {
		t.Fatalf("expected sorted decomposition issues, got %v", meta.DecompositionIssues)
	}
}

func TestMarkDesignDocumentDecomposed_AddsFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "design.md")
	original := "# Design\n\n## Implementation Notes\n- Build component\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatalf("failed writing fixture: %v", err)
	}

	if err := MarkDesignDocumentDecomposed(path, "orch-go-555", []string{"orch-go-10", "orch-go-11"}); err != nil {
		t.Fatalf("MarkDesignDocumentDecomposed failed: %v", err)
	}

	updatedBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading updated file: %v", err)
	}
	updated := string(updatedBytes)

	if !strings.HasPrefix(updated, "---\n") {
		t.Fatalf("expected YAML frontmatter prefix, got:\n%s", updated)
	}
	if !strings.Contains(updated, "decomposed: true") {
		t.Fatalf("expected decomposed flag in frontmatter, got:\n%s", updated)
	}
	if !strings.Contains(updated, "decomposition_parent: orch-go-555") {
		t.Fatalf("expected decomposition_parent in frontmatter, got:\n%s", updated)
	}
	if !strings.Contains(updated, "- orch-go-10") || !strings.Contains(updated, "- orch-go-11") {
		t.Fatalf("expected decomposition issue IDs in frontmatter, got:\n%s", updated)
	}
	if !strings.Contains(updated, "# Design") {
		t.Fatalf("expected original markdown body preserved, got:\n%s", updated)
	}
}
