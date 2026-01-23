// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"testing"
)

// TestBug_CrossProjectPreviewVsOnce tests the discrepancy where Preview shows
// an issue as spawnable but OnceExcluding doesn't spawn it.
//
// Bug report: "Daemon doesn't see newly created issues with triage:ready label"
// - Created pw-ww8p in price-watch with triage:ready label
// - Verified label exists in local DB and JSONL file
// - Daemon polls price-watch but pw-ww8p doesn't appear in skip list or spawn list
// - Other pw-* issues from same project DO appear
// - Restarting daemon didn't help
// - orch daemon preview DOES show the issue as spawnable
func TestBug_CrossProjectPreviewVsOnce(t *testing.T) {
	projects := []Project{
		{Name: "price-watch", Path: "/Users/test/price-watch"},
	}

	issues := []Issue{
		{ID: "pw-abc1", Title: "Older issue", Priority: 1, IssueType: "feature", Status: "open", Labels: []string{"triage:ready"}},
		{ID: "pw-ww8p", Title: "Newly created", Priority: 2, IssueType: "bug", Status: "open", Labels: []string{"triage:ready"}},
	}

	config := Config{
		Label:        "triage:ready",
		CrossProject: true,
	}

	d := &Daemon{
		Config:        config,
		SpawnedIssues: NewSpawnedIssueTracker(),
		listProjectsFunc: func() ([]Project, error) {
			return projects, nil
		},
		listIssuesForProjectFunc: func(projectPath string) ([]Issue, error) {
			return issues, nil
		},
		spawnForProjectFunc: func(beadsID, projectPath string) error {
			t.Logf("Would spawn: %s in %s", beadsID, projectPath)
			return nil
		},
	}

	// Run Preview
	previewResult, err := d.CrossProjectPreview()
	if err != nil {
		t.Fatalf("CrossProjectPreview() error: %v", err)
	}

	t.Logf("Preview: NextIssue=%v, SpawnableCount=%d, RejectedCount=%d",
		previewResult.NextIssue != nil,
		len(previewResult.SpawnableIssues),
		len(previewResult.RejectedIssues))

	if previewResult.NextIssue == nil {
		t.Error("Preview should show at least one spawnable issue")
	} else {
		t.Logf("Preview NextIssue: %s", previewResult.NextIssue.ID)
	}

	for _, s := range previewResult.SpawnableIssues {
		t.Logf("  Spawnable: %s (%s)", s.Issue.ID, s.Issue.Title)
	}

	for _, r := range previewResult.RejectedIssues {
		t.Logf("  Rejected: %s - %s", r.Issue.ID, r.Reason)
	}

	// Run OnceExcluding
	onceResult, err := d.CrossProjectOnceExcluding(nil)
	if err != nil {
		t.Fatalf("CrossProjectOnceExcluding() error: %v", err)
	}

	t.Logf("Once: Processed=%v, Issue=%v, Message=%s",
		onceResult.Processed,
		onceResult.Issue != nil,
		onceResult.Message)

	if onceResult.Issue != nil {
		t.Logf("Once Issue: %s", onceResult.Issue.ID)
	}

	// Both should see the same issue (highest priority with label)
	if previewResult.NextIssue != nil && onceResult.Issue != nil {
		if previewResult.NextIssue.ID != onceResult.Issue.ID {
			t.Errorf("Preview and Once disagree on next issue: Preview=%s, Once=%s",
				previewResult.NextIssue.ID, onceResult.Issue.ID)
		}
	}

	// Check if both issues are visible to preview
	foundNewIssue := false
	for _, s := range previewResult.SpawnableIssues {
		if s.Issue.ID == "pw-ww8p" {
			foundNewIssue = true
			break
		}
	}
	if !foundNewIssue {
		t.Error("Preview should see pw-ww8p as spawnable")
	}
}

