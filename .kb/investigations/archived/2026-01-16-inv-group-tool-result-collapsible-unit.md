<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Current activity tab renders each SSE event independently; tool calls and step-finish events appear as separate list items instead of grouped units.

**Evidence:** Activity-tab.svelte loops through agentEvents rendering each in its own container (lines 442-487); step-finish events get ✓ icon as separate items.

**Knowledge:** Grouping requires pre-processing agentEvents to correlate tool events with their related step-finish events, then rendering groups as collapsible containers instead of individual events.

**Next:** Implement grouped rendering with sequence-based correlation (tool followed by step-finish), preserve expand/collapse UX, validate with visual testing.

**Promote to Decision:** recommend-no (tactical UI improvement, not architectural pattern worth preserving globally)

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

# Investigation: Group Tool Result Collapsible Unit

**Question:** How should tool calls and their results be visually grouped as a single collapsible unit instead of appearing as separate list items?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None - moving to implementation
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Tool results are already displayed with tool calls

**Evidence:** Activity tab already shows tool output alongside tool invocation in the same container div (lines 464-485). Tool results are shown via `part.state.output` field with expand/collapse functionality.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/components/agent-detail/activity-tab.svelte:464-485`

**Significance:** Recent investigation (2026-01-16-inv-show-tool-result-preview-expand.md) implemented tool result preview. This means the current task is NOT about adding result display, but about grouping different event types.

---

### Finding 2: Multiple SSE event types exist for tool execution

**Evidence:** SSE events have different types: 'tool', 'tool-invocation', 'step-start', 'step-finish'. Each appears as a separate event in the stream. Current rendering shows each event type as a separate list item with its own icon (🔧 for tool, ▶️ for step-start, ✓ for step-finish).

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/stores/agents.ts:83-92` (getActivityIcon function in activity-tab.svelte)

**Significance:** The spawn context shows "tool" and "step-finish" as currently appearing separately. This suggests we need to correlate tool events with their corresponding step events and group them visually.

---

### Finding 3: Event IDs can be shared for deduplication

**Evidence:** SSEEvent deduplication uses `part.id` field to identify related events. When events share the same `part.id`, they update in place rather than creating duplicates. extractEventId() function extracts this stable ID.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/stores/agents.ts:152-157, 386-405`

**Significance:** Events may share IDs, which could be used for grouping. Need to understand if tool and step-finish events share IDs or if they need different correlation logic.

---

### Finding 4: Current rendering treats each event independently

**Evidence:** Activity tab loops through agentEvents and renders each event in its own container div (lines 442-487). Each event with part.type === 'tool' gets displayed with icon and name, each 'step-finish' gets displayed separately with its own icon (✓). The loop is `{#each agentEvents as event}` with each iteration creating a separate visual item.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/components/agent-detail/activity-tab.svelte:442-487`

**Significance:** To group tool+result as single unit, we need to change the rendering logic to: (1) identify related events, (2) group them together, (3) render as single collapsible container. This requires detecting event relationships (likely by sequence or shared ID) and changing the loop structure to handle grouped events rather than individual events.

---

## Synthesis

**Key Insights:**

1. **Event-per-item rendering creates visual fragmentation** - Each SSE event (tool, step-finish) gets its own list item with icon, causing tool execution to appear as 2+ separate items rather than a logical unit (Finding 4).

2. **Tool results already exist but may come from multiple sources** - Tool events can include `part.state.output` directly (Finding 1), but step-finish events may also carry results. Current code only displays output from tool events themselves.

3. **Event correlation will require sequence-based or ID-based grouping** - Since each event is separate in the stream, we need logic to identify which step-finish belongs to which tool call. Event deduplication already uses `part.id` (Finding 3), which may provide correlation mechanism.

**Answer to Investigation Question:**

Tool calls and results should be grouped by changing the rendering loop from event-by-event to grouped rendering. The approach: (1) Pre-process agentEvents to create groups where tool events are paired with their following step-finish events, (2) Render each group as a single collapsible container with tool call as header and result(s) indented below, (3) Maintain expand/collapse state per group (already exists per-event via event ID). The main challenge is correlation logic - determining which events belong together, likely by sequence (step-finish immediately follows tool) or shared parent ID.

---

## Structured Uncertainty

**What's tested:**

- ✅ Current code renders each event independently (verified: read activity-tab.svelte lines 442-487)
- ✅ Tool results can come from part.state.output (verified: existing code at lines 464-485)
- ✅ Event deduplication uses part.id (verified: agents.ts lines 152-157, 386-405)

**What's untested:**

- ⚠️ Whether step-finish events share part.id with their tool events (not observed in actual SSE stream)
- ⚠️ Whether step-finish events always immediately follow their tool event (sequence assumption not validated)
- ⚠️ Whether all tool results come via step-finish or if part.state.output is sufficient (not clear from spawn context which path is problematic)

**What would change this:**

- Finding would be wrong if step-finish events don't contain tool results (they might just be lifecycle markers)
- Approach would change if events share IDs (simpler correlation) vs require sequence-based grouping (more complex)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Grouped Event Rendering with Collapsible Tool Calls** - Pre-process agentEvents array to create groups where tool events contain their related step/result events, then render each group as a single collapsible container.

**Why this approach:**
- Preserves existing expand/collapse infrastructure (Finding 1 - already have expandedResults Map)
- Handles both part.state.output and step-finish event results (addresses uncertainty about result sources)
- Minimal disruption to existing event filtering and deduplication logic (Finding 3)
- Aligns with target UI shown in spawn context (▶ Bash(git status) with indented result)

**Trade-offs accepted:**
- Pre-processing adds small overhead before rendering (acceptable - event arrays are small, max 500 per agent)
- May group unrelated events if correlation logic is imperfect (mitigated by conservative grouping - only group when clear relationship exists)

**Implementation sequence:**
1. Add grouping function that processes agentEvents and returns grouped structure - foundational, enables all UI changes
2. Update rendering loop to iterate over groups instead of individual events - builds on grouping
3. Style grouped items with proper indentation/nesting - visual polish on top of structural changes

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
- Grouping function `groupToolEvents(events: SSEEvent[])` - returns array of groups where each group has primary event + related events
- Start with simple sequence-based correlation: tool event followed by step-finish gets grouped
- Preserve existing event structure for non-tool events (text, reasoning, standalone steps)

**Things to watch out for:**
- ⚠️ Edge case: Tool with no step-finish (should still render, just without step event)
- ⚠️ Edge case: Step-finish with no preceding tool (render independently, don't crash)
- ⚠️ Edge case: Multiple step-finish events for one tool (group all of them)
- ⚠️ Preserve keyed rendering for Svelte reactivity (groups need stable IDs based on primary event ID)
- ⚠️ Existing expand/collapse state keyed by event.id - ensure compatibility with new grouped structure

**Areas needing further investigation:**
- Whether step-finish events actually carry tool results or if they're just lifecycle markers (test with real agent activity)
- Optimal correlation strategy: sequence-based vs ID-based vs hybrid
- Whether to show step-start events within the group or hide them entirely

**Success criteria:**
- ✅ Tool calls and their results appear as single visual unit (not separate list items)
- ✅ Collapsible via click (same UX as current tool result expand/collapse)
- ✅ Indented result below tool name (matches target UI: ▶ Bash(git status) / └ result)
- ✅ No visual regressions for non-tool events (text, reasoning, standalone steps)
- ✅ Visual verification via browser with active agent showing tool calls

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
