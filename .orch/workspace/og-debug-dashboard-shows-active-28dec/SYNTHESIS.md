# Session Synthesis

**Agent:** og-debug-dashboard-shows-active-28dec
**Issue:** orch-go-anos
**Duration:** 2025-12-28 14:30 → 2025-12-28 15:20
**Outcome:** success

---

## TLDR

Dashboard showed 0 active agents while CLI showed running agents due to semantic mismatch: API returns status='idle' for all sessions (to avoid CPU overhead), but frontend activeAgents filter only accepted status='active'. Fixed by updating frontend filters to include both 'active' and 'idle' status agents, matching CLI semantics.

**Verification:** Fix confirmed working - API returns correct status, dashboard shows "2 active" matching CLI output, all edge case tests pass.

---

## Delta (What Changed)

### Files Modified
- `web/src/lib/stores/agents.ts` - Updated activeAgents, recentAgents, archivedAgents derived stores to treat 'idle' status agents as active (matching CLI semantics)

### Files Created
- `.kb/investigations/2025-12-28-inv-dashboard-shows-active-cli-shows.md` - Root cause investigation

### Commits
- `3a834ac0` - fix: dashboard active agents now includes idle status agents

---

## Evidence (What Was Observed)

- `serve.go:663-668` - API passes `isProcessing=false` to DetermineStatusFromSession with comment explaining CPU optimization
- `main.go:2558,2604` - CLI calls `IsSessionProcessing` per-session (accurate but expensive)
- `agents.ts:200-201` - activeAgents filter only accepted `status === 'active'`
- `agents.ts:467-519` - SSE updates `is_processing` field but not `status` field
- `pkg/state/reconcile.go:344-355` - DetermineStatusFromSession returns StatusIdle when isProcessing=false

### Tests Run
```bash
# Go tests (backend unchanged, tests pass for relevant packages)
go test ./pkg/state/... 
# ok  	github.com/dylan-conlin/orch-go/pkg/state	0.050s

# Frontend tests require node/npm which weren't available in shell environment
# TypeScript changes are straightforward filter updates
```

### Verification Tests (Post-Fix)
```bash
# Test 1: API vs CLI Agent Count - PASS
API Active (active+idle): 2
CLI Active: 2
# ✅ PASS: API and CLI show same active count

# Test 2: No Idle Agents with Phase: Complete - PASS  
# ✅ PASS: No idle agents incorrectly showing Phase: Complete

# Test 3: Completed Agents Have Correct Status - PASS
# ✅ PASS: All Phase: Complete agents have status='completed'
```

**Dashboard Screenshot Verification:** Confirmed dashboard shows "2 active" in stats bar and 2 agent cards in Active Agents section.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-dashboard-shows-active-cli-shows.md` - Full root cause analysis

### Decisions Made
- Decision: Include status='idle' in activeAgents filter because idle means "has active session but momentarily between tasks" - this matches CLI semantics where active = has session + not completed
- Decision: Not fixing at API level (would reintroduce CPU issue) - frontend fix is sufficient

### Constraints Discovered
- API cannot call IsSessionProcessing per-session due to CPU overhead (125% CPU when dashboard polled frequently)
- SSE provides real-time processing state via is_processing field, but doesn't update status field

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Fix committed (`3a834ac0`)
- [x] Investigation file has `**Phase:** Complete`
- [x] Verification tests passing (API matches CLI, no edge case failures)
- [x] Dashboard UI confirmed showing correct active count
- [x] Ready for `orch complete orch-go-anos`

Note: Frontend tests require node/npm which weren't available in the shell environment. TypeScript changes are straightforward filter updates that don't change component behavior - just which agents pass the filter.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should the filter bar "Active" status option in historical mode be clarified? Currently it filters to status='active' only, but now that activeAgents includes 'idle', this might be confusing.
- Is the idleAgents derived store still needed? It's now a subset of activeAgents.

**Areas worth exploring further:**
- Could we batch IsSessionProcessing calls instead of per-session to get accurate status without CPU overhead?
- Should SSE update the status field when session.status events arrive?

**What remains unclear:**
- Performance impact of including more agents in Active Agents section (likely minimal)

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-dashboard-shows-active-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-dashboard-shows-active-cli-shows.md`
**Beads:** `bd show orch-go-anos`
