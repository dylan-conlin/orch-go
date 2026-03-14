package kbgate

import (
	"fmt"
	"os"
	"strings"
)

// Required columns for the claim ledger table in model files.
var requiredClaimColumns = []string{
	"claim_id", "claim_text", "claim_type", "scope", "novelty_level", "evidence_refs",
}

// Required columns for the vocabulary canonicalization table in model files.
var requiredCanonColumns = []string{
	"term", "plain_language", "nearest_existing_concepts", "claimed_delta", "verdict",
}

// Valid claim types for model claim ledger entries.
var validModelClaimTypes = map[string]bool{
	"observation":    true,
	"mechanism":      true,
	"generalization": true,
	"recommendation": true,
}

// Valid novelty levels for model claim ledger entries.
var validNoveltyLevels = map[string]bool{
	"restatement": true,
	"synthesis":   true,
	"novel":       true,
}

// ModelClaimEntry represents a parsed row from a claim ledger markdown table.
type ModelClaimEntry struct {
	ClaimID      string
	ClaimText    string
	ClaimType    string
	Scope        string
	NoveltyLevel string
	EvidenceRefs string
}

// CanonEntry represents a parsed row from a vocabulary canonicalization table.
type CanonEntry struct {
	Term                    string
	PlainLanguage           string
	NearestExistingConcepts string
	ClaimedDelta            string
	Verdict                 string
}

// CheckModel runs model gate checks on a model.md file.
// Validates claim ledger and vocabulary canonicalization tables.
func CheckModel(modelPath string) GateResult {
	result := GateResult{Pass: true}

	content, err := os.ReadFile(modelPath)
	if err != nil {
		result.Pass = false
		result.Verdicts = append(result.Verdicts, Verdict{
			Code:   "FILE_NOT_FOUND",
			Status: "fail",
			Note:   fmt.Sprintf("cannot read model: %v", err),
		})
		return result
	}

	text := string(content)

	// Check 1: Claim ledger table presence and schema
	claims := checkModelClaimTable(text, &result)

	// Check 2: Vocabulary canonicalization table presence and schema
	canonEntries := checkModelCanonTable(text, &result)

	// Check 3: Validate individual claim entries
	if claims != nil {
		checkModelClaimEntries(claims, &result)
	}

	// Check 4: Vocabulary inflation warnings
	if canonEntries != nil {
		checkVocabInflation(canonEntries, &result)
	}

	return result
}

// checkModelClaimTable validates the claim ledger table exists and has required columns.
func checkModelClaimTable(content string, result *GateResult) []ModelClaimEntry {
	table := findMarkdownTable(content, "claim_id")
	if table == nil {
		result.Pass = false
		result.Verdicts = append(result.Verdicts, Verdict{
			Code:      "MISSING_CLAIM_TABLE",
			Status:    "fail",
			AppliesTo: "model",
			Note:      "model must contain a claim ledger table with columns: " + strings.Join(requiredClaimColumns, ", "),
		})
		return nil
	}

	// Validate required columns
	for _, col := range requiredClaimColumns {
		if !table.hasColumn(col) {
			result.Pass = false
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "INVALID_CLAIM_TABLE",
				Status:    "fail",
				AppliesTo: "model",
				Note:      fmt.Sprintf("claim table missing required column: %s", col),
			})
			return nil
		}
	}

	// Parse rows into entries
	var entries []ModelClaimEntry
	for _, row := range table.rows {
		entries = append(entries, ModelClaimEntry{
			ClaimID:      row[table.colIndex("claim_id")],
			ClaimText:    row[table.colIndex("claim_text")],
			ClaimType:    row[table.colIndex("claim_type")],
			Scope:        row[table.colIndex("scope")],
			NoveltyLevel: row[table.colIndex("novelty_level")],
			EvidenceRefs: row[table.colIndex("evidence_refs")],
		})
	}

	return entries
}

// checkModelCanonTable validates the vocabulary canonicalization table.
func checkModelCanonTable(content string, result *GateResult) []CanonEntry {
	table := findMarkdownTable(content, "plain_language")
	if table == nil {
		result.Pass = false
		result.Verdicts = append(result.Verdicts, Verdict{
			Code:      "MISSING_CANON_TABLE",
			Status:    "fail",
			AppliesTo: "model",
			Note:      "model must contain a vocabulary canonicalization table with columns: " + strings.Join(requiredCanonColumns, ", "),
		})
		return nil
	}

	// Validate required columns
	for _, col := range requiredCanonColumns {
		if !table.hasColumn(col) {
			result.Pass = false
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "INVALID_CANON_TABLE",
				Status:    "fail",
				AppliesTo: "model",
				Note:      fmt.Sprintf("canonicalization table missing required column: %s", col),
			})
			return nil
		}
	}

	var entries []CanonEntry
	for _, row := range table.rows {
		entries = append(entries, CanonEntry{
			Term:                    row[table.colIndex("term")],
			PlainLanguage:           row[table.colIndex("plain_language")],
			NearestExistingConcepts: row[table.colIndex("nearest_existing_concepts")],
			ClaimedDelta:            row[table.colIndex("claimed_delta")],
			Verdict:                 row[table.colIndex("verdict")],
		})
	}

	return entries
}

