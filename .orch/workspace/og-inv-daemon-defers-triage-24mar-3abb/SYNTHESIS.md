# Session Synthesis

**Agent:** og-inv-daemon-defers-triage-24mar-3abb
**Issue:** orch-go-natal
**Outcome:** success

---

## Plain-Language Summary

The daemon's test-issue deferral logic trusted the beads ready list blindly. When a ghost issue (`orch-go-ehz`) appeared in the ready list but didn't actually exist in beads, it permanently blocked 4 test issues from spawning. The fix adds an optional sibling validation step: before deferring a test issue, the daemon now verifies the blocking sibling actually exists by calling `GetIssueStatus`. Ghost siblings are skipped.

---

## TLDR

Ghost issue `orch-go-ehz` in the beads ready queue permanently blocked 4 test issues. Fixed by adding sibling existence verification to `ShouldDeferTestIssue` — siblings that don't exist in beads are now skipped.

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/sibling_sequencing.go` - Added `SiblingExistsFunc` type and optional validator parameter to `ShouldDeferTestIssue`
- `pkg/daemon/ooda.go` - Wired sibling validator into `Decide()` using `GetIssueStatus` with per-cycle caching
- `pkg/daemon/preview.go` - Wired sibling validator into `Preview()` matching Decide() pattern
- `pkg/daemon/sibling_sequencing_test.go` - Added 4 ghost-sibling tests + 1 Decide integration test, updated existing tests for new signature

### Files Created
- `.kb/investigations/2026-03-24-inv-daemon-defers-triage-ready-issues.md` - Investigation document

---

## Evidence (What Was Observed)

- `ShouldDeferTestIssue` at `sibling_sequencing.go:59` iterated `allIssues` without verifying siblings exist in beads
- `IssueQuerier.GetIssueStatus()` at `interfaces.go:10` already provides existence checks via `bd show`
- Ghost issue `orch-go-ehz` appears in `ListReadyIssues()` but `GetIssueStatus()` returns error (not found)

### Tests Run
```bash
go test ./pkg/daemon/ -count=1
# ok  github.com/dylan-conlin/orch-go/pkg/daemon  17.770s

go test ./pkg/daemon/ -run "TestShouldDeferTestIssue|TestDecide_Defers|TestDecide_Spawns|TestDecide_Ignores" -v
# 16/16 PASS
```

---

## Architectural Choices

### Validator function parameter vs verify-in-Decide
- **What I chose:** `SiblingExistsFunc` parameter on `ShouldDeferTestIssue` — keeps the function testable and the validation explicit
- **What I rejected:** Extracting sibling ID from the reason string in Decide() and verifying there
- **Why:** String parsing is fragile; the function parameter is type-safe, testable, and allows nil for backwards compatibility
- **Risk accepted:** Extra `GetIssueStatus` calls per cycle (mitigated by per-cycle cache)

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

Key outcomes:
- Ghost siblings are skipped during test issue deferral
- Real siblings still correctly block test issues
- Nil validator preserves backwards-compatible behavior
- All 170+ daemon tests pass

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Beads ready list can contain ghost issues (exist in ready set but not in show path) — data inconsistency bug in beads itself

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (17.7s, all pass)
- [x] Investigation file has Phase: Complete
- [x] Ready for `orch complete orch-go-natal`

---

## Unexplored Questions

- Why does `orch-go-ehz` appear in `bd ready` but not `bd show`? (Beads data inconsistency — separate bug)

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-daemon-defers-triage-24mar-3abb/`
**Investigation:** `.kb/investigations/2026-03-24-inv-daemon-defers-triage-ready-issues.md`
**Beads:** `bd show orch-go-natal`
