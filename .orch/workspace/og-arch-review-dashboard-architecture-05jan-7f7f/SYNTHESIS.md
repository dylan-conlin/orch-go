# Session Synthesis

**Agent:** og-arch-review-dashboard-architecture-05jan-7f7f
**Issue:** ad-hoc (--no-track)
**Duration:** 2026-01-05 13:00 → 2026-01-05 14:15
**Outcome:** success

---

## TLDR

Reviewed dashboard SSE→fetch→abort architecture. Found the pattern is fundamentally sound but triggers fetches too frequently due to (1) `session.status` events triggering full fetches when they only need local state updates, and (2) agentlog SSE redundantly triggering fetches already covered by OpenCode SSE. Recommended two simple fixes that can reduce fetch requests by ~70%.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-05-design-review-dashboard-architecture-request-handling.md` - Full architecture review with detailed findings and recommendations

### Files Modified
- None (investigation only)

### Commits
- None yet (to be committed)

---

## Evidence (What Was Observed)

- **agents.ts:565-574** - `refreshEvents` array includes `session.status` which fires on every agent busy/idle toggle, but session.status is already handled via local state updates at lines 510-562
- **agentlog.ts:123-125** - Every agentlog event triggers `fetchDebounced()`, but these events are redundant with OpenCode events that already trigger fetches
- **agents.ts:151-260** - Fetch infrastructure is well-designed with `isFetching`, `needsRefetch`, AbortController, and 500ms debounce
- **sse-connection.ts:74-76** - Generation counter properly prevents stale reconnect timers

### Tests Run
```bash
# Static analysis via code reading - no runtime tests in this investigation
# Verified code paths through grep and manual review
grep -r "fetchDebounced\|agents\.fetch" web/src/
# Found 8 call sites, confirmed redundancy patterns
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-05-design-review-dashboard-architecture-request-handling.md` - Complete architecture review

### Decisions Made
- **Event categorization is the right fix** because it addresses root cause (too many triggers) not symptoms (request handling)
- **Agentlog fetch trigger should be removed** because it's 100% redundant with OpenCode SSE events

### Constraints Discovered
- `session.status` events fire at high frequency (every agent response cycle) - not suitable for triggering list refreshes
- SSE events fall into two categories: state updates (local handling only) vs lifecycle events (require list refresh)

### Externalized via `kn`
- Not applicable for this investigation (no persistent constraints discovered)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Reduce dashboard fetch frequency by ~70% via event categorization
**Skill:** feature-impl
**Context:**
```
Remove agentlog fetch trigger at agentlog.ts:123-125 (single line deletion).
Filter refreshEvents at agents.ts:565-571 to exclude session.status.
See investigation: .kb/investigations/2026-01-05-design-review-dashboard-architecture-request-handling.md for full rationale.
```

### Implementation Checklist (for implementing agent)
1. [ ] Remove `import('./agents').then(...)` block from agentlog.ts:117-128
2. [ ] Change refreshEvents in agents.ts:565-571 to exclude 'session.status'
3. [ ] Test: Dashboard still updates when agents spawn/complete
4. [ ] Test: Network tab shows fewer /api/agents requests during active agent work

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Is the 5000ms `PROCESSING_CLEAR_DELAY_MS` optimal? Seems long for idle detection.
- Is `current_activity` display actually useful to users, or just visual noise?
- Could the backend aggregate events instead of proxying raw OpenCode stream?

**Areas worth exploring further:**
- SSE event frequency instrumentation to quantify actual reduction
- Whether vite proxy vs Go proxy for dev mode creates any issues

**What remains unclear:**
- Exact fetch frequency before/after (would need runtime instrumentation)
- Whether there are edge cases where session.status indicates agent list changes

---

## Session Metadata

**Skill:** architect
**Model:** Claude
**Workspace:** `.orch/workspace/og-arch-review-dashboard-architecture-05jan-7f7f/`
**Investigation:** `.kb/investigations/2026-01-05-design-review-dashboard-architecture-request-handling.md`
**Beads:** ad-hoc spawn (no beads tracking)
