package kbgate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ClaimHit represents a single detected claim-upgrade signal.
type ClaimHit struct {
	File  string `json:"file"`
	Line  int    `json:"line"`
	Match string `json:"match"`
	Code  string `json:"code"` // NOVELTY_LANGUAGE, SELF_VALIDATING_PROBE, CAUSAL_LANGUAGE
}

func (h ClaimHit) String() string {
	return fmt.Sprintf("[%s] %s:%d: %s", h.Code, h.File, h.Line, h.Match)
}

// ClaimScanResult aggregates results from all three scanners.
type ClaimScanResult struct {
	Novelty          []ClaimHit `json:"novelty"`
	ProbeConclusions []ClaimHit `json:"probe_conclusions"`
	CausalLanguage   []ClaimHit `json:"causal_language"`
}

// Total returns the total number of claim hits across all scanners.
func (r ClaimScanResult) Total() int {
	return len(r.Novelty) + len(r.ProbeConclusions) + len(r.CausalLanguage)
}

// noveltyPatterns are regex patterns for novelty language detection.
// Each pattern is compiled once and reused.
var noveltyPatterns = []struct {
	re   *regexp.Regexp
	term string
}{
	{regexp.MustCompile(`(?i)\bnovel\b`), "novel"},
	{regexp.MustCompile(`(?i)\bfirst\b`), "first"},
	{regexp.MustCompile(`(?i)\bnew framework\b`), "new framework"},
	{regexp.MustCompile(`(?i)\bsubstrate-independent\b`), "substrate-independent"},
	{regexp.MustCompile(`(?i)\bphysics\b`), "physics"},
	{regexp.MustCompile(`(?i)\bdiscovered\b`), "discovered"},
	{regexp.MustCompile(`(?i)absent from\b`), "absent from"},
	{regexp.MustCompile(`(?i)\bnew discipline\b`), "new discipline"},
}

// causalPatterns are regex patterns for causal language detection.
var causalPatterns = []struct {
	re   *regexp.Regexp
	term string
}{
	{regexp.MustCompile(`(?i)\bpredict[s]?\b`), "predict"},
	{regexp.MustCompile(`(?i)\bcause[s]?\b`), "cause"},
	{regexp.MustCompile(`(?i)\bproduce[s]?\b`), "produce"},
	{regexp.MustCompile(`(?i)\bdetermine[s]?\b`), "determine"},
	{regexp.MustCompile(`(?i)\bguarantee[s]?\b`), "guarantee"},
	{regexp.MustCompile(`(?i)\bensure[s]?\b`), "ensure"},
	{regexp.MustCompile(`(?i)\balways\b`), "always"},
	{regexp.MustCompile(`(?i)\bnever\b`), "never"},
}

// probeVerdictRe matches "confirms" or "extends" (with optional bold markdown).
var probeVerdictRe = regexp.MustCompile(`(?i)\b(confirms?|extends?)\b`)

// externalCitationRe detects URLs or parenthetical citations like (Author, Year).
var externalCitationRe = regexp.MustCompile(`https?://|(?:\([A-Z][a-z]+(?:\s+(?:et al\.)?)?,?\s*\d{4}\))`)

// ScanNoveltyLanguage scans publications and model.md files for novelty claims.
func ScanNoveltyLanguage(kbDir string) []ClaimHit {
	var hits []ClaimHit

	// Scan publications
	pubDir := filepath.Join(kbDir, "publications")
	pubFiles, _ := filepath.Glob(filepath.Join(pubDir, "*.md"))
	for _, f := range pubFiles {
		hits = append(hits, scanFileForNovelty(f)...)
	}

	// Scan model.md files
	modelFiles, _ := filepath.Glob(filepath.Join(kbDir, "models", "*", "model.md"))
	for _, f := range modelFiles {
		hits = append(hits, scanFileForNovelty(f)...)
	}

	return hits
}

func scanFileForNovelty(path string) []ClaimHit {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var hits []ClaimHit

	for i, line := range lines {
		for _, p := range noveltyPatterns {
			if !p.re.MatchString(line) {
				continue
			}

			// Skip "first" when used as ordinal (e.g., "first step", "first action")
			if p.term == "first" && isOrdinalFirst(line) {
				continue
			}

			// Skip "physics" if it only appears in a heading that matches the filename
			if p.term == "physics" && isPhysicsInTitle(line, path) {
				continue
			}

			hits = append(hits, ClaimHit{
				File:  path,
				Line:  i + 1,
				Match: strings.TrimSpace(line),
				Code:  "NOVELTY_LANGUAGE",
			})
			break // one hit per line
		}
	}

	return hits
}

// isOrdinalFirst returns true if "first" is used as an ordinal, not a novelty claim.
// e.g., "first step", "first action", "first 3 actions" — not "first systematic treatment".
var ordinalFirstRe = regexp.MustCompile(`(?i)\bfirst\s+(?:step|action|time|thing|call|attempt|try|pass|round|iteration|\d)`)

func isOrdinalFirst(line string) bool {
	return ordinalFirstRe.MatchString(line)
}

