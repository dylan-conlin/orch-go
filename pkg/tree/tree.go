package tree

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// BuildKnowledgeTree builds the full knowledge tree from .kb/ directory
func BuildKnowledgeTree(kbDir string, opts TreeOptions) (*KnowledgeNode, []*Cluster, error) {
	root := &KnowledgeNode{
		ID:       "root",
		Type:     NodeTypeCluster,
		Title:    "orch-go knowledge",
		Children: []*KnowledgeNode{},
	}

	// 1. Parse all artifacts
	var allNodes []*KnowledgeNode
	var allRelationships []Relationship

	// Parse investigations
	invDir := filepath.Join(kbDir, "investigations")
	if _, err := os.Stat(invDir); err == nil {
		invNodes, invRels, err := ParseInvestigations(invDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse investigations: %w", err)
		}
		allNodes = append(allNodes, invNodes...)
		allRelationships = append(allRelationships, invRels...)
	}

	// Parse decisions
	decDir := filepath.Join(kbDir, "decisions")
	if _, err := os.Stat(decDir); err == nil {
		decNodes, decRels, err := ParseDecisions(decDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse decisions: %w", err)
		}
		allNodes = append(allNodes, decNodes...)
		allRelationships = append(allRelationships, decRels...)
	}

	// Parse models
	modelDir := filepath.Join(kbDir, "models")
	if _, err := os.Stat(modelDir); err == nil {
		modelNodes, modelRels, err := ParseModels(modelDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse models: %w", err)
		}
		allNodes = append(allNodes, modelNodes...)
		allRelationships = append(allRelationships, modelRels...)
	}

	// Parse guides
	guideDir := filepath.Join(kbDir, "guides")
	if _, err := os.Stat(guideDir); err == nil {
		guideNodes, err := ParseGuides(guideDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse guides: %w", err)
		}
		allNodes = append(allNodes, guideNodes...)
	}

	// 2. Build relationship graph
	BuildRelationshipGraph(allNodes, allRelationships)

	// 3. Detect clusters
	clusters, err := DetectClusters(kbDir, allNodes, allRelationships, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect clusters: %w", err)
	}

	// 4. Filter by cluster if specified
	if opts.ClusterFilter != "" {
		for _, cluster := range clusters {
			if cluster.Name == opts.ClusterFilter {
				clusterNode := &KnowledgeNode{
					ID:       cluster.Name,
					Type:     NodeTypeCluster,
					Title:    cluster.Name,
					Children: buildClusterTree(cluster),
					Metadata: map[string]interface{}{
						"smells": cluster.Smells,
					},
				}
				root.Children = append(root.Children, clusterNode)
				return root, clusters, nil
			}
		}
		return nil, nil, fmt.Errorf("cluster %q not found", opts.ClusterFilter)
	}

	// 5. Add all clusters to root
	for _, cluster := range clusters {
		clusterNode := &KnowledgeNode{
			ID:       cluster.Name,
			Type:     NodeTypeCluster,
			Title:    cluster.Name,
			Children: buildClusterTree(cluster),
			Metadata: map[string]interface{}{
				"smells": cluster.Smells,
			},
		}
		root.Children = append(root.Children, clusterNode)
	}

	return root, clusters, nil
}

// buildClusterTree builds a tree for a cluster's nodes using relationship graph
func buildClusterTree(cluster *Cluster) []*KnowledgeNode {
	// Find root nodes (nodes that are not children of other nodes within this cluster)
	childMap := make(map[string]bool)
	nodeMap := make(map[string]*KnowledgeNode)

	// Index nodes in this cluster
	for _, node := range cluster.Nodes {
		nodeMap[node.ID] = node
	}

	// Mark all children (nodes that have parents in this cluster)
	for _, node := range cluster.Nodes {
		for _, child := range node.Children {
			// Only mark as child if the child is also in this cluster
			if _, inCluster := nodeMap[child.ID]; inCluster {
				childMap[child.ID] = true
			}
		}
	}

	// Collect root nodes (nodes without parents in this cluster)
	var rootNodes []*KnowledgeNode
	for _, node := range cluster.Nodes {
		if !childMap[node.ID] {
			// This is a root node - it has no parent in this cluster
			// Clone the node to avoid modifying the original and filter children
			clonedNode := cloneNodeForTree(node, nodeMap)
			rootNodes = append(rootNodes, clonedNode)
		}
	}

	// If no root nodes found (all nodes are in cycles or have external parents),
	// return all nodes with cycle-safe traversal
	if len(rootNodes) == 0 {
		for _, node := range cluster.Nodes {
			clonedNode := cloneNodeForTree(node, nodeMap)
			rootNodes = append(rootNodes, clonedNode)
		}
	}

	return rootNodes
}

// cloneNodeForTree clones a node and filters children to only include those in the cluster
// Also implements cycle detection by tracking visited nodes
func cloneNodeForTree(node *KnowledgeNode, clusterNodes map[string]*KnowledgeNode) *KnowledgeNode {
	visited := make(map[string]bool)
	return cloneNodeRecursive(node, clusterNodes, visited)
}

