package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

const defaultProofSpecTimeout = 60 * time.Second

type proofSpecGateResult struct {
	specParsed    bool
	specVersion   int
	workspaceTier verify.VerificationTier
	methodCounts  string
	commandHash   string
	executed      int
	passed        int
	failed        int
	manualPending int
	failedStepIDs []string
	runtime       time.Duration
	errors        []string
	warnings      []string
}

func evaluateProofSpecGate(target *CompletionTarget) proofSpecGateResult {
	result := proofSpecGateResult{workspaceTier: determineProofSpecTier(target.WorkspacePath)}

	if target.WorkspacePath == "" {
		result.warnings = append(result.warnings, "verification spec check skipped: workspace path not available")
		return result
	}

	spec, err := verify.LoadProofSpec(target.WorkspacePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.warnings = append(result.warnings, "VERIFICATION_SPEC.yaml missing (Phase A advisory: completion proceeds)")
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("verification spec parse failed: %v", err))
		return result
	}

	result.specParsed = true
	result.specVersion = spec.Version
	result.methodCounts = formatMethodCountsByTier(spec)

	if target.BeadsID != "" && spec.Scope.BeadsID != target.BeadsID {
		result.errors = append(result.errors,
			fmt.Sprintf("verification spec scope.beads_id mismatch: spec=%s completion=%s", spec.Scope.BeadsID, target.BeadsID))
	}

	if err := verify.ValidateProofSpecCommandSyntax(spec); err != nil {
		result.errors = append(result.errors, fmt.Sprintf("verification spec command syntax invalid: %v", err))
		return result
	}

	applicable := filterProofSpecEntriesForTier(spec, result.workspaceTier)
	result.commandHash = hashProofSpecCommandList(applicable)

	if len(applicable) == 0 {
		result.warnings = append(result.warnings,
			fmt.Sprintf("verification spec has no entries for tier %q", result.workspaceTier))
		return result
	}

	start := time.Now()
	for _, entry := range applicable {
		if entry.Method == verify.VerificationMethodManual {
			result.manualPending++
			continue
		}

		result.executed++
		stepErr := executeProofSpecEntry(entry, target.WorkspacePath)
		if stepErr != nil {
			result.failed++
			result.failedStepIDs = append(result.failedStepIDs, entry.ID)
			result.errors = append(result.errors, fmt.Sprintf("verification_spec[%s]: %v", entry.ID, stepErr))
			continue
		}
		result.passed++
	}
	result.runtime = time.Since(start)

	if result.manualPending > 0 {
		result.warnings = append(result.warnings,
			fmt.Sprintf("verification spec has %d manual step(s) pending", result.manualPending))
	}

	return result
}

func (r proofSpecGateResult) gateResult() verify.GateResult {
	if len(r.errors) > 0 {
		return verify.GateResult{
			Gate:   verify.GateVerificationSpec,
			Passed: false,
			Error:  strings.Join(r.errors, "; "),
		}
	}

	return verify.GateResult{Gate: verify.GateVerificationSpec, Passed: true}
}

func (r proofSpecGateResult) digestComment() string {
	if !r.specParsed {
		return ""
	}

	status := "pass"
	if r.failed > 0 || len(r.errors) > 0 {
		status = "fail"
	}

	comment := fmt.Sprintf(
		"Verification spec digest: v%d tier=%s methods=%s cmd_hash=%s status=%s pass=%d fail=%d manual_pending=%d runtime=%s",
		r.specVersion,
		r.workspaceTier,
		r.methodCounts,
		r.commandHash,
		status,
		r.passed,
		r.failed,
		r.manualPending,
		formatProofSpecRuntime(r.runtime),
	)

	if len(r.failedStepIDs) > 0 {
		comment += " failed_ids=" + strings.Join(r.failedStepIDs, ",")
	}

	return comment
}

func postProofSpecDigestComment(beadsID, comment string) error {
	if beadsID == "" || strings.TrimSpace(comment) == "" {
		return nil
	}

	return withBeadsFallback("", func(client *beads.Client) error {
		return client.AddComment(beadsID, "orchestrator", comment)
	}, func() error {
		return beads.FallbackAddComment(beadsID, comment)
	}, beads.WithAutoReconnect(3))
}

func determineProofSpecTier(workspacePath string) verify.VerificationTier {
	tier := strings.TrimSpace(strings.ToLower(verify.ReadTierFromWorkspace(workspacePath)))
	switch tier {
	case string(verify.VerificationTierLight):
		return verify.VerificationTierLight
	case string(verify.VerificationTierOrchestrator):
		return verify.VerificationTierOrchestrator
	default:
		return verify.VerificationTierFull
	}
}

