package tree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildKnowledgeTree(t *testing.T) {
	// Use the actual .kb directory for testing
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Navigate up to project root
	projectRoot := filepath.Join(cwd, "../..")
	kbDir := filepath.Join(projectRoot, ".kb")

	// Skip test if .kb directory doesn't exist
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		t.Skip(".kb directory not found, skipping test")
	}

	opts := TreeOptions{
		Depth:  2,
		Format: "text",
	}

	root, _, err := BuildKnowledgeTree(kbDir, opts)
	if err != nil {
		t.Fatalf("BuildKnowledgeTree failed: %v", err)
	}

	if root == nil {
		t.Fatal("Expected non-nil root node")
	}

	if root.Type != NodeTypeCluster {
		t.Errorf("Expected root type to be cluster, got %s", root.Type)
	}

	if len(root.Children) == 0 {
		t.Error("Expected root to have children")
	}
}

func TestRenderTree(t *testing.T) {
	root := &KnowledgeNode{
		ID:    "root",
		Type:  NodeTypeCluster,
		Title: "test tree",
		Children: []*KnowledgeNode{
			{
				ID:     "cluster1",
				Type:   NodeTypeCluster,
				Title:  "Test Cluster",
				Status: StatusComplete,
				Children: []*KnowledgeNode{
					{
						ID:     "inv1",
						Type:   NodeTypeInvestigation,
						Title:  "Test Investigation",
						Path:   ".kb/investigations/test.md",
						Status: StatusComplete,
					},
				},
			},
		},
	}

	opts := TreeOptions{
		Format: "text",
		Depth:  0,
	}

	output, err := RenderTree(root, opts, nil)
	if err != nil {
		t.Fatalf("RenderTree failed: %v", err)
	}

	// Root title is in the header
	if !strings.Contains(output, "knowledge tree") {
		t.Error("Expected output to contain knowledge tree header")
	}

	if !strings.Contains(output, "Test Cluster") {
		t.Error("Expected output to contain cluster title")
	}
}

func TestRenderJSON(t *testing.T) {
	root := &KnowledgeNode{
		ID:    "root",
		Type:  NodeTypeCluster,
		Title: "test tree",
		Children: []*KnowledgeNode{
			{
				ID:     "inv1",
				Type:   NodeTypeInvestigation,
				Title:  "Test Investigation",
				Status: StatusComplete,
			},
		},
	}

	opts := TreeOptions{
		Format: "json",
	}

	output, err := RenderTree(root, opts, nil)
	if err != nil {
		t.Fatalf("RenderTree failed: %v", err)
	}

	if !strings.Contains(output, `"Type"`) || !strings.Contains(output, `"cluster"`) {
		t.Error("Expected JSON output to contain Type field with cluster value")
	}

	if !strings.Contains(output, `"test tree"`) {
		t.Error("Expected JSON output to contain title")
	}
}

func TestParseInvestigation(t *testing.T) {
	// Create a temporary investigation file
	tmpDir := t.TempDir()
	invPath := filepath.Join(tmpDir, "test-investigation.md")

	content := `# Investigation: Test Investigation

**Status:** Complete
**Started:** 2026-02-15

**Prior-Work:**

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/other.md | synthesizes | yes | None |

## Findings

Test findings here.
`

	if err := os.WriteFile(invPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	node, rels, err := parseInvestigation(invPath)
	if err != nil {
		t.Fatalf("parseInvestigation failed: %v", err)
	}

	if node == nil {
		t.Fatal("Expected non-nil node")
	}

	if node.Title != "Test Investigation" {
		t.Errorf("Expected title 'Test Investigation', got %q", node.Title)
	}

	if node.Type != NodeTypeInvestigation {
		t.Errorf("Expected type investigation, got %s", node.Type)
	}

	if node.Status != StatusComplete {
		t.Errorf("Expected status complete, got %s", node.Status)
	}

	// Should have 1 relationship (table header row should be skipped)
	if len(rels) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(rels))
	}

	// After Phase 1b changes, Prior-Work relationships are reversed:
	// From = parent (Prior-Work source), To = child (this investigation)
	if !strings.HasSuffix(rels[0].From, ".kb/investigations/other.md") {
		t.Errorf("Expected relationship From to end with .kb/investigations/other.md, got %s", rels[0].From)
	}

	if rels[0].To != invPath {
		t.Errorf("Expected relationship To to be %s, got %s", invPath, rels[0].To)
	}

	if rels[0].RelationType != "synthesizes" {
		t.Errorf("Expected relationship type 'synthesizes', got %s", rels[0].RelationType)
	}
}
