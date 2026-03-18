package kbmetrics

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AutoLink represents a suggested link between an orphaned investigation and a target.
type AutoLink struct {
	InvestigationPath string `json:"investigation_path"` // relative path like .kb/investigations/...
	TargetPath        string `json:"target_path"`        // absolute path to target file
	TargetName        string `json:"target_name"`        // human-readable name
	TargetType        string `json:"target_type"`        // "model", "thread", "decision"
	Score             int    `json:"score"`              // match score (higher = better)
	MatchedKeywords   []string `json:"matched_keywords,omitempty"`
}

// AutoLinkReport summarizes the auto-linking run.
type AutoLinkReport struct {
	Scanned   int        `json:"scanned"`   // positive-unlinked investigations scanned
	Matched   int        `json:"matched"`   // investigations with at least one match
	Links     []AutoLink `json:"links"`     // all suggested links
}

// kbTarget represents a linkable KB artifact (model, thread, decision).
type kbTarget struct {
	Name     string
	Type     string   // "model", "thread", "decision"
	Path     string   // absolute path to the file
	Keywords []string // extracted keywords for matching
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"is": true, "it": true, "by": true, "as": true, "be": true, "do": true,
	"if": true, "no": true, "so": true, "up": true, "we": true, "my": true,
	"with": true, "from": true, "this": true, "that": true, "was": true,
	"are": true, "has": true, "had": true, "not": true, "how": true,
	"what": true, "when": true, "where": true, "why": true, "who": true,
	"will": true, "can": true, "did": true, "does": true, "should": true,
	"would": true, "could": true, "may": true, "might": true, "must": true,
	"its": true, "all": true, "each": true, "some": true, "any": true,
	"into": true, "over": true, "also": true, "than": true, "then": true,
	"been": true, "have": true, "were": true, "being": true, "more": true,
	"which": true, "about": true, "between": true, "through": true,
	"investigation": true, "inv": true, "see": true, "based": true,
	"question": true, "findings": true, "finding": true, "result": true,
	"orch": true, "go": true, "cmd": true, "pkg": true,
	"model": true, "design": true, "implement": true, "add": true,
	"fix": true, "update": true, "new": true, "use": true, "work": true,
	"extract": true, "remove": true, "check": true, "test": true,
	"orchestrator": true, "synthesis": true, "agent": true, "session": true,
	"system": true, "review": true, "self": true, "create": true,
}

var wordSplitter = regexp.MustCompile(`[^a-z0-9]+`)

// ExtractKeywords extracts meaningful keywords from an investigation filename and content.
func ExtractKeywords(filename string, content string) []string {
	seen := make(map[string]bool)
	var keywords []string

	addWord := func(w string) {
		w = strings.ToLower(w)
		if len(w) < 3 || stopWords[w] || seen[w] {
			return
		}
		seen[w] = true
		keywords = append(keywords, w)
	}

	// Extract from filename (strip date prefix and extension)
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Remove date prefix like "2026-03-17-"
	if len(base) > 11 && base[4] == '-' && base[7] == '-' && base[10] == '-' {
		base = base[11:]
	}
	for _, part := range wordSplitter.Split(base, -1) {
		addWord(part)
	}

	// Extract from title (first heading)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			title = strings.TrimPrefix(title, "Investigation: ")
			title = strings.TrimPrefix(title, "Investigation — ")
			for _, part := range wordSplitter.Split(strings.ToLower(title), -1) {
				addWord(part)
			}
			break
		}
	}

	// Extract from H2 headings and bold terms
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			heading := strings.TrimPrefix(trimmed, "## ")
			for _, part := range wordSplitter.Split(strings.ToLower(heading), -1) {
				addWord(part)
			}
		}
		// Extract **Domain:** value
		if strings.HasPrefix(trimmed, "**Domain:**") {
			domain := strings.TrimPrefix(trimmed, "**Domain:**")
			for _, part := range wordSplitter.Split(strings.ToLower(domain), -1) {
				addWord(part)
			}
		}
	}

	return keywords
}

// ScoreMatch computes a match score between investigation keywords and a target.
// Returns 0 for no match. Higher is better.
// Requires at least one keyword to match a name part (not just content keywords).
func ScoreMatch(invKeywords []string, targetName string, targetKeywords []string) int {
	// Build target keyword set
	targetSet := make(map[string]bool, len(targetKeywords))
	for _, kw := range targetKeywords {
		targetSet[strings.ToLower(kw)] = true
	}

	// Build target name parts set
	nameParts := make(map[string]bool)
	for _, part := range wordSplitter.Split(strings.ToLower(targetName), -1) {
		if len(part) >= 3 && !stopWords[part] {
			nameParts[part] = true
		}
	}

	score := 0
	nameMatches := 0

	for _, kw := range invKeywords {
		kw = strings.ToLower(kw)
		// Direct keyword match: +1
		if targetSet[kw] {
			score++
		}
		// Model name match: +2 (bonus for matching the name itself)
		if nameParts[kw] {
			score += 2
			nameMatches++
		}
	}

	// Require at least one name-part match to avoid spurious content-only matches.
	if nameMatches == 0 {
		return 0
	}

	return score
}

