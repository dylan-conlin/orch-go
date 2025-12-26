# Session Synthesis

**Agent:** og-debug-dashboard-active-section-26dec
**Issue:** orch-go-yz06
**Duration:** 2025-12-26 09:42 → 2025-12-26 09:58
**Outcome:** success

---

## TLDR

Fixed dashboard Active section incorrectly showing agents with both 'active' and 'Complete' badges by removing the guards that prevented Phase: Complete agents from being marked as completed even when their OpenCode session still exists. Phase: Complete is the definitive completion signal.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve.go:666-691` - Removed `status != "active"` guards from Phase: Complete and SYNTHESIS.md checks

### Commits
- (pending) - fix: mark Phase: Complete agents as completed regardless of session state

---

## Evidence (What Was Observed)

- API returned agents with `status: 'active'` AND `phase: 'Complete'` simultaneously (3 such agents)
- Prior fix (orch-go-6xya) added guards that prevented status change to "completed" for agents with active sessions
- This created the visual bug: agents showing both "active" and "Complete" badges in dashboard
- After removing guards, API correctly returns `status: 'completed'` for all Phase: Complete agents

### Tests Run
```bash
# Before fix - 3 agents with active + Complete
curl -s http://127.0.0.1:3348/api/agents | jq '[.[] | select(.status == "active" and .phase == "Complete")] | length'
# 3

# Build and restart
make install
orch servers stop orch-go && orch servers start orch-go

# After fix - 0 agents with both badges
curl -s http://127.0.0.1:3348/api/agents | jq '[.[] | select(.status == "active" and .phase == "Complete")] | length'
# 0

# Unit tests
go test ./cmd/orch/... -v -run TestServe
# PASS: 10 tests

# Playwright tests
npx playwright test filtering.spec.ts
# PASS: 8 tests
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Phase: Complete is the definitive signal for agent work completion
- An open OpenCode session just means the agent hasn't called /exit yet, but the work is done
- If an agent is resumed after Phase: Complete, a new Phase comment (e.g., "Phase: Implementing") would supersede the Complete status

### Constraints Discovered
- Prior fix (guards for active sessions) was overly conservative and caused the opposite bug
- The correct hierarchy: Phase: Complete > session activity state

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (go test, playwright)
- [x] Dashboard correctly shows completed agents in Recent section
- [x] Ready for `orch complete orch-go-yz06`

---

## Unexplored Questions

**Edge case: Agent resumed after Phase: Complete**
- If an agent is resumed (via orch resume), it should report a new Phase comment to update status
- Worth verifying this flow works correctly

*(Straightforward fix, minimal unexplored territory)*

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** Claude Opus
**Workspace:** `.orch/workspace/og-debug-dashboard-active-section-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-dashboard-active-section-not-showing.md` (prior investigation, different bug)
**Beads:** `bd show orch-go-yz06`
