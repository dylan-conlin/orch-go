## Summary (D.E.K.N.)

**Delta:** Moved Live Activity section above Quick Copy and Context sections in dashboard detail panel.

**Evidence:** Section reorder in agent-detail-panel.svelte - Live Activity now appears immediately after Status Bar for active agents.

**Knowledge:** Live Activity is the most important information when viewing an active agent; placing it higher reduces scrolling.

**Next:** Close - implementation complete.

**Confidence:** High (95%) - Simple UI reorder, verified with svelte-check.

---

# Investigation: Dashboard Live Activity Should Above

**Question:** Should Live Activity section appear above Quick Copy/Context in the agent detail panel?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Original Layout Order

**Evidence:** The detail panel sections were ordered: Status Bar → Quick Copy → Context → Live Activity → Synthesis

**Source:** `web/src/lib/components/agent-detail/agent-detail-panel.svelte:180-310`

**Significance:** Live Activity (the most operationally important section for active agents) was buried below less important sections.

---

### Finding 2: Live Activity Only Shows for Active Agents

**Evidence:** The Live Activity section is conditionally rendered with `{#if $selectedAgent.status === 'active'}`.

**Source:** `agent-detail-panel.svelte:267`

**Significance:** Moving it higher only affects active agents - completed agents still show Quick Copy/Context first (which is appropriate since there's no live activity to show).

---

## Synthesis

**Key Insights:**

1. **Operational Priority** - When viewing an active agent, the user wants to see what it's doing NOW, not copy commands.

2. **Conditional Visibility** - The reorder is smart because inactive agents don't have this section at all.

**Answer to Investigation Question:**

Yes, Live Activity should be above Quick Copy/Context. This prioritizes real-time information over reference data.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Simple UI reorder with no logic changes. svelte-check passes with 0 errors.

**What's certain:**

- Live Activity section now renders first (after Status Bar) for active agents
- No functional changes, just visual reordering
- Code passes type checking

**What's uncertain:**

- Visual verification requires running the dashboard (light tier spawn)

---

## References

**Files Examined:**
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Full detail panel layout

**Commands Run:**
```bash
bun run check  # Passed with 0 errors
```