// TestBug_SessionDedupContinuesToNextIssue tests that session dedup check
// properly continues to the next issue instead of stopping.
//
// Before fix: When HasExistingSessionForBeadsID returns true for the first issue,
// CrossProjectOnceExcluding returned with Error=nil, which caused the caller to
// break the loop instead of retrying with the next issue.
//
// After fix: CrossProjectOnceExcluding iterates through all candidates until
// finding one that passes all checks.
func TestBug_SessionDedupContinuesToNextIssue(t *testing.T) {
	// Note: This test validates the fix by checking that with the skip set,
	// the second issue is processed. The actual session dedup is tested via
	// the HasExistingSessionForBeadsID mock in integration tests.

	projects := []Project{
		{Name: "test-project", Path: "/test"},
	}

	// Two issues, both spawnable
	issues := []Issue{
		{ID: "issue-1", Title: "First (would have session)", Priority: 0, IssueType: "bug", Status: "open", Labels: []string{"triage:ready"}},
		{ID: "issue-2", Title: "Second (no session)", Priority: 1, IssueType: "bug", Status: "open", Labels: []string{"triage:ready"}},
	}

	config := Config{
		Label:        "triage:ready",
		CrossProject: true,
	}

	spawnedID := ""
	d := &Daemon{
		Config:        config,
		SpawnedIssues: NewSpawnedIssueTracker(),
		listProjectsFunc: func() ([]Project, error) {
			return projects, nil
		},
		listIssuesForProjectFunc: func(projectPath string) ([]Issue, error) {
			return issues, nil
		},
		spawnForProjectFunc: func(beadsID, projectPath string) error {
			spawnedID = beadsID
			return nil
		},
	}

	// Simulate issue-1 already spawned (in SpawnedIssues tracker)
	// This mimics what happens when session dedup would skip the issue
	d.SpawnedIssues.MarkSpawned("issue-1")

	// Call CrossProjectOnceExcluding - should skip issue-1 and spawn issue-2
	result, err := d.CrossProjectOnceExcluding(nil)
	if err != nil {
		t.Fatalf("CrossProjectOnceExcluding error: %v", err)
	}

	t.Logf("Result: Processed=%v, Issue=%v, Message=%s",
		result.Processed, result.Issue != nil, result.Message)

	if !result.Processed {
		t.Error("Should have processed an issue")
	}

	if result.Issue == nil {
		t.Fatal("Should return the processed issue")
	}

	if result.Issue.ID != "issue-2" {
		t.Errorf("Should have spawned issue-2 (skipping issue-1), got %s", result.Issue.ID)
	}

	if spawnedID != "issue-2" {
		t.Errorf("spawnForProjectFunc should have been called with issue-2, got %s", spawnedID)
	}
}

// TestBug_SingleProjectSessionDedupContinuesToNextIssue tests that the
// single-project OnceExcluding properly continues to the next issue when
// session dedup or Phase: Complete check fails.
func TestBug_SingleProjectSessionDedupContinuesToNextIssue(t *testing.T) {
	// Two issues, both spawnable
	issues := []Issue{
		{ID: "issue-1", Title: "First (would have session)", Priority: 0, IssueType: "bug", Status: "open", Labels: []string{"triage:ready"}},
		{ID: "issue-2", Title: "Second (no session)", Priority: 1, IssueType: "bug", Status: "open", Labels: []string{"triage:ready"}},
	}

	config := Config{
		Label: "triage:ready",
	}

	spawnedID := ""
	d := &Daemon{
		Config:        config,
		SpawnedIssues: NewSpawnedIssueTracker(),
		listIssuesFunc: func() ([]Issue, error) {
			return issues, nil
		},
		spawnFunc: func(beadsID string) error {
			spawnedID = beadsID
			return nil
		},
	}

	// Simulate issue-1 already spawned (in SpawnedIssues tracker)
	d.SpawnedIssues.MarkSpawned("issue-1")

	// Call OnceExcluding - should skip issue-1 and spawn issue-2
	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding error: %v", err)
	}

	t.Logf("Result: Processed=%v, Issue=%v, Message=%s",
		result.Processed, result.Issue != nil, result.Message)

	if !result.Processed {
		t.Error("Should have processed an issue")
	}

	if result.Issue == nil {
		t.Fatal("Should return the processed issue")
	}

	if result.Issue.ID != "issue-2" {
		t.Errorf("Should have spawned issue-2 (skipping issue-1), got %s", result.Issue.ID)
	}

	if spawnedID != "issue-2" {
		t.Errorf("spawnFunc should have been called with issue-2, got %s", spawnedID)
	}
}

// TestBug_ListReadyIssuesReturnsNewlyCreated verifies that the issue list
// function returns newly created issues with the triage:ready label.
func TestBug_ListReadyIssuesReturnsNewlyCreated(t *testing.T) {
	// This test verifies that the mock list function returns all issues
	// including newly created ones

	issues := []Issue{
		{ID: "old-issue", Title: "Old", Priority: 2, IssueType: "feature", Status: "open", Labels: []string{"triage:ready"}},
		{ID: "new-issue", Title: "Newly created", Priority: 2, IssueType: "bug", Status: "open", Labels: []string{"triage:ready"}},
	}

	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		listIssuesFunc: func() ([]Issue, error) {
			return issues, nil
		},
	}

	// Get preview - should show both issues
	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview error: %v", err)
	}

	if result.Issue == nil {
		t.Error("Preview should return a spawnable issue")
	}

	// Check that new-issue is in the spawnable list (it's in RejectedIssues if rejected,
	// otherwise it's spawnable and one of them is NextIssue)
	foundNew := result.Issue != nil && result.Issue.ID == "new-issue"
	if !foundNew {
		for _, r := range result.RejectedIssues {
			if r.Issue.ID == "new-issue" {
				t.Errorf("new-issue was rejected: %s", r.Reason)
			}
		}
	}

	t.Logf("Preview NextIssue: %v", result.Issue)
	t.Logf("Preview RejectedCount: %d", len(result.RejectedIssues))
}
