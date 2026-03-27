package daemon

import (
	"testing"
	"time"
)

// =============================================================================
// extractReferencedBeadsIDs Tests
// =============================================================================

func TestExtractReferencedBeadsIDs_FindsIDs(t *testing.T) {
	desc := "Follow-up from architect orch-go-k6c0v. See Issue: orch-go-paatt for context."
	refs := extractReferencedBeadsIDs(desc, "orch-go-94bxz")

	if len(refs) != 2 {
		t.Fatalf("extractReferencedBeadsIDs() got %d refs, want 2", len(refs))
	}
	// Check both IDs found (order may vary since FindAllString is left-to-right)
	found := map[string]bool{}
	for _, r := range refs {
		found[r] = true
	}
	if !found["orch-go-k6c0v"] {
		t.Error("expected orch-go-k6c0v in refs")
	}
	if !found["orch-go-paatt"] {
		t.Error("expected orch-go-paatt in refs")
	}
}

func TestExtractReferencedBeadsIDs_ExcludesSelf(t *testing.T) {
	desc := "Auto-created from orch-go-k6c0v. Self reference: orch-go-94bxz."
	refs := extractReferencedBeadsIDs(desc, "orch-go-94bxz")

	if len(refs) != 1 {
		t.Fatalf("extractReferencedBeadsIDs() got %d refs, want 1 (self excluded)", len(refs))
	}
	if refs[0] != "orch-go-k6c0v" {
		t.Errorf("extractReferencedBeadsIDs()[0] = %q, want orch-go-k6c0v", refs[0])
	}
}

func TestExtractReferencedBeadsIDs_Deduplicates(t *testing.T) {
	desc := "See orch-go-paatt and also orch-go-paatt for details."
	refs := extractReferencedBeadsIDs(desc, "orch-go-self1")

	if len(refs) != 1 {
		t.Fatalf("extractReferencedBeadsIDs() got %d refs, want 1 (deduped)", len(refs))
	}
}

func TestExtractReferencedBeadsIDs_EmptyDescription(t *testing.T) {
	refs := extractReferencedBeadsIDs("", "orch-go-self1")
	if len(refs) != 0 {
		t.Errorf("extractReferencedBeadsIDs() got %d refs, want 0 for empty description", len(refs))
	}
}

func TestExtractReferencedBeadsIDs_NoMatches(t *testing.T) {
	refs := extractReferencedBeadsIDs("Just a regular description with no IDs.", "orch-go-self1")
	if len(refs) != 0 {
		t.Errorf("extractReferencedBeadsIDs() got %d refs, want 0", len(refs))
	}
}

func TestExtractReferencedBeadsIDs_CrossProjectIDs(t *testing.T) {
	desc := "Related to price-watch-a1b2c issue."
	refs := extractReferencedBeadsIDs(desc, "orch-go-self1")

	if len(refs) != 1 {
		t.Fatalf("extractReferencedBeadsIDs() got %d refs, want 1", len(refs))
	}
	if refs[0] != "price-watch-a1b2c" {
		t.Errorf("extractReferencedBeadsIDs()[0] = %q, want price-watch-a1b2c", refs[0])
	}
}

// =============================================================================
// CommitDedupGate Tests
// =============================================================================

func TestCommitDedupGate_AllowsWhenNoCommits(t *testing.T) {
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool { return false },
	}
	issue := &Issue{ID: "orch-go-test1", Title: "Test", Description: "Some desc"}
	result := gate.Check(issue)

	if result.Verdict != GateAllow {
		t.Errorf("CommitDedupGate.Check() = %v, want GateAllow when no commits", result.Verdict)
	}
}

func TestCommitDedupGate_RejectsSelfWithCommits(t *testing.T) {
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			return beadsID == "orch-go-test1"
		},
	}
	issue := &Issue{ID: "orch-go-test1", Title: "Test", Description: "Some desc"}
	result := gate.Check(issue)

	if result.Verdict != GateReject {
		t.Errorf("CommitDedupGate.Check() = %v, want GateReject when self has commits", result.Verdict)
	}
	if result.Gate != "commit-dedup" {
		t.Errorf("CommitDedupGate.Check() gate = %q, want commit-dedup", result.Gate)
	}
}

