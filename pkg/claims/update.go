package claims

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ProbeClaimRef represents a claim reference extracted from a probe file.
type ProbeClaimRef struct {
	ClaimID   string // e.g., "AE-08"
	ModelName string // e.g., "architectural-enforcement"
	Verdict   string // confirms, contradicts, extends
	Source    string // Evidence source description
}

var claimRefRe = regexp.MustCompile(`(?i)claim:\s*([A-Z]+-\d+)`)
var impactClaimRefRe = regexp.MustCompile(`(?mi)\*\*(?:Confirms|Contradicts|Extends)\*\*[^A-Z0-9]*([A-Z]+-\d+)`)

// ExtractClaimRef extracts a claim reference from probe investigation content.
// Looks first for "claim: XX-NN" in frontmatter/body, then falls back to
// claim IDs embedded in Model Impact verdict lines.
func ExtractClaimRef(content string) *ProbeClaimRef {
	matches := claimRefRe.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return &ProbeClaimRef{
			ClaimID: matches[1],
		}
	}

	impactSection := extractSection(content, "Model Impact")
	if impactSection == "" {
		return nil
	}

	matches = impactClaimRefRe.FindStringSubmatch(impactSection)
	if len(matches) < 2 {
		return nil
	}

	return &ProbeClaimRef{
		ClaimID: matches[1],
	}
}

func extractSection(content, sectionName string) string {
	pattern := regexp.MustCompile(`(?s)## ` + regexp.QuoteMeta(sectionName) + `\s*\n(.*?)(?:\n---\n|\n## |\z)`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// CheckEvidenceIndependence checks if a probe's evidence source overlaps with
// existing evidence on a claim. Returns true if overlap is detected (self-validating).
// This prevents circular validation where a probe "confirms" using the same data
// the claim already cites.
func CheckEvidenceIndependence(claim Claim, probeSource string) bool {
	if len(claim.Evidence) == 0 {
		return false
	}
	probeLower := strings.ToLower(probeSource)
	// Extract significant words (4+ chars) from probe source
	probeWords := extractSignificantWords(probeLower)
	if len(probeWords) == 0 {
		return false
	}

	for _, ev := range claim.Evidence {
		evLower := strings.ToLower(ev.Source)
		evWords := extractSignificantWords(evLower)
		// Count overlapping significant words
		overlap := 0
		for _, pw := range probeWords {
			for _, ew := range evWords {
				if pw == ew {
					overlap++
					break
				}
			}
		}
		// If >40% of probe's significant words match an existing source, flag it
		if len(probeWords) > 0 && float64(overlap)/float64(len(probeWords)) > 0.4 {
			return true
		}
	}
	return false
}

// extractSignificantWords returns words of 4+ characters, lowercased, excluding
// common stop words and date patterns.
func extractSignificantWords(s string) []string {
	stopWords := map[string]bool{
		"probe": true, "from": true, "with": true, "that": true, "this": true,
		"have": true, "been": true, "were": true, "will": true, "when": true,
		"2026": true, "2025": true, "2024": true,
	}
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	var result []string
	for _, w := range words {
		if len(w) >= 4 && !stopWords[w] {
			result = append(result, w)
		}
	}
	return result
}

// UpdateResult describes what happened when updating a claim.
type UpdateResult struct {
	ClaimID   string
	ModelName string
	Action    string // "confirmed", "contested", "extended", "not_found"
	Message   string
}

// ApplyProbeVerdict updates claims.yaml for a model based on a probe's verdict.
// Returns the update result and any error.
func ApplyProbeVerdict(modelsDir string, ref ProbeClaimRef, now time.Time) (*UpdateResult, error) {
	if ref.ClaimID == "" || ref.ModelName == "" {
		return nil, fmt.Errorf("missing claim ID or model name")
	}

	claimsPath := filepath.Join(modelsDir, ref.ModelName, "claims.yaml")
	f, err := LoadFile(claimsPath)
	if err != nil {
		return nil, fmt.Errorf("load claims for model %s: %w", ref.ModelName, err)
	}

	// Find the claim
	var claim *Claim
	for i := range f.Claims {
		if f.Claims[i].ID == ref.ClaimID {
			claim = &f.Claims[i]
			break
		}
	}
	if claim == nil {
		return &UpdateResult{
			ClaimID:   ref.ClaimID,
			ModelName: ref.ModelName,
			Action:    "not_found",
			Message:   fmt.Sprintf("claim %s not found in model %s", ref.ClaimID, ref.ModelName),
		}, nil
	}

	dateStr := now.Format("2006-01-02")
	result := &UpdateResult{
		ClaimID:   ref.ClaimID,
		ModelName: ref.ModelName,
	}

	switch strings.ToLower(ref.Verdict) {
	case "confirms":
		// Safeguard 1: Evidence independence check
		if CheckEvidenceIndependence(*claim, ref.Source) {
			result.Action = "self_validating"
			result.Message = fmt.Sprintf("claim %s probe evidence overlaps existing sources — flagged as self-validating", ref.ClaimID)
			// Do NOT update confidence or save — return early
			return result, nil
		}
		claim.Confidence = Confirmed
		claim.LastValidated = dateStr
		claim.Evidence = append(claim.Evidence, Evidence{
			Source:  ref.Source,
			Date:    dateStr,
			Verdict: "confirms",
		})
		result.Action = "confirmed"
		result.Message = fmt.Sprintf("claim %s confirmed, last_validated updated to %s", ref.ClaimID, dateStr)

	case "contradicts":
		claim.Confidence = Contested
		claim.Evidence = append(claim.Evidence, Evidence{
			Source:  ref.Source,
			Date:    dateStr,
			Verdict: "contradicts",
		})
		result.Action = "contested"
		result.Message = fmt.Sprintf("claim %s contested — requires orchestrator review", ref.ClaimID)

	case "extends":
		claim.LastValidated = dateStr
		claim.Evidence = append(claim.Evidence, Evidence{
			Source:  ref.Source,
			Date:    dateStr,
			Verdict: "extends",
		})
		result.Action = "extended"
		result.Message = fmt.Sprintf("claim %s extended with new evidence", ref.ClaimID)

	default:
		return &UpdateResult{
			ClaimID:   ref.ClaimID,
			ModelName: ref.ModelName,
			Action:    "unknown_verdict",
			Message:   fmt.Sprintf("unknown verdict %q for claim %s", ref.Verdict, ref.ClaimID),
		}, nil
	}

	// Save updated claims.yaml
	if err := SaveFile(claimsPath, f); err != nil {
		return nil, fmt.Errorf("save claims for model %s: %w", ref.ModelName, err)
	}

	return result, nil
}
