<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Context exhaustion with uncommitted work can be detected by monitoring token usage thresholds combined with git status checks in the agent's working directory.

**Evidence:** Token stats are already fetched per-agent in status_cmd.go:387-394; git status patterns exist in handoff.go:545-556; OpenCode provides token usage per message.

**Knowledge:** The key signals are: (1) high token usage (>80% of typical context limit ~180K), (2) agent is not actively processing, and (3) git status shows uncommitted changes in agent's project directory.

**Next:** Implement ContextExhaustionRisk detection in status command that combines token threshold + uncommitted work detection.

---

# Investigation: Detect Agents Exhausting Context with Uncommitted Work

**Question:** How can we detect when agents are exhausting their context window while having uncommitted work, and alert the orchestrator via orch status or monitor?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Agent via systematic-debugging skill
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Token statistics are already collected per-agent in orch status

**Evidence:** The status_cmd.go already fetches token usage for each agent:
```go
// status_cmd.go:387-394
for i := range filteredAgents {
    if filteredAgents[i].SessionID != "" && filteredAgents[i].SessionID != "tmux-stalled" {
        tokens, err := client.GetSessionTokens(filteredAgents[i].SessionID)
        if err == nil && tokens != nil {
            filteredAgents[i].Tokens = tokens
        }
    }
}
```

**Source:** cmd/orch/status_cmd.go:387-394, pkg/opencode/client.go:856-868

**Significance:** We already have the token usage data available. We can add a threshold check (e.g., >150K total tokens) to identify agents at risk of context exhaustion.

---

### Finding 2: Git status pattern exists in handoff.go for detecting uncommitted changes

**Evidence:** The handoff.go file has a function that checks for uncommitted changes:
```go
// handoff.go:545-556
statusCmd := exec.Command("git", "status", "--porcelain")
statusCmd.Dir = projectDir
if output, err := statusCmd.Output(); err == nil {
    changes := strings.TrimSpace(string(output))
    if changes != "" {
        state.HasUncommitted = true
        // Count changes
        lines := strings.Split(changes, "\n")
        state.Summary = fmt.Sprintf("%d uncommitted changes", len(lines))
    }
}
```

**Source:** cmd/orch/handoff.go:545-556

**Significance:** We can reuse this pattern to check for uncommitted work in the agent's project directory, which we can derive from the beads ID or workspace SPAWN_CONTEXT.md.

---

### Finding 3: Agent project directory can be determined from workspace or beads ID

**Evidence:** The status command already tracks project per agent (AgentInfo.Project) and can find workspace paths:
- `findWorkspaceByBeadsID(projectDir, beadsID)` returns workspace path
- `extractProjectDirFromWorkspace(workspacePath)` extracts PROJECT_DIR from SPAWN_CONTEXT.md

**Source:** cmd/orch/status_cmd.go:220-228, main.go workspace lookup patterns

**Significance:** We can determine the correct project directory to run git status against for each agent.

---

### Finding 4: Claude's context limit is approximately 200K tokens

**Evidence:** Based on Anthropic's published model specifications, Claude Opus has a context window of approximately 200K tokens. However, effective working limit is lower due to output reservation.

**Source:** Anthropic model documentation, common knowledge

**Significance:** A reasonable warning threshold would be ~150K tokens (75% of 200K) with a critical threshold at ~180K (90% of 200K).

---

## Synthesis

**Key Insights:**

1. **All necessary data is already available** - Token stats, project directories, and git status patterns exist. We just need to combine them into a detection function.

2. **Risk detection should be multi-factor** - An agent is at risk only when: high token usage AND uncommitted changes AND not actively processing. Just high tokens or just uncommitted changes alone isn't necessarily risky.

3. **Status display should add a warning column** - The existing agent table format in status_cmd.go can be extended with a RISK column showing context exhaustion warnings.

**Answer to Investigation Question:**

Context exhaustion with uncommitted work can be detected by adding a `ContextExhaustionRisk` struct to AgentInfo that combines:
1. Token usage threshold check (>150K warning, >180K critical)
2. Git status check for uncommitted changes in agent's project directory
3. Optional check for processing state (idle agents at high tokens are higher risk)

