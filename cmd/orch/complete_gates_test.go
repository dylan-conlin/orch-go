package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	statedb "github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestRequiresDesignDecomposition(t *testing.T) {
	tests := []struct {
		name  string
		skill string
		want  bool
	}{
		{name: "design session", skill: "design-session", want: true},
		{name: "architect", skill: "architect", want: true},
		{name: "mixed case", skill: "  ArChItEcT  ", want: true},
		{name: "feature impl", skill: "feature-impl", want: false},
		{name: "empty", skill: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiresDesignDecomposition(tt.skill)
			if got != tt.want {
				t.Fatalf("requiresDesignDecomposition(%q) = %v, want %v", tt.skill, got, tt.want)
			}
		})
	}
}

func TestBuildDesignDecompositionIssueTitle(t *testing.T) {
	item := verify.DesignActionItem{Section: "Components to Build", Text: "`WorkInProgressSection`"}
	title := buildDesignDecompositionIssueTitle(item)

	if !strings.HasPrefix(title, "Design follow-up: ") {
		t.Fatalf("unexpected prefix in title: %q", title)
	}
	if strings.Contains(title, "`") {
		t.Fatalf("title should strip markdown backticks, got %q", title)
	}

	longText := strings.Repeat("a", 200)
	longTitle := buildDesignDecompositionIssueTitle(verify.DesignActionItem{Text: longText})
	if len(longTitle) > 140 {
		t.Fatalf("title length should be capped at 140, got %d", len(longTitle))
	}
}

