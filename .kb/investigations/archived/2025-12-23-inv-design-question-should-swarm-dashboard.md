<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard should use progressive disclosure (Active/Recent/Archive sections) rather than flat list or active-only filter.

**Evidence:** Dashboard shows 26 agents (2 active, 24 idle) because OpenCode persists all sessions indefinitely; "Active Only" toggle exists but users need both focus and history; `orch clean` doesn't delete sessions, creating semantic gap.

**Knowledge:** The tension is presentation (cluttered UI) not persistence (valuable for debugging); time-based filtering (6h threshold) is insufficient; users need operational focus AND historical debugging AND health monitoring - no single view satisfies all three.

**Next:** Implement collapsible sections with Active expanded by default; optionally add `orch clean --sessions` for permanent deletion (complementary, not alternative).

**Confidence:** High (85%) - 24h threshold for Recent untested with users, Archive visibility default unclear.

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

# Investigation: Design Question Should Swarm Dashboard

**Question:** Should the swarm dashboard show idle/completed OpenCode sessions, or only active ones?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** architect
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Dashboard shows all historical sessions with time-based filtering

**Evidence:** The `/api/agents` endpoint returns sessions from OpenCode's session list with two filtering rules:
- Sessions updated within last 10 minutes marked as "active"
- Sessions older than 6 hours filtered out (unless active)
- Currently shows 26 agents: 2 active, 24 idle (old completed sessions)

**Source:** 
- `cmd/orch/serve.go:156-176` - filtering logic in `handleAgents()`
- `web/src/lib/stores/agents.ts:54-316` - frontend agent store and SSE handling
- `web/src/routes/+page.svelte:31-43` - UI filtering with "Active Only" toggle

**Significance:** The dashboard becomes cluttered with historical sessions. The 6-hour threshold helps but doesn't solve the core issue - OpenCode persists all sessions indefinitely, so the list keeps growing until manual intervention.

---

### Finding 2: orch clean doesn't delete OpenCode sessions

