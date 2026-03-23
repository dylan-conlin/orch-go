package kbmetrics

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ProvenanceReport summarizes evidence quality gaps for one model.
type ProvenanceReport struct {
	Name                 string                `json:"name"`
	Path                 string                `json:"path"`
	TotalClaims          int                   `json:"total_claims"`
	AnnotatedClaims      int                   `json:"annotated_claims"`
	CoveragePercent      float64               `json:"coverage_percent"`
	UnannotatedClaims    []UnannotatedClaim    `json:"unannotated_claims,omitempty"`
	LowConfidenceClaims  []LowConfidenceClaim  `json:"low_confidence_claims,omitempty"`
	OrphanContradictions []OrphanContradiction `json:"orphan_contradictions,omitempty"`
	DriftFlags           []DriftFlag           `json:"drift_flags,omitempty"`
}

// UnannotatedClaim is a claim missing an **Evidence quality:** annotation.
type UnannotatedClaim struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}

// LowConfidenceClaim is a claim annotated as single-source or assumed.
type LowConfidenceClaim struct {
	Line  int    `json:"line"`
	Text  string `json:"text"`
	Level string `json:"level"` // "single-source" or "assumed"
}

// OrphanContradiction is a probe that contradicts a model claim but the
// model wasn't updated after the probe date.
type OrphanContradiction struct {
	ProbePath        string `json:"probe_path"`
	ProbeDate        string `json:"probe_date"`
	ContradictionText string `json:"contradiction_text"`
}

var (
	// Evidence quality annotation line
	evidenceQualityRe = regexp.MustCompile(`(?i)^\*\*Evidence quality:\*\*\s*(.+)`)

	// Last Updated metadata
	lastUpdatedRe = regexp.MustCompile(`(?i)^\*\*Last Updated:\*\*\s*(\d{4}-\d{2}-\d{2})`)

	// Probe contradiction marker: "- [x] **Contradicts**"
	probeContradictRe = regexp.MustCompile(`^\s*-\s*\[x\]\s*\*\*Contradicts\*\*\s*(.*)`)

	// Probe date from filename: "2026-02-15-probe-whatever.md"
	probeDateRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-`)

	// Section-level claim headings: "### Claim N:" or "### Invariant N:"
	claimHeadingRe = regexp.MustCompile(`^###\s+(Claim|Invariant)\s+\d+`)

	// Low confidence levels
	lowConfidenceLevels = []string{"single-source", "assumed"}
)

// AuditProvenance scans all model.md files under kbDir/models/ for evidence
// quality gaps: unannotated claims, low-confidence claims, and orphan
// contradictions (probes that contradict but model wasn't updated).
func AuditProvenance(kbDir string) ([]ProvenanceReport, error) {
	modelsDir := filepath.Join(kbDir, "models")
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, fmt.Errorf("read models dir: %w", err)
	}

	var reports []ProvenanceReport

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modelPath := filepath.Join(modelsDir, entry.Name(), "model.md")
		data, err := os.ReadFile(modelPath)
		if err != nil {
			continue // skip models without model.md
		}

		report := analyzeModelProvenance(entry.Name(), modelPath, string(data))

		// Check for orphan contradictions in probes
		probeDir := filepath.Join(modelsDir, entry.Name(), "probes")
		report.OrphanContradictions = findOrphanContradictions(probeDir, string(data))

		reports = append(reports, report)
	}

	// Sort by coverage ascending (worst first)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].CoveragePercent < reports[j].CoveragePercent
	})

	return reports, nil
}

