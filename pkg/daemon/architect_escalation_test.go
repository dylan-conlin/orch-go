package daemon

import (
	"fmt"
	"testing"
)

// --- Unit tests for isImplementationSkill ---

func TestIsImplementationSkill(t *testing.T) {
	tests := []struct {
		skill string
		want  bool
	}{
		{"feature-impl", true},
		{"systematic-debugging", true},
		{"architect", false},
		{"investigation", false},
		{"research", false},
		{"codebase-audit", false},
		{"kb-reflect", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			got := isImplementationSkill(tt.skill)
			if got != tt.want {
				t.Errorf("isImplementationSkill(%q) = %v, want %v", tt.skill, got, tt.want)
			}
		})
	}
}

// --- Unit tests for FindMatchingHotspot ---

func TestFindMatchingHotspot_ExactMatch(t *testing.T) {
	files := []string{"pkg/daemon/daemon.go"}
	hotspots := []HotspotWarning{
		{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
	}

	match := FindMatchingHotspot(files, hotspots)
	if match == nil {
		t.Fatal("FindMatchingHotspot() returned nil, expected match")
	}
	if match.Path != "pkg/daemon/daemon.go" {
		t.Errorf("match.Path = %q, want 'pkg/daemon/daemon.go'", match.Path)
	}
	if match.Type != "fix-density" {
		t.Errorf("match.Type = %q, want 'fix-density'", match.Type)
	}
}

func TestFindMatchingHotspot_PartialMatch(t *testing.T) {
	// Bare filename should match full path hotspot
	files := []string{"daemon.go"}
	hotspots := []HotspotWarning{
		{Path: "pkg/daemon/daemon.go", Type: "investigation-cluster", Score: 5},
	}

	match := FindMatchingHotspot(files, hotspots)
	if match == nil {
		t.Fatal("FindMatchingHotspot() returned nil, expected match on partial filename")
	}
}

func TestFindMatchingHotspot_NoMatch(t *testing.T) {
	files := []string{"pkg/other/other.go"}
	hotspots := []HotspotWarning{
		{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
	}

	match := FindMatchingHotspot(files, hotspots)
	if match != nil {
		t.Errorf("FindMatchingHotspot() returned %v, expected nil", match)
	}
}

func TestFindMatchingHotspot_EmptyInputs(t *testing.T) {
	if match := FindMatchingHotspot(nil, nil); match != nil {
		t.Error("expected nil for nil inputs")
	}
	if match := FindMatchingHotspot([]string{}, nil); match != nil {
		t.Error("expected nil for empty files")
	}
	if match := FindMatchingHotspot([]string{"foo.go"}, nil); match != nil {
		t.Error("expected nil for nil hotspots")
	}
	if match := FindMatchingHotspot(nil, []HotspotWarning{{Path: "foo.go"}}); match != nil {
		t.Error("expected nil for nil files")
	}
}

func TestFindMatchingHotspot_MatchesAnyType(t *testing.T) {
	// Unlike FindCriticalHotspot, this should match ANY hotspot type
	files := []string{"pkg/daemon/daemon.go"}

	types := []string{"fix-density", "investigation-cluster", "bloat-size", "coupling"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			hotspots := []HotspotWarning{
				{Path: "pkg/daemon/daemon.go", Type: typ, Score: 5},
			}
			match := FindMatchingHotspot(files, hotspots)
			if match == nil {
				t.Errorf("FindMatchingHotspot() returned nil for type %q, expected match", typ)
			}
		})
	}
}

// --- Unit tests for CheckArchitectEscalation ---

func TestCheckArchitectEscalation_EscalatesFeatureImpl(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
		},
	}

	issue := &Issue{
		ID:        "proj-1",
		Title:     "Add retry logic to pkg/daemon/daemon.go",
		IssueType: "feature",
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, nil)
	if result == nil {
		t.Fatal("CheckArchitectEscalation() returned nil, expected escalation")
	}
	if result.HotspotFile != "pkg/daemon/daemon.go" {
		t.Errorf("HotspotFile = %q, want 'pkg/daemon/daemon.go'", result.HotspotFile)
	}
	if result.HotspotType != "fix-density" {
		t.Errorf("HotspotType = %q, want 'fix-density'", result.HotspotType)
	}
	if result.HotspotScore != 8 {
		t.Errorf("HotspotScore = %d, want 8", result.HotspotScore)
	}
}

func TestCheckArchitectEscalation_EscalatesSystematicDebugging(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 1200},
		},
	}

	issue := &Issue{
		ID:        "proj-2",
		Title:     "Fix spawn race in cmd/orch/spawn_cmd.go",
		IssueType: "bug",
	}

	result := CheckArchitectEscalation(issue, "systematic-debugging", checker, nil)
	if result == nil {
		t.Fatal("CheckArchitectEscalation() returned nil, expected escalation")
	}
	if result.HotspotFile != "cmd/orch/spawn_cmd.go" {
		t.Errorf("HotspotFile = %q, want 'cmd/orch/spawn_cmd.go'", result.HotspotFile)
	}
}

