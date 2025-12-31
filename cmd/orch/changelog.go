// changelog.go - Show aggregated changelog across ecosystem repos
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

var (
	// Changelog command flags
	changelogDays    int
	changelogProject string
	changelogJSON    bool
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Show aggregated changelog across ecosystem repos",
	Long: `Show aggregated changelog across ecosystem repos.

Aggregates git commits from all repos in Dylan's orchestration ecosystem
and groups them by date and category (skills, .kb, cmd, etc.).

By default, shows commits from the last 7 days across all ecosystem repos.
Use --project to filter to a specific repo, or --days to change the time range.

Examples:
  orch changelog                    # Show last 7 days across all repos
  orch changelog --days 14          # Show last 14 days
  orch changelog --project orch-go  # Show only orch-go commits
  orch changelog --project all      # Explicitly show all repos (default)
  orch changelog --json             # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChangelog()
	},
}

func init() {
	changelogCmd.Flags().IntVar(&changelogDays, "days", 7, "Number of days to include")
	changelogCmd.Flags().StringVar(&changelogProject, "project", "all", "Project to filter (or 'all' for all repos)")
	changelogCmd.Flags().BoolVar(&changelogJSON, "json", false, "Output as JSON")

	rootCmd.AddCommand(changelogCmd)
}

// ChangeType represents the semantic type of a change.
type ChangeType string

const (
	ChangeTypeDocumentation ChangeType = "documentation"
	ChangeTypeBehavioral    ChangeType = "behavioral"
	ChangeTypeStructural    ChangeType = "structural"
	ChangeTypeUnknown       ChangeType = "unknown"
)

// BlastRadius represents how many skills/components a change affects.
type BlastRadius string

const (
	BlastRadiusLocal          BlastRadius = "local"          // Single skill/file
	BlastRadiusCrossSkill     BlastRadius = "cross-skill"    // 2-5 skills affected
	BlastRadiusInfrastructure BlastRadius = "infrastructure" // Spawn system, all skills
)

// SemanticInfo contains parsed semantic information about a commit.
type SemanticInfo struct {
	ChangeType    ChangeType  `json:"change_type"`
	BlastRadius   BlastRadius `json:"blast_radius"`
	IsBreaking    bool        `json:"is_breaking"`
	CommitType    string      `json:"commit_type"`      // feat, fix, docs, etc.
	SemanticLabel string      `json:"semantic_label"`   // Human-readable badge
}

// CommitInfo represents a single git commit.
type CommitInfo struct {
	Hash         string       `json:"hash"`
	Subject      string       `json:"subject"`
	Author       string       `json:"author"`
	Date         time.Time    `json:"date"`
	DateStr      string       `json:"date_str"`
	Repo         string       `json:"repo"`
	Category     string       `json:"category"`
	Files        []string     `json:"files,omitempty"`
	SemanticInfo SemanticInfo `json:"semantic_info"`
}

// ChangelogResult represents the aggregated changelog.
type ChangelogResult struct {
	DateRange    DateRange               `json:"date_range"`
	TotalCommits int                     `json:"total_commits"`
	RepoCount    int                     `json:"repo_count"`
	MissingRepos []string                `json:"missing_repos,omitempty"`
	CommitsByDate map[string][]CommitInfo `json:"commits_by_date"`
	CommitsByCategory map[string]int     `json:"commits_by_category"`
	RepoStats    map[string]int          `json:"repo_stats"`
}

// getEcosystemRepos returns the list of ecosystem repos to scan.
// If a specific project is requested, returns only that repo.
func getEcosystemRepos() []string {
	if changelogProject != "all" {
		// Single project requested
		return []string{changelogProject}
	}
	
	// All ecosystem repos
	var repos []string
	for repo := range spawn.ExpandedOrchEcosystemRepos {
		repos = append(repos, repo)
	}
	sort.Strings(repos)
	return repos
}

// findRepoPath attempts to find the local path for a repo.
// Checks common locations like ~/Documents/personal/ and ~/
func findRepoPath(repoName string) (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	
	// Common paths to check
	paths := []string{
		filepath.Join(home, "Documents", "personal", repoName),
		filepath.Join(home, repoName),
		filepath.Join(home, "projects", repoName),
		filepath.Join(home, "code", repoName),
	}
	
	for _, path := range paths {
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			// Verify it's a git repo
			gitDir := filepath.Join(path, ".git")
			if _, err := os.Stat(gitDir); err == nil {
				return path, true
			}
		}
	}
	
	return "", false
}

// getGitLog retrieves git log for a repo in the specified time range.
// Returns nil if the command fails (repo doesn't exist, etc.).
func getGitLog(repoPath string, days int) ([]CommitInfo, error) {
	// Format: hash|subject|author|date
	format := "--format=%H|%s|%an|%aI"
	since := fmt.Sprintf("--since=%d days ago", days)
	
	cmd := exec.Command("git", "log", format, since, "--name-only")
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	return parseGitLog(string(output), filepath.Base(repoPath))
}

// parseGitLog parses git log output into CommitInfo structs.
func parseGitLog(output, repoName string) ([]CommitInfo, error) {
	var commits []CommitInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var current *CommitInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a commit line (contains |)
		if strings.Contains(line, "|") {
			// Save previous commit if exists
			if current != nil {
				current.Category = categorizeCommitByFiles(current.Files)
				// Check for more specific semantic category
				if semanticCat := inferSemanticCategory(current.Files); semanticCat != "" {
					current.Category = semanticCat
				}
				current.SemanticInfo = parseSemanticInfo(current.Subject, current.Files)
				commits = append(commits, *current)
			}

			// Parse new commit
			parts := strings.SplitN(line, "|", 4)
			if len(parts) < 4 {
				continue
			}

			date, _ := time.Parse(time.RFC3339, parts[3])
			current = &CommitInfo{
				Hash:    parts[0][:8], // Short hash
				Subject: parts[1],
				Author:  parts[2],
				Date:    date,
				DateStr: date.Format("2006-01-02"),
				Repo:    repoName,
				Files:   []string{},
			}
		} else if current != nil {
			// This is a file name
			current.Files = append(current.Files, line)
		}
	}

	// Don't forget the last commit
	if current != nil {
		current.Category = categorizeCommitByFiles(current.Files)
		// Check for more specific semantic category
		if semanticCat := inferSemanticCategory(current.Files); semanticCat != "" {
			current.Category = semanticCat
		}
		current.SemanticInfo = parseSemanticInfo(current.Subject, current.Files)
		commits = append(commits, *current)
	}

	return commits, nil
}

// categorizeCommitByFiles determines the category of a commit based on files changed.
// Categories: skills, kb, cmd, pkg, web, docs, config, other
func categorizeCommitByFiles(files []string) string {
	categories := map[string]int{
		"skills": 0,
		"kb":     0,
		"cmd":    0,
		"pkg":    0,
		"web":    0,
		"docs":   0,
		"config": 0,
		"other":  0,
	}
	
	for _, file := range files {
		switch {
		case strings.HasPrefix(file, "skills/") || strings.Contains(file, "/skills/"):
			categories["skills"]++
		case strings.HasPrefix(file, ".kb/") || strings.Contains(file, "/.kb/"):
			categories["kb"]++
		case strings.HasPrefix(file, "cmd/"):
			categories["cmd"]++
		case strings.HasPrefix(file, "pkg/"):
			categories["pkg"]++
		case strings.HasPrefix(file, "web/") || strings.HasPrefix(file, "src/"):
			categories["web"]++
		case strings.HasPrefix(file, "docs/"):
			categories["docs"]++
		case strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".json") || 
			strings.HasSuffix(file, ".toml") || file == "Makefile" || 
			file == "go.mod" || file == "go.sum" || file == "package.json":
			categories["config"]++
		default:
			categories["other"]++
		}
	}
	
	// Return the category with most files
	// Priority order ensures deterministic tie-breaking (more specific wins over "other")
	categoryPriority := []string{"skills", "kb", "cmd", "pkg", "web", "docs", "config", "other"}
	
	maxCategory := "other"
	maxCount := 0
	for _, cat := range categoryPriority {
		count := categories[cat]
		if count > maxCount {
			maxCategory = cat
			maxCount = count
		}
	}
	
	return maxCategory
}

// parseSemanticInfo extracts semantic information from a commit.
func parseSemanticInfo(subject string, files []string) SemanticInfo {
	info := SemanticInfo{
		ChangeType:  ChangeTypeUnknown,
		BlastRadius: BlastRadiusLocal,
		IsBreaking:  false,
		CommitType:  "",
	}

	// Parse conventional commit prefix
	info.CommitType, info.IsBreaking = parseConventionalCommit(subject)

	// Determine change type from commit prefix and files
	info.ChangeType = inferChangeType(info.CommitType, files)

	// Determine blast radius from files
	info.BlastRadius = inferBlastRadius(files)

	// Generate semantic label
	info.SemanticLabel = generateSemanticLabel(info)

	return info
}

// parseConventionalCommit parses conventional commit format (type: message or type(scope): message).
// Returns the commit type and whether it's a breaking change.
func parseConventionalCommit(subject string) (commitType string, isBreaking bool) {
	subject = strings.TrimSpace(subject)

	// Check for BREAKING prefix
	if strings.HasPrefix(strings.ToUpper(subject), "BREAKING") {
		isBreaking = true
		// Remove BREAKING prefix to continue parsing
		subject = strings.TrimPrefix(strings.ToUpper(subject), "BREAKING")
		subject = strings.TrimPrefix(subject, ":")
		subject = strings.TrimPrefix(subject, " ")
		subject = strings.TrimSpace(subject)
	}

	// Check for BREAKING CHANGE in the message
	if strings.Contains(strings.ToUpper(subject), "BREAKING CHANGE") {
		isBreaking = true
	}

	// Check for ! before : (breaking change indicator)
	colonIdx := strings.Index(subject, ":")
	if colonIdx > 0 && colonIdx < len(subject)-1 {
		prefix := subject[:colonIdx]
		if strings.HasSuffix(prefix, "!") {
			isBreaking = true
			prefix = strings.TrimSuffix(prefix, "!")
		}

		// Remove scope if present: feat(scope) -> feat
		if parenIdx := strings.Index(prefix, "("); parenIdx > 0 {
			prefix = prefix[:parenIdx]
		}

		commitType = strings.ToLower(strings.TrimSpace(prefix))
	}

	return commitType, isBreaking
}

// inferChangeType determines the ChangeType from commit type and files.
func inferChangeType(commitType string, files []string) ChangeType {
	// First, try to infer from conventional commit type
	switch commitType {
	case "docs", "doc":
		return ChangeTypeDocumentation
	case "feat", "fix", "perf", "refactor":
		return ChangeTypeBehavioral
	case "build", "ci", "chore":
		return ChangeTypeStructural
	}

	// Fallback: infer from files
	docFiles := 0
	structuralFiles := 0
	behavioralFiles := 0

	for _, file := range files {
		lower := strings.ToLower(file)
		switch {
		// Documentation files
		case strings.HasSuffix(lower, ".md") ||
			strings.HasSuffix(lower, ".rst") ||
			strings.HasSuffix(lower, ".txt") ||
			strings.HasPrefix(file, "docs/") ||
			strings.Contains(file, "/docs/"):
			docFiles++

		// Structural files (config, build, etc.)
		case strings.HasSuffix(lower, ".yaml") ||
			strings.HasSuffix(lower, ".yml") ||
			strings.HasSuffix(lower, ".json") ||
			strings.HasSuffix(lower, ".toml") ||
			file == "Makefile" ||
			file == "go.mod" ||
			file == "go.sum" ||
			file == "package.json" ||
			strings.Contains(file, ".skillc/") ||
			strings.HasPrefix(file, ".github/"):
			structuralFiles++

		// Behavioral files (code)
		case strings.HasSuffix(lower, ".go") ||
			strings.HasSuffix(lower, ".ts") ||
			strings.HasSuffix(lower, ".js") ||
			strings.HasSuffix(lower, ".py") ||
			strings.HasSuffix(lower, ".svelte"):
			behavioralFiles++
		}
	}

	// Determine by majority
	if docFiles > 0 && docFiles >= structuralFiles && docFiles >= behavioralFiles {
		return ChangeTypeDocumentation
	}
	if structuralFiles > 0 && structuralFiles > behavioralFiles {
		return ChangeTypeStructural
	}
	if behavioralFiles > 0 {
		return ChangeTypeBehavioral
	}

	return ChangeTypeUnknown
}

// inferBlastRadius determines how many components a change affects.
func inferBlastRadius(files []string) BlastRadius {
	// Track what's affected
	skillsAffected := make(map[string]bool)
	hasInfrastructureChange := false
	hasSkillYamlChange := false
	hasCrossRepoImplication := false

	for _, file := range files {
		// Check for infrastructure changes
		if strings.Contains(file, "pkg/spawn/") ||
			strings.Contains(file, "pkg/verify/") ||
			strings.Contains(file, "skillc") ||
			file == "skill.yaml" ||
			strings.HasSuffix(file, "/skill.yaml") {
			hasInfrastructureChange = true
		}

		// Check for skill.yaml schema changes
		if strings.HasSuffix(file, "skill.yaml") || strings.HasSuffix(file, ".skillc/skill.yaml") {
			hasSkillYamlChange = true
		}

		// Track individual skills affected
		if strings.Contains(file, "skills/") {
			// Extract skill name from path like skills/worker/feature-impl/...
			parts := strings.Split(file, "/")
			for i, part := range parts {
				if part == "skills" && i+2 < len(parts) {
					skillName := parts[i+2] // e.g., "feature-impl"
					skillsAffected[skillName] = true
				}
			}
		}

		// Cross-repo implications
		if strings.Contains(file, "SPAWN_CONTEXT") ||
			strings.Contains(file, "orchestrator") && strings.HasSuffix(file, ".md") {
			hasCrossRepoImplication = true
		}
	}

	// Infrastructure takes precedence
	if hasInfrastructureChange || hasSkillYamlChange || hasCrossRepoImplication {
		return BlastRadiusInfrastructure
	}

	// Cross-skill if multiple skills affected
	if len(skillsAffected) >= 2 {
		return BlastRadiusCrossSkill
	}

	return BlastRadiusLocal
}

// generateSemanticLabel creates a human-readable badge for the commit.
func generateSemanticLabel(info SemanticInfo) string {
	var parts []string

	// Breaking indicator
	if info.IsBreaking {
		parts = append(parts, "BREAKING")
	}

	// Change type
	switch info.ChangeType {
	case ChangeTypeDocumentation:
		parts = append(parts, "docs")
	case ChangeTypeBehavioral:
		parts = append(parts, "behavioral")
	case ChangeTypeStructural:
		parts = append(parts, "structural")
	}

	// Blast radius (only if not local)
	switch info.BlastRadius {
	case BlastRadiusCrossSkill:
		parts = append(parts, "cross-skill")
	case BlastRadiusInfrastructure:
		parts = append(parts, "infrastructure")
	}

	if len(parts) == 0 {
		return ""
	}

	return "[" + strings.Join(parts, " | ") + "]"
}

// inferSemanticCategory provides more specific categorization for skill changes.
// Returns categories like: skill-behavioral, skill-docs, decision-record, investigation, etc.
func inferSemanticCategory(files []string) string {
	// Check for specific .kb patterns
	for _, file := range files {
		if strings.Contains(file, ".kb/decisions/") {
			return "decision-record"
		}
		if strings.Contains(file, ".kb/investigations/") {
			return "investigation"
		}
	}

	// Check for skill patterns
	skillFiles := 0
	skillDocsFiles := 0
	for _, file := range files {
		if strings.Contains(file, "skills/") || strings.Contains(file, "/skills/") {
			skillFiles++
			if strings.HasSuffix(file, ".md") {
				skillDocsFiles++
			}
		}
	}

	if skillFiles > 0 {
		if skillDocsFiles == skillFiles {
			return "skill-docs"
		}
		return "skill-behavioral"
	}

	return "" // Use default category
}

func runChangelog() error {
	repos := getEcosystemRepos()
	
	var allCommits []CommitInfo
	var missingRepos []string
	repoStats := make(map[string]int)
	
	for _, repoName := range repos {
		repoPath, found := findRepoPath(repoName)
		if !found {
			missingRepos = append(missingRepos, repoName)
			continue
		}
		
		commits, err := getGitLog(repoPath, changelogDays)
		if err != nil {
			// Warn but continue with other repos
			fmt.Fprintf(os.Stderr, "Warning: failed to get git log for %s: %v\n", repoName, err)
			continue
		}
		
		repoStats[repoName] = len(commits)
		allCommits = append(allCommits, commits...)
	}
	
	// Sort commits by date (newest first)
	sort.Slice(allCommits, func(i, j int) bool {
		return allCommits[i].Date.After(allCommits[j].Date)
	})
	
	// Group by date
	commitsByDate := make(map[string][]CommitInfo)
	commitsByCategory := make(map[string]int)
	for _, commit := range allCommits {
		commitsByDate[commit.DateStr] = append(commitsByDate[commit.DateStr], commit)
		commitsByCategory[commit.Category]++
	}
	
	// Build result
	result := ChangelogResult{
		DateRange: DateRange{
			Start: time.Now().AddDate(0, 0, -changelogDays).Format("2006-01-02"),
			End:   time.Now().Format("2006-01-02"),
		},
		TotalCommits:      len(allCommits),
		RepoCount:         len(repos) - len(missingRepos),
		MissingRepos:      missingRepos,
		CommitsByDate:     commitsByDate,
		CommitsByCategory: commitsByCategory,
		RepoStats:         repoStats,
	}
	
	if changelogJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	
	// Format human-readable output
	output := formatChangelog(result)
	fmt.Println(output)
	return nil
}

// formatChangelog formats the changelog for human-readable output.
func formatChangelog(result ChangelogResult) string {
	var lines []string
	
	// Header
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("=", 70))
	lines = append(lines, fmt.Sprintf("📋 ECOSYSTEM CHANGELOG (%s to %s)", result.DateRange.Start, result.DateRange.End))
	lines = append(lines, strings.Repeat("=", 70))
	
	// Summary
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("📊 Summary: %d commits across %d repos", result.TotalCommits, result.RepoCount))
	
	// Missing repos warning
	if len(result.MissingRepos) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("⚠️  Missing repos (not found locally): %s", strings.Join(result.MissingRepos, ", ")))
	}
	
	// Category breakdown
	if len(result.CommitsByCategory) > 0 {
		lines = append(lines, "")
		lines = append(lines, "📁 By Category:")
		// Sort categories by count
		type catCount struct {
			cat   string
			count int
		}
		var cats []catCount
		for cat, count := range result.CommitsByCategory {
			cats = append(cats, catCount{cat, count})
		}
		sort.Slice(cats, func(i, j int) bool {
			return cats[i].count > cats[j].count
		})
		for _, cc := range cats {
			lines = append(lines, fmt.Sprintf("   %s: %d", cc.cat, cc.count))
		}
	}
	
	// Repo breakdown
	if len(result.RepoStats) > 1 {
		lines = append(lines, "")
		lines = append(lines, "📦 By Repo:")
		// Sort repos by count
		type repoCount struct {
			repo  string
			count int
		}
		var repos []repoCount
		for repo, count := range result.RepoStats {
			repos = append(repos, repoCount{repo, count})
		}
		sort.Slice(repos, func(i, j int) bool {
			return repos[i].count > repos[j].count
		})
		for _, rc := range repos {
			lines = append(lines, fmt.Sprintf("   %s: %d", rc.repo, rc.count))
		}
	}
	
	// Commits by date
	if len(result.CommitsByDate) > 0 {
		lines = append(lines, "")
		lines = append(lines, strings.Repeat("-", 70))
		lines = append(lines, "")
		
		// Sort dates (newest first)
		var dates []string
		for date := range result.CommitsByDate {
			dates = append(dates, date)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(dates)))
		
		for _, date := range dates {
			commits := result.CommitsByDate[date]
			lines = append(lines, fmt.Sprintf("📅 %s (%d commits)", date, len(commits)))
			lines = append(lines, "")
			
			for _, commit := range commits {
				// Format: [repo] subject (category) - author
				categoryIcon := getCategoryIcon(commit.Category)
				// Add semantic label if present
				semanticLabel := ""
				if commit.SemanticInfo.SemanticLabel != "" {
					semanticLabel = " " + commit.SemanticInfo.SemanticLabel
				}
				// Show breaking changes prominently
				if commit.SemanticInfo.IsBreaking {
					lines = append(lines, fmt.Sprintf("  🚨 [%s] %s%s", commit.Repo, commit.Subject, semanticLabel))
				} else {
					lines = append(lines, fmt.Sprintf("  %s [%s] %s%s", categoryIcon, commit.Repo, commit.Subject, semanticLabel))
				}
				lines = append(lines, fmt.Sprintf("      by %s • %s", commit.Author, commit.Hash))
			}
			lines = append(lines, "")
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, "No commits found in the specified time range.")
	}
	
	lines = append(lines, strings.Repeat("=", 70))
	
	return strings.Join(lines, "\n")
}

// getCategoryIcon returns an emoji icon for a category.
func getCategoryIcon(category string) string {
	switch category {
	case "skills":
		return "🎯"
	case "skill-behavioral":
		return "🎯"
	case "skill-docs":
		return "📖"
	case "kb":
		return "📚"
	case "decision-record":
		return "📜"
	case "investigation":
		return "🔍"
	case "cmd":
		return "⚡"
	case "pkg":
		return "📦"
	case "web":
		return "🌐"
	case "docs":
		return "📝"
	case "config":
		return "⚙️"
	default:
		return "📄"
	}
}