// cloneNodeRecursive clones a node recursively with cycle detection
func cloneNodeRecursive(node *KnowledgeNode, clusterNodes map[string]*KnowledgeNode, visited map[string]bool) *KnowledgeNode {
	// Detect cycle - if we've already visited this node in this path, stop
	if visited[node.ID] {
		// Return a shallow clone without children to break the cycle
		return &KnowledgeNode{
			ID:       node.ID,
			Type:     node.Type,
			Title:    node.Title,
			Path:     node.Path,
			Status:   node.Status,
			Date:     node.Date,
			Children: nil, // Break cycle by not including children
			Metadata: node.Metadata,
		}
	}

	// Mark this node as visited in current path
	visited[node.ID] = true
	defer func() {
		// Unmark when we return (for backtracking)
		delete(visited, node.ID)
	}()

	// Clone the node
	cloned := &KnowledgeNode{
		ID:       node.ID,
		Type:     node.Type,
		Title:    node.Title,
		Path:     node.Path,
		Status:   node.Status,
		Date:     node.Date,
		Children: []*KnowledgeNode{},
		Metadata: node.Metadata,
	}

	// Include ALL children (not filtered by cluster) to show cross-cluster relationships
	// like investigations → decisions → issues
	for _, child := range node.Children {
		clonedChild := cloneNodeRecursive(child, clusterNodes, visited)
		cloned.Children = append(cloned.Children, clonedChild)
	}

	return cloned
}

// BuildWorkTree builds the work view tree (issues as primary nodes)
func BuildWorkTree(kbDir string, projectDir string, opts TreeOptions) ([]*KnowledgeNode, error) {
	// Initialize beads client (CLI client for simplicity)
	var client beads.BeadsClient
	client = beads.NewCLIClient()

	// Get all issues
	issues, err := client.List(&beads.ListArgs{})
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Parse all knowledge artifacts to build a reference map
	// This allows us to find which artifacts issues reference
	artifactMap := make(map[string]*KnowledgeNode)

	// Parse investigations
	invDir := filepath.Join(kbDir, "investigations")
	if invNodes, _, err := ParseInvestigations(invDir); err == nil {
		for _, node := range invNodes {
			// Index by filename
			filename := filepath.Base(node.Path)
			artifactMap[filename] = node
			artifactMap[node.Path] = node
		}
	}

	// Parse decisions
	decDir := filepath.Join(kbDir, "decisions")
	if decNodes, _, err := ParseDecisions(decDir); err == nil {
		for _, node := range decNodes {
			filename := filepath.Base(node.Path)
			artifactMap[filename] = node
			artifactMap[node.Path] = node
		}
	}

	// Parse models
	modelDir := filepath.Join(kbDir, "models")
	if modelNodes, _, err := ParseModels(modelDir); err == nil {
		for _, node := range modelNodes {
			artifactMap[node.ID] = node
			artifactMap[node.Path] = node
		}
	}

	// Convert beads issues to KnowledgeNodes and link to artifacts
	var issueNodes []*KnowledgeNode
	for _, issue := range issues {
		issueNode := convertIssueToNode(&issue)

		// Parse issue description to find references to knowledge artifacts
		// Look for file paths in description
		linkedArtifacts := findReferencedArtifacts(issue.Description, artifactMap)
		issueNode.Children = linkedArtifacts

		issueNodes = append(issueNodes, issueNode)
	}

	return issueNodes, nil
}

// convertIssueToNode converts a beads Issue to a KnowledgeNode
func convertIssueToNode(issue *beads.Issue) *KnowledgeNode {
	status := StatusOpen
	switch issue.Status {
	case "closed":
		status = StatusClosed
	case "in_progress":
		status = StatusInProgress
	case "open":
		if hasLabel(issue.Labels, "triage:review") {
			status = StatusTriage
		} else {
			status = StatusOpen
		}
	}

	return &KnowledgeNode{
		ID:       issue.ID,
		Type:     NodeTypeIssue,
		Title:    issue.Title,
		Path:     issue.ID, // Use ID as path for issues
		Status:   status,
		Children: []*KnowledgeNode{},
		Metadata: map[string]interface{}{
			"priority": issue.Priority,
			"type":     issue.IssueType,
		},
	}
}

// hasLabel checks if a label exists in the labels slice
func hasLabel(labels []string, target string) bool {
	for _, label := range labels {
		if label == target {
			return true
		}
	}
	return false
}

// findReferencedArtifacts finds knowledge artifacts referenced in text
func findReferencedArtifacts(text string, artifactMap map[string]*KnowledgeNode) []*KnowledgeNode {
	var artifacts []*KnowledgeNode
	seen := make(map[string]bool)

	// Look for .kb/ paths in the text
	// Match patterns like: .kb/investigations/2026-02-14-inv-something.md
	// or decisions/2026-02-14-something.md
	pathRegex := regexp.MustCompile(`(?:\.kb/)?(?:investigations|decisions|models)/[^\s\)]+\.md`)
	matches := pathRegex.FindAllString(text, -1)

	for _, match := range matches {
		// Normalize the path
		normalized := match
		if !strings.HasPrefix(normalized, ".kb/") {
			normalized = ".kb/" + normalized
		}

		// Try to find the artifact
		if artifact, ok := artifactMap[normalized]; ok && !seen[artifact.ID] {
			artifacts = append(artifacts, artifact)
			seen[artifact.ID] = true
		} else {
			// Try just the filename
			filename := filepath.Base(match)
			if artifact, ok := artifactMap[filename]; ok && !seen[artifact.ID] {
				artifacts = append(artifacts, artifact)
				seen[artifact.ID] = true
			}
		}
	}

	return artifacts
}
