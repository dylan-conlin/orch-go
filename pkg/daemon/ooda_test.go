package daemon

import (
	"fmt"
	"testing"
)

// =============================================================================
// Tests for OODA Sense phase
// =============================================================================

func TestSense_CollectsIssues(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Test 2", Priority: 1, IssueType: "bug", Status: "open"},
			}, nil
		}},
	}

	result := d.Sense(nil)
	if result.IssueErr != nil {
		t.Fatalf("Sense() unexpected error: %v", result.IssueErr)
	}
	if len(result.Issues) != 2 {
		t.Errorf("Sense() issues = %d, want 2", len(result.Issues))
	}
	if !result.GateSignal.Allowed {
		t.Error("Sense() gate should be allowed with no trackers set")
	}
}

func TestSense_GatesBlock(t *testing.T) {
	tracker := NewVerificationTracker(1)
	tracker.RecordCompletion("test-1") // triggers pause at threshold=1

	d := &Daemon{
		VerificationTracker: tracker,
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		}},
	}

	result := d.Sense(nil)
	if result.GateSignal.Allowed {
		t.Error("Sense() gate should be blocked when verification paused")
	}
	if result.GateSignal.Reason == "" {
		t.Error("Sense() should provide block reason")
	}
}

func TestSense_IssueQueryError(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return nil, fmt.Errorf("beads unavailable")
		}},
	}

	result := d.Sense(nil)
	if result.IssueErr == nil {
		t.Error("Sense() should propagate issue query error")
	}
}

// =============================================================================
// Tests for OODA Orient phase
// =============================================================================

func TestOrient_PrioritizesIssues(t *testing.T) {
	d := &Daemon{Config: Config{Label: "triage:ready"}}

	sense := SenseResult{
		GateSignal: SpawnGateSignal{Allowed: true},
		Issues: []Issue{
			{ID: "proj-2", Title: "Low prio", Priority: 2, IssueType: "bug", Status: "open", Labels: []string{"triage:ready"}},
			{ID: "proj-1", Title: "High prio", Priority: 0, IssueType: "feature", Status: "open", Labels: []string{"triage:ready"}},
		},
	}

	result := d.Orient(sense)
	if result.OrientErr != nil {
		t.Fatalf("Orient() unexpected error: %v", result.OrientErr)
	}
	if len(result.PrioritizedIssues) != 2 {
		t.Fatalf("Orient() issues = %d, want 2", len(result.PrioritizedIssues))
	}
	// Higher priority (lower number) should come first
	if result.PrioritizedIssues[0].ID != "proj-1" {
		t.Errorf("Orient() first issue = %s, want proj-1 (higher priority)", result.PrioritizedIssues[0].ID)
	}
}

func TestOrient_PropagatesIssueError(t *testing.T) {
	d := &Daemon{}

	sense := SenseResult{
		GateSignal: SpawnGateSignal{Allowed: true},
		IssueErr:   fmt.Errorf("beads unavailable"),
	}

	result := d.Orient(sense)
	if result.OrientErr == nil {
		t.Error("Orient() should propagate issue error from Sense")
	}
}

// =============================================================================
// Tests for OODA Decide phase
// =============================================================================

func TestDecide_SelectsHighestPriority(t *testing.T) {
	d := &Daemon{}

	orient := OrientResult{
		Sense:        SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		EpicChildIDs: map[string]bool{},
		PrioritizedIssues: []Issue{
			{ID: "proj-1", Title: "High prio feature", Priority: 0, IssueType: "feature", Status: "open"},
			{ID: "proj-2", Title: "Low prio bug", Priority: 1, IssueType: "bug", Status: "open"},
		},
	}

	decision := d.Decide(orient, nil)
	if !decision.ShouldSpawn {
		t.Fatalf("Decide() ShouldSpawn = false, want true; reason: %s", decision.BlockReason)
	}
	if decision.Issue.ID != "proj-1" {
		t.Errorf("Decide() selected %s, want proj-1", decision.Issue.ID)
	}
	if decision.Skill == "" {
		t.Error("Decide() should infer skill")
	}
	// Model may be empty — InferModelFromSkill returns "" for skills
	// without explicit model mapping, letting the resolve pipeline decide.
}

