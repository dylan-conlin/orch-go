// Package findingdedup detects duplicate findings across investigation and
// synthesis files. When the same insight appears in 3+ investigations with
// different wording, the system is narrating rather than learning —
// regenerating conclusions instead of building on prior work.
package findingdedup

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Finding represents an extracted finding from an investigation or synthesis file.
type Finding struct {
	Title      string // finding title (from "### Finding N: Title")
	Body       string // concatenated evidence/significance text
	SourceFile string // relative path to source file
	FindingNum int    // finding number within the file (1-based)
	tokens     []string
}

// Cluster represents a group of findings that express the same insight.
type Cluster struct {
	Findings   []Finding // all findings in this cluster
	Similarity float64   // average pairwise similarity
	Label      string    // representative label for the cluster
}

// Detector configures finding deduplication.
type Detector struct {
	Threshold      float64 // similarity threshold (default 0.30)
	MinClusterSize int     // minimum cluster size to report (default 3)
}

// NewDetector returns a Detector with sensible defaults.
func NewDetector() *Detector {
	return &Detector{
		Threshold:      0.20,
		MinClusterSize: 3,
	}
}

// ScanDir scans all .md files in a directory, extracts findings, and returns
// clusters of duplicate findings.
func (d *Detector) ScanDir(dir string) ([]Cluster, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var allFindings []Finding
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		findings := ExtractFindings(entry.Name(), string(data))
		allFindings = append(allFindings, findings...)
	}

	return d.FindClusters(allFindings), nil
}

// ScanDirs scans multiple directories and returns clusters across all of them.
func (d *Detector) ScanDirs(dirs ...string) ([]Cluster, error) {
	var allFindings []Finding
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			relPath, _ := filepath.Rel(filepath.Dir(dir), filepath.Join(dir, entry.Name()))
			if relPath == "" {
				relPath = entry.Name()
			}
			findings := ExtractFindings(relPath, string(data))
			allFindings = append(allFindings, findings...)
		}
	}
	return d.FindClusters(allFindings), nil
}