// checkModelClaimEntries validates individual claim entries.
func checkModelClaimEntries(claims []ModelClaimEntry, result *GateResult) {
	for _, c := range claims {
		if !validModelClaimTypes[c.ClaimType] {
			result.Pass = false
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "INVALID_CLAIM_ENTRY",
				Status:    "fail",
				AppliesTo: c.ClaimID,
				Note:      fmt.Sprintf("invalid claim_type %q — must be one of: observation, mechanism, generalization, recommendation", c.ClaimType),
			})
		}
		if !validNoveltyLevels[c.NoveltyLevel] {
			result.Pass = false
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "INVALID_CLAIM_ENTRY",
				Status:    "fail",
				AppliesTo: c.ClaimID,
				Note:      fmt.Sprintf("invalid novelty_level %q — must be one of: restatement, synthesis, novel", c.NoveltyLevel),
			})
		}

		// Non-observation claims need evidence
		if c.ClaimType != "observation" && strings.TrimSpace(c.EvidenceRefs) == "" {
			result.Pass = false
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "MISSING_EVIDENCE",
				Status:    "fail",
				AppliesTo: c.ClaimID,
				Note:      "non-observation claims must have evidence_refs",
			})
		}

		// Warn on endogenous-only evidence for generalization/novel claims
		if (c.ClaimType == "generalization" || c.NoveltyLevel == "novel") && strings.TrimSpace(c.EvidenceRefs) != "" {
			refs := strings.Split(c.EvidenceRefs, ",")
			allEndogenous := true
			for _, ref := range refs {
				ref = strings.TrimSpace(ref)
				if ref == "" {
					continue
				}
				if !isModelEndogenousRef(ref) {
					allEndogenous = false
					break
				}
			}
			if allEndogenous {
				// Warning, not failure — model gate surfaces early, publish gate blocks
				result.Verdicts = append(result.Verdicts, Verdict{
					Code:      "ENDOGENOUS_EVIDENCE_WARNING",
					Status:    "warn",
					AppliesTo: c.ClaimID,
					Note:      "claim references only models/probes — will fail publication lineage gate",
				})
			}
		}
	}
}

// isModelEndogenousRef checks if a ref points to model/probe artifacts.
func isModelEndogenousRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	return strings.HasPrefix(ref, "model:") || strings.HasPrefix(ref, "probe:")
}

// checkVocabInflation validates canonicalization entries for vocabulary inflation.
func checkVocabInflation(entries []CanonEntry, result *GateResult) {
	for _, e := range entries {
		// Missing prior-art mapping
		if strings.TrimSpace(e.NearestExistingConcepts) == "" {
			result.Pass = false
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "MISSING_PRIOR_ART",
				Status:    "fail",
				AppliesTo: e.Term,
				Note:      "nearest_existing_concepts must not be empty — map to known concepts",
			})
		}

		// Empty claimed_delta with non-empty prior art = vocabulary inflation
		if strings.TrimSpace(e.ClaimedDelta) == "" && strings.TrimSpace(e.NearestExistingConcepts) != "" {
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "VOCABULARY_INFLATION",
				Status:    "warn",
				AppliesTo: e.Term,
				Note:      "claimed_delta is empty — term may be a restatement of existing concepts with no predictive residue",
			})
		}
	}
}

// mdTable represents a parsed markdown table.
type mdTable struct {
	headers []string
	rows    [][]string
}

func (t *mdTable) hasColumn(name string) bool {
	for _, h := range t.headers {
		if h == name {
			return true
		}
	}
	return false
}

func (t *mdTable) colIndex(name string) int {
	for i, h := range t.headers {
		if h == name {
			return i
		}
	}
	return -1
}

// findMarkdownTable finds the first markdown table containing a specific column name.
func findMarkdownTable(content string, identifyingColumn string) *mdTable {
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			continue
		}

		// Parse header row
		headers := parseTableRow(line)
		hasIdentifier := false
		for _, h := range headers {
			if h == identifyingColumn {
				hasIdentifier = true
				break
			}
		}
		if !hasIdentifier {
			continue
		}

		// Skip separator row
		if i+1 < len(lines) {
			sep := strings.TrimSpace(lines[i+1])
			if strings.HasPrefix(sep, "|") && strings.Contains(sep, "-") {
				i++ // skip separator
			}
		}

		// Parse data rows
		table := &mdTable{headers: headers}
		for i+1 < len(lines) {
			i++
			row := strings.TrimSpace(lines[i])
			if !strings.HasPrefix(row, "|") || !strings.HasSuffix(row, "|") {
				break
			}
			// Skip if it looks like another separator
			if strings.Contains(row, "---") {
				continue
			}
			cells := parseTableRow(row)
			// Pad or trim to match header count
			for len(cells) < len(headers) {
				cells = append(cells, "")
			}
			table.rows = append(table.rows, cells[:len(headers)])
		}

		return table
	}

	return nil
}

// parseTableRow splits a markdown table row into cells.
func parseTableRow(line string) []string {
	// Trim leading/trailing |
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// FormatModelResult produces a human-readable summary of model gate results.
func FormatModelResult(result GateResult) string {
	var sb strings.Builder

	if result.Pass {
		sb.WriteString("✓ Model gate: PASS\n")
	} else {
		sb.WriteString("✗ Model gate: FAIL\n")
	}

	if len(result.Verdicts) > 0 {
		sb.WriteString("\nVerdicts:\n")
		for _, v := range result.Verdicts {
			icon := "✓"
			if v.Status == "fail" {
				icon = "✗"
			} else if v.Status == "warn" {
				icon = "⚠"
			}
			if v.AppliesTo != "" {
				sb.WriteString(fmt.Sprintf("  %s %-30s [%s] %s\n", icon, v.Code, v.AppliesTo, v.Note))
			} else {
				sb.WriteString(fmt.Sprintf("  %s %-30s %s\n", icon, v.Code, v.Note))
			}
		}
	}

	return sb.String()
}
