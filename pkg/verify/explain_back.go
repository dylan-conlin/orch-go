package verify

import "fmt"

// ExplainBackResult contains the explanation provided by the orchestrator.
type ExplainBackResult struct {
	FullExplanation string // The explanation text for beads comment storage
}

// FormatExplainBack takes explanation text from --explain flag and formats it
// for storage as a beads comment. Returns error if explanation is empty.
//
// The conversational quality check (is the explanation sufficient?) stays with
// the AI orchestrator. The CLI's job is: accept explanation, store it, gate on non-empty.
func FormatExplainBack(explanation string) (*ExplainBackResult, error) {
	if explanation == "" {
		return nil, fmt.Errorf("explanation cannot be empty")
	}

	return &ExplainBackResult{
		FullExplanation: fmt.Sprintf("EXPLAIN-BACK:\n%s", explanation),
	}, nil
}
