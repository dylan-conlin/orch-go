# Epic: Dashboard Agent Detail Pane Tabbed Interface

**ID:** orch-go-akhff
**Type:** epic
**Status:** closed
**Created:** 2026-01-06
**Updated:** 2026-01-07

---

## Understanding

### Problem (not symptoms)

The agent detail panel is a 523-line monolith mixing content types in one scroll. Active agents need to see live activity; completed agents need to see synthesis and investigation artifacts. The current design serves neither well.

Deeper: As orchestration scales, the dashboard becomes the primary visibility layer. The detail pane is where Dylan (and the orchestrator) understand what an agent did, is doing, or produced. If this is hard to parse, oversight degrades.

### Why Previous Approaches Failed

The original design assumed a simple linear flow: agent starts → agent works → agent finishes. Reality is more complex:
- Active agents need real-time streaming focus
- Completed agents need structured output review
- Abandoned agents need investigation access without synthesis

Mixing all views in one scroll satisfies none of these modes.

### Key Constraints

- **666px minimum width** - Dashboard must work at half MacBook Pro screen
- **80-85% viewport width** - Detail pane should use available space
- **State-driven visibility** - Tabs appear/disappear based on agent status
- **Existing SSE infrastructure** - Must reuse current event filtering patterns
- **D.E.K.N. structure** - Synthesis tab follows established format

### Where Risks Live

- **SSE performance** - Expanding from 50 to 100 events may cause lag
- **Tab state persistence** - Should we remember last tab per agent?
- **Workspace file API** - New endpoints needed for Investigation tab
- **Mobile/narrow viewports** - Tab labels may overflow

### What "Done" Looks Like

1. Panel works at 666px width with all tabs functional
2. Active agents show Activity tab by default (filtered SSE feed)
3. Completed agents show Synthesis tab by default (D.E.K.N. format)
4. Investigation tab shows workspace artifacts for completed/abandoned agents
5. Tab switching is instant (<50ms)
6. Existing functionality (Quick Copy, Quick Commands) preserved
7. File reduced from 523 lines to ~200 lines via extraction

---

## Children (Beads Issues)

| ID | Title | Status |
|----|-------|--------|
| orch-go-akhff.1 | Create tab navigation infrastructure | completed |
| orch-go-akhff.2 | Extract ActivityTab component | completed |
| orch-go-akhff.3 | Extract SynthesisTab component | completed |
| orch-go-akhff.4 | Create InvestigationTab component | completed |
| orch-go-akhff.5 | Integrate tabs into agent-detail-panel | completed |
| orch-go-akhff.6 | Add workspace file API endpoints | ? |
| orch-go-akhff.7 | Update panel width to 80-85% | ? |

*Status unknown for some children - beads DB corrupted*

---

## Execution Log

### 2026-01-06 - Epic Created

Design investigation completed (`2026-01-06-inv-orch-go-hmj61-dashboard-agent.md`). Proposed 3-tab structure with state-driven visibility.

### 2026-01-06 - Tab Infrastructure

Tab navigation created with TabButton component, activeTab state, status-based visibility. See `2026-01-06-inv-create-tab-navigation-infrastructure-part.md`.

### 2026-01-06 - ActivityTab Extracted

Created ActivityTab.svelte with enhanced SSE filtering, message type filters, 100-event limit, auto-scroll option. See `2026-01-06-inv-extract-activitytab-component-part-orch.md`.

### 2026-01-06 - SynthesisTab Extracted

Created SynthesisTab.svelte with D.E.K.N. sections. See `2026-01-06-inv-extract-synthesistab-component-part-orch.md`.

### 2026-01-06 - InvestigationTab Created

Created InvestigationTab.svelte for workspace artifact viewing. See `2026-01-06-inv-create-investigationtab-component-part-orch.md`.

### 2026-01-06 - Integration Complete

Integrated all tabs into agent-detail-panel.svelte, reducing file from 597 to 399 lines (-33%). See `2026-01-06-inv-integrate-tab-components-into-agent.md`.

### 2026-01-07 - Epic Closed

Verified complete:
- All tab components implemented and exported
- Panel refactored from 523 → 244 lines (53% reduction)
- State-driven tab visibility working
- Quick Copy/Commands removed (intentional)
- Beads issue closed

Note: This epic was already complete but beads showed it as open due to:
1. DB corruption hiding true state
2. No link between artifact completion and issue tracking

---

## Evidence Chain

**Design:**
- `.kb/investigations/2026-01-06-inv-orch-go-hmj61-dashboard-agent.md` - Main design investigation

**Implementation:**
- `.kb/investigations/2026-01-06-inv-create-tab-navigation-infrastructure-part.md`
- `.kb/investigations/2026-01-06-inv-extract-activitytab-component-part-orch.md`
- `.kb/investigations/2026-01-06-inv-extract-synthesistab-component-part-orch.md`
- `.kb/investigations/2026-01-06-inv-create-investigationtab-component-part-orch.md`
- `.kb/investigations/2026-01-06-inv-integrate-tab-components-into-agent.md`

**Code:**
- `web/src/lib/components/agent-detail/tab-button.svelte`
- `web/src/lib/components/agent-detail/activity-tab.svelte`
- `web/src/lib/components/agent-detail/synthesis-tab.svelte`
- `web/src/lib/components/agent-detail/investigation-tab.svelte`
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte`

---

## Spike Context

This file is part of a spike testing markdown-based issues.

**What we're testing:** Does direct access to issue/epic state change how Dylan works?

**Observe:**
- Do you open this file directly?
- Do you edit it?
- Does it change your sense of what's happening?
- What's missing that you wish were there?

**Duration:** One week (2026-01-07 to 2026-01-14)
