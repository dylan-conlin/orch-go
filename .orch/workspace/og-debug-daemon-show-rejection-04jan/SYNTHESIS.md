# Session Synthesis

**Agent:** og-debug-daemon-show-rejection-04jan
**Issue:** orch-go-78jw
**Duration:** 2026-01-04 ~22:30 UTC
**Outcome:** success

---

## TLDR

When `orch daemon preview` reported "no spawnable issues" without explanation, orchestrators had to debug invisible daemon state. Added `RejectedIssue` struct and modified `Preview()` to surface per-issue rejection reasons (missing type, wrong status, missing label, blocked by dependencies).

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/daemon.go` - Added `RejectedIssue` struct, `RejectedIssues` field to `PreviewResult`, new `checkRejectionReason()` method, and `FormatRejectedIssues()` function. Modified `Preview()` to collect all rejected issues with reasons.
- `cmd/orch/daemon.go` - Updated `runDaemonPreview()` and `runDaemonDryRun()` to display rejected issues section.
- `pkg/daemon/daemon_test.go` - Added tests: `TestDaemon_Preview_ShowsRejectionReasons`, `TestDaemon_Preview_ShowsMissingLabelRejection`, `TestFormatRejectedIssues`, `TestFormatRejectedIssues_Empty`.

### Commits
- (to be committed) - `fix: daemon preview shows rejection reasons for non-spawnable issues`

---

## Evidence (What Was Observed)

- Root cause: `Preview()` called `NextIssue()` which silently filtered issues, returning only "No spawnable issues" with no detail.
- Rejection logic existed in `NextIssueExcluding()` (lines 169-244) but was only visible in debug mode.
- All rejection scenarios identified:
  - Missing/empty type (null type in JSON was the original trigger)
  - Non-spawnable type (epic, chore)
  - Status is blocked
  - Status is in_progress
  - Missing required label
  - Has blocking dependencies

### Tests Run
```bash
# Unit tests for new functionality
go test ./pkg/daemon/... -v -run "TestDaemon_Preview_Shows|TestFormatRejected" -timeout 60s
# Result: PASS (10.331s)

# Smoke test - daemon preview
orch daemon preview
# Output shows rejected issues with reasons:
#   orch-go-78jw: status is in_progress (already being worked on)
#   orch-go-eysk: type 'epic' not spawnable (must be bug/feature/task/investigation)
#   orch-go-eysk.4: missing label 'triage:ready'
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-04-debug-daemon-show-rejection-reasons-preview.md` - Investigation file

### Decisions Made
- Decision: Collect ALL rejected issues (not just first rejection) to give orchestrator full visibility
- Decision: Check for empty type BEFORE `IsSpawnableType()` to give clearer error message

### Constraints Discovered
- beads.CheckBlockingDependencies() calls bd CLI and can hang in tests without mocking
- Tests that use real beads dependency checking need longer timeouts

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (new tests: 4 pass)
- [x] Smoke test verified fix works
- [ ] Investigation file updated (Phase: Complete)
- [ ] Ready for `orch complete orch-go-78jw`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Consider adding rejection reasons to daemon run verbose output (currently only in preview)
- Could track rejection metrics for daemon health monitoring

**Areas worth exploring further:**
- N/A - straightforward fix

**What remains unclear:**
- N/A

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude
**Workspace:** `.orch/workspace/og-debug-daemon-show-rejection-04jan/`
**Investigation:** `.kb/investigations/2026-01-04-debug-daemon-show-rejection-reasons-preview.md`
**Beads:** `bd show orch-go-78jw`
