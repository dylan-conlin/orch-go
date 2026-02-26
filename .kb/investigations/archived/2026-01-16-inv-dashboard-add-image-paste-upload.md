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

# Investigation: Dashboard Add Image Paste Upload

**Question:** How should the dashboard support image paste/upload for sending images to agents?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Worker Agent
**Phase:** Investigating
**Next Step:** Complete codebase exploration and move to design phase
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Dashboard activity tab displays events but has no message input UI

**Evidence:** Activity tab (activity-tab.svelte) shows SSE events in a terminal-style feed with filters and auto-scroll, but contains no input field or send functionality. The component is read-only, focused on displaying agent activity.

**Source:** web/src/lib/components/agent-detail/activity-tab.svelte:1-369

**Significance:** Need to add a message input component to the activity tab UI, likely at the bottom of the panel, to support sending messages (text and images) to active agents.

---

### Finding 2: OpenCode send API already exists with text-only support

**Evidence:** OpenCode client (pkg/opencode/client.go) has SendMessageAsync() that POSTs to `/session/{sessionID}/prompt_async` with payload structure: `{"parts": [{"type": "text", "text": "..."}], "agent": "build"}`. The orch send command (cmd/orch/send_cmd.go) uses this API to send text messages to sessions.

**Source:** 
- pkg/opencode/client.go:229-263 (SendMessageAsync function)
- cmd/orch/send_cmd.go:78-114 (sendViaOpenCodeAPI function)

**Significance:** The API already supports a "parts" array structure, which is extensible for adding image parts. We'll need to extend both the client library and the dashboard to support `{"type": "image", "data": "base64..."}` parts alongside text parts.

---

### Finding 3: No /api/send endpoint exists in orch serve yet

**Evidence:** Searched serve.go for all API endpoints - found /api/agents, /api/events, /api/approve, /api/issues, etc., but no /api/send or message-sending endpoint. Current send functionality only exists in the CLI command (orch send).

**Source:** cmd/orch/serve.go:259-363 (API endpoint registration)

**Significance:** Need to create a new POST /api/send endpoint in serve.go that accepts session_id and parts (text + images), then calls the OpenCode client's SendMessageAsync with the parts payload.

---

### Finding 4: Dashboard already has image loading pattern via /api/file endpoint

**Evidence:** Screenshots tab loads images from workspace by fetching via /api/file?path=... endpoint, which returns file content (base64-encoded for binary files). Images are displayed using this endpoint as the img src.

**Source:** web/src/lib/components/agent-detail/screenshots-tab.svelte:48-66

**Significance:** For displaying pasted/uploaded images in the activity feed, we can either use data URLs (base64-encoded inline) or save to workspace and use /api/file. Data URLs are simpler for paste/upload preview but increase payload size.

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
