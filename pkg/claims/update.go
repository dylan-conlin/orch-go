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

// ExtractClaimRef extracts a claim reference from probe investigation content.
// Looks for "claim: XX-NN" in frontmatter or body.
func ExtractClaimRef(content string) *ProbeClaimRef {
	matches := claimRefRe.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil
	}
	return &ProbeClaimRef{
		ClaimID: matches[1],
	}
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
