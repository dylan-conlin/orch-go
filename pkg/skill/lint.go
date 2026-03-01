// Package skill provides static analysis for skill markdown documents.
package skill

import (
	"fmt"
	"strings"
)

// Rule names for lint checks.
const (
	RuleMustDensity        = "must-density"
	RuleCosmeticRedundancy = "cosmetic-redundancy"
	RuleSectionSprawl      = "section-sprawl"
	RuleSignalImbalance    = "signal-imbalance"
	RuleDeadConstraint     = "dead-constraint"
)

// Severity levels for lint results.
const (
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// LintResult represents a single lint finding.
type LintResult struct {
	Rule     string
	Severity string
	Message  string
}

// Thresholds for lint rules (from architect design orch-go-dlw9).
const (
	mustDensityThreshold     = 3.0  // per 100 words
	cosmeticRedundancyMax    = 2    // same phrase > this count triggers warning
	sectionSprawlMax         = 30   // total constraints across all sections
	signalImbalanceThreshold = 3    // same behavior reinforced > this triggers warning
)

// mustKeywords are the directive keywords that contribute to MUST-density.
var mustKeywords = []string{"MUST", "NEVER", "CRITICAL", "ALWAYS"}

// LintContent runs all 5 lint rules against skill markdown content.
// testCoverage is an optional list of test descriptions; if nil, dead constraint check
// reports info-level findings.
func LintContent(content string, testCoverage []string) []LintResult {
	var results []LintResult

	results = append(results, checkMustDensity(content)...)
	results = append(results, checkCosmeticRedundancy(content)...)
	results = append(results, checkSectionSprawl(content)...)
	results = append(results, checkSignalImbalance(content)...)
	results = append(results, checkDeadConstraint(content, testCoverage)...)

	return results
}

// checkMustDensity counts MUST/NEVER/CRITICAL/ALWAYS per 100 words.
// Threshold: >3 per 100 words = warning.
// Source: DSL design principles investigation.
func checkMustDensity(content string) []LintResult {
	words := strings.Fields(content)
	if len(words) == 0 {
		return nil
	}

	count := 0
	for _, w := range words {
		cleaned := strings.Trim(w, ".,;:!?\"'`()[]{}*_#-")
		for _, kw := range mustKeywords {
			if cleaned == kw {
				count++
				break
			}
		}
	}

	density := float64(count) / float64(len(words)) * 100
	if density > mustDensityThreshold {
		return []LintResult{{
			Rule:     RuleMustDensity,
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("MUST-density %.1f/100 words (threshold: %.1f) — %d directive keywords in %d words", density, mustDensityThreshold, count, len(words)),
		}}
	}
	return nil
}

// checkCosmeticRedundancy finds constraint phrases repeated >2 times.
// A constraint phrase is extracted by normalizing lines containing directive keywords.
// Source: Defense-in-depth investigation.
func checkCosmeticRedundancy(content string) []LintResult {
	lines := strings.Split(content, "\n")
	phrases := make(map[string]int)

	for _, line := range lines {
		normalized := extractConstraintPhrase(line)
		if normalized != "" {
			phrases[normalized]++
		}
	}

	var results []LintResult
	for phrase, count := range phrases {
		if count > cosmeticRedundancyMax {
			results = append(results, LintResult{
				Rule:     RuleCosmeticRedundancy,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("phrase repeated %d times (max %d): %q", count, cosmeticRedundancyMax, truncate(phrase, 60)),
			})
		}
	}
	return results
}

// extractConstraintPhrase normalizes a line to its core constraint phrase.
// Strips directive keywords, list markers, and common prefixes to find the
// underlying action being constrained.
func extractConstraintPhrase(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}

	// Remove list markers
	trimmed = strings.TrimPrefix(trimmed, "- ")
	trimmed = strings.TrimPrefix(trimmed, "* ")

	// Must contain a directive keyword to be a constraint
	hasDirective := false
	for _, kw := range mustKeywords {
		if strings.Contains(strings.ToUpper(trimmed), kw) {
			hasDirective = true
			break
		}
	}
	if !hasDirective {
		return ""
	}

	// Strip directive keywords and common prefixes to get the action
	action := strings.ToLower(trimmed)
	for _, prefix := range []string{
		"you must ", "must ", "never ", "always ", "critical: ", "critical ",
		"you must never ", "you must always ",
	} {
		action = strings.TrimPrefix(action, prefix)
	}

	// Strip trailing punctuation
	action = strings.TrimRight(action, ".,;:!?")
	action = strings.TrimSpace(action)

	if len(action) < 5 {
		return ""
	}
	return action
}

