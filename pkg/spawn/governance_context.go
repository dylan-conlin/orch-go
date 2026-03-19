package spawn

import (
	"fmt"
	"strings"
)

// GovernanceProtectedPath describes a path protected by governance hooks.
type GovernanceProtectedPath struct {
	// Pattern is the human-readable glob-like pattern (e.g., "pkg/spawn/gates/*")
	Pattern string
	// Description explains what the path contains
	Description string
}

// GovernanceProtectedPaths lists all paths protected by the governance file
// protection hook (gate-governance-file-protection.py). Workers cannot modify
// these files — edits are denied by PreToolUse hooks.
//
// This list MUST stay in sync with GOVERNANCE_PATTERNS in
// ~/.orch/hooks/gate-governance-file-protection.py
var GovernanceProtectedPaths = []GovernanceProtectedPath{
	{Pattern: "pkg/spawn/gates/*", Description: "spawn gate logic"},
	{Pattern: "pkg/verify/precommit.go", Description: "pre-commit verification gate"},
	{Pattern: "pkg/verify/accretion.go", Description: "completion accretion gate"},
	{Pattern: "cmd/orch/*_lint_test.go", Description: "structural lint tests"},
	{Pattern: "cmd/orch/governance_checksum_test.go", Description: "governance checksum test"},
	{Pattern: "cmd/orch/testdata/governance_checksums.json", Description: "governance checksum manifest"},
	{Pattern: "scripts/pre-commit*", Description: "pre-commit gate scripts"},
	{Pattern: "~/.orch/hooks/*.py", Description: "deny hooks"},
}

// GenerateGovernanceContext produces a markdown section listing governance-protected
// paths for injection into SPAWN_CONTEXT.md. This tells agents what they cannot
// modify BEFORE they plan, preventing wasted work from mid-session hook denials.
// When noTrack is true, the escalation action omits beads-specific instructions.
func GenerateGovernanceContext(noTrack bool) string {
	var b strings.Builder
	b.WriteString("## GOVERNANCE-PROTECTED PATHS\n\n")
	b.WriteString("The following paths are protected by governance hooks. **Edits to these files will be denied.**\n")
	b.WriteString("If your task requires changes to any of these, escalate immediately.\n\n")

	for _, p := range GovernanceProtectedPaths {
		fmt.Fprintf(&b, "- `%s` — %s\n", p.Pattern, p.Description)
	}

	if noTrack {
		b.WriteString("\n**Action on denial:** Document in your investigation file: \"DISCOVERED: governance file <path> needs update - <reason>\"\n")
	} else {
		b.WriteString("\n**Action on denial:** Report via beads: `bd comments add <id> \"DISCOVERED: governance file <path> needs update - <reason>\"`\n")
	}

	return b.String()
}
