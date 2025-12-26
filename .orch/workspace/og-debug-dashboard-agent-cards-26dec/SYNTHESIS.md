# Session Synthesis

**Agent:** og-debug-dashboard-agent-cards-26dec
**Issue:** orch-go-g0ja
**Duration:** 2025-12-26 09:29 → 2025-12-26 10:20
**Outcome:** success

---

## TLDR

Fixed dashboard agent card jostling and gold border flashing by removing `is_processing` from sort order when stable sort is enabled, and adding 1-second debounce before clearing processing state to prevent rapid visual flapping.

---

## Delta (What Changed)

### Files Modified
- `web/src/routes/+page.svelte` - Skip `is_processing` comparison in `sortAgents()` when `useStableSort=true`
- `web/src/lib/stores/agents.ts` - Added debounced clearing of `is_processing` state (1000ms delay)

### Commits
- (pending) Fix dashboard agent card jostling and gold border flashing

---

## Evidence (What Was Observed)

- Traced SSE event flow: `session.status` events toggle `is_processing` via `busy`/`idle` states
- In `sortAgents()`, `is_processing` comparison happens BEFORE `spawned_at` stable sort: `+page.svelte:198-201`
- Multiple active agents cycling between busy/idle causes constant sort order changes
- API correctly returns `spawned_at` field - confirmed via `curl http://127.0.0.1:3348/api/agents`

### Tests Run
```bash
# TypeScript check
cd web && bun check
# svelte-check found 0 errors and 0 warnings

# Playwright tests
cd web && bunx playwright test
# 34 passed, 4 skipped (agent-detail panel tests need mock setup)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-dashboard-agent-cards-rapidly-jostling.md` - Root cause analysis and fix documentation

### Decisions Made
- Decision: Skip `is_processing` in sort when `useStableSort=true` rather than removing it entirely, because it may still be useful for non-stable sort modes (Recent/Archive sections)
- Decision: Use 1000ms debounce for clearing `is_processing` state - immediate set, delayed clear prevents flapping while staying responsive

### Constraints Discovered
- Stable sort only works if ALL dynamic/volatile criteria are excluded from sort logic
- Visual state changes need debouncing when triggered by high-frequency SSE events

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-g0ja`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `is_processing` sorting be available as a user-configurable option? (e.g., "Sort active to top" toggle)
- Is 1000ms the optimal debounce delay, or should it be configurable?

**Areas worth exploring further:**
- Performance profiling to confirm reduced re-renders
- Whether memoization of sort results could further reduce unnecessary updates

**What remains unclear:**
- Exact frequency of SSE events in heavy usage scenarios

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-20250514
**Workspace:** `.orch/workspace/og-debug-dashboard-agent-cards-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-dashboard-agent-cards-rapidly-jostling.md`
**Beads:** `bd show orch-go-g0ja`
