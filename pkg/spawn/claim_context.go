package spawn

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/claims"
)

// ClaimContext holds the claim details to inject into SPAWN_CONTEXT.md
// when a probe agent is spawned for a specific claim.
type ClaimContext struct {
	ClaimID     string
	ModelName   string
	ClaimText   string
	FalsifiesIf string
	Evidence    []claims.Evidence
	ClaimsFile  string // path to claims.yaml for reference
}

// ExtractClaimIDFromLabels scans beads labels for a "claim:XX-NN" label
// and returns the claim ID. Returns empty string if no claim label found.
func ExtractClaimIDFromLabels(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, "claim:") {
			return strings.TrimPrefix(label, "claim:")
		}
	}
	return ""
}

// LookupClaimContext finds the claim details for a given claim ID by scanning
// all claims.yaml files under .kb/models/. Returns nil if not found.
func LookupClaimContext(claimID, projectDir string) *ClaimContext {
	if claimID == "" || projectDir == "" {
		return nil
	}

	modelsDir := filepath.Join(projectDir, ".kb", "models")
	files, err := claims.ScanAll(modelsDir)
	if err != nil || len(files) == 0 {
		return nil
	}

	for modelName, f := range files {
		for _, c := range f.Claims {
			if c.ID == claimID {
				return &ClaimContext{
					ClaimID:     c.ID,
					ModelName:   modelName,
					ClaimText:   c.Text,
					FalsifiesIf: c.FalsifiesIf,
					Evidence:    c.Evidence,
					ClaimsFile:  filepath.Join(modelsDir, modelName, "claims.yaml"),
				}
			}
		}
	}
	return nil
}

// FormatClaimContext renders the claim context as a markdown section for SPAWN_CONTEXT.md.
func FormatClaimContext(cc *ClaimContext) string {
	if cc == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## CLAIM PROBE CONTEXT\n\n")
	sb.WriteString(fmt.Sprintf("**Claim ID:** %s\n", cc.ClaimID))
	sb.WriteString(fmt.Sprintf("**Model:** %s\n", cc.ModelName))
	sb.WriteString(fmt.Sprintf("**Claim:** %s\n", cc.ClaimText))
	sb.WriteString(fmt.Sprintf("**Falsifies if:** %s\n", cc.FalsifiesIf))

	if len(cc.Evidence) > 0 {
		sb.WriteString("\n**Existing evidence (provide INDEPENDENT evidence — do not re-cite these):**\n")
		for _, e := range cc.Evidence {
			sb.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", e.Verdict, e.Source, e.Date))
		}
	}

	sb.WriteString(fmt.Sprintf("\n**Claims file:** `%s`\n", cc.ClaimsFile))
	sb.WriteString("\n**Evidence independence constraint:** Your probe must provide evidence from sources NOT already cited above. Re-citing existing evidence is self-validating and will be flagged.\n")

	return sb.String()
}