func TestDecide_BlockedByGate(t *testing.T) {
	d := &Daemon{}

	orient := OrientResult{
		Sense: SenseResult{
			GateSignal: SpawnGateSignal{Allowed: false, Reason: "rate limited"},
		},
	}

	decision := d.Decide(orient, nil)
	if decision.ShouldSpawn {
		t.Error("Decide() should not spawn when gates block")
	}
	if !decision.Blocked {
		t.Error("Decide() should be blocked")
	}
	if decision.BlockReason != "rate limited" {
		t.Errorf("Decide() reason = %q, want 'rate limited'", decision.BlockReason)
	}
}

func TestDecide_NoIssues(t *testing.T) {
	d := &Daemon{}

	orient := OrientResult{
		Sense:             SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		EpicChildIDs:      map[string]bool{},
		PrioritizedIssues: []Issue{},
	}

	decision := d.Decide(orient, nil)
	if decision.ShouldSpawn {
		t.Error("Decide() should not spawn with empty queue")
	}
	if decision.BlockReason == "" {
		t.Error("Decide() should provide reason for empty queue")
	}
}

func TestDecide_SkipsFilteredIssues(t *testing.T) {
	d := &Daemon{}

	orient := OrientResult{
		Sense:        SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		EpicChildIDs: map[string]bool{},
		PrioritizedIssues: []Issue{
			{ID: "proj-1", Title: "In progress", Priority: 0, IssueType: "feature", Status: "in_progress"},
			{ID: "proj-2", Title: "Spawnable", Priority: 1, IssueType: "bug", Status: "open"},
		},
	}

	decision := d.Decide(orient, nil)
	if !decision.ShouldSpawn {
		t.Fatalf("Decide() should spawn proj-2 after filtering proj-1; reason: %s", decision.BlockReason)
	}
	if decision.Issue.ID != "proj-2" {
		t.Errorf("Decide() selected %s, want proj-2", decision.Issue.ID)
	}
}

func TestDecide_SkipSet(t *testing.T) {
	d := &Daemon{}

	orient := OrientResult{
		Sense:        SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		EpicChildIDs: map[string]bool{},
		PrioritizedIssues: []Issue{
			{ID: "proj-1", Title: "Skipped", Priority: 0, IssueType: "feature", Status: "open"},
			{ID: "proj-2", Title: "Available", Priority: 1, IssueType: "bug", Status: "open"},
		},
	}

	skip := map[string]bool{"proj-1": true}
	decision := d.Decide(orient, skip)
	if !decision.ShouldSpawn {
		t.Fatalf("Decide() should spawn proj-2 after skip; reason: %s", decision.BlockReason)
	}
	if decision.Issue.ID != "proj-2" {
		t.Errorf("Decide() selected %s, want proj-2", decision.Issue.ID)
	}
}

// =============================================================================
// Tests for OODA Act phase
// =============================================================================

func TestAct_SpawnsIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	decision := SpawnDecision{
		ShouldSpawn: true,
		Issue:       &Issue{ID: "proj-1", Title: "Test", IssueType: "feature", Status: "open"},
		Skill:       "feature-impl",
		Model:       "sonnet",
		Route:       SkillRoute{Skill: "feature-impl", Model: "sonnet"},
	}

	result, err := d.Act(decision)
	if err != nil {
		t.Fatalf("Act() error: %v", err)
	}
	if !result.Processed {
		t.Errorf("Act() Processed = false, want true; message: %s", result.Message)
	}
	if !spawnCalled {
		t.Error("Act() should call spawner")
	}
}

func TestAct_NoSpawnWhenBlocked(t *testing.T) {
	d := &Daemon{}

	decision := SpawnDecision{
		ShouldSpawn: false,
		Blocked:     true,
		BlockReason: "rate limited",
	}

	result, err := d.Act(decision)
	if err != nil {
		t.Fatalf("Act() error: %v", err)
	}
	if result.Processed {
		t.Error("Act() should not process when decision is blocked")
	}
	if result.Message != "rate limited" {
		t.Errorf("Act() message = %q, want 'rate limited'", result.Message)
	}
}

