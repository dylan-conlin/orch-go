<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Light-tier spawns bypass worker detection because they inject context into the prompt instead of reading SPAWN_CONTEXT.md, even though the workspace file exists.

**Evidence:** Zero `context_usage` or worker-specific metrics in coaching-metrics.jsonl despite caching fix being correctly implemented. This session's workspace (`og-inv-verify-worker-metrics-17jan-c152/`) contains SPAWN_CONTEXT.md but no `read` tool call was made to it.

**Knowledge:** Worker detection relies on observing `read` tool calls to SPAWN_CONTEXT.md or file paths in `.orch/workspace/`. Light-tier spawns embed context in prompt, bypassing this signal. Additionally, context_usage only emits every 50 tool calls (verification with 10 calls was insufficient even if detection worked).

**Next:** Modify spawn template to force agents to read SPAWN_CONTEXT.md at session start for all spawn tiers. This aligns light-tier spawns with existing detection signals.

**Promote to Decision:** recommend-no (tactical fix, not architectural choice)

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

# Investigation: Verify Worker Metrics Perform 10

**Question:** Does the coaching-metrics.jsonl file log `context_usage` metric entries for worker sessions after the recent fix?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Initial State - Metrics file exists with various metric types but no context_usage

**Evidence:** The coaching-metrics.jsonl file exists and contains entries for `action_ratio`, `analysis_paralysis`, and `compensation_pattern` metric types from recent sessions (ses_432bab772ffeWNx8eoSr9ZXss3 and ses_432b46404ffe5r6DLiYLIRoABj). No `context_usage` entries visible in last 5 lines.

**Source:** `tail -5 ~/.orch/coaching-metrics.jsonl`

**Significance:** Establishes baseline - if context_usage metrics appear after 10 tool calls, the fix is working.

---

### Finding 2: detectWorkerSession caching fix IS implemented correctly

**Evidence:**
- Lines 1233-1236: Only returns early if `cached === true` (not if cached at all)
- Lines 1264-1267: Only caches positive results with `workerSessions.set(sessionId, true)` when isWorker is true
- This matches the recommended fix from the prior investigation

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts:1231-1272`

**Significance:** The fix was correctly implemented, ruling out the caching bug as the cause of missing metrics.

---

### Finding 3: Worker detection requires reading SPAWN_CONTEXT.md or accessing .orch/workspace/ paths

**Evidence:**
Detection signals in coaching.ts (lines 1240-1262):
1. `read` tool accessing file ending with `SPAWN_CONTEXT.md`
2. Any tool accessing files containing `.orch/workspace/` in path

My session's workspace exists at `og-inv-verify-worker-metrics-17jan-c152/` with SPAWN_CONTEXT.md, but I NEVER read it because:
- Light-tier spawns inject context directly into the prompt
- My investigation file is at `.kb/investigations/`, not `.orch/workspace/`
- None of my 10+ file reads accessed workspace paths

**Source:**
- `coaching.ts:1240-1262` - detection signals
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-verify-worker-metrics-17jan-c152/` - has SPAWN_CONTEXT.md but never read

**Significance:** This is the root cause! Light-tier spawns bypass the worker detection because their context is injected into the prompt rather than read from SPAWN_CONTEXT.md file.

---

### Finding 4: context_usage metric only emits at 50 tool call intervals

**Evidence:** Lines 1116 in coaching.ts: `if (state.totalToolCalls % 50 === 0 || state.estimatedTokensUsed > CONTEXT_WARNING_THRESHOLD)`

This means context_usage only emits when:
- Tool call count is exactly divisible by 50 (50, 100, 150, etc.)
- OR estimated tokens exceed 80k threshold

With only 15-20 tool calls, I would never trigger this metric even if detected as worker.

**Source:** `coaching.ts:1114-1130`

**Significance:** Even if worker detection was fixed, context_usage would not appear until 50 tool calls. The verification task's 10 tool calls was insufficient.

---

## Synthesis

**Key Insights:**

