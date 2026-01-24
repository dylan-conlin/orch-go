// Package kb provides access to knowledge base artifacts (investigations, decisions).
package kb

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ArtifactType represents the type of kb artifact.
type ArtifactType string

const (
	ArtifactInvestigation ArtifactType = "investigation"
	ArtifactDecision      ArtifactType = "decision"
)

// Artifact represents a kb artifact (investigation or decision).
type Artifact struct {
	ID        string       // filename without extension
	Type      ArtifactType // investigation or decision
	Title     string       // extracted from # heading
	Date      string       // YYYY-MM-DD from filename or metadata
	Status    string       // Phase/Status from metadata
	Path      string       // full path to file
	References []string    // beads IDs mentioned in file
}

// beadsIDPattern matches beads issue IDs like "orch-go-abc123"
var beadsIDPattern = regexp.MustCompile(`\b(orch-go-[a-z0-9]+)\b`)

// ListArtifacts returns all kb artifacts from the given kb directory.
func ListArtifacts(kbDir string) ([]Artifact, error) {
	var artifacts []Artifact

	// Scan investigations
	invDir := filepath.Join(kbDir, "investigations")
	invArtifacts, err := scanDirectory(invDir, ArtifactInvestigation)
	if err == nil {
		artifacts = append(artifacts, invArtifacts...)
	}

	// Scan decisions
	decDir := filepath.Join(kbDir, "decisions")
	decArtifacts, err := scanDirectory(decDir, ArtifactDecision)
	if err == nil {
		artifacts = append(artifacts, decArtifacts...)
	}

	return artifacts, nil
}

// ListRecentArtifacts returns artifacts from the last N days.
func ListRecentArtifacts(kbDir string, days int) ([]Artifact, error) {
	all, err := ListArtifacts(kbDir)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var recent []Artifact

	for _, a := range all {
		// Parse date from artifact
		if a.Date != "" {
			t, err := time.Parse("2006-01-02", a.Date)
			if err == nil && t.After(cutoff) {
				recent = append(recent, a)
			}
		}
	}

	return recent, nil
}

func scanDirectory(dir string, artifactType ArtifactType) ([]Artifact, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var artifacts []Artifact
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		artifact, err := parseArtifact(path, artifactType)
		if err != nil {
			continue // skip unparseable files
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func parseArtifact(path string, artifactType ArtifactType) (Artifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()

	filename := filepath.Base(path)
	id := strings.TrimSuffix(filename, ".md")

	artifact := Artifact{
		ID:   id,
		Type: artifactType,
		Path: path,
	}

	// Extract date from filename (YYYY-MM-DD prefix)
	if len(id) >= 10 {
		artifact.Date = id[:10]
	}

	// Scan file for metadata and references
	scanner := bufio.NewScanner(file)
	lineCount := 0
	var references []string
	seenRefs := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Only scan first 100 lines for metadata, but all lines for references
		if lineCount <= 100 {
			// Extract title from first # heading
			if artifact.Title == "" && strings.HasPrefix(line, "# ") {
				artifact.Title = strings.TrimPrefix(line, "# ")
				// Clean up common prefixes
				artifact.Title = strings.TrimPrefix(artifact.Title, "Investigation: ")
				artifact.Title = strings.TrimPrefix(artifact.Title, "Decision: ")
			}

			// Extract status/phase
			if strings.HasPrefix(line, "**Phase:**") || strings.HasPrefix(line, "**Status:**") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					artifact.Status = strings.TrimSpace(strings.Trim(parts[1], "*"))
				}
			}
		}

		// Find beads ID references throughout the file
		matches := beadsIDPattern.FindAllString(line, -1)
		for _, match := range matches {
			if !seenRefs[match] {
				seenRefs[match] = true
				references = append(references, match)
			}
		}
	}

	artifact.References = references
	return artifact, nil
}
