package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestEvaluateProofSpecGateMissingSpecAdvisory(t *testing.T) {
	workspace := t.TempDir()
	target := &CompletionTarget{
		WorkspacePath: workspace,
		BeadsID:       "orch-go-test1",
	}

	result := evaluateProofSpecGate(target)

	if len(result.errors) != 0 {
		t.Fatalf("expected no blocking errors, got: %v", result.errors)
	}
	if len(result.warnings) == 0 {
		t.Fatal("expected advisory warning for missing spec")
	}
	if !strings.Contains(strings.Join(result.warnings, " "), "missing") {
		t.Fatalf("expected missing spec warning, got: %v", result.warnings)
	}
	if !result.gateResult().Passed {
		t.Fatalf("expected gate to pass in Phase A advisory mode")
	}
	if got := result.digestComment(); got != "" {
		t.Fatalf("expected no digest comment when spec missing, got: %q", got)
	}
}

func TestEvaluateProofSpecGateExecutesMatchingTier(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".tier"), []byte("light\n"), 0644); err != nil {
		t.Fatalf("write .tier: %v", err)
	}

	spec := `version: 1
scope:
  beads_id: orch-go-test2
  workspace: ws-proofspec
  skill: feature-impl
verification:
  - id: light-cli
    method: cli_smoke
    tier: light
    command: "printf 'daemon_status ok'"
    expect:
      stdout_contains:
        - daemon_status
  - id: light-manual
    method: manual
    tier: light
    manual_steps:
      - Open dashboard
    expect:
      human_approval_required: true
  - id: full-only
    method: integration
    tier: full
    command: "exit 1"
    expect:
      exit_code: 1
`
	if err := os.WriteFile(filepath.Join(workspace, verify.VerificationSpecFileName), []byte(spec), 0644); err != nil {
		t.Fatalf("write VERIFICATION_SPEC.yaml: %v", err)
	}

	target := &CompletionTarget{
		WorkspacePath: workspace,
		BeadsID:       "orch-go-test2",
	}

	result := evaluateProofSpecGate(target)

	if len(result.errors) != 0 {
		t.Fatalf("expected no errors, got: %v", result.errors)
	}
	if result.executed != 1 || result.passed != 1 || result.failed != 0 {
		t.Fatalf("unexpected execution counts: executed=%d passed=%d failed=%d", result.executed, result.passed, result.failed)
	}
	if result.manualPending != 1 {
		t.Fatalf("expected manual pending count 1, got %d", result.manualPending)
	}
	if len(result.commandHash) != 64 {
		t.Fatalf("expected sha256 hash, got %q", result.commandHash)
	}
	if !strings.Contains(result.methodCounts, "light[cli_smoke=1,manual=1]") {
		t.Fatalf("unexpected method counts: %s", result.methodCounts)
	}
	if !strings.Contains(result.methodCounts, "full[integration=1]") {
		t.Fatalf("unexpected method counts: %s", result.methodCounts)
	}

	digest := result.digestComment()
	if !strings.Contains(digest, "status=pass") {
		t.Fatalf("expected pass digest, got: %s", digest)
	}
	if !strings.Contains(digest, "manual_pending=1") {
		t.Fatalf("expected manual pending in digest, got: %s", digest)
	}
}

func TestEvaluateProofSpecGateFailsOnStdoutMismatch(t *testing.T) {
	workspace := t.TempDir()

	spec := `version: 1
scope:
  beads_id: orch-go-test3
  workspace: ws-proofspec
  skill: feature-impl
verification:
  - id: bad-stdout
    method: static
    tier: full
    command: "printf 'hello'"
    expect:
      stdout_contains:
        - world
`
	if err := os.WriteFile(filepath.Join(workspace, verify.VerificationSpecFileName), []byte(spec), 0644); err != nil {
		t.Fatalf("write VERIFICATION_SPEC.yaml: %v", err)
	}

	target := &CompletionTarget{
		WorkspacePath: workspace,
		BeadsID:       "orch-go-test3",
	}

	result := evaluateProofSpecGate(target)

	if len(result.errors) == 0 {
		t.Fatal("expected blocking errors for stdout mismatch")
	}
	if result.failed != 1 {
		t.Fatalf("expected failed count 1, got %d", result.failed)
	}
	if result.gateResult().Passed {
		t.Fatal("expected proof spec gate to fail")
	}
}

func TestEvaluateProofSpecGateFailsOnMalformedCommandSyntax(t *testing.T) {
	workspace := t.TempDir()

	spec := `version: 1
scope:
  beads_id: orch-go-test-malformed
  workspace: ws-proofspec
  skill: feature-impl
verification:
  - id: bad-syntax
    method: cli_smoke
    tier: full
    command: "echo hi )"
    expect:
      exit_code: 0
`
	if err := os.WriteFile(filepath.Join(workspace, verify.VerificationSpecFileName), []byte(spec), 0644); err != nil {
		t.Fatalf("write VERIFICATION_SPEC.yaml: %v", err)
	}

	target := &CompletionTarget{
		WorkspacePath: workspace,
		BeadsID:       "orch-go-test-malformed",
	}

	result := evaluateProofSpecGate(target)

	if len(result.errors) == 0 {
		t.Fatal("expected blocking errors for malformed command syntax")
	}
	if result.executed != 0 {
		t.Fatalf("expected no commands executed when syntax is malformed, got executed=%d", result.executed)
	}
	if !strings.Contains(strings.Join(result.errors, " "), "command syntax invalid") {
		t.Fatalf("expected syntax validation error, got: %v", result.errors)
	}
	if result.gateResult().Passed {
		t.Fatal("expected proof spec gate to fail")
	}
}

func TestVerifyRegularAgentSkipVerificationSpecGate(t *testing.T) {
	origVerify := verifyCompletionFullFunc
	origEvaluator := proofSpecEvaluator
	origPoster := proofSpecDigestPoster
	t.Cleanup(func() {
		verifyCompletionFullFunc = origVerify
		proofSpecEvaluator = origEvaluator
		proofSpecDigestPoster = origPoster
	})

	verifyCompletionFullFunc = func(beadsID, workspacePath, projectDir, tier, serverURL string) (verify.VerificationResult, error) {
		return verify.VerificationResult{Passed: true, Skill: "feature-impl"}, nil
	}

	proofSpecEvaluator = func(target *CompletionTarget) proofSpecGateResult {
		return proofSpecGateResult{
			specParsed:    true,
			specVersion:   1,
			workspaceTier: verify.VerificationTierLight,
			methodCounts:  "light[cli_smoke=1]",
			commandHash:   "abc",
			failed:        1,
			errors:        []string{"verification_spec[test]: command failed"},
		}
	}
	proofSpecDigestPoster = func(beadsID, comment string) error { return nil }

	target := &CompletionTarget{
		BeadsID:         "orch-go-test4",
		AgentName:       "og-proofspec-test",
		WorkspacePath:   t.TempDir(),
		BeadsProjectDir: t.TempDir(),
	}

	outcome := &VerificationOutcome{Passed: true}
	if _, err := verifyRegularAgent(target, SkipConfig{}, outcome); err == nil {
		t.Fatal("expected failure without skip flag")
	}

	skip := SkipConfig{VerificationSpec: true, Reason: "Skip for rollout"}
	if _, err := verifyRegularAgent(target, skip, outcome); err != nil {
		t.Fatalf("expected skip-verification-spec to bypass gate, got error: %v", err)
	}
}
