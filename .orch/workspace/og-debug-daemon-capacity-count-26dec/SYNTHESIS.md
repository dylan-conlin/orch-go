# Session Synthesis

**Agent:** og-debug-daemon-capacity-count-26dec
**Issue:** orch-go-per9
**Duration:** 16:47 → 17:10
**Outcome:** success

---

## TLDR

Fixed daemon capacity staleness by adding beads issue status check to `DefaultActiveCount()`. The function now excludes sessions whose beads issues are closed, correctly reflecting actual running agents rather than all recently-updated sessions.

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/daemon.go:418-550` - Modified `DefaultActiveCount()` to check beads issue status via new `getClosedIssuesBatch()` helper
- `pkg/daemon/daemon_test.go` - Added tests for `getClosedIssuesBatch()`

### Commits
- `09a90b0d` - Session Dec 26 evening: Review UI, multi-project design, daemon fixes (includes this fix)

---

## Evidence (What Was Observed)

- Daemon showed 3/3 capacity while `orch status` showed only 1-2 running agents
- `DefaultActiveCount()` was returning 7 (all recent sessions) instead of 2 (only open issues)
- OpenCode sessions persist after agent completion with recent "updated" timestamps
- After fix, daemon correctly reconciled: pool went from 3 → 2, then spawned 1 new agent

### Root Cause Chain
1. Agent completes work and calls `/exit`
2. `orch complete` closes beads issue
3. OpenCode session remains with recent "updated" timestamp
4. `DefaultActiveCount()` counted session as "active" (updated in last 30 min)
5. `Pool.Reconcile()` received inflated count (7 >= 3), so no slots freed
6. Daemon stuck at capacity even though agents completed

### Tests Run
```bash
# All daemon tests pass
go test ./pkg/daemon/... -count=1
# ok  	github.com/dylan-conlin/orch-go/pkg/daemon	0.166s

# Verified DefaultActiveCount returns correct count
go run /tmp/test_daemon_count.go
# DefaultActiveCount() = 2 (correctly excludes 5 closed issues)

# Verified daemon spawns after reconciliation
tail ~/.orch/daemon.log
# [16:59:40] Spawned: orch-go-afsz (feature-impl)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-debug-daemon-capacity-stale-after-complete.md` - Root cause analysis documenting session persistence issue

### Decisions Made
- Check beads status via RPC rather than tracking session IDs explicitly - simpler, uses existing infrastructure
- Batch beads lookups for performance - `getClosedIssuesBatch()` queries multiple issues in one call

### Constraints Discovered
- OpenCode sessions persist indefinitely after agent completion
- Session "updated" timestamp reflects last output, not agent state
- Beads issue status is authoritative for agent lifecycle (open/in_progress = running, closed = done)

### Prior Fixes (Incremental)
This issue had multiple prior investigations:
1. Added `Pool.Reconcile()` - but received wrong counts
2. Added 30-min recency filter - didn't account for recent completion
3. Added untracked agent filter - didn't account for closed issues
4. **This fix** - checks beads status, finally correct

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-per9`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should OpenCode auto-close sessions after agent exit? Would simplify capacity tracking
- Should `orch complete` notify daemon directly? Would provide instant slot release without polling

**Areas worth exploring further:**
- Performance of beads batch lookups at high session counts
- Daemon behavior when beads daemon is unavailable (currently falls back to CLI)

**What remains unclear:**
- OpenCode session lifecycle - why do sessions persist indefinitely?

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-daemon-capacity-count-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-debug-daemon-capacity-stale-after-complete.md`
**Beads:** `bd show orch-go-per9`
