package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// KBArtifactsResponse is the JSON structure returned by /api/kb/artifacts.
type KBArtifactsResponse struct {
	NeedsDecision []ArtifactFeedItem            `json:"needs_decision"`
	Recent        []ArtifactFeedItem            `json:"recent"`
	ByType        map[string][]ArtifactFeedItem `json:"by_type"`
	ProjectDir    string                        `json:"project_dir,omitempty"`
	Error         string                        `json:"error,omitempty"`
}

// ArtifactFeedItem represents a knowledge base artifact in the artifact feed.
type ArtifactFeedItem struct {
	Path           string    `json:"path"`           // Relative path from project root
	Title          string    `json:"title"`          // From frontmatter or filename
	Type           string    `json:"type"`           // investigation, decision, model, guide, principle
	Status         string    `json:"status"`         // Status field from frontmatter
	Date           string    `json:"date"`           // Date from frontmatter or filename
	Summary        string    `json:"summary"`        // First paragraph or summary from frontmatter
	Recommendation bool      `json:"recommendation"` // True if investigation has recommendation section
	ModifiedAt     time.Time `json:"modified_at"`    // File modification time
	RelativeTime   string    `json:"relative_time"`  // Human-readable relative time (e.g., "2h ago")
}

// ArtifactFrontmatter represents common YAML frontmatter fields in KB artifacts.
type ArtifactFrontmatter struct {
	Title   string `yaml:"title"`
	Status  string `yaml:"status"`
	Date    string `yaml:"date"`
	Summary string `yaml:"summary"`
}