func TestAct_PropagatesExtractionMetadata(t *testing.T) {
	d := &Daemon{
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	decision := SpawnDecision{
		ShouldSpawn: true,
		Issue:       &Issue{ID: "ext-1", Title: "Extract", IssueType: "task", Status: "open"},
		Skill:       "feature-impl",
		Model:       "sonnet",
		Route: SkillRoute{
			Skill:             "feature-impl",
			Model:             "sonnet",
			ExtractionSpawned: true,
			OriginalIssueID:   "proj-1",
		},
	}

	result, err := d.Act(decision)
	if err != nil {
		t.Fatalf("Act() error: %v", err)
	}
	if !result.ExtractionSpawned {
		t.Error("Act() should propagate ExtractionSpawned")
	}
	if result.OriginalIssueID != "proj-1" {
		t.Errorf("Act() OriginalIssueID = %q, want 'proj-1'", result.OriginalIssueID)
	}
}

// =============================================================================
// Tests for full OODA cycle (Sense → Orient → Decide → Act)
// =============================================================================

func TestOODA_FullCycle_SpawnsIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	// Full OODA cycle
	sense := d.Sense(nil)
	orient := d.Orient(sense)
	decision := d.Decide(orient, nil)
	result, err := d.Act(decision)

	if err != nil {
		t.Fatalf("OODA cycle error: %v", err)
	}
	if !result.Processed {
		t.Errorf("OODA cycle should spawn; message: %s", result.Message)
	}
	if !spawnCalled {
		t.Error("OODA cycle should call spawner")
	}
	if result.Issue == nil || result.Issue.ID != "proj-1" {
		t.Error("OODA cycle should spawn proj-1")
	}
}

func TestOODA_FullCycle_EmptyQueue(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		}},
	}

	sense := d.Sense(nil)
	orient := d.Orient(sense)
	decision := d.Decide(orient, nil)
	result, err := d.Act(decision)

	if err != nil {
		t.Fatalf("OODA cycle error: %v", err)
	}
	if result.Processed {
		t.Error("OODA cycle should not process with empty queue")
	}
}

func TestOODA_FullCycle_GatesBlock(t *testing.T) {
	tracker := NewVerificationTracker(1)
	tracker.RecordCompletion("test-1")

	d := &Daemon{
		VerificationTracker: tracker,
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
	}

	sense := d.Sense(nil)
	orient := d.Orient(sense)
	decision := d.Decide(orient, nil)
	result, err := d.Act(decision)

	if err != nil {
		t.Fatalf("OODA cycle error: %v", err)
	}
	if result.Processed {
		t.Error("OODA cycle should not process when gates block")
	}
}

// TestOODA_BehavioralEquivalence verifies that the OODA cycle produces
// the same result as the original OnceExcluding for the same inputs.
func TestOODA_BehavioralEquivalence(t *testing.T) {
	makeIssues := func() []Issue {
		return []Issue{
			{ID: "proj-1", Title: "High prio", Priority: 0, IssueType: "feature", Status: "open"},
			{ID: "proj-2", Title: "Low prio", Priority: 1, IssueType: "bug", Status: "open"},
		}
	}

	makeDaemon := func() *Daemon {
		return &Daemon{
			Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
				return makeIssues(), nil
			}},
			Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				return nil
			}},
			StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
				return nil
			}},
		}
	}

	// Run via OnceExcluding
	d1 := makeDaemon()
	result1, err1 := d1.OnceExcluding(nil)

	// Run via OODA
	d2 := makeDaemon()
	sense := d2.Sense(nil)
	orient := d2.Orient(sense)
	decision := d2.Decide(orient, nil)
	result2, err2 := d2.Act(decision)

	// Compare
	if err1 != nil || err2 != nil {
		t.Fatalf("errors: OnceExcluding=%v, OODA=%v", err1, err2)
	}
	if result1.Processed != result2.Processed {
		t.Errorf("Processed: OnceExcluding=%v, OODA=%v", result1.Processed, result2.Processed)
	}
	if result1.Issue != nil && result2.Issue != nil {
		if result1.Issue.ID != result2.Issue.ID {
			t.Errorf("Issue.ID: OnceExcluding=%s, OODA=%s", result1.Issue.ID, result2.Issue.ID)
		}
	}
	if result1.Skill != result2.Skill {
		t.Errorf("Skill: OnceExcluding=%s, OODA=%s", result1.Skill, result2.Skill)
	}
}
