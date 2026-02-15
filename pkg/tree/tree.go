package tree

import (
	"fmt"
	"os"
	"path/filepath"
)

// BuildKnowledgeTree builds the full knowledge tree from .kb/ directory
func BuildKnowledgeTree(kbDir string, opts TreeOptions) (*KnowledgeNode, error) {
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
			return nil, fmt.Errorf("failed to parse investigations: %w", err)
		}
		allNodes = append(allNodes, invNodes...)
		allRelationships = append(allRelationships, invRels...)
	}

	// Parse decisions
	decDir := filepath.Join(kbDir, "decisions")
	if _, err := os.Stat(decDir); err == nil {
		decNodes, decRels, err := ParseDecisions(decDir)
		if err != nil {
			return nil, fmt.Errorf("failed to parse decisions: %w", err)
		}
		allNodes = append(allNodes, decNodes...)
		allRelationships = append(allRelationships, decRels...)
	}

	// Parse models
	modelDir := filepath.Join(kbDir, "models")
	if _, err := os.Stat(modelDir); err == nil {
		modelNodes, modelRels, err := ParseModels(modelDir)
		if err != nil {
			return nil, fmt.Errorf("failed to parse models: %w", err)
		}
		allNodes = append(allNodes, modelNodes...)
		allRelationships = append(allRelationships, modelRels...)
	}

	// Parse guides
	guideDir := filepath.Join(kbDir, "guides")
	if _, err := os.Stat(guideDir); err == nil {
		guideNodes, err := ParseGuides(guideDir)
		if err != nil {
			return nil, fmt.Errorf("failed to parse guides: %w", err)
		}
		allNodes = append(allNodes, guideNodes...)
	}

	// 2. Build relationship graph
	BuildRelationshipGraph(allNodes, allRelationships)

	// 3. Detect clusters
	clusters, err := DetectClusters(kbDir, allNodes, allRelationships)
	if err != nil {
		return nil, fmt.Errorf("failed to detect clusters: %w", err)
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
				}
				root.Children = append(root.Children, clusterNode)
				return root, nil
			}
		}
		return nil, fmt.Errorf("cluster %q not found", opts.ClusterFilter)
	}

	// 5. Add all clusters to root
	for _, cluster := range clusters {
		clusterNode := &KnowledgeNode{
			ID:       cluster.Name,
			Type:     NodeTypeCluster,
			Title:    cluster.Name,
			Children: buildClusterTree(cluster),
		}
		root.Children = append(root.Children, clusterNode)
	}

	return root, nil
}

// buildClusterTree builds a tree for a cluster's nodes
func buildClusterTree(cluster *Cluster) []*KnowledgeNode {
	// Find root nodes (nodes that are not children of other nodes)
	childMap := make(map[string]bool)
	for _, node := range cluster.Nodes {
		for _, child := range node.Children {
			childMap[child.ID] = true
		}
	}

	var rootNodes []*KnowledgeNode
	for _, node := range cluster.Nodes {
		if !childMap[node.ID] {
			rootNodes = append(rootNodes, node)
		}
	}

	// If no root nodes found, return all nodes
	if len(rootNodes) == 0 {
		return cluster.Nodes
	}

	return rootNodes
}

// BuildWorkTree builds the work view tree (issues as primary nodes)
func BuildWorkTree(kbDir string, projectDir string, opts TreeOptions) ([]*KnowledgeNode, error) {
	// For Phase 1, we'll create a placeholder implementation
	// In Phase 2, this would integrate with beads to get actual issues

	// For now, return empty slice since beads integration is out of scope for Phase 1
	return []*KnowledgeNode{}, nil
}
