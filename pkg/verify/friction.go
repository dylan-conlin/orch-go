package verify

import (
	"fmt"
	"strings"
)

// FrictionItem represents a single friction report from an agent's beads comments.
type FrictionItem struct {
	Category    string // bug, gap, ceremony, tooling
	Description string
}

// ParseFrictionComments extracts friction items from beads comments.
// Expected format: "Friction: <category>: <description>" or "Friction: <category>"
// "Friction: none" is treated as no friction and returns nil.
func ParseFrictionComments(comments []Comment) []FrictionItem {
	var items []FrictionItem

	for _, c := range comments {
		text := strings.TrimSpace(c.Text)
		if !strings.HasPrefix(text, "Friction:") {
			continue
		}

		rest := strings.TrimSpace(strings.TrimPrefix(text, "Friction:"))
		if rest == "" || strings.EqualFold(rest, "none") {
			continue
		}

		// Parse "category: description" or just "category"
		parts := strings.SplitN(rest, ":", 2)
		category := strings.TrimSpace(parts[0])
		description := ""
		if len(parts) > 1 {
			description = strings.TrimSpace(parts[1])
		}

		items = append(items, FrictionItem{
			Category:    category,
			Description: description,
		})
	}

	return items
}

// FetchAndParseFriction fetches beads comments and parses friction items.
// Returns nil if beadsID is empty or comments cannot be fetched.
func FetchAndParseFriction(beadsID, projectDir string) []FrictionItem {
	if beadsID == "" {
		return nil
	}

	comments, err := GetComments(beadsID, projectDir)
	if err != nil {
		return nil
	}

	return ParseFrictionComments(comments)
}

// FormatFrictionAdvisory formats friction items for display during orch complete.
// Returns empty string if no items.
func FormatFrictionAdvisory(items []FrictionItem) string {
	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n--- Friction Report ---\n")

	// Group by category
	grouped := make(map[string][]string)
	order := []string{}
	for _, item := range items {
		if _, seen := grouped[item.Category]; !seen {
			order = append(order, item.Category)
		}
		grouped[item.Category] = append(grouped[item.Category], item.Description)
	}

	// Display with category icons
	for _, cat := range order {
		descs := grouped[cat]
		icon := frictionCategoryIcon(cat)
		sb.WriteString(fmt.Sprintf("%s  %s (%d):\n", icon, cat, len(descs)))
		for _, d := range descs {
			if d != "" {
				sb.WriteString(fmt.Sprintf("    - %s\n", d))
			}
		}
	}

	sb.WriteString("------------------------\n")
	return sb.String()
}

func frictionCategoryIcon(category string) string {
	switch category {
	case "bug":
		return "\U0001F41B" // bug emoji
	case "gap":
		return "\U0001F50D" // magnifying glass
	case "ceremony":
		return "\U0001F4CB" // clipboard
	case "tooling":
		return "\U0001F527" // wrench
	default:
		return "\u2022" // bullet
	}
}
