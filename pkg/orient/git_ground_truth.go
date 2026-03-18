package orient

import (
	"regexp"
	"strconv"
	"strings"
)

// GitCommitInfo holds a parsed git commit with its beads ID(s).
type GitCommitInfo struct {
	Hash     string
	Subject  string
	BeadsIDs []string
}

// ExtractBeadsIDs extracts beads IDs matching the given project prefix from a commit message.
// Pattern: prefix-XXXXX (e.g., "orch-go-abc12").
func ExtractBeadsIDs(message, projectPrefix string) []string {
	pattern := regexp.MustCompile(regexp.QuoteMeta(projectPrefix) + `-[a-z0-9]{3,10}`)
	matches := pattern.FindAllString(message, -1)
	if len(matches) == 0 {
		return nil
	}
	return matches
}

// ParseGitLogForGroundTruth parses `git log --format='%h %s'` output and returns
// commits that contain beads IDs for the given project.
func ParseGitLogForGroundTruth(gitLog, projectPrefix string) []GitCommitInfo {
	var commits []GitCommitInfo
	for _, line := range strings.Split(strings.TrimSpace(gitLog), "\n") {
		if line == "" {
			continue
		}
		// Split: hash + subject
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		hash := parts[0]
		subject := parts[1]

		ids := ExtractBeadsIDs(subject, projectPrefix)
		if len(ids) > 0 {
			commits = append(commits, GitCommitInfo{
				Hash:     hash,
				Subject:  subject,
				BeadsIDs: ids,
			})
		}
	}
	return commits
}

// UniqueBeadsIDs returns deduplicated beads IDs from a set of commits.
func UniqueBeadsIDs(commits []GitCommitInfo) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, c := range commits {
		for _, id := range c.BeadsIDs {
			if !seen[id] {
				seen[id] = true
				unique = append(unique, id)
			}
		}
	}
	return unique
}

// ParseGitNumstat parses `git log --numstat` output and returns total lines added and deleted.
// Each line is: <added>\t<deleted>\t<file>
// Binary files show as "-\t-\t<file>" and are skipped.
func ParseGitNumstat(numstat string) (added, deleted int) {
	for _, line := range strings.Split(strings.TrimSpace(numstat), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Skip binary files
		if fields[0] == "-" || fields[1] == "-" {
			continue
		}
		a, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		d, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		added += a
		deleted += d
	}
	return
}