func TestCommitDedupGate_RejectsReferencedWithCommits(t *testing.T) {
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			return beadsID == "orch-go-paatt" // The referenced issue has commits
		},
	}
	issue := &Issue{
		ID:          "orch-go-94bxz",
		Title:       "Implement fix",
		Description: "Follow-up from architect orch-go-k6c0v. Issue: orch-go-paatt.",
	}
	result := gate.Check(issue)

	if result.Verdict != GateReject {
		t.Errorf("CommitDedupGate.Check() = %v, want GateReject when referenced issue has commits", result.Verdict)
	}
	if result.Message == "" {
		t.Error("CommitDedupGate.Check() should have rejection message")
	}
}

func TestCommitDedupGate_AllowsWhenReferencedHasNoCommits(t *testing.T) {
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool { return false },
	}
	issue := &Issue{
		ID:          "orch-go-test1",
		Title:       "Fix something",
		Description: "See orch-go-other for context",
	}
	result := gate.Check(issue)

	if result.Verdict != GateAllow {
		t.Errorf("CommitDedupGate.Check() = %v, want GateAllow when no commits found", result.Verdict)
	}
}

func TestCommitDedupGate_NilFuncAllows(t *testing.T) {
	gate := &CommitDedupGate{HasCommitsFunc: nil}
	issue := &Issue{ID: "orch-go-test1", Title: "Test"}
	result := gate.Check(issue)

	if result.Verdict != GateAllow {
		t.Errorf("CommitDedupGate.Check() with nil func = %v, want GateAllow", result.Verdict)
	}
}

func TestCommitDedupGate_EmptyDescription(t *testing.T) {
	called := false
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			if beadsID != "orch-go-test1" {
				called = true // Should not be called for non-self IDs
			}
			return false
		},
	}
	issue := &Issue{ID: "orch-go-test1", Title: "Test", Description: ""}
	result := gate.Check(issue)

	if result.Verdict != GateAllow {
		t.Errorf("CommitDedupGate.Check() = %v, want GateAllow for empty description", result.Verdict)
	}
	if called {
		t.Error("HasCommitsFunc should not be called for non-self IDs with empty description")
	}
}

func TestCommitDedupGate_FailMode(t *testing.T) {
	gate := &CommitDedupGate{}
	if gate.FailMode() != FailOpen {
		t.Errorf("CommitDedupGate.FailMode() = %v, want FailOpen", gate.FailMode())
	}
}

// =============================================================================
// CommitDedupGate Cross-Type Reference Tests (false positive fix)
// =============================================================================

func TestCommitDedupGate_AllowsCrossTypeReference(t *testing.T) {
	// Reproduction of the false positive: a task issue referencing a completed
	// investigation should be ALLOWED — it's follow-up work, not a duplicate.
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			return beadsID == "orch-go-y85zx" // Investigation has commits
		},
		GetIssueTypeFunc: func(beadsID string) string {
			if beadsID == "orch-go-y85zx" {
				return "investigation"
			}
			return ""
		},
	}
	issue := &Issue{
		ID:          "orch-go-efw7c",
		Title:       "Extract probe claim/verdict",
		IssueType:   "task",
		Description: "Follow-up from probe orch-go-y85zx recommendations.",
	}
	result := gate.Check(issue)

	if result.Verdict != GateAllow {
		t.Errorf("CommitDedupGate.Check() = %v, want GateAllow for cross-type reference (task→investigation)", result.Verdict)
	}
}

func TestCommitDedupGate_RejectsSameTypeReference(t *testing.T) {
	// Same-type references (task→task) should still be checked for duplication.
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			return beadsID == "orch-go-paatt"
		},
		GetIssueTypeFunc: func(beadsID string) string {
			if beadsID == "orch-go-paatt" {
				return "task"
			}
			return ""
		},
	}
	issue := &Issue{
		ID:          "orch-go-94bxz",
		Title:       "Implement fix",
		IssueType:   "task",
		Description: "Follow-up from orch-go-paatt.",
	}
	result := gate.Check(issue)

	if result.Verdict != GateReject {
		t.Errorf("CommitDedupGate.Check() = %v, want GateReject for same-type reference (task→task)", result.Verdict)
	}
}

