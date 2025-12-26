# Session Synthesis

**Agent:** og-arch-design-how-better-26dec
**Issue:** ad-hoc (--no-track)
**Duration:** 2025-12-26
**Outcome:** success

---

## TLDR

Designed "Up Next" section for dashboard to surface high-priority ready queue items without requiring expand/collapse interaction. Recommends dedicated collapsible section (parallel to Pending Reviews) with auto-expand when P0/P1 items exist.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-design-how-better-surface-ready-queue.md` - Design investigation with recommendation

### Files Modified
- None (design-only session)

### Commits
- Pending (will commit investigation file)

---

## Evidence (What Was Observed)

- Current ready queue expands inline from stats bar, creating jarring UX that pushes content down (`web/src/routes/+page.svelte:487-550`)
- Stats bar already crowded with 6 indicators (errors, focus, servers, beads, daemon, connection)
- Pending Reviews section provides established pattern for collapsible actionable sections
- `/api/beads/ready` already returns priority, labels, age data needed for filtering
- `readyIssues` store already fetches every 60 seconds

### Code References
- Stats bar: `web/src/routes/+page.svelte:311-485`
- Ready queue section: `web/src/routes/+page.svelte:487-550`
- Pending Reviews pattern: `web/src/lib/components/pending-reviews-section/`
- Ready issues store: `web/src/lib/stores/beads.ts:67-93`
- API endpoint: `cmd/orch/serve.go:1391-1445`

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-design-how-better-surface-ready-queue.md` - Design recommendation

### Decisions Made
- **Approach 3: Dedicated "Up Next" Section** - Parallels Pending Reviews pattern, provides clean separation from stats bar, supports auto-expand for urgent items

### Constraints Discovered
- Stats bar space is limited - adding more inline content would make it unwieldy
- Dashboard is primarily for agent monitoring; queue is secondary concern
- Existing Pending Reviews pattern should be followed for consistency

### Externalized via `kn`
- N/A (design session, no operational decisions)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (design investigation produced)
- [x] Investigation file has `**Status:** Active` (will be updated to Complete)
- [ ] Ready for implementation via feature-impl spawn

### Implementation Ready

**Issue:** Implement Up Next section for ready queue visibility
**Skill:** feature-impl
**Context:**
```
Create UpNextSection component following Pending Reviews pattern. Shows top 5 priority 
items from ready queue, auto-expands when P0/P1 items exist, persists collapse state.
See design: .kb/investigations/2025-12-26-design-how-better-surface-ready-queue.md
```

**File Targets:**
- `web/src/lib/components/up-next-section/up-next-section.svelte` (create)
- `web/src/lib/components/up-next-section/index.ts` (create)
- `web/src/routes/+page.svelte` (modify - add section)

---

## Unexplored Questions

**Questions that emerged during this session:**
- Should clicking an item open beads-ui or show inline detail?
- How to handle focus alignment when no focus is set?
- Should age show absolute time or relative staleness threshold?

**Areas worth exploring further:**
- Desktop notifications for new P0/P1 items (Approach 4 as future enhancement)
- Blocking count as priority signal (requires dependency data from beads)

**What remains unclear:**
- Whether focus alignment check should be substring match or label match

---

## Session Metadata

**Skill:** architect
**Model:** Claude
**Workspace:** `.orch/workspace/og-arch-design-how-better-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-design-how-better-surface-ready-queue.md`
**Beads:** ad-hoc (--no-track)
