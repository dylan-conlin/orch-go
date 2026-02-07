# Session Synthesis

**Agent:** og-debug-daemon-spawns-duplicate-06jan-504f
**Issue:** orch-go-rlew4
**Duration:** 2026-01-06 → 2026-01-06
**Outcome:** success

---

## TLDR

Fixed daemon spawning duplicate agents for the same issue by adding SpawnedIssueTracker that tracks issue IDs before calling spawnFunc, preventing the race condition where beads status hasn't updated to "in_progress" yet.

---

## Delta (What Changed)

### Files Created
- `pkg/daemon/spawn_tracker.go` - SpawnedIssueTracker type with TTL-based tracking of spawned issue IDs
- `pkg/daemon/spawn_tracker_test.go` - Comprehensive test coverage for tracker and daemon integration

### Files Modified
- `pkg/daemon/daemon.go` - Added SpawnedIssues field to Daemon struct; integrated tracking into NextIssueExcluding, OnceExcluding, OnceWithSlot, and ReconcileWithOpenCode
- `.kb/investigations/2026-01-06-inv-daemon-spawns-duplicate-agents-same.md` - Investigation documentation with D.E.K.N. summary

### Commits
- (to be committed) - fix: prevent daemon from spawning duplicate agents for same issue

---

## Evidence (What Was Observed)

- Original bug: 4 agents spawned for same issue kb-cli-0kk during single polling period
- Root cause: Race between daemon polling and beads status update - daemon sees issue as "open" before `orch work` marks it "in_progress"
- Existing in_progress check (daemon.go:244) only works AFTER status update propagates
- WorkerPool, RateLimiter, and skippedThisCycle don't prevent per-issue duplicates

### Tests Run
```bash
# All daemon tests pass
go test ./pkg/daemon/... -count=1
# ok  	github.com/dylan-conlin/orch-go/pkg/daemon	3.645s

# New spawn tracker tests pass
go test ./pkg/daemon/... -v -count=1 -run "SpawnedIssue|SkipsRecentlySpawned|OnceMarks|PreventsDuplicate"
# PASS: 10 tests
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-06-inv-daemon-spawns-duplicate-agents-same.md` - Full investigation with root cause analysis

### Decisions Made
- Decision 1: Use local spawn tracker (not synchronous status update) because it doesn't slow down spawn flow
- Decision 2: 5-minute TTL for tracked entries because it's conservative and entries expire naturally
- Decision 3: Clean stale entries in ReconcileWithOpenCode because it runs at start of each poll cycle

### Constraints Discovered
- Beads status update is async - happens late in spawn flow (spawn_cmd.go:698)
- Daemon's stateless polling design requires explicit tracking to prevent race conditions

### Externalized via `kn`
- N/A - Findings documented in investigation file

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (go test ./pkg/daemon/... passes)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-rlew4`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should TTL be configurable via daemon config? (current hardcoded 5 minutes is likely sufficient)
- Could we use the existing Pool.slots BeadsID tracking instead? (simpler but Pool is optional)

**Areas worth exploring further:**
- Reconcile tracked issues with actual beads status (remove entries when confirmed in_progress/closed)

**What remains unclear:**
- Optimal TTL value for spawn tracking (5 minutes is conservative estimate based on typical spawn times)

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-daemon-spawns-duplicate-06jan-504f/`
**Investigation:** `.kb/investigations/2026-01-06-inv-daemon-spawns-duplicate-agents-same.md`
**Beads:** `bd show orch-go-rlew4`