// FindClusters groups findings by similarity and returns clusters at or above
// MinClusterSize. Uses complete-linkage clustering: a finding joins a cluster
// only if it is similar to ALL existing members, preventing chain drift.
func (d *Detector) FindClusters(findings []Finding) []Cluster {
	if len(findings) < d.MinClusterSize {
		return nil
	}

	// Ensure all findings are tokenized (title + body for richer comparison)
	for i := range findings {
		if findings[i].tokens == nil {
			findings[i].tokens = tokenize(findings[i].Title + " " + findings[i].Body)
		}
	}

	// Precompute pairwise similarity matrix
	n := len(findings)
	simMatrix := make([][]float64, n)
	for i := range simMatrix {
		simMatrix[i] = make([]float64, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			sim := jaccardSimilarity(findings[i].tokens, findings[j].tokens)
			simMatrix[i][j] = sim
			simMatrix[j][i] = sim
		}
	}

	// Greedy complete-linkage clustering:
	// For each finding, try to add it to an existing cluster where it meets
	// threshold with ALL members. If no cluster fits, start a new one.
	type cluster struct {
		members []int
	}
	var rawClusters []cluster

	assigned := make([]bool, n)
	// Sort findings by number of high-similarity neighbors (descending)
	// to seed clusters with the most-connected findings first
	type scored struct {
		idx   int
		count int
	}
	var scored_list []scored
	for i := 0; i < n; i++ {
		count := 0
		for j := 0; j < n; j++ {
			if i != j && simMatrix[i][j] >= d.Threshold {
				count++
			}
		}
		scored_list = append(scored_list, scored{i, count})
	}
	sort.Slice(scored_list, func(a, b int) bool {
		return scored_list[a].count > scored_list[b].count
	})

	for _, s := range scored_list {
		i := s.idx
		if assigned[i] {
			continue
		}

		// Try to join existing cluster
		bestCluster := -1
		bestAvgSim := 0.0
		for ci, c := range rawClusters {
			allAbove := true
			totalSim := 0.0
			for _, m := range c.members {
				if simMatrix[i][m] < d.Threshold {
					allAbove = false
					break
				}
				totalSim += simMatrix[i][m]
			}
			if allAbove {
				avgSim := totalSim / float64(len(c.members))
				if avgSim > bestAvgSim {
					bestAvgSim = avgSim
					bestCluster = ci
				}
			}
		}

		if bestCluster >= 0 {
			rawClusters[bestCluster].members = append(rawClusters[bestCluster].members, i)
			assigned[i] = true
		} else {
			rawClusters = append(rawClusters, cluster{members: []int{i}})
			assigned[i] = true
		}
	}

	// Collect clusters meeting min size
	clusterMap := map[int][]int{}
	for ci, c := range rawClusters {
		if len(c.members) >= d.MinClusterSize {
			clusterMap[ci] = c.members
		}
	}

	var clusters []Cluster
	for _, indices := range clusterMap {
		if len(indices) < d.MinClusterSize {
			continue
		}

		var clusterFindings []Finding
		for _, idx := range indices {
			clusterFindings = append(clusterFindings, findings[idx])
		}

		// Compute average pairwise similarity
		var totalSim float64
		var pairCount int
		for i := 0; i < len(indices); i++ {
			for j := i + 1; j < len(indices); j++ {
				totalSim += jaccardSimilarity(findings[indices[i]].tokens, findings[indices[j]].tokens)
				pairCount++
			}
		}
		avgSim := 0.0
		if pairCount > 0 {
			avgSim = totalSim / float64(pairCount)
		}

		// Use shortest title as label (most concise expression)
		label := clusterFindings[0].Title
		for _, f := range clusterFindings[1:] {
			if f.Title != "" && (label == "" || len(f.Title) < len(label)) {
				label = f.Title
			}
		}
		if label == "" {
			// Fallback: use most common tokens
			label = topTokensLabel(clusterFindings)
		}

		clusters = append(clusters, Cluster{
			Findings:   clusterFindings,
			Similarity: avgSim,
			Label:      label,
		})
	}

	// Sort by cluster size descending
	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i].Findings) > len(clusters[j].Findings)
	})

	return clusters
}

// regexes for extracting findings from investigation files
var (
	regexFinding      = regexp.MustCompile(`(?m)^###\s+Finding\s+(\d+):\s*(.+)$`)
	regexEvidenceLine = regexp.MustCompile(`(?m)^\*\*Evidence:\*\*\s*(.+)$`)
	regexSignifLine   = regexp.MustCompile(`(?m)^\*\*Significance:\*\*\s*(.+)$`)
	regexSourceLine   = regexp.MustCompile(`(?m)^\*\*Source:\*\*\s*(.+)$`)
	// Section header pattern for splitting
	regexSectionH2 = regexp.MustCompile(`(?m)^## .+$`)
	regexSectionH3 = regexp.MustCompile(`(?m)^### .+$`)
)

// ExtractFindings extracts findings from a markdown file.
// Supports two formats:
//  1. Investigation format: "### Finding N: Title" with Evidence/Source/Significance
//  2. Synthesis format: bullet points in Evidence/Knowledge sections
func ExtractFindings(filename, content string) []Finding {
	var findings []Finding

	// Try investigation format first
	findings = append(findings, extractInvestigationFindings(filename, content)...)

	// Also try synthesis format (Evidence/Knowledge sections with bullet points)
	findings = append(findings, extractSynthesisFindings(filename, content)...)

	return findings
}

