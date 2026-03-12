package artifactsync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest represents the ARTIFACT_MANIFEST.yaml structure.
type Manifest struct {
	Artifacts []ArtifactEntry `yaml:"artifacts"`
}

// ArtifactEntry maps an artifact file to its coverage and triggers.
type ArtifactEntry struct {
	Path     string           `yaml:"path"`
	Covers   []string         `yaml:"covers,omitempty"`
	Triggers []string         `yaml:"triggers,omitempty"`
	Sections []ArtifactSection `yaml:"sections,omitempty"`
}

// ArtifactSection describes a named section within an artifact (e.g., CLAUDE.md:Commands).
type ArtifactSection struct {
	Name     string   `yaml:"name"`
	Covers   []string `yaml:"covers"`
	Triggers []string `yaml:"triggers"`
}

// LoadManifest reads and parses ARTIFACT_MANIFEST.yaml from the given project directory.
func LoadManifest(projectDir string) (*Manifest, error) {
	path := filepath.Join(projectDir, "ARTIFACT_MANIFEST.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &m, nil
}

// DriftReport describes which artifacts are affected by drift events.
type DriftReport struct {
	Entries []DriftReportEntry
}

// DriftReportEntry is a single affected artifact or section.
type DriftReportEntry struct {
	ArtifactPath string
	SectionName  string   // empty if artifact-level (no sections)
	Triggers     []string // which change-scope categories matched
	Events       []DriftEvent
}

// AnalyzeDrift cross-references drift events against the manifest to produce a report.
func AnalyzeDrift(manifest *Manifest, events []DriftEvent) *DriftReport {
	report := &DriftReport{}

	// Build set of all triggered scopes across all events
	allScopes := make(map[string]bool)
	for _, ev := range events {
		for _, scope := range ev.ChangeScopes {
			allScopes[scope] = true
		}
	}

	for _, artifact := range manifest.Artifacts {
		if len(artifact.Sections) > 0 {
			// Section-level matching
			for _, section := range artifact.Sections {
				matched := matchTriggers(section.Triggers, allScopes)
				if len(matched) > 0 {
					report.Entries = append(report.Entries, DriftReportEntry{
						ArtifactPath: artifact.Path,
						SectionName:  section.Name,
						Triggers:     matched,
						Events:       filterEventsByScopes(events, section.Triggers),
					})
				}
			}
		} else {
			// Artifact-level matching
			matched := matchTriggers(artifact.Triggers, allScopes)
			if len(matched) > 0 {
				report.Entries = append(report.Entries, DriftReportEntry{
					ArtifactPath: artifact.Path,
					Triggers:     matched,
					Events:       filterEventsByScopes(events, artifact.Triggers),
				})
			}
		}
	}

	return report
}

// ReadDriftEvents reads and parses drift events from the JSONL log file.
func ReadDriftEvents(path string) ([]DriftEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read drift log: %w", err)
	}

	var events []DriftEvent
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var envelope driftEventEnvelope
		if err := unmarshalJSON([]byte(line), &envelope); err != nil {
			continue // skip malformed lines
		}
		events = append(events, envelope.Data)
	}

	return events, nil
}

func matchTriggers(triggers []string, scopes map[string]bool) []string {
	var matched []string
	for _, t := range triggers {
		if scopes[t] {
			matched = append(matched, t)
		}
	}
	return matched
}

func filterEventsByScopes(events []DriftEvent, triggers []string) []DriftEvent {
	triggerSet := make(map[string]bool)
	for _, t := range triggers {
		triggerSet[t] = true
	}

	var filtered []DriftEvent
	for _, ev := range events {
		for _, scope := range ev.ChangeScopes {
			if triggerSet[scope] {
				filtered = append(filtered, ev)
				break
			}
		}
	}
	return filtered
}

func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
