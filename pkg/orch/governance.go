package orch

import (
	"fmt"
	"os"
	"strings"
)

// GovernanceProtectedPath describes a file path pattern that is governance-protected.
// Workers targeting these paths will be blocked mid-execution by hooks, so we detect
// them at spawn time and warn before the worker starts.
type GovernanceProtectedPath struct {
	Pattern string // Path substring to match (case-insensitive)
	Reason  string // Why this path is protected
}

// GovernanceResult contains the result of a governance file check.
type GovernanceResult struct {
	MatchedPaths []GovernanceProtectedPath // Which protected paths were detected in the task
	Warning      string                    // Formatted warning message
}

// governanceProtectedPaths defines paths that are protected by governance hooks.
// These mirror the patterns in ~/.orch/hooks/gate-governance-file-protection.py.
// Workers targeting these files will be blocked by hooks at edit time — detecting
// them at spawn time prevents wasted worker sessions.
var governanceProtectedPaths = []GovernanceProtectedPath{
	{Pattern: "pkg/spawn/gates/", Reason: "spawn gate infrastructure"},
	{Pattern: "pkg/verify/", Reason: "verification gate infrastructure"},
	{Pattern: ".orch/hooks/", Reason: "governance hooks"},
	{Pattern: "scripts/pre-commit", Reason: "pre-commit gate scripts"},
	{Pattern: "skills/src/shared/worker-base", Reason: "worker base skill (shared protocols)"},
	{Pattern: "_lint_test.go", Reason: "structural lint tests"},
	{Pattern: "governance_checksum", Reason: "governance checksum manifest"},
}

// GovernanceProtectedPaths returns the list of governance-protected path patterns.
func GovernanceProtectedPaths() []GovernanceProtectedPath {
	return governanceProtectedPaths
}

// CheckGovernance scans a task description for references to governance-protected file paths.
// Returns nil if no governance paths are detected. This is a warning-only check — it does
// not block spawning.
func CheckGovernance(task string, skillName string, daemonDriven bool) *GovernanceResult {
	taskLower := strings.ToLower(task)

	var matched []GovernanceProtectedPath
	seen := map[string]bool{}
	for _, p := range governanceProtectedPaths {
		patternLower := strings.ToLower(p.Pattern)
		if strings.Contains(taskLower, patternLower) && !seen[p.Pattern] {
			matched = append(matched, p)
			seen[p.Pattern] = true
		}
	}

	if len(matched) == 0 {
		return nil
	}

	warning := formatGovernanceWarning(matched, skillName)

	result := &GovernanceResult{
		MatchedPaths: matched,
		Warning:      warning,
	}

	if !daemonDriven {
		fmt.Fprint(os.Stderr, warning)
	}

	return result
}

func formatGovernanceWarning(matched []GovernanceProtectedPath, skillName string) string {
	var b strings.Builder
	b.WriteString("\n⚠️  GOVERNANCE-PROTECTED FILES DETECTED\n")
	b.WriteString("   Task references paths protected by governance hooks:\n")
	for _, p := range matched {
		fmt.Fprintf(&b, "     • %s (%s)\n", p.Pattern, p.Reason)
	}
	b.WriteString("\n")
	b.WriteString("   Workers editing these files will be BLOCKED by hooks at runtime.\n")
	b.WriteString("   Consider: route this work to an orchestrator session instead.\n\n")
	return b.String()
}
