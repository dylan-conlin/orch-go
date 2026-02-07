<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** expandTriageReadyEpics() was including ALL epic children without filtering by status, causing spawn failures on closed issues.

**Evidence:** Added status filter in daemon.go:358-363 that skips closed children; test TestExpandTriageReadyEpics_FiltersClosedChildren verifies only open/in_progress children are included.

**Knowledge:** Epic child expansion must respect issue status - closed children should never enter the spawn queue.

**Next:** Fix deployed, all tests passing, issue can be closed.

**Promote to Decision:** recommend-no (implementation detail, not architectural pattern)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Fix Daemon Epic Child Status

**Question:** Why does expandTriageReadyEpics() include closed children when expanding epic children, causing spawn failures?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: expandTriageReadyEpics() adds ALL children without status filtering

**Evidence:** In `pkg/daemon/daemon.go:356-369`, the loop that adds epic children has NO status check:
```go
for _, child := range children {
    // Only add if not already in the list
    if !existingIDs[child.ID] {
        issues = append(issues, child)  // <-- No status filter here
        existingIDs[child.ID] = true
        epicChildIDs[child.ID] = true
        // ...
    }
}
```

**Source:** 
- `pkg/daemon/daemon.go:313-373` - expandTriageReadyEpics function
- `pkg/daemon/issue_adapter.go:79-107` - ListEpicChildren function

**Significance:** This is the root cause. ALL children (including closed ones) are added to the spawn queue, which causes the daemon to attempt spawning work on closed issues.

---

### Finding 2: NextIssueExcluding already filters in_progress but not closed

**Evidence:** The NextIssueExcluding function has explicit filters for other statuses:
- Line 258-262: Skips "blocked" status
- Line 265-270: Skips "in_progress" status
- But there's NO check for "closed" status

**Source:** `pkg/daemon/daemon.go:214-311` - NextIssueExcluding function

**Significance:** The daemon assumes that closed issues won't be in the issue list. The bug bypasses this assumption by adding closed children during epic expansion.

---

### Finding 3: Valid status values are "open", "in_progress", "blocked", and "closed"

**Evidence:** Found status string literals throughout test files and type checking code:
- `pkg/beads/mock_client.go`: Multiple uses of "open", "closed", "in_progress"
- `pkg/beads/types.go:150`: Dependency checking logic uses `!= "closed"`

**Source:** 
- `pkg/beads/mock_client.go` - Status value usage
- `pkg/beads/types.go` - Type definitions

**Significance:** We know exactly which status value to filter - "closed" issues should not be included in epic child expansion.

---

## Synthesis

**Key Insights:**

1. **Epic expansion lacked status awareness** - The expandTriageReadyEpics function added ALL children from ListEpicChildren without checking if they were closed, blocked, or otherwise unspawnable.

2. **Inconsistent filtering logic** - NextIssueExcluding filters "blocked" and "in_progress" but assumed closed issues wouldn't be in the list. Epic expansion bypassed this assumption.

3. **Simple fix with high impact** - Adding a single status check (child.Status == "closed") prevents spawn failures and aligns epic expansion with the rest of the daemon's filtering logic.

**Answer to Investigation Question:**

expandTriageReadyEpics() included closed children because it had no status filtering logic in the child expansion loop (Finding 1). The fix adds a status check at daemon.go:358-363 to skip closed children before adding them to the spawn queue (Finding 2, 3). This prevents spawn failures and aligns with existing filtering patterns elsewhere in the daemon.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

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