// checkSectionSprawl counts total constraint lines across the document.
// Threshold: >30 = warning (general-purpose drift signal).
// Source: DSL investigation.
func checkSectionSprawl(content string) []LintResult {
	lines := strings.Split(content, "\n")
	count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isConstraintLine(trimmed) {
			count++
		}
	}

	if count > sectionSprawlMax {
		return []LintResult{{
			Rule:     RuleSectionSprawl,
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("%d constraints found (threshold: %d) — possible general-purpose drift", count, sectionSprawlMax),
		}}
	}
	return nil
}

// isConstraintLine returns true if a line expresses a behavioral constraint.
func isConstraintLine(line string) bool {
	upper := strings.ToUpper(line)
	for _, kw := range mustKeywords {
		if strings.Contains(upper, kw) {
			return true
		}
	}
	return false
}

// checkSignalImbalance finds the same behavior being reinforced >3 times.
// Groups constraints by significant bigrams (two-word pairs) to detect when
// the same concept is reinforced from multiple directive angles.
// Source: Framework investigation (17:1 ratio finding).
func checkSignalImbalance(content string) []LintResult {
	lines := strings.Split(content, "\n")

	// Extract all constraint phrases
	var phrases []string
	for _, line := range lines {
		phrase := extractConstraintPhrase(line)
		if phrase != "" {
			phrases = append(phrases, phrase)
		}
	}

	// Count significant terms (single words and bigrams) across constraint phrases.
	// If the same term appears in >threshold distinct constraint lines, it's signal imbalance.
	termLines := make(map[string]int)
	for _, phrase := range phrases {
		terms := extractSignificantTerms(phrase)
		seen := make(map[string]bool)
		for _, term := range terms {
			if !seen[term] {
				seen[term] = true
				termLines[term]++
			}
		}
	}

	var results []LintResult
	reported := make(map[string]bool)
	for term, count := range termLines {
		if count > signalImbalanceThreshold && !reported[term] {
			reported[term] = true
			results = append(results, LintResult{
				Rule:     RuleSignalImbalance,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("behavior %q reinforced %d times (threshold: %d) — competing signals on same behavior", term, count, signalImbalanceThreshold),
			})
		}
	}
	return results
}

// extractSignificantTerms returns significant single words and bigrams from a phrase.
// Used to detect when the same behavior is targeted by multiple constraints.
func extractSignificantTerms(phrase string) []string {
	words := strings.Fields(strings.ToLower(phrase))
	stopWords := map[string]bool{
		"the": true, "for": true, "and": true, "with": true,
		"that": true, "this": true, "from": true, "into": true,
		"all": true, "not": true, "are": true, "was": true,
		"been": true, "has": true, "have": true, "does": true,
		"use": true, "skip": true, "called": true, "every": true,
		"must": true, "should": true, "can": true, "will": true,
	}
	var significant []string
	for _, w := range words {
		if len(w) > 3 && !stopWords[w] {
			significant = append(significant, w)
		}
	}

	var terms []string
	// Add single significant words (nouns/verbs that identify the behavior)
	for _, w := range significant {
		terms = append(terms, w)
	}
	// Add bigrams for more precise matching
	for i := 0; i+1 < len(significant); i++ {
		terms = append(terms, significant[i]+" "+significant[i+1])
	}
	return terms
}

// checkDeadConstraint checks if constraints have corresponding test coverage.
// If no testCoverage is provided, reports all constraints as info-level findings.
// Source: Defense-in-depth investigation.
func checkDeadConstraint(content string, testCoverage []string) []LintResult {
	lines := strings.Split(content, "\n")
	var constraints []string

	for _, line := range lines {
		phrase := extractConstraintPhrase(line)
		if phrase != "" {
			constraints = append(constraints, phrase)
		}
	}

	if len(constraints) == 0 {
		return nil
	}

	// If no test coverage provided, report as info
	if testCoverage == nil {
		return []LintResult{{
			Rule:     RuleDeadConstraint,
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("%d constraints with no behavioral test coverage (provide --tests to check)", len(constraints)),
		}}
	}

	// Check each constraint against test coverage
	uncovered := 0
	for _, constraint := range constraints {
		covered := false
		for _, test := range testCoverage {
			if phraseOverlap(constraint, test) {
				covered = true
				break
			}
		}
		if !covered {
			uncovered++
		}
	}

	if uncovered > 0 {
		return []LintResult{{
			Rule:     RuleDeadConstraint,
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("%d of %d constraints have no behavioral test coverage", uncovered, len(constraints)),
		}}
	}
	return nil
}

// phraseOverlap returns true if the constraint phrase and test description
// share significant words, suggesting the test covers the constraint.
func phraseOverlap(constraint, test string) bool {
	constraintWords := strings.Fields(strings.ToLower(constraint))
	testLower := strings.ToLower(test)

	matches := 0
	for _, w := range constraintWords {
		if len(w) > 3 && strings.Contains(testLower, w) {
			matches++
		}
	}
	// At least 2 significant word overlap
	return matches >= 2
}

// truncate shortens a string to maxLen, adding ellipsis if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