func TestCommitDedupGate_NilGetIssueTypeFuncStillRejects(t *testing.T) {
	// Backward compatibility: when GetIssueTypeFunc is nil, behaves as before.
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			return beadsID == "orch-go-paatt"
		},
		// GetIssueTypeFunc is nil — no type lookup available
	}
	issue := &Issue{
		ID:          "orch-go-94bxz",
		Title:       "Implement fix",
		IssueType:   "task",
		Description: "Follow-up. Issue: orch-go-paatt.",
	}
	result := gate.Check(issue)

	if result.Verdict != GateReject {
		t.Errorf("CommitDedupGate.Check() = %v, want GateReject when GetIssueTypeFunc is nil", result.Verdict)
	}
}

func TestCommitDedupGate_UnknownRefTypeStillRejects(t *testing.T) {
	// When GetIssueTypeFunc returns empty (issue not found), fail open to rejection.
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			return beadsID == "orch-go-paatt"
		},
		GetIssueTypeFunc: func(beadsID string) string {
			return "" // Unknown type
		},
	}
	issue := &Issue{
		ID:          "orch-go-94bxz",
		Title:       "Implement fix",
		IssueType:   "task",
		Description: "Refs orch-go-paatt.",
	}
	result := gate.Check(issue)

	if result.Verdict != GateReject {
		t.Errorf("CommitDedupGate.Check() = %v, want GateReject when referenced type is unknown", result.Verdict)
	}
}

func TestCommitDedupGate_SelfCheckIgnoresTypeFunc(t *testing.T) {
	// Check 1 (self-ID commit check) should still reject regardless of type comparison.
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			return beadsID == "orch-go-test1"
		},
		GetIssueTypeFunc: func(beadsID string) string {
			return "investigation" // Different type — but self-check shouldn't care
		},
	}
	issue := &Issue{
		ID:          "orch-go-test1",
		Title:       "Test",
		IssueType:   "task",
		Description: "Some desc",
	}
	result := gate.Check(issue)

	if result.Verdict != GateReject {
		t.Errorf("CommitDedupGate.Check() = %v, want GateReject for self-ID even with type func", result.Verdict)
	}
}

func TestCommitDedupGate_MultipleCrossTypeRefsAllAllowed(t *testing.T) {
	// Multiple cross-type references should all be skipped.
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			// Both referenced issues have commits
			return beadsID == "orch-go-inv01" || beadsID == "orch-go-inv02"
		},
		GetIssueTypeFunc: func(beadsID string) string {
			if beadsID == "orch-go-inv01" || beadsID == "orch-go-inv02" {
				return "investigation"
			}
			return ""
		},
	}
	issue := &Issue{
		ID:          "orch-go-impl1",
		Title:       "Implement recommendations",
		IssueType:   "task",
		Description: "From probe orch-go-inv01 and orch-go-inv02 findings.",
	}
	result := gate.Check(issue)

	if result.Verdict != GateAllow {
		t.Errorf("CommitDedupGate.Check() = %v, want GateAllow for multiple cross-type references", result.Verdict)
	}
}

func TestCommitDedupGate_MixedRefsRejectsOnSameType(t *testing.T) {
	// If one ref is cross-type (allow) but another is same-type with commits (reject),
	// the same-type ref should still trigger rejection.
	gate := &CommitDedupGate{
		HasCommitsFunc: func(beadsID string) bool {
			return beadsID == "orch-go-inv01" || beadsID == "orch-go-task1"
		},
		GetIssueTypeFunc: func(beadsID string) string {
			if beadsID == "orch-go-inv01" {
				return "investigation"
			}
			if beadsID == "orch-go-task1" {
				return "task"
			}
			return ""
		},
	}
	issue := &Issue{
		ID:          "orch-go-impl1",
		Title:       "Implement fix",
		IssueType:   "task",
		Description: "From investigation orch-go-inv01. Related to orch-go-task1.",
	}
	result := gate.Check(issue)

	if result.Verdict != GateReject {
		t.Errorf("CommitDedupGate.Check() = %v, want GateReject when same-type ref has commits", result.Verdict)
	}
}

// =============================================================================
// extractKeywords Tests
// =============================================================================

