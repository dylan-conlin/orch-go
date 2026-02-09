package verify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// VerificationCWDToken represents a reserved cwd token in VERIFICATION_SPEC.yaml.
type VerificationCWDToken string

const (
	// RuntimeCWDToken resolves to the agent runtime directory from AGENT_MANIFEST.json.
	// This lets proof-spec commands execute against the git worktree while keeping
	// relative cwd values confined to the workspace.
	RuntimeCWDToken VerificationCWDToken = "$GIT_WORKTREE_DIR"
)

// ProofStepStatus is the execution outcome for a proof spec step.
type ProofStepStatus string

const (
	ProofStepStatusPass    ProofStepStatus = "pass"
	ProofStepStatusFail    ProofStepStatus = "fail"
	ProofStepStatusPending ProofStepStatus = "pending"
	ProofStepStatusSkipped ProofStepStatus = "skipped"
)

// ProofStepResult captures execution details for a single proof spec entry.
type ProofStepResult struct {
	ID               string             `json:"id"`
	Method           VerificationMethod `json:"method"`
	Tier             VerificationTier   `json:"tier"`
	Status           ProofStepStatus    `json:"status"`
	Command          string             `json:"command,omitempty"`
	CWD              string             `json:"cwd,omitempty"`
	ExpectedExitCode int                `json:"expected_exit_code,omitempty"`
	ActualExitCode   int                `json:"actual_exit_code,omitempty"`
	Error            string             `json:"error,omitempty"`
}

// ProofReplayMetadata provides deterministic replay metadata for an execution.
type ProofReplayMetadata struct {
	SpecHash            string   `json:"spec_hash"`
	CommandsRun         []string `json:"commands_run"`
	ExpectationsChecked []string `json:"expectations_checked"`
	FailedStepIDs       []string `json:"failed_step_ids"`
}

// ProofSpecExecutionResult is the full execution report for one workspace spec.
type ProofSpecExecutionResult struct {
	WorkspacePath string              `json:"workspace_path"`
	WorkspaceTier VerificationTier    `json:"workspace_tier"`
	BeadsID       string              `json:"beads_id,omitempty"`
	Status        ProofStepStatus     `json:"status"`
	Error         string              `json:"error,omitempty"`
	Steps         []ProofStepResult   `json:"steps"`
	Replay        ProofReplayMetadata `json:"replay"`
}

// ProofCommandOutcome is the result of invoking a proof spec command.
type ProofCommandOutcome struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// ProofCommandExecutor runs a proof spec command.
type ProofCommandExecutor func(ctx context.Context, command, cwd string, env []string) ProofCommandOutcome

// ProofSpecRunnerOptions controls proof spec execution behavior.
type ProofSpecRunnerOptions struct {
	WorkspacePath      string
	WorkspaceTier      VerificationTier
	HasManualApproval  bool
	Env                []string
	CommandExecutor    ProofCommandExecutor
	DefaultTimeoutSecs int
}

