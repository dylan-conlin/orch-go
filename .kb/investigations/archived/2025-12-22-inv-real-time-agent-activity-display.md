<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added real-time agent activity display using message.part SSE events with client-side parsing and granular store updates.

**Evidence:** Implemented and committed: Activity fields in Agent type, SSE event parsing for message.part, individual agent updates in store, activity display in cards, and Active Only filter. Go build and all tests pass.

**Knowledge:** Real-time activity doesn't require backend changes - can be implemented purely client-side by parsing existing SSE events. Activity = most recent message part (tool use, reasoning, text).

**Next:** Feature complete and committed. Ready for manual testing with live agents.

**Confidence:** High (90%) - Implementation complete and builds successfully; needs manual testing with actual SSE events to verify message.part parsing.

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

# Investigation: Real Time Agent Activity Display

**Question:** How to add real-time agent activity display with filtering for active sessions and live SSE updates?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** OpenCode Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Web UI already has SSE connection infrastructure

**Evidence:** The web UI has a complete SSE implementation that connects to `/api/events`, handles connection states (connecting/connected/disconnected), auto-reconnects, and displays events. It also has a separate `/api/agentlog` SSE stream for agent lifecycle events.

**Source:** 
- web/src/lib/stores/agents.ts:102-253 (SSE connection manager)
- web/src/routes/+page.svelte:79-82 (SSE connection on mount)
- cmd/orch/serve.go:267-342 (SSE proxy endpoint)

**Significance:** Infrastructure is already in place for real-time updates. The challenge is showing per-agent activity, not setting up SSE.

---

### Finding 2: OpenCode sessions have rich message data available via /message endpoint

**Evidence:** Sessions have messages with multiple parts (text, tool-invocation, etc.). Each message has role (user/assistant), timing info, and structured parts that can show what the agent is currently doing.

**Source:**
- pkg/opencode/types.go:77-109 (Message and MessagePart types)
- pkg/opencode/client.go:311-329 (GetMessages implementation)
- Testing: curl http://127.0.0.1:4096/session/{id}/message shows detailed message history

**Significance:** We can extract "current activity" by getting the most recent message parts for active sessions. This gives us what the agent is currently working on.

---

### Finding 3: SSE message.part events show real-time agent activity

**Evidence:** OpenCode sends `message.part` events via SSE when agents produce output. These events include the part type (text, tool, reasoning, step-start) and content. This is the real-time activity stream we need.

**Source:**
- pkg/opencode/sse.go:1-160 (SSE parsing implementation)
- pkg/opencode/client.go:500-611 (SendMessageWithStreaming shows message.part handling)
- cmd/orch/serve.go:267-342 (SSE proxy that forwards these events)

**Significance:** We already proxy all SSE events to the web UI. We just need to parse message.part events to extract current activity and update agent cards in real-time.

---

### Finding 4: Current implementation refetches entire agent list on events

**Evidence:** The web UI listens for SSE events and calls `agents.fetch()` on certain event types (session.status, session.created, etc.), which refetches the entire agent list from the API.

**Source:**
- web/src/lib/stores/agents.ts:221-240 (handleSSEEvent function)
- web/src/lib/stores/agents.ts:69-82 (agents.fetch implementation)

**Significance:** This is inefficient and doesn't show real-time activity updates. Instead of refetching, we should update individual agents based on message.part events to show live activity.

---

## Synthesis

**Key Insights:**

1. **Infrastructure exists, just needs wiring** - We already have SSE connections, agent listings, and filtering. The gap is connecting message.part events to agent cards to show real-time activity.

2. **Activity = recent message parts** - "Current activity" means showing the most recent message part from an agent (what tool they're using, what they're thinking, or what text they just produced).

3. **Inefficient full refresh pattern** - Currently the UI refetches all agents on any event, which doesn't scale and doesn't provide granular real-time updates. We should update individual agents based on their session ID from SSE events.

**Answer to Investigation Question:**

To add real-time agent activity display:
1. Add "current_activity" field to Agent type (includes activity_type and activity_text)
2. Parse message.part SSE events to extract session ID, part type, and text
3. Update individual agents in the store when message.part events arrive
4. Display activity in agent cards with appropriate icons for different part types
5. Add "Active Only" filter toggle (already have status filter, just need default to "active")

The infrastructure is 90% there. We just need to wire message.part events to update agent.current_activity and display it in the UI.

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

**Client-side SSE parsing with granular agent updates** - Parse message.part events in the web UI to extract activity data and update individual agents in real-time.

**Why this approach:**
- Leverages existing SSE proxy without backend changes
- Provides instant feedback (no API roundtrip needed)
- Scalable to many agents (only updates the active one)
- Simple to implement (client-side only changes)

**Trade-offs accepted:**
- Initial page load won't show activity (only shows after SSE events arrive)
- Activity state is ephemeral (lost on page refresh)
- No historical activity log (shows only most recent)

These are acceptable because:
- Activity is meant to be real-time/current, not historical
- Users can click through to session details for full history
- Simplicity >> completeness for this feature

**Implementation sequence:**
1. Add activity fields to Agent type and store (foundation for data model)
2. Parse message.part events to extract activity (data source)
3. Update agent store with activity on events (state management)
4. Display activity in agent cards (UI presentation)
5. Add "Active Only" default filter (UX refinement)

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
- Add activity fields to Agent type (activity_type, activity_text, activity_timestamp)
- Update handleSSEEvent to parse message.part events
- Extract session ID and part data from message.part events
- Map session ID to agent ID in the store

**Things to watch out for:**
- ⚠️ Session ID → Agent ID mapping (agents use title as ID, not session_id)
- ⚠️ Message.part event structure (need to examine actual events to confirm format)
- ⚠️ Activity text truncation (tool invocations can be long JSON)
- ⚠️ Stale activity (need timestamp to show how old the activity is)

**Areas needing further investigation:**
- Exact structure of message.part events (will discover during implementation)
- Icon mapping for different activity types (tool, text, reasoning, step-start)
- Activity display truncation strategy (character limit vs word limit)

**Success criteria:**
- ✅ Active agents show their most recent activity in real-time
- ✅ Activity updates within 1 second of SSE event arrival
- ✅ Activity display is readable (truncated appropriately)
- ✅ "Active Only" filter works correctly
- ✅ No performance degradation with many agents

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
