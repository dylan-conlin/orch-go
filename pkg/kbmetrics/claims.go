// Package kbmetrics provides metrics for knowledge base health analysis.
// Currently implements claims-per-model extraction — the knowledge equivalent
// of lines-per-file for code bloat detection.
//
// Thresholds (from .kb/models/knowledge-physics/model.md):
//   - healthy: < 30 claims
//   - warning: 30-49 claims (model may need splitting)
//   - critical: >= 50 claims (model needs splitting)
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

const (
	BloatWarning  = 30
	BloatCritical = 50
)

// ClaimType categorizes extracted claims.
type ClaimType string

const (
	ClaimTypeCore       ClaimType = "core"       // Core claim section
	ClaimTypeInvariant  ClaimType = "invariant"   // Numbered items in Critical Invariants
	ClaimTypeAssertion  ClaimType = "assertion"   // Bold-prefixed bullet points
	ClaimTypeData       ClaimType = "data"        // Table data rows
	ClaimTypeConstraint ClaimType = "constraint"  // Constraint/Implication pairs
	ClaimTypeFailure    ClaimType = "failure"     // Failure mode root causes
)

// Claim is a single extractable assertion from a model.md file.
type Claim struct {
	Type    ClaimType
	Text    string
	Section string // parent section heading
	Line    int
}

// ModelReport summarizes claims for one model.
type ModelReport struct {
	Name       string
	Path       string
	ClaimCount int
	Claims     []Claim
	BloatLevel string // "healthy", "warning", "critical"
	ByType     map[ClaimType]int
}

var (
	// Numbered list items: "1. **Bold text** — rest" or "1. **Bold text.** rest"
	numberedItemRe = regexp.MustCompile(`^\d+\.\s+\*\*`)

	// Bold-prefixed bullets: "- **Something** — description"
	boldBulletRe = regexp.MustCompile(`^[-*]\s+\*\*[^*]+\*\*\s*[—–\-]`)

	// Table data rows (not header/separator): "| content | content |"
	tableRowRe = regexp.MustCompile(`^\|.+\|$`)
	tableSepRe = regexp.MustCompile(`^\|[-|\s]+\|$`)

	// Constraint/Implication prefixed lines
	constraintRe  = regexp.MustCompile(`^\*\*Constraint:\*\*`)
	implicationRe = regexp.MustCompile(`^\*\*Implication:\*\*`)

	// Failure mode markers
	rootCauseRe = regexp.MustCompile(`^\*\*Root cause.*:\*\*`)

	// Section headings
	h2Re = regexp.MustCompile(`^##\s+(.+)`)
	h3Re = regexp.MustCompile(`^###\s+(.+)`)

	// Sections to skip (non-claim content)
	skipSections = map[string]bool{
		"references":      true,
		"evolution":       true,
		"merged probes":   true,
		"observability":   true,
		"related models":  true,
		"related guides":  true,
		"open questions":  true,
		"primary evidence": true,
	}

	// Date-prefixed lines in Evolution sections: "**2026-01-10:** Something"
	dateLineRe = regexp.MustCompile(`^\*\*\d{4}-\d{2}-\d{2}`)
)

// ExtractClaims parses a model.md file content and returns extracted claims.
func ExtractClaims(content string) []Claim {
	var claims []Claim
	scanner := bufio.NewScanner(strings.NewReader(content))

	currentSection := ""
	inSkipSection := false
	inCoreClaim := false
	lineNum := 0
	seenTableHeader := false

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Track section headings
		if m := h2Re.FindStringSubmatch(line); m != nil {
			currentSection = m[1]
			sectionLower := strings.ToLower(currentSection)
			inSkipSection = skipSections[sectionLower]
			inCoreClaim = sectionLower == "core claim"
			seenTableHeader = false
			continue
		}
		if m := h3Re.FindStringSubmatch(line); m != nil {
			seenTableHeader = false
			// Don't reset inSkipSection for h3 within a skip section
			continue
		}

		if inSkipSection {
			continue
		}

		if line == "" {
			continue
		}

		// Skip date-prefixed lines (Evolution entries that leaked through)
		if dateLineRe.MatchString(line) {
			continue
		}

		// Core Claim section: first non-empty, non-heading line is the core claim
		if inCoreClaim && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") {
			claims = append(claims, Claim{
				Type:    ClaimTypeCore,
				Text:    line,
				Section: currentSection,
				Line:    lineNum,
			})
			inCoreClaim = false
			continue
		}

		// Numbered items (typically in Critical Invariants)
		if numberedItemRe.MatchString(line) {
			claims = append(claims, Claim{
				Type:    ClaimTypeInvariant,
				Text:    line,
				Section: currentSection,
				Line:    lineNum,
			})
			continue
		}

		// Bold-prefixed bullets
		if boldBulletRe.MatchString(line) {
			claims = append(claims, Claim{
				Type:    ClaimTypeAssertion,
				Text:    line,
				Section: currentSection,
				Line:    lineNum,
			})
			continue
		}

		// Table rows (skip header row and separator)
		if tableRowRe.MatchString(line) {
			if tableSepRe.MatchString(line) {
				seenTableHeader = true
				continue
			}
			if !seenTableHeader {
				// This is the header row
				continue
			}
			// Data row after separator
			claims = append(claims, Claim{
				Type:    ClaimTypeData,
				Text:    line,
				Section: currentSection,
				Line:    lineNum,
			})
			continue
		} else {
			seenTableHeader = false
		}

		// Constraint/Implication pairs
		if constraintRe.MatchString(line) {
			claims = append(claims, Claim{
				Type:    ClaimTypeConstraint,
				Text:    line,
				Section: currentSection,
				Line:    lineNum,
			})
			continue
		}

		// Root cause in failure modes
		if rootCauseRe.MatchString(line) {
			claims = append(claims, Claim{
				Type:    ClaimTypeFailure,
				Text:    line,
				Section: currentSection,
				Line:    lineNum,
			})
			continue
		}
	}

	return claims
}

