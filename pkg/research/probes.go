// Package research provides claim status aggregation for the orch research command.
//
// It scans probe files from .kb/models/*/probes/, extracts claim references and
// verdicts, and cross-references them with claims from claims.yaml to produce
// a per-model, per-claim research status view.
package research

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ProbeResult represents a parsed probe file with its claim references and verdict.
type ProbeResult struct {
	Path    string   // full path to probe file
	Date    string   // from filename prefix (YYYY-MM-DD)
	Claims  []string // claim IDs referenced (e.g., ["NI-01", "NI-03"])
	Verdict string   // confirms, contradicts, extends, scopes, disconfirms, etc.
	Status  string   // Active, Complete
	Title   string   // from # Probe: line
}

// ScanProbes reads all probe markdown files from a probes/ directory.
// Extracts claim references and verdicts from the frontmatter-style header.
func ScanProbes(probesDir string) ([]ProbeResult, error) {
	entries, err := os.ReadDir(probesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []ProbeResult
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(probesDir, entry.Name())
		pr, err := parseProbeFile(path)
		if err != nil {
			continue // skip unparseable probes
		}
		results = append(results, *pr)
	}
	return results, nil
}

// parseProbeFile extracts frontmatter fields from a probe markdown file.
// Expected format:
//
//	# Probe: {title}
//	**Model:** {model}
//	**Date:** {date}
//	**Status:** {status}
//	**claim:** {claim-ids}
//	**verdict:** {verdict}
func parseProbeFile(path string) (*ProbeResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	pr := &ProbeResult{Path: path}

	// Extract date from filename (YYYY-MM-DD prefix)
	base := filepath.Base(path)
	if len(base) >= 10 {
		pr.Date = base[:10]
	}

	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount > 15 { // frontmatter is in first ~10 lines
			break
		}
		line := scanner.Text()

		if strings.HasPrefix(line, "# Probe:") {
			pr.Title = strings.TrimSpace(strings.TrimPrefix(line, "# Probe:"))
			continue
		}

		if strings.HasPrefix(line, "**claim:**") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "**claim:**"))
			pr.Claims = parseClaimRefs(raw)
			continue
		}

		if strings.HasPrefix(line, "**verdict:**") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "**verdict:**"))
			pr.Verdict = normalizeVerdict(raw)
			continue
		}

		if strings.HasPrefix(line, "**Status:**") {
			pr.Status = strings.TrimSpace(strings.TrimPrefix(line, "**Status:**"))
			continue
		}
	}

	return pr, scanner.Err()
}

// parseClaimRefs extracts claim IDs from a claim field value.
// Handles formats like:
//
//	"NI-01"
//	"NI-01, NI-03"
//	"CA-01, CA-02, CA-03, CA-04"
//	"n/a" -> empty
//	"extends (no prior claim)" -> empty
func parseClaimRefs(raw string) []string {
	lower := strings.ToLower(raw)
	if lower == "n/a" || strings.Contains(lower, "no prior claim") ||
		strings.Contains(lower, "no single claim") || strings.Contains(lower, "implicit") {
		return nil
	}

	parts := strings.Split(raw, ",")
	var ids []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Extract just the ID portion — stop at whitespace or parenthetical
		if idx := strings.IndexAny(p, " ("); idx > 0 {
			p = p[:idx]
		}
		// Validate it looks like a claim ID (PREFIX-NN)
		if isClaimID(p) {
			ids = append(ids, strings.ToUpper(p))
		}
	}
	return ids
}

// isClaimID returns true if s looks like a claim ID (e.g., NI-01, CA-06, DAO-15).
func isClaimID(s string) bool {
	if len(s) < 4 { // minimum: X-01
		return false
	}
	dashIdx := strings.LastIndex(s, "-")
	if dashIdx <= 0 || dashIdx >= len(s)-1 {
		return false
	}
	prefix := s[:dashIdx]
	suffix := s[dashIdx+1:]
	// prefix must be alphabetic
	for _, c := range prefix {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	// suffix must be numeric
	for _, c := range suffix {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(prefix) > 0 && len(suffix) > 0
}

// normalizeVerdict extracts the core verdict from potentially decorated values.
// "confirms (with extensions)" -> "confirms"
// "disconfirms (with extension)" -> "disconfirms"
// "scopes" -> "scopes"
func normalizeVerdict(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Take the first word as the core verdict
	if idx := strings.IndexAny(raw, " ("); idx > 0 {
		return strings.ToLower(raw[:idx])
	}
	return strings.ToLower(raw)
}
