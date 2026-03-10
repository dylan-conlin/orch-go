package verify

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestEscalationLevel_String(t *testing.T) {
	tests := []struct {
		level    EscalationLevel
		expected string
	}{
		{EscalationNone, "none"},
		{EscalationInfo, "info"},
		{EscalationReview, "review"},
		{EscalationBlock, "block"},
		{EscalationFailed, "failed"},
		{EscalationLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("EscalationLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEscalationLevel_ShouldAutoComplete(t *testing.T) {
	tests := []struct {
		level    EscalationLevel
		expected bool
	}{
		{EscalationNone, true},
		{EscalationInfo, true},
		{EscalationReview, true},
		{EscalationBlock, false},
		{EscalationFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.ShouldAutoComplete(); got != tt.expected {
				t.Errorf("EscalationLevel.ShouldAutoComplete() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEscalationLevel_RequiresHumanReview(t *testing.T) {
	tests := []struct {
		level    EscalationLevel
		expected bool
	}{
		{EscalationNone, false},
		{EscalationInfo, false},
		{EscalationReview, true},
		{EscalationBlock, true},
		{EscalationFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.RequiresHumanReview(); got != tt.expected {
				t.Errorf("EscalationLevel.RequiresHumanReview() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsKnowledgeProducingSkill(t *testing.T) {
	tests := []struct {
		skillName string
		expected  bool
	}{
		{"investigation", true},
		{"architect", true},
		{"research", true},
		{"design-session", true},
		{"codebase-audit", true},
		{"issue-creation", true},
		{"INVESTIGATION", true}, // case insensitive
		{"Architect", true},
		{"feature-impl", false},
		{"systematic-debugging", false},
		{"reliability-testing", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.skillName, func(t *testing.T) {
			if got := IsKnowledgeProducingSkill(tt.skillName); got != tt.expected {
				t.Errorf("IsKnowledgeProducingSkill(%q) = %v, want %v", tt.skillName, got, tt.expected)
			}
		})
	}
}

func TestDetermineEscalation_VerificationFailed(t *testing.T) {
	// Rule 1: Verification failed -> EscalationFailed
	input := EscalationInput{
		VerificationPassed: false,
		VerificationErrors: []string{"Phase not complete"},
	}

	got := DetermineEscalation(input)
	if got != EscalationFailed {
		t.Errorf("DetermineEscalation() = %v, want EscalationFailed", got)
	}
}

func TestDetermineEscalation_KnowledgeProducingSkillWithRecommendations(t *testing.T) {
	// Rule 2: Knowledge skill + recommendations -> EscalationReview
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "investigation",
		NextActions:        []string{"Create follow-up issue"},
	}

	got := DetermineEscalation(input)
	if got != EscalationReview {
		t.Errorf("DetermineEscalation() = %v, want EscalationReview", got)
	}
}

func TestDetermineEscalation_KnowledgeProducingSkillWithoutRecommendations(t *testing.T) {
	// Rule 2: Knowledge skill without recommendations -> EscalationInfo
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "investigation",
		Recommendation:     "close",
	}

	got := DetermineEscalation(input)
	if got != EscalationInfo {
		t.Errorf("DetermineEscalation() = %v, want EscalationInfo", got)
	}
}

func TestDetermineEscalation_VisualApprovalNeeded(t *testing.T) {
	// Rule 3: Visual verification needs approval -> EscalationBlock
	input := EscalationInput{
		VerificationPassed:  true,
		SkillName:           "feature-impl",
		HasWebChanges:       true,
		HasVisualEvidence:   true,
		NeedsVisualApproval: true,
	}

	got := DetermineEscalation(input)
	if got != EscalationBlock {
		t.Errorf("DetermineEscalation() = %v, want EscalationBlock", got)
	}
}

func TestDetermineEscalation_NonSuccessOutcome(t *testing.T) {
	// Rule 4: Non-success outcome -> EscalationReview
	tests := []struct {
		outcome string
	}{
		{"partial"},
		{"blocked"},
		{"failed"},
	}

	for _, tt := range tests {
		t.Run(tt.outcome, func(t *testing.T) {
			input := EscalationInput{
				VerificationPassed: true,
				SkillName:          "feature-impl",
				Outcome:            tt.outcome,
			}

			got := DetermineEscalation(input)
			if got != EscalationReview {
				t.Errorf("DetermineEscalation(outcome=%q) = %v, want EscalationReview", tt.outcome, got)
			}
		})
	}
}

func TestDetermineEscalation_RecommendationsWithLargeScope(t *testing.T) {
	// Rule 5: Recommendations + large scope -> EscalationReview
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "spawn-follow-up",
		FileCount:          15, // > 10
	}

	got := DetermineEscalation(input)
	if got != EscalationReview {
		t.Errorf("DetermineEscalation() = %v, want EscalationReview", got)
	}
}

func TestDetermineEscalation_RecommendationsWithNormalScope(t *testing.T) {
	// Rule 5: Recommendations + normal scope -> EscalationInfo
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		NextActions:        []string{"Consider adding more tests"},
		FileCount:          5, // < 10
	}

	got := DetermineEscalation(input)
	if got != EscalationInfo {
		t.Errorf("DetermineEscalation() = %v, want EscalationInfo", got)
	}
}

func TestDetermineEscalation_LargeScopeWithoutRecommendations(t *testing.T) {
	// Rule 6: Large scope without recommendations -> EscalationInfo
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "close",
		FileCount:          12, // > 10
	}

	got := DetermineEscalation(input)
	if got != EscalationInfo {
		t.Errorf("DetermineEscalation() = %v, want EscalationInfo", got)
	}
}

func TestDetermineEscalation_CleanCompletion(t *testing.T) {
	// Rule 7: Clean completion -> EscalationNone
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "close",
		FileCount:          3, // < 10
	}

	got := DetermineEscalation(input)
	if got != EscalationNone {
		t.Errorf("DetermineEscalation() = %v, want EscalationNone", got)
	}
}

func TestDetermineEscalation_EmptyInput(t *testing.T) {
	// Empty input with verification passed should be clean
	input := EscalationInput{
		VerificationPassed: true,
	}

	got := DetermineEscalation(input)
	if got != EscalationNone {
		t.Errorf("DetermineEscalation() = %v, want EscalationNone", got)
	}
}

func TestDetermineEscalation_RecommendationTypes(t *testing.T) {
	// Test different recommendation types trigger escalation
	recommendationsNeedingReview := []string{
		"spawn-follow-up",
		"escalate",
		"resume",
		"continue",
		"SPAWN-FOLLOW-UP", // case insensitive
	}

	for _, rec := range recommendationsNeedingReview {
		t.Run(rec, func(t *testing.T) {
			input := EscalationInput{
				VerificationPassed: true,
				SkillName:          "feature-impl",
				Recommendation:     rec,
				FileCount:          5,
			}

			got := DetermineEscalation(input)
			if got != EscalationInfo {
				t.Errorf("DetermineEscalation(recommendation=%q) = %v, want EscalationInfo", rec, got)
			}
		})
	}
}

func TestDetermineEscalation_CloseRecommendation(t *testing.T) {
	// "close" recommendation should not trigger escalation by itself
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Recommendation:     "close",
		FileCount:          5,
	}

	got := DetermineEscalation(input)
	if got != EscalationNone {
		t.Errorf("DetermineEscalation() = %v, want EscalationNone", got)
	}
}

func TestExplainEscalation_Failed(t *testing.T) {
	input := EscalationInput{
		VerificationPassed: false,
		VerificationErrors: []string{"Phase not complete", "SYNTHESIS.md missing"},
	}

	reason := ExplainEscalation(input)
	if reason.Level != EscalationFailed {
		t.Errorf("ExplainEscalation().Level = %v, want EscalationFailed", reason.Level)
	}
	if reason.Reason != "Verification failed" {
		t.Errorf("ExplainEscalation().Reason = %q, want 'Verification failed'", reason.Reason)
	}
	if len(reason.Details) != 2 {
		t.Errorf("ExplainEscalation().Details length = %d, want 2", len(reason.Details))
	}
	if reason.CanOverride {
		t.Error("ExplainEscalation().CanOverride = true, want false for Failed")
	}
}

func TestExplainEscalation_Block(t *testing.T) {
	input := EscalationInput{
		VerificationPassed:  true,
		NeedsVisualApproval: true,
	}

	reason := ExplainEscalation(input)
	if reason.Level != EscalationBlock {
		t.Errorf("ExplainEscalation().Level = %v, want EscalationBlock", reason.Level)
	}
	if !reason.CanOverride {
		t.Error("ExplainEscalation().CanOverride = false, want true for Block")
	}
}

func TestExplainEscalation_KnowledgeProducing(t *testing.T) {
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "investigation",
		NextActions:        []string{"Create beads issue for follow-up"},
	}

	reason := ExplainEscalation(input)
	if reason.Level != EscalationReview {
		t.Errorf("ExplainEscalation().Level = %v, want EscalationReview", reason.Level)
	}
	if reason.Reason != "Knowledge-producing skill with recommendations" {
		t.Errorf("ExplainEscalation().Reason = %q, unexpected", reason.Reason)
	}
}

func TestExplainEscalation_CleanCompletion(t *testing.T) {
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "close",
	}

	reason := ExplainEscalation(input)
	if reason.Level != EscalationNone {
		t.Errorf("ExplainEscalation().Level = %v, want EscalationNone", reason.Level)
	}
	if reason.Reason != "Clean completion" {
		t.Errorf("ExplainEscalation().Reason = %q, want 'Clean completion'", reason.Reason)
	}
}

func TestDetermineEscalation_HyphenatedRecommendationFromParser(t *testing.T) {
	// End-to-end: parsed "spawn-follow-up" from synthesis should trigger escalation.
	// This is the bug reproduction: \w+ regex truncated "spawn-follow-up" to "spawn",
	// so hasSignificantRecommendations never matched.
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "spawn-follow-up",
		FileCount:          5,
	}

	got := DetermineEscalation(input)
	if got != EscalationInfo {
		t.Errorf("DetermineEscalation(recommendation='spawn-follow-up') = %v, want EscalationInfo", got)
	}

	// Verify hasSignificantRecommendations returns true
	if !hasSignificantRecommendations(input) {
		t.Error("hasSignificantRecommendations() = false for 'spawn-follow-up', want true")
	}
}

// --- Review Tier Escalation Tests ---

func TestCheckReviewTierEscalation_NoSignals(t *testing.T) {
	signals := ReviewTierEscalationSignals{}
	result := CheckReviewTierEscalation(signals, "auto")
	if result.Escalated {
		t.Error("expected no escalation with empty signals")
	}
	if result.EscalatedTier != "auto" {
		t.Errorf("expected tier auto, got %s", result.EscalatedTier)
	}
}

func TestCheckReviewTierEscalation_HotspotOnly(t *testing.T) {
	signals := ReviewTierEscalationSignals{HotspotMatchCount: 3}
	result := CheckReviewTierEscalation(signals, "auto")
	if !result.Escalated {
		t.Error("expected escalation with hotspot matches")
	}
	if result.EscalatedTier != "scan" {
		t.Errorf("expected tier scan, got %s", result.EscalatedTier)
	}
	if len(result.Reasons) != 1 {
		t.Errorf("expected 1 reason, got %d", len(result.Reasons))
	}
}

func TestCheckReviewTierEscalation_ArchChoicesOnly(t *testing.T) {
	signals := ReviewTierEscalationSignals{HasArchChoices: true}
	result := CheckReviewTierEscalation(signals, "scan")
	if !result.Escalated {
		t.Error("expected escalation with arch choices")
	}
	if result.EscalatedTier != "review" {
		t.Errorf("expected tier review, got %s", result.EscalatedTier)
	}
}

func TestCheckReviewTierEscalation_LargeDiffOnly(t *testing.T) {
	signals := ReviewTierEscalationSignals{DiffLineCount: 600}
	result := CheckReviewTierEscalation(signals, "auto")
	if !result.Escalated {
		t.Error("expected escalation with large diff")
	}
	if result.EscalatedTier != "scan" {
		t.Errorf("expected tier scan, got %s", result.EscalatedTier)
	}
}

func TestCheckReviewTierEscalation_DiffAtThreshold(t *testing.T) {
	// Exactly at threshold should NOT trigger
	signals := ReviewTierEscalationSignals{DiffLineCount: 500}
	result := CheckReviewTierEscalation(signals, "auto")
	if result.Escalated {
		t.Error("expected no escalation at exactly threshold")
	}
}

func TestCheckReviewTierEscalation_MultipleSignals(t *testing.T) {
	// Two signals should bump twice: auto → review
	signals := ReviewTierEscalationSignals{
		HotspotMatchCount: 1,
		DiffLineCount:     800,
	}
	result := CheckReviewTierEscalation(signals, "auto")
	if !result.Escalated {
		t.Error("expected escalation")
	}
	if result.EscalatedTier != "review" {
		t.Errorf("expected tier review (2 bumps from auto), got %s", result.EscalatedTier)
	}
	if len(result.Reasons) != 2 {
		t.Errorf("expected 2 reasons, got %d", len(result.Reasons))
	}
}

func TestCheckReviewTierEscalation_AllThreeSignals(t *testing.T) {
	// Three signals should bump three times: auto → deep
	signals := ReviewTierEscalationSignals{
		HotspotMatchCount: 2,
		HasArchChoices:    true,
		DiffLineCount:     1000,
	}
	result := CheckReviewTierEscalation(signals, "auto")
	if result.EscalatedTier != "deep" {
		t.Errorf("expected tier deep (3 bumps from auto), got %s", result.EscalatedTier)
	}
	if len(result.Reasons) != 3 {
		t.Errorf("expected 3 reasons, got %d", len(result.Reasons))
	}
}

func TestCheckReviewTierEscalation_CapsAtDeep(t *testing.T) {
	// Already at review + 3 signals should cap at deep (not overflow)
	signals := ReviewTierEscalationSignals{
		HotspotMatchCount: 1,
		HasArchChoices:    true,
		DiffLineCount:     999,
	}
	result := CheckReviewTierEscalation(signals, "review")
	if result.EscalatedTier != "deep" {
		t.Errorf("expected tier deep (capped), got %s", result.EscalatedTier)
	}
}

func TestCheckReviewTierEscalation_AlreadyDeep(t *testing.T) {
	// Already at deep, signals should not escalate further
	signals := ReviewTierEscalationSignals{HotspotMatchCount: 5}
	result := CheckReviewTierEscalation(signals, "deep")
	if result.Escalated {
		t.Error("expected no escalation when already at deep")
	}
	if result.EscalatedTier != "deep" {
		t.Errorf("expected tier deep, got %s", result.EscalatedTier)
	}
}

func TestCheckReviewTierEscalation_UnknownTierTreatedAsReview(t *testing.T) {
	signals := ReviewTierEscalationSignals{HasArchChoices: true}
	result := CheckReviewTierEscalation(signals, "unknown-tier")
	if result.EscalatedTier != "deep" {
		t.Errorf("expected unknown tier treated as review, bump to deep; got %s", result.EscalatedTier)
	}
}

func TestCheckReviewTierEscalation_PreservesOriginalTier(t *testing.T) {
	signals := ReviewTierEscalationSignals{HasArchChoices: true}
	result := CheckReviewTierEscalation(signals, "scan")
	if result.OriginalTier != "scan" {
		t.Errorf("expected original tier scan, got %s", result.OriginalTier)
	}
}

func TestParseShortstatLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"typical", " 5 files changed, 120 insertions(+), 30 deletions(-)", 150},
		{"insertions only", " 3 files changed, 200 insertions(+)", 200},
		{"deletions only", " 1 file changed, 50 deletions(-)", 50},
		{"empty", "", 0},
		{"large", " 20 files changed, 1500 insertions(+), 200 deletions(-)", 1700},
		{"singular", " 1 file changed, 1 insertion(+), 1 deletion(-)", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseShortstatLines(tt.input)
			if got != tt.expected {
				t.Errorf("parseShortstatLines(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// Test the decision tree order of precedence
func TestDetermineEscalation_Precedence(t *testing.T) {
	// Verification failure should override everything
	input := EscalationInput{
		VerificationPassed:  false,
		SkillName:           "investigation", // Would be Review
		NeedsVisualApproval: true,            // Would be Block
	}
	if got := DetermineEscalation(input); got != EscalationFailed {
		t.Errorf("Failed should override knowledge skill and visual: got %v", got)
	}

	// Knowledge skill should override visual approval check for non-feature-impl
	input = EscalationInput{
		VerificationPassed:  true,
		SkillName:           "investigation",
		NeedsVisualApproval: false, // Investigation wouldn't need this anyway
		NextActions:         []string{"follow up"},
	}
	if got := DetermineEscalation(input); got != EscalationReview {
		t.Errorf("Knowledge skill with recommendations should be Review: got %v", got)
	}
}

// --- Baseline-scoped diff tests ---

// setupTestGitRepo creates a temp dir with a git repo, makes N commits with unique files,
// and returns the repo dir plus the SHA of the commit at position `baselineAt` (0-indexed).
// Commits are named "commit-0", "commit-1", etc. Each adds a file "file-N.txt".
func setupTestGitRepo(t *testing.T, commitCount int, baselineAt int) (repoDir, baselineSHA string) {
	t.Helper()
	repoDir = t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init")

	for i := 0; i < commitCount; i++ {
		filePath := filepath.Join(repoDir, "file-"+string(rune('A'+i))+".txt")
		// Write multiple lines so shortstat shows insertions
		content := ""
		for j := 0; j < 50; j++ {
			content += "line " + string(rune('0'+j%10)) + "\n"
		}
		os.WriteFile(filePath, []byte(content), 0644)
		run("git", "add", "-A")
		run("git", "commit", "-m", "commit-"+string(rune('0'+i)))

		if i == baselineAt {
			cmd := exec.Command("git", "rev-parse", "HEAD")
			cmd.Dir = repoDir
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("git rev-parse failed: %v", err)
			}
			baselineSHA = string(out[:len(out)-1]) // trim newline
		}
	}

	return repoDir, baselineSHA
}

func TestCountGitDiffLines_WithBaseline(t *testing.T) {
	// Create 8 commits, baseline at commit 5 (0-indexed).
	// Only commits 6 and 7 should be counted with baseline (2 files × 50 lines = 100 insertions).
	// Without baseline: HEAD~5..HEAD sees commits 3-7 (5 files × 50 lines = 250 insertions).
	repoDir, baseline := setupTestGitRepo(t, 8, 5)

	// With baseline: should only see changes after commit 5
	withBaseline := countGitDiffLines(repoDir, baseline)

	// Without baseline: falls back to HEAD~5..HEAD, sees 5 commits
	withoutBaseline := countGitDiffLines(repoDir, "")

	if withBaseline >= withoutBaseline {
		t.Errorf("baseline-scoped diff (%d lines) should be less than unscoped (%d lines)",
			withBaseline, withoutBaseline)
	}

	if withBaseline == 0 {
		t.Error("baseline-scoped diff should show some changes (commits after baseline)")
	}
}

func TestCountRecentFileChanges_WithBaseline(t *testing.T) {
	// Create 8 commits, baseline at commit 5.
	// Only 2 files should show up (commits 6 and 7).
	// Without baseline: HEAD~5..HEAD sees 5 files.
	repoDir, baseline := setupTestGitRepo(t, 8, 5)

	withBaseline := countRecentFileChanges(repoDir, baseline)
	withoutBaseline := countRecentFileChanges(repoDir, "")

	if withBaseline >= withoutBaseline {
		t.Errorf("baseline-scoped file count (%d) should be less than unscoped (%d)",
			withBaseline, withoutBaseline)
	}

	// Should see exactly 2 files (commits 6 and 7 each add one file)
	if withBaseline != 2 {
		t.Errorf("expected 2 files changed since baseline, got %d", withBaseline)
	}
}

func TestCountGitDiffLines_EmptyBaseline_FallsBack(t *testing.T) {
	repoDir, _ := setupTestGitRepo(t, 3, 0)

	// Empty baseline should fall back to HEAD~5..HEAD (which works for 3 commits too)
	result := countGitDiffLines(repoDir, "")
	if result == 0 {
		t.Error("expected non-zero diff lines with fallback")
	}
}

func TestCountGitDiffLines_InvalidBaseline_FallsBack(t *testing.T) {
	repoDir, _ := setupTestGitRepo(t, 3, 0)

	// Invalid baseline should fall back gracefully
	result := countGitDiffLines(repoDir, "deadbeef00000000000000000000000000000000")
	if result == 0 {
		t.Error("expected non-zero diff lines after fallback from invalid baseline")
	}
}

func TestReadBaselineFromManifest(t *testing.T) {
	// Test with valid manifest
	tmpDir := t.TempDir()
	manifest := spawn.AgentManifest{
		GitBaseline: "abc123def456",
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(tmpDir, "AGENT_MANIFEST.json"), data, 0644)

	baseline := readBaselineFromManifest(tmpDir)
	if baseline != "abc123def456" {
		t.Errorf("expected baseline abc123def456, got %q", baseline)
	}

	// Test with empty workspace path
	baseline = readBaselineFromManifest("")
	if baseline != "" {
		t.Errorf("expected empty baseline for empty workspace, got %q", baseline)
	}

	// Test with missing manifest
	baseline = readBaselineFromManifest(t.TempDir())
	if baseline != "" {
		t.Errorf("expected empty baseline for missing manifest, got %q", baseline)
	}
}

func TestBuildEscalationSignals_UsesBaseline(t *testing.T) {
	// Create a git repo with 8 commits, baseline at commit 5.
	// Commits 6-7 are "this agent's" work. HEAD~5 sees commits 3-7.
	// With baseline, should see fewer lines than without.
	repoDir, baseline := setupTestGitRepo(t, 8, 5)

	// Create workspace with manifest containing the baseline
	workspaceDir := t.TempDir()
	manifest := spawn.AgentManifest{
		GitBaseline: baseline,
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(workspaceDir, "AGENT_MANIFEST.json"), data, 0644)

	// Build signals with workspace (should use baseline)
	signalsWithBaseline := BuildEscalationSignals(workspaceDir, repoDir)

	// Build signals without workspace (should fall back to HEAD~5)
	signalsWithout := BuildEscalationSignals("", repoDir)

	// Baseline-scoped: commits 6-7 = 2 files × 50 lines = 100 lines
	// Unscoped HEAD~5: commits 3-7 = 5 files × 50 lines = 250 lines
	if signalsWithBaseline.DiffLineCount >= signalsWithout.DiffLineCount {
		t.Errorf("baseline-scoped DiffLineCount (%d) should be less than unscoped (%d)",
			signalsWithBaseline.DiffLineCount, signalsWithout.DiffLineCount)
	}
}
