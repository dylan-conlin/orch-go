package spawn

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
)

// GenerateGovernanceContext produces a markdown section listing governance-protected
// paths for injection into SPAWN_CONTEXT.md. This tells agents what they cannot
// modify BEFORE they plan, preventing wasted work from mid-session hook denials.
// When noTrack is true, the escalation action omits beads-specific instructions.
func GenerateGovernanceContext(noTrack bool) string {
	var b strings.Builder
	b.WriteString("## GOVERNANCE-PROTECTED PATHS\n\n")
	b.WriteString("The following paths are protected by governance hooks. **Edits to these files will be denied.**\n")
	b.WriteString("If your task requires changes to any of these, escalate immediately.\n\n")

	for _, p := range gates.GovernanceProtectedPaths {
		fmt.Fprintf(&b, "- `%s` — %s\n", p.Pattern, p.Reason)
		if p.RedirectHint != "" {
			fmt.Fprintf(&b, "  - **Instead:** %s\n", p.RedirectHint)
		}
	}

	if noTrack {
		b.WriteString("\n**Action on denial:** Document in your investigation file: \"DISCOVERED: governance file <path> needs update - <reason>\"\n")
	} else {
		b.WriteString("\n**Action on denial:** Report via beads: `bd comments add <id> \"DISCOVERED: governance file <path> needs update - <reason>\"`\n")
	}

	return b.String()
}