// FindAutoLinks scans for positive-unlinked orphaned investigations and suggests
// links to models, threads, and decisions based on topic matching.
// minScore is the minimum match score required to suggest a link.
func FindAutoLinks(kbDir string, minScore int) ([]AutoLink, error) {
	if minScore < 1 {
		minScore = 1
	}

	// Step 1: Find positive-unlinked orphans
	orphans, err := findPositiveUnlinkedOrphans(kbDir)
	if err != nil {
		return nil, fmt.Errorf("find orphans: %w", err)
	}
	if len(orphans) == 0 {
		return nil, nil
	}

	// Step 2: Collect linkable targets (models, threads, decisions)
	targets, err := collectTargets(kbDir)
	if err != nil {
		return nil, fmt.Errorf("collect targets: %w", err)
	}
	if len(targets) == 0 {
		return nil, nil
	}

	// Step 3: Match each orphan against targets
	var links []AutoLink
	for _, orphan := range orphans {
		data, err := os.ReadFile(orphan.absPath)
		if err != nil {
			continue
		}
		content := string(data)
		invKeywords := ExtractKeywords(filepath.Base(orphan.absPath), content)
		if len(invKeywords) == 0 {
			continue
		}

		// Find best matching target
		var bestLink *AutoLink
		bestScore := 0

		for _, target := range targets {
			score := ScoreMatch(invKeywords, target.Name, target.Keywords)
			if score >= minScore && score > bestScore {
				// Collect matched keywords for transparency
				var matched []string
				nameSet := make(map[string]bool)
				for _, p := range wordSplitter.Split(strings.ToLower(target.Name), -1) {
					if len(p) >= 3 {
						nameSet[p] = true
					}
				}
				kwSet := make(map[string]bool)
				for _, kw := range target.Keywords {
					kwSet[strings.ToLower(kw)] = true
				}
				for _, kw := range invKeywords {
					if nameSet[kw] || kwSet[kw] {
						matched = append(matched, kw)
					}
				}

				bestScore = score
				bestLink = &AutoLink{
					InvestigationPath: orphan.relPath,
					TargetPath:        target.Path,
					TargetName:        target.Name,
					TargetType:        target.Type,
					Score:             score,
					MatchedKeywords:   matched,
				}
			}
		}

		if bestLink != nil {
			links = append(links, *bestLink)
		}
	}

	return links, nil
}

type orphanInfo struct {
	relPath string
	absPath string
}

// findPositiveUnlinkedOrphans returns orphaned investigations classified as positive-unlinked.
func findPositiveUnlinkedOrphans(kbDir string) ([]orphanInfo, error) {
	report, err := ComputeStratifiedOrphanRate(kbDir)
	if err != nil {
		return nil, err
	}

	paths := report.CategoryPaths[CategoryPositiveUnlinked]
	var orphans []orphanInfo
	for _, relPath := range paths {
		absPath := filepath.Join(filepath.Dir(kbDir), relPath)
		orphans = append(orphans, orphanInfo{relPath: relPath, absPath: absPath})
	}
	return orphans, nil
}

// collectTargets gathers all models, threads, and decisions as linkable targets.
func collectTargets(kbDir string) ([]kbTarget, error) {
	var targets []kbTarget

	// Models
	modelsDir := filepath.Join(kbDir, "models")
	if entries, err := os.ReadDir(modelsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name == "archived" || name == "model-relationships" {
				continue
			}
			modelPath := filepath.Join(modelsDir, name, "model.md")
			if _, err := os.Stat(modelPath); err != nil {
				continue
			}
			keywords := extractTargetKeywords(modelPath, name)
			targets = append(targets, kbTarget{
				Name:     name,
				Type:     "model",
				Path:     modelPath,
				Keywords: keywords,
			})
		}
	}

	// Threads
	threadsDir := filepath.Join(kbDir, "threads")
	if entries, err := os.ReadDir(threadsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			threadPath := filepath.Join(threadsDir, entry.Name())
			name := strings.TrimSuffix(entry.Name(), ".md")
			// Strip date prefix
			if len(name) > 11 && name[4] == '-' && name[7] == '-' && name[10] == '-' {
				name = name[11:]
			}
			keywords := extractTargetKeywords(threadPath, name)
			targets = append(targets, kbTarget{
				Name:     name,
				Type:     "thread",
				Path:     threadPath,
				Keywords: keywords,
			})
		}
	}

	// Decisions
	decisionsDir := filepath.Join(kbDir, "decisions")
	if entries, err := os.ReadDir(decisionsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			decPath := filepath.Join(decisionsDir, entry.Name())
			name := strings.TrimSuffix(entry.Name(), ".md")
			if len(name) > 11 && name[4] == '-' && name[7] == '-' && name[10] == '-' {
				name = name[11:]
			}
			keywords := extractTargetKeywords(decPath, name)
			targets = append(targets, kbTarget{
				Name:     name,
				Type:     "decision",
				Path:     decPath,
				Keywords: keywords,
			})
		}
	}

	return targets, nil
}

