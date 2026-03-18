package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
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

// --- probeContainsContradictionSignal Tests ---

func TestProbeContainsContradictionSignal_ExplicitKeywords(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"contradicts", "This contradicts the model's claim about X.", true},
		{"contradiction", "Found a contradiction in the model.", true},
		{"refutes", "Evidence refutes the stated failure mode.", true},
		{"disproven", "Claim has been disproven by measurement.", true},
		{"debunked", "The assumption was debunked.", true},
		{"overturned", "New evidence overturned the previous finding.", true},
		{"invalidated", "The constraint was invalidated by the refactor.", true},
		{"no match", "This confirms the model's claim about X.", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := probeContainsContradictionSignal(tt.content); got != tt.want {
				t.Errorf("probeContainsContradictionSignal() = %v, want %v for %q", got, tt.want, tt.content)
			}
		})
	}
}

func TestProbeContainsContradictionSignal_NegationPatterns(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"not true", "The claim that X improves performance is not true.", true},
		{"NOT true caps", "X is NOT true in the current codebase.", true},
		{"not accurate", "The stated threshold is not accurate.", true},
		{"not valid", "This assumption is not valid after the refactor.", true},
		{"not correct", "The documented behavior is not correct.", true},
		{"not supported", "The claim is not supported by evidence.", true},
		{"no longer true", "X is no longer true after the migration.", true},
		{"no longer valid", "The constraint is no longer valid.", true},
		{"no longer holds", "The invariant no longer holds.", true},
		{"does not hold", "The assumption does not hold in practice.", true},
		{"doesn't hold", "This doesn't hold when tested against real data.", true},
		{"contrary to", "Contrary to the model's claim, X works fine.", true},
		{"counter to", "Results are counter to the stated hypothesis.", true},
		{"opposite of", "We observed the opposite of what the model predicts.", true},
		{"no negation", "The model's claim is confirmed by this test.", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := probeContainsContradictionSignal(tt.content); got != tt.want {
				t.Errorf("probeContainsContradictionSignal() = %v, want %v for %q", got, tt.want, tt.content)
			}
		})
	}
}

func TestProbeContainsContradictionSignal_IncorrectnessTerms(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"is incorrect", "The model's claim about spawn rates is incorrect.", true},
		{"is inaccurate", "The documented threshold is inaccurate.", true},
		{"is wrong", "The stated assumption is wrong.", true},
		{"is false", "The claim is false based on measurements.", true},
		{"is mistaken", "The model is mistaken about the default behavior.", true},
		{"was incorrect", "The previous estimate was incorrect.", true},
		{"was wrong", "The hypothesis was wrong.", true},
		{"are false", "Several claims are false.", true},
		{"found to be false", "The claim was found to be false.", true},
		{"found to be incorrect", "Two assertions were found to be incorrect.", true},
		{"found to be wrong", "The thresholds were found to be wrong.", true},
		{"no incorrectness", "The model is accurate and up to date.", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := probeContainsContradictionSignal(tt.content); got != tt.want {
				t.Errorf("probeContainsContradictionSignal() = %v, want %v for %q", got, tt.want, tt.content)
			}
		})
	}
}

func TestProbeContainsContradictionSignal_CaseInsensitive(t *testing.T) {
	// All matching should be case-insensitive
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"uppercase NOT TRUE", "Claim is NOT TRUE.", true},
		{"mixed case Contradicts", "This Contradicts the model.", true},
		{"all caps REFUTES", "Evidence REFUTES the claim.", true},
		{"mixed No Longer Valid", "The constraint is No Longer Valid.", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := probeContainsContradictionSignal(tt.content); got != tt.want {
				t.Errorf("probeContainsContradictionSignal() = %v, want %v for %q", got, tt.want, tt.content)
			}
		})
	}
}