func TestExtractKeywords_BasicExtraction(t *testing.T) {
	kw := extractKeywords("Fix failing spawn exploration judge flag test")
	expected := map[string]bool{
		"fix": true, "failing": true, "spawn": true,
		"exploration": true, "judge": true, "flag": true, "test": true,
	}
	for k := range expected {
		if !kw[k] {
			t.Errorf("extractKeywords missing keyword %q", k)
		}
	}
}

func TestExtractKeywords_FiltersShortWords(t *testing.T) {
	kw := extractKeywords("Fix a bug in the spawn system")
	// "a" and "in" and "the" should be filtered (< 3 chars)
	if kw["a"] {
		t.Error("extractKeywords should filter 'a'")
	}
	if kw["in"] {
		t.Error("extractKeywords should filter 'in'")
	}
	if kw["the"] {
		// "the" is 3 chars, so it passes the length filter
		// This is fine — it's a common word but not worth adding a stop word list
	}
}

func TestExtractKeywords_SplitsSlashPaths(t *testing.T) {
	kw := extractKeywords("Fix pkg/spawn explore test")
	if !kw["pkg"] {
		t.Error("extractKeywords should split 'pkg/spawn' into 'pkg'")
	}
	if !kw["spawn"] {
		t.Error("extractKeywords should split 'pkg/spawn' into 'spawn'")
	}
}

func TestExtractKeywords_StripsPunctuation(t *testing.T) {
	kw := extractKeywords("Fix: spawn (judge) test.")
	if !kw["fix"] {
		t.Error("extractKeywords should strip colon from 'Fix:'")
	}
	if !kw["judge"] {
		t.Error("extractKeywords should strip parens from '(judge)'")
	}
	if !kw["test"] {
		t.Error("extractKeywords should strip period from 'test.'")
	}
}

func TestExtractKeywords_CaseInsensitive(t *testing.T) {
	kw := extractKeywords("Fix SPAWN Judge")
	if !kw["fix"] {
		t.Error("extractKeywords should lowercase 'Fix'")
	}
	if !kw["spawn"] {
		t.Error("extractKeywords should lowercase 'SPAWN'")
	}
}

func TestExtractKeywords_EmptyTitle(t *testing.T) {
	kw := extractKeywords("")
	if len(kw) != 0 {
		t.Errorf("extractKeywords('') got %d keywords, want 0", len(kw))
	}
}

// =============================================================================
// overlapCoefficient Tests
// =============================================================================

func TestOverlapCoefficient_IdenticalSets(t *testing.T) {
	a := map[string]bool{"fix": true, "spawn": true, "test": true}
	b := map[string]bool{"fix": true, "spawn": true, "test": true}
	coeff := overlapCoefficient(a, b)
	if coeff != 1.0 {
		t.Errorf("overlapCoefficient(identical) = %f, want 1.0", coeff)
	}
}

func TestOverlapCoefficient_DisjointSets(t *testing.T) {
	a := map[string]bool{"fix": true, "spawn": true}
	b := map[string]bool{"add": true, "feature": true}
	coeff := overlapCoefficient(a, b)
	if coeff != 0.0 {
		t.Errorf("overlapCoefficient(disjoint) = %f, want 0.0", coeff)
	}
}

func TestOverlapCoefficient_PartialOverlap(t *testing.T) {
	// A has 7 words, B has 9 words, 4 in common
	// min(7,9) = 7, overlap = 4/7 ≈ 0.571
	a := map[string]bool{
		"fix": true, "failing": true, "spawn": true,
		"exploration": true, "judge": true, "flag": true, "test": true,
	}
	b := map[string]bool{
		"fix": true, "unrelated": true, "pkg": true, "spawn": true,
		"explore": true, "judge": true, "model": true, "test": true, "failure": true,
	}
	coeff := overlapCoefficient(a, b)
	// Common: fix, spawn, judge, test = 4, min(7,9) = 7, 4/7 ≈ 0.571
	if coeff < 0.55 || coeff > 0.58 {
		t.Errorf("overlapCoefficient(partial) = %f, want ~0.571", coeff)
	}
}

