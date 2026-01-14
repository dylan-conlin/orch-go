# Session Synthesis

**Agent:** og-debug-fix-dashboard-excessive-05jan-c973
**Issue:** orch-go-xeppr
**Duration:** 2026-01-05 21:00 → 21:10
**Outcome:** success

---

## TLDR

Fixed dashboard request storm by adding in-flight tracking (`isFetching`, `needsRefetch`) to prevent concurrent agent fetch requests - the debounce alone wasn't sufficient because it didn't prevent new requests from starting while one was already in-flight.

---

## Delta (What Changed)

### Files Modified
- `web/src/lib/stores/agents.ts` - Added `isFetching` and `needsRefetch` state tracking, modified `fetch()` to prevent concurrent requests and queue follow-up fetches, updated `cancelPending()` to reset new state variables

### Commits
- Pending commit

---

## Evidence (What Was Observed)

- SSE `session.status` events trigger `fetchDebounced()` on every agent busy/idle transition (agents.ts:541-551)
- The 500ms debounce correctly collapses rapid calls, but when the timer fires and `fetch()` is called, it would abort any in-flight request and start a new one (agents.ts:190-216)
- Multiple SSE events arriving during a fetch caused cascade of aborted/pending requests
- `onOpen` callback bypasses debounce, calling `fetch()` directly (agents.ts:412-415)

### Tests Run
```bash
# Build verification
/opt/homebrew/bin/bun run build
# PASS: build completed successfully

# State variable usage verification
grep -n "isFetching\|needsRefetch" web/src/lib/stores/agents.ts
# All 12 usages consistent with fix design
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-05-inv-fix-dashboard-excessive-agents-fetch.md` - Full investigation details

### Decisions Made
- Decision: Use in-flight tracking pattern instead of more aggressive debounce - Because we still want fresh data after each SSE event batch, just without request storms
- Decision: Queue follow-up fetch via `fetchDebounced()` (not direct `fetch()`) - Because this gives another 500ms window to collapse any additional events that arrived

### Constraints Discovered
- Debouncing alone doesn't prevent concurrent requests - must track in-flight state
- SSE events arrive frequently during active agent work (every busy/idle transition)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Build passing
- [x] Investigation file updated
- [x] Fix implemented and verified via build
- [ ] Browser verification recommended (orchestrator should verify Network panel)

### Browser Verification Required

**To verify fix works correctly:**
1. Open dashboard at http://localhost:5188
2. Open browser DevTools → Network tab
3. Filter by "agents"
4. With active agents running, observe network requests
5. **Expected (fixed):** Single clean requests with 200 responses, no canceled or pending storms
6. **Before fix (broken):** Many rapid requests, many "(canceled)", many pending

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could the `onOpen` callback also use debounced fetch? Currently it calls `fetch()` directly (agents.ts:412-415)
- Are there other SSE event handlers that trigger fetches that might have similar issues?

**What remains unclear:**
- Exact frequency of SSE events during heavy agent activity (would require instrumentation to measure)

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** Claude
**Workspace:** `.orch/workspace/og-debug-fix-dashboard-excessive-05jan-c973/`
**Investigation:** `.kb/investigations/2026-01-05-inv-fix-dashboard-excessive-agents-fetch.md`
**Beads:** `bd show orch-go-xeppr`
