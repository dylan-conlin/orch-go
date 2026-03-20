package claims

import (
	"sort"
	"testing"
)

func TestFindClusters_EmptyInput(t *testing.T) {
	clusters := FindClusters(nil, 3)
	if len(clusters) != 0 {
		t.Fatalf("expected 0 clusters, got %d", len(clusters))
	}
}

func TestFindClusters_BelowThreshold(t *testing.T) {
	// 2 tensions pointing at same target — below threshold of 3
	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{
					ID:   "A-01",
					Text: "claim A-01",
					Tensions: []Tension{
						{Claim: "T-01", Model: "target-model", Type: "extends", Note: "note1"},
					},
				},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{
					ID:   "B-01",
					Text: "claim B-01",
					Tensions: []Tension{
						{Claim: "T-01", Model: "target-model", Type: "extends", Note: "note2"},
					},
				},
			},
		},
	}

	clusters := FindClusters(files, 3)
	if len(clusters) != 0 {
		t.Fatalf("expected 0 clusters (below threshold), got %d", len(clusters))
	}
}

func TestFindClusters_MeetsThreshold(t *testing.T) {
	// 3 tensions from 2 models pointing at same target
	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{
					ID:   "A-01",
					Text: "claim A-01",
					Tensions: []Tension{
						{Claim: "T-01", Model: "target-model", Type: "extends", Note: "deepens"},
					},
				},
				{
					ID:   "A-02",
					Text: "claim A-02",
					Tensions: []Tension{
						{Claim: "T-01", Model: "target-model", Type: "contradicts", Note: "conflicts"},
					},
				},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{
					ID:   "B-01",
					Text: "claim B-01",
					Tensions: []Tension{
						{Claim: "T-01", Model: "target-model", Type: "confirms", Note: "agrees"},
					},
				},
			},
		},
	}

	clusters := FindClusters(files, 3)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}

	c := clusters[0]
	if c.TargetClaim != "T-01" {
		t.Errorf("expected target T-01, got %s", c.TargetClaim)
	}
	if c.TargetModel != "target-model" {
		t.Errorf("expected target model target-model, got %s", c.TargetModel)
	}
	if len(c.Claims) != 3 {
		t.Errorf("expected 3 claims, got %d", len(c.Claims))
	}
	if len(c.Models) != 2 {
		t.Errorf("expected 2 models, got %d", len(c.Models))
	}

	// Score: 1 contradicts*3 + 1 extends*2 + 1 confirms*1 + (2-1)*2 = 3+2+1+2 = 8
	if c.Score != 8 {
		t.Errorf("expected score 8, got %f", c.Score)
	}
}

func TestFindClusters_SingleModelFiltered(t *testing.T) {
	// 3 tensions all from same model — fails 2+ models requirement
	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{ID: "A-01", Text: "c1", Tensions: []Tension{{Claim: "T-01", Model: "target", Type: "extends", Note: "n"}}},
				{ID: "A-02", Text: "c2", Tensions: []Tension{{Claim: "T-01", Model: "target", Type: "extends", Note: "n"}}},
				{ID: "A-03", Text: "c3", Tensions: []Tension{{Claim: "T-01", Model: "target", Type: "extends", Note: "n"}}},
			},
		},
	}

	clusters := FindClusters(files, 3)
	if len(clusters) != 0 {
		t.Fatalf("expected 0 clusters (single model), got %d", len(clusters))
	}
}

func TestFindClusters_MultipleClusters(t *testing.T) {
	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{ID: "A-01", Text: "c1", Tensions: []Tension{{Claim: "T-01", Model: "target", Type: "extends", Note: "n"}}},
				{ID: "A-02", Text: "c2", Tensions: []Tension{{Claim: "T-01", Model: "target", Type: "extends", Note: "n"}}},
				{ID: "A-03", Text: "c3", Tensions: []Tension{{Claim: "T-02", Model: "target", Type: "contradicts", Note: "n"}}},
				{ID: "A-04", Text: "c4", Tensions: []Tension{{Claim: "T-02", Model: "target", Type: "extends", Note: "n"}}},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{ID: "B-01", Text: "c5", Tensions: []Tension{{Claim: "T-01", Model: "target", Type: "confirms", Note: "n"}}},
				{ID: "B-02", Text: "c6", Tensions: []Tension{{Claim: "T-02", Model: "target", Type: "extends", Note: "n"}}},
			},
		},
	}

	clusters := FindClusters(files, 3)
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}

	// Sort by target for deterministic checks
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].TargetClaim < clusters[j].TargetClaim
	})

	if clusters[0].TargetClaim != "T-01" {
		t.Errorf("expected first cluster target T-01, got %s", clusters[0].TargetClaim)
	}
	if clusters[1].TargetClaim != "T-02" {
		t.Errorf("expected second cluster target T-02, got %s", clusters[1].TargetClaim)
	}
}

