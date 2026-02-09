package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseProofSpecYAML(t *testing.T) {
	t.Run("parses valid spec and applies defaults", func(t *testing.T) {
		specYAML := `version: 1
scope:
  beads_id: orch-go-21492
  workspace: og-proofspec-demo
  skill: feature-impl
verification:
  - id: cli-health
    method: cli_smoke
    tier: light
    command: "orch health --json"
    cwd: "."
    timeout_seconds: 30
    expect:
      stdout_contains:
        - daemon_status
  - id: human-check
    method: manual
    tier: full
    manual_steps:
      - Open dashboard in browser
      - Confirm cards update in under 2s
    expect:
      human_approval_required: true
`

		spec, err := ParseProofSpecYAML([]byte(specYAML))
		if err != nil {
			t.Fatalf("ParseProofSpecYAML failed: %v", err)
		}

		if spec.Version != 1 {
			t.Fatalf("Version = %d, want 1", spec.Version)
		}

		if spec.Scope.BeadsID != "orch-go-21492" {
			t.Errorf("Scope.BeadsID = %q, want %q", spec.Scope.BeadsID, "orch-go-21492")
		}

		if len(spec.Verification) != 2 {
			t.Fatalf("Verification len = %d, want 2", len(spec.Verification))
		}

		if spec.Verification[0].Expect.ExitCode != 0 {
			t.Errorf("command method default exit code = %d, want 0", spec.Verification[0].Expect.ExitCode)
		}

		if spec.Verification[0].CWD != "." {
			t.Errorf("Verification[0].CWD = %q, want %q", spec.Verification[0].CWD, ".")
		}

		if !spec.Verification[1].Expect.HumanApprovalRequired {
			t.Error("manual method should preserve expect.human_approval_required=true")
		}
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		specYAML := `version: 1
scope:
  beads_id: orch-go-21492
  workspace: og-proofspec-demo
  skill: feature-impl
unexpected_field: true
verification:
  - id: cli-health
    method: cli_smoke
    tier: light
    command: "orch health --json"
`

		_, err := ParseProofSpecYAML([]byte(specYAML))
		if err == nil {
			t.Fatal("expected error for unknown field, got nil")
		}
		if !strings.Contains(err.Error(), "field unexpected_field not found") {
			t.Fatalf("expected unknown field error, got: %v", err)
		}
	})

	t.Run("rejects manual method without steps", func(t *testing.T) {
		specYAML := `version: 1
scope:
  beads_id: orch-go-21492
  workspace: og-proofspec-demo
  skill: feature-impl
verification:
  - id: human-check
    method: manual
    tier: full
    expect:
      human_approval_required: true
`

		_, err := ParseProofSpecYAML([]byte(specYAML))
		if err == nil {
			t.Fatal("expected error for manual method without steps, got nil")
		}
		if !strings.Contains(err.Error(), "manual_steps is required") {
			t.Fatalf("expected manual_steps error, got: %v", err)
		}
	})

	t.Run("rejects non-manual method without command", func(t *testing.T) {
		specYAML := `version: 1
scope:
  beads_id: orch-go-21492
  workspace: og-proofspec-demo
  skill: feature-impl
verification:
  - id: run-tests
    method: integration
    tier: full
`

		_, err := ParseProofSpecYAML([]byte(specYAML))
		if err == nil {
			t.Fatal("expected error for non-manual method without command, got nil")
		}
		if !strings.Contains(err.Error(), "command is required") {
			t.Fatalf("expected command required error, got: %v", err)
		}
	})

	t.Run("rejects duplicate verification ids", func(t *testing.T) {
		specYAML := `version: 1
scope:
  beads_id: orch-go-21492
  workspace: og-proofspec-demo
  skill: feature-impl
verification:
  - id: duplicate
    method: cli_smoke
    tier: light
    command: "orch health --json"
  - id: duplicate
    method: static
    tier: light
    command: "go test ./..."
`

		_, err := ParseProofSpecYAML([]byte(specYAML))
		if err == nil {
			t.Fatal("expected duplicate id error, got nil")
		}
		if !strings.Contains(err.Error(), "duplicated") {
			t.Fatalf("expected duplicate id error, got: %v", err)
		}
	})

	t.Run("requires human approval for manual method", func(t *testing.T) {
		specYAML := `version: 1
scope:
  beads_id: orch-go-21492
  workspace: og-proofspec-demo
  skill: feature-impl
verification:
  - id: human-check
    method: manual
    tier: full
    manual_steps:
      - Open dashboard in browser
`

		_, err := ParseProofSpecYAML([]byte(specYAML))
		if err == nil {
			t.Fatal("expected manual human approval error, got nil")
		}
		if !strings.Contains(err.Error(), "human_approval_required") {
			t.Fatalf("expected human approval error, got: %v", err)
		}
	})
}

func TestParseProofSpecFileAndLoadProofSpec(t *testing.T) {
	workspace := t.TempDir()

	contents := `version: 1
scope:
  beads_id: orch-go-21492
  workspace: og-proofspec-demo
  skill: feature-impl
verification:
  - id: cli-health
    method: cli_smoke
    tier: light
    command: "orch health --json"
`

	filePath := filepath.Join(workspace, VerificationSpecFileName)
	if err := os.WriteFile(filePath, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fromFile, err := ParseProofSpecFile(filePath)
	if err != nil {
		t.Fatalf("ParseProofSpecFile failed: %v", err)
	}

	fromWorkspace, err := LoadProofSpec(workspace)
	if err != nil {
		t.Fatalf("LoadProofSpec failed: %v", err)
	}

	if fromFile.Scope.Workspace != fromWorkspace.Scope.Workspace {
		t.Errorf("workspace scope mismatch: file=%q load=%q", fromFile.Scope.Workspace, fromWorkspace.Scope.Workspace)
	}
}

func TestValidateProofSpecCommandSyntax(t *testing.T) {
	t.Run("accepts valid commands", func(t *testing.T) {
		specYAML := `version: 1
scope:
  beads_id: orch-go-valid
  workspace: og-proofspec-valid
  skill: feature-impl
verification:
  - id: build
    method: cli_smoke
    tier: light
    command: "echo ok && exit 0"
`

		spec, err := ParseProofSpecYAML([]byte(specYAML))
		if err != nil {
			t.Fatalf("ParseProofSpecYAML failed: %v", err)
		}

		if err := ValidateProofSpecCommandSyntax(spec); err != nil {
			t.Fatalf("expected valid command syntax, got error: %v", err)
		}
	})

	t.Run("rejects malformed bash", func(t *testing.T) {
		specYAML := `version: 1
scope:
  beads_id: orch-go-invalid
  workspace: og-proofspec-invalid
  skill: feature-impl
verification:
  - id: build
    method: cli_smoke
    tier: light
    command: "echo ok )"
`

		spec, err := ParseProofSpecYAML([]byte(specYAML))
		if err != nil {
			t.Fatalf("ParseProofSpecYAML failed: %v", err)
		}

		err = ValidateProofSpecCommandSyntax(spec)
		if err == nil {
			t.Fatal("expected malformed command syntax error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid bash syntax") {
			t.Fatalf("expected invalid bash syntax error, got: %v", err)
		}
	})
}