func TestOverlapCoefficient_EmptySet(t *testing.T) {
	a := map[string]bool{"fix": true}
	b := map[string]bool{}
	if overlapCoefficient(a, b) != 0.0 {
		t.Error("overlapCoefficient with empty set should be 0")
	}
	if overlapCoefficient(b, a) != 0.0 {
		t.Error("overlapCoefficient with empty set should be 0")
	}
}

// =============================================================================
// countCommon Tests
// =============================================================================

func TestCountCommon_Basic(t *testing.T) {
	a := map[string]bool{"fix": true, "spawn": true, "test": true, "judge": true}
	b := map[string]bool{"fix": true, "model": true, "test": true, "explore": true}
	count := countCommon(a, b)
	if count != 2 {
		t.Errorf("countCommon() = %d, want 2 (fix, test)", count)
	}
}

// =============================================================================
// KeywordDedupGate Tests
// =============================================================================

func TestKeywordDedupGate_AllowsNoOverlap(t *testing.T) {
	gate := &KeywordDedupGate{
		FindOverlapFunc: func(title, selfID string) (bool, string) {
			return false, ""
		},
	}
	issue := &Issue{ID: "test-1", Title: "Unique feature request"}
	result := gate.Check(issue)

	if result.Verdict != GateAllow {
		t.Errorf("KeywordDedupGate.Check() = %v, want GateAllow when no overlap", result.Verdict)
	}
}

func TestKeywordDedupGate_RejectsOverlap(t *testing.T) {
	gate := &KeywordDedupGate{
		FindOverlapFunc: func(title, selfID string) (bool, string) {
			return true, "orch-go-0vm6n"
		},
	}
	issue := &Issue{ID: "orch-go-0ocus", Title: "Fix unrelated pkg/spawn explore judge model test failure"}
	result := gate.Check(issue)

	if result.Verdict != GateReject {
		t.Errorf("KeywordDedupGate.Check() = %v, want GateReject when overlap found", result.Verdict)
	}
	if result.Gate != "keyword-dedup" {
		t.Errorf("KeywordDedupGate.Check() gate = %q, want keyword-dedup", result.Gate)
	}
}

func TestKeywordDedupGate_NilFuncAllows(t *testing.T) {
	gate := &KeywordDedupGate{FindOverlapFunc: nil}
	issue := &Issue{ID: "test-1", Title: "Test"}
	result := gate.Check(issue)

	if result.Verdict != GateAllow {
		t.Errorf("KeywordDedupGate.Check() with nil func = %v, want GateAllow", result.Verdict)
	}
}

func TestKeywordDedupGate_FailMode(t *testing.T) {
	gate := &KeywordDedupGate{}
	if gate.FailMode() != FailOpen {
		t.Errorf("KeywordDedupGate.FailMode() = %v, want FailOpen", gate.FailMode())
	}
}

// =============================================================================
// FindKeywordOverlap Integration Tests
// =============================================================================

func TestFindKeywordOverlap_CatchesSemanticDuplicate(t *testing.T) {
	// Reproduces the orch-go-0vm6n / orch-go-0ocus case
	tracker := NewSpawnedIssueTrackerWithTTL(1 * time.Hour)
	tracker.MarkSpawnedWithTitle("orch-go-0vm6n", "Fix failing spawn exploration judge flag test")

	found, matchID := FindKeywordOverlap(tracker,
		"Fix unrelated pkg/spawn explore judge model test failure",
		"orch-go-0ocus")

	if !found {
		t.Error("FindKeywordOverlap should detect semantic duplicate")
	}
	if matchID != "orch-go-0vm6n" {
		t.Errorf("FindKeywordOverlap matchID = %q, want orch-go-0vm6n", matchID)
	}
}

func TestFindKeywordOverlap_AllowsDifferentWork(t *testing.T) {
	tracker := NewSpawnedIssueTrackerWithTTL(1 * time.Hour)
	tracker.MarkSpawnedWithTitle("orch-go-aaaaa", "Add comprehension snapshot to daemon status")

	found, _ := FindKeywordOverlap(tracker,
		"Fix failing spawn exploration judge flag test",
		"orch-go-bbbbb")

	if found {
		t.Error("FindKeywordOverlap should not flag unrelated titles")
	}
}

