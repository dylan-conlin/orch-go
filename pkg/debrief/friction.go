package debrief

import (
	"fmt"
	"sort"
	"strings"
)

// FrictionCategory aggregates friction reports by category across all agents.
type FrictionCategory struct {
	Category     string   // bug, gap, ceremony, tooling, etc.
	Count        int      // Total reports in this category
	Descriptions []string // Unique descriptions
}

// FrictionSummaryInput represents friction data from a single agent.
type FrictionSummaryInput struct {
	BeadsID      string
	Category     string
	Description  string
}

// CollectFrictionSummary aggregates friction inputs by category.
// Returns sorted by count descending.
func CollectFrictionSummary(inputs []FrictionSummaryInput) []FrictionCategory {
	if len(inputs) == 0 {
		return nil
	}

	catMap := map[string]*FrictionCategory{}
	order := []string{}
	seen := map[string]map[string]bool{} // category -> description -> seen

	for _, input := range inputs {
		cat := strings.TrimSpace(input.Category)
		if cat == "" {
			continue
		}

		fc := catMap[cat]
		if fc == nil {
			fc = &FrictionCategory{Category: cat}
			catMap[cat] = fc
			order = append(order, cat)
			seen[cat] = make(map[string]bool)
		}
		fc.Count++

		desc := strings.TrimSpace(input.Description)
		if desc != "" && !seen[cat][desc] {
			fc.Descriptions = append(fc.Descriptions, desc)
			seen[cat][desc] = true
		}
	}

	if len(catMap) == 0 {
		return nil
	}

	var result []FrictionCategory
	for _, cat := range order {
		result = append(result, *catMap[cat])
	}

	// Sort by count descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

// FormatFrictionSummary formats friction categories into debrief bullet list lines.
func FormatFrictionSummary(categories []FrictionCategory) []string {
	if len(categories) == 0 {
		return nil
	}

	var lines []string
	for _, cat := range categories {
		if len(cat.Descriptions) == 0 {
			lines = append(lines, fmt.Sprintf("**%s:** %d report(s)", cat.Category, cat.Count))
		} else if len(cat.Descriptions) == 1 {
			lines = append(lines, fmt.Sprintf("**%s:** %s", cat.Category, cat.Descriptions[0]))
		} else {
			descs := strings.Join(cat.Descriptions, "; ")
			lines = append(lines, fmt.Sprintf("**%s** (%d): %s", cat.Category, cat.Count, truncate(descs, 150)))
		}
	}
	return lines
}
