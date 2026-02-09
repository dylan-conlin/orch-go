package main

import (
	"errors"
	"strings"
	"testing"
	"time"

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