// handleKBArtifacts returns knowledge base artifacts organized by attention category.
// Used by Work Graph Artifact Feed (Phase 3).
//
// Query params:
//   - project_dir: Project directory to query (defaults to sourceDir)
//   - since: Time filter for "recently updated" (e.g., "7d", "24h", "30d", "all")
func (s *Server) handleKBArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query params
	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		projectDir, _ = s.currentProjectDir()
	}

	sinceParam := r.URL.Query().Get("since")
	if sinceParam == "" {
		sinceParam = "7d" // Default to 7 days
	}

	// Parse since parameter
	sinceDuration, err := parseSinceDuration(sinceParam)
	if err != nil {
		resp := KBArtifactsResponse{
			NeedsDecision: []ArtifactFeedItem{},
			Recent:        []ArtifactFeedItem{},
			ByType:        map[string][]ArtifactFeedItem{},
			ProjectDir:    projectDir,
			Error:         fmt.Sprintf("Invalid since parameter: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Scan .kb/ directory
	kbDir := filepath.Join(projectDir, ".kb")
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		resp := KBArtifactsResponse{
			NeedsDecision: []ArtifactFeedItem{},
			Recent:        []ArtifactFeedItem{},
			ByType:        map[string][]ArtifactFeedItem{},
			ProjectDir:    projectDir,
			Error:         "No .kb/ directory found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Scan all artifact types
	artifacts, err := scanKBArtifacts(projectDir, kbDir, gitArtifactModifiedAt(projectDir))
	if err != nil {
		resp := KBArtifactsResponse{
			NeedsDecision: []ArtifactFeedItem{},
			Recent:        []ArtifactFeedItem{},
			ByType:        map[string][]ArtifactFeedItem{},
			ProjectDir:    projectDir,
			Error:         fmt.Sprintf("Failed to scan artifacts: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Keep ordering deterministic and newest-first for all feed sections.
	sortArtifactsByRecency(artifacts)

	// Categorize artifacts
	needsDecision := filterNeedsDecision(artifacts)
	recent := filterRecent(artifacts, sinceDuration)
	byType := groupByType(artifacts)

	resp := KBArtifactsResponse{
		NeedsDecision: needsDecision,
		Recent:        recent,
		ByType:        byType,
		ProjectDir:    projectDir,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func gitArtifactModifiedAt(projectDir string) map[string]time.Time {
	cmd := exec.Command("git", "log", "--pretty=format:__TS__%ct", "--name-only", "--", ".kb")
	cmd.Dir = projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	return parseGitArtifactModifiedAt(string(output))
}

func parseGitArtifactModifiedAt(output string) map[string]time.Time {
	result := map[string]time.Time{}
	var current time.Time

	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "__TS__") {
			unix, err := strconv.ParseInt(strings.TrimPrefix(line, "__TS__"), 10, 64)
			if err != nil {
				current = time.Time{}
				continue
			}
			current = time.Unix(unix, 0).UTC()
			continue
		}

		if current.IsZero() {
			continue
		}

		path := filepath.ToSlash(line)
		if _, ok := result[path]; ok {
			continue
		}
		result[path] = current
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// parseSinceDuration parses time duration strings like "7d", "24h", "30d", "all".
func parseSinceDuration(since string) (time.Duration, error) {
	if since == "all" {
		return time.Duration(0), nil // 0 means no filter
	}

	// Parse format like "7d", "24h"
	re := regexp.MustCompile(`^(\d+)([dhm])$`)
	matches := re.FindStringSubmatch(since)
	if matches == nil {
		return 0, fmt.Errorf("invalid format, expected format like '7d', '24h', '30d'")
	}

	value := matches[1]
	unit := matches[2]

	var duration time.Duration
	switch unit {
	case "d":
		days := 0
		fmt.Sscanf(value, "%d", &days)
		duration = time.Duration(days) * 24 * time.Hour
	case "h":
		hours := 0
		fmt.Sscanf(value, "%d", &hours)
		duration = time.Duration(hours) * time.Hour
	case "m":
		minutes := 0
		fmt.Sscanf(value, "%d", &minutes)
		duration = time.Duration(minutes) * time.Minute
	}

	return duration, nil
}

// scanKBArtifacts scans the .kb/ directory and returns all artifacts.
func scanKBArtifacts(projectDir, kbDir string, gitModifiedAt map[string]time.Time) ([]ArtifactFeedItem, error) {
	var artifacts []ArtifactFeedItem

	// Scan investigations
	invDir := filepath.Join(kbDir, "investigations")
	if _, err := os.Stat(invDir); err == nil {
		invArtifacts, err := scanArtifactDir(projectDir, invDir, "investigation", gitModifiedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan investigations: %w", err)
		}
		artifacts = append(artifacts, invArtifacts...)
	}

	// Scan decisions
	decDir := filepath.Join(kbDir, "decisions")
	if _, err := os.Stat(decDir); err == nil {
		decArtifacts, err := scanArtifactDir(projectDir, decDir, "decision", gitModifiedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan decisions: %w", err)
		}
		artifacts = append(artifacts, decArtifacts...)
	}

	// Scan models
	modelDir := filepath.Join(kbDir, "models")
	if _, err := os.Stat(modelDir); err == nil {
		modelArtifacts, err := scanArtifactDir(projectDir, modelDir, "model", gitModifiedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan models: %w", err)
		}
		artifacts = append(artifacts, modelArtifacts...)
	}

	// Scan guides
	guideDir := filepath.Join(kbDir, "guides")
	if _, err := os.Stat(guideDir); err == nil {
		guideArtifacts, err := scanArtifactDir(projectDir, guideDir, "guide", gitModifiedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guides: %w", err)
		}
		artifacts = append(artifacts, guideArtifacts...)
	}

	// Scan principles.md
	principlesPath := filepath.Join(kbDir, "principles.md")
	if _, err := os.Stat(principlesPath); err == nil {
		artifact, err := parseArtifact(projectDir, principlesPath, "principle", gitModifiedAt)
		if err == nil {
			artifacts = append(artifacts, artifact)
		}
	}

	return artifacts, nil
}

// scanArtifactDir scans a directory for markdown files and parses them as artifacts.
func scanArtifactDir(projectDir, dir string, artifactType string, gitModifiedAt map[string]time.Time) ([]ArtifactFeedItem, error) {
	var artifacts []ArtifactFeedItem

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Recursively scan subdirectories
			subDir := filepath.Join(dir, entry.Name())
			subArtifacts, err := scanArtifactDir(projectDir, subDir, artifactType, gitModifiedAt)
			if err != nil {
				continue // Skip directories that fail to scan
			}
			artifacts = append(artifacts, subArtifacts...)
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue // Skip non-markdown files
		}

		path := filepath.Join(dir, entry.Name())
		artifact, err := parseArtifact(projectDir, path, artifactType, gitModifiedAt)
		if err != nil {
			continue // Skip files that fail to parse
		}

		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// parseArtifact parses a markdown file and extracts artifact metadata.
func parseArtifact(projectDir, path string, artifactType string, gitModifiedAt map[string]time.Time) (ArtifactFeedItem, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ArtifactFeedItem{}, err
	}

	// Get file info for modification time
	info, err := os.Stat(path)
	if err != nil {
		return ArtifactFeedItem{}, err
	}

	// Parse frontmatter
	frontmatter := parseFrontmatter(string(content))

	// Extract title (from frontmatter or filename)
	title := frontmatter.Title
	if title == "" {
		title = extractTitleFromFilename(filepath.Base(path))
	}

	// Extract status
	status := frontmatter.Status

	// Extract date (from frontmatter or filename)
	date := frontmatter.Date
	if date == "" {
		date = extractDateFromFilename(filepath.Base(path))
	}

	// Extract summary (from frontmatter or first paragraph)
	summary := frontmatter.Summary
	if summary == "" {
		summary = extractFirstParagraph(string(content))
	}

	// Check if investigation has recommendation section
	hasRecommendation := false
	if artifactType == "investigation" {
		hasRecommendation = hasRecommendationSection(string(content))
	}

	// Calculate relative path from project root
	relativePath, err := filepath.Rel(projectDir, path)
	if err != nil {
		relativePath = path
	}

	modifiedAt := info.ModTime()
	if gitModifiedAt != nil {
		if gitTime, ok := gitModifiedAt[filepath.ToSlash(relativePath)]; ok {
			modifiedAt = gitTime
		}
	}

	// Calculate relative time
	relativeTime := formatRelativeTime(modifiedAt)

	return ArtifactFeedItem{
		Path:           relativePath,
		Title:          title,
		Type:           artifactType,
		Status:         status,
		Date:           date,
		Summary:        summary,
		Recommendation: hasRecommendation,
		ModifiedAt:     modifiedAt,
		RelativeTime:   relativeTime,
	}, nil
}

// parseFrontmatter extracts YAML frontmatter from markdown content.
func parseFrontmatter(content string) ArtifactFrontmatter {
	var fm ArtifactFrontmatter

	// Check if content starts with YAML frontmatter (---)
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return fm
	}

	// Find end of frontmatter
	lines := strings.Split(content, "\n")
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return fm
	}

	// Extract YAML content
	yamlContent := strings.Join(lines[1:endIndex], "\n")

	// Parse YAML
	yaml.Unmarshal([]byte(yamlContent), &fm)

	return fm
}

// extractTitleFromFilename extracts a human-readable title from a filename.
// Example: "2026-01-31-design-work-graph-phase3-artifact-feed.md" -> "Design Work Graph Phase3 Artifact Feed"
func extractTitleFromFilename(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, ".md")

	// Remove date prefix (YYYY-MM-DD-)
	datePrefix := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
	name = datePrefix.ReplaceAllString(name, "")

	// Remove type prefix (inv-, design-, etc.)
	typePrefix := regexp.MustCompile(`^(inv|design|decision|model|guide)-`)
	name = typePrefix.ReplaceAllString(name, "")

	// Replace hyphens with spaces and title case
	words := strings.Split(name, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// extractDateFromFilename extracts date from filename.
// Example: "2026-01-31-design-..." -> "2026-01-31"
func extractDateFromFilename(filename string) string {
	datePattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)
	matches := datePattern.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractFirstParagraph extracts a meaningful summary from artifact content.
// Priority: 1) **Delta:** line (D.E.K.N. format), 2) First non-comment paragraph
func extractFirstParagraph(content string) string {
	lines := strings.Split(content, "\n")

	// Skip frontmatter
	startIndex := 0
	if strings.HasPrefix(content, "---") {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				startIndex = i + 1
				break
			}
		}
	}

	// First pass: Look for **Delta:** line (D.E.K.N. format summary)
	for i := startIndex; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "**Delta:**") {
			// Extract content after "**Delta:**"
			delta := strings.TrimPrefix(line, "**Delta:**")
			delta = strings.TrimSpace(delta)
			if delta != "" {
				if len(delta) > 300 {
					delta = delta[:300] + "..."
				}
				return delta
			}
		}
	}

	// Second pass: Find first non-empty, non-heading, non-comment paragraph
	inHTMLComment := false
	for i := startIndex; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Track HTML comment blocks
		if strings.Contains(line, "<!--") {
			inHTMLComment = true
		}
		if strings.Contains(line, "-->") {
			inHTMLComment = false
			continue // Skip the closing comment line
		}
		if inHTMLComment {
			continue
		}

		// Skip empty lines, headings, and standalone comment markers
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "<!--") {
			continue
		}

		// Skip metadata lines (e.g., **Date:**, **Status:**, etc.)
		if strings.HasPrefix(line, "**") && strings.Contains(line, ":**") {
			// But not Delta - we already handled that above
			if !strings.HasPrefix(line, "**Delta:**") {
				continue
			}
		}

		// Found first paragraph - collect until empty line or heading
		var paragraph []string
		for j := i; j < len(lines); j++ {
			pline := strings.TrimSpace(lines[j])
			if pline == "" || strings.HasPrefix(pline, "#") {
				break
			}
			// Stop at HTML comments
			if strings.HasPrefix(pline, "<!--") {
				break
			}
			paragraph = append(paragraph, pline)
		}

		result := strings.Join(paragraph, " ")
		if len(result) > 300 {
			result = result[:300] + "..."
		}
		return result
	}

	return ""
}

