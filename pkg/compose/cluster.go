package compose

import (
	"fmt"
	"sort"
	"strings"
)

// MinKeywordOverlap is the minimum number of shared discriminative keywords
// to cluster two briefs. After TF-IDF scoring keeps only the top 15 keywords
// per brief, 3+ overlap between those top keywords signals a real pattern.
const MinKeywordOverlap = 3

// Cluster groups related briefs with the rationale for clustering.
type Cluster struct {
	Name           string   // Generated cluster name from top keywords
	Briefs         []*Brief // Member briefs
	SharedKeywords []string // Keywords shared across all members
	Rationale      string   // Why these were clustered
}

// ClusterBriefs groups briefs by keyword overlap using seed-based clustering.
// Unlike single-linkage (where any member can recruit), seed-based clustering
// picks the most-connected brief as a seed and only adds briefs that overlap
// with the seed. This prevents long chains that merge unrelated groups.
func ClusterBriefs(briefs []*Brief) []*Cluster {
	if len(briefs) == 0 {
		return nil
	}

	n := len(briefs)

	// Build pairwise overlap counts
	overlaps := make([][]int, n)
	for i := 0; i < n; i++ {
		overlaps[i] = make([]int, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			shared := KeywordOverlap(briefs[i].Keywords, briefs[j].Keywords)
			overlaps[i][j] = len(shared)
			overlaps[j][i] = len(shared)
		}
	}

	// Seed-based clustering:
	// 1. Find the brief with the most high-overlap connections → seed
	// 2. Add all briefs connected to the seed (above threshold) → cluster
	// 3. Remove clustered briefs, repeat
	assigned := make([]bool, n)
	var clusters []*Cluster

	for {
		// Find the best seed: the unassigned brief with the most connections
		bestSeed := -1
		bestConnections := 0

		for i := 0; i < n; i++ {
			if assigned[i] {
				continue
			}
			connections := 0
			for j := 0; j < n; j++ {
				if !assigned[j] && i != j && overlaps[i][j] >= MinKeywordOverlap {
					connections++
				}
			}
			if connections > bestConnections {
				bestConnections = connections
				bestSeed = i
			}
		}

		// No more seeds with connections → done
		if bestSeed < 0 || bestConnections < 1 {
			break
		}

		// Build cluster: seed + all unassigned briefs connected to seed
		members := []*Brief{briefs[bestSeed]}
		assigned[bestSeed] = true

		for j := 0; j < n; j++ {
			if !assigned[j] && overlaps[bestSeed][j] >= MinKeywordOverlap {
				members = append(members, briefs[j])
				assigned[j] = true
			}
		}

		if len(members) < 2 {
			continue
		}

		shared := findSharedKeywords(members)
		clusters = append(clusters, &Cluster{
			Name:           generateClusterName(shared),
			Briefs:         members,
			SharedKeywords: shared,
			Rationale:      generateRationale(members, shared),
		})
	}

	// Sort clusters by size (largest first)
	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i].Briefs) > len(clusters[j].Briefs)
	})

	return clusters
}

// UnclusteredBriefs returns briefs that didn't join any cluster.
func UnclusteredBriefs(briefs []*Brief, clusters []*Cluster) []*Brief {
	clustered := make(map[string]bool)
	for _, c := range clusters {
		for _, b := range c.Briefs {
			clustered[b.ID] = true
		}
	}

	var unclustered []*Brief
	for _, b := range briefs {
		if !clustered[b.ID] {
			unclustered = append(unclustered, b)
		}
	}
	return unclustered
}

// findSharedKeywords returns the top keywords by frequency across cluster members.
// Returns up to 10 keywords that appear in at least 2 members, sorted by frequency.
func findSharedKeywords(briefs []*Brief) []string {
	freq := make(map[string]int)
	for _, b := range briefs {
		seen := make(map[string]bool)
		for _, kw := range b.Keywords {
			if !seen[kw] {
				freq[kw]++
				seen[kw] = true
			}
		}
	}

	type kwFreq struct {
		word  string
		count int
	}
	var pairs []kwFreq
	for kw, count := range freq {
		if count >= 2 {
			pairs = append(pairs, kwFreq{kw, count})
		}
	}

	// Sort by frequency descending, then alphabetically
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].word < pairs[j].word
	})

	limit := 10
	if len(pairs) < limit {
		limit = len(pairs)
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = pairs[i].word
	}
	return result
}

// generateClusterName creates a human-readable name from top shared keywords.
func generateClusterName(keywords []string) string {
	if len(keywords) == 0 {
		return "Uncategorized"
	}

	limit := 3
	if len(keywords) < limit {
		limit = len(keywords)
	}

	parts := make([]string, limit)
	for i := 0; i < limit; i++ {
		parts[i] = keywords[i]
	}

	return strings.Join(parts, " / ")
}

// generateRationale explains why these briefs were grouped.
func generateRationale(briefs []*Brief, shared []string) string {
	return fmt.Sprintf(
		"%d briefs share %d keywords (%s). Content overlap detected across Frame/Resolution/Tension sections.",
		len(briefs), len(shared), strings.Join(shared, ", "),
	)
}
