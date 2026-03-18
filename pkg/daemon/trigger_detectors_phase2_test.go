package daemon

import (
	"fmt"
	"testing"
	"time"
)

// --- Model Contradictions Detector Tests ---

type mockModelContradictionsSource struct {
	listFunc func() ([]UnresolvedContradiction, error)
}

func (m *mockModelContradictionsSource) ListUnresolvedContradictions() ([]UnresolvedContradiction, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return nil, nil
}

func TestModelContradictionsDetector_NoSource(t *testing.T) {
	d := &ModelContradictionsDetector{}
	_, err := d.Detect()
	if err == nil {
		t.Error("expected error for nil source")
	}
}

func TestModelContradictionsDetector_NoContradictions(t *testing.T) {
	d := &ModelContradictionsDetector{
		Source: &mockModelContradictionsSource{
			listFunc: func() ([]UnresolvedContradiction, error) { return nil, nil },
		},
	}
	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("got %d suggestions, want 0", len(suggestions))
	}
}

func TestModelContradictionsDetector_FindsContradictions(t *testing.T) {
	d := &ModelContradictionsDetector{
		Source: &mockModelContradictionsSource{
			listFunc: func() ([]UnresolvedContradiction, error) {
				return []UnresolvedContradiction{
					{
						ModelSlug:     "daemon-autonomous-operation",
						ProbeFilename: "2026-03-10-contradicts-spawn-rate.md",
						ProbeDate:     time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
						ModelUpdated:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
					},
				}, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	s := suggestions[0]
	if s.Detector != "model_contradictions" {
		t.Errorf("Detector = %q", s.Detector)
	}
	if s.Key != "daemon-autonomous-operation:2026-03-10-contradicts-spawn-rate.md" {
		t.Errorf("Key = %q", s.Key)
	}
	if s.IssueType != "task" {
		t.Errorf("IssueType = %q, want task", s.IssueType)
	}
	if s.Priority != 2 {
		t.Errorf("Priority = %d, want 2", s.Priority)
	}
}

func TestModelContradictionsDetector_Name(t *testing.T) {
	d := &ModelContradictionsDetector{}
	if d.Name() != "model_contradictions" {
		t.Errorf("Name() = %q", d.Name())
	}
}

// --- Hotspot Acceleration Detector Tests ---

type mockHotspotAccelerationSource struct {
	listFunc func(threshold int) ([]FastGrowingFile, error)
}

func (m *mockHotspotAccelerationSource) ListFastGrowingFiles(threshold int) ([]FastGrowingFile, error) {
	if m.listFunc != nil {
		return m.listFunc(threshold)
	}
	return nil, nil
}

func TestHotspotAccelerationDetector_NoSource(t *testing.T) {
	d := &HotspotAccelerationDetector{}
	_, err := d.Detect()
	if err == nil {
		t.Error("expected error for nil source")
	}
}

func TestHotspotAccelerationDetector_FindsFastGrowing(t *testing.T) {
	d := &HotspotAccelerationDetector{
		Source: &mockHotspotAccelerationSource{
			listFunc: func(threshold int) ([]FastGrowingFile, error) {
				return []FastGrowingFile{
					{Path: "pkg/daemon/ooda.go", NetGrowth: 350, CurrentSize: 800, HistoricalSize: 450},
				}, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	s := suggestions[0]
	if s.Detector != "hotspot_acceleration" {
		t.Errorf("Detector = %q", s.Detector)
	}
	if s.Key != "pkg/daemon/ooda.go" {
		t.Errorf("Key = %q", s.Key)
	}
	if s.IssueType != "investigation" {
		t.Errorf("IssueType = %q, want investigation", s.IssueType)
	}
	if s.Priority != 3 {
		t.Errorf("Priority = %d, want 3", s.Priority)
	}
}

func TestHotspotAccelerationDetector_Name(t *testing.T) {
	d := &HotspotAccelerationDetector{}
	if d.Name() != "hotspot_acceleration" {
		t.Errorf("Name() = %q", d.Name())
	}
}

// --- Knowledge Decay Detector Tests ---

type mockKnowledgeDecaySource struct {
	listFunc func(maxAge time.Duration) ([]DecayedModel, error)
}

func (m *mockKnowledgeDecaySource) ListDecayedModels(maxAge time.Duration) ([]DecayedModel, error) {
	if m.listFunc != nil {
		return m.listFunc(maxAge)
	}
	return nil, nil
}

func TestKnowledgeDecayDetector_NoSource(t *testing.T) {
	d := &KnowledgeDecayDetector{}
	_, err := d.Detect()
	if err == nil {
		t.Error("expected error for nil source")
	}
}

func TestKnowledgeDecayDetector_FindsDecayedModels(t *testing.T) {
	d := &KnowledgeDecayDetector{
		Source: &mockKnowledgeDecaySource{
			listFunc: func(maxAge time.Duration) ([]DecayedModel, error) {
				return []DecayedModel{
					{Slug: "spawn-architecture", DaysSinceProbe: 45},
				}, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	s := suggestions[0]
	if s.Detector != "knowledge_decay" {
		t.Errorf("Detector = %q", s.Detector)
	}
	if s.Key != "spawn-architecture" {
		t.Errorf("Key = %q", s.Key)
	}
	if s.Priority != 4 {
		t.Errorf("Priority = %d, want 4", s.Priority)
	}
}

func TestKnowledgeDecayDetector_Name(t *testing.T) {
	d := &KnowledgeDecayDetector{}
	if d.Name() != "knowledge_decay" {
		t.Errorf("Name() = %q", d.Name())
	}
}

// --- Skill Performance Drift Detector Tests ---

type mockSkillPerformanceDriftSource struct {
	listFunc func(currentThreshold, previousMin float64) ([]DriftedSkill, error)
}

func (m *mockSkillPerformanceDriftSource) ListDriftedSkills(currentThreshold, previousMin float64) ([]DriftedSkill, error) {
	if m.listFunc != nil {
		return m.listFunc(currentThreshold, previousMin)
	}
	return nil, nil
}

func TestSkillPerformanceDriftDetector_NoSource(t *testing.T) {
	d := &SkillPerformanceDriftDetector{}
	_, err := d.Detect()
	if err == nil {
		t.Error("expected error for nil source")
	}
}

func TestSkillPerformanceDriftDetector_FindsDrift(t *testing.T) {
	d := &SkillPerformanceDriftDetector{
		Source: &mockSkillPerformanceDriftSource{
			listFunc: func(currentThreshold, previousMin float64) ([]DriftedSkill, error) {
				return []DriftedSkill{
					{Name: "feature-impl", CurrentRate: 0.3, PreviousRate: 0.8, RecentSpawns: 10},
				}, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	s := suggestions[0]
	if s.Detector != "skill_performance_drift" {
		t.Errorf("Detector = %q", s.Detector)
	}
	if s.Key != "feature-impl" {
		t.Errorf("Key = %q", s.Key)
	}
	if s.IssueType != "investigation" {
		t.Errorf("IssueType = %q, want investigation", s.IssueType)
	}
	if s.Priority != 2 {
		t.Errorf("Priority = %d, want 2", s.Priority)
	}
}

func TestSkillPerformanceDriftDetector_Name(t *testing.T) {
	d := &SkillPerformanceDriftDetector{}
	if d.Name() != "skill_performance_drift" {
		t.Errorf("Name() = %q", d.Name())
	}
}

// --- Integration: All Phase 2 Detectors ---

func TestDaemon_RunPeriodicTriggerScan_AllPhase2Detectors(t *testing.T) {
	createCount := 0
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
		TriggerBudgetMax:    20,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{
			CountOpenFunc: func() (int, error) { return 0, nil },
			HasOpenFunc:   func(_, _ string) (bool, error) { return false, nil },
			CreateIssueFunc: func(s TriggerSuggestion) (string, error) {
				createCount++
				return fmt.Sprintf("orch-go-t%d", createCount), nil
			},
		},
	}

	detectors := []PatternDetector{
		&ModelContradictionsDetector{
			Source: &mockModelContradictionsSource{
				listFunc: func() ([]UnresolvedContradiction, error) {
					return []UnresolvedContradiction{
						{ModelSlug: "test-model", ProbeFilename: "2026-03-10-contradicts.md",
							ProbeDate: time.Now(), ModelUpdated: time.Now().Add(-48 * time.Hour)},
					}, nil
				},
			},
		},
		&HotspotAccelerationDetector{
			Source: &mockHotspotAccelerationSource{
				listFunc: func(threshold int) ([]FastGrowingFile, error) {
					return []FastGrowingFile{
						{Path: "pkg/daemon/big.go", NetGrowth: 300, CurrentSize: 1000, HistoricalSize: 700},
					}, nil
				},
			},
		},
		&KnowledgeDecayDetector{
			Source: &mockKnowledgeDecaySource{
				listFunc: func(maxAge time.Duration) ([]DecayedModel, error) {
					return []DecayedModel{
						{Slug: "old-model", DaysSinceProbe: 60},
					}, nil
				},
			},
		},
		&SkillPerformanceDriftDetector{
			Source: &mockSkillPerformanceDriftSource{
				listFunc: func(currentThreshold, previousMin float64) ([]DriftedSkill, error) {
					return []DriftedSkill{
						{Name: "feature-impl", CurrentRate: 0.3, PreviousRate: 0.8, RecentSpawns: 15},
					}, nil
				},
			},
		},
	}

	result := d.RunPeriodicTriggerScan(detectors)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Detected != 4 {
		t.Errorf("Detected = %d, want 4", result.Detected)
	}
	if result.Created != 4 {
		t.Errorf("Created = %d, want 4", result.Created)
	}
}

// --- parseGitDiffNumstat Tests ---

func TestParseGitDiffNumstat(t *testing.T) {
	// git diff --numstat gives: <added>\t<deleted>\t<path>
	input := "100\t20\tpkg/daemon/trigger.go\n50\t10\tpkg/daemon/ooda.go\n-\t-\tbinary.png\n"
	result := parseGitDiffNumstat(input)

	if result["pkg/daemon/trigger.go"] != 80 {
		t.Errorf("trigger.go net = %d, want 80 (100-20)", result["pkg/daemon/trigger.go"])
	}
	if result["pkg/daemon/ooda.go"] != 40 {
		t.Errorf("ooda.go net = %d, want 40 (50-10)", result["pkg/daemon/ooda.go"])
	}
	if _, exists := result["binary.png"]; exists {
		t.Error("binary files should be skipped")
	}
}

func TestParseGitDiffNumstat_ChurnProducesLowNetGrowth(t *testing.T) {
	// Simulates extraction churn: file had 800 lines added but 750 deleted
	// (rewrites during extraction). Net growth is only 50 — below threshold.
	input := "800\t750\tcmd/orch/stats_cmd.go\n"
	result := parseGitDiffNumstat(input)

	if result["cmd/orch/stats_cmd.go"] != 50 {
		t.Errorf("stats_cmd.go net = %d, want 50 (800-750)", result["cmd/orch/stats_cmd.go"])
	}
}

func TestParseGitDiffNumstat_NegativeNetGrowth(t *testing.T) {
	// File shrank (more deleted than added) — net growth is negative
	input := "100\t400\tcmd/orch/plan_cmd.go\n"
	result := parseGitDiffNumstat(input)

	if result["cmd/orch/plan_cmd.go"] != -300 {
		t.Errorf("plan_cmd.go net = %d, want -300 (100-400)", result["cmd/orch/plan_cmd.go"])
	}
}

func TestParseGitDiffNumstat_Empty(t *testing.T) {
	result := parseGitDiffNumstat("")
	if len(result) != 0 {
		t.Errorf("got %d entries, want 0", len(result))
	}
}

// --- Churn False Positive Elimination Test ---

func TestHotspotAccelerationDetector_ChurnNotFlaggedAsGrowth(t *testing.T) {
	// Reproduction case from orch-go-9aicv:
	// stats_cmd.go had +806 additions but is only 302 lines (was extracted/split).
	// With net growth calculation, the source returns no files because
	// net growth (current - historical) is small or negative.
	d := &HotspotAccelerationDetector{
		Source: &mockHotspotAccelerationSource{
			listFunc: func(threshold int) ([]FastGrowingFile, error) {
				// Net growth approach: stats_cmd.go went from 900 to 302 lines
				// Net growth = -598 — well below threshold, not returned by source
				return nil, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("got %d suggestions, want 0 (churn should not trigger)", len(suggestions))
	}
}

func TestHotspotAccelerationDetector_GenuineGrowthStillDetected(t *testing.T) {
	// A file genuinely growing from 500 to 800 lines (net +300) should be detected.
	d := &HotspotAccelerationDetector{
		Source: &mockHotspotAccelerationSource{
			listFunc: func(threshold int) ([]FastGrowingFile, error) {
				return []FastGrowingFile{
					{Path: "pkg/daemon/ooda.go", NetGrowth: 300, CurrentSize: 800, HistoricalSize: 500},
				}, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	if suggestions[0].Key != "pkg/daemon/ooda.go" {
		t.Errorf("Key = %q, want pkg/daemon/ooda.go", suggestions[0].Key)
	}
}

func TestIsAccelerationExcluded(t *testing.T) {
	tests := []struct {
		path     string
		excluded bool
	}{
		// Production code — NOT excluded
		{"pkg/daemon/ooda.go", false},
		{"cmd/orch/main.go", false},
		{"pkg/spawn/gates/hotspot.go", false},

		// Test files — excluded
		{"pkg/daemon/ooda_test.go", true},
		{"pkg/thread/thread_test.go", true},

		// experiments/ — excluded
		{"experiments/coordination-demo/display_test.go", true},
		{"experiments/foo/bar.go", true},

		// Non-production directories — excluded
		{".orch/workspace/foo.go", true},
		{".beads/hooks/foo.go", true},
		{".claude/worktrees/foo.go", true},
		{"vendor/github.com/foo/bar.go", true},
		{"node_modules/foo/bar.go", true},
		{"web/.svelte-kit/output/foo.go", true},

		// Generated files — excluded
		{"pkg/generated/types.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isAccelerationExcluded(tt.path)
			if got != tt.excluded {
				t.Errorf("isAccelerationExcluded(%q) = %v, want %v", tt.path, got, tt.excluded)
			}
		})
	}
}
