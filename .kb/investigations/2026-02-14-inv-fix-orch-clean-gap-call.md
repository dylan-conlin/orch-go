<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** cleanUntrackedDiskSessions() skips IsSessionProcessing() check for sessions idle >5min, deleting active TUI sessions.

**Evidence:** Code at clean_cmd.go:475-482 only calls IsSessionProcessing() for sessions updated within 5 minutes; sessions idle longer bypass the check.

**Knowledge:** Interactive/TUI orchestrator sessions have no workspace files (Layer 1 protection), and idle sessions bypass IsSessionProcessing() (Layer 3), leaving them with zero protection.

**Next:** Remove 5-minute recency threshold; call IsSessionProcessing() for ALL untracked sessions.

**Authority:** implementation - Fixes confirmed bug within existing cleanup logic, no architectural changes needed.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Fix Orch Clean Gap Call

**Question:** How does cleanUntrackedDiskSessions() fail to protect active sessions idle >5min, and what's the minimal fix?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** Agent orch-go-zs6
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: 5-minute recency threshold creates protection gap

**Evidence:** Lines 467-482 in clean_cmd.go implement a two-tier check:
1. Quick check: session updated within 5 minutes (isRecentlyActive)
2. Expensive check: IsSessionProcessing() - but ONLY for recently active sessions

Sessions idle >5min skip IsSessionProcessing() entirely and proceed to deletion.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/clean_cmd.go:467-485`

**Significance:** This is the root cause. TUI sessions have no workspace files (Layer 1 protection bypassed). If idle >5min, they also bypass Layer 3 (IsSessionProcessing check). Result: active sessions get deleted.

---

### Finding 2: Comment acknowledges the gap but implements incomplete protection

**Evidence:** Comment at lines 458-459 states:
> "The orchestrator/interactive sessions don't have workspace .session_id files, but they're still valid sessions that should not be deleted."

Then lines 461-463:
> "We use two heuristics to detect active sessions (no extra API calls needed):
> 1. Recently updated sessions (within last 5 minutes) - likely in use
> 2. Sessions that are currently processing (expensive check, only if recently updated)"

The phrase "no extra API calls needed" is incorrect - heuristic #2 DOES make an API call (IsSessionProcessing).

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/clean_cmd.go:458-463`

**Significance:** The code was designed to minimize API calls, but the optimization is wrong. It prioritizes performance over correctness and creates a deletion vector.

---

### Finding 3: Model confirms this as Vector #2 (HIGH risk, OPEN status)

**Evidence:** Session Deletion Vectors model documents this exact failure mode:
- Vector #2: "orch clean --sessions (untracked)"
- Protection: "3-layer: workspace tracking → 5min recency → IsSessionProcessing()"
- Gap: "Sessions idle >5min WITHOUT .session_id workspace file bypass Layer 2+3 and get DELETED"
- Risk: HIGH
- Status: OPEN

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors.md:73`

**Significance:** This confirms the bug is known, high-risk, and has clear reproduction steps. Fix is well-scoped.

---

## Synthesis

**Key Insights:**

1. **Premature optimization traded correctness for performance** - The code was designed to minimize API calls by only checking IsSessionProcessing() for recently-active sessions, but this optimization created a deletion vector. The comment even claimed "no extra API calls needed" while making API calls conditionally.

2. **Three-layer defense had a gap in Layer 3** - TUI sessions have no workspace files (Layer 1), so they rely entirely on IsSessionProcessing() (Layer 3). The 5-minute recency check (Layer 2) bypassed Layer 3 for idle sessions, leaving them with zero protection.

3. **The fix is simple because the logic was wrong, not complex** - Removing the recency threshold and calling IsSessionProcessing() for ALL untracked sessions closes the gap completely. The performance cost (one API call per untracked session) is acceptable for correctness.

**Answer to Investigation Question:**

cleanUntrackedDiskSessions() fails to protect active sessions idle >5min because it only calls IsSessionProcessing() for sessions updated within the last 5 minutes (lines 475-481). Sessions idle longer bypass this check entirely. The minimal fix is to remove the recency threshold and call IsSessionProcessing() for ALL untracked sessions, accepting the performance cost of additional API calls in exchange for correctness. This closes Vector #2 from the Session Deletion Vectors model.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles after fix (verified: `go build ./cmd/orch/...` succeeded)
- ✅ All existing tests pass (verified: `go test ./cmd/orch/... -v` - all pass, 1 skip)
- ✅ IsSessionProcessing() now called for ALL untracked sessions (verified: code inspection at clean_cmd.go:470-476)

**What's untested:**

- ⚠️ Fix prevents TUI session deletion in production (manual reproduction requires >5min wait)
- ⚠️ Performance impact of additional API calls (depends on number of untracked sessions)
- ⚠️ No regression in orchestrator session preservation (heuristic title matching still works)

**What would change this:**

- Finding would be wrong if IsSessionProcessing() API call is unreliable (returns false positives/negatives)
- Finding would be wrong if sessions can be deleted through other vectors not addressed by this fix
- Finding would be wrong if the 5-minute threshold had a different purpose we didn't understand

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Remove recency threshold** - Call IsSessionProcessing() for ALL untracked sessions, not just those updated within 5 minutes.

**Why this approach:**
- Closes the protection gap completely - no active session can be deleted
- Minimal code change - remove conditional, keep the safety check
- Cost is acceptable - one API call per untracked session is reasonable for correctness

**Trade-offs accepted:**
- More API calls to OpenCode (one per untracked session)
- Slightly slower `orch clean --sessions` when many untracked sessions exist
- This is acceptable because deleting active sessions is catastrophic; performance is secondary

**Implementation sequence:**
1. Remove lines 467-482 (recency threshold logic)
2. Replace with direct IsSessionProcessing() call for all untracked sessions
3. Update comment to reflect new logic (no heuristics, direct check)
4. Test with reproduction scenario (TUI session idle >5min)

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
