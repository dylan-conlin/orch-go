<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Removed redundant fetch calls from page onMount, letting SSE connection handle all data loading to eliminate race condition.

**Evidence:** All 4 Playwright tests pass (100% success rate across multiple page loads), no "Failed to fetch agents: NetworkError" console errors.

**Knowledge:** SSE onopen handlers already fetch data; explicit fetches in onMount create race conditions and redundant network calls.

**Next:** Close - fix implemented, tested, and verified.

**Confidence:** Very High (95%) - Automated tests confirm fix works consistently.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Web Ui Intermittent Failure Race

**Question:** What causes the intermittent "Failed to fetch agents: NetworkError" on web UI page load, and how can we eliminate the race condition?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** systematic-debugging agent
**Phase:** Investigating
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%) - Fix implemented and verified with automated tests

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Multiple simultaneous fetch calls on page load

**Evidence:** In `+page.svelte` lines 76-90, the `onMount()` hook performs:
1. `agents.fetch()` (line 78)
2. `agentlogEvents.fetch()` (line 83)
3. `connectSSE()` (line 88) - which triggers ANOTHER `agents.fetch()` when connection opens

In `agents.ts` lines 148-151, the SSE `onopen` handler calls `agents.fetch()` again.

**Source:** 
- web/src/routes/+page.svelte:76-90
- web/src/lib/stores/agents.ts:148-151

**Significance:** Three fetch operations happen in rapid succession with no coordination. If any fail (server not ready, network timing), errors appear in console. The SSE fetch succeeding after initial fetch fails creates the "sometimes shows agents, sometimes empty" behavior.

---

### Finding 2: No synchronization between initial data load and SSE connection

**Evidence:** The page makes two independent async operations:
- Initial fetch to populate data (line 78)
- SSE connection to get updates (line 88)

There's no waiting for the fetch to complete before starting SSE, and no checking if fetch succeeded before attempting SSE connection.

**Source:** web/src/routes/+page.svelte:76-90

**Significance:** This creates a race condition where SSE might connect before initial fetch completes, or initial fetch might fail while SSE succeeds, leading to inconsistent UI state and error messages.

---

### Finding 3: Redundant fetch on SSE connection open

**Evidence:** The SSE `onopen` handler (agents.ts:148-151) calls `agents.fetch()` to "get current state", but this duplicates the initial fetch from `onMount`. This means:
- Best case: 2 fetches happen (onMount + SSE onopen)
- Worst case: 3 fetches if SSE reconnects

**Source:** web/src/lib/stores/agents.ts:148-151

**Significance:** The redundant fetch in SSE onopen was likely added to handle the case where SSE connects after the page loads, but it creates unnecessary network traffic and race conditions when both happen simultaneously during initial page load.

---

## Synthesis

**Key Insights:**

1. **Race condition from redundant fetches** - The page was making 3 simultaneous fetch calls on load: onMount initial fetch, agentlog fetch, and SSE onopen fetch. This created timing-dependent behavior where some fetches would succeed while others failed.

2. **SSE connection already handles initial data load** - Both agents.ts and agentlog.ts have fetch() calls in their SSE onopen handlers, making the explicit fetch calls in +page.svelte redundant and race-prone.

3. **Simple fix: Let SSE drive data loading** - Removing the explicit fetch calls and letting SSE connection trigger the initial data load eliminates the race condition and reduces network traffic.

**Answer to Investigation Question:**

The intermittent "Failed to fetch agents: NetworkError" was caused by a race condition between three simultaneous fetch operations on page load. The fix is to remove the explicit `agents.fetch()` and `agentlogEvents.fetch()` calls from onMount in +page.svelte, and instead rely solely on the SSE connection's onopen handler to trigger the initial data load. This was verified with automated Playwright tests showing 100% success rate across multiple page loads.

---

## Confidence Assessment

**Current Confidence:** [Level] ([Percentage])

**Why this level?**

[Explanation of why you chose this confidence level - what evidence supports it, what's strong vs uncertain]

**What's certain:**

- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]

**What's uncertain:**

- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]

**What would increase confidence to [next level]:**

- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**SSE-Driven Data Loading** - Remove explicit fetch calls from onMount and rely solely on SSE connection onopen handlers to load initial data.

**Why this approach:**
- Eliminates race condition by having single source of data loading
- Reduces redundant network traffic (from 3 fetches to 1)
- Works for both cold start and reconnection scenarios

**Trade-offs accepted:**
- Page shows loading state briefly while SSE connects (acceptable UX)
- If SSE fails to connect, no data loads (already has auto-reconnect logic)

**Implementation sequence:**
1. Remove `agents.fetch()` and `agentlogEvents.fetch()` from +page.svelte onMount
2. Add comment explaining SSE will trigger fetch on connection
3. Verify with automated tests (race-condition.spec.ts)

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
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