func TestFindClusters_DomainTags(t *testing.T) {
	// Verify domain tags are collected from member claims
	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{ID: "A-01", Text: "c1", DomainTags: []string{"gates", "enforcement"},
					Tensions: []Tension{{Claim: "T-01", Model: "target", Type: "extends", Note: "n"}}},
				{ID: "A-02", Text: "c2", DomainTags: []string{"gates", "measurement"},
					Tensions: []Tension{{Claim: "T-01", Model: "target", Type: "extends", Note: "n"}}},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{ID: "B-01", Text: "c3", DomainTags: []string{"enforcement", "displacement"},
					Tensions: []Tension{{Claim: "T-01", Model: "target", Type: "confirms", Note: "n"}}},
			},
		},
	}

	clusters := FindClusters(files, 3)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}

	tags := clusters[0].DomainTags
	sort.Strings(tags)
	expected := []string{"displacement", "enforcement", "gates", "measurement"}
	if len(tags) != len(expected) {
		t.Fatalf("expected %d domain tags, got %d: %v", len(expected), len(tags), tags)
	}
	for i, tag := range tags {
		if tag != expected[i] {
			t.Errorf("tag[%d]: expected %s, got %s", i, expected[i], tag)
		}
	}
}

func TestFindClusters_ScoreWeighting(t *testing.T) {
	// All contradicts should score higher than all extends
	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{ID: "A-01", Text: "c1", Tensions: []Tension{{Claim: "T-01", Model: "t", Type: "contradicts", Note: "n"}}},
				{ID: "A-02", Text: "c2", Tensions: []Tension{{Claim: "T-01", Model: "t", Type: "contradicts", Note: "n"}}},
				{ID: "A-03", Text: "c3", Tensions: []Tension{{Claim: "T-02", Model: "t", Type: "extends", Note: "n"}}},
				{ID: "A-04", Text: "c4", Tensions: []Tension{{Claim: "T-02", Model: "t", Type: "extends", Note: "n"}}},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{ID: "B-01", Text: "c5", Tensions: []Tension{{Claim: "T-01", Model: "t", Type: "contradicts", Note: "n"}}},
				{ID: "B-02", Text: "c6", Tensions: []Tension{{Claim: "T-02", Model: "t", Type: "extends", Note: "n"}}},
			},
		},
	}

	clusters := FindClusters(files, 3)
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}

	// Sort descending by score
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Score > clusters[j].Score
	})

	// T-01 (all contradicts): 3*3 + (2-1)*2 = 11
	// T-02 (all extends): 3*2 + (2-1)*2 = 8
	if clusters[0].TargetClaim != "T-01" {
		t.Errorf("expected highest scoring cluster to be T-01, got %s", clusters[0].TargetClaim)
	}
	if clusters[0].Score != 11 {
		t.Errorf("expected score 11, got %f", clusters[0].Score)
	}
	if clusters[1].Score != 8 {
		t.Errorf("expected score 8, got %f", clusters[1].Score)
	}
}

func TestFindClusters_CustomThreshold(t *testing.T) {
	// With threshold=2, a 2-member cluster should pass
	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{ID: "A-01", Text: "c1", Tensions: []Tension{{Claim: "T-01", Model: "t", Type: "extends", Note: "n"}}},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{ID: "B-01", Text: "c2", Tensions: []Tension{{Claim: "T-01", Model: "t", Type: "extends", Note: "n"}}},
			},
		},
	}

	clusters := FindClusters(files, 2)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster with threshold=2, got %d", len(clusters))
	}
}

func TestFindClusters_SortedByScoreDescending(t *testing.T) {
	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{ID: "A-01", Text: "c1", Tensions: []Tension{{Claim: "T-01", Model: "t", Type: "confirms", Note: "n"}}},
				{ID: "A-02", Text: "c2", Tensions: []Tension{{Claim: "T-01", Model: "t", Type: "confirms", Note: "n"}}},
				{ID: "A-03", Text: "c3", Tensions: []Tension{{Claim: "T-02", Model: "t", Type: "contradicts", Note: "n"}}},
				{ID: "A-04", Text: "c4", Tensions: []Tension{{Claim: "T-02", Model: "t", Type: "contradicts", Note: "n"}}},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{ID: "B-01", Text: "c5", Tensions: []Tension{{Claim: "T-01", Model: "t", Type: "confirms", Note: "n"}}},
				{ID: "B-02", Text: "c6", Tensions: []Tension{{Claim: "T-02", Model: "t", Type: "contradicts", Note: "n"}}},
			},
		},
	}

	clusters := FindClusters(files, 3)
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}

	// T-02 (contradicts) should score higher than T-01 (confirms)
	if clusters[0].Score <= clusters[1].Score {
		t.Errorf("expected clusters sorted by score descending, got %f <= %f", clusters[0].Score, clusters[1].Score)
	}
}