func TestCheckArchitectEscalation_SkipsExemptSkills(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
		},
	}

	issue := &Issue{
		ID:        "proj-3",
		Title:     "Investigate pkg/daemon/daemon.go architecture",
		IssueType: "investigation",
	}

	exemptSkills := []string{"architect", "investigation", "research", "codebase-audit", "kb-reflect"}
	for _, skill := range exemptSkills {
		t.Run(skill, func(t *testing.T) {
			result := CheckArchitectEscalation(issue, skill, checker, nil)
			if result != nil {
				t.Errorf("CheckArchitectEscalation() for skill %q returned non-nil, expected nil (exempt)", skill)
			}
		})
	}
}

func TestCheckArchitectEscalation_SkipsExtractionIssues(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
		},
	}

	issue := &Issue{
		ID:        "proj-ext1",
		Title:     "Extract spawn logic from cmd/orch/spawn_cmd.go into pkg/spawn/",
		IssueType: "task",
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, nil)
	if result != nil {
		t.Error("CheckArchitectEscalation() should skip extraction issues (title starts with 'Extract ')")
	}
}

func TestCheckArchitectEscalation_SkipsExplicitSkillLabel(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
		},
	}

	issue := &Issue{
		ID:        "proj-4",
		Title:     "Add feature to pkg/daemon/daemon.go",
		IssueType: "feature",
		Labels:    []string{"skill:feature-impl"},
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, nil)
	if result != nil {
		t.Error("CheckArchitectEscalation() should skip when issue has explicit skill:* label")
	}
}

func TestCheckArchitectEscalation_NoTargetFiles(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
		},
	}

	issue := &Issue{
		ID:        "proj-5",
		Title:     "Add retry logic to the daemon",
		IssueType: "feature",
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, nil)
	if result != nil {
		t.Error("CheckArchitectEscalation() should return nil when no target files can be inferred")
	}
}

func TestCheckArchitectEscalation_NoHotspots(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{},
	}

	issue := &Issue{
		ID:        "proj-6",
		Title:     "Add feature to pkg/daemon/daemon.go",
		IssueType: "feature",
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, nil)
	if result != nil {
		t.Error("CheckArchitectEscalation() should return nil when no hotspots exist")
	}
}

func TestCheckArchitectEscalation_NoMatchingHotspot(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "pkg/other/other.go", Type: "fix-density", Score: 8},
		},
	}

	issue := &Issue{
		ID:        "proj-7",
		Title:     "Add feature to pkg/daemon/daemon.go",
		IssueType: "feature",
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, nil)
	if result != nil {
		t.Error("CheckArchitectEscalation() should return nil when no inferred files match hotspots")
	}
}

func TestCheckArchitectEscalation_NilIssue(t *testing.T) {
	checker := &mockEscalationChecker{}
	result := CheckArchitectEscalation(nil, "feature-impl", checker, nil)
	if result != nil {
		t.Error("CheckArchitectEscalation() should return nil for nil issue")
	}
}

func TestCheckArchitectEscalation_NilChecker(t *testing.T) {
	issue := &Issue{ID: "proj-8", Title: "test", IssueType: "feature"}
	result := CheckArchitectEscalation(issue, "feature-impl", nil, nil)
	if result != nil {
		t.Error("CheckArchitectEscalation() should return nil for nil checker")
	}
}

func TestCheckArchitectEscalation_CheckerError(t *testing.T) {
	checker := &mockEscalationChecker{
		err: fmt.Errorf("hotspot check failed"),
	}

	issue := &Issue{
		ID:        "proj-9",
		Title:     "Add feature to pkg/daemon/daemon.go",
		IssueType: "feature",
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, nil)
	if result != nil {
		t.Error("CheckArchitectEscalation() should return nil on checker error (graceful degradation)")
	}
}

// --- Integration test: daemon Once() with architect escalation ---

