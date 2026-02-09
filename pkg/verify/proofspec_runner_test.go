package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteProofSpecInWorkspacePassAndReplayMetadata(t *testing.T) {
	workspace := t.TempDir()

	spec := `version: 1
scope:
  beads_id: orch-go-abc1
  workspace: test-workspace
  skill: feature-impl
verification:
  - id: smoke
    method: cli_smoke
    tier: full
    command: "printf 'daemon_status ok'"
    cwd: "."
    expect:
      exit_code: 0
      stdout_contains:
        - daemon_status
`

	if err := os.WriteFile(filepath.Join(workspace, VerificationSpecFileName), []byte(spec), 0644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".tier"), []byte("full"), 0644); err != nil {
		t.Fatalf("write tier: %v", err)
	}

	result := ExecuteProofSpecInWorkspace(ProofSpecRunnerOptions{WorkspacePath: workspace})

	if result.Status != ProofStepStatusPass {
		t.Fatalf("status = %s, want %s", result.Status, ProofStepStatusPass)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(result.Steps))
	}
	if result.Steps[0].Status != ProofStepStatusPass {
		t.Fatalf("step status = %s, want %s", result.Steps[0].Status, ProofStepStatusPass)
	}
	if len(result.Replay.SpecHash) != 64 {
		t.Fatalf("spec hash length = %d, want 64", len(result.Replay.SpecHash))
	}
	if len(result.Replay.CommandsRun) != 1 {
		t.Fatalf("commands run len = %d, want 1", len(result.Replay.CommandsRun))
	}
	if len(result.Replay.ExpectationsChecked) < 2 {
		t.Fatalf("expectations checked len = %d, want at least 2", len(result.Replay.ExpectationsChecked))
	}
	if len(result.Replay.FailedStepIDs) != 0 {
		t.Fatalf("failed step ids len = %d, want 0", len(result.Replay.FailedStepIDs))
	}
}

func TestExecuteProofSpecInWorkspaceManualEntry(t *testing.T) {
	workspace := t.TempDir()

	spec := `version: 1
scope:
  beads_id: orch-go-abc1
  workspace: test-workspace
  skill: feature-impl
verification:
  - id: human-check
    method: manual
    tier: full
    manual_steps:
      - Open dashboard
    expect:
      human_approval_required: true
`

	if err := os.WriteFile(filepath.Join(workspace, VerificationSpecFileName), []byte(spec), 0644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	t.Run("pending without approval token", func(t *testing.T) {
		result := ExecuteProofSpecInWorkspace(ProofSpecRunnerOptions{WorkspacePath: workspace})
		if result.Status != ProofStepStatusPending {
			t.Fatalf("status = %s, want %s", result.Status, ProofStepStatusPending)
		}
		if len(result.Steps) != 1 || result.Steps[0].Status != ProofStepStatusPending {
			t.Fatalf("manual step status = %v, want pending", result.Steps)
		}
	})

	t.Run("passes with approval token", func(t *testing.T) {
		result := ExecuteProofSpecInWorkspace(ProofSpecRunnerOptions{WorkspacePath: workspace, HasManualApproval: true})
		if result.Status != ProofStepStatusPass {
			t.Fatalf("status = %s, want %s", result.Status, ProofStepStatusPass)
		}
		if len(result.Steps) != 1 || result.Steps[0].Status != ProofStepStatusPass {
			t.Fatalf("manual step status = %v, want pass", result.Steps)
		}
	})
}

func TestExecuteProofSpecInWorkspaceCommandExpectationFailure(t *testing.T) {
	workspace := t.TempDir()

	spec := `version: 1
scope:
  beads_id: orch-go-abc1
  workspace: test-workspace
  skill: feature-impl
verification:
  - id: bad-smoke
    method: cli_smoke
    tier: full
    command: "printf 'hello'"
    expect:
      exit_code: 0
      stdout_contains:
        - daemon_status
`

	if err := os.WriteFile(filepath.Join(workspace, VerificationSpecFileName), []byte(spec), 0644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	result := ExecuteProofSpecInWorkspace(ProofSpecRunnerOptions{WorkspacePath: workspace})

	if result.Status != ProofStepStatusFail {
		t.Fatalf("status = %s, want %s", result.Status, ProofStepStatusFail)
	}
	if len(result.Replay.FailedStepIDs) != 1 || result.Replay.FailedStepIDs[0] != "bad-smoke" {
		t.Fatalf("failed step ids = %v, want [bad-smoke]", result.Replay.FailedStepIDs)
	}
	if len(result.Steps) != 1 || !strings.Contains(result.Steps[0].Error, "stdout missing tokens") {
		t.Fatalf("step error = %q, want stdout token failure", result.Steps[0].Error)
	}
}

func TestExecuteProofSpecInWorkspaceTierMismatchIsSkipped(t *testing.T) {
	workspace := t.TempDir()

	spec := `version: 1
scope:
  beads_id: orch-go-abc1
  workspace: test-workspace
  skill: feature-impl
verification:
  - id: full-only
    method: cli_smoke
    tier: full
    command: "printf 'ok'"
    expect:
      exit_code: 0
`

	if err := os.WriteFile(filepath.Join(workspace, VerificationSpecFileName), []byte(spec), 0644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".tier"), []byte("light"), 0644); err != nil {
		t.Fatalf("write tier: %v", err)
	}

	result := ExecuteProofSpecInWorkspace(ProofSpecRunnerOptions{WorkspacePath: workspace})

	if result.Status != ProofStepStatusSkipped {
		t.Fatalf("status = %s, want %s", result.Status, ProofStepStatusSkipped)
	}
	if len(result.Steps) != 1 || result.Steps[0].Status != ProofStepStatusSkipped {
		t.Fatalf("step status = %v, want skipped", result.Steps)
	}
}

func TestExecuteProofSpecInWorkspaceRejectsEscapingCWD(t *testing.T) {
	workspace := t.TempDir()

	spec := `version: 1
scope:
  beads_id: orch-go-abc1
  workspace: test-workspace
  skill: feature-impl
verification:
  - id: bad-cwd
    method: cli_smoke
    tier: full
    command: "printf 'ok'"
    cwd: ".."
    expect:
      exit_code: 0
`

	if err := os.WriteFile(filepath.Join(workspace, VerificationSpecFileName), []byte(spec), 0644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	result := ExecuteProofSpecInWorkspace(ProofSpecRunnerOptions{WorkspacePath: workspace})

	if result.Status != ProofStepStatusFail {
		t.Fatalf("status = %s, want %s", result.Status, ProofStepStatusFail)
	}
	if len(result.Steps) != 1 || !strings.Contains(result.Steps[0].Error, "escapes workspace") {
		t.Fatalf("step error = %q, want cwd escape error", result.Steps[0].Error)
	}
}
