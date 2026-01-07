<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created SynthesisTab.svelte component (195 lines) with D.E.K.N. section headers, outcome badges, Create Issue buttons, and close_reason fallback.

**Evidence:** Build passes with `bun run build`, component exports via index.ts, follows pattern established by ActivityTab.svelte.

**Knowledge:** The Synthesis interface only has 5 fields (tldr, outcome, recommendation, delta_summary, next_actions); Evidence/Knowledge sections require backend expansion.

**Next:** close - component ready for integration into agent-detail-panel.svelte.

**Promote to Decision:** recommend-no (tactical component extraction, not architectural)

---

# Investigation: Extract SynthesisTab Component Part Orch

**Question:** How to extract the Synthesis section from agent-detail-panel.svelte into a standalone SynthesisTab.svelte component with D.E.K.N. sections?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current Synthesis Interface is Limited

**Evidence:** The `Synthesis` interface in agents.ts only has 5 fields:
```typescript
export interface Synthesis {
  tldr?: string;
  outcome?: string; // success, partial, blocked, failed
  recommendation?: string; // close, continue, escalate
  delta_summary?: string; // e.g., "3 files created, 2 modified, 5 commits"
  next_actions?: string[]; // Follow-up items
}
```

This maps to TLDR, Delta (partial via delta_summary), and Next (via recommendation + next_actions). Evidence and Knowledge sections from the full D.E.K.N. template are NOT currently parsed by the backend.

**Source:** `web/src/lib/stores/agents.ts:8-14`

**Significance:** The SynthesisTab can only display the fields available in the interface. Evidence and Knowledge D.E.K.N. sections need backend work to populate - added placeholder comments for future expansion.

---

### Finding 2: Pattern Established by ActivityTab.svelte

**Evidence:** The existing ActivityTab.svelte demonstrates the pattern for tab components:
- Props interface with agent: Agent
- Uses `$props()` for Svelte 5 runes
- Imports from `$lib/stores/agents` and `$lib/components/ui/badge`
- Self-contained state management (filter toggles, auto-scroll)

**Source:** `web/src/lib/components/agent-detail/activity-tab.svelte:1-134`

**Significance:** Followed same pattern for SynthesisTab - consistent API and state management approach across tabs.

---

### Finding 3: Issue Creation Logic Already Exists in Parent

**Evidence:** The agent-detail-panel.svelte already has issue creation logic (lines 102-134):
- `handleCreateIssue(action: string)` function
- State for `creatingIssue`, `issueCreationError`, `createdIssueId`
- Uses `createIssue()` from agents store

**Source:** `web/src/lib/components/agent-detail/agent-detail-panel.svelte:102-134`

**Significance:** Replicated this logic in SynthesisTab using Svelte 5 `$state()` runes for proper reactivity. The function is identical but component-local.

---

## Synthesis

**Key Insights:**

1. **D.E.K.N. Partial Implementation** - Only Delta and Next sections can be fully displayed with current backend data. Evidence and Knowledge sections are placeholders for future expansion.

2. **Close Reason Fallback Works** - When no synthesis is available, the component falls back to displaying `agent.close_reason` as a "Completion Summary" - this provides value even for older agents or light-tier spawns.

3. **Visual Consistency with SynthesisCard** - The existing synthesis-card.svelte component provides a condensed view. SynthesisTab is the expanded version with full D.E.K.N. headers.

**Answer to Investigation Question:**

The SynthesisTab component extracts the Synthesis section with D.E.K.N. organization. It displays:
- Header with outcome badge (success/partial/blocked/failed)
- TLDR or close_reason fallback
- Delta section (What Changed) from delta_summary
- Next section (What Should Happen) with recommendation and Create Issue buttons
- Placeholder comments for Evidence/Knowledge when backend supports them

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes with `bun run build` (verified)
- ✅ Component exports from index.ts (verified in file)
- ✅ Follows Svelte 5 runes pattern (`$state()`, `$props()`)

**What's untested:**

- ⚠️ Visual appearance in browser (not verified via Playwright)
- ⚠️ Issue creation flow works end-to-end (not tested with live API)
- ⚠️ Integration with agent-detail-panel.svelte (separate task)

**What would change this:**

- If Synthesis interface is expanded with evidence/knowledge fields, component should be updated to display them
- If outcome types change from success/partial/blocked/failed, badge styling needs update

---

## Implementation Recommendations

### Recommended Approach ⭐

**Component ready for integration** - SynthesisTab.svelte is complete and exported. Next step is integration task (orch-go-akhff.11).

**Why this approach:**
- Component follows established pattern from ActivityTab
- Uses Svelte 5 runes for proper reactivity
- Self-contained with local state management

**Trade-offs accepted:**
- Evidence/Knowledge sections are placeholders (backend limitation)
- Issue creation logic duplicated from parent (acceptable for component isolation)

**Implementation sequence:**
1. ✅ Create SynthesisTab.svelte with D.E.K.N. structure
2. ✅ Add outcome badges and Create Issue buttons
3. ✅ Export from index.ts
4. → Next: Integration into agent-detail-panel.svelte (separate task)

---

## References

**Files Examined:**
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Original synthesis section and issue creation logic
- `web/src/lib/stores/agents.ts` - Synthesis interface definition
- `web/src/lib/components/agent-detail/activity-tab.svelte` - Pattern reference
- `.orch/templates/SYNTHESIS.md` - Full D.E.K.N. structure

**Commands Run:**
```bash
# Build verification
cd /Users/dylanconlin/Documents/personal/orch-go/web && bun run build
# Result: ✔ done
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-orch-go-hmj61-dashboard-agent.md` - Design specification
- **Epic:** `orch-go-akhff` - Parent epic for tabbed interface

---

## Investigation History

**2026-01-06:** Investigation started
- Initial question: Extract SynthesisTab component with D.E.K.N. sections
- Context: Part of orch-go-akhff epic for dashboard agent detail pane

**2026-01-06:** Implementation complete
- Created SynthesisTab.svelte (195 lines)
- Added to index.ts exports
- Build verified
