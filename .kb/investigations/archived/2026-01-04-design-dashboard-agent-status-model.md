## Summary (D.E.K.N.)

**Delta:** Designed a simple 3-priority status model to replace ~10 scattered conditions across 900+ lines.

**Evidence:** Traced all status assignments in serve_agents.go - found duplicated SYNTHESIS.md checks (lines 862-868 and 909-930), an optimization at line 609 that skips idle agents from beads fetching causing status checks to be bypassed.

**Knowledge:** The complexity comes from incremental additions without a coherent model. A priority-based cascade (Beads Closed > Phase Complete > SYNTHESIS.md exists > session activity) makes the logic deterministic and debuggable.

**Next:** Implement the Priority Cascade approach - consolidate into a single `determineAgentStatus()` function, remove the line 609 optimization.

---

# Investigation: Dashboard Agent Status Model

**Question:** How should we simplify the dashboard agent status logic to be correct, debuggable, and maintainable?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - recommendation ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: There are 10+ conditions determining agent status across scattered code locations

**Evidence:** Status assignment locations in serve_agents.go:
1. Line 564-566: Initial status = "active" or "idle" based on session activity (10min threshold)
2. Line 601-603: Track pendingFilterByBeadsID for idle agents >30min
3. Line 609: **CRITICAL OPTIMIZATION** - Only add to beadsIDsToFetch if `status == "active"` - this skips ALL subsequent beads checks for idle agents
4. Line 728-731: Completed workspaces (those with SYNTHESIS.md in archive) set status = "completed"
5. Line 833-834: Inside beadsIDsToFetch block - Phase: Complete → status = "completed"
6. Line 845-855: Inside beadsIDsToFetch block - Beads issue closed → status = "completed"
7. Line 862-868: Inside beadsIDsToFetch block - SYNTHESIS.md exists → status = "completed"
8. Line 894-899: Post-filter removes idle non-completed agents (pendingFilterByBeadsID)
9. Line 909-930: **OUTSIDE beadsIDsToFetch block** - Second SYNTHESIS.md check for idle/untracked agents

**Source:** `cmd/orch/serve_agents.go` lines 564-930

**Significance:** The logic is spread across 350+ lines with overlapping conditions. The line 609 optimization causes idle agents to skip the entire beads-based status checking (Phase: Complete, beads closed, SYNTHESIS.md check inside the block), but a second SYNTHESIS.md check was added outside to try to catch them. This duplication reveals the underlying problem: there's no coherent status model, just patches.

---

### Finding 2: The line 609 optimization causes incorrect status for idle agents

**Evidence:** The optimization at line 609:
```go
if status == "active" && agent.BeadsID != "" && !seenBeadsIDs[agent.BeadsID] {
    beadsIDsToFetch = append(beadsIDsToFetch, agent.BeadsID)
    // ...
}
```

This means idle agents (session inactive >10min) are NOT added to `beadsIDsToFetch`, so they skip:
- Phase: Complete check from beads comments (lines 823-836)
- Beads issue closed check (lines 845-855)
- First SYNTHESIS.md check (lines 862-868)

The fix attempt added a second SYNTHESIS.md check (lines 909-930), but:
- It doesn't check Phase: Complete (only SYNTHESIS.md)
- The workspace lookup by beads ID may fail for untracked agents
- The fallback by workspace name parsing is fragile

**Source:** `cmd/orch/serve_agents.go:609`, SESSION_HANDOFF.md

**Significance:** The optimization saves CPU by not fetching beads for idle sessions, but breaks correctness. An idle agent that reported Phase: Complete or has SYNTHESIS.md won't show as completed in the dashboard.

---

### Finding 3: The prior decision establishes "Phase: Complete from beads" as authoritative

**Evidence:** From SPAWN_CONTEXT.md prior knowledge section:
```
- Dashboard agent status derived from beads phase, not session time
  - Reason: Phase: Complete from beads comments is authoritative for completion status, session idle time is secondary
```

**Source:** SPAWN_CONTEXT.md, `.kb/decisions/` (referenced)

**Significance:** There's already a decision about the priority hierarchy:
1. Phase: Complete (from beads) is authoritative
2. Session activity is secondary

This validates that the line 609 optimization is wrong - it inverts the priority by checking session activity BEFORE checking Phase: Complete.

---

## Synthesis

**Key Insights:**

1. **Lack of coherent model** - Status is determined by 10+ conditions scattered across 350+ lines without a single source of truth. Each condition was added incrementally without stepping back to design a proper model.

2. **Optimization inverts priority** - The line 609 optimization checks session activity BEFORE beads/Phase, causing idle agents with Phase: Complete to show as "idle" instead of "completed". This violates the prior decision.

3. **Duplication signals design problem** - The SYNTHESIS.md check appears in two places (lines 862-868 and 909-930), revealing that patches were applied without addressing root cause.

**Answer to Investigation Question:**

The dashboard agent status logic should use a **Priority Cascade Model**:

```
Priority 1: Beads issue closed → "completed" (orchestrator verified completion)
Priority 2: Phase: Complete reported → "completed" (agent declared done)  
Priority 3: SYNTHESIS.md exists → "completed" (artifact proves completion)
Priority 4: Session activity → "active" (<10min) or "idle" (>=10min)
```