// isPhysicsInTitle returns true if "physics" appears only in a markdown heading
// that matches the filename/directory name (e.g., "# Knowledge Physics" in knowledge-physics/model.md).
func isPhysicsInTitle(line, path string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return false
	}
	// Check if the directory name contains "physics"
	dir := filepath.Dir(path)
	return strings.Contains(strings.ToLower(filepath.Base(dir)), "physics")
}

// ScanProbeConclusions scans probe files for confirms/extends in Model Impact sections
// without external citations.
func ScanProbeConclusions(kbDir string) []ClaimHit {
	var hits []ClaimHit

	probeFiles, _ := filepath.Glob(filepath.Join(kbDir, "models", "*", "probes", "*.md"))
	for _, f := range probeFiles {
		hits = append(hits, scanProbeFile(f)...)
	}

	return hits
}

func scanProbeFile(path string) []ClaimHit {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var hits []ClaimHit
	inModelImpact := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track section boundaries
		if strings.HasPrefix(trimmed, "## ") {
			inModelImpact = strings.Contains(strings.ToLower(trimmed), "model impact")
			continue
		}

		if !inModelImpact {
			continue
		}

		// Look for confirms/extends
		if !probeVerdictRe.MatchString(line) {
			continue
		}

		// Check for external citation on the same line
		if externalCitationRe.MatchString(line) {
			continue
		}

		hits = append(hits, ClaimHit{
			File:  path,
			Line:  i + 1,
			Match: strings.TrimSpace(line),
			Code:  "SELF_VALIDATING_PROBE",
		})
	}

	return hits
}

// ScanCausalLanguage scans model.md Summary sections for causal claims.
func ScanCausalLanguage(kbDir string) []ClaimHit {
	var hits []ClaimHit

	modelFiles, _ := filepath.Glob(filepath.Join(kbDir, "models", "*", "model.md"))
	for _, f := range modelFiles {
		hits = append(hits, scanModelForCausal(f)...)
	}

	return hits
}

func scanModelForCausal(path string) []ClaimHit {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var hits []ClaimHit
	inSummary := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track section boundaries
		if strings.HasPrefix(trimmed, "## ") {
			inSummary = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "##")), "Summary")
			continue
		}

		if !inSummary {
			continue
		}

		for _, p := range causalPatterns {
			if p.re.MatchString(line) {
				hits = append(hits, ClaimHit{
					File:  path,
					Line:  i + 1,
					Match: strings.TrimSpace(line),
					Code:  "CAUSAL_LANGUAGE",
				})
				break // one hit per line
			}
		}
	}

	return hits
}

// ScanAllClaims runs all three scanners and returns aggregated results.
func ScanAllClaims(kbDir string) ClaimScanResult {
	return ClaimScanResult{
		Novelty:          ScanNoveltyLanguage(kbDir),
		ProbeConclusions: ScanProbeConclusions(kbDir),
		CausalLanguage:   ScanCausalLanguage(kbDir),
	}
}

// ScanFile runs all three claim scanners against a single file.
// Used by the publish gate to scope signals to the target publication.
func ScanFile(path string) ClaimScanResult {
	return ClaimScanResult{
		Novelty:          scanFileForNovelty(path),
		ProbeConclusions: scanProbeFile(path),
		CausalLanguage:   scanModelForCausal(path),
	}
}

// maxExamplesPerCategory is the number of example hits shown per signal category.
// Remaining hits are summarized as a count.
const maxExamplesPerCategory = 3

// FormatClaimScanResult produces a human-readable summary of claim scan results.
// Shows counts per category with up to maxExamplesPerCategory examples each.
func FormatClaimScanResult(result ClaimScanResult) string {
	var sb strings.Builder

	total := result.Total()
	if total == 0 {
		sb.WriteString("No claim-upgrade signals detected.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("⚠ Found %d claim-upgrade signal(s)\n\n", total))

	formatCategory(&sb, "Novelty Language", result.Novelty)
	formatCategory(&sb, "Self-Validating Probes", result.ProbeConclusions)
	formatCategory(&sb, "Causal Language", result.CausalLanguage)

	sb.WriteString("To proceed with publish, use --acknowledge-claims or reference an external review artifact.\n")

	return sb.String()
}

// formatCategory writes a summarized category section: count + top N examples.
func formatCategory(sb *strings.Builder, name string, hits []ClaimHit) {
	if len(hits) == 0 {
		return
	}

	sb.WriteString(fmt.Sprintf("── %s (%d) ──\n", name, len(hits)))

	limit := len(hits)
	if limit > maxExamplesPerCategory {
		limit = maxExamplesPerCategory
	}
	for _, h := range hits[:limit] {
		sb.WriteString(fmt.Sprintf("  %s:%d: %s\n", h.File, h.Line, h.Match))
	}
	if remaining := len(hits) - limit; remaining > 0 {
		sb.WriteString(fmt.Sprintf("  ... and %d more\n", remaining))
	}
	sb.WriteString("\n")
}
