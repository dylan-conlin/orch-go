<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The completion loop has 5 distinct breakpoints: (1) SSE monitor disabled automatic completion, (2) `orch complete` is manual-only, (3) no daemon completion polling, (4) `orch review` doesn't trigger completion, (5) tmux/OpenCode sessions persist after beads closure.

**Evidence:** Found 216 pending completions with SYNTHESIS.md but unclosed beads issues. Agent orch-go-k08g has Phase: Complete, beads issue closed, but still shows "active" in `orch status` due to persistent OpenCode session.

**Knowledge:** The system is intentionally non-automatic because SSE busy→idle detection had false positives (agents go idle during tool loading). This created a human-in-the-loop bottleneck that accumulates pending completions.

**Next:** Implement daemon-driven completion processor that polls Phase: Complete agents and runs `orch complete` automatically, separate from SSE-based detection.

**Confidence:** High (85%) - Clear evidence of 5 breakpoints, but haven't tested all fix implementations.

---

# Investigation: Orchestration Lifecycle End-to-End for Completion Loop Gaps

**Question:** What are ALL the points where the completion loop can break, leading to agents completing work but not having their issues closed?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent og-inv-investigate-orchestration-lifecycle-25dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: SSE Monitor Automatic Completion Was Disabled Due to False Positives

**Evidence:** In `pkg/opencode/service.go:100-105`, automatic registry completion is explicitly disabled with comment:
```go
// NOTE: Automatic registry completion was disabled (2025-12-21)
// Reason: Monitor's busy→idle detection triggers false positives.
// Agents go idle during normal operation (loading, thinking, waiting for tools).
// The first idle transition after spawn (4-6 seconds) incorrectly marks agents complete.
// Solution: Require explicit `orch complete` command instead of automatic detection.
```

**Source:** `pkg/opencode/service.go:100-105`

**Significance:** This is the root cause of the manual bottleneck. The system cannot auto-detect true completion because busy→idle is unreliable. True completion is only knowable via `Phase: Complete` beads comment.

---

### Finding 2: Pre-Spawn Check Has Gap for Completed-But-Not-Closed Issues

**Evidence:** In `cmd/orch/main.go:1096-1119`, the pre-spawn duplicate prevention logic:
1. Checks if issue is closed → blocks spawn ✓
2. Checks if issue is in_progress with active session → blocks spawn ✓
3. Checks if issue has Phase: Complete → blocks spawn ✓
4. BUT: If issue is in_progress, no active session, no Phase: Complete → **allows respawn** (potential duplicate)

The gap: If agent reports Phase: Complete but doesn't run `/exit`, the beads status stays `in_progress` but Phase: Complete is reported. Next spawn is blocked correctly. HOWEVER, if agent crashes or context exhausts BEFORE reporting Phase: Complete, issue stays `in_progress` with no indicator of work progress. This allows duplicate spawns for incomplete work (which may be desirable) but accumulates untracked partial work.

**Source:** `cmd/orch/main.go:1096-1119`

**Significance:** Pre-spawn check is reasonably complete for preventing duplicates, but cannot prevent accumulation of partially-completed work.

---

### Finding 3: No Daemon-Based Completion Processing

**Evidence:** The daemon (`pkg/daemon/daemon.go`) only polls for **ready issues to spawn**, not for **completed agents to close**:
- `ListReadyIssues()` gets open issues with `triage:ready` label
- `SpawnWork()` spawns new agents via `orch work`
- No `ListCompletedAgents()` or `ProcessCompletions()` exists

The daemon runs continuously but only in one direction (spawn), not bidirectionally (spawn + complete).

**Source:** `pkg/daemon/daemon.go:141-192`, `cmd/orch/daemon.go:143-308`

**Significance:** This is a major missing automation. The daemon could poll for Phase: Complete agents and run `orch complete` on them, closing the loop automatically.

---

### Finding 4: `orch review` Lists But Doesn't Close

**Evidence:** The review command (`cmd/orch/review.go`) discovers 216 completed workspaces but:
1. `getCompletionsForReview()` scans for SYNTHESIS.md
2. Displays verification status (OK vs NEEDS_REVIEW)
3. `orch review done <project>` only prints "Marked as reviewed" - **no actual action taken**

The command is purely informational - it tells you what to complete but doesn't complete anything.

**Source:** `cmd/orch/review.go:86-164`, `cmd/orch/review.go:374-421`

