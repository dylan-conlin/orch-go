package verify

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// DispositionAction represents what to do with a discovered work item.
type DispositionAction int

const (
	// DispositionFileIssue means the item should be filed as a beads issue.
	DispositionFileIssue DispositionAction = iota
	// DispositionSkip means the item was explicitly skipped (individual item).
	DispositionSkip
	// DispositionSkipAll means all remaining items were skipped with a reason.
	DispositionSkipAll
)

// DiscoveredWorkItem represents a single piece of discovered work from SYNTHESIS.md.
type DiscoveredWorkItem struct {
	Description string // The item text (e.g., "- Deploy to staging")
	Source      string // Where it came from (e.g., "Next Actions", "Areas to Explore")
}

// DiscoveredWorkDisposition represents the disposition of a single work item.
type DiscoveredWorkDisposition struct {
	Item   DiscoveredWorkItem
	Action DispositionAction
}

// DiscoveredWorkResult contains the result of prompting for all discovered work.
type DiscoveredWorkResult struct {
	AllDispositioned bool                        // True if all items were handled
	Dispositions     []DiscoveredWorkDisposition // Disposition for each item
	SkipAllReason    string                      // If skip-all was used, the reason
}

// FiledItems returns the items that were marked for filing.
func (r *DiscoveredWorkResult) FiledItems() []DiscoveredWorkItem {
	var items []DiscoveredWorkItem
	for _, d := range r.Dispositions {
		if d.Action == DispositionFileIssue {
			items = append(items, d.Item)
		}
	}
	return items
}

// SkippedItems returns the items that were skipped.
func (r *DiscoveredWorkResult) SkippedItems() []DiscoveredWorkItem {
	var items []DiscoveredWorkItem
	for _, d := range r.Dispositions {
		if d.Action == DispositionSkip || d.Action == DispositionSkipAll {
			items = append(items, d.Item)
		}
	}
	return items
}

// CollectDiscoveredWork extracts all discovered work items from a synthesis.
// Returns items from NextActions, AreasToExplore, and Uncertainties sections.
func CollectDiscoveredWork(synth *Synthesis) []DiscoveredWorkItem {
	if synth == nil {
		return nil
	}

	var items []DiscoveredWorkItem

	// Collect from NextActions
	for _, action := range synth.NextActions {
		items = append(items, DiscoveredWorkItem{
			Description: action,
			Source:      "Next Actions",
		})
	}

	// Collect from AreasToExplore
	for _, area := range synth.AreasToExplore {
		items = append(items, DiscoveredWorkItem{
			Description: area,
			Source:      "Areas to Explore",
		})
	}

	// Collect from Uncertainties
	for _, uncertainty := range synth.Uncertainties {
		items = append(items, DiscoveredWorkItem{
			Description: uncertainty,
			Source:      "Uncertainties",
		})
	}

	return items
}

// PromptDiscoveredWorkDisposition prompts the user to disposition each discovered work item.
// For each item, the user can:
//   - 'y' or 'yes': File as a beads issue
//   - 'n' or 'no': Skip this item
//   - 's' or 'skip-all': Skip all remaining items (requires a reason)
//
// Returns a DiscoveredWorkResult with the disposition of each item.
// If the input ends before all items are dispositioned, returns an error.
func PromptDiscoveredWorkDisposition(items []DiscoveredWorkItem, input io.Reader, output io.Writer) (*DiscoveredWorkResult, error) {
	result := &DiscoveredWorkResult{
		Dispositions: make([]DiscoveredWorkDisposition, 0, len(items)),
	}

	// Empty list - nothing to disposition
	if len(items) == 0 {
		result.AllDispositioned = false // Nothing was dispositioned because nothing existed
		return result, nil
	}

	reader := bufio.NewReader(input)

	for i, item := range items {
		// Print the item
		fmt.Fprintf(output, "\n[%d/%d] (%s) %s\n", i+1, len(items), item.Source, item.Description)
		fmt.Fprint(output, "File as issue? [y/n/s(kip-all)]: ")

		response, err := reader.ReadString('\n')
		if err != nil {
			// EOF or error before all items handled
			result.AllDispositioned = false
			return result, fmt.Errorf("incomplete disposition: %d/%d items handled", len(result.Dispositions), len(items))
		}

		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "y", "yes":
			result.Dispositions = append(result.Dispositions, DiscoveredWorkDisposition{
				Item:   item,
				Action: DispositionFileIssue,
			})

		case "n", "no":
			result.Dispositions = append(result.Dispositions, DiscoveredWorkDisposition{
				Item:   item,
				Action: DispositionSkip,
			})

		case "s", "skip-all":
			// Require a reason for skip-all
			var reason string
			for reason == "" {
				fmt.Fprint(output, "Skip reason required (prevents lazy dismissal): ")
				reason, err = reader.ReadString('\n')
				if err != nil {
					result.AllDispositioned = false
					return result, fmt.Errorf("incomplete disposition: skip-all without reason")
				}
				reason = strings.TrimSpace(reason)
				if reason == "" {
					fmt.Fprintln(output, "A reason is required for skip-all.")
				}
			}

			result.SkipAllReason = reason

			// Mark current item as skip-all
			result.Dispositions = append(result.Dispositions, DiscoveredWorkDisposition{
				Item:   item,
				Action: DispositionSkipAll,
			})

			// Mark all remaining items as skip-all
			for j := i + 1; j < len(items); j++ {
				result.Dispositions = append(result.Dispositions, DiscoveredWorkDisposition{
					Item:   items[j],
					Action: DispositionSkipAll,
				})
			}

			result.AllDispositioned = true
			return result, nil

		default:
			// Invalid response, treat as skip (user can re-run if needed)
			fmt.Fprintf(output, "Unknown response %q, treating as skip\n", response)
			result.Dispositions = append(result.Dispositions, DiscoveredWorkDisposition{
				Item:   item,
				Action: DispositionSkip,
			})
		}
	}

	result.AllDispositioned = true
	return result, nil
}