1. **Caching fix implemented but detection still fails** - The detectWorkerSession caching fix from the prior investigation was correctly implemented (only cache positive results). However, zero worker-specific metrics exist in the file, indicating detection itself is failing.

2. **Light-tier spawns bypass worker detection** - Light-tier spawns (like this one) inject context directly into the conversation prompt rather than having agents read SPAWN_CONTEXT.md. Since detection relies on observing a `read` tool call to SPAWN_CONTEXT.md, light-tier workers are never detected.

3. **Metric emission thresholds prevent early verification** - context_usage only emits every 50 tool calls OR when tokens exceed 80k. The verification task's 10 tool calls was fundamentally insufficient to trigger the metric even if detection had worked.

**Answer to Investigation Question:**

**Does the coaching-metrics.jsonl file log `context_usage` metric entries for worker sessions after the recent fix?**

**NO.** Zero `context_usage` (or any worker-specific) metrics exist in the file despite the caching fix being implemented.

The root cause is NOT the caching bug (that was fixed). The root cause is that light-tier spawns bypass worker detection entirely because they inject context into the prompt instead of having agents read SPAWN_CONTEXT.md from the workspace directory.

Additionally, even if detection worked, context_usage only emits at 50 tool call intervals, making 10 tool calls insufficient for verification.

---

## Structured Uncertainty

**What's tested:**

- ✅ Zero `context_usage` metrics exist in file (verified: `grep '"metric_type":"context_usage"'` returned 0 matches)
- ✅ Zero worker-specific metrics exist (verified: `grep -E "tool_failure_rate|time_in_phase|commit_gap"` returned empty)
- ✅ Caching fix is implemented correctly (verified: read coaching.ts lines 1231-1272, confirms only-cache-positive logic)
- ✅ Light-tier spawn has workspace with SPAWN_CONTEXT.md but I never read it (verified: ls workspace directory, no read tool calls to it)

**What's untested:**

- ⚠️ Whether explicitly reading SPAWN_CONTEXT.md would trigger detection (not tested - would require spawn redesign)
- ⚠️ Whether non-light-tier spawns properly read SPAWN_CONTEXT.md (no observation of other spawn types in this session)
- ⚠️ Whether reaching 50 tool calls would emit context_usage (did not perform 50 tool calls)

**What would change this:**

- Finding would be wrong if there's another detection signal we haven't discovered
- Finding would be wrong if light-tier spawns DO read SPAWN_CONTEXT.md in some configurations
- Finding would be wrong if plugin hasn't been reloaded since fix was applied (check server restart time)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Force SPAWN_CONTEXT.md read for all spawns** - Modify spawn template to include explicit instruction for agents to read SPAWN_CONTEXT.md at session start, even for light-tier spawns.

**Why this approach:**
- Maintains consistency with existing detection signals (no plugin changes needed)
- Light-tier spawns already have workspace with SPAWN_CONTEXT.md file
- Simple change to spawn templates, not complex detection logic

**Trade-offs accepted:**
- Slightly more context consumed by reading the file (marginal, SPAWN_CONTEXT already embedded in prompt)
- Requires spawn template changes in orch-go

**Implementation sequence:**
1. Add to spawn template: "First, read your SPAWN_CONTEXT.md from [workspace path]"
2. Verify worker detection triggers after the read
3. Verify context_usage metrics appear after 50 tool calls

### Alternative Approaches Considered

**Option B: Add session ID registration at spawn time**
- **Pros:** Would definitively mark workers without relying on tool call observation
- **Cons:** Requires modifying plugin initialization or adding API endpoint
- **When to use instead:** If template-based approach proves unreliable

**Option C: Lower context_usage emission threshold**
- **Pros:** Would allow verification with fewer tool calls
- **Cons:** Doesn't address root cause (detection failure); just makes verification easier
- **When to use instead:** After detection is fixed, if 50-call interval is too infrequent

**Rationale for recommendation:** Option A is the simplest fix that aligns spawns with existing detection logic. The workspace and SPAWN_CONTEXT.md already exist for light-tier spawns; agents just need to read the file.

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
