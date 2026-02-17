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

// TestDeduplicationAcrossParents tests that when an investigation references multiple models
// in its Prior-Work table, it only appears once in the tree (not duplicated under each model)
func TestDeduplicationAcrossParents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two model files
	model1Dir := filepath.Join(tmpDir, ".kb", "models", "model1")
	model2Dir := filepath.Join(tmpDir, ".kb", "models", "model2")
	if err := os.MkdirAll(model1Dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(model2Dir, 0755); err != nil {
		t.Fatal(err)
	}

	model1Path := filepath.Join(tmpDir, ".kb", "models", "model1.md")
	model2Path := filepath.Join(tmpDir, ".kb", "models", "model2.md")

	model1Content := `# Model 1

This is the first model.
`
	model2Content := `# Model 2

This is the second model.
`

	if err := os.WriteFile(model1Path, []byte(model1Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(model2Path, []byte(model2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an investigation that references BOTH models in Prior-Work
	invDir := filepath.Join(tmpDir, ".kb", "investigations")
	if err := os.MkdirAll(invDir, 0755); err != nil {
		t.Fatal(err)
	}

	invPath := filepath.Join(invDir, "test-inv.md")
	invContent := `# Investigation: Shared Investigation

**Status:** Complete

**Prior-Work:**

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/model1.md | extends | yes | None |
| .kb/models/model2.md | extends | yes | None |

## Findings

This investigation extends both models.
`

	if err := os.WriteFile(invPath, []byte(invContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Build the tree
	kbDir := filepath.Join(tmpDir, ".kb")
	opts := TreeOptions{
		Format: "text",
		Depth:  0,
	}

	root, clusters, err := BuildKnowledgeTree(kbDir, opts)
	if err != nil {
		t.Fatalf("BuildKnowledgeTree failed: %v", err)
	}

	// Find the "models" cluster
	var modelsCluster *KnowledgeNode
	for _, child := range root.Children {
		if child.Title == "models" {
			modelsCluster = child
			break
		}
	}

	if modelsCluster == nil {
		t.Fatal("Expected to find 'models' cluster")
	}

	// Count how many times the investigation appears in the tree
	invCount := countNodeOccurrences(modelsCluster, "Shared Investigation")

	if invCount > 1 {
		t.Errorf("Investigation appears %d times in the tree, expected at most 1 (deduplication should prevent duplicates)", invCount)
		// Print the tree structure for debugging
		output, _ := RenderTree(root, opts, clusters)
		t.Logf("Tree structure:\n%s", output)
	}

	if invCount == 0 {
		t.Error("Investigation doesn't appear in the tree at all, expected 1 occurrence")
	}
}

// countNodeOccurrences recursively counts how many times a node with the given title appears in the tree
func countNodeOccurrences(node *KnowledgeNode, title string) int {
	count := 0
	if node.Title == title {
		count = 1
	}

	for _, child := range node.Children {
		count += countNodeOccurrences(child, title)
	}

	return count
}