**Evidence:** The `orch clean` command has three modes:
- Default: Reports cleanable workspaces (doesn't actually delete them per line 2533-2534)
- `--windows`: Closes tmux windows for completed agents
- `--verify-opencode`: Cleans orphaned disk sessions, NOT API sessions

**Source:** 
- `cmd/orch/main.go:2319-2580` - clean command implementation
- `cmd/orch/main.go:2533-2534` - "Note: We don't delete the workspace directory itself"

**Significance:** There's a design gap. Users expect `orch clean` to clean up "completed agents" but it doesn't touch the OpenCode session list that powers the dashboard. This creates the 26-agent clutter problem.

---

### Finding 3: Dashboard already has "Active Only" toggle for focused view

**Evidence:** The UI includes an "Active Only" checkbox that filters to `status === 'active'` sessions:
```typescript
if (activeOnly) {
    result = result.filter(a => a.status === 'active');
}
```

**Source:** 
- `web/src/routes/+page.svelte:31-43` - filtering logic
- `web/src/routes/+page.svelte:236-243` - Active Only checkbox UI

**Significance:** Users already have a quick way to focus on active work without permanently hiding historical sessions. This suggests the need is for both views: focused (active only) and historical (all sessions).

---

### Finding 4: OpenCode session persistence is by design, not a bug

**Evidence:** OpenCode maintains a persistent session list across all operations. The `/sessions` endpoint returns all historical sessions. The API doesn't provide session archiving or bulk deletion.

**Source:**
- `pkg/opencode/client.go` - OpenCode client implementation
- OpenCode API behavior (observed via serve.go proxy)

**Significance:** Session persistence is valuable for debugging and historical reference. The question isn't whether to preserve sessions, but how to present them in the UI without clutter.

---

## Synthesis

**Key Insights:**

1. **The problem is presentation, not persistence** - OpenCode's session persistence is valuable for debugging. The 26-agent clutter comes from mixing active work with historical sessions in a flat list. The "Active Only" toggle (Finding 3) shows users want both views.

2. **"Clean" has a semantic gap** - Users expect `orch clean` to clean up the dashboard clutter, but it only manages workspace directories and tmux windows (Finding 2). The command name promises more than it delivers.

3. **Time-based filtering is insufficient** - The current 6-hour threshold (Finding 1) helps but creates an artificial boundary. A completed agent from 8 hours ago is still valuable for reference, but shouldn't dominate the view.

4. **Operational context drives visibility needs** - Active development needs focus (2 active agents visible). Debugging needs history (recent completions accessible). Review needs overview (all agents for health monitoring).

**Answer to Investigation Question:**

**Show all sessions but with smart grouping and progressive disclosure.**

The dashboard should present three collapsed/expandable sections:
1. **Active** (always expanded) - Shows agents with activity in last 10 minutes
2. **Recent** (collapsed by default) - Idle/completed agents from last 24 hours  
3. **Archive** (collapsed by default) - Older sessions beyond 24 hours

Additionally, `orch clean` should gain the ability to permanently delete OpenCode sessions (with confirmation), aligning the command's behavior with user expectations.

This approach:
- Preserves operational visibility (active work always visible)
- Enables historical debugging (expand Recent/Archive as needed)
- Reduces UI clutter (collapsed sections hide noise)
- Provides cleanup path (`orch clean --sessions` for permanent removal)

The existing "Active Only" toggle remains useful as a keyboard-accessible quick filter.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong evidence from codebase analysis shows the current architecture and pain points. The recommendation is based on established UX patterns (progressive disclosure) rather than novel invention. Uncertainty exists around specific thresholds and whether session deletion is needed.

**What's certain:**

- ✅ Current implementation shows all sessions in a flat list with time-based filtering (6h threshold)
- ✅ Dashboard already has "Active Only" toggle showing users want focused view
- ✅ `orch clean` doesn't delete OpenCode sessions, creating semantic gap
- ✅ Progressive disclosure pattern solves "focus vs history" tension

**What's uncertain:**

- ⚠️ Whether 24-hour threshold for "Recent" matches user mental model (could be 12h or 48h)
- ⚠️ Whether Archive section should be visible by default or require opt-in
- ⚠️ Whether session deletion via `orch clean` is actually needed or if better UI is sufficient
- ⚠️ How users will respond to three sections vs two (Active vs Rest)

**What would increase confidence to Very High (95%+):**

- User testing with Dylan to validate 24h threshold feels right
- Prototype the UI to verify collapse/expand UX is smooth
- Check if very large Archive counts (100+) cause performance issues

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

**Progressive Disclosure with Smart Grouping** - Group sessions into Active/Recent/Archive sections with collapse/expand controls

**Why this approach:**
- Preserves both operational focus (active work) and debugging capability (historical sessions)
- Aligns with established UX pattern (progressive disclosure reduces cognitive load)
- Requires no backend changes - pure UI enhancement
- Leverages existing filtering logic (active threshold, time-based filtering)

**Trade-offs accepted:**
- More UI complexity (3 sections vs 1 flat list with toggle)
- Requires localStorage to persist collapse state across refreshes
- Doesn't reduce the underlying session count (that's addressed separately via clean command)

**Implementation sequence:**
1. **UI grouping logic** - Modify `+page.svelte` to group filteredAgents into three arrays (active/recent/archive) based on status and time thresholds
2. **Collapsible sections** - Add collapse/expand controls to each section header, with Active expanded by default
3. **Persistence** - Store collapse state in localStorage to remember user preferences across page reloads
4. **Stats bar update** - Show counts for each section (e.g., "2 active, 8 recent, 16 archived")

### Alternative Approaches Considered

**Option A: Filter to active only (hide idle/completed by default)**
- **Pros:** Simplest implementation, cleanest initial view
- **Cons:** Loses debugging capability - no way to see recent completions without toggling. "Active Only" checkbox already provides this behavior (Finding 3).
- **When to use instead:** If users rarely need to reference completed agents (not true per Finding 4)

**Option B: Add session archiving to orch clean (delete sessions permanently)**
- **Pros:** Solves clutter at the source, aligns with "clean" semantics
- **Cons:** Destructive operation, requires OpenCode API support for session deletion (may not exist), loses historical debugging value
- **When to use instead:** As a **complementary** feature, not alternative. Should be `orch clean --sessions` with confirmation prompt.

