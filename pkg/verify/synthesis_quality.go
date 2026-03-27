// Package verify provides verification helpers for agent completion.
// synthesis_quality.go computes quality signals from parsed Synthesis structs.
// Signals are advisory metadata for comprehension queue ordering.
package verify

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/debrief"
)

// QualitySignal represents a single detectable quality indicator in a synthesis.
type QualitySignal struct {
	Name     string // e.g., "structural_completeness"
	Detected bool   // Whether the signal was found
	Score    string // e.g., "4/4" or "3/5 lines"
	Evidence string // Excerpt that triggered detection
}

// SynthesisQuality holds the full set of quality signals for a synthesis.
type SynthesisQuality struct {
	Signals     []QualitySignal
	SignalCount int // How many of 6 fired (natural sort key)
	Total       int // Always 6 (denominator)
}

// Evidence specificity patterns: file paths, test output, concrete references.
var evidenceSpecificityPattern = regexp.MustCompile(
	`(?i)(pkg/|cmd/|\.go\b|\.ts\b|\.md\b|\.svelte\b|PASS|FAIL|\btest\b|\bassert)`,
)

// Model connection patterns: kb model references or confirms/contradicts/extends language.
var modelConnectionPattern = regexp.MustCompile(
	`(?i)(\.kb/models/|confirms?\b|contradicts?\b|extends?\b)`,
)

// ComputeSynthesisQuality evaluates 6 mechanically-detectable quality signals
// from a parsed Synthesis. Returns signal list with counts for ordering.
func ComputeSynthesisQuality(s *Synthesis) SynthesisQuality {
	signals := []QualitySignal{
		checkStructuralCompleteness(s),
		checkEvidenceSpecificity(s),
		checkModelConnection(s),
		checkConnectiveReasoning(s),
		checkTensionQuality(s),
		checkInsightVsReport(s),
	}

	count := 0
	for _, sig := range signals {
		if sig.Detected {
			count++
		}
	}

	return SynthesisQuality{
		Signals:     signals,
		SignalCount: count,
		Total:       6,
	}
}

// checkStructuralCompleteness counts populated D.E.K.N. sections (TLDR, Delta, Evidence, Knowledge).
// Detected when 3+ of 4 sections are non-empty.
func checkStructuralCompleteness(s *Synthesis) QualitySignal {
	count := 0
	var populated []string
	if s.TLDR != "" {
		count++
		populated = append(populated, "TLDR")
	}
	if s.Delta != "" {
		count++
		populated = append(populated, "Delta")
	}
	if s.Evidence != "" {
		count++
		populated = append(populated, "Evidence")
	}
	if s.Knowledge != "" {
		count++
		populated = append(populated, "Knowledge")
	}

	return QualitySignal{
		Name:     "structural_completeness",
		Detected: count >= 3,
		Score:    fmt.Sprintf("%d/4", count),
		Evidence: strings.Join(populated, ", "),
	}
}

// checkEvidenceSpecificity looks for concrete references (file paths, test output)
// in Evidence and Delta sections.
func checkEvidenceSpecificity(s *Synthesis) QualitySignal {
	combined := s.Evidence + "\n" + s.Delta
	match := evidenceSpecificityPattern.FindString(combined)
	return QualitySignal{
		Name:     "evidence_specificity",
		Detected: match != "",
		Score:    boolScore(match != ""),
		Evidence: match,
	}
}

// checkModelConnection looks for .kb/models/ references or confirms/contradicts/extends
// language in Knowledge section.
func checkModelConnection(s *Synthesis) QualitySignal {
	match := modelConnectionPattern.FindString(s.Knowledge)
	return QualitySignal{
		Name:     "model_connection",
		Detected: match != "",
		Score:    boolScore(match != ""),
		Evidence: match,
	}
}

// checkConnectiveReasoning reuses debrief.HasConnectiveLanguage on Knowledge + TLDR.
func checkConnectiveReasoning(s *Synthesis) QualitySignal {
	combined := s.Knowledge + "\n" + s.TLDR
	detected := debrief.HasConnectiveLanguage(combined)
	return QualitySignal{
		Name:     "connective_reasoning",
		Detected: detected,
		Score:    boolScore(detected),
	}
}

// checkTensionQuality requires UnexploredQuestions to be non-empty AND contain question marks.
func checkTensionQuality(s *Synthesis) QualitySignal {
	detected := s.UnexploredQuestions != "" && strings.Contains(s.UnexploredQuestions, "?")
	return QualitySignal{
		Name:     "tension_quality",
		Detected: detected,
		Score:    boolScore(detected),
	}
}

// checkInsightVsReport evaluates Knowledge lines for insight content vs action-verb reporting.
// Detected when majority of non-empty Knowledge lines are NOT action-verb sentences.
func checkInsightVsReport(s *Synthesis) QualitySignal {
	if s.Knowledge == "" {
		return QualitySignal{
			Name:     "insight_vs_report",
			Detected: false,
			Score:    "0/0",
		}
	}

	lines := strings.Split(s.Knowledge, "\n")
	total := 0
	insightCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		total++
		if !debrief.IsActionVerbSentence(line) {
			insightCount++
		}
	}

	if total == 0 {
		return QualitySignal{
			Name:     "insight_vs_report",
			Detected: false,
			Score:    "0/0",
		}
	}

	// Detected when majority (>50%) of lines are insight, not report
	detected := insightCount > total/2
	return QualitySignal{
		Name:     "insight_vs_report",
		Detected: detected,
		Score:    fmt.Sprintf("%d/%d", insightCount, total),
	}
}

// QualityThresholds defines configurable thresholds for quality assessment.
type QualityThresholds struct {
	// MinSignals is the minimum number of signals that must fire to pass.
	MinSignals int
	// RequiredSignals lists signal names that MUST be detected (regardless of count).
	RequiredSignals []string
}

// DefaultQualityThresholds returns sensible defaults: 3/6 signals minimum,
// structural_completeness required.
func DefaultQualityThresholds() QualityThresholds {
	return QualityThresholds{
		MinSignals:      3,
		RequiredSignals: []string{"structural_completeness"},
	}
}

// MeetsThreshold checks if this quality result satisfies the given thresholds.
func (q SynthesisQuality) MeetsThreshold(th QualityThresholds) bool {
	if q.SignalCount < th.MinSignals {
		return false
	}
	for _, req := range th.RequiredSignals {
		found := false
		for _, sig := range q.Signals {
			if sig.Name == req && sig.Detected {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Summary returns a human-readable summary of quality signals.
func (q SynthesisQuality) Summary() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Quality: %d/%d signals\n", q.SignalCount, q.Total))
	for _, sig := range q.Signals {
		marker := "  ✗"
		if sig.Detected {
			marker = "  ✓"
		}
		b.WriteString(fmt.Sprintf("%s %s", marker, sig.Name))
		if sig.Evidence != "" {
			b.WriteString(fmt.Sprintf(" (%s)", sig.Evidence))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func boolScore(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
