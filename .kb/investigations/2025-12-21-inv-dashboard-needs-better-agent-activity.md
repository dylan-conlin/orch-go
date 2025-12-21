<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Confidence:** [Level] ([Percentage]) - [Key limitation in one phrase]

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

# Investigation: Dashboard Agent Activity Visibility

**Question:** What agent activity indicators would most effectively solve the "can't tell if agents are working" problem?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** design-session agent
**Phase:** Design Synthesis
**Next Step:** Present findings to Dylan and determine best solution approach
**Status:** In Progress
**Confidence:** High (85%)

---

## Findings

### Finding 1: Current Dashboard Shows Only Static Status

**Evidence:**

- Agent cards display status badge (active/completed/abandoned) and pulse indicator
- Duration shown as time since spawn (e.g., "2h15m")
- No indication of recent activity or what agent is currently doing
- Agent appears "active" even if it hasn't done anything in 30+ minutes

**Source:**

- `/web/src/lib/components/agent-card/agent-card.svelte:45-60` - Status rendering
- `/web/src/lib/stores/agents.ts:15-31` - Agent interface (no activity timestamp)

**Significance:** Users can't distinguish between actively working agents and stuck/idle agents, leading to uncertainty about whether to wait or intervene.

---

### Finding 2: Rich Data Already Available But Not Surfaced

**Evidence:**

- SSE stream from OpenCode provides `session.status` events with busy/idle state transitions
- Agentlog (`~/.orch/events.jsonl`) tracks spawn/complete/error lifecycle events with timestamps
- Beads comments exist but are not exposed in dashboard API
- OpenCode SSE includes message events but dashboard doesn't capture them

**Source:**

- `/pkg/opencode/sse.go:86-135` - SSE event parsing
- `/pkg/opencode/monitor.go:136-189` - Event handling
- `/pkg/events/logger.go:13-23` - Event types available
- `/cmd/orch/serve.go:176-250` - SSE proxy (forwards but doesn't enhance)

**Significance:** The infrastructure for real-time activity tracking exists; it's a frontend presentation problem, not a data availability problem.

---

### Finding 3: Dashboard Already Has Two Event Streams

**Evidence:**

- SSE panel shows OpenCode events (session.status) in real-time
- Agentlog panel shows lifecycle events (spawn/complete/error) in real-time
- Both streams update automatically when connected
- Neither stream is agent-specific or easily filterable by agent

**Source:**

- `/web/src/routes/+page.svelte:288-368` - Event panels implementation
- `/web/src/lib/stores/agentlog.ts:76-140` - Agentlog SSE connection
- `/web/src/lib/stores/agents.ts:124-218` - OpenCode SSE connection

**Significance:** Event data exists but is presented globally rather than per-agent, making it hard to track individual agent activity.

---

### Finding 4: Agent Cards Are Compact But Limited

**Evidence:**

- Cards display ID, skill, beads_id, duration, synthesis (for completed)
- Cards use 2px layout with minimal spacing for density
- No expandable sections or click-to-see-more behavior
- Grid layout allows 4-5 agents visible simultaneously

**Source:**

- `/web/src/lib/components/agent-card/agent-card.svelte` - Full component (97 lines)
- `/web/src/routes/+page.svelte:266-284` - Grid layout

**Significance:** Current design prioritizes at-a-glance overview; adding activity details needs careful UX to maintain density.

---

### Finding 5: No Last Activity Timestamp Tracked

**Evidence:**

- Registry stores spawned_at, updated_at, completed_at, abandoned_at
- No field for "last activity" or "last message sent"
- SSE events have timestamps but aren't persisted per-agent
- Monitor tracks session state but not last update time per session

**Source:**

- `/pkg/registry/registry.go` (Agent struct - implied from API)
- `/pkg/opencode/monitor.go:11-17` - SessionState (has LastUpdated but not exposed via API)

**Significance:** Backend would need to track and expose last activity timestamp for frontend to display it.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

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

- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
