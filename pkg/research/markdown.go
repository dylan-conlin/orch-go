package research

import (
	"bufio"
	"os"
	"strings"
)

// MarkdownClaim represents a claim parsed from a model.md Claims (Testable) table.
type MarkdownClaim struct {
	ID          string
	Text        string
	HowToVerify string
}

// ParseMarkdownClaims extracts claims from a model.md file's
// ## Claims (Testable) section with markdown table format.
//
// Expected format:
//
//	## Claims (Testable)
//
//	| ID | Claim | How to Verify |
//	|----|-------|---------------|
//	| NI-01 | Named gaps compose... | Compare clustering... |
func ParseMarkdownClaims(modelMdPath string) ([]MarkdownClaim, error) {
	f, err := os.Open(modelMdPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Increase buffer for long lines in model.md files
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	inClaimsSection := false
	headerParsed := false
	var claims []MarkdownClaim

	for scanner.Scan() {
		line := scanner.Text()

		// Detect claims section header
		if strings.HasPrefix(line, "## Claims") {
			inClaimsSection = true
			headerParsed = false
			continue
		}

		// Exit claims section on next ## heading
		if inClaimsSection && strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "## Claims") {
			break
		}

		if !inClaimsSection {
			continue
		}

		// Skip non-table lines
		if !strings.HasPrefix(line, "|") {
			continue
		}

		// Skip table header row (| ID | Claim | How to Verify |)
		if !headerParsed {
			if strings.Contains(strings.ToLower(line), "id") && strings.Contains(strings.ToLower(line), "claim") {
				headerParsed = true
				continue
			}
		}

		// Skip separator row (|---|---|---|)
		if strings.Contains(line, "---") {
			continue
		}

		// Parse claim row
		claim := parseTableRow(line)
		if claim != nil {
			claims = append(claims, *claim)
		}
	}

	return claims, scanner.Err()
}

// parseTableRow extracts a claim from a markdown table row.
// Format: | ID | Claim text | How to verify |
func parseTableRow(line string) *MarkdownClaim {
	// Split by | and trim
	parts := strings.Split(line, "|")
	if len(parts) < 4 { // empty | ID | Claim | HowToVerify | empty
		return nil
	}

	// Remove first and last empty elements from leading/trailing |
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}

	// Find the non-empty cells (skip leading/trailing empties)
	var nonEmpty []string
	for _, c := range cells {
		if c != "" || len(nonEmpty) > 0 {
			nonEmpty = append(nonEmpty, c)
		}
	}

	if len(nonEmpty) < 2 {
		return nil
	}

	id := nonEmpty[0]
	if !isClaimID(id) {
		return nil
	}

	claim := &MarkdownClaim{ID: strings.ToUpper(id)}
	if len(nonEmpty) > 1 {
		claim.Text = nonEmpty[1]
	}
	if len(nonEmpty) > 2 {
		claim.HowToVerify = nonEmpty[2]
	}

	return claim
}