func TestShouldRetryVerification(t *testing.T) {
	tests := []struct {
		name   string
		result verify.VerificationResult
		err    error
		want   bool
	}{
		{
			name: "retry on transient gate failure",
			result: verify.VerificationResult{
				Passed:      false,
				GatesFailed: []string{verify.GateDashboardHealth},
			},
			want: true,
		},
		{
			name: "retry on transient message",
			result: verify.VerificationResult{
				Passed: false,
				Errors: []string{"failed to connect to /api/beads/graph"},
			},
			want: true,
		},
		{
			name: "retry on transient error",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "no retry on non-transient gate",
			result: verify.VerificationResult{
				Passed:      false,
				GatesFailed: []string{verify.GateBuild},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := shouldRetryVerification(tt.result, tt.err)
			if got != tt.want {
				t.Fatalf("shouldRetryVerification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyRegularAgentRetriesTransientGateFailure(t *testing.T) {
	origVerify := verifyCompletionFullFunc
	origSleep := verificationRetrySleep
	t.Cleanup(func() {
		verifyCompletionFullFunc = origVerify
		verificationRetrySleep = origSleep
	})

	verifyCalls := 0
	verifyCompletionFullFunc = func(beadsID, workspacePath, projectDir, tier, serverURL string) (verify.VerificationResult, error) {
		verifyCalls++
		if verifyCalls == 1 {
			return verify.VerificationResult{
				Passed:      false,
				Skill:       "systematic-debugging",
				GatesFailed: []string{verify.GateDashboardHealth},
				Errors:      []string{"Dashboard health check failed: failed to connect"},
			}, nil
		}
		return verify.VerificationResult{Passed: true, Skill: "systematic-debugging"}, nil
	}

	sleepCalls := 0
	verificationRetrySleep = func(d time.Duration) {
		sleepCalls++
		if d != transientVerificationRetryDelay {
			t.Fatalf("retry delay = %v, want %v", d, transientVerificationRetryDelay)
		}
	}

	target := &CompletionTarget{
		AgentName:       "og-debug-test",
		WorkspacePath:   "/tmp/workspace",
		BeadsProjectDir: "/tmp/project",
	}
	outcome := &VerificationOutcome{Passed: true}

	_, err := verifyRegularAgent(target, SkipConfig{}, outcome)
	if err != nil {
		t.Fatalf("verifyRegularAgent() unexpected error: %v", err)
	}
	if verifyCalls != 2 {
		t.Fatalf("verify calls = %d, want 2", verifyCalls)
	}
	if sleepCalls != 1 {
		t.Fatalf("sleep calls = %d, want 1", sleepCalls)
	}
}

func TestVerifyRegularAgentNoRetryOnNonTransientFailure(t *testing.T) {
	origVerify := verifyCompletionFullFunc
	origSleep := verificationRetrySleep
	t.Cleanup(func() {
		verifyCompletionFullFunc = origVerify
		verificationRetrySleep = origSleep
	})

	verifyCalls := 0
	verifyCompletionFullFunc = func(beadsID, workspacePath, projectDir, tier, serverURL string) (verify.VerificationResult, error) {
		verifyCalls++
		return verify.VerificationResult{
			Passed:      false,
			Skill:       "systematic-debugging",
			GatesFailed: []string{verify.GateBuild},
			Errors:      []string{"build failed"},
		}, nil
	}

	sleepCalls := 0
	verificationRetrySleep = func(time.Duration) { sleepCalls++ }

	target := &CompletionTarget{
		AgentName:       "og-debug-test",
		WorkspacePath:   "/tmp/workspace",
		BeadsProjectDir: "/tmp/project",
	}
	outcome := &VerificationOutcome{Passed: true}

	_, err := verifyRegularAgent(target, SkipConfig{}, outcome)
	if err == nil {
		t.Fatal("verifyRegularAgent() expected error, got nil")
	}
	if verifyCalls != 1 {
		t.Fatalf("verify calls = %d, want 1", verifyCalls)
	}
	if sleepCalls != 0 {
		t.Fatalf("sleep calls = %d, want 0", sleepCalls)
	}
}

func TestVerifyRegularAgentUsesGitWorktreeDir(t *testing.T) {
	origVerify := verifyCompletionFullFunc
	t.Cleanup(func() {
		verifyCompletionFullFunc = origVerify
	})

	projectDir := ""
	verifyCompletionFullFunc = func(beadsID, workspacePath, dir, tier, serverURL string) (verify.VerificationResult, error) {
		projectDir = dir
		return verify.VerificationResult{Passed: true, Skill: "feature-impl"}, nil
	}

	target := &CompletionTarget{
		BeadsID:         "orch-go-test",
		AgentName:       "og-feat-test",
		WorkspacePath:   "/tmp/workspace",
		BeadsProjectDir: "/tmp/source",
		GitWorktreeDir:  "/tmp/worktree",
	}

	_, err := verifyRegularAgent(target, SkipConfig{}, &VerificationOutcome{Passed: true})
	if err != nil {
		t.Fatalf("verifyRegularAgent() unexpected error: %v", err)
	}
	if projectDir != "/tmp/worktree" {
		t.Fatalf("verifyCompletionFull projectDir = %q, want %q", projectDir, "/tmp/worktree")
	}
}

func TestCheckLivenessSkipAgentRunning(t *testing.T) {
	origGetLiveness := getLiveness
	t.Cleanup(func() {
		getLiveness = origGetLiveness
	})

	called := 0
	getLiveness = func(beadsID, serverURL, projectDir string) statedb.LivenessResult {
		called++
		return statedb.LivenessResult{TmuxLive: true}
	}

	target := &CompletionTarget{
		BeadsID:         "orch-go-test",
		AgentName:       "og-test-liveness",
		BeadsProjectDir: "/tmp/project",
	}

	err := checkLiveness(target, SkipConfig{AgentRunning: true, Reason: "GPT model liveness bypass"})
	if err != nil {
		t.Fatalf("checkLiveness() unexpected error: %v", err)
	}
	if called != 0 {
		t.Fatalf("expected liveness lookup to be skipped, got %d call(s)", called)
	}
}

func TestCheckCommitEvidenceNoBranch(t *testing.T) {
	// No git branch = gate not applicable, should pass with warning
	target := &CompletionTarget{
		BeadsID:   "orch-go-test",
		AgentName: "og-test-no-branch",
	}
	result := checkCommitEvidence(target)
	if !result.passed {
		t.Fatal("expected pass for target without GitBranch")
	}
	if len(result.warnings) == 0 {
		t.Fatal("expected warning about skipped gate")
	}
}

func TestCheckCommitEvidenceNilTarget(t *testing.T) {
	result := checkCommitEvidence(nil)
	if !result.passed {
		t.Fatal("expected pass for nil target")
	}
}

func TestCheckCommitEvidenceZeroCommits(t *testing.T) {
	// Set up a real git repo with an agent branch that has zero commits
	tmpDir := t.TempDir()

	// Init repo, create a commit on master
	cmds := [][]string{
		{"git", "-C", tmpDir, "init"},
		{"git", "-C", tmpDir, "config", "user.email", "test@test.com"},
		{"git", "-C", tmpDir, "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v failed: %v\n%s", args, err, out)
		}
	}

	// Create initial commit
	testFile := tmpDir + "/main.go"
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	setupCmds := [][]string{
		{"git", "-C", tmpDir, "add", "."},
		{"git", "-C", tmpDir, "commit", "-m", "initial"},
		{"git", "-C", tmpDir, "branch", "agent/test-zero"},
	}
	for _, args := range setupCmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v failed: %v\n%s", args, err, out)
		}
	}
	// SourceProjectDir stays on master; GitWorktreeDir is the same repo
	// (in production these are separate dirs, but the git state is the same)
	target := &CompletionTarget{
		BeadsID:          "orch-go-test",
		AgentName:        "og-test-zero-commits",
		GitBranch:        "agent/test-zero",
		GitWorktreeDir:   tmpDir,
		SourceProjectDir: tmpDir,
	}

	result := checkCommitEvidence(target)
	if result.passed {
		t.Fatal("expected fail for branch with zero commits")
	}
	if len(result.errors) == 0 {
		t.Fatal("expected error message about ghost completion")
	}
	found := false
	for _, e := range result.errors {
		if strings.Contains(e, "0 commits") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected error mentioning '0 commits', got: %v", result.errors)
	}
}

func TestCheckCommitEvidenceWithCommits(t *testing.T) {
	// Set up a real git repo with an agent branch that has commits
	tmpDir := t.TempDir()

	cmds := [][]string{
		{"git", "-C", tmpDir, "init"},
		{"git", "-C", tmpDir, "config", "user.email", "test@test.com"},
		{"git", "-C", tmpDir, "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v failed: %v\n%s", args, err, out)
		}
	}

	// Create initial commit on master
	testFile := tmpDir + "/main.go"
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	setupCmds := [][]string{
		{"git", "-C", tmpDir, "add", "."},
		{"git", "-C", tmpDir, "commit", "-m", "initial"},
		{"git", "-C", tmpDir, "checkout", "-b", "agent/test-with-commits"},
	}
	for _, args := range setupCmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v failed: %v\n%s", args, err, out)
		}
	}

	// Add a commit on the agent branch
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commitCmds := [][]string{
		{"git", "-C", tmpDir, "add", "."},
		{"git", "-C", tmpDir, "commit", "-m", "agent work"},
	}
	for _, args := range commitCmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("commit %v failed: %v\n%s", args, err, out)
		}
	}

	target := &CompletionTarget{
		BeadsID:          "orch-go-test",
		AgentName:        "og-test-with-commits",
		GitBranch:        "agent/test-with-commits",
		GitWorktreeDir:   tmpDir,
		SourceProjectDir: tmpDir,
	}

	result := checkCommitEvidence(target)
	if !result.passed {
		t.Fatalf("expected pass for branch with commits, got errors: %v", result.errors)
	}
}

func TestCheckLivenessNonInteractiveSuggestsSkipFlag(t *testing.T) {
	origGetLiveness := getLiveness
	t.Cleanup(func() {
		getLiveness = origGetLiveness
	})

	getLiveness = func(beadsID, serverURL, projectDir string) statedb.LivenessResult {
		return statedb.LivenessResult{TmuxLive: true}
	}

	target := &CompletionTarget{
		AgentName:       "og-test-liveness",
		BeadsProjectDir: "/tmp/project",
	}

	err := checkLiveness(target, SkipConfig{})
	if err == nil {
		t.Fatal("checkLiveness() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--skip-agent-running") {
		t.Fatalf("expected error to suggest --skip-agent-running, got: %v", err)
	}
}
