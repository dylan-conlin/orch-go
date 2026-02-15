package tree

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectClusters detects clusters from filesystem organization
func DetectClusters(kbDir string, nodes []*KnowledgeNode, relationships []Relationship) ([]*Cluster, error) {
	clusters := make(map[string]*Cluster)

	// 1. Filesystem-based clustering from investigations/synthesized/{cluster}/
	synthesizedDir := filepath.Join(kbDir, "investigations", "synthesized")
	if entries, err := os.ReadDir(synthesizedDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			clusterName := entry.Name()
			clusters[clusterName] = &Cluster{
				Name:  clusterName,
				Nodes: []*KnowledgeNode{},
			}
		}
	}

	// 2. Assign nodes to clusters based on their path
	for _, node := range nodes {
		// Check if node is in a synthesized subdirectory
		if strings.Contains(node.Path, "/investigations/synthesized/") {
			parts := strings.Split(node.Path, "/investigations/synthesized/")
			if len(parts) > 1 {
				clusterName := strings.Split(parts[1], "/")[0]
				if cluster, ok := clusters[clusterName]; ok {
					cluster.Nodes = append(cluster.Nodes, node)
					continue
				}
			}
		}

		// If not in a cluster, assign to "uncategorized"
		if _, ok := clusters["uncategorized"]; !ok {
			clusters["uncategorized"] = &Cluster{
				Name:  "uncategorized",
				Nodes: []*KnowledgeNode{},
			}
		}
		clusters["uncategorized"].Nodes = append(clusters["uncategorized"].Nodes, node)
	}

	// Convert map to slice
	var result []*Cluster
	for _, cluster := range clusters {
		if len(cluster.Nodes) > 0 {
			result = append(result, cluster)
		}
	}

	return result, nil
}

// BuildRelationshipGraph builds a graph of nodes connected by relationships
func BuildRelationshipGraph(nodes []*KnowledgeNode, relationships []Relationship) map[string][]*KnowledgeNode {
	graph := make(map[string][]*KnowledgeNode)
	nodeMap := make(map[string]*KnowledgeNode)

	// Index nodes by their path
	for _, node := range nodes {
		nodeMap[node.Path] = node
	}

	// Build parent-child relationships
	for _, rel := range relationships {
		parent, parentExists := nodeMap[rel.From]
		child, childExists := nodeMap[rel.To]

		if parentExists && childExists {
			// Add child to parent's children if not already there
			found := false
			for _, existingChild := range parent.Children {
				if existingChild.ID == child.ID {
					found = true
					break
				}
			}
			if !found {
				parent.Children = append(parent.Children, child)
			}

			// Track in graph
			graph[parent.ID] = append(graph[parent.ID], child)
		}
	}

	return graph
}