// hasRecommendationSection checks if content has a "## Recommendation" section.
func hasRecommendationSection(content string) bool {
	// Look for ## Recommendation or ## Recommendations heading
	re := regexp.MustCompile(`(?m)^##\s+Recommendations?`)
	return re.MatchString(content)
}

// formatRelativeTime formats a time.Time as a human-readable relative time.
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", minutes)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	}
	if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}

	weeks := int(duration.Hours() / 24 / 7)
	if weeks == 1 {
		return "1w ago"
	}
	return fmt.Sprintf("%dw ago", weeks)
}

// filterNeedsDecision returns artifacts that need human decision/action.
func filterNeedsDecision(artifacts []ArtifactFeedItem) []ArtifactFeedItem {
	var result []ArtifactFeedItem

	for _, a := range artifacts {
		needsAction := false

		// Investigations with recommendations
		if a.Type == "investigation" && a.Status == "Active" && a.Recommendation {
			needsAction = true
		}

		// Stale investigations (Active but > 7 days old)
		if a.Type == "investigation" && a.Status == "Active" {
			if time.Since(a.ModifiedAt) > 7*24*time.Hour {
				needsAction = true
			}
		}

		// Proposed decisions
		if a.Type == "decision" && a.Status == "Proposed" {
			needsAction = true
		}

		if needsAction {
			result = append(result, a)
		}
	}

	return result
}

