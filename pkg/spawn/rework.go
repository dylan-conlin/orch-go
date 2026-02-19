// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

var reworkCommentRegex = regexp.MustCompile(`(?i)^rework\s+#(\d+)`)

// FindArchivedWorkspaceByBeadsID scans the archived workspace directory for a matching beads ID.
// Returns the most recent matching archived workspace path.
func FindArchivedWorkspaceByBeadsID(projectDir, beadsID string) (string, error) {
	if beadsID == "" {
		return "", fmt.Errorf("beads ID is required")
	}

	archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")
	entries, err := os.ReadDir(archivedDir)
	if err != nil {
		return "", fmt.Errorf("failed to read archived workspaces: %w", err)
	}

	type candidate struct {
		path string
		time time.Time
	}

	var matches []candidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wsPath := filepath.Join(archivedDir, entry.Name())
		manifest := ReadAgentManifestWithFallback(wsPath)
		if manifest.BeadsID != beadsID {
			continue
		}

		ts := manifest.ParseSpawnTime()
		if ts.IsZero() {
			if info, err := entry.Info(); err == nil {
				ts = info.ModTime()
			}
		}
		matches = append(matches, candidate{path: wsPath, time: ts})
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no archived workspace found for %s", beadsID)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].time.After(matches[j].time)
	})

	return matches[0].path, nil
}

// ExtractReworkSummary extracts TLDR + Delta sections from a prior SYNTHESIS.md.
// Returns formatted markdown suitable for embedding in SPAWN_CONTEXT.md.
func ExtractReworkSummary(synthesisPath string) (string, error) {
	content, err := os.ReadFile(synthesisPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	tldr := extractSection(lines, "tldr")
	delta := extractSection(lines, "delta")

	var sb strings.Builder
	if strings.TrimSpace(tldr) != "" {
		sb.WriteString("#### TLDR\n\n")
		sb.WriteString(strings.TrimSpace(tldr))
		sb.WriteString("\n\n")
	}
	if strings.TrimSpace(delta) != "" {
		sb.WriteString("#### Delta\n\n")
		sb.WriteString(strings.TrimSpace(delta))
		sb.WriteString("\n")
	}

	summary := strings.TrimSpace(sb.String())
	if summary == "" {
		return "", fmt.Errorf("no TLDR or Delta sections found")
	}
	return summary, nil
}

// CountReworks returns the highest rework number found in beads comments.
// If no rework comments are found, returns 0.
func CountReworks(beadsID, projectDir string) (int, error) {
	comments, err := fetchBeadsComments(beadsID, projectDir)
	if err != nil {
		return 0, err
	}

	max := 0
	for _, comment := range comments {
		text := strings.TrimSpace(comment.Text)
		matches := reworkCommentRegex.FindStringSubmatch(text)
		if len(matches) < 2 {
			continue
		}
		n, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}

	return max, nil
}

func fetchBeadsComments(beadsID, projectDir string) ([]beads.Comment, error) {
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
		}
		client := beads.NewClient(socketPath, opts...)
		if err := client.Connect(); err == nil {
			defer client.Close()
			if comments, err := client.Comments(beadsID); err == nil {
				return comments, nil
			}
		}
	}

	cli := beads.NewCLIClient(beads.WithWorkDir(projectDir))
	return cli.Comments(beadsID)
}

func extractSection(lines []string, heading string) string {
	target := "## " + heading
	start := -1
	for i, line := range lines {
		normalized := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(normalized, target) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return ""
	}

	var collected []string
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		normalized := strings.ToLower(trimmed)
		if strings.HasPrefix(normalized, "## ") {
			break
		}
		if strings.HasPrefix(trimmed, "---") {
			break
		}
		collected = append(collected, lines[i])
	}

	return strings.TrimSpace(strings.Join(collected, "\n"))
}