// extractTargetKeywords extracts keywords from a target file's name and content.
func extractTargetKeywords(path string, name string) []string {
	seen := make(map[string]bool)
	var keywords []string

	addWord := func(w string) {
		w = strings.ToLower(w)
		if len(w) < 3 || stopWords[w] || seen[w] {
			return
		}
		seen[w] = true
		keywords = append(keywords, w)
	}

	// Name parts
	for _, part := range wordSplitter.Split(strings.ToLower(name), -1) {
		addWord(part)
	}

	// Read file content for title and domain
	data, err := os.ReadFile(path)
	if err != nil {
		return keywords
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Title
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			for _, part := range wordSplitter.Split(strings.ToLower(title), -1) {
				addWord(part)
			}
		}
		// Domain field
		if strings.HasPrefix(trimmed, "**Domain:**") {
			domain := strings.TrimPrefix(trimmed, "**Domain:**")
			for _, part := range wordSplitter.Split(strings.ToLower(domain), -1) {
				addWord(part)
			}
		}
		// title: in frontmatter
		if strings.HasPrefix(trimmed, "title:") {
			title := strings.Trim(strings.TrimPrefix(trimmed, "title:"), " \"'")
			for _, part := range wordSplitter.Split(strings.ToLower(title), -1) {
				addWord(part)
			}
		}
	}

	return keywords
}

// ApplyAutoLinks writes investigation references to target files.
// Returns the number of links successfully applied.
func ApplyAutoLinks(links []AutoLink) (int, error) {
	applied := 0
	for _, link := range links {
		if err := appendReference(link); err != nil {
			return applied, fmt.Errorf("apply link %s → %s: %w",
				link.InvestigationPath, link.TargetName, err)
		}
		applied++
	}
	return applied, nil
}

const autoLinkSection = "## Auto-Linked Investigations"

// appendReference adds an investigation reference to the target file.
func appendReference(link AutoLink) error {
	data, err := os.ReadFile(link.TargetPath)
	if err != nil {
		return err
	}

	content := string(data)
	refLine := fmt.Sprintf("- %s", link.InvestigationPath)

	// Check if already referenced (defensive)
	if strings.Contains(content, link.InvestigationPath) {
		return nil
	}

	// If section exists, append to it
	if strings.Contains(content, autoLinkSection) {
		// Find the section and append after existing entries
		lines := strings.Split(content, "\n")
		var result []string
		inSection := false
		inserted := false

		for i, line := range lines {
			result = append(result, line)
			if strings.TrimSpace(line) == autoLinkSection {
				inSection = true
				continue
			}
			if inSection && !inserted {
				// Find end of list items in this section
				trimmed := strings.TrimSpace(line)
				isListItem := strings.HasPrefix(trimmed, "- ")
				nextIsListItem := false
				if i+1 < len(lines) {
					nextIsListItem = strings.HasPrefix(strings.TrimSpace(lines[i+1]), "- ")
				}
				// Insert after last list item in section
				if isListItem && !nextIsListItem {
					result = append(result, refLine)
					inserted = true
					inSection = false
				}
				// Empty section — insert after header + blank line
				if trimmed == "" && !isListItem {
					result = append(result, refLine)
					inserted = true
					inSection = false
				}
			}
		}
		// If section was at the end with no items after it
		if inSection && !inserted {
			result = append(result, refLine)
		}

		return os.WriteFile(link.TargetPath, []byte(strings.Join(result, "\n")), 0644)
	}

	// Append new section at end
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += fmt.Sprintf("\n%s\n\n%s\n", autoLinkSection, refLine)

	return os.WriteFile(link.TargetPath, []byte(content), 0644)
}