func TestOnceExcluding_ArchitectEscalation_EscalatesFeatureImpl(t *testing.T) {
	// When a feature-impl issue targets a hotspot area (not CRITICAL >1500),
	// the daemon should escalate the skill to architect.
	var spawnedID, spawnedModel string
	d := &Daemon{
		Config: Config{Verbose: true},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add retry logic to pkg/daemon/daemon.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				spawnedID = beadsID
				spawnedModel = model
				return nil
			},
		},
		HotspotChecker: &mockEscalationChecker{
			hotspots: []HotspotWarning{
				{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned the original issue (not an extraction issue)
	if spawnedID != "proj-1" {
		t.Errorf("spawnFunc called with %q, want 'proj-1'", spawnedID)
	}

	// Skill should be escalated to architect
	if result.Skill != "architect" {
		t.Errorf("result.Skill = %q, want 'architect' (escalated from feature-impl)", result.Skill)
	}

	// Model should be opus (architect skill)
	if spawnedModel != "opus" {
		t.Errorf("spawnFunc model = %q, want 'opus' (architect model)", spawnedModel)
	}
	if result.Model != "opus" {
		t.Errorf("result.Model = %q, want 'opus'", result.Model)
	}

	// Should be marked as escalated
	if !result.ArchitectEscalated {
		t.Error("result.ArchitectEscalated should be true")
	}

	// Should NOT be an extraction spawn
	if result.ExtractionSpawned {
		t.Error("result.ExtractionSpawned should be false (escalation, not extraction)")
	}
}

func TestOnceExcluding_ArchitectEscalation_ExtractionTakesPrecedence(t *testing.T) {
	// When a file is CRITICAL (>1500 lines, bloat-size), extraction should happen
	// instead of architect escalation. Extraction handles the most severe case.
	var spawnedID string
	d := &Daemon{
		Config: Config{Verbose: true},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to cmd/orch/spawn_cmd.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
			CreateExtractionIssueFunc: func(task, parentID string) (string, error) {
				return "proj-ext1", nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				spawnedID = beadsID
				return nil
			},
		},
		HotspotChecker: &mockEscalationChecker{
			hotspots: []HotspotWarning{
				// CRITICAL: >1500 lines bloat-size triggers extraction
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Extraction should have taken precedence
	if !result.ExtractionSpawned {
		t.Error("ExtractionSpawned should be true (extraction takes precedence over escalation)")
	}
	if spawnedID != "proj-ext1" {
		t.Errorf("spawnFunc called with %q, want 'proj-ext1' (extraction issue)", spawnedID)
	}

	// Should NOT be marked as architect escalated
	if result.ArchitectEscalated {
		t.Error("result.ArchitectEscalated should be false when extraction happens")
	}
}

func TestOnceExcluding_ArchitectEscalation_SkipsNonHotspotIssues(t *testing.T) {
	// Issues targeting non-hotspot files should proceed normally as feature-impl.
	var spawnedModel string
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to pkg/clean/clean.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) { return "open", nil },
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				spawnedModel = model
				return nil
			},
		},
		HotspotChecker: &mockEscalationChecker{
			hotspots: []HotspotWarning{
				{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should proceed as feature-impl (not escalated)
	if result.Skill != "feature-impl" {
		t.Errorf("result.Skill = %q, want 'feature-impl' (no escalation)", result.Skill)
	}
	if spawnedModel != "" {
		t.Errorf("spawnFunc model = %q, want empty string (resolve pipeline handles default for feature-impl)", spawnedModel)
	}
	if result.ArchitectEscalated {
		t.Error("result.ArchitectEscalated should be false (no hotspot match)")
	}
}

func TestOnceExcluding_ArchitectEscalation_SkipsWithExplicitSkillLabel(t *testing.T) {
	// Issues with explicit skill:feature-impl label should NOT be escalated.
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to pkg/daemon/daemon.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
						Labels:    []string{"skill:feature-impl", "triage:ready"},
					},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) { return "open", nil },
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				return nil
			},
		},
		HotspotChecker: &mockEscalationChecker{
			hotspots: []HotspotWarning{
				{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Explicit skill label should prevent escalation
	if result.Skill != "feature-impl" {
		t.Errorf("result.Skill = %q, want 'feature-impl' (explicit label prevents escalation)", result.Skill)
	}
	if result.ArchitectEscalated {
		t.Error("result.ArchitectEscalated should be false (explicit skill label)")
	}
}

// --- FindMatchingHotspot additional edge cases ---

func TestFindMatchingHotspot_ReturnsFirstMatch(t *testing.T) {
	// When multiple files match different hotspots, the first match should be returned.
	files := []string{"pkg/daemon/daemon.go", "cmd/orch/spawn_cmd.go"}
	hotspots := []HotspotWarning{
		{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
		{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 12},
	}

	match := FindMatchingHotspot(files, hotspots)
	if match == nil {
		t.Fatal("FindMatchingHotspot() returned nil, expected match")
	}
	// First file in the list matches, so it should return that hotspot
	if match.Path != "pkg/daemon/daemon.go" {
		t.Errorf("expected first match 'pkg/daemon/daemon.go', got %q", match.Path)
	}
	if match.Score != 8 {
		t.Errorf("expected score 8, got %d", match.Score)
	}
}

func TestFindMatchingHotspot_MultipleHotspotsOnSameFile(t *testing.T) {
	// A file could theoretically appear as multiple hotspot types.
	// The first matching hotspot entry should be returned.
	files := []string{"pkg/daemon/daemon.go"}
	hotspots := []HotspotWarning{
		{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
		{Path: "pkg/daemon/daemon.go", Type: "investigation-cluster", Score: 5},
	}

	match := FindMatchingHotspot(files, hotspots)
	if match == nil {
		t.Fatal("FindMatchingHotspot() returned nil, expected match")
	}
	if match.Type != "fix-density" {
		t.Errorf("expected first hotspot type 'fix-density', got %q", match.Type)
	}
}

func TestFindMatchingHotspot_SecondFileMatchesFirstHotspot(t *testing.T) {
	// First file doesn't match any hotspot, second file does.
	files := []string{"pkg/clean/clean.go", "pkg/daemon/daemon.go"}
	hotspots := []HotspotWarning{
		{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
	}

	match := FindMatchingHotspot(files, hotspots)
	if match == nil {
		t.Fatal("FindMatchingHotspot() returned nil, expected match on second file")
	}
	if match.Path != "pkg/daemon/daemon.go" {
		t.Errorf("expected match on 'pkg/daemon/daemon.go', got %q", match.Path)
	}
}

// --- CheckArchitectEscalation: escalation fields ---

func TestCheckArchitectEscalation_EscalationFieldsPopulated(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 1200},
		},
	}

	issue := &Issue{
		ID:        "proj-10",
		Title:     "Refactor cmd/orch/spawn_cmd.go for readability",
		IssueType: "task",
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, nil)
	if result == nil {
		t.Fatal("expected escalation result")
	}
	if result.HotspotFile != "cmd/orch/spawn_cmd.go" {
		t.Errorf("HotspotFile = %q, want 'cmd/orch/spawn_cmd.go'", result.HotspotFile)
	}
	if result.HotspotType != "bloat-size" {
		t.Errorf("HotspotType = %q, want 'bloat-size'", result.HotspotType)
	}
	if result.HotspotScore != 1200 {
		t.Errorf("HotspotScore = %d, want 1200", result.HotspotScore)
	}
}

// --- Prior architect finder tests ---

func TestCheckArchitectEscalation_SkipsWhenPriorArchitectExists(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
		},
	}

	issue := &Issue{
		ID:        "proj-prior-1",
		Title:     "Add retry logic to pkg/daemon/daemon.go",
		IssueType: "feature",
	}

	// Prior architect finder returns a matching closed architect issue
	finder := func(files []string) (string, error) {
		for _, f := range files {
			if f == "pkg/daemon/daemon.go" {
				return "orch-go-1119", nil
			}
		}
		return "", nil
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, finder)
	if result == nil {
		t.Fatal("CheckArchitectEscalation() should return non-nil decision when hotspot matches")
	}
	if result.Escalated {
		t.Error("result.Escalated should be false when prior architect review exists")
	}
	if result.PriorArchitectRef != "orch-go-1119" {
		t.Errorf("result.PriorArchitectRef = %q, want %q", result.PriorArchitectRef, "orch-go-1119")
	}
}

func TestCheckArchitectEscalation_EscalatesWhenFinderReturnsEmpty(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
		},
	}

	issue := &Issue{
		ID:        "proj-prior-2",
		Title:     "Add retry logic to pkg/daemon/daemon.go",
		IssueType: "feature",
	}

	// Prior architect finder returns no match
	finder := func(files []string) (string, error) {
		return "", nil
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, finder)
	if result == nil {
		t.Fatal("CheckArchitectEscalation() should escalate when no prior architect found")
	}
}

func TestCheckArchitectEscalation_EscalatesWhenFinderErrors(t *testing.T) {
	checker := &mockEscalationChecker{
		hotspots: []HotspotWarning{
			{Path: "pkg/daemon/daemon.go", Type: "fix-density", Score: 8},
		},
	}

	issue := &Issue{
		ID:        "proj-prior-3",
		Title:     "Add retry logic to pkg/daemon/daemon.go",
		IssueType: "feature",
	}

	// Prior architect finder returns an error (graceful degradation)
	finder := func(files []string) (string, error) {
		return "", fmt.Errorf("beads query failed")
	}

	result := CheckArchitectEscalation(issue, "feature-impl", checker, finder)
	if result == nil {
		t.Fatal("CheckArchitectEscalation() should escalate when finder errors (graceful degradation)")
	}
}

// --- Mock implementation ---

// mockEscalationChecker implements HotspotChecker for architect escalation tests.
type mockEscalationChecker struct {
	hotspots []HotspotWarning
	err      error
}

func (m *mockEscalationChecker) CheckHotspots(projectDir string) ([]HotspotWarning, error) {
	return m.hotspots, m.err
}
