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

# Investigation: Persist Activity Feed Completed Agents

**Question:** Why is activity feed only visible while agent is 'Processing', and how can we persist it for completed agents?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** feature-impl agent
**Phase:** Implementing
**Next Step:** Fix missing Tool/State fields in backend endpoint
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Hybrid SSE + API architecture already implemented

**Evidence:** 
- `sessionHistory` store exists in `web/src/lib/stores/agents.ts:429-518`
- `activity-tab.svelte:42-75` calls `fetchHistory()` on session change
- Backend endpoint `/api/session/:sessionID/messages` exists in `cmd/orch/serve_agents.go:1384-1461`
- Frontend endpoint registered in `cmd/orch/serve.go:360`

**Source:** 
- `web/src/lib/stores/agents.ts` 
- `web/src/lib/components/agent-detail/activity-tab.svelte`
- `cmd/orch/serve_agents.go:1384-1461` (handleSessionMessages)
- Prior investigation: `.kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md`

**Significance:** The architecture for persisting activity feed (hybrid SSE for real-time + API for historical) was already designed and partially implemented. This is not a greenfield task - we're fixing an incomplete implementation.

---

### Finding 2: Backend response missing Tool and State fields

**Evidence:**
- Backend creates `PartDetails` struct (serve_agents.go:1443-1448) with only ID, Type, Text, SessionID
- Frontend needs Tool field (activity-tab.svelte:172 - `part.tool`)
- Frontend needs State field with input/output (activity-tab.svelte:156 - `part.state?.input`, line 161 - `part.state.output`)
- OpenCode `MessagePart` type includes Tool (types.go:125) and State (types.go:127) fields

**Source:**
- `cmd/orch/serve_agents.go:1443-1448` (incomplete transformation)
- `web/src/lib/components/agent-detail/activity-tab.svelte:150-172` (tool display logic)
- `pkg/opencode/types.go:119-137` (MessagePart type definition)

**Significance:** The endpoint returns data, but the transformation is incomplete. Tool calls appear in the feed but without tool names, descriptions, or output - making the activity feed less useful for completed agents.

---

### Finding 3: Frontend merge logic already handles deduplication

**Evidence:**
- `activity-tab.svelte:220-244` implements merge logic: historical events first, then SSE events
- Deduplication by event ID (line 226: `!seenIds.has(event.id)`)
- Both stores return events in SSE-compatible format with `id` field

**Source:** `web/src/lib/components/agent-detail/activity-tab.svelte:220-244`

**Significance:** Once backend returns complete data, no frontend changes needed for merge/dedup logic. The frontend is already built to handle hybrid SSE + API data.

---

## Synthesis

**Key Insights:**

1. **Implemented but incomplete** - The hybrid SSE + API architecture was designed in January 2026 and implemented, but the backend transformation is incomplete (missing Tool/State fields).

2. **Frontend is ready** - The dashboard already has merge/dedup logic, loading states, and display logic for tool calls. No frontend changes needed once backend is fixed.

3. **Single-point fix** - The issue is localized to one function: `handleSessionMessages` in serve_agents.go needs to populate Tool and State fields when transforming OpenCode MessagePart to SSE-compatible format.

**Answer to Investigation Question:**

Activity feed is only visible while agent is 'Processing' because the backend endpoint returns incomplete data - it includes event IDs and text but omits tool names and states. The frontend's tool display logic (formatToolCall, extractToolArg) expects `part.tool` and `part.state` fields that the backend doesn't populate. Fix: Update serve_agents.go:1443-1448 to copy Tool and State from OpenCode MessagePart to the response PartDetails.

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

**Complete backend transformation** - Add Tool and State fields to PartDetails response in handleSessionMessages

**Why this approach:**
- Minimal change: only touches serve_agents.go:1443-1448 (5 lines)
- Frontend already handles the data format correctly
- No new code needed - just populate existing OpenCode fields

**Trade-offs accepted:**
- None - this is the obvious fix

**Implementation sequence:**
1. Update PartDetails struct creation to include `Tool: part.Tool`
2. Add State transformation: convert OpenCode ToolState to response ToolState struct
3. Test with a completed agent to verify tool calls display correctly

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
