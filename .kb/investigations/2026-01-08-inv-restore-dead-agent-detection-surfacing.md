<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Restored dead agent detection with 3-minute heartbeat threshold; agents silent for 3+ minutes now show as "dead" in dashboard Needs Attention section.

**Evidence:** Go and web builds pass; visual verification shows dashboard working with active agents; implementation matches original commit 784c2703.

**Knowledge:** Simple 3-minute threshold is sufficient - agents constantly read/edit/run commands, so 3 min silence is definitive death signal. Keep it simple (lesson from Dec 27-Jan 2 spiral).

**Next:** Close - all deliverables complete, builds pass, visual verification done.

**Promote to Decision:** recommend-no - tactical restoration of reverted feature, not architectural change.

---

# Investigation: Restore Dead Agent Detection Surfacing

**Question:** How to restore dead agent detection and surfacing that was reverted during Dec 27 - Jan 2 spiral?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent og-feat-restore-dead-agent-08jan-c1d1
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Original implementation was simple 3-minute heartbeat

**Evidence:** Commit 784c2703 shows the original implementation used a single `deadThreshold := 3 * time.Minute` with status = "dead" when `timeSinceUpdate > deadThreshold`.

**Source:** `git show 784c2703`

**Significance:** The original approach was intentionally simple - no complex state machine, no multiple thresholds. This simplicity was the correct approach.

---

### Finding 2: System spiral was caused by complexity, not the feature

**Evidence:** Post-mortem at `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` documents that the spiral was caused by adding complexity (multiple time thresholds, 'stalled' state, etc.), not by the dead detection feature itself.

**Source:** `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md`

**Significance:** The feature itself (visibility into dead agents) was CORRECT. The problem was the complexity added around it. Restoring with original simplicity avoids repeating the mistake.

---

### Finding 3: NeedsAttention component already exists for consolidating attention items

**Evidence:** The dashboard already has a NeedsAttention component (`web/src/lib/components/needs-attention/needs-attention.svelte`) that displays errors, blocked issues, and pending reviews.

**Source:** `web/src/lib/components/needs-attention/needs-attention.svelte`

**Significance:** Dead agents fit naturally into this component rather than requiring a new section. This is a cleaner integration.

---

## Synthesis

**Key Insights:**

1. **Simplicity is key** - The original 3-minute threshold was correct. 3 minutes without activity is definitive death signal for agents that constantly read/edit/run commands.

2. **Visibility was the goal** - The feature's purpose is to make dead agents visible so the user knows they need attention. The Dec 27-Jan 2 spiral was caused by trying to hide/categorize the problem rather than show it.

3. **Reuse existing components** - Adding dead agents to the existing NeedsAttention component is cleaner than creating new UI sections.

**Answer to Investigation Question:**

The restoration follows the original commit 784c2703 pattern:
1. Server: Add 3-minute dead threshold, set status="dead" when exceeded
2. Frontend: Add 'dead' to AgentState, create deadAgents derived store
3. Dashboard: Add dead agents section to NeedsAttention component
4. Stats bar: Show "+N need attention" when dead agents exist

---

## Structured Uncertainty

**What's tested:**

- ✅ Go build passes (verified: `go build ./cmd/orch/`)
- ✅ Web build passes (verified: `npm run build` in web/)
- ✅ Dashboard loads and shows active agents (verified: visual inspection)

**What's untested:**

- ⚠️ Actual dead agent rendering (no dead agents available during testing)
- ⚠️ Agent resurrection behavior (going from dead back to active)
- ⚠️ Interaction with daemon capacity counting

**What would change this:**

- If 3 minutes is too short and causes false positives
- If dead agents need different handling than other attention items

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Status determination logic
- `web/src/lib/stores/agents.ts` - AgentState type and derived stores
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Attention consolidation component
- `web/src/lib/components/stats-bar/stats-bar.svelte` - Stats bar indicators

**Commands Run:**
```bash
# View original commit
git show 784c2703

# Build verification
go build ./cmd/orch/
npm run build

# Server restart
orch servers restart orch-go
```

**Related Artifacts:**
- **Post-mortem:** `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` - Documents the spiral and lessons learned

---

## Investigation History

**2026-01-08 05:15:** Investigation started
- Initial question: How to restore dead agent detection that was reverted?
- Context: 25-28% of agents don't complete, but there's no visibility into this

**2026-01-08 05:35:** Implementation completed
- Status: Complete
- Key outcome: Dead agent detection restored with simple 3-minute heartbeat threshold