// AnalyzeModels scans all model.md files under the given directory
// and returns a report per model, sorted by claim count descending.
func AnalyzeModels(modelsDir string) ([]ModelReport, error) {
	var results []ModelReport

	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, fmt.Errorf("read models dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modelPath := filepath.Join(modelsDir, entry.Name(), "model.md")
		data, err := os.ReadFile(modelPath)
		if err != nil {
			continue // skip models without model.md
		}

		claims := ExtractClaims(string(data))

		byType := make(map[ClaimType]int)
		for _, c := range claims {
			byType[c.Type]++
		}

		results = append(results, ModelReport{
			Name:       entry.Name(),
			Path:       modelPath,
			ClaimCount: len(claims),
			Claims:     claims,
			BloatLevel: BloatLevel(len(claims)),
			ByType:     byType,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ClaimCount > results[j].ClaimCount
	})

	return results, nil
}

// BloatLevel returns the bloat level for a given claim count.
func BloatLevel(count int) string {
	switch {
	case count >= BloatCritical:
		return "critical"
	case count >= BloatWarning:
		return "warning"
	default:
		return "healthy"
	}
}

// FormatText produces a human-readable summary of model claim analysis.
func FormatText(results []ModelReport) string {
	var b strings.Builder

	totalClaims := 0
	bloated := 0
	critical := 0

	for _, r := range results {
		totalClaims += r.ClaimCount
		if r.BloatLevel == "warning" {
			bloated++
		}
		if r.BloatLevel == "critical" {
			critical++
		}
	}

	b.WriteString(fmt.Sprintf("Knowledge Claims Report — %d models, %d total claims\n", len(results), totalClaims))
	b.WriteString(strings.Repeat("=", 60) + "\n\n")

	if critical > 0 {
		b.WriteString(fmt.Sprintf("!! %d CRITICAL models (>=%d claims, need splitting) !!\n", critical, BloatCritical))
	}
	if bloated > 0 {
		b.WriteString(fmt.Sprintf("~  %d WARNING models (>=%d claims, may need splitting)\n", bloated, BloatWarning))
	}
	if critical == 0 && bloated == 0 {
		b.WriteString("All models within healthy claim thresholds.\n")
	}
	b.WriteString("\n")

	for _, r := range results {
		prefix := "  "
		switch r.BloatLevel {
		case "critical":
			prefix = "!! "
		case "warning":
			prefix = "~  "
		}
		b.WriteString(fmt.Sprintf("%s%-45s %3d claims", prefix, r.Name, r.ClaimCount))

		// Show type breakdown for non-trivial models
		if r.ClaimCount > 0 {
			var parts []string
			typeOrder := []ClaimType{ClaimTypeCore, ClaimTypeInvariant, ClaimTypeAssertion, ClaimTypeData, ClaimTypeConstraint, ClaimTypeFailure}
			for _, t := range typeOrder {
				if n, ok := r.ByType[t]; ok && n > 0 {
					parts = append(parts, fmt.Sprintf("%s:%d", t, n))
				}
			}
			if len(parts) > 0 {
				b.WriteString(" (" + strings.Join(parts, ", ") + ")")
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}
