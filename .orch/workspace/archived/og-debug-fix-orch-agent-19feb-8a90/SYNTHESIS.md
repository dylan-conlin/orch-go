# Session Synthesis

**Agent:** og-debug-fix-orch-agent-19feb-8a90
**Issue:** orch-go-1096
**Outcome:** success

---

## Plain-Language Summary

The `orch:agent` label was added to beads issues at spawn time but never removed when agents completed or closed. This meant `bd list -l orch:agent` returned all historically spawned agents (not just active ones), making any query that scans by this label O(historical) instead of O(active). The fix adds label removal in two places: the `orch complete` flow (Go code) and the `.beads/hooks/on_close` hook (shell script), ensuring both completion paths clean up the label. Existing closed issues were also cleaned up — count went from 13 to 2 (only actually active agents).

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

---

## Delta (What Changed)

### Files Modified
- `pkg/verify/beads_api.go` - Added `RemoveOrchAgentLabel()` function (follows same RPC+CLI fallback pattern as `RemoveTriageReadyLabel`)
- `cmd/orch/complete_cmd.go` - Added call to `verify.RemoveOrchAgentLabel(beadsID)` after closing beads issue
- `.beads/hooks/on_close` - Added `bd label remove "$ISSUE_ID" orch:agent` for the `bd close` bypass path

---

## Evidence (What Was Observed)

- Before fix: `bd list -l orch:agent` returned 13 issues (11 closed, 2 active)
- After cleanup: `bd list -l orch:agent` returns 2 issues (both active)
- `untagBeadsAgent()` already existed in `pkg/spawn/atomic.go` but was only used for rollback during failed spawns
- The `RemoveTriageReadyLabel` pattern in `pkg/verify/beads_api.go` provided the exact template for the new function

### Tests Run
```bash
go build ./cmd/orch/     # Clean build
go vet ./cmd/orch/       # No issues
go test ./pkg/verify/    # PASS (5.762s)
go test ./cmd/orch/      # PASS (2.552s)
```

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for `orch complete orch-go-1096`
