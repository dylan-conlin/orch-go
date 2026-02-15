package tree

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectClusters detects clusters from filesystem organization
func DetectClusters(kbDir string, nodes []*KnowledgeNode, relationships []Relationship, opts TreeOptions) ([]*Cluster, error) {
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

	// 2. Add a "models" cluster for all model nodes
	clusters["models"] = &Cluster{
		Name:  "models",
		Nodes: []*KnowledgeNode{},
	}

	// 3. Add a "decisions" cluster for all decision nodes
	clusters["decisions"] = &Cluster{
		Name:  "decisions",
		Nodes: []*KnowledgeNode{},
	}

	// 4. Assign nodes to clusters based on their path and type
	for _, node := range nodes {
		// Check if node is a model
		if node.Type == NodeTypeModel {
			clusters["models"].Nodes = append(clusters["models"].Nodes, node)
			continue
		}

		// Check if node is a decision
		if node.Type == NodeTypeDecision {
			clusters["decisions"].Nodes = append(clusters["decisions"].Nodes, node)
			continue
		}

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

	// Convert map to slice and detect health smells
	var result []*Cluster
	for _, cluster := range clusters {
		if len(cluster.Nodes) > 0 {
			// Detect health smells for this cluster
			cluster.Smells = DetectHealthSmells(cluster, kbDir)

			// If SmellsOnly flag is set, only include clusters with smells
			if opts.SmellsOnly && len(cluster.Smells) == 0 {
				continue
			}

			result = append(result, cluster)
		}
	}

	return result, nil
}

// BuildRelationshipGraph builds a graph of nodes connected by relationships
func BuildRelationshipGraph(nodes []*KnowledgeNode, relationships []Relationship) map[string][]*KnowledgeNode {
	graph := make(map[string][]*KnowledgeNode)
	nodeMap := make(map[string]*KnowledgeNode)

	// Index nodes by their path AND by their ID (for models, ID is directory, Path is .md file)
	for _, node := range nodes {
		nodeMap[node.Path] = node
		if node.ID != node.Path {
			nodeMap[node.ID] = node
		}
	}

	// Build parent-child relationships
	for _, rel := range relationships {
		parent := findNodeByPath(nodeMap, rel.From)
		child := findNodeByPath(nodeMap, rel.To)

		if parent != nil && child != nil {
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

// findNodeByPath finds a node by path, handling both exact matches and directory matches
func findNodeByPath(nodeMap map[string]*KnowledgeNode, path string) *KnowledgeNode {
	// Try exact match first
	if node, ok := nodeMap[path]; ok {
		return node
	}

	// For directory paths (like .kb/models/completion-verification/), try to match by ID
	// Models use directory as ID, so check if any node's ID matches this path
	for _, node := range nodeMap {
		if node.ID == path {
			return node
		}
		// Also try matching directory to directory
		if strings.HasSuffix(path, "/") && node.Type == NodeTypeModel {
			if strings.HasSuffix(node.ID, "/") && node.ID == path {
				return node
			}
			// Try without trailing slash
			if node.ID == strings.TrimSuffix(path, "/") {
				return node
			}
		}
	}

	return nil
}