// filterRecent returns artifacts modified within the given duration.
func filterRecent(artifacts []ArtifactFeedItem, since time.Duration) []ArtifactFeedItem {
	if since == 0 {
		return artifacts // "all" filter
	}

	var result []ArtifactFeedItem
	cutoff := time.Now().Add(-since)

	for _, a := range artifacts {
		if a.ModifiedAt.After(cutoff) {
			result = append(result, a)
		}
	}

	return result
}

// groupByType groups artifacts by their type.
func groupByType(artifacts []ArtifactFeedItem) map[string][]ArtifactFeedItem {
	byType := make(map[string][]ArtifactFeedItem)

	for _, a := range artifacts {
		byType[a.Type] = append(byType[a.Type], a)
	}

	return byType
}

// sortArtifactsByRecency sorts in-place by modified time descending, with path tie-break.
func sortArtifactsByRecency(artifacts []ArtifactFeedItem) {
	sort.SliceStable(artifacts, func(i, j int) bool {
		if !artifacts[i].ModifiedAt.Equal(artifacts[j].ModifiedAt) {
			return artifacts[i].ModifiedAt.After(artifacts[j].ModifiedAt)
		}
		return artifacts[i].Path < artifacts[j].Path
	})
}

// ArtifactContentResponse is the JSON structure returned by /api/kb/artifact/content.
type ArtifactContentResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// handleKBArtifactContent returns the full content of a specific artifact.
// Query params:
//   - path: Relative path to the artifact (e.g., ".kb/investigations/2026-01-31-design-example.md")
//   - project_dir: Project directory (defaults to sourceDir)
func (s *Server) handleKBArtifactContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query params
	artifactPath := r.URL.Query().Get("path")
	if artifactPath == "" {
		resp := ArtifactContentResponse{
			Error: "Missing required 'path' parameter",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		projectDir, _ = s.currentProjectDir()
	}

	// Construct full path and validate it's within project
	fullPath := filepath.Join(projectDir, artifactPath)
	fullPath = filepath.Clean(fullPath)

	// Security: Ensure path is within project directory
	if !strings.HasPrefix(fullPath, projectDir) {
		resp := ArtifactContentResponse{
			Path:  artifactPath,
			Error: "Invalid path: must be within project directory",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		resp := ArtifactContentResponse{
			Path:  artifactPath,
			Error: fmt.Sprintf("Failed to read file: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := ArtifactContentResponse{
		Path:    artifactPath,
		Content: string(content),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