func TestProbeContainsContradictionSignal_RealisticProbeContent(t *testing.T) {
	// Test with realistic probe file content that uses negation without "contradict"
	probeWithNegation := `---
question: Does the 1500-line threshold still apply?
---

## What I Tested
Checked current file sizes against the documented threshold.

## What I Observed
The 1500-line extraction threshold is no longer valid. After the March refactor,
the effective limit was raised to 2000 lines but the model still states 1500.

## Model Impact
The documented threshold is incorrect and should be updated.
`
	if !probeContainsContradictionSignal(probeWithNegation) {
		t.Error("expected contradiction signal in realistic probe with negation language")
	}

	// Probe that confirms the model — should NOT trigger
	probeConfirming := `---
question: Does the spawn rate match predictions?
---

## What I Tested
Measured actual spawn rates over 24 hours.

## What I Observed
Spawn rates match the model's predictions within 5% margin.
The documented throughput ceiling is accurate.

## Model Impact
Confirms the model's spawn rate claims. No updates needed.
`
	if probeContainsContradictionSignal(probeConfirming) {
		t.Error("expected no contradiction signal in confirming probe")
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

func TestDefaultSkillPerformanceDriftSource_RealWindowedRates(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")

	now := time.Now()
	oldTS := now.Add(-45 * 24 * time.Hour) // 45 days ago — in "previous" window
	newTS := now.Add(-10 * 24 * time.Hour)  // 10 days ago — in "recent" window

	// Previous window: 8 successes out of 10 = 80% success rate
	var evts []events.Event
	for i := 0; i < 8; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentCompleted, Timestamp: oldTS.Unix() + int64(i*100),
			Data: map[string]interface{}{"skill": "feature-impl", "outcome": "success"},
		})
	}
	for i := 0; i < 2; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentAbandonedTelemetry, Timestamp: oldTS.Unix() + int64(800+i*100),
			Data: map[string]interface{}{"skill": "feature-impl"},
		})
	}

	// Recent window: 2 successes out of 8 = 25% success rate (drifted)
	for i := 0; i < 2; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentCompleted, Timestamp: newTS.Unix() + int64(i*100),
			Data: map[string]interface{}{"skill": "feature-impl", "outcome": "success"},
		})
	}
	for i := 0; i < 6; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentAbandonedTelemetry, Timestamp: newTS.Unix() + int64(200+i*100),
			Data: map[string]interface{}{"skill": "feature-impl"},
		})
	}

	writeTestEvents(t, eventsPath, evts)

	src := &defaultSkillPerformanceDriftSource{
		EventsPath: eventsPath,
		Now:        func() time.Time { return now },
	}

	drifted, err := src.ListDriftedSkills(0.5, 0.7)
	if err != nil {
		t.Fatalf("ListDriftedSkills() error = %v", err)
	}
	if len(drifted) != 1 {
		t.Fatalf("got %d drifted skills, want 1", len(drifted))
	}

	d := drifted[0]
	if d.Name != "feature-impl" {
		t.Errorf("Name = %q, want feature-impl", d.Name)
	}
	// Previous rate should be ~0.8 (8/10), not 0.7 placeholder
	if d.PreviousRate < 0.79 || d.PreviousRate > 0.81 {
		t.Errorf("PreviousRate = %f, want ~0.8 (real measured rate)", d.PreviousRate)
	}
	// Current rate should be ~0.25 (2/8)
	if d.CurrentRate < 0.24 || d.CurrentRate > 0.26 {
		t.Errorf("CurrentRate = %f, want ~0.25", d.CurrentRate)
	}
}

