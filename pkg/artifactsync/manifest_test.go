package artifactsync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	manifestContent := `artifacts:
  - path: CLAUDE.md
    sections:
      - name: Commands
        covers: ["cmd/orch/*_cmd.go"]
        triggers: [new-command, new-flag]
      - name: Event Types
        covers: ["pkg/events/"]
        triggers: [new-event]
  - path: .kb/guides/spawn.md
    covers: ["pkg/spawn/"]
    triggers: [new-flag, config-change]
`
	if err := os.WriteFile(filepath.Join(dir, "ARTIFACT_MANIFEST.yaml"), []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if len(m.Artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(m.Artifacts))
	}

	if m.Artifacts[0].Path != "CLAUDE.md" {
		t.Errorf("expected CLAUDE.md, got %q", m.Artifacts[0].Path)
	}
	if len(m.Artifacts[0].Sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(m.Artifacts[0].Sections))
	}
	if m.Artifacts[0].Sections[0].Name != "Commands" {
		t.Errorf("expected Commands section, got %q", m.Artifacts[0].Sections[0].Name)
	}

	// Second artifact has no sections, just covers/triggers
	if m.Artifacts[1].Path != ".kb/guides/spawn.md" {
		t.Errorf("expected .kb/guides/spawn.md, got %q", m.Artifacts[1].Path)
	}
	if len(m.Artifacts[1].Triggers) != 2 {
		t.Errorf("expected 2 triggers, got %d", len(m.Artifacts[1].Triggers))
	}
}

func TestLoadManifest_NotFound(t *testing.T) {
	_, err := LoadManifest(t.TempDir())
	if err == nil {
		t.Error("expected error for missing manifest")
	}
}

func TestAnalyzeDrift_SectionLevel(t *testing.T) {
	manifest := &Manifest{
		Artifacts: []ArtifactEntry{
			{
				Path: "CLAUDE.md",
				Sections: []ArtifactSection{
					{Name: "Commands", Triggers: []string{"new-command", "new-flag"}},
					{Name: "Event Types", Triggers: []string{"new-event"}},
				},
			},
		},
	}

	events := []DriftEvent{
		{BeadsID: "proj-1", ChangeScopes: []string{"new-command"}},
		{BeadsID: "proj-2", ChangeScopes: []string{"new-event"}},
	}

	report := AnalyzeDrift(manifest, events)

	if len(report.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(report.Entries))
	}

	// Commands section matched by new-command
	if report.Entries[0].SectionName != "Commands" {
		t.Errorf("expected Commands, got %q", report.Entries[0].SectionName)
	}
	if len(report.Entries[0].Triggers) != 1 || report.Entries[0].Triggers[0] != "new-command" {
		t.Errorf("expected [new-command], got %v", report.Entries[0].Triggers)
	}
	if len(report.Entries[0].Events) != 1 {
		t.Errorf("expected 1 event for Commands, got %d", len(report.Entries[0].Events))
	}

	// Event Types section matched by new-event
	if report.Entries[1].SectionName != "Event Types" {
		t.Errorf("expected Event Types, got %q", report.Entries[1].SectionName)
	}
}

func TestAnalyzeDrift_ArtifactLevel(t *testing.T) {
	manifest := &Manifest{
		Artifacts: []ArtifactEntry{
			{
				Path:     ".kb/guides/spawn.md",
				Triggers: []string{"new-flag", "config-change"},
			},
		},
	}

	events := []DriftEvent{
		{BeadsID: "proj-1", ChangeScopes: []string{"new-flag"}},
	}

	report := AnalyzeDrift(manifest, events)

	if len(report.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(report.Entries))
	}
	if report.Entries[0].ArtifactPath != ".kb/guides/spawn.md" {
		t.Errorf("expected .kb/guides/spawn.md, got %q", report.Entries[0].ArtifactPath)
	}
	if report.Entries[0].SectionName != "" {
		t.Errorf("expected empty section name, got %q", report.Entries[0].SectionName)
	}
}

func TestAnalyzeDrift_NoMatch(t *testing.T) {
	manifest := &Manifest{
		Artifacts: []ArtifactEntry{
			{Path: "CLAUDE.md", Sections: []ArtifactSection{
				{Name: "Commands", Triggers: []string{"new-command"}},
			}},
		},
	}

	events := []DriftEvent{
		{BeadsID: "proj-1", ChangeScopes: []string{"new-event"}},
	}

	report := AnalyzeDrift(manifest, events)

	if len(report.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(report.Entries))
	}
}

func TestAnalyzeDrift_MultipleEventsPerSection(t *testing.T) {
	manifest := &Manifest{
		Artifacts: []ArtifactEntry{
			{Path: "CLAUDE.md", Sections: []ArtifactSection{
				{Name: "Commands", Triggers: []string{"new-command", "new-flag"}},
			}},
		},
	}

	events := []DriftEvent{
		{BeadsID: "proj-1", ChangeScopes: []string{"new-command"}},
		{BeadsID: "proj-2", ChangeScopes: []string{"new-flag"}},
		{BeadsID: "proj-3", ChangeScopes: []string{"api-change"}}, // should not match
	}

	report := AnalyzeDrift(manifest, events)

	if len(report.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(report.Entries))
	}
	if len(report.Entries[0].Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(report.Entries[0].Events))
	}
}

func TestReadDriftEvents(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "drift.jsonl")

	// Write some events
	for _, ev := range []DriftEvent{
		{BeadsID: "proj-1", ChangeScopes: []string{"new-command"}, FilesChanged: []string{"cmd/orch/sync_cmd.go"}},
		{BeadsID: "proj-2", ChangeScopes: []string{"new-event"}, FilesChanged: []string{"pkg/events/logger.go"}},
	} {
		if err := LogDriftEvent(logPath, ev); err != nil {
			t.Fatal(err)
		}
	}

	events, err := ReadDriftEvents(logPath)
	if err != nil {
		t.Fatalf("ReadDriftEvents failed: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].BeadsID != "proj-1" {
		t.Errorf("expected proj-1, got %q", events[0].BeadsID)
	}
	if events[1].BeadsID != "proj-2" {
		t.Errorf("expected proj-2, got %q", events[1].BeadsID)
	}
}

func TestReadDriftEvents_NotFound(t *testing.T) {
	events, err := ReadDriftEvents(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestReadDriftEvents_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "drift.jsonl")

	content := `{"type":"artifact.drift","timestamp":1234,"data":{"beads_id":"proj-1","change_scopes":["new-command"]}}
not valid json
{"type":"artifact.drift","timestamp":1235,"data":{"beads_id":"proj-2","change_scopes":["new-flag"]}}
`
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := ReadDriftEvents(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 valid events (skipping malformed), got %d", len(events))
	}
}
