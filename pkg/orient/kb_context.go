package orient

import (
	"encoding/json"
)

// KBEntry represents a single knowledge base entry relevant to an issue.
type KBEntry struct {
	Type    string `json:"type"`    // "constraint", "decision", "attempt"
	Content string `json:"content"`
	Reason  string `json:"reason,omitempty"`
}

// kbContextJSON mirrors the JSON output of `kb context --format json`.
type kbContextJSON struct {
	Constraints []kbEntryJSON `json:"constraints"`
	Decisions   []kbEntryJSON `json:"decisions"`
	Attempts    []kbEntryJSON `json:"attempts"`
}

type kbEntryJSON struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Reason  string `json:"reason"`
	Result  string `json:"result"`
}

// ParseKBContext parses the JSON output of `kb context --format json` and
// returns the top entries (up to maxPerType per category).
func ParseKBContext(jsonData []byte, maxPerType int) []KBEntry {
	var raw kbContextJSON
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return nil
	}

	var entries []KBEntry

	// Constraints first (most actionable)
	for i, c := range raw.Constraints {
		if i >= maxPerType {
			break
		}
		entries = append(entries, KBEntry{
			Type:    "constraint",
			Content: c.Content,
			Reason:  c.Reason,
		})
	}

	// Decisions
	for i, d := range raw.Decisions {
		if i >= maxPerType {
			break
		}
		entries = append(entries, KBEntry{
			Type:    "decision",
			Content: d.Content,
			Reason:  d.Reason,
		})
	}

	// Failed attempts
	for i, a := range raw.Attempts {
		if i >= maxPerType {
			break
		}
		reason := a.Reason
		if a.Result != "" {
			reason = a.Result // For attempts, "result" contains what failed
		}
		entries = append(entries, KBEntry{
			Type:    "attempt",
			Content: a.Content,
			Reason:  reason,
		})
	}

	return entries
}

// SelectTopEntries returns the most relevant entries, limited to maxTotal.
// Prioritizes: constraints > attempts > decisions.
func SelectTopEntries(entries []KBEntry, maxTotal int) []KBEntry {
	if len(entries) <= maxTotal {
		return entries
	}

	// Priority order: constraints, attempts, decisions
	var constraints, attempts, decisions []KBEntry
	for _, e := range entries {
		switch e.Type {
		case "constraint":
			constraints = append(constraints, e)
		case "attempt":
			attempts = append(attempts, e)
		case "decision":
			decisions = append(decisions, e)
		}
	}

	var result []KBEntry
	sources := [][]KBEntry{constraints, attempts, decisions}
	for _, src := range sources {
		for _, e := range src {
			if len(result) >= maxTotal {
				return result
			}
			result = append(result, e)
		}
	}
	return result
}