// extractInvestigationFindings extracts "### Finding N: Title" blocks.
func extractInvestigationFindings(filename, content string) []Finding {
	matches := regexFinding.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil
	}

	var findings []Finding
	for i, match := range matches {
		num := 0
		fmt.Sscanf(content[match[2]:match[3]], "%d", &num)
		title := strings.TrimSpace(content[match[4]:match[5]])

		// Extract body: from end of title line to next finding or next ## section
		bodyStart := match[1]
		var bodyEnd int
		if i+1 < len(matches) {
			bodyEnd = matches[i+1][0]
		} else {
			// Find next ## section or end of file
			rest := content[bodyStart:]
			nextSection := regexSectionH2.FindStringIndex(rest)
			if nextSection != nil {
				bodyEnd = bodyStart + nextSection[0]
			} else {
				bodyEnd = len(content)
			}
		}

		body := strings.TrimSpace(content[bodyStart:bodyEnd])

		// Extract evidence and significance for richer comparison
		var parts []string
		if ev := regexEvidenceLine.FindStringSubmatch(body); len(ev) >= 2 {
			parts = append(parts, ev[1])
		}
		if sig := regexSignifLine.FindStringSubmatch(body); len(sig) >= 2 {
			parts = append(parts, sig[1])
		}
		combinedBody := body
		if len(parts) > 0 {
			combinedBody = strings.Join(parts, " ")
		}

		// Skip template placeholders (unfilled investigation templates)
		if isTemplatePlaceholder(title) && isTemplatePlaceholder(combinedBody) {
			continue
		}

		findings = append(findings, Finding{
			Title:      title,
			Body:       combinedBody,
			SourceFile: filename,
			FindingNum: num,
		})
	}
	return findings
}

// extractSynthesisFindings extracts bullet-point findings from Evidence/Knowledge sections.
func extractSynthesisFindings(filename, content string) []Finding {
	var findings []Finding
	findingNum := 0

	// Extract Evidence section
	evidenceSection := extractMarkdownSection(content, "Evidence")
	if evidenceSection == "" {
		evidenceSection = extractMarkdownSection(content, "Evidence (What Was Observed)")
	}
	for _, bullet := range extractBullets(evidenceSection) {
		findingNum++
		findings = append(findings, Finding{
			Title:      truncateTitle(bullet, 60),
			Body:       bullet,
			SourceFile: filename,
			FindingNum: findingNum,
		})
	}

	// Extract Knowledge section
	knowledgeSection := extractMarkdownSection(content, "Knowledge")
	if knowledgeSection == "" {
		knowledgeSection = extractMarkdownSection(content, "Knowledge (What Was Learned)")
	}
	for _, bullet := range extractBullets(knowledgeSection) {
		findingNum++
		findings = append(findings, Finding{
			Title:      truncateTitle(bullet, 60),
			Body:       bullet,
			SourceFile: filename,
			FindingNum: findingNum,
		})
	}

	return findings
}