This can be displayed in `orch status` with a new RISK column and optionally trigger desktop notifications via `orch monitor`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Token stats are available per-agent (verified: code review of status_cmd.go:387-394)
- ✅ Git status pattern exists and works (verified: code review of handoff.go:545-556)
- ✅ Project directory extraction exists (verified: code review of extractProjectDirFromWorkspace)

**What's untested:**

- ⚠️ Performance impact of running git status for each agent (not benchmarked)
- ⚠️ Exact token threshold values (150K/180K are estimates, need real-world validation)
- ⚠️ Monitor SSE integration for real-time alerting (not implemented yet)

**What would change this:**

- If token stats become unavailable or unreliable, we'd need alternative detection
- If agents typically use >150K tokens without issues, thresholds need adjustment
- If git status is too slow, we might need caching or sampling

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Add ContextExhaustionRisk to status_cmd.go** - Extend AgentInfo struct and status display to include risk assessment based on token usage + uncommitted changes.

**Why this approach:**
- Minimal changes to existing code (extends existing patterns)
- Uses already-fetched token data (no additional API calls)
- Provides visibility in existing workflow (orch status)

**Trade-offs accepted:**
- Adds git status call per agent (acceptable overhead for occasional status checks)
- Thresholds are heuristic (can be tuned based on real-world feedback)

**Implementation sequence:**
1. Add `HasUncommittedWork(projectDir)` function (reuse handoff.go pattern)
2. Add `ContextExhaustionRisk` struct to AgentInfo
3. Add risk detection logic after token fetch in runStatus()
4. Update printAgentsWideFormat() to show risk column
5. Optionally add monitor SSE handling for real-time alerts

### Alternative Approaches Considered

**Option B: SSE-only detection**
- **Pros:** Real-time, no polling overhead
- **Cons:** SSE doesn't provide git status; requires additional polling anyway
- **When to use instead:** If we want real-time alerting without status command

**Option C: Daemon-based continuous monitoring**
- **Pros:** Fully automatic detection without user intervention
- **Cons:** More complex, adds daemon responsibilities
- **When to use instead:** If we want fully autonomous remediation

**Rationale for recommendation:** Option A provides immediate value with minimal changes and integrates naturally with the existing `orch status` workflow that orchestrators already use.

---

### Implementation Details

**What to implement first:**
- `HasUncommittedWork(projectDir string) (bool, int)` function
- `ContextExhaustionRisk` field in AgentInfo struct
- Risk detection in runStatus() after token fetch

**Things to watch out for:**
- ⚠️ Git status may fail if projectDir doesn't exist or isn't a git repo
- ⚠️ Cross-project agents may have different project directories
- ⚠️ Need to handle case where tokens are nil (no stats available)

**Areas needing further investigation:**
- Optimal token thresholds based on real usage patterns
- Whether to auto-send messages to at-risk agents via orch send
- Integration with daemon for automatic detection

**Success criteria:**
- ✅ `orch status` shows warning for agents with high tokens + uncommitted work
- ✅ Warning is visible and actionable (orchestrator can intervene)
- ✅ No false positives for agents just doing normal work

---

## References

**Files Examined:**
- cmd/orch/status_cmd.go - Current status implementation, token fetch pattern
- cmd/orch/handoff.go - Git status pattern for uncommitted changes
- pkg/opencode/client.go - Token stats aggregation (GetSessionTokens)
- pkg/opencode/types.go - Message and token types

**Commands Run:**
```bash
# Search for existing context/token handling
grep -r "context\|token\|limit\|exhaust" pkg/opencode/*.go

# Check for git status patterns
grep -r "uncommitted\|git.*status" --include="*.go" .
```

**Related Artifacts:**
- **Constraint:** "SSE busy->idle cannot detect true agent completion" - affects monitor design
- **Decision:** "Daemon completion uses beads polling not SSE" - related detection pattern

---

## Investigation History

**2026-01-03 09:00:** Investigation started
- Initial question: How to detect agents exhausting context with uncommitted work?
- Context: Orchestrator needs visibility into at-risk agents to intervene before context limit

**2026-01-03 09:15:** Key findings identified
- Token stats already fetched in status_cmd.go
- Git status pattern exists in handoff.go
- Project directory extraction available

**2026-01-03 09:30:** Synthesis complete
- Recommended approach: Extend status_cmd.go with risk detection
- Implementation plan defined