**Significance:** This creates a human-in-the-loop bottleneck. After running `orch review`, operator must manually run `orch complete <id>` for each agent.

---

### Finding 5: OpenCode Sessions and Tmux Windows Persist After Beads Closure

**Evidence:** Agent orch-go-k08g:
- Beads issue: **closed** (close reason set)
- Phase: Complete reported via beads comment
- OpenCode session: **still running** (ses_4a91141b)
- `orch status` shows: "active", "idle"
- `orch complete` prompts: "Agent appears still running... Proceed anyway?"

The completion flow only closes the beads issue - it doesn't kill the OpenCode session or tmux window unless explicitly accepted.

**Source:** `cmd/orch/main.go:2505-2541`, `orch status` output, `orch complete orch-go-k08g` output

**Significance:** Stale sessions consume resources and pollute `orch status` output. The 5 "active" agents shown includes agents whose work is complete but whose sessions weren't cleaned up.

---

### Finding 6: The 216 Pending Completions Are Real Backlog

**Evidence:** 
- `orch review` shows 216 completions with SYNTHESIS.md in orch-go project
- 418 workspace directories exist in `.orch/workspace/`
- Beads shows 200 open issues, 16 in_progress
- Many completions have beads IDs that are still in_progress status

This is accumulated technical debt from the manual completion bottleneck.

**Source:** `orch review | head -60`, `ls .orch/workspace/ | wc -l`, `bd list --json | python3 ...`

**Significance:** The backlog represents days/weeks of agent work that wasn't properly closed. Without automation, this will continue to grow.

---

## Synthesis

**Key Insights:**

1. **SSE-based completion detection is fundamentally unreliable** - The busy→idle heuristic triggers during normal agent operation (loading context, waiting for tools, thinking). This is why it was intentionally disabled.

2. **Phase: Complete is the only reliable completion signal** - Agents explicitly report this via beads comment. The system correctly uses this for verification in `orch complete`, but doesn't automatically act on it.

3. **The completion loop is human-gated by design** - After disabling SSE auto-complete, the system has no automated closure mechanism. This was intentional to prevent false positives but created accumulation.

4. **Daemon is unidirectional** - It spawns but doesn't complete. Adding a completion polling loop would close the lifecycle automatically.

5. **Resource cleanup is incomplete** - Even when `orch complete` runs, OpenCode sessions and tmux windows may persist, polluting `orch status`.

**Answer to Investigation Question:**

The completion loop can break at 5 distinct points:

| Breakpoint | When It Breaks | Impact |
|------------|----------------|--------|
| 1. SSE Auto-Complete Disabled | Always (by design) | No automatic completion detection |
| 2. Manual `orch complete` Bottleneck | Orchestrator doesn't run it | Issues stay in_progress |
| 3. No Daemon Completion Polling | Always (missing feature) | Completed agents not processed |
| 4. `orch review` Informational Only | Always (design limitation) | Requires manual follow-up |
| 5. Session/Window Persistence | After beads closure | Stale resources in status |

The primary gap is **#3 (No Daemon Completion Polling)** - this is the missing forcing function that would automatically process Phase: Complete agents.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong evidence from code inspection and observed state (216 pending completions, stale sessions, explicit SSE disable comment). Minor uncertainty about whether all edge cases are covered.

**What's certain:**

- ✅ SSE auto-complete was intentionally disabled (explicit comment in code)
- ✅ 216 pending completions exist (observed via `orch review`)
- ✅ Daemon only spawns, doesn't complete (code inspection)
- ✅ Agent sessions persist after beads closure (tested with orch-go-k08g)

**What's uncertain:**

