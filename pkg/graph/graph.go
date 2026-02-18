package graph

const (
	DepBlocks            = "blocks"
	DepParentChild       = "parent-child"
	DepConditionalBlocks = "conditional-blocks"
	DepWaitsFor          = "waits-for"
)

// Node represents a minimal graph node for dependency computations.
type Node struct {
	ID       string
	Priority int
}

// Edge represents a dependency edge between nodes.
// From depends on To, so To blocks From.
type Edge struct {
	From string
	To   string
	Type string
}

// ComputeLayers assigns topological layers using longest-path layering.
// Layer 0 indicates nodes with no blocking dependencies.
// Only "blocks" dependencies are considered for layering (matching beads computeLayout()).
func ComputeLayers(nodes []Node, edges []Edge) map[string]int {
	layers := make(map[string]int, len(nodes))
	nodeSet := make(map[string]struct{}, len(nodes))
	dependsOn := make(map[string][]string, len(nodes))

	for _, node := range nodes {
		layers[node.ID] = -1
		nodeSet[node.ID] = struct{}{}
	}

	for _, edge := range edges {
		if normalizeEdgeType(edge.Type) != DepBlocks {
			continue
		}
		if _, ok := nodeSet[edge.From]; !ok {
			continue
		}
		if _, ok := nodeSet[edge.To]; !ok {
			continue
		}
		dependsOn[edge.From] = append(dependsOn[edge.From], edge.To)
	}

	changed := true
	for changed {
		changed = false
		for _, node := range nodes {
			if layers[node.ID] >= 0 {
				continue
			}

			deps := dependsOn[node.ID]
			if len(deps) == 0 {
				layers[node.ID] = 0
				changed = true
				continue
			}

			maxDepLayer := -1
			allAssigned := true
			for _, depID := range deps {
				depLayer, ok := layers[depID]
				if !ok || depLayer < 0 {
					allAssigned = false
					break
				}
				if depLayer > maxDepLayer {
					maxDepLayer = depLayer
				}
			}

			if allAssigned {
				layers[node.ID] = maxDepLayer + 1
				changed = true
			}
		}
	}

	for _, node := range nodes {
		if layers[node.ID] < 0 {
			layers[node.ID] = 0
		}
	}

	return layers
}

// ComputeEffectivePriority computes the minimum priority reachable downstream
// through blocking dependencies (transitive closure). Lower priority values
// indicate higher urgency (P0 < P1 < ...).
func ComputeEffectivePriority(nodes []Node, edges []Edge) map[string]int {
	blockedBy := make(map[string][]string, len(nodes))
	priorityByID := make(map[string]int, len(nodes))
	for _, node := range nodes {
		priorityByID[node.ID] = node.Priority
	}

	for _, edge := range edges {
		if !affectsReadyWork(edge.Type) {
			continue
		}
		if _, ok := priorityByID[edge.From]; !ok {
			continue
		}
		if _, ok := priorityByID[edge.To]; !ok {
			continue
		}
		blockedBy[edge.To] = append(blockedBy[edge.To], edge.From)
	}

	result := make(map[string]int, len(nodes))
	visiting := make(map[string]bool, len(nodes))

	var dfs func(string) int
	dfs = func(id string) int {
		if val, ok := result[id]; ok {
			return val
		}
		if visiting[id] {
			return priorityByID[id]
		}
		visiting[id] = true

		minPriority := priorityByID[id]
		for _, child := range blockedBy[id] {
			childPriority := dfs(child)
			if childPriority < minPriority {
				minPriority = childPriority
			}
		}

		visiting[id] = false
		result[id] = minPriority
		return minPriority
	}

	for _, node := range nodes {
		result[node.ID] = dfs(node.ID)
	}

	return result
}

func normalizeEdgeType(edgeType string) string {
	if edgeType == "" {
		return DepBlocks
	}
	return edgeType
}

func affectsReadyWork(edgeType string) bool {
	switch normalizeEdgeType(edgeType) {
	case DepBlocks, DepParentChild, DepConditionalBlocks, DepWaitsFor:
		return true
	default:
		return false
	}
}
