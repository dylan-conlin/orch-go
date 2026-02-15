package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type CommitType string

const (
	TypeFeat      CommitType = "feat"
	TypeFix       CommitType = "fix"
	TypeInv       CommitType = "inv"
	TypeArchitect CommitType = "architect"
	TypeBdSync    CommitType = "bd sync"
	TypeChore     CommitType = "chore"
	TypeRefactor  CommitType = "refactor"
	TypeTest      CommitType = "test"
	TypeDocs      CommitType = "docs"
	TypeWip       CommitType = "wip"
	TypeOther     CommitType = "other"
)

type Commit struct {
	Hash    string
	Date    string
	Message string
	Type    CommitType
}

type KnowledgeArtifact struct {
	Path string
	Type string // "investigation", "decision", "model"
}

type DayStats struct {
	Date               string              `json:"date"`
	TotalCommits       int                 `json:"total_commits"`
	CommitsByType      map[CommitType]int  `json:"commits_by_type"`
	KnowledgeArtifacts []KnowledgeArtifact `json:"knowledge_artifacts"`
	FeatureCommits     []string            `json:"feature_commits"`
	FixCommits         []string            `json:"fix_commits"`
	NetLOC             int                 `json:"net_loc"`
	FilesChanged       int                 `json:"files_changed"`
}

type Timeline struct {
	StartDate string               `json:"start_date"`
	EndDate   string               `json:"end_date"`
	TotalDays int                  `json:"total_days"`
	Days      map[string]*DayStats `json:"days"`
}

func parseCommitType(message string) CommitType {
	lower := strings.ToLower(message)

	if strings.HasPrefix(lower, "feat:") || strings.HasPrefix(lower, "feature:") {
		return TypeFeat
	}
	if strings.HasPrefix(lower, "fix:") {
		return TypeFix
	}
	if strings.HasPrefix(lower, "inv:") || strings.HasPrefix(lower, "investigation:") {
		return TypeInv
	}
	if strings.HasPrefix(lower, "architect:") {
		return TypeArchitect
	}
	if strings.HasPrefix(lower, "bd sync:") || strings.HasPrefix(lower, "bd:") {
		return TypeBdSync
	}
	if strings.HasPrefix(lower, "chore:") {
		return TypeChore
	}
	if strings.HasPrefix(lower, "refactor:") {
		return TypeRefactor
	}
	if strings.HasPrefix(lower, "test:") {
		return TypeTest
	}
	if strings.HasPrefix(lower, "docs:") {
		return TypeDocs
	}
	if strings.HasPrefix(lower, "wip:") {
		return TypeWip
	}

	return TypeOther
}

func parseCommitFile(path string) ([]Commit, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var commits []Commit
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}

		commit := Commit{
			Hash:    parts[0],
			Date:    parts[1],
			Message: parts[2],
			Type:    parseCommitType(parts[2]),
		}
		commits = append(commits, commit)
	}

	return commits, scanner.Err()
}

func getKnowledgeArtifacts(hash string, branch string) ([]KnowledgeArtifact, error) {
	var cmd *exec.Cmd
	if branch != "" {
		cmd = exec.Command("git", "show", "--name-only", "--format=", branch+":"+hash)
	} else {
		cmd = exec.Command("git", "show", "--name-only", "--format=", hash)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var artifacts []KnowledgeArtifact
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, ".kb/investigations/") && strings.HasSuffix(line, ".md") {
			artifacts = append(artifacts, KnowledgeArtifact{
				Path: line,
				Type: "investigation",
			})
		} else if strings.HasPrefix(line, ".kb/decisions/") && strings.HasSuffix(line, ".md") {
			artifacts = append(artifacts, KnowledgeArtifact{
				Path: line,
				Type: "decision",
			})
		} else if strings.HasPrefix(line, ".kb/models/") && strings.HasSuffix(line, ".md") {
			artifacts = append(artifacts, KnowledgeArtifact{
				Path: line,
				Type: "model",
			})
		}
	}

	return artifacts, nil
}