// extractMarkdownSection extracts content under a ## heading.
func extractMarkdownSection(content, sectionName string) string {
	pattern := regexp.MustCompile(`(?s)## ` + regexp.QuoteMeta(sectionName) + `\s*\n(.*?)(?:\n## |\z)`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractBullets extracts bullet-point lines from a section.
// Filters out metadata lines that are structural formatting, not findings.
func extractBullets(section string) []string {
	if section == "" {
		return nil
	}
	var bullets []string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		var text string
		if strings.HasPrefix(line, "- ") {
			text = strings.TrimPrefix(line, "- ")
		} else if strings.HasPrefix(line, "* ") {
			text = strings.TrimPrefix(line, "* ")
		} else {
			continue
		}

		// Skip metadata lines
		lower := strings.ToLower(text)
		isMetadata := false
		for _, prefix := range metadataLinePrefixes {
			if strings.HasPrefix(lower, prefix) {
				isMetadata = true
				break
			}
		}
		if isMetadata {
			continue
		}

		bullets = append(bullets, text)
	}
	return bullets
}

// truncateTitle creates a short title from a bullet point.
func truncateTitle(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// templatePatterns are placeholder strings from investigation templates that
// should be filtered out — they represent unfilled templates, not real findings.
var templatePatterns = []string{
	"[brief, descriptive title]",
	"[what was observed]",
	"[why this matters]",
	"[where evidence comes from]",
	"[describe what you found]",
	"[describe significance]",
	"[todo]",
	"[placeholder]",
}

// metadataLinePrefixes are structured metadata prefixes in SYNTHESIS.md bullets
// that should not be treated as findings (they're formatting, not insights).
var metadataLinePrefixes = []string{
	"**layer:**",
	"**source:**",
	"**status:**",
	"**type:**",
	"**priority:**",
	"**category:**",
	"`.kb/",
}

// isTemplatePlaceholder returns true if text appears to be an unfilled template.
func isTemplatePlaceholder(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return true
	}
	for _, p := range templatePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	// Text that is entirely bracketed placeholders
	if strings.HasPrefix(lower, "[") && strings.HasSuffix(lower, "]") {
		return true
	}
	return false
}

// stopwords is a set of common English words to exclude from similarity comparison.
var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true,
	"did": true, "will": true, "would": true, "could": true, "should": true,
	"may": true, "might": true, "shall": true, "can": true,
	"not": true, "no": true, "nor": true, "but": true, "or": true,
	"and": true, "if": true, "then": true, "else": true, "when": true,
	"at": true, "by": true, "for": true, "with": true, "about": true,
	"against": true, "between": true, "through": true, "during": true,
	"before": true, "after": true, "above": true, "below": true,
	"to": true, "from": true, "up": true, "down": true, "in": true,
	"out": true, "on": true, "off": true, "over": true, "under": true,
	"of": true, "into": true, "as": true, "so": true, "than": true,
	"that": true, "this": true, "these": true, "those": true,
	"it": true, "its": true, "they": true, "them": true, "their": true,
	"we": true, "our": true, "you": true, "your": true, "he": true,
	"she": true, "his": true, "her": true, "which": true, "what": true,
	"who": true, "whom": true, "where": true, "how": true, "all": true,
	"each": true, "every": true, "both": true, "few": true, "more": true,
	"most": true, "other": true, "some": true, "such": true, "only": true,
	"own": true, "same": true, "just": true, "also": true, "very": true,
}

// tokenize splits text into normalized, lowercased tokens with stopwords removed.
func tokenize(text string) []string {
	lower := strings.ToLower(text)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var tokens []string
	for _, w := range words {
		if len(w) < 2 {
			continue
		}
		if stopwords[w] {
			continue
		}
		tokens = append(tokens, w)
	}
	return tokens
}

// jaccardSimilarity computes the Jaccard index of two token sets.
func jaccardSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	setA := make(map[string]bool, len(a))
	for _, t := range a {
		setA[t] = true
	}
	setB := make(map[string]bool, len(b))
	for _, t := range b {
		setB[t] = true
	}

	intersection := 0
	for t := range setA {
		if setB[t] {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

// topTokensLabel generates a label from the most common tokens across findings.
func topTokensLabel(findings []Finding) string {
	freq := map[string]int{}
	for _, f := range findings {
		seen := map[string]bool{}
		for _, t := range f.tokens {
			if !seen[t] {
				freq[t]++
				seen[t] = true
			}
		}
	}

	type kv struct {
		key   string
		count int
	}
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	var top []string
	for i := 0; i < len(sorted) && i < 4; i++ {
		top = append(top, sorted[i].key)
	}
	return strings.Join(top, " / ")
}

// FormatReport generates a human-readable report of duplicate finding clusters.
func FormatReport(clusters []Cluster) string {
	if len(clusters) == 0 {
		return "No duplicate findings detected.\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d duplicate finding cluster(s) — system may be narrating, not learning:\n\n", len(clusters)))

	for i, c := range clusters {
		sb.WriteString(fmt.Sprintf("Cluster %d: %q (%.0f%% avg similarity, %d occurrences)\n",
			i+1, c.Label, c.Similarity*100, len(c.Findings)))
		for _, f := range c.Findings {
			sb.WriteString(fmt.Sprintf("  - %s (finding #%d): %s\n", f.SourceFile, f.FindingNum, f.Title))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
