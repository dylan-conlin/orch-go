<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard needs error count in stats bar, last activity on agent cards, and larger event panels for better visibility.

**Evidence:** Error events exist but are buried in 32px-high collapsed panels; active agents show no activity detail; errorEvents and agentlogEvents stores already provide necessary data.

**Knowledge:** All required data exists in frontend stores - no backend changes needed; improvements are purely additive UI enhancements.

**Next:** Implement three UI changes: (1) add error count to stats bar, (2) show last event on active agent cards, (3) expand event panel height and font size.

**Confidence:** High (85%) - Clear gaps identified with specific solutions; main uncertainty is event-to-agent correlation by session_id.

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

**Question:** What specific improvements to the dashboard UI would provide better visibility into agent activity, real-time status, recent events, and error states?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** orch-go-36b
**Phase:** Investigating
**Next Step:** Analyze current implementation for gaps in error visibility, status detail, and event presentation
**Status:** In Progress
**Confidence:** Medium (60-79%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Error events exist but are not prominently visible

**Evidence:** 
- Error events are tracked in `errorEvents` derived store (web/src/lib/stores/agentlog.ts:65-67)
- Error events display in collapsed event panel with ❌ icon and red text for error message (web/src/routes/+page.svelte:323-325)
- No error count in stats bar (only active/completed/abandoned counts shown)
- Errors are mixed with other events in 32px-high panel showing only 10 events

**Source:** 
- web/src/lib/stores/agentlog.ts:65-67 (errorEvents derived store)
- web/src/routes/+page.svelte:106-133 (event icon/label functions)
- web/src/routes/+page.svelte:156-204 (stats bar with no error count)
- web/src/routes/+page.svelte:312-336 (collapsed event panel)

**Significance:** Users cannot quickly see if agents are encountering errors; errors are discoverable only by scrolling through event logs.

---

### Finding 2: Active agents show only status, no current activity detail

**Evidence:**
- Agent cards show status badge ("active") with green pulse indicator
- No current phase information displayed (e.g., "Planning", "Implementing", "Testing")
- No recent activity or last event timestamp
- Backend provides session_id but no phase or activity data in AgentAPIResponse

**Source:**
- web/src/lib/components/agent-card/agent-card.svelte:45-60 (status badge and pulse)
- cmd/orch/serve.go:106-117 (AgentAPIResponse structure)
- No phase tracking in Agent interface (web/src/lib/stores/agents.ts:4-31)

**Significance:** Users cannot distinguish between actively working agents and stuck/idle agents; all active agents look identical.

---

### Finding 3: Event panels are collapsed and hard to scan

**Evidence:**
- Both event panels have max-height of 32px (8rem/128px) with overflow-y-auto (web/src/routes/+page.svelte:312, 348)
- Only displays last 10 events per panel (web/src/routes/+page.svelte:313, 349)
- Events show in tiny 10px font (text-xs) with minimal information
- Two separate panels (Agent Lifecycle and SSE Stream) split attention
- Events auto-scroll in reverse chronological order, making it hard to follow

**Source:**
- web/src/routes/+page.svelte:288-368 (collapsed event panels)
- web/src/routes/+page.svelte:312-336 (Agent Lifecycle panel with max-h-32)
- web/src/routes/+page.svelte:340-367 (SSE Stream panel with max-h-32)

**Significance:** Recent activity is difficult to track; users must actively scroll tiny panels to understand what's happening.

---

## Synthesis

**Key Insights:**

1. **Error visibility is the most critical gap** - Error events are tracked but buried in collapsed panels, making system health invisible at a glance (Finding 1). Adding error count to stats bar and highlighting errors would provide immediate health visibility.

2. **Active agent cards lack context** - All active agents show identical status with no indication of current phase or recent activity (Finding 2). Users can't distinguish productive agents from stuck ones without manually inspecting each.

3. **Event presentation prioritizes space over information** - Collapsed 32px panels with 10px font optimize for screen real estate but sacrifice visibility of recent activity (Finding 3). Users need event information to be scannable at a glance.

**Answer to Investigation Question:**

The dashboard needs three specific improvements to provide better agent activity visibility:

1. **Error visibility:** Add error count to stats bar (alongside active/completed/abandoned) and highlight recent errors prominently
2. **Status detail:** Show current phase or last activity for active agents (e.g., "Planning", "Implementing - 5m ago")
3. **Event presentation:** Expand event panels or provide condensed "recent activity" feed with larger text and better visual hierarchy

These improvements address the core visibility gaps (Findings 1-3) without requiring backend changes - all necessary data already exists in the stores.

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

**Incremental UI enhancements** - Add error visibility, last event display for agents, and expand event panels using existing data stores

**Why this approach:**
- Addresses all three visibility gaps (error, status detail, event presentation) without backend changes
- Uses existing errorEvents, agentlogEvents stores - no new data fetching required
- Can be implemented incrementally with immediate value from each change
- Low risk - purely additive UI changes

**Trade-offs accepted:**
- Won't show current phase from beads comments (would require backend parsing) - using "last event" instead
- Larger event panels reduce visible agent cards - acceptable for better visibility
- Error count includes all errors, not just recent (could add time filtering later)

**Implementation sequence:**
1. Add error count to stats bar - quick win, immediate visibility (Finding 1)
2. Add last event display to active agent cards - shows activity without expanding (Finding 2)
3. Expand event panels or add dedicated "Recent Activity" section - better event visibility (Finding 3)

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
- Stats bar: Add error count badge next to existing active/completed/abandoned counts using $errorEvents.length
- Agent cards: Display most recent agentlog event type and timestamp for each agent
- Event panels: Increase max-height from 32px to 64px and font from text-xs (10px) to text-sm (12px)

**Things to watch out for:**
- ⚠️ Need to correlate agentlog events to agents by session_id - may not always match
- ⚠️ Expanding event panels will push agent grid down - verify layout doesn't break on small screens
- ⚠️ Error count includes all historical errors - may want to filter to recent (last hour/day)

**Areas needing further investigation:**
- Backend support for current phase from beads comments (would require parsing `bd comment` output)
- Real-time agent health checks (detect stuck agents vs actively working)
- Event filtering by time range or type

**Success criteria:**
- ✅ Users can see error count at a glance without scrolling
- ✅ Active agents show last activity (event type + timestamp)
- ✅ Event panels are readable without squinting or scrolling
- ✅ Dashboard updates in real-time as new events arrive

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