This model is:
- **Deterministic** - Each agent gets ONE status based on highest-priority match
- **Correct** - Completion signals always override activity signals
- **Debuggable** - Single function with clear priority order
- **Consistent with prior decision** - Phase: Complete is authoritative over session time

---

## Structured Uncertainty

**What's tested:**

- ✅ Line 609 optimization skips idle agents from beadsIDsToFetch (verified: read code at serve_agents.go:609)
- ✅ Phase: Complete check only happens inside beadsIDsToFetch block (verified: read code at serve_agents.go:823-836)
- ✅ SYNTHESIS.md check exists in two places (verified: read code at lines 862-868 and 909-930)

**What's untested:**

- ⚠️ CPU impact of removing line 609 optimization (not benchmarked - session handoff suggests it "causes more bugs than CPU it saves")
- ⚠️ Actual dashboard behavior for the test untracked agent (not verified via curl - would require running dashboard)

**What would change this:**

- If CPU impact of removing optimization is severe (>10% CPU increase), might need alternative approach
- If there are other edge cases not covered by the 4-priority model

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Priority Cascade with Single Function** - Consolidate all status logic into one `determineAgentStatus()` function with explicit priority order.

**Why this approach:**
- Eliminates scattered conditions - all logic in one place
- Removes the line 609 optimization that causes bugs
- Makes priority order explicit and testable
- Consistent with prior decision about Phase: Complete being authoritative

**Trade-offs accepted:**
- Slightly more beads fetches for idle agents (CPU cost)
- Why acceptable: SESSION_HANDOFF.md states it "causes more bugs than CPU it saves"

**Implementation sequence:**
1. Create `determineAgentStatus(agent, issue, comments, workspacePath) string` function with priority cascade
2. Remove line 609 optimization - add ALL agents with beadsID to beadsIDsToFetch
3. Replace all inline status assignments with calls to `determineAgentStatus()`
4. Remove duplicate SYNTHESIS.md check at lines 909-930 (will be handled in single function)

### Alternative Approaches Considered

**Option B: Keep optimization, fix edge cases**
- **Pros:** Preserves CPU optimization for idle agents
- **Cons:** Adds more complexity to work around the fundamental inversion of priority; more edge cases will emerge
- **When to use instead:** Only if CPU benchmarks show severe impact (>20% increase)

**Option C: Cache completion status**
- **Pros:** Could avoid re-checking completed agents entirely
- **Cons:** Adds cache invalidation complexity; doesn't solve the root model problem
- **When to use instead:** If performance is still a problem after Option A

**Rationale for recommendation:** The fundamental problem is a missing coherent model. Options B and C add complexity without fixing the root cause. Option A directly addresses the design flaw by establishing a clear priority hierarchy.

---

### Implementation Details

**What to implement first:**
- The `determineAgentStatus()` function - this is the core design change
- Remove line 609 optimization - enables correct priority cascade
- Single pass through all agents calling the new function

**Things to watch out for:**
- ⚠️ The function needs workspace path to check SYNTHESIS.md - ensure cache is available
- ⚠️ Untracked agents have fake beads IDs - beads checks will fail, SYNTHESIS.md is the fallback
- ⚠️ Session title parsing for workspace lookup is fragile (lines 918-924) - consider workspace cache by name

**Areas needing further investigation:**
- Performance impact of fetching beads for all agents (mitigated by existing TTL cache)
- Whether workspace cache needs entries by workspace name (not just beads ID)

**Success criteria:**
- ✅ Idle agent with Phase: Complete shows as "completed" in dashboard
- ✅ Idle agent with SYNTHESIS.md shows as "completed" in dashboard
- ✅ All status determination happens in one function
- ✅ No duplicate SYNTHESIS.md checks in code

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Main file with all status logic (1403 lines)
- `pkg/verify/check.go` - ParsePhaseFromComments and related functions
- `pkg/verify/phase_gates.go` - Phase extraction from comments
- `.orch/SESSION_HANDOFF.md` - Prior debugging context

**Commands Run:**
```bash
# Find all status assignments
grep -n "Status.*=\|status.*=" cmd/orch/serve_agents.go

# Find all references to ParsePhaseFromComments
grep -rn "ParsePhaseFromComments" --include="*.go"
```

**Related Artifacts:**
- **Decision:** Prior decision about Phase: Complete being authoritative (referenced in SPAWN_CONTEXT.md)
- **Investigation:** This investigation establishes the Priority Cascade model

---

## Investigation History

**2026-01-04 11:30:** Investigation started
- Initial question: How to simplify dashboard agent status logic?
- Context: ~10 conditions, multiple code paths, optimization causing edge case bugs

**2026-01-04 12:00:** Problem Framing complete
- Identified 10+ conditions in serve_agents.go
- Found line 609 optimization as root cause of idle agent bugs
- Discovered duplicate SYNTHESIS.md checks

**2026-01-04 12:30:** Investigation completed
- Status: Complete
- Key outcome: Recommend Priority Cascade model with single `determineAgentStatus()` function
