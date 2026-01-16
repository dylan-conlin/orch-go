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

# Investigation: Dashboard Add Message Input Agent

**Question:** How to add message input to dashboard for agent communication?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Dashboard Agent
**Phase:** Investigating
**Next Step:** Document findings and design implementation
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Backend API already exists via OpenCode client

**Evidence:** The `orch send` command uses `client.SendMessageAsync(sessionID, message, "")` which makes a POST to `/session/{sessionID}/prompt_async` with payload: `{"parts": [{"type": "text", "text": content}], "agent": "build"}`

**Source:** 
- `pkg/opencode/client.go:229-263` - SendMessageAsync implementation
- `cmd/orch/send_cmd.go:96-102` - Usage in orch send command

**Significance:** No new backend API needed - can reuse existing OpenCode client pattern. Dashboard just needs to construct the same POST request to the OpenCode server.

---

### Finding 2: Activity feed is a Svelte component with terminal-style layout

**Evidence:** Activity tab shows events in a scrollable feed with auto-scroll, filtering by type (text/tool/reasoning/step), and displays historical events from API + real-time SSE events.

**Source:** `web/src/lib/components/agent-detail/activity-tab.svelte:272-368` - Feed rendering and layout

**Significance:** Input should be positioned at bottom of existing feed container, chat-style. Feed handles scrolling internally, so input needs to be outside the scrollable area.

---

### Finding 3: Agent state includes session_id and is_processing flags

**Evidence:** Agent type has `session_id` (OpenCode session) and `is_processing` (boolean for actively generating response) fields. Agents store also tracks `status: 'active' | 'completed' | ...`.

**Source:** `web/src/lib/stores/agents.ts:30-68` - Agent interface and state tracking

**Significance:** Input should be disabled when `agent.status !== 'active'` or when agent doesn't have a `session_id`. The `is_processing` flag can show loading state during message send.

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

**Add Message Input Component to Activity Tab** - Modify `activity-tab.svelte` to include a message input field at the bottom, using the same OpenCode API pattern as `orch send`.

**Why this approach:**
- Reuses proven API pattern from `orch send` - no new backend needed
- Keeps message input contextual to the activity feed (chat-style UX)
- Leverages existing agent state (session_id, is_processing) for disabled state

**Trade-offs accepted:**
- Input is inside activity-tab component (not shared across tabs) - acceptable since messaging only makes sense in activity context
- No real-time typing indicators - acceptable for v1, can add later via SSE

**Implementation sequence:**
1. Add message input component to activity-tab.svelte (bottom of flex container, after feed)
2. Implement send function using fetch POST to OpenCode `/session/{session_id}/prompt_async`
3. Handle Enter to send, Shift+Enter for newline (via textarea onkeydown)
4. Disable input when agent.status !== 'active' or !agent.session_id
5. Show sent messages in feed (optimistic update or wait for SSE)

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