func TestDefaultSkillPerformanceDriftSource_NoDriftWhenPreviousTooLow(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")

	now := time.Now()
	oldTS := now.Add(-45 * 24 * time.Hour)
	newTS := now.Add(-10 * 24 * time.Hour)

	// Previous window: 3 successes out of 10 = 30% (below previousMin=0.7)
	var evts []events.Event
	for i := 0; i < 3; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentCompleted, Timestamp: oldTS.Unix() + int64(i*100),
			Data: map[string]interface{}{"skill": "feature-impl", "outcome": "success"},
		})
	}
	for i := 0; i < 7; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentAbandonedTelemetry, Timestamp: oldTS.Unix() + int64(300+i*100),
			Data: map[string]interface{}{"skill": "feature-impl"},
		})
	}

	// Recent window: 2 successes out of 8 = 25% — but previous was already low
	for i := 0; i < 2; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentCompleted, Timestamp: newTS.Unix() + int64(i*100),
			Data: map[string]interface{}{"skill": "feature-impl", "outcome": "success"},
		})
	}
	for i := 0; i < 6; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentAbandonedTelemetry, Timestamp: newTS.Unix() + int64(200+i*100),
			Data: map[string]interface{}{"skill": "feature-impl"},
		})
	}

	writeTestEvents(t, eventsPath, evts)

	src := &defaultSkillPerformanceDriftSource{
		EventsPath: eventsPath,
		Now:        func() time.Time { return now },
	}

	drifted, err := src.ListDriftedSkills(0.5, 0.7)
	if err != nil {
		t.Fatalf("ListDriftedSkills() error = %v", err)
	}
	if len(drifted) != 0 {
		t.Errorf("got %d drifted skills, want 0 (previous rate was already low)", len(drifted))
	}
}

func TestDefaultSkillPerformanceDriftSource_SkipsInsufficientSamples(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")

	now := time.Now()
	oldTS := now.Add(-45 * 24 * time.Hour)
	newTS := now.Add(-10 * 24 * time.Hour)

	// Previous window: plenty of data (8 events)
	var evts []events.Event
	for i := 0; i < 8; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentCompleted, Timestamp: oldTS.Unix() + int64(i*100),
			Data: map[string]interface{}{"skill": "feature-impl", "outcome": "success"},
		})
	}

	// Recent window: only 3 events — below minOutcomesPerWindow (5)
	for i := 0; i < 3; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentAbandonedTelemetry, Timestamp: newTS.Unix() + int64(i*100),
			Data: map[string]interface{}{"skill": "feature-impl"},
		})
	}

	writeTestEvents(t, eventsPath, evts)

	src := &defaultSkillPerformanceDriftSource{
		EventsPath: eventsPath,
		Now:        func() time.Time { return now },
	}

	drifted, err := src.ListDriftedSkills(0.5, 0.7)
	if err != nil {
		t.Fatalf("ListDriftedSkills() error = %v", err)
	}
	if len(drifted) != 0 {
		t.Errorf("got %d drifted skills, want 0 (insufficient recent samples)", len(drifted))
	}
}

func TestDefaultSkillPerformanceDriftSource_NoPreviousData(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")

	now := time.Now()
	newTS := now.Add(-10 * 24 * time.Hour)

	// Only recent data, no previous window data
	var evts []events.Event
	for i := 0; i < 2; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentCompleted, Timestamp: newTS.Unix() + int64(i*100),
			Data: map[string]interface{}{"skill": "feature-impl", "outcome": "success"},
		})
	}
	for i := 0; i < 6; i++ {
		evts = append(evts, events.Event{
			Type: events.EventTypeAgentAbandonedTelemetry, Timestamp: newTS.Unix() + int64(200+i*100),
			Data: map[string]interface{}{"skill": "feature-impl"},
		})
	}

	writeTestEvents(t, eventsPath, evts)

	src := &defaultSkillPerformanceDriftSource{
		EventsPath: eventsPath,
		Now:        func() time.Time { return now },
	}

	drifted, err := src.ListDriftedSkills(0.5, 0.7)
	if err != nil {
		t.Fatalf("ListDriftedSkills() error = %v", err)
	}
	if len(drifted) != 0 {
		t.Errorf("got %d drifted skills, want 0 (no previous data to compare against)", len(drifted))
	}
}

// writeTestEvents writes events to a JSONL file for testing.
func writeTestEvents(t *testing.T, path string, evts []events.Event) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, e := range evts {
		data, _ := json.Marshal(e)
		f.Write(append(data, '\n'))
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
