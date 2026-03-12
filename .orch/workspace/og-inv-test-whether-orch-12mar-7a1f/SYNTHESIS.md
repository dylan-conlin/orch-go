# Session Synthesis

**Agent:** og-inv-test-whether-orch-12mar-7a1f
**Issue:** orch-go-84uad
**Duration:** 2026-03-12 → 2026-03-12
**Outcome:** success

---

## Plain-Language Summary

Tested the orch spawn system for duplicate spawn race conditions by writing 8 Go tests exercising concurrent and sequential spawn scenarios. Found that the daemon's production path (sequential polling with PID lock) is correctly protected against duplicates — the spawn tracker blocks sequential re-spawns, and the beads status update (FreshStatusGate) serializes concurrent access. A real data race bug was found in `session_dedup.go`'s lazy initialization of the default checker (not using sync.Once), which was fixed. The manual spawn path (`orch spawn --issue`) has a theoretical TOCTOU between status check and update, but this is low-risk because manual spawns are infrequent and mitigated by triage label removal.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace for verification claims and evidence.

Key outcomes:
- 8 race condition tests pass with Go `-race` detector (no data races after fix)
- Full `pkg/daemon/` test suite passes (15.9s, no regressions)
- `session_dedup.go` data race fixed with `sync.Once`

---

## TLDR

Investigated the orch spawn system for duplicate spawn race conditions. The daemon path is safe (sequential + PID lock). Found and fixed a real data race in session_dedup.go. The spawn pipeline gates have a TOCTOU under concurrent stress, but beads status is the structural backstop that prevents actual production duplicates.

---

## Delta (What Changed)

### Files Created
- `pkg/daemon/spawn_race_test.go` — 8 race condition tests covering concurrent, sequential, and TOCTOU scenarios
- `.kb/investigations/2026-03-12-inv-test-duplicate-spawn-race-condition.md` — Full investigation with findings, evidence, and recommendations

### Files Modified
- `pkg/daemon/session_dedup.go` — Fixed data race in `initDefaultSessionDedupChecker()`: replaced manual check-then-set with `sync.Once` to prevent concurrent read/write on package-level variable

### Commits
- (pending)

---

## Evidence (What Was Observed)

- `TestConcurrentOnce_SameIssue`: 10/10 goroutines passed all gates when FreshStatusGate always returns "open" — TOCTOU confirmed
- `TestConcurrentSpawnIssue_FreshStatusSerializes`: Only 1/10 goroutines succeeded when status reflects actual changes — beads is the real protection
- `TestSpawnTrackerGate_TOCTOU_Window`: 2/100 goroutines passed before first mark — quantifies the check-then-mark window
- `TestSequentialOnce_PreventsDuplicate`: Second sequential call correctly blocked — production pattern validated
- `go test -race`: Data race in session_dedup.go caught and fixed; all tests pass cleanly after fix
- `go test ./pkg/daemon/`: Full suite passes (15.9s, no regressions)

### Tests Run
```bash
# Race condition tests with Go race detector
go test -race -run "TestConcurrent|TestSpawnTracker|TestSequential|TestSpawnIssue_Sequential|TestManualSpawn" -v ./pkg/daemon/
# PASS: all 8 tests pass, no data races

# Full daemon package tests
go test ./pkg/daemon/
# ok github.com/dylan-conlin/orch-go/pkg/daemon 15.950s
```

---

## Architectural Choices

No architectural choices — investigation tested existing architecture and found it correct for the production use case.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-12-inv-test-duplicate-spawn-race-condition.md` — Full race condition investigation

### Constraints Discovered
- SpawnTrackerGate check-then-mark is NOT atomic (TOCTOU under concurrent stress)
- FreshStatusGate + beads status update is the real serialization mechanism
- Daemon loop is sequential (daemon_loop.go:355), making concurrent TOCTOU benign
- PID lock prevents multiple daemon instances (daemon_loop.go:63)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (8 race tests + full suite)
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-84uad`

---

## Unexplored Questions

- Whether beads supports CAS (compare-and-swap) semantics that could close the manual spawn TOCTOU structurally
- Whether two tmux windows can actually be created for the same beads ID in a real daemon + manual spawn race

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-test-whether-orch-12mar-7a1f/`
**Investigation:** `.kb/investigations/2026-03-12-inv-test-duplicate-spawn-race-condition.md`
**Beads:** `bd show orch-go-84uad`