// ExecuteProofSpecInWorkspace loads VERIFICATION_SPEC.yaml and executes applicable entries.
func ExecuteProofSpecInWorkspace(opts ProofSpecRunnerOptions) ProofSpecExecutionResult {
	workspacePath := strings.TrimSpace(opts.WorkspacePath)
	result := ProofSpecExecutionResult{WorkspacePath: workspacePath, Status: ProofStepStatusFail}

	if workspacePath == "" {
		result.Error = "workspace path is required"
		return result
	}

	tier := normalizeWorkspaceTier(opts.WorkspaceTier, workspacePath)
	result.WorkspaceTier = tier

	specPath := filepath.Join(workspacePath, VerificationSpecFileName)
	rawSpec, err := os.ReadFile(specPath)
	if err != nil {
		result.Error = fmt.Sprintf("read verification spec: %v", err)
		return result
	}

	hash := sha256.Sum256(rawSpec)
	result.Replay.SpecHash = hex.EncodeToString(hash[:])

	spec, err := ParseProofSpecYAML(rawSpec)
	if err != nil {
		result.Error = fmt.Sprintf("parse verification spec: %v", err)
		return result
	}
	result.BeadsID = spec.Scope.BeadsID

	executor := opts.CommandExecutor
	if executor == nil {
		executor = defaultProofCommandExecutor
	}

	defaultTimeout := opts.DefaultTimeoutSecs
	if defaultTimeout <= 0 {
		defaultTimeout = 120
	}

	hasApplicable := false
	for _, entry := range spec.Verification {
		step := ProofStepResult{
			ID:               entry.ID,
			Method:           entry.Method,
			Tier:             entry.Tier,
			ExpectedExitCode: entry.Expect.ExitCode,
		}

		if !entryAppliesToWorkspaceTier(entry.Tier, tier) {
			step.Status = ProofStepStatusSkipped
			result.Steps = append(result.Steps, step)
			continue
		}

		hasApplicable = true

		if entry.Method == VerificationMethodManual {
			result.Replay.ExpectationsChecked = append(result.Replay.ExpectationsChecked,
				fmt.Sprintf("%s: human_approval_required==%t", entry.ID, entry.Expect.HumanApprovalRequired),
			)
			if opts.HasManualApproval {
				step.Status = ProofStepStatusPass
			} else {
				step.Status = ProofStepStatusPending
			}
			result.Steps = append(result.Steps, step)
			continue
		}

		runCWD, cwdErr := ResolveProofStepCWD(workspacePath, entry.CWD)
		if cwdErr != nil {
			step.Status = ProofStepStatusFail
			step.Error = cwdErr.Error()
			result.Steps = append(result.Steps, step)
			result.Replay.FailedStepIDs = append(result.Replay.FailedStepIDs, entry.ID)
			continue
		}

		timeout := defaultTimeout
		if entry.TimeoutSeconds > 0 {
			timeout = entry.TimeoutSeconds
		}

		ctx, cancel := context.WithTimeout(context.Background(), secondsToDuration(timeout))
		outcome := executor(ctx, entry.Command, runCWD, opts.Env)
		cancel()

		step.Command = entry.Command
		step.CWD = runCWD
		step.ActualExitCode = outcome.ExitCode

		result.Replay.CommandsRun = append(result.Replay.CommandsRun, fmt.Sprintf("%s :: %s", runCWD, entry.Command))
		result.Replay.ExpectationsChecked = append(result.Replay.ExpectationsChecked,
			fmt.Sprintf("%s: exit_code==%d", entry.ID, entry.Expect.ExitCode),
		)
		for _, token := range entry.Expect.StdoutContains {
			result.Replay.ExpectationsChecked = append(result.Replay.ExpectationsChecked,
				fmt.Sprintf("%s: stdout_contains[%q]", entry.ID, token),
			)
		}

		missingTokens := missingStdoutTokens(outcome.Stdout, entry.Expect.StdoutContains)
		if outcome.Err != nil || outcome.ExitCode != entry.Expect.ExitCode || len(missingTokens) > 0 {
			step.Status = ProofStepStatusFail
			step.Error = formatProofStepError(outcome, entry.Expect.ExitCode, missingTokens)
			result.Replay.FailedStepIDs = append(result.Replay.FailedStepIDs, entry.ID)
		} else {
			step.Status = ProofStepStatusPass
		}

		result.Steps = append(result.Steps, step)
	}

	if !hasApplicable {
		result.Status = ProofStepStatusSkipped
		return result
	}

	result.Status = summarizeProofStatus(result.Steps)
	return result
}

func normalizeWorkspaceTier(tier VerificationTier, workspacePath string) VerificationTier {
	if tier == "" {
		tier = VerificationTier(ReadTierFromWorkspace(workspacePath))
	}
	switch tier {
	case VerificationTierLight, VerificationTierFull, VerificationTierOrchestrator:
		return tier
	default:
		return VerificationTierFull
	}
}

func entryAppliesToWorkspaceTier(stepTier, workspaceTier VerificationTier) bool {
	switch workspaceTier {
	case VerificationTierLight:
		return stepTier == VerificationTierLight
	case VerificationTierFull:
		return stepTier == VerificationTierLight || stepTier == VerificationTierFull
	case VerificationTierOrchestrator:
		return stepTier == VerificationTierOrchestrator
	default:
		return stepTier == workspaceTier
	}
}