func TestFindKeywordOverlap_SkipsSelf(t *testing.T) {
	tracker := NewSpawnedIssueTrackerWithTTL(1 * time.Hour)
	tracker.MarkSpawnedWithTitle("orch-go-self1", "Fix spawn judge test")

	found, _ := FindKeywordOverlap(tracker,
		"Fix spawn judge test",
		"orch-go-self1") // Same issue ID

	if found {
		t.Error("FindKeywordOverlap should skip self")
	}
}

func TestFindKeywordOverlap_SkipsExpiredEntries(t *testing.T) {
	tracker := NewSpawnedIssueTrackerWithTTL(1 * time.Millisecond)
	tracker.MarkSpawnedWithTitle("orch-go-old01", "Fix spawn judge test exploration flag")

	// Wait for TTL to expire
	time.Sleep(5 * time.Millisecond)

	found, _ := FindKeywordOverlap(tracker,
		"Fix spawn judge test exploration flag duplicate",
		"orch-go-new01")

	if found {
		t.Error("FindKeywordOverlap should skip expired entries")
	}
}

func TestFindKeywordOverlap_NilTracker(t *testing.T) {
	found, _ := FindKeywordOverlap(nil, "some title", "test-1")
	if found {
		t.Error("FindKeywordOverlap should return false for nil tracker")
	}
}

func TestFindKeywordOverlap_ShortTitle(t *testing.T) {
	tracker := NewSpawnedIssueTrackerWithTTL(1 * time.Hour)
	tracker.MarkSpawnedWithTitle("orch-go-aaaaa", "Fix bug")

	found, _ := FindKeywordOverlap(tracker, "Fix bug", "orch-go-bbbbb")
	if found {
		t.Error("FindKeywordOverlap should skip titles with < 3 keywords")
	}
}

func TestFindKeywordOverlap_RequiresMinCommonKeywords(t *testing.T) {
	// Two titles with 50% overlap but only 2 common keywords
	// Should NOT match because common < 3
	tracker := NewSpawnedIssueTrackerWithTTL(1 * time.Hour)
	tracker.MarkSpawnedWithTitle("orch-go-aaaaa", "Fix spawn timeout issue")

	found, _ := FindKeywordOverlap(tracker,
		"Fix spawn context problem",
		"orch-go-bbbbb")

	// Common keywords: "fix", "spawn" = 2 (below threshold of 3)
	if found {
		t.Error("FindKeywordOverlap should require >= 3 common keywords")
	}
}

// =============================================================================
// Pipeline Integration: New Gates in Full Pipeline
// =============================================================================

func TestSpawnPipeline_CommitDedupRejects(t *testing.T) {
	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&alwaysAllowGate{name: "gate-1"},
			&CommitDedupGate{
				HasCommitsFunc: func(beadsID string) bool {
					return beadsID == "orch-go-paatt"
				},
			},
		},
	}
	issue := &Issue{
		ID:          "orch-go-94bxz",
		Title:       "Implement fix",
		Description: "Follow-up. Issue: orch-go-paatt.",
	}
	result := pipeline.Run(issue)

	if result.Allowed {
		t.Error("pipeline should reject when referenced issue has commits")
	}
	if result.RejectedBy != "commit-dedup" {
		t.Errorf("RejectedBy = %q, want 'commit-dedup'", result.RejectedBy)
	}
}

func TestSpawnPipeline_KeywordDedupRejects(t *testing.T) {
	tracker := NewSpawnedIssueTrackerWithTTL(1 * time.Hour)
	tracker.MarkSpawnedWithTitle("orch-go-0vm6n", "Fix failing spawn exploration judge flag test")

	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&alwaysAllowGate{name: "gate-1"},
			&KeywordDedupGate{
				FindOverlapFunc: func(title, selfID string) (bool, string) {
					return FindKeywordOverlap(tracker, title, selfID)
				},
			},
		},
	}
	issue := &Issue{
		ID:    "orch-go-0ocus",
		Title: "Fix unrelated pkg/spawn explore judge model test failure",
	}
	result := pipeline.Run(issue)

	if result.Allowed {
		t.Error("pipeline should reject when keyword overlap detected")
	}
	if result.RejectedBy != "keyword-dedup" {
		t.Errorf("RejectedBy = %q, want 'keyword-dedup'", result.RejectedBy)
	}
}