// analyzeModelProvenance extracts claims and checks for evidence quality
// annotations. A claim is "annotated" if an **Evidence quality:** line
// appears between it and the next claim (or end of section).
//
// Claims are detected from two sources:
// 1. ExtractClaims (numbered items, bold bullets, table rows, etc.)
// 2. Section-level claim headings (### Claim N, ### Invariant N)
func analyzeModelProvenance(name, path, content string) ProvenanceReport {
	claims := extractProvenanceClaims(content)

	report := ProvenanceReport{
		Name:        name,
		Path:        path,
		TotalClaims: len(claims),
	}

	if len(claims) == 0 {
		return report
	}

	// Build a map of line numbers where evidence quality annotations appear
	evidenceLines := findEvidenceQualityLines(content)

	// For each claim, check if there's an evidence quality annotation
	// between it and the next claim
	for i, claim := range claims {
		nextClaimLine := len(strings.Split(content, "\n")) + 1 // EOF
		if i+1 < len(claims) {
			nextClaimLine = claims[i+1].Line
		}

		annotated := false
		annotationLevel := ""
		for _, el := range evidenceLines {
			if el.line > claim.Line && el.line < nextClaimLine {
				annotated = true
				annotationLevel = el.text
				break
			}
		}

		if annotated {
			report.AnnotatedClaims++
			// Check if low confidence
			if isLowConfidence(annotationLevel) {
				report.LowConfidenceClaims = append(report.LowConfidenceClaims, LowConfidenceClaim{
					Line:  claim.Line,
					Text:  truncate(claim.Text, 120),
					Level: classifyConfidence(annotationLevel),
				})
			}
		} else {
			report.UnannotatedClaims = append(report.UnannotatedClaims, UnannotatedClaim{
				Line: claim.Line,
				Text: truncate(claim.Text, 120),
			})
		}
	}

	if report.TotalClaims > 0 {
		report.CoveragePercent = float64(report.AnnotatedClaims) / float64(report.TotalClaims) * 100
	}

	// Drift detection: check if claim prose exceeds declared tier
	var driftInputs []DriftInput
	for i, claim := range claims {
		nextClaimLine := len(strings.Split(content, "\n")) + 1
		if i+1 < len(claims) {
			nextClaimLine = claims[i+1].Line
		}

		for _, el := range evidenceLines {
			if el.line > claim.Line && el.line < nextClaimLine {
				tier := ClassifyTier(el.text)
				if tier != TierUnclassified {
					driftInputs = append(driftInputs, DriftInput{
						ClaimText: claim.Text,
						Tier:      tier,
						ClaimLine: claim.Line,
					})
				}
				break
			}
		}
	}
	report.DriftFlags = DetectDrift(driftInputs)

	return report
}

// extractProvenanceClaims combines ExtractClaims with section-level claim
// headings (### Claim N, ### Invariant N). Deduplicates by line number.
func extractProvenanceClaims(content string) []Claim {
	claims := ExtractClaims(content)

	// Also detect ### Claim N and ### Invariant N headings
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	currentSection := ""
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if m := h2Re.FindStringSubmatch(line); m != nil {
			currentSection = m[1]
			continue
		}

		if claimHeadingRe.MatchString(line) {
			claims = append(claims, Claim{
				Type:    ClaimTypeCore,
				Text:    line,
				Section: currentSection,
				Line:    lineNum,
			})
		}
	}

	// Sort by line number and deduplicate
	sort.Slice(claims, func(i, j int) bool {
		return claims[i].Line < claims[j].Line
	})

	// Remove duplicates at the same line
	if len(claims) > 1 {
		deduped := claims[:1]
		for i := 1; i < len(claims); i++ {
			if claims[i].Line != claims[i-1].Line {
				deduped = append(deduped, claims[i])
			}
		}
		claims = deduped
	}

	return claims
}

type evidenceLine struct {
	line int
	text string
}

// findEvidenceQualityLines returns line numbers of **Evidence quality:** annotations.
func findEvidenceQualityLines(content string) []evidenceLine {
	var results []evidenceLine
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if m := evidenceQualityRe.FindStringSubmatch(line); m != nil {
			results = append(results, evidenceLine{line: lineNum, text: m[1]})
		}
	}
	return results
}

// findOrphanContradictions scans probes for contradiction verdicts and checks
// if the model was updated after the probe date.
func findOrphanContradictions(probeDir, modelContent string) []OrphanContradiction {
	if _, err := os.Stat(probeDir); os.IsNotExist(err) {
		return nil
	}

	// Extract model's last updated date
	modelDate := extractLastUpdated(modelContent)

	var orphans []OrphanContradiction

	entries, err := os.ReadDir(probeDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		probeDate := extractProbeDate(entry.Name())
		if probeDate == "" {
			continue
		}

		// If model was updated after probe, no orphan
		if modelDate >= probeDate {
			continue
		}

		// Check if this probe contains a contradiction
		data, err := os.ReadFile(filepath.Join(probeDir, entry.Name()))
		if err != nil {
			continue
		}

		contradictions := extractContradictions(string(data))
		for _, c := range contradictions {
			orphans = append(orphans, OrphanContradiction{
				ProbePath:        filepath.Join("probes", entry.Name()),
				ProbeDate:        probeDate,
				ContradictionText: truncate(c, 120),
			})
		}
	}

	return orphans
}

// extractLastUpdated finds the **Last Updated:** date from model content.
func extractLastUpdated(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		if m := lastUpdatedRe.FindStringSubmatch(strings.TrimSpace(scanner.Text())); m != nil {
			return m[1]
		}
	}
	return ""
}

// extractProbeDate extracts the date from a probe filename (YYYY-MM-DD-...).
func extractProbeDate(filename string) string {
	if m := probeDateRe.FindStringSubmatch(filename); m != nil {
		return m[1]
	}
	return ""
}

