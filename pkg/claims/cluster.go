package claims

import (
	"fmt"
	"sort"
)

// TensionCluster represents a group of related tensions converging on a domain area.
type TensionCluster struct {
	ID          string          // e.g., "tc-T-01"
	DomainTags  []string        // union of domain_tags from clustered claims
	Claims      []ClusterMember // claims that tension-reference the hub
	TargetClaim string          // the claim most tensions point to (hub)
	TargetModel string          // model that owns the target claim
	Models      []string        // distinct source models involved
	Score       float64         // urgency score
}

// ClusterMember is a single claim participating in a tension cluster.
type ClusterMember struct {
	ClaimID     string
	ModelName   string
	Text        string
	TensionType string // extends, contradicts, confirms
	Note        string
}

// tensionTypeWeight returns the scoring weight for a tension type.
func tensionTypeWeight(t string) float64 {
	switch t {
	case "contradicts":
		return 3
	case "extends":
		return 2
	case "confirms":
		return 1
	default:
		return 0
	}
}

// hubKey uniquely identifies a tension target (claim + model).
type hubKey struct {
	claim string
	model string
}

// FindClusters scans all claims files and returns tension clusters
// that meet the threshold (N+ claims from 2+ models).
// Results are sorted by score descending.
func FindClusters(files map[string]*File, threshold int) []TensionCluster {
	if len(files) == 0 {
		return nil
	}

	// Step 1: Build hub map — group tensions by their target claim.
	hubs := make(map[hubKey][]ClusterMember)

	for modelName, f := range files {
		for _, c := range f.Claims {
			for _, t := range c.Tensions {
				key := hubKey{claim: t.Claim, model: t.Model}
				hubs[key] = append(hubs[key], ClusterMember{
					ClaimID:     c.ID,
					ModelName:   modelName,
					Text:        c.Text,
					TensionType: t.Type,
					Note:        t.Note,
				})
			}
		}
	}

	// Step 2: Filter hubs by threshold and 2+ models, build clusters.
	var clusters []TensionCluster

	for key, members := range hubs {
		if len(members) < threshold {
			continue
		}

		// Count distinct models.
		modelSet := make(map[string]struct{})
		for _, m := range members {
			modelSet[m.ModelName] = struct{}{}
		}
		if len(modelSet) < 2 {
			continue
		}

		// Collect domain tags from source claims.
		tagSet := make(map[string]struct{})
		for modelName, f := range files {
			for _, c := range f.Claims {
				for _, m := range members {
					if c.ID == m.ClaimID && modelName == m.ModelName {
						for _, tag := range c.DomainTags {
							tagSet[tag] = struct{}{}
						}
					}
				}
			}
		}

		tags := make([]string, 0, len(tagSet))
		for tag := range tagSet {
			tags = append(tags, tag)
		}
		sort.Strings(tags)

		models := make([]string, 0, len(modelSet))
		for model := range modelSet {
			models = append(models, model)
		}
		sort.Strings(models)

		// Score: sum of tension type weights + (distinct_models - 1) * 2
		var score float64
		for _, m := range members {
			score += tensionTypeWeight(m.TensionType)
		}
		score += float64(len(models)-1) * 2

		clusters = append(clusters, TensionCluster{
			ID:          fmt.Sprintf("tc-%s", key.claim),
			DomainTags:  tags,
			Claims:      members,
			TargetClaim: key.claim,
			TargetModel: key.model,
			Models:      models,
			Score:       score,
		})
	}

	// Step 3: Sort by score descending.
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Score > clusters[j].Score
	})

	return clusters
}
