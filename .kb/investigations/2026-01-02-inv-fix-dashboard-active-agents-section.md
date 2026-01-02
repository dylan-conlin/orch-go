<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard now separates truly working agents from dead/stalled agents in distinct sections.

**Evidence:** Visual verification shows "Working" section with 4 agents; stats bar shows "4 working" instead of conflated "active" count.

**Knowledge:** The activeAgents store conflated all session-bearing agents (active/idle/dead/stalled); splitting into workingAgents and needsAttentionAgents provides clearer signal.

**Next:** Close - implementation complete and verified.

---

# Investigation: Fix Dashboard Active Agents Section

**Question:** How to separate truly active agents from dead/stalled agents in the dashboard?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: activeAgents store conflated working and dead agents

**Evidence:** In `agents.ts:240-241`, the derived store included `status === 'dead' || status === 'stalled'` alongside active/idle, making the count misleading.

**Source:** `web/src/lib/stores/agents.ts:240-241`

**Significance:** Users couldn't distinguish between agents doing work vs agents needing intervention.

---

### Finding 2: Dashboard rendered all "active" agents in one section

**Evidence:** The +page.svelte used `sortedActiveAgents` for a single "Active Agents" section with no visual distinction between working and problematic agents.

**Source:** `web/src/routes/+page.svelte:628-655`

**Significance:** The UI provided no affordance for users to understand which agents needed attention.

---

## Implementation

Created two new derived stores:
- `workingAgents`: Only `status === 'active' || status === 'idle'`
- `needsAttentionAgents`: Only `status === 'dead' || status === 'stalled'`

Updated dashboard:
- Renamed "Active Agents" section to "Working"
- Added conditional "Needs Attention" section (amber border, only renders when dead/stalled agents exist)
- Updated stats bar to show "4 working +2 warning" format when dead/stalled agents exist

---

## References

**Files Modified:**
- `web/src/lib/stores/agents.ts` - Added workingAgents and needsAttentionAgents derived stores
- `web/src/routes/+page.svelte` - Split UI into Working and Needs Attention sections

---

## Investigation History

**2026-01-02 00:30:** Investigation started
- Initial question: Separate dead agents from active agents in dashboard
- Context: Dashboard showed misleading "active" count that included dead agents

**2026-01-02 00:31:** Implementation completed
- Created workingAgents and needsAttentionAgents stores
- Updated dashboard UI with two distinct sections
- Visual verification confirmed working section shows 4 agents, stats bar shows "4 working"

**2026-01-02 00:32:** Investigation completed
- Status: Complete
- Key outcome: Dashboard now clearly separates working agents from agents needing attention
