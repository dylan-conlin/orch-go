package attention

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// DuplicateCandidateCollector implements the Collector interface for detecting
// issues with similar titles that may be duplicates. Uses simple text matching
// (normalized word overlap) as v1 - embedding similarity can be added later.
type DuplicateCandidateCollector struct {
	client    beads.BeadsClient
	threshold float64 // Similarity threshold (0.0-1.0), default 0.6
}

// NewDuplicateCandidateCollector creates a new DuplicateCandidateCollector.
// threshold is the minimum similarity score (0.0-1.0) to flag as duplicate candidate (default: 0.6).
func NewDuplicateCandidateCollector(client beads.BeadsClient, threshold float64) *DuplicateCandidateCollector {
	if threshold <= 0 || threshold > 1.0 {
		threshold = 0.6
	}
	return &DuplicateCandidateCollector{
		client:    client,
		threshold: threshold,
	}
}

// Collect gathers attention items for issues that appear to be duplicates of each other.
func (c *DuplicateCandidateCollector) Collect(role string) ([]AttentionItem, error) {
	issues, err := c.client.List(&beads.ListArgs{
		Status: "open",
		Limit:  beads.IntPtr(0),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query open issues: %w", err)
	}

	if len(issues) < 2 {
		return nil, nil
	}

	now := time.Now()
	items := make([]AttentionItem, 0)
	// Track pairs already reported to avoid A-B and B-A duplicates
	seen := make(map[string]bool)

	for i := 0; i < len(issues); i++ {
		for j := i + 1; j < len(issues); j++ {
			score := titleSimilarity(issues[i].Title, issues[j].Title)
			if score < c.threshold {
				continue
			}

			// Create a stable pair key
			key := issues[i].ID + ":" + issues[j].ID
			if seen[key] {
				continue
			}
			seen[key] = true

			priority := calculateDuplicatePriority(score, role)

			// Report signal on the newer issue (higher ID typically = newer)
			// Both issues are referenced in metadata
			item := AttentionItem{
				ID:          fmt.Sprintf("duplicate-%s-%s", issues[i].ID, issues[j].ID),
				Source:      "beads",
				Concern:     Observability,
				Signal:      "duplicate-candidate",
				Subject:     issues[j].ID, // Surface on the newer issue
				Summary:     fmt.Sprintf("Possible duplicate (%.0f%%): %s", score*100, issues[i].Title),
				Priority:    priority,
				Role:        role,
				ActionHint:  fmt.Sprintf("Compare: bd show %s && bd show %s", issues[i].ID, issues[j].ID),
				CollectedAt: now,
				Metadata: map[string]any{
					"similar_to":    issues[i].ID,
					"similar_title": issues[i].Title,
					"this_title":    issues[j].Title,
					"score":         score,
				},
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// titleSimilarity computes the similarity between two issue titles using
// normalized word overlap (Jaccard-like coefficient on significant words).
func titleSimilarity(a, b string) float64 {
	wordsA := significantWords(a)
	wordsB := significantWords(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}

	// Count intersection
	intersection := 0
	for word := range wordsA {
		if wordsB[word] {
			intersection++
		}
	}

	// Jaccard coefficient: |A ∩ B| / |A ∪ B|
	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// significantWords extracts meaningful words from a title, filtering out
// common stop words and normalizing to lowercase.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true,
	"did": true, "will": true, "would": true, "could": true, "should": true,
	"may": true, "might": true, "shall": true, "can": true,
	"in": true, "on": true, "at": true, "to": true, "for": true,
	"of": true, "with": true, "by": true, "from": true, "as": true,
	"into": true, "through": true, "during": true, "before": true, "after": true,
	"and": true, "but": true, "or": true, "nor": true, "not": true,
	"so": true, "yet": true, "both": true, "either": true, "neither": true,
	"it": true, "its": true, "this": true, "that": true, "these": true,
	"those": true, "if": true, "when": true, "where": true, "how": true,
	"what": true, "which": true, "who": true, "whom": true, "whose": true,
	"all": true, "each": true, "every": true, "any": true, "some": true,
	"no": true, "more": true, "most": true, "other": true, "than": true,
}

func significantWords(title string) map[string]bool {
	words := make(map[string]bool)
	for _, word := range strings.Fields(strings.ToLower(title)) {
		// Strip common punctuation
		word = strings.Trim(word, ".,;:!?()[]{}\"'-/\\")
		if len(word) < 2 {
			continue
		}
		if stopWords[word] {
			continue
		}
		words[word] = true
	}
	return words
}

// calculateDuplicatePriority determines priority based on similarity score and role.
func calculateDuplicatePriority(score float64, role string) int {
	base := 180

	// Higher similarity = more urgent
	if score > 0.9 {
		base -= 40 // Very likely duplicate
	} else if score > 0.75 {
		base -= 25
	} else if score > 0.6 {
		base -= 10
	}

	switch role {
	case "human":
		return base - 10 // Humans should resolve duplicates
	case "orchestrator":
		return base
	case "daemon":
		return base + 100
	default:
		return base
	}
}
