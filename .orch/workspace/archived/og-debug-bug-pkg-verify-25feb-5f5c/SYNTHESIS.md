# Session Synthesis

**Agent:** og-debug-bug-pkg-verify-25feb-5f5c
**Issue:** orch-go-1222
**Outcome:** success

---

## Plain-Language Summary

`FallbackList()` in `pkg/beads/client.go` was running `bd list --json` without a `--limit` flag. The `bd` CLI defaults to returning 50 issues, so any project with more than 50 issues silently lost results. This affected 11 call sites across the codebase (verify, serve, daemon, swarm, handoff, reconcile). The fix adds `--limit 0` (meaning "no limit") to `FallbackList` itself, fixing all callers at once. Verified: this project has 497 open issues but only 50 were being returned before the fix.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

---

## Delta (What Changed)

### Files Modified
- `pkg/beads/client.go` - Added `"--limit", "0"` to `FallbackList` args (line 817)
- `pkg/beads/client_test.go` - Added `TestFallbackList_IncludesLimitFlag` with mock bd script

---

## Evidence (What Was Observed)

- `bd list --json` returns 50 issues (default limit)
- `bd list --json --limit 0` returns 497 issues (all open)
- `FallbackListWithLabel` (line 851) already had `--limit 0` — `FallbackList` was the inconsistent one
- `FallbackListByParent` (line 917) also already had `--limit 0`
- `CLIClient.List()` (cli_client.go:140) always passes `--limit` when args provided
- 11 callers of `FallbackList` across the codebase, all silently truncated

### Tests Run
```bash
go test ./pkg/beads/ -run TestFallbackList_IncludesLimitFlag -v
# PASS (0.01s) - both subtests pass

go test ./pkg/beads/ -v
# PASS (0.015s) - all beads tests pass

go test ./pkg/verify/ -v
# PASS (6.766s) - all verify tests pass

go build ./...
# Success - clean build
```

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for `orch complete orch-go-1222`

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-bug-pkg-verify-25feb-5f5c/`
**Beads:** `bd show orch-go-1222`
