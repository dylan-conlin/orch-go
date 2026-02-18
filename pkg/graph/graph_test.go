package graph

import "testing"

func TestComputeLayersLinearChain(t *testing.T) {
	nodes := []Node{
		{ID: "A", Priority: 2},
		{ID: "B", Priority: 1},
		{ID: "C", Priority: 0},
	}
	edges := []Edge{
		{From: "B", To: "A", Type: DepBlocks},
		{From: "C", To: "B", Type: DepBlocks},
	}

	got := ComputeLayers(nodes, edges)
	expectLayer(t, got, "A", 0)
	expectLayer(t, got, "B", 1)
	expectLayer(t, got, "C", 2)
}

func TestComputeLayersDiamond(t *testing.T) {
	nodes := []Node{
		{ID: "A", Priority: 2},
		{ID: "B", Priority: 1},
		{ID: "C", Priority: 1},
		{ID: "D", Priority: 0},
	}
	edges := []Edge{
		{From: "B", To: "A", Type: DepBlocks},
		{From: "C", To: "A", Type: DepBlocks},
		{From: "D", To: "B", Type: DepBlocks},
		{From: "D", To: "C", Type: DepBlocks},
	}

	got := ComputeLayers(nodes, edges)
	expectLayer(t, got, "A", 0)
	expectLayer(t, got, "B", 1)
	expectLayer(t, got, "C", 1)
	expectLayer(t, got, "D", 2)
}

func TestComputeLayersCycle(t *testing.T) {
	nodes := []Node{
		{ID: "A", Priority: 2},
		{ID: "B", Priority: 1},
	}
	edges := []Edge{
		{From: "A", To: "B", Type: DepBlocks},
		{From: "B", To: "A", Type: DepBlocks},
	}

	got := ComputeLayers(nodes, edges)
	expectLayer(t, got, "A", 0)
	expectLayer(t, got, "B", 0)
}

func TestComputeLayersDisconnected(t *testing.T) {
	nodes := []Node{
		{ID: "A", Priority: 2},
		{ID: "B", Priority: 1},
		{ID: "C", Priority: 0},
	}
	edges := []Edge{
		{From: "B", To: "A", Type: DepBlocks},
	}

	got := ComputeLayers(nodes, edges)
	expectLayer(t, got, "A", 0)
	expectLayer(t, got, "B", 1)
	expectLayer(t, got, "C", 0)
}

func TestComputeEffectivePriorityLinearChain(t *testing.T) {
	nodes := []Node{
		{ID: "A", Priority: 2},
		{ID: "B", Priority: 1},
		{ID: "C", Priority: 0},
	}
	edges := []Edge{
		{From: "B", To: "A", Type: DepBlocks},
		{From: "C", To: "B", Type: DepBlocks},
	}

	got := ComputeEffectivePriority(nodes, edges)
	expectPriority(t, got, "A", 0)
	expectPriority(t, got, "B", 0)
	expectPriority(t, got, "C", 0)
}

func TestComputeEffectivePriorityDiamond(t *testing.T) {
	nodes := []Node{
		{ID: "A", Priority: 2},
		{ID: "B", Priority: 3},
		{ID: "C", Priority: 1},
		{ID: "D", Priority: 0},
	}
	edges := []Edge{
		{From: "B", To: "A", Type: DepBlocks},
		{From: "C", To: "A", Type: DepBlocks},
		{From: "D", To: "B", Type: DepBlocks},
		{From: "D", To: "C", Type: DepBlocks},
	}

	got := ComputeEffectivePriority(nodes, edges)
	expectPriority(t, got, "A", 0)
	expectPriority(t, got, "B", 0)
	expectPriority(t, got, "C", 0)
	expectPriority(t, got, "D", 0)
}

func TestComputeEffectivePriorityCycle(t *testing.T) {
	nodes := []Node{
		{ID: "A", Priority: 2},
		{ID: "B", Priority: 1},
	}
	edges := []Edge{
		{From: "A", To: "B", Type: DepBlocks},
		{From: "B", To: "A", Type: DepBlocks},
	}

	got := ComputeEffectivePriority(nodes, edges)
	expectPriority(t, got, "A", 1)
	expectPriority(t, got, "B", 1)
}

func TestComputeEffectivePriorityDisconnected(t *testing.T) {
	nodes := []Node{
		{ID: "A", Priority: 3},
		{ID: "B", Priority: 0},
		{ID: "C", Priority: 2},
	}
	edges := []Edge{
		{From: "B", To: "A", Type: DepBlocks},
	}

	got := ComputeEffectivePriority(nodes, edges)
	expectPriority(t, got, "A", 0)
	expectPriority(t, got, "B", 0)
	expectPriority(t, got, "C", 2)
}

func expectLayer(t *testing.T, layers map[string]int, id string, want int) {
	t.Helper()
	if got := layers[id]; got != want {
		t.Fatalf("layer[%s] = %d, want %d", id, got, want)
	}
}

func expectPriority(t *testing.T, priorities map[string]int, id string, want int) {
	t.Helper()
	if got := priorities[id]; got != want {
		t.Fatalf("priority[%s] = %d, want %d", id, got, want)
	}
}
