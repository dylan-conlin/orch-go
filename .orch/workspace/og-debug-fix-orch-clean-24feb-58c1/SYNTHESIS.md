# Session Synthesis

**Agent:** og-debug-fix-orch-clean-24feb-58c1
**Issue:** orch-go-1221
**Outcome:** success

---

## Plain-Language Summary

`orch clean --sessions` was killing active daemon-spawned tmux windows because it only checked for active OpenCode sessions to determine if a tmux window was still in use. Daemon-spawned agents using Claude CLI have tmux windows but no OpenCode sessions, so they were always classified as stale and killed. The fix adds a second layer of protection: before marking a tmux window as stale, check if its beads issue is still open (in_progress/open/blocked). If the issue is open, the window is protected. This required using `ListOpenIssuesWithDir(projectDir)` instead of `ListOpenIssues()` because the latter had a hidden limit of 50 issues that missed recent in_progress issues.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace directory.

Key outcomes:
- `go test ./cmd/orch/ -run TestClassifyTmuxWindows` — 8/8 pass
- `go run ./cmd/orch/ clean --sessions --dry-run` — shows "Protected 5 windows with open beads issues" instead of killing them
- `go vet ./cmd/orch/` — clean

---

## Delta (What Changed)

### Files Created
- `cmd/orch/clean_cmd_test.go` - 8 test cases for `classifyTmuxWindows` covering all protection scenarios

### Files Modified
- `cmd/orch/clean_cmd.go` - Extracted `classifyTmuxWindows()` pure function, added beads issue check to `cleanStaleTmuxWindows()`, added `projectDir` parameter

### Key Changes
1. **New `staleTmuxWindow` struct** - Replaces anonymous struct for stale window tracking
2. **New `classifyTmuxWindows()` function** - Pure function that takes windows, active beads IDs, and open issues map; returns stale windows and protected count. Testable without mocking tmux/OpenCode.
3. **Beads check in `cleanStaleTmuxWindows()`** - Calls `verify.ListOpenIssuesWithDir(projectDir)` to get all open issues, then passes to `classifyTmuxWindows` for cross-reference
4. **`projectDir` parameter** - Added to `cleanStaleTmuxWindows()` signature so it can pass to `ListOpenIssuesWithDir` (which uses `--limit 0` for all issues)

---

## Evidence (What Was Observed)

- `ListOpenIssues()` via `FallbackList("")` only returns 50 issues (default `bd list` limit), missing in_progress issues beyond the top 50
- `ListOpenIssuesWithDir(projectDir)` uses `CLIClient.List(&ListArgs{Limit: 0})` which passes `--limit 0` to get ALL issues — correctly returns 47 open issues including orch-go-1221, orch-go-1217, orch-go-1210
- Before fix: dry-run showed "Found 5 stale tmux windows" (all would be killed)
- After fix: dry-run showed "Protected 5 windows with open beads issues" (none killed)

### Tests Run
```bash
go test ./cmd/orch/ -run TestClassifyTmuxWindows -v
# 8/8 PASS (0.01s)

go test ./cmd/orch/ -v -count=1
# All tests PASS (2.75s)

go vet ./cmd/orch/
# Clean

go run ./cmd/orch/ clean --sessions --dry-run
# Protected 5 windows with open beads issues (no OpenCode session)
# No stale tmux windows found
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `ListOpenIssues()` has a hidden 50-issue limit bug — `FallbackList("")` doesn't pass `--limit 0`, so projects with >50 issues may miss open ones. Use `ListOpenIssuesWithDir(projectDir)` instead.
- The beads RPC socket (`.beads/bd.sock`) may not be running, in which case `ListOpenIssues()` falls through to CLI fallback

### Decisions Made
- Extracted `classifyTmuxWindows()` as pure function for testability rather than adding beads mock infrastructure
- Used `ListOpenIssuesWithDir(projectDir)` instead of fixing `ListOpenIssues()` (out of scope, different root cause)
- If beads check fails, `cleanStaleTmuxWindows` returns error (safety-first: don't kill windows if can't verify beads status)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (8 new + all existing)
- [x] Ready for `orch complete orch-go-1221`

### Discovered Work
- `ListOpenIssues()` limit bug: `FallbackList("")` doesn't use `--limit 0`, so it only returns 50 issues. Any caller depending on it to find ALL open issues will miss some. Should be filed as a separate bug.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-fix-orch-clean-24feb-58c1/`
**Beads:** `bd show orch-go-1221`