func getCommitStats(hash string, branch string) (int, int, error) {
	var cmd *exec.Cmd
	if branch != "" {
		cmd = exec.Command("git", "show", "--numstat", "--format=", hash)
	} else {
		cmd = exec.Command("git", "show", "--numstat", "--format=", hash)
	}

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(output), "\n")
	netLOC := 0
	filesChanged := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Skip binary files
		if parts[0] == "-" || parts[1] == "-" {
			continue
		}

		var added, deleted int
		fmt.Sscanf(parts[0], "%d", &added)
		fmt.Sscanf(parts[1], "%d", &deleted)

		netLOC += added - deleted
		filesChanged++
	}

	return netLOC, filesChanged, nil
}

func buildTimeline(commits []Commit, branch string) (*Timeline, error) {
	timeline := &Timeline{
		Days: make(map[string]*DayStats),
	}

	for i, commit := range commits {
		date := commit.Date

		// Initialize day stats if not exists
		if _, exists := timeline.Days[date]; !exists {
			timeline.Days[date] = &DayStats{
				Date:           date,
				CommitsByType:  make(map[CommitType]int),
				FeatureCommits: []string{},
				FixCommits:     []string{},
			}
		}

		stats := timeline.Days[date]
		stats.TotalCommits++
		stats.CommitsByType[commit.Type]++

		// Track feature and fix commits
		if commit.Type == TypeFeat {
			stats.FeatureCommits = append(stats.FeatureCommits, commit.Message)
		} else if commit.Type == TypeFix {
			stats.FixCommits = append(stats.FixCommits, commit.Message)
		}

		// Get knowledge artifacts (only for first 100 and every 10th commit to avoid slowness)
		if i < 100 || i%10 == 0 {
			artifacts, err := getKnowledgeArtifacts(commit.Hash, branch)
			if err == nil && len(artifacts) > 0 {
				stats.KnowledgeArtifacts = append(stats.KnowledgeArtifacts, artifacts...)
			}
		}

		// Get LOC stats (sample every 5th commit to speed up)
		if i%5 == 0 {
			netLOC, filesChanged, err := getCommitStats(commit.Hash, branch)
			if err == nil {
				stats.NetLOC += netLOC
				stats.FilesChanged += filesChanged
			}
		}

		if i > 0 && i%100 == 0 {
			fmt.Fprintf(os.Stderr, "Processed %d/%d commits...\n", i, len(commits))
		}
	}

	return timeline, nil
}

func main() {
	fmt.Fprintln(os.Stderr, "Parsing commits from Dec 19 - Jan 18...")
	commits1, err := parseCommitFile("/tmp/commits-dec19-jan18.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing commits: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Parsing commits from entropy spiral...")
	commits2, err := parseCommitFile("/tmp/commits-entropy-spiral.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing entropy spiral commits: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Total commits: %d (Dec19-Jan18) + %d (entropy) = %d\n",
		len(commits1), len(commits2), len(commits1)+len(commits2))

	// Build timeline for Dec 19 - Jan 18
	fmt.Fprintln(os.Stderr, "Building timeline for Dec 19 - Jan 18...")
	timeline1, err := buildTimeline(commits1, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building timeline: %v\n", err)
		os.Exit(1)
	}

	// Build timeline for entropy spiral
	fmt.Fprintln(os.Stderr, "Building timeline for entropy spiral...")
	timeline2, err := buildTimeline(commits2, "entropy-spiral-feb2026")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building entropy timeline: %v\n", err)
		os.Exit(1)
	}

	// Merge timelines
	for date, stats := range timeline2.Days {
		timeline1.Days[date] = stats
	}

	// Get date range
	var dates []string
	for date := range timeline1.Days {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	if len(dates) > 0 {
		timeline1.StartDate = dates[0]
		timeline1.EndDate = dates[len(dates)-1]
		timeline1.TotalDays = len(dates)

		// Parse dates for day count
		start, _ := time.Parse("2006-01-02", dates[0])
		end, _ := time.Parse("2006-01-02", dates[len(dates)-1])
		timeline1.TotalDays = int(end.Sub(start).Hours()/24) + 1
	}

	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(timeline1); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