**Option D: Time-based filtering only (extend 6h to 7 days)**
- **Pros:** Already partially implemented, simple logic
- **Cons:** Arbitrary cutoff loses valuable history. 7-day-old completion may still be relevant for review (Finding 4).
- **When to use instead:** As part of the grouping logic (Archive = >24h) but not as the only solution

**Rationale for recommendation:** Progressive disclosure (recommended approach) is the only option that satisfies all four user needs identified in synthesis: operational visibility, historical debugging, UI clarity, and health monitoring. Options A/D sacrifice debugging capability. Option B is destructive and should complement, not replace, better UI presentation.

---

### Implementation Details

**What to implement first:**
- **Grouping logic in `+page.svelte`** - Create derived stores for `activeAgents`, `recentAgents`, `archivedAgents` based on status and timestamps
  - Active: `status === 'active'` (existing logic)
  - Recent: `status === 'idle' || status === 'completed'` AND updated within 24h
  - Archive: Everything else older than 24h
- **Section UI components** - Create `<CollapsibleSection>` component with header, count badge, and expand/collapse control
- **localStorage persistence** - Save/restore collapse state on mount/update

**Things to watch out for:**
- ⚠️ **Empty section states** - Handle case where Archive is empty (don't show section) or Active is empty (show helpful message)
- ⚠️ **Transition between sections** - When an active agent becomes idle, it should smoothly move to Recent section (reactivity via derived stores handles this)
- ⚠️ **Filter interaction** - "Active Only" toggle should still work, but may need to show/hide entire sections rather than filter within them
- ⚠️ **Mobile responsiveness** - Collapsible sections need touch-friendly expand/collapse targets

**Areas needing further investigation:**
- Whether to show archived sessions at all or require explicit opt-in (e.g., "Show archived" button)
- Whether 24-hour threshold for Recent is the right balance (could make configurable via settings)
- Whether to add session deletion to `orch clean` in same PR or separate feature
- How to handle very large Archive counts (100+ sessions) - pagination or virtual scrolling?

**Success criteria:**
- ✅ Dashboard loads showing only Active section expanded (2 agents visible, not 26)
- ✅ User can expand Recent to see last 24h of completions (debugging use case)
- ✅ User can expand Archive to see older sessions (health monitoring use case)
- ✅ Collapse state persists across page refreshes
- ✅ Stats bar accurately reflects counts for all three sections
- ✅ Existing "Active Only" toggle still works and provides keyboard-accessible focus mode

---

## References

**Files Examined:**
- `cmd/orch/serve.go:156-176` - Filtering logic in handleAgents endpoint
- `cmd/orch/main.go:2319-2580` - Clean command implementation
- `web/src/routes/+page.svelte:31-43` - Dashboard filtering UI
- `web/src/lib/stores/agents.ts:54-316` - Agent store and SSE handling
- `pkg/opencode/client.go` - OpenCode client integration

**Commands Run:**
```bash
# Search for clean command definition
grep -r "cleanCmd" cmd/orch/*.go

# Find cleanCmd line number
grep -n "var cleanCmd" cmd/orch/main.go
```

**External Documentation:**
- Progressive Disclosure (UX pattern) - Show details on demand, reduce initial cognitive load

**Related Artifacts:**
- **Features:** `.orch/features.json` - Related to API infrastructure and server management
- **Investigation:** `.kb/investigations/2025-12-23-inv-design-question-should-orch-servers.md` - API vs project server distinction

---

## Investigation History

**2025-12-23 (start):** Investigation started
- Initial question: Should the swarm dashboard show idle/completed OpenCode sessions, or only active ones?
- Context: Dashboard at :5188 shows 26 agents (2 active, 24 idle). `orch clean` removes workspaces but not OpenCode sessions.

**2025-12-23 (analysis):** Key findings documented
- Discovered "Active Only" toggle already exists (users want both views)
- Identified semantic gap in `orch clean` command (doesn't clean dashboard clutter)
- Analyzed 4 options: filter-only, session deletion, UI grouping, time-based

**2025-12-23 (synthesis):** Recommendation finalized
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend progressive disclosure with Active/Recent/Archive grouping, plus optional session deletion in clean command
