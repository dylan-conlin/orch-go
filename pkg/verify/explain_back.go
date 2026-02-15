package verify

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ExplainBackResult contains the structured explanation from the human.
type ExplainBackResult struct {
	WhatBuilt       string // What was built/implemented
	WhyMatters      string // Why it matters/the value it provides
	HowVerified     string // How it was verified to work
	FullExplanation string // Combined explanation for beads comment
}

// PromptExplainBack prompts the human to explain what was built and why.
// This creates an unfakeable verification gate - you can't rubber-stamp a conversational explanation.
// The explanation is captured as a structured artifact that future sessions can reference.
//
// The prompts are designed to force processing, similar to pilot callouts in instrument flying:
// - What did you build? (forces articulation of the actual work)
// - Why does it matter? (forces understanding of value/purpose)
// - How did you verify it works? (forces reflection on validation)
func PromptExplainBack(stdin io.Reader, stdout io.Writer) (*ExplainBackResult, error) {
	reader := bufio.NewReader(stdin)
	result := &ExplainBackResult{}

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "┌─────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(stdout, "│  🎯 EXPLAIN-BACK VERIFICATION                               │")
	fmt.Fprintln(stdout, "├─────────────────────────────────────────────────────────────┤")
	fmt.Fprintln(stdout, "│  Before completing, explain what was built in your own     │")
	fmt.Fprintln(stdout, "│  words. This ensures human comprehension, not just agent   │")
	fmt.Fprintln(stdout, "│  self-reported success.                                     │")
	fmt.Fprintln(stdout, "└─────────────────────────────────────────────────────────────┘")
	fmt.Fprintln(stdout, "")

	// Prompt 1: What did you build?
	fmt.Fprint(stdout, "What did you build? (1-2 sentences)\n> ")
	whatBuilt, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read 'what built' response: %w", err)
	}
	whatBuilt = strings.TrimSpace(whatBuilt)
	if whatBuilt == "" {
		return nil, fmt.Errorf("'what built' cannot be empty")
	}
	result.WhatBuilt = whatBuilt

	// Prompt 2: Why does it matter?
	fmt.Fprint(stdout, "\nWhy does it matter? (1-2 sentences)\n> ")
	whyMatters, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read 'why matters' response: %w", err)
	}
	whyMatters = strings.TrimSpace(whyMatters)
	if whyMatters == "" {
		return nil, fmt.Errorf("'why matters' cannot be empty")
	}
	result.WhyMatters = whyMatters

	// Prompt 3: How did you verify it works?
	fmt.Fprint(stdout, "\nHow did you verify it works? (1-2 sentences)\n> ")
	howVerified, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read 'how verified' response: %w", err)
	}
	howVerified = strings.TrimSpace(howVerified)
	if howVerified == "" {
		return nil, fmt.Errorf("'how verified' cannot be empty")
	}
	result.HowVerified = howVerified

	// Build the full explanation for storage
	result.FullExplanation = fmt.Sprintf(`EXPLAIN-BACK:
What: %s
Why: %s
Verification: %s`, whatBuilt, whyMatters, howVerified)

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "✓ Explanation captured")
	fmt.Fprintln(stdout, "")

	return result, nil
}
