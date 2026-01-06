# Session Synthesis

**Agent:** og-work-dashboard-queue-visibility-26dec
**Issue:** orch-go-qnwi
**Duration:** 2025-12-26 ~12:00 → ~13:00
**Outcome:** success

---

## TLDR

Dashboard stats bar shows "50 ready" but no way to see actual issues. Investigated 4 options; recommend expandable queue section that reveals issue list inline when clicking the beads indicator.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-dashboard-queue-visibility-stats-bar.md` - Full investigation with D.E.K.N. summary and implementation recommendations

### Files Modified
- None (design investigation only)

### Commits
- None (no code changes in this design session)

---

## Evidence (What Was Observed)

- Current beads indicator (`+page.svelte:370-391`) shows only aggregate count with tooltip
- `bd ready --json` returns full issue objects with title, priority, type, labels
- Dashboard already uses `CollapsibleSection` pattern for Active/Recent/Archive
- Constraint exists: dashboard must work at 666px width (half MacBook Pro screen)
- Daemon indicator already shows queue relationship with ready_count + capacity

### Key Files Examined
```
web/src/routes/+page.svelte         # Stats bar, beads indicator
web/src/lib/stores/beads.ts         # Stats-only store
web/src/lib/stores/daemon.ts        # Daemon store pattern
cmd/orch/serve.go                   # Beads API endpoint
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-dashboard-queue-visibility-stats-bar.md` - Full design investigation

### Decisions Made
- Expandable queue section is best approach because:
  - Consistent with existing CollapsibleSection pattern
  - No new UI paradigms to learn
  - Respects 666px width constraint
  - No context switching required

### Alternatives Considered & Rejected
1. **Sidebar Panel** - Takes horizontal space, violates 666px constraint
2. **Separate /queue Route** - Requires navigation, defeats purpose
3. **Modal/Overlay** - Feels heavy for quick glance

### Implementation Sequence
1. Add `/api/beads/ready` endpoint (returns `bd ready --json`)
2. Create ReadyQueue svelte store
3. Make beads indicator clickable → toggle expanded state
4. Add CollapsibleSection below stats bar for queue items
5. Each item shows: title, priority badge, type, labels
6. Optional: Quick-action buttons (spawn, review)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-qnwi`

### Follow-up Implementation Work

**Create beads issue for implementation:**

```
Title: Implement expandable queue visibility in dashboard stats bar
Description: 
Following design investigation .kb/investigations/2025-12-26-inv-dashboard-queue-visibility-stats-bar.md:
- Add /api/beads/ready endpoint
- Make beads indicator clickable to expand
- Show ready queue inline with CollapsibleSection
- Respect 666px width constraint

Skill: feature-impl
Labels: triage:ready
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should triage:ready vs triage:review issues be visually differentiated?
- Should clicking an issue spawn immediately or show details modal?
- Should queue view have inline filtering/sorting?

**Areas worth exploring further:**
- Integration with daemon: could "spawn next" button auto-select from queue based on capacity?
- Priority-based queue ordering in the UI

**What remains unclear:**
- Performance with 100+ ready issues - may need virtual scrolling
- Exact spacing/layout at 666px width

---

## Session Metadata

**Skill:** design-session
**Model:** opus
**Workspace:** `.orch/workspace/og-work-dashboard-queue-visibility-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-dashboard-queue-visibility-stats-bar.md`
**Beads:** `bd show orch-go-qnwi`