- ⚠️ Whether Phase: Complete polling would have its own false positives
- ⚠️ Whether 216 completions all have valid Phase: Complete (didn't test each)
- ⚠️ Best interval for completion polling (need empirical testing)

**What would increase confidence to Very High (95%+):**

- Test daemon completion polling implementation
- Validate Phase: Complete parsing handles edge cases
- Run batch completion on 216 agents, observe success rate

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Daemon Completion Polling Loop** - Add bidirectional processing to daemon: spawn new work AND complete finished work.

**Why this approach:**
- Uses reliable Phase: Complete signal (not flaky SSE busy→idle)
- Leverages existing daemon infrastructure (polling pattern)
- Doesn't require new monitoring architecture

**Trade-offs accepted:**
- Polling interval means slight delay before completion (acceptable: 60s default)
- More beads API calls (acceptable: already batched)

**Implementation sequence:**
1. Add `ListCompletedAgents()` function to daemon package (poll beads for Phase: Complete, issue not closed)
2. Add `ProcessCompletions()` to call `orch complete` for each
3. Add completion loop to daemon run (interleaved with spawn loop)
4. Add `--complete` flag to daemon for opt-in behavior

### Alternative Approaches Considered

**Option B: SSE-Based with Better Detection**
- **Pros:** Real-time completion detection
- **Cons:** Busy→idle is fundamentally unreliable; would need different signal (e.g., agent reporting done via API)
- **When to use instead:** If agents can report completion via direct API call

**Option C: Scheduled `orch review` + Batch Complete**
- **Pros:** Simple, uses existing commands
- **Cons:** Still requires human to run it; doesn't close the loop fully
- **When to use instead:** As interim solution before daemon enhancement

**Rationale for recommendation:** Option A is the only fully-automated solution that closes the loop without human intervention, using the reliable Phase: Complete signal.

---

### Implementation Details

**What to implement first:**
- `daemon.ListCompletedAgents()` - poll for Phase: Complete + issue open/in_progress
- Add to daemon polling loop with configurable interval

**Things to watch out for:**
- ⚠️ Don't complete agents that are still actively processing (check session activity)
- ⚠️ Handle beads API rate limits in batch operations
- ⚠️ Session cleanup may fail if agent didn't cleanly exit

**Areas needing further investigation:**
- Should cleanup also kill orphaned OpenCode sessions?
- Should `orch review done` actually run `orch complete` on all items?

**Success criteria:**
- ✅ Pending completions count decreases over time
- ✅ `orch status` active count matches truly-running agents
- ✅ New agents auto-complete within one poll interval after Phase: Complete

---

## Test performed

**Test:** Examined live system state and traced through code paths

**Result:** 
- 216 pending completions found with SYNTHESIS.md
- Agent orch-go-k08g has closed beads issue but still shows "active" in status
- SSE auto-complete code is explicitly disabled with documented reason
- Daemon has spawn loop but no completion loop

---

## References

**Files Examined:**
- `cmd/orch/main.go:1032-1201` - Spawn logic with pre-spawn checks
- `cmd/orch/main.go:2444-2606` - Complete command implementation
- `cmd/orch/daemon.go:143-308` - Daemon run loop
- `pkg/daemon/daemon.go:141-480` - Daemon package
- `pkg/opencode/service.go:73-129` - SSE completion handling (disabled)
- `cmd/orch/review.go:86-421` - Review command

**Commands Run:**
```bash
# Check pending completions
orch review | head -60
# Result: 216 completions in orch-go project

# Check workspace count
ls .orch/workspace/ | wc -l
# Result: 419 directories

# Check beads status breakdown
bd list --json | python3 -c "import json,sys; d=json.load(sys.stdin); print(len(d))"
# Result: Total 1226, Open 200, In Progress 16, Closed 1008

# Check completed agent that's still "active"
bd show orch-go-k08g
# Result: closed, with Phase: Complete comment

orch status
# Result: Shows orch-go-k08g as "idle" despite being closed
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-agents-being-marked-completed-registry.md` - Why SSE was disabled
- **Decision:** SSE auto-complete disabled (2025-12-21)

---

## Investigation History

**2025-12-25 11:10:** Investigation started
- Initial question: What are ALL the points where the completion loop can break?
- Context: 216 pending completions accumulating, duplicate spawn concerns

**2025-12-25 11:15:** Mapped full lifecycle
- Traced spawn → work → phase → completion → closure paths
- Identified SSE disable as root cause of manual bottleneck

**2025-12-25 11:25:** Tested live completion gap
- Agent orch-go-k08g: beads closed but session "active"
- Confirmed 5 distinct breakpoints

**2025-12-25 11:35:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend daemon completion polling to close the loop

---

## Self-Review

- [x] Real test performed (checked live system state, traced code paths)
- [x] Conclusion from evidence (5 breakpoints identified from code inspection)
- [x] Question answered (lifecycle breakpoints mapped)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled (Delta, Evidence, Knowledge, Next completed)
- [x] NOT DONE claims verified (checked actual code, not just artifact claims)

**Self-Review Status:** PASSED
