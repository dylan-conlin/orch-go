package research

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/claims"
)

// TestStatus represents the aggregate test status for a claim.
type TestStatus string

const (
	StatusUntested      TestStatus = "untested"
	StatusConfirmed     TestStatus = "confirmed"
	StatusContradicted  TestStatus = "contradicted"
	StatusExtended      TestStatus = "extended"
	StatusMixed         TestStatus = "mixed" // multiple probes with different verdicts
)

// ClaimStatus aggregates a single claim with its probe results.
type ClaimStatus struct {
	ID          string
	Text        string
	Confidence  string // from claims.yaml (confirmed, unconfirmed, etc.)
	Priority    string // from claims.yaml (core, supporting, peripheral)
	HowToVerify string // from model.md table
	Probes      []ProbeResult
	TestStatus  TestStatus
}

// ModelStatus holds all claims and their status for a single model.
type ModelStatus struct {
	ModelName    string
	Claims       []ClaimStatus
	TotalClaims  int
	TestedClaims int
	Source        string // "claims.yaml", "model.md", "both"
}

// LoadModelStatus loads claims and probes for a single model directory,
// cross-references them, and returns the aggregated status.
func LoadModelStatus(modelDir string) (*ModelStatus, error) {
	modelName := filepath.Base(modelDir)
	ms := &ModelStatus{ModelName: modelName}

	// Load claims from claims.yaml
	yamlClaims := loadYAMLClaims(modelDir)

	// Load claims from model.md markdown table
	mdClaims := loadMarkdownClaims(modelDir)

	// Load probes
	probesDir := filepath.Join(modelDir, "probes")
	probes, _ := ScanProbes(probesDir) // nil if no probes dir

	// Build claim status list
	if len(yamlClaims) > 0 && len(mdClaims) > 0 {
		ms.Source = "both"
	} else if len(yamlClaims) > 0 {
		ms.Source = "claims.yaml"
	} else if len(mdClaims) > 0 {
		ms.Source = "model.md"
	} else {
		return nil, nil // no claims
	}

	// Index probes by claim ID for fast lookup
	probeIndex := indexProbesByClaim(probes)

	// Build unified claim list — prefer claims.yaml, augment with model.md
	seen := make(map[string]bool)

	for _, yc := range yamlClaims {
		id := strings.ToUpper(yc.ID)
		seen[id] = true
		cs := ClaimStatus{
			ID:         id,
			Text:       truncateText(yc.Text, 120),
			Confidence: string(yc.Confidence),
			Priority:   string(yc.Priority),
			Probes:     probeIndex[id],
		}
		// Add HowToVerify from model.md if available
		for _, mc := range mdClaims {
			if strings.ToUpper(mc.ID) == id {
				cs.HowToVerify = mc.HowToVerify
				break
			}
		}
		cs.TestStatus = deriveTestStatus(cs)
		ms.Claims = append(ms.Claims, cs)
	}

	// Add any claims from model.md not in claims.yaml
	for _, mc := range mdClaims {
		id := strings.ToUpper(mc.ID)
		if seen[id] {
			continue
		}
		cs := ClaimStatus{
			ID:          id,
			Text:        truncateText(mc.Text, 120),
			HowToVerify: mc.HowToVerify,
			Probes:      probeIndex[id],
		}
		cs.TestStatus = deriveTestStatus(cs)
		ms.Claims = append(ms.Claims, cs)
	}

	ms.TotalClaims = len(ms.Claims)
	for _, cs := range ms.Claims {
		if cs.TestStatus != StatusUntested {
			ms.TestedClaims++
		}
	}

	return ms, nil
}

// LoadAllModels scans all model directories under kbDir/models/ and returns
// their research status. Skips models with no claims.
func LoadAllModels(kbDir string) ([]ModelStatus, error) {
	modelsDir := filepath.Join(kbDir, "models")
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read models dir: %w", err)
	}

	var results []ModelStatus
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip archived and template directories
		if entry.Name() == "archived" || entry.Name() == "TEMPLATE.md" {
			continue
		}
		modelDir := filepath.Join(modelsDir, entry.Name())
		ms, err := LoadModelStatus(modelDir)
		if err != nil || ms == nil {
			continue
		}
		results = append(results, *ms)
	}

	return results, nil
}

// FindModel finds a model by name (exact or prefix match).
func FindModel(kbDir, name string) (string, error) {
	modelsDir := filepath.Join(kbDir, "models")
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return "", fmt.Errorf("read models dir: %w", err)
	}

	// Exact match first
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == name {
			return filepath.Join(modelsDir, entry.Name()), nil
		}
	}

	// Prefix match
	var matches []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), name) {
			matches = append(matches, filepath.Join(modelsDir, entry.Name()))
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = filepath.Base(m)
		}
		return "", fmt.Errorf("ambiguous model name %q matches: %s", name, strings.Join(names, ", "))
	}

	return "", fmt.Errorf("model not found: %s", name)
}

// FindClaim finds a claim by ID within a model's claims.
func FindClaim(ms *ModelStatus, claimID string) *ClaimStatus {
	upper := strings.ToUpper(claimID)
	for i := range ms.Claims {
		if ms.Claims[i].ID == upper {
			return &ms.Claims[i]
		}
	}
	return nil
}

// loadYAMLClaims loads claims from claims.yaml if it exists.
func loadYAMLClaims(modelDir string) []claims.Claim {
	path := filepath.Join(modelDir, "claims.yaml")
	f, err := claims.LoadFile(path)
	if err != nil {
		return nil
	}
	return f.Claims
}

// loadMarkdownClaims loads claims from model.md if it exists.
func loadMarkdownClaims(modelDir string) []MarkdownClaim {
	path := filepath.Join(modelDir, "model.md")
	mcs, err := ParseMarkdownClaims(path)
	if err != nil {
		return nil
	}
	return mcs
}

// indexProbesByClaim builds a map from claim ID to its probe results.
func indexProbesByClaim(probes []ProbeResult) map[string][]ProbeResult {
	index := make(map[string][]ProbeResult)
	for _, p := range probes {
		for _, cid := range p.Claims {
			index[cid] = append(index[cid], p)
		}
	}
	return index
}

// deriveTestStatus determines the aggregate test status from probes and confidence.
func deriveTestStatus(cs ClaimStatus) TestStatus {
	if len(cs.Probes) == 0 {
		// If claims.yaml says confirmed but no probes reference it,
		// trust the YAML — evidence may predate structured probes
		if cs.Confidence == "confirmed" {
			return StatusConfirmed
		}
		return StatusUntested
	}

	verdicts := make(map[string]int)
	for _, p := range cs.Probes {
		if p.Verdict != "" {
			verdicts[p.Verdict]++
		}
	}

	if len(verdicts) == 0 {
		return StatusUntested
	}

	// Single verdict type
	if len(verdicts) == 1 {
		for v := range verdicts {
			switch v {
			case "confirms":
				return StatusConfirmed
			case "contradicts", "disconfirms":
				return StatusContradicted
			case "extends", "scopes":
				return StatusExtended
			}
		}
	}

	return StatusMixed
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
