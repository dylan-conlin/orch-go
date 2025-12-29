<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Identified three critical gaps causing 49 zombie in_progress issues: (1) triage:ready label never removed after spawn, (2) no reconciliation to detect issues without active agents, (3) NEEDS_REVIEW completions stuck because test evidence gate not aware of design-session/investigation skills.

**Evidence:** 49 in_progress issues vs 5 active agents = 44 zombies. Code analysis shows UpdateIssueStatus changes status but RemoveLabel never called. Test evidence gate blocks completion for code-changing skills but NEEDS_REVIEW persists for already-closed issues.

**Knowledge:** State synchronization requires explicit transitions at spawn (remove triage:ready) and periodic reconciliation (detect orphaned in_progress issues). Completion verification gates need to be aware of issue state (closed issues shouldn't fail verification).

**Next:** Create epic with 4 children: (1) Remove triage:ready on spawn, (2) Add reconciliation command, (3) Fix NEEDS_REVIEW for closed issues, (4) Add stale zombie detection.

---

# Investigation: Completion Lifecycle Broken - 49 Zombie in_progress Issues

**Question:** Why are 49 issues stuck in in_progress with triage:ready labels, 9 NEEDS_REVIEW completions stuck on test evidence, and what state synchronization fixes are needed?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** design-session agent (og-work-completion-lifecycle-broken-29dec)
**Phase:** Complete
**Next Step:** Create epic with children for state synchronization fixes
**Status:** Complete

---

## Findings

### Finding 1: triage:ready labels never removed after spawn

**Evidence:** 
- 20+ issues have BOTH `in_progress` status AND `triage:ready` label
- Code analysis: `cmd/orch/main.go:1315` calls `verify.UpdateIssueStatus(beadsID, "in_progress")` but never calls `RemoveLabel`
- `grep -rn "RemoveLabel" /cmd/orch/` returns empty - RemoveLabel is never used in spawn/complete flow
- Example: `orch-go-hop2` is `in_progress` with `triage:ready` label

**Source:** 
- `cmd/orch/main.go:1313-1318` - spawn status update
- `pkg/beads/client.go:590-591` - RemoveLabel function exists but unused

**Significance:** This causes confusion for:
1. Daemon could re-spawn issues that are already being worked on (prevented only by in_progress status check)
2. Dashboard/reporting tools show misleading state
3. Orchestrator can't distinguish "spawned but not started" from "ready for spawn"

---

### Finding 2: 44 zombie issues - in_progress but no active agent

**Evidence:**
- `bd list --status in_progress | wc -l` = 49 issues
- `orch status` shows only 5 active agents
- 49 - 5 = 44 zombie issues stuck in in_progress with no agent working on them

**Source:**
- `orch status --json` output showing 5 agents
- `bd list --status in_progress` output showing 49 issues

**Significance:** These zombies are caused by:
1. Agents completing but `orch complete` not being run (most common)
2. Agents crashing or being abandoned without cleanup
3. Sessions timing out without state transition

The system has no reconciliation mechanism to detect and clean up these orphaned states.

---

### Finding 3: NEEDS_REVIEW completions persist for closed issues

**Evidence:**
- `orch review` shows 10 NEEDS_REVIEW completions, all with error: "code files modified but no test execution evidence found"
- Example: `orch-go-ptvu` shows as NEEDS_REVIEW but `bd show orch-go-ptvu` shows `Status: closed`
- The completion verification runs against workspace artifacts but doesn't check if the beads issue is already closed

**Source:**
- `cmd/orch/review.go:577-627` - Review logic doesn't filter by issue status
- `pkg/verify/test_evidence.go:21-40` - Skills requiring test evidence include systematic-debugging but the issue type may have changed

**Significance:** Workspaces persist even after issues are closed. The review command verifies workspace state without checking issue state, causing stale NEEDS_REVIEW results that clutter the output and confuse the orchestrator.

---

### Finding 4: 712 workspaces accumulated in .orch/workspace/

**Evidence:**
- `ls .orch/workspace/ | wc -l` = 712 directories
- `orch review` reports 382 completions (subset of 712)
- Workspaces never cleaned up after completion

**Source:**
- `.orch/workspace/` directory listing
- `cmd/orch/main.go:3536-3542` - Complete cleans tmux window but not workspace

**Significance:** Workspace accumulation causes:
1. Slow `orch status` (reads all 712 workspaces to find beads ID matches)
2. Stale verification results (old workspaces still processed by orch review)
3. Disk space consumption

---

## Synthesis

### Key Insights

1. **State transitions are incomplete** - Spawn updates status but not labels. Complete closes issue but doesn't clean workspace. No single place manages the full state lifecycle.

2. **No reconciliation mechanism exists** - The system has no way to detect when an issue is in_progress but has no active agent. This leads to zombie accumulation.

3. **Verification ignores beads state** - Completion verification checks workspace artifacts but doesn't check if the beads issue is already closed/resolved. This causes NEEDS_REVIEW to persist for already-closed work.

4. **Workspace lifecycle undefined** - Workspaces are created but never cleaned up, leading to accumulation and stale state.

### Answer to Investigation Question

The completion lifecycle is broken due to three gaps:

1. **Spawn gap:** `triage:ready` label not removed when issue transitions to `in_progress`. Fix: Add `RemoveLabel(beadsID, "triage:ready")` after `UpdateIssueStatus(beadsID, "in_progress")`.

2. **Reconciliation gap:** No mechanism to detect issues that are `in_progress` without active agents. Fix: Add `orch reconcile` or `orch stale` command that compares beads state vs OpenCode sessions.

3. **Verification gap:** `orch review` shows NEEDS_REVIEW for closed issues. Fix: Filter out closed issues from review, or mark their workspaces as stale.

---

## Structured Uncertainty

**What's tested:**

- ✅ 49 in_progress issues confirmed (verified: ran `bd list --status in_progress | wc -l`)
- ✅ 5 active agents confirmed (verified: ran `orch status --json`)
- ✅ RemoveLabel never called in cmd/orch (verified: ran `grep -rn "RemoveLabel" cmd/orch/`)
- ✅ orch-go-ptvu is closed but shows NEEDS_REVIEW (verified: ran `bd show orch-go-ptvu`)

**What's untested:**

- ⚠️ Whether daemon would actually re-spawn in_progress issues (status check prevents this, but behavior not tested)
- ⚠️ Performance impact of 712 workspaces on orch status (hypothesized from investigation findings but not benchmarked)
- ⚠️ Whether all 44 zombies are from missed completions vs crashes (would require log analysis)

**What would change this:**

- If RemoveLabel was called elsewhere (not in cmd/orch) - finding would be wrong
- If issues are in_progress for valid reasons (e.g., paused work) - zombie count would be lower

---

## Implementation Recommendations

### Recommended Approach ⭐: Epic with 4 children for state synchronization

Create an epic "Fix completion lifecycle state synchronization" with these children:

1. **Remove triage:ready on spawn** - Add `RemoveLabel(beadsID, "triage:ready")` after status update
2. **Add orch reconcile command** - Compare in_progress issues vs active agents, offer to fix
3. **Filter closed issues from review** - Check beads status before showing NEEDS_REVIEW
4. **Add workspace cleanup** - Option to clean workspaces older than N days

**Why this approach:**
- Each fix is independent and can be tested separately
- Addresses root causes identified in findings
- Minimal risk - adds new behavior without changing existing flows

**Trade-offs accepted:**
- Not implementing automatic cleanup (requires user confirmation)
- Not adding real-time state sync (daemon polling is sufficient)

**Implementation sequence:**
1. Remove triage:ready on spawn (quick fix, prevents future issues)
2. Add orch reconcile command (enables cleanup of existing zombies)
3. Filter closed issues from review (reduces noise)
4. Add workspace cleanup (optional, long-term maintenance)

### Alternative Approaches Considered

**Option B: Event-driven state machine**
- **Pros:** Guaranteed state consistency, real-time sync
- **Cons:** Major refactor, requires daemon changes, complex
- **When to use instead:** If state bugs continue after targeted fixes

**Option C: Periodic reconciliation daemon**
- **Pros:** Automatic cleanup, no manual intervention
- **Cons:** Hidden behavior, may close issues unexpectedly
- **When to use instead:** If manual reconciliation proves too tedious

**Rationale for recommendation:** Option A (targeted fixes) addresses immediate pain points with minimal risk. Can evolve to Option B/C if issues persist.

---

### Implementation Details

**What to implement first:**
- Remove triage:ready on spawn (prevents new zombies)
- This is the highest-impact fix - stops the bleeding

**Things to watch out for:**
- ⚠️ Labels may be added for other purposes (e.g., area:auth) - only remove triage:ready
- ⚠️ Issue may not have triage:ready label if spawned directly - handle gracefully
- ⚠️ RemoveLabel may fail if beads daemon is down - don't block spawn on failure

**Areas needing further investigation:**
- Why some issues have NEEDS_REVIEW when already closed - may need deeper analysis
- Whether workspace cleanup should be automatic vs manual
- Performance impact of 712 workspaces on orch status

**Success criteria:**
- ✅ After spawn, issue has in_progress status and no triage:ready label
- ✅ `orch reconcile` identifies and can fix zombie issues
- ✅ `orch review` doesn't show NEEDS_REVIEW for closed issues
- ✅ `orch status` performance acceptable with large workspace count

---

## References

**Files Examined:**
- `cmd/orch/main.go:1280-1360` - Spawn status update logic
- `cmd/orch/main.go:3270-3593` - Complete command implementation
- `cmd/orch/review.go:570-650` - Review display logic
- `pkg/verify/test_evidence.go:1-311` - Test evidence verification
- `pkg/beads/client.go:590-591` - RemoveLabel function

**Commands Run:**
```bash
# Count in_progress issues
bd list --status in_progress | wc -l  # 49

# Count active agents
orch status --json | jq '.swarm.active'  # 5

# Check if RemoveLabel is used
grep -rn "RemoveLabel" cmd/orch/  # empty

# Check specific issue
bd show orch-go-ptvu  # Status: closed
bd show orch-go-hop2  # Status: in_progress, Labels: [triage:ready]

# Count workspaces
ls .orch/workspace/ | wc -l  # 712
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-25-design-orchestrator-completion-lifecycle-two.md` - Prior completion lifecycle design
- **Workspace:** `.orch/workspace/og-work-completion-lifecycle-broken-29dec/` - This session

---

## Investigation History

**[2025-12-29 08:45]:** Investigation started
- Initial question: Why are 49 issues stuck in in_progress with triage:ready labels?
- Context: Spawned to design epic for state synchronization fixes

**[2025-12-29 09:00]:** Findings gathered
- Discovered RemoveLabel never called after spawn
- Found 44 zombie issues (49 in_progress vs 5 active agents)
- Found NEEDS_REVIEW persisting for closed issues

**[2025-12-29 09:15]:** Investigation completed
- Status: Complete
- Key outcome: Epic designed with 4 children for state synchronization fixes