// extractContradictions finds all "[x] **Contradicts**" lines in probe content.
func extractContradictions(content string) []string {
	var results []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if m := probeContradictRe.FindStringSubmatch(line); m != nil {
			text := strings.TrimSpace(m[1])
			if text == "" {
				text = "unspecified contradiction"
			}
			results = append(results, text)
		}
	}
	return results
}

// isLowConfidence checks if an evidence quality annotation indicates
// low confidence (single-source or assumed).
func isLowConfidence(annotation string) bool {
	lower := strings.ToLower(annotation)
	for _, level := range lowConfidenceLevels {
		if strings.Contains(lower, level) {
			return true
		}
	}
	return false
}

// classifyConfidence returns the low-confidence level from an annotation.
func classifyConfidence(annotation string) string {
	lower := strings.ToLower(annotation)
	if strings.Contains(lower, "assumed") {
		return "assumed"
	}
	if strings.Contains(lower, "single-source") {
		return "single-source"
	}
	return "low"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// FormatProvenanceText produces a human-readable provenance audit report.
func FormatProvenanceText(reports []ProvenanceReport) string {
	var b strings.Builder

	totalModels := len(reports)
	totalClaims := 0
	totalAnnotated := 0
	totalOrphans := 0
	totalDrifts := 0
	modelsWithGaps := 0

	for _, r := range reports {
		totalClaims += r.TotalClaims
		totalAnnotated += r.AnnotatedClaims
		totalOrphans += len(r.OrphanContradictions)
		totalDrifts += len(r.DriftFlags)
		if r.CoveragePercent < 100 && r.TotalClaims > 0 {
			modelsWithGaps++
		}
	}

	overallCoverage := 0.0
	if totalClaims > 0 {
		overallCoverage = float64(totalAnnotated) / float64(totalClaims) * 100
	}

	b.WriteString(fmt.Sprintf("Provenance Audit — %d models, %d claims\n", totalModels, totalClaims))
	b.WriteString(strings.Repeat("=", 55) + "\n\n")
	b.WriteString(fmt.Sprintf("Overall coverage:      %.1f%% (%d/%d annotated)\n", overallCoverage, totalAnnotated, totalClaims))
	b.WriteString(fmt.Sprintf("Models with gaps:      %d\n", modelsWithGaps))
	b.WriteString(fmt.Sprintf("Orphan contradictions: %d\n", totalOrphans))
	b.WriteString(fmt.Sprintf("Drift flags:           %d\n\n", totalDrifts))

	for _, r := range reports {
		if r.TotalClaims == 0 {
			continue
		}

		b.WriteString(fmt.Sprintf("Model: %s (%d claims)\n", r.Name, r.TotalClaims))
		b.WriteString(fmt.Sprintf("  Coverage: %.1f%% (%d/%d annotated)\n", r.CoveragePercent, r.AnnotatedClaims, r.TotalClaims))

		if len(r.LowConfidenceClaims) > 0 {
			b.WriteString(fmt.Sprintf("  Risk: %d low-confidence claims\n", len(r.LowConfidenceClaims)))
			for _, lc := range r.LowConfidenceClaims {
				text := lc.Text
				if len(text) > 80 {
					text = text[:80] + "..."
				}
				b.WriteString(fmt.Sprintf("    L%-4d [%s] %s\n", lc.Line, lc.Level, text))
			}
		}

		if len(r.OrphanContradictions) > 0 {
			b.WriteString(fmt.Sprintf("  Orphan contradictions: %d\n", len(r.OrphanContradictions)))
			for _, oc := range r.OrphanContradictions {
				text := oc.ContradictionText
				if len(text) > 80 {
					text = text[:80] + "..."
				}
				b.WriteString(fmt.Sprintf("    %s: %s\n", oc.ProbeDate, text))
			}
		}

		if len(r.UnannotatedClaims) > 0 {
			b.WriteString(fmt.Sprintf("  Unannotated: %d claims\n", len(r.UnannotatedClaims)))
			for _, uc := range r.UnannotatedClaims {
				text := uc.Text
				if len(text) > 80 {
					text = text[:80] + "..."
				}
				b.WriteString(fmt.Sprintf("    L%-4d %s\n", uc.Line, text))
			}
		}

		if len(r.DriftFlags) > 0 {
			b.WriteString(fmt.Sprintf("  Drift: %d claims exceed declared tier\n", len(r.DriftFlags)))
			for _, df := range r.DriftFlags {
				text := df.Claim
				if len(text) > 60 {
					text = text[:60] + "..."
				}
				b.WriteString(fmt.Sprintf("    L%-4d [%s] %s (triggers: %s)\n",
					df.Line, df.Tier, text, strings.Join(df.Triggers, ", ")))
			}
		}

		b.WriteString("\n")
	}

	return b.String()
}
