## Summary (D.E.K.N.)

**Delta:** Implemented BLOCKED status detection in orch status - agents can now report BLOCKED: comments and orchestrator sees them with 🚫 blocked indicator.

**Evidence:** Tests pass for BLOCKED pattern parsing, build succeeds, getAgentStatus returns "🚫 blocked" when IsBlocked is true.

**Knowledge:** BLOCKED status is cleared when agent reports a later Phase: update, preventing stale blocked indicators.

**Next:** Close - implementation complete, skill documentation updated.

---

# Investigation: Agents Report Blocked Status Instead

**Question:** How should agents report BLOCKED status and how should orch status display it?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** og-feat-agents-report-blocked-29dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: PhaseStatus struct can track blocked state

**Evidence:** Added IsBlocked (bool) and BlockedReason (string) fields to PhaseStatus struct in pkg/verify/check.go.

**Source:** pkg/verify/check.go:29-36

**Significance:** Provides the data structure to track blocked status alongside phase information.

---

### Finding 2: BLOCKED pattern parsing with temporal awareness

**Evidence:** ParsePhaseFromComments now detects `BLOCKED: <reason>` pattern. Uses index tracking to clear blocked status if a Phase: update comes after the BLOCKED comment (agent resumed work).

**Source:** pkg/verify/check.go:75-117

**Significance:** Prevents stale blocked indicators when agents become unblocked and continue working.

---

### Finding 3: Status display shows 🚫 blocked indicator

**Evidence:** getAgentStatus() checks IsBlocked and returns "🚫 blocked" - positioned after completed/phantom checks but before stalled/running/idle.

**Source:** cmd/orch/main.go:3376-3397

**Significance:** Orchestrator can now visually identify blocked agents in `orch status` output.

---

## Synthesis

**Key Insights:**

1. **BLOCKED is a transient state** - Unlike phase, blocked status can be cleared by subsequent activity, so we track the relative ordering of BLOCKED vs Phase comments.

2. **Visible indicator needed** - Added 🚫 emoji to make blocked status immediately visible in terminal output, similar to ⚠️ for stalled agents.

3. **Skill documentation is key** - Updated feature-impl skill with BLOCKED pattern documentation to ensure agents know how to report blocked status.

**Answer to Investigation Question:**

Agents should run `bd comment <beads-id> "BLOCKED: <specific reason>"` when stuck. The orchestrator detects this pattern via PhaseStatus parsing and displays "🚫 blocked" in orch status output. The blocked status is automatically cleared when the agent reports a subsequent Phase: update.

---

## Implementation Summary

**Files modified:**
- `pkg/verify/check.go` - PhaseStatus struct + ParsePhaseFromComments with BLOCKED detection
- `pkg/verify/check_test.go` - Tests for BLOCKED parsing scenarios
- `cmd/orch/main.go` - AgentInfo with IsBlocked/BlockedReason + getAgentStatus returns "🚫 blocked"
- `~/orch-knowledge/skills/src/worker/feature-impl/.skillc/SKILL.md.template` - BLOCKED pattern documentation

**Tests added:**
- simple blocked comment
- blocked cleared by later phase update
- blocked case insensitive
- blocked with no phase
- multiple blocked - latest reason kept
