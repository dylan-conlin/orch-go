package tree

import (
	"sort"
	"strings"
	"time"
)

// SortClusters sorts clusters based on the specified sort mode
func SortClusters(clusters []*Cluster, mode SortMode) {
	switch mode {
	case SortModeRecency:
		sortClustersByRecency(clusters)
	case SortModeConnectivity:
		sortClustersByConnectivity(clusters)
	case SortModeAlphabetical:
		sortClustersAlphabetically(clusters)
	default:
		// Default to recency
		sortClustersByRecency(clusters)
	}
}

// SortNodes sorts nodes based on the specified sort mode
func SortNodes(nodes []*KnowledgeNode, mode SortMode) {
	switch mode {
	case SortModeRecency:
		sortNodesByRecency(nodes)
	case SortModeConnectivity:
		sortNodesByConnectivity(nodes)
	case SortModeAlphabetical:
		sortNodesAlphabetically(nodes)
	default:
		// Default to recency
		sortNodesByRecency(nodes)
	}

	// Recursively sort children
	for _, node := range nodes {
		if len(node.Children) > 0 {
			SortNodes(node.Children, mode)
		}
	}
}

// sortClustersByRecency sorts clusters by most recent node date
func sortClustersByRecency(clusters []*Cluster) {
	sort.Slice(clusters, func(i, j int) bool {
		iRecent := getMostRecentDate(clusters[i].Nodes)
		jRecent := getMostRecentDate(clusters[j].Nodes)
		return iRecent.After(jRecent)
	})
}

// sortClustersByConnectivity sorts clusters by total connectivity (sum of node connections)
func sortClustersByConnectivity(clusters []*Cluster) {
	sort.Slice(clusters, func(i, j int) bool {
		iConnectivity := getTotalConnectivity(clusters[i].Nodes)
		jConnectivity := getTotalConnectivity(clusters[j].Nodes)
		return iConnectivity > jConnectivity
	})
}

// sortClustersAlphabetically sorts clusters by name
func sortClustersAlphabetically(clusters []*Cluster) {
	sort.Slice(clusters, func(i, j int) bool {
		return strings.ToLower(clusters[i].Name) < strings.ToLower(clusters[j].Name)
	})
}

// sortNodesByRecency sorts nodes by date (most recent first)
func sortNodesByRecency(nodes []*KnowledgeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Date.After(nodes[j].Date)
	})
}

// sortNodesByConnectivity sorts nodes by connectivity (most connected first)
func sortNodesByConnectivity(nodes []*KnowledgeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		iConnectivity := getNodeConnectivity(nodes[i])
		jConnectivity := getNodeConnectivity(nodes[j])
		return iConnectivity > jConnectivity
	})
}

// sortNodesAlphabetically sorts nodes by title
func sortNodesAlphabetically(nodes []*KnowledgeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return strings.ToLower(nodes[i].Title) < strings.ToLower(nodes[j].Title)
	})
}

// getMostRecentDate finds the most recent date among nodes
func getMostRecentDate(nodes []*KnowledgeNode) time.Time {
	var mostRecent time.Time
	for _, node := range nodes {
		if node.Date.After(mostRecent) {
			mostRecent = node.Date
		}
		// Check children recursively
		childRecent := getMostRecentDate(node.Children)
		if childRecent.After(mostRecent) {
			mostRecent = childRecent
		}
	}
	return mostRecent
}

// getTotalConnectivity calculates total connectivity for a cluster
func getTotalConnectivity(nodes []*KnowledgeNode) int {
	total := 0
	for _, node := range nodes {
		total += getNodeConnectivity(node)
	}
	return total
}

// getNodeConnectivity calculates connectivity for a node
// Connectivity = number of children + number of parent references
func getNodeConnectivity(node *KnowledgeNode) int {
	connectivity := len(node.Children)
	
	// For now, we primarily count children as the relationship graph
	// already builds the parent-child connections
	// In the future, we could enhance this by tracking parent references too
	
	// Recursively add children's connectivity
	for _, child := range node.Children {
		connectivity += getNodeConnectivity(child)
	}
	
	return connectivity
}