// ResolveProofStepCWD resolves a proof step cwd relative to workspacePath.
// Relative paths remain workspace-confined; RuntimeCWDToken is resolved from
// AGENT_MANIFEST.json to target the actual runtime/worktree directory.
func ResolveProofStepCWD(workspacePath, specCWD string) (string, error) {
	base := filepath.Clean(workspacePath)
	if base == "" {
		return "", fmt.Errorf("workspace path is required")
	}

	raw := strings.TrimSpace(specCWD)
	if raw == string(RuntimeCWDToken) {
		runtimeDir, err := resolveRuntimeDirFromManifest(base)
		if err != nil {
			return "", err
		}
		return validateResolvedCWD(runtimeDir)
	}

	if specCWD == "" || specCWD == "." {
		return validateResolvedCWD(base)
	}

	var resolved string
	if filepath.IsAbs(specCWD) {
		resolved = filepath.Clean(specCWD)
	} else {
		resolved = filepath.Clean(filepath.Join(base, specCWD))
	}

	rel, err := filepath.Rel(base, resolved)
	if err != nil {
		return "", fmt.Errorf("resolve cwd: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("cwd %q escapes workspace %q", specCWD, workspacePath)
	}

	return validateResolvedCWD(resolved)
}

func resolveRuntimeDirFromManifest(workspacePath string) (string, error) {
	manifestPath := filepath.Join(workspacePath, "AGENT_MANIFEST.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("cwd %q requires %s: %w", RuntimeCWDToken, manifestPath, err)
	}

	var manifest struct {
		GitWorktreeDir   string `json:"git_worktree_dir"`
		ProjectDir       string `json:"project_dir"`
		SourceProjectDir string `json:"source_project_dir"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("cwd %q invalid manifest %s: %w", RuntimeCWDToken, manifestPath, err)
	}

	runtimeDir := strings.TrimSpace(manifest.GitWorktreeDir)
	if runtimeDir == "" {
		runtimeDir = strings.TrimSpace(manifest.ProjectDir)
	}
	if runtimeDir == "" {
		runtimeDir = strings.TrimSpace(manifest.SourceProjectDir)
	}
	if runtimeDir == "" {
		return "", fmt.Errorf("cwd %q missing git_worktree_dir/project_dir in %s", RuntimeCWDToken, manifestPath)
	}

	return filepath.Clean(runtimeDir), nil
}

func validateResolvedCWD(resolved string) (string, error) {
	if stat, err := os.Stat(resolved); err != nil || !stat.IsDir() {
		if err != nil {
			return "", fmt.Errorf("cwd %q does not exist: %w", resolved, err)
		}
		return "", fmt.Errorf("cwd %q is not a directory", resolved)
	}

	return resolved, nil
}

func summarizeProofStatus(steps []ProofStepResult) ProofStepStatus {
	hasPass := false
	hasPending := false
	hasSkipped := false

	for _, step := range steps {
		switch step.Status {
		case ProofStepStatusFail:
			return ProofStepStatusFail
		case ProofStepStatusPending:
			hasPending = true
		case ProofStepStatusPass:
			hasPass = true
		case ProofStepStatusSkipped:
			hasSkipped = true
		}
	}

	if hasPending {
		return ProofStepStatusPending
	}
	if hasPass {
		return ProofStepStatusPass
	}
	if hasSkipped {
		return ProofStepStatusSkipped
	}
	return ProofStepStatusSkipped
}

func formatProofStepError(outcome ProofCommandOutcome, expectedExit int, missingTokens []string) string {
	parts := make([]string, 0, 3)
	if outcome.Err != nil {
		parts = append(parts, outcome.Err.Error())
	}
	if outcome.ExitCode != expectedExit {
		parts = append(parts, fmt.Sprintf("exit code %d (expected %d)", outcome.ExitCode, expectedExit))
	}
	if len(missingTokens) > 0 {
		parts = append(parts, fmt.Sprintf("stdout missing tokens: %s", strings.Join(missingTokens, ", ")))
	}
	if len(parts) == 0 {
		return "verification failed"
	}
	return strings.Join(parts, "; ")
}

func missingStdoutTokens(stdout string, required []string) []string {
	if len(required) == 0 {
		return nil
	}

	missing := make([]string, 0, len(required))
	for _, token := range required {
		if !strings.Contains(stdout, token) {
			missing = append(missing, token)
		}
	}
	return missing
}

func defaultProofCommandExecutor(ctx context.Context, command, cwd string, env []string) ProofCommandOutcome {
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), env...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return ProofCommandOutcome{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Err:      err,
	}
}

func secondsToDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