func filterProofSpecEntriesForTier(spec *verify.ProofSpec, tier verify.VerificationTier) []verify.ProofVerification {
	entries := make([]verify.ProofVerification, 0)
	for _, entry := range spec.Verification {
		if entry.Tier == tier {
			entries = append(entries, entry)
		}
	}
	return entries
}

func hashProofSpecCommandList(entries []verify.ProofVerification) string {
	if len(entries) == 0 {
		sum := sha256.Sum256(nil)
		return hex.EncodeToString(sum[:])
	}

	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Method == verify.VerificationMethodManual {
			continue
		}

		normalizedCWD := strings.TrimSpace(entry.CWD)
		if normalizedCWD == "" {
			normalizedCWD = "."
		}
		normalizedCmd := strings.Join(strings.Fields(entry.Command), " ")
		lines = append(lines, normalizedCWD+"|"+normalizedCmd)
	}

	joined := strings.Join(lines, "\n")
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:])
}

func formatMethodCountsByTier(spec *verify.ProofSpec) string {
	tiers := []verify.VerificationTier{
		verify.VerificationTierLight,
		verify.VerificationTierFull,
		verify.VerificationTierOrchestrator,
	}
	methods := []verify.VerificationMethod{
		verify.VerificationMethodCLISmoke,
		verify.VerificationMethodIntegration,
		verify.VerificationMethodBrowser,
		verify.VerificationMethodManual,
		verify.VerificationMethodStatic,
	}

	counts := make(map[verify.VerificationTier]map[verify.VerificationMethod]int)
	for _, entry := range spec.Verification {
		if _, ok := counts[entry.Tier]; !ok {
			counts[entry.Tier] = make(map[verify.VerificationMethod]int)
		}
		counts[entry.Tier][entry.Method]++
	}

	parts := make([]string, 0, len(tiers))
	for _, tier := range tiers {
		tierCounts, ok := counts[tier]
		if !ok || len(tierCounts) == 0 {
			continue
		}

		methodParts := make([]string, 0, len(methods))
		for _, method := range methods {
			count := tierCounts[method]
			if count == 0 {
				continue
			}
			methodParts = append(methodParts, fmt.Sprintf("%s=%d", method, count))
		}

		if len(methodParts) > 0 {
			parts = append(parts, fmt.Sprintf("%s[%s]", tier, strings.Join(methodParts, ",")))
		}
	}

	if len(parts) == 0 {
		return "none"
	}

	return strings.Join(parts, ";")
}

func executeProofSpecEntry(entry verify.ProofVerification, workspacePath string) error {
	timeout := defaultProofSpecTimeout
	if entry.TimeoutSeconds > 0 {
		timeout = time.Duration(entry.TimeoutSeconds) * time.Second
	}

	cwd := resolveProofSpecCWD(workspacePath, entry.CWD)
	if _, err := os.Stat(cwd); err != nil {
		return fmt.Errorf("cwd %q invalid: %w", cwd, err)
	}

	stdout, stderr, exitCode, err := runProofSpecCommand(entry.Command, cwd, timeout)
	if exitCode != entry.Expect.ExitCode {
		return fmt.Errorf("exit code %d (expected %d) stdout=%q stderr=%q", exitCode, entry.Expect.ExitCode, truncateProofSpecOutput(stdout), truncateProofSpecOutput(stderr))
	}

	for _, token := range entry.Expect.StdoutContains {
		if !strings.Contains(stdout, token) {
			return fmt.Errorf("stdout missing required token %q (stdout=%q)", token, truncateProofSpecOutput(stdout))
		}
	}

	if err != nil && exitCode == 0 {
		return err
	}

	return nil
}

func resolveProofSpecCWD(workspacePath, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return workspacePath
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	return filepath.Clean(filepath.Join(workspacePath, raw))
}

func runProofSpecCommand(command, cwd string, timeout time.Duration) (string, string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = cwd

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return stdout.String(), stderr.String(), exitCode, fmt.Errorf("command timed out after %s", timeout)
	}

	return stdout.String(), stderr.String(), exitCode, err
}

func truncateProofSpecOutput(out string) string {
	out = strings.TrimSpace(out)
	if len(out) <= 200 {
		return out
	}
	return out[:200] + "... (truncated)"
}

func formatProofSpecRuntime(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	return d.Round(time.Millisecond).String()
}
