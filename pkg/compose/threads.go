package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ThreadInfo represents a parsed thread with its title and keywords.
type ThreadInfo struct {
	Slug     string // filename-based slug
	Title    string // from YAML frontmatter or first heading
	FilePath string
	Keywords []string
}

// ThreadMatch represents a potential connection between a cluster and a thread.
type ThreadMatch struct {
	Thread         *ThreadInfo
	SharedKeywords []string
	Score          int // number of shared keywords
}

var threadTitleRe = regexp.MustCompile(`title:\s*"([^"]+)"`)
var threadHeadingRe = regexp.MustCompile(`(?m)^#\s+(.+)$`)

// LoadThreads reads all thread files from the threads directory.
func LoadThreads(threadsDir string) ([]*ThreadInfo, error) {
	entries, err := os.ReadDir(threadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading threads directory: %w", err)
	}

	var threads []*ThreadInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		path := filepath.Join(threadsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)
		t := &ThreadInfo{
			Slug:     strings.TrimSuffix(e.Name(), ".md"),
			FilePath: path,
		}

		// Extract title from YAML frontmatter
		if m := threadTitleRe.FindStringSubmatch(content); len(m) >= 2 {
			t.Title = m[1]
		} else if m := threadHeadingRe.FindStringSubmatch(content); len(m) >= 2 {
			// Fall back to first heading
			t.Title = m[1]
		}

		if t.Title == "" {
			t.Title = t.Slug
		}

		t.Keywords = ExtractKeywords(t.Title + " " + content)
		threads = append(threads, t)
	}

	return threads, nil
}

// MatchClusterToThreads finds threads related to a cluster's content.
// Returns matches sorted by score (highest first), limited to top 3.
func MatchClusterToThreads(cluster *Cluster, threads []*ThreadInfo) []ThreadMatch {
	// Combine all keywords from cluster members
	clusterKeywords := make(map[string]bool)
	for _, kw := range cluster.SharedKeywords {
		clusterKeywords[kw] = true
	}
	// Also include keywords from individual briefs for broader matching
	for _, b := range cluster.Briefs {
		for _, kw := range b.Keywords {
			clusterKeywords[kw] = true
		}
	}

	clusterKWList := make([]string, 0, len(clusterKeywords))
	for kw := range clusterKeywords {
		clusterKWList = append(clusterKWList, kw)
	}

	var matches []ThreadMatch
	for _, t := range threads {
		shared := KeywordOverlap(clusterKWList, t.Keywords)
		if len(shared) >= MinKeywordOverlap {
			matches = append(matches, ThreadMatch{
				Thread:         t,
				SharedKeywords: shared,
				Score:          len(shared),
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit to top 3 matches
	if len(matches) > 3 {
		matches = matches[:3]
	}

	return matches
}
