# Session Synthesis

**Agent:** og-debug-orientation-frame-backend-19feb-6cce
**Issue:** orch-go-1129
**Outcome:** success

---

## Plain-Language Summary

When `--backend claude` was passed to `orch spawn` without also passing `--tmux`, the agent ran headless through the OpenCode API instead of in a tmux window via Claude CLI. The root cause was that the resolve pipeline computed backend and spawn mode independently — so even when backend was "claude", the spawn mode defaulted to "headless", and the headless check in `DispatchSpawn` ran before the claude backend check, short-circuiting the tmux path. The fix adds a post-resolution step: when backend is "claude" and spawn mode was not explicitly set, the spawn mode is derived as "tmux".

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

---

## Delta (What Changed)

### Files Modified
- `pkg/spawn/resolve.go` - Added post-resolution step in `Resolve()` that derives spawn mode to "tmux" when backend is "claude" and no explicit spawn mode flag was set
- `pkg/spawn/resolve_test.go` - Added 4 tests covering: claude-implies-tmux, explicit-headless-overrides, explicit-tmux-stays-explicit, infra-escape-hatch-also-implies-tmux

---

## Evidence (What Was Observed)

- Root cause traced through 3 layers:
  1. `resolveSpawnMode()` (resolve.go:292) defaults to `SpawnModeHeadless` when no CLI flag set
  2. `applyResolvedSpawnMode()` (spawn_cmd.go:368) sets `input.Headless = true` from resolved mode
  3. `DispatchSpawn()` (extraction.go:990) checks `input.Headless` BEFORE `cfg.SpawnMode == "claude"`, so claude path is never reached

### Tests Run
```bash
go test ./pkg/spawn/... -run "TestResolve_BugClass13" -v
# PASS: 4 new tests (13, 13b, 13c, 13d)

go build ./cmd/orch/ && go vet ./cmd/orch/
# PASS: builds and vets clean

go test ./pkg/orch/...
# PASS: all orch tests pass

# 6 pre-existing failures in resolve_test.go (not caused by this change, verified by stash/test/pop)
```

---

## Next (What Should Happen)

**Recommendation:** close

- [x] Fix implemented addressing root cause
- [x] 4 regression tests added
- [x] Build and vet clean
- [x] No new test failures introduced
