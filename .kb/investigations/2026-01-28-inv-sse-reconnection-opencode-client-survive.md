<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

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

# Investigation: SSE Reconnection in OpenCode Client to Survive Server Restarts

**Question:** How can the OpenCode client implement SSE reconnection to survive server restarts without losing agent work?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** Worker agent (investigation)
**Phase:** Investigating
**Next Step:** Test why reconnection isn't working despite built-in support
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** None
**Extracted-From:** Issue orch-go-20979
**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: Starting Exploration - Prior Context

**Evidence:** From Jan 26 investigation, agent sessions die when OpenCode server crashes/restarts because the SSE stream breaks. The `opencode run --attach` command uses a `for await (const event of events.stream)` loop at run.ts:154-158. When the SSE connection drops, the loop terminates and the client process exits.

**Source:** `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md` (Finding 3), orch-go issue orch-go-20979

**Significance:** This establishes the problem context - we need to implement reconnection logic so the SSE stream can automatically reconnect when the server comes back up, allowing agents to resume without losing work.

---

### Finding 2: SSE Client Already Has Reconnection Logic Built-In

**Evidence:** The OpenCode SDK's `createSseClient` function (serverSentEvents.gen.ts:78-239) already implements automatic reconnection:
- Line 100-232: while(true) loop that retries on connection failure
- Line 110-112: Sets `Last-Event-ID` header for event resumption
- Line 221-232: Error handler with exponential backoff (doubles delay each attempt)
- Line 230: Backoff capped at `sseMaxRetryDelay` (default 30000ms)
- Line 225-227: Only stops after `sseMaxRetryAttempts` (if specified)
- Line 220: Only exits loop on normal stream completion

Configuration options available:
- `sseDefaultRetryDelay` (default: 3000ms)
- `sseMaxRetryAttempts` (default: undefined = retry indefinitely)
- `sseMaxRetryDelay` (default: 30000ms)
- `sseSleepFn` (default: setTimeout wrapper)

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/sdk/js/src/v2/gen/core/serverSentEvents.gen.ts:78-239`

**Significance:** This is a major finding - **the SSE client already supports reconnection!** The question shifts from "how to implement reconnection" to "why isn't it working?" or "is it configured correctly?"

---

### Finding 3: run.ts Uses Default SSE Configuration (No Options Passed)

**Evidence:** In run.ts:154, the SDK is called with no options: `const events = await sdk.event.subscribe()`. Searching the codebase shows no use of `sseMaxRetryAttempts` or `sseDefaultRetryDelay` configuration anywhere in packages/opencode/src/.

This means the defaults are used:
- sseDefaultRetryDelay: 3000ms (3 second initial retry)
- sseMaxRetryAttempts: undefined (retry indefinitely)
- sseMaxRetryDelay: 30000ms (30 second max backoff)

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/cli/cmd/run.ts:154`, grep search across opencode source

**Significance:** The SSE client should already be retrying automatically with exponential backoff! If agents are dying on server restart (per Jan 26 investigation), either: (1) the retry logic isn't working as expected, (2) there's a different failure mode, or (3) the issue was misdiagnosed. Need to test actual behavior.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

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
