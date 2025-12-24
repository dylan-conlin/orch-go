<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Slide-out panel is the optimal UX pattern for agent card click interaction, providing detail without losing list context.

**Evidence:** Analyzed existing card design, API data availability, and UX requirements - cards are small (compact grid), data is available via existing APIs (messages, synthesis, workspace path), modal/expand patterns don't fit the dashboard monitoring context.

**Knowledge:** Agent detail view should differ by state: active agents need live streaming + control actions, completed agents need synthesis review + artifact navigation, all need copy-able identifiers for CLI workflows.

**Next:** Implement slide-out panel component with state-aware content and SSE-based live output streaming for active agents.

**Confidence:** High (85%) - Design decisions are well-grounded but implementation details need validation via prototype.

---

# Investigation: Design Agent Card Click Interaction

**Question:** What should clicking an agent card reveal, which UI pattern fits best, and how should behavior differ by agent state?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete
**Confidence:** High (85%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Agent Card Currently Has No Click Interaction

**Evidence:** The `agent-card.svelte` component (web/src/lib/components/agent-card/agent-card.svelte) is a display-only component with no click handlers. Cards show:
- Status badge + phase badge + runtime
- Display title (TLDR for completed, task for active)
- Workspace ID as subtitle
- Project + skill + beads_id badges
- Current activity (active agents) or synthesis preview (completed agents)

**Source:** `agent-card.svelte:163-252` - Component is a styled `<div>` with hover effects but no interaction.

**Significance:** Starting from scratch for click interaction - need to add click handler, selected state, and detail panel.

---

### Finding 2: API Has Rich Data Available for Detail View

**Evidence:** The `/api/agents` endpoint (serve.go:110-356) already provides:
- `id`, `session_id`, `beads_id`, `beads_title` - identifiers
- `skill`, `project`, `phase`, `task` - context
- `status`, `is_processing`, `runtime`, `spawned_at`, `updated_at` - state
- `synthesis` object with `tldr`, `outcome`, `recommendation`, `delta_summary`, `next_actions`
- `window` target for tmux-based agents

Additional data available via OpenCode client:
- `GetMessages(sessionID)` - full message history for live output
- `IsSessionProcessing(sessionID)` - activity detection
- Real-time updates via SSE `/api/events` stream

**Source:** `serve.go:110-141` (AgentAPIResponse struct), `client.go:429-446` (GetMessages)

**Significance:** No new API endpoints needed - can build rich detail view with existing data. Live streaming needs SSE connection per-agent.

---

### Finding 3: Workspace Path is Derivable, Not Explicitly Stored

**Evidence:** Workspace path follows pattern: `{PROJECT_DIR}/.orch/workspace/{agent.id}/`
The agent `id` IS the workspace name (e.g., `og-inv-design-agent-card-24dec`).
This is established in spawn.go and context.go where workspace is created.

**Source:** `spawn/context.go:272-273` - `workspacePath := cfg.WorkspacePath()` uses `cfg.WorkspaceName`

**Significance:** Can construct workspace path for file navigation. Should expose in API response for clarity (currently implicit).

---

### Finding 4: Dashboard Uses Progressive Disclosure Pattern

**Evidence:** Current dashboard (`+page.svelte`) uses collapsible sections (Active, Recent, Archive) with grid layout (2-5 columns responsive). Cards are compact by design - meant to show many agents at once for swarm monitoring.

**Source:** `+page.svelte:481-566` - CollapsibleSection components with responsive grid

**Significance:** Detail view should NOT replace card visibility - user needs to see swarm overview while examining one agent. This rules out full-page modals and suggests slide-out panel or expand-in-place.

---

### Finding 5: State-Specific Actions are Different

**Evidence:** From reviewing orch CLI commands and serve.go:
- **Active agents:** `orch send` (Q&A), `orch abandon` (give up), open workspace
- **Completed agents:** `orch complete` (verify & close), view artifacts, copy for review
- **Abandoned agents:** Read-only review, respawn capability
- **All agents:** Copy beads ID, copy session ID, open workspace in editor

**Source:** CLAUDE.md command reference, `main.go` command implementations

**Significance:** Detail panel needs state-aware actions section - different buttons/links depending on agent status.

---

## Synthesis

**Key Insights:**

1. **Slide-out panel is optimal** - Modal blocks swarm view, inline expand disrupts grid layout, slide-out preserves context while showing detail. This matches Vercel's deployment dashboard pattern.

2. **Live output streaming is feasible** - SSE infrastructure exists (`/api/events`). For active agents, subscribe to `message.part` events filtered by session ID to show real-time tool calls and text output.

3. **State-aware content reduces noise** - Don't show "abandon" button for completed agents. Don't show "live output" section for inactive agents. Tailor the detail view to what's relevant.

4. **Copy actions are critical for CLI workflow** - Users will want to quickly copy beads ID or session ID to run CLI commands. Make identifiers easily copyable.

**Answer to Investigation Question:**

**Q1: What should clicking an agent card reveal?**
- **Header:** Status, phase, runtime, title (always visible)
- **Identifiers section:** Beads ID (copyable), Session ID (copyable), Workspace path (link to open)
- **Context section:** Task description, skill, project
- **Live Output section (active only):** Streaming tool calls and text from SSE
- **Synthesis section (completed only):** Full TLDR, outcome, recommendation, delta summary, next actions
- **Actions section:** State-appropriate buttons (send message, abandon, complete, view artifacts)

**Q2: What UI pattern fits best?**
Slide-out panel from the right (40-50% width). Reasons:
- Preserves swarm grid view on left
- Standard pattern for detail-on-selection
- Can close by clicking outside or X button
- Mobile: full-width overlay

**Q3: What quick actions should be available?**
| State | Actions |
|-------|---------|
| Active | Send message, Abandon, Open workspace |
| Completed | Complete, View synthesis, View investigation, Open workspace |
| Abandoned | View failure report, Respawn (redirect to orch spawn prompt) |

**Q4: Should behavior differ by agent state?**
Yes. See actions table above, plus:
- Active: Show live output section, hide synthesis
- Completed: Show full synthesis, hide live output
- Abandoned: Show failure report section if exists

**Q5: How to handle live output streaming for active agents?**
1. When detail panel opens for active agent, connect to existing SSE stream
2. Filter `message.part` events by `sessionID` matching agent's `session_id`
3. Display tool-invocation events as "🔧 Using {tool}"
4. Display text events as scrolling output
5. Cap at last ~50 lines for performance
6. Show "⏳ Waiting for response..." when idle

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
Design decisions are grounded in analysis of existing code, UX best practices, and available infrastructure. The main uncertainty is around implementation details that need validation via prototype.

**What's certain:**

- ✅ Slide-out panel is the correct pattern for this dashboard context
- ✅ API data is sufficient for detail view (no new endpoints needed for MVP)
- ✅ SSE infrastructure can support live streaming with filtering

**What's uncertain:**

- ⚠️ Exact layout and information hierarchy within the panel
- ⚠️ Performance of SSE filtering when many agents are active
- ⚠️ Mobile UX for slide-out (may need full-screen mode)

**What would increase confidence to Very High (95%+):**

- Build prototype and test with real agent sessions
- Get user feedback on information hierarchy
- Test mobile responsiveness

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Slide-out Panel with SSE Live Streaming** - Implement a right-side slide-out panel that shows state-aware detail content with real-time output streaming for active agents.

**Why this approach:**
- Preserves swarm grid context (Finding 4)
- Uses existing API data (Finding 2)
- Standard UX pattern for detail-on-selection
- Progressive disclosure: show more when needed

**Trade-offs accepted:**
- More complex than simple modal
- SSE filtering adds client-side logic
- Accepting: complexity is justified by better UX

**Implementation sequence:**
1. Add click handler to AgentCard + selected state styling
2. Create AgentDetail slide-out panel component
3. Wire up state management (selected agent ID in store)
4. Implement static detail view (no streaming yet)
5. Add SSE-based live output for active agents
6. Add action buttons with CLI command generators

### Alternative Approaches Considered

**Option B: Inline Expand**
- **Pros:** No overlay, simpler implementation
- **Cons:** Disrupts grid layout, pushes other cards around, hard to show much content
- **When to use instead:** If users strongly prefer not having overlays

**Option C: Full Modal**
- **Pros:** Maximum space for content, familiar pattern
- **Cons:** Blocks swarm view (Finding 4 disqualifies this), modal fatigue
- **When to use instead:** Never for this dashboard context

**Option D: Detail Page (routing)**
- **Pros:** URL for each agent, full page for detail
- **Cons:** Context switch, loses swarm view, overkill for monitoring
- **When to use instead:** If building a dedicated agent deep-dive feature

**Rationale for recommendation:** Slide-out is standard for "select item to see details" UX (Gmail, Vercel, GitHub). It maintains the primary content (swarm grid) while allowing exploration.

---

### Implementation Details

**What to implement first:**
1. `selectedAgentId` store in `agents.ts`
2. Click handler on AgentCard that sets `selectedAgentId`
3. AgentDetailPanel.svelte component (slide-out)
4. Integration in +page.svelte

**Things to watch out for:**
- ⚠️ SSE event filtering must be efficient - don't process all events, filter by sessionID early
- ⚠️ Panel close on Escape key and click-outside
- ⚠️ Mobile view needs full-width panel with proper touch dismissal
- ⚠️ Avoid re-rendering full grid when selection changes (use keyed each block)

**Areas needing further investigation:**
- Exact behavior of "Open workspace" action (which editor? how to detect?)
- Whether to add `/api/agents/{id}` endpoint for single-agent fetch
- Integration with `orch send` command (inline in panel vs terminal?)

**Success criteria:**
- ✅ User can click agent card to see detail panel
- ✅ Panel shows all relevant information for agent state
- ✅ Active agents show real-time tool calls
- ✅ Users can copy identifiers with one click
- ✅ Actions execute relevant CLI commands

---

## References

**Files Examined:**
- `web/src/lib/components/agent-card/agent-card.svelte` - Current card implementation
- `web/src/routes/+page.svelte` - Dashboard layout and structure
- `web/src/lib/stores/agents.ts` - Agent store and SSE handling
- `cmd/orch/serve.go` - API endpoint definitions
- `pkg/opencode/client.go` - OpenCode client and message fetching
- `pkg/spawn/context.go` - Workspace path derivation
- `pkg/verify/review.go` - AgentReview struct for synthesis data

**Commands Run:**
```bash
# Searched for existing click handlers
rg "onclick" web/src/

# Checked SSE event handling
rg "message.part" web/src/
```

**External Documentation:**
- shadcn-svelte Sheet component - standard slide-out pattern

**Related Artifacts:**
- **Design:** `.orch/docs/designs/2025-12-20-swarm-dashboard-ui-iterations.md` - Previous UI iteration

---

## Proposed Component Structure

```
web/src/lib/components/
├── agent-card/
│   ├── agent-card.svelte (add onclick)
│   └── index.ts
├── agent-detail/              # NEW
│   ├── agent-detail-panel.svelte  # Main slide-out panel
│   ├── agent-detail-header.svelte # Status, phase, runtime
│   ├── agent-detail-ids.svelte    # Copyable identifiers
│   ├── agent-detail-context.svelte # Task, skill, project
│   ├── agent-detail-live.svelte   # SSE streaming output (active)
│   ├── agent-detail-synthesis.svelte # Full synthesis (completed)
│   ├── agent-detail-actions.svelte   # State-aware action buttons
│   └── index.ts
```

## Proposed Data Flow

```
User clicks AgentCard
    ↓
Set selectedAgentId in store
    ↓
AgentDetailPanel renders (conditional on selectedAgentId)
    ↓
Panel reads agent data from agents store
    ↓
If active: Subscribe to SSE events filtered by session_id
    ↓
Display state-appropriate content
    ↓
User clicks action → Execute (copy to clipboard, open URL, or show command)
    ↓
User closes panel → Clear selectedAgentId
```

---

## Investigation History

**2025-12-24 07:30:** Investigation started
- Initial question: Design click interaction for agent cards in swarm dashboard
- Context: Dashboard exists but cards have no click interaction

**2025-12-24 08:00:** Completed codebase analysis
- Found: Cards are display-only, API has rich data, SSE exists
- Key insight: Slide-out preserves swarm context

**2025-12-24 08:30:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Slide-out panel with state-aware content and SSE live streaming is the recommended approach

---

## Self-Review

- [x] Real test performed (analyzed actual code and API responses)
- [x] Conclusion from evidence (recommendations based on findings)
- [x] Question answered (all 5 sub-questions addressed)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED
