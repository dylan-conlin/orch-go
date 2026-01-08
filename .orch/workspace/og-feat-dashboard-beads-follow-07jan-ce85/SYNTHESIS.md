# Session Synthesis

**Agent:** og-feat-dashboard-beads-follow-07jan-ce85
**Issue:** orch-go-tatzw
**Duration:** 2026-01-07T23:40:00 → 2026-01-08T00:05:00
**Outcome:** success

---

## TLDR

Added project_dir parameter to /api/beads and /api/beads/ready endpoints, making beads stats and ready queue follow the orchestrator's tmux context. The dashboard now shows beads from the project the orchestrator is currently working in, instead of always showing orch-go's beads.

---

## Delta (What Changed)

### Files Created
- `cmd/orch/serve_beads_test.go` - Tests for project-aware beads API endpoints

### Files Modified
- `cmd/orch/serve_beads.go` - Added project_dir query param to /api/beads and /api/beads/ready, made cache project-aware
- `cmd/orch/serve_agents.go` - Updated invalidate() call to pass empty string (clears all project caches)
- `web/src/lib/stores/beads.ts` - Updated fetch methods to accept optional projectDir parameter
- `web/src/routes/+page.svelte` - Added reactive refetch of beads when orchestrator context changes

### Commits
- Not yet committed (changes ready for commit)

---

## Evidence (What Was Observed)

- API endpoint correctly returns project_dir in response: `/api/beads?project_dir=/Users/dylanconlin/Documents/personal/orch-go` returns `{"project_dir":"/Users/dylanconlin/Documents/personal/orch-go",...}`
- Different project directories return different results (orch-knowledge returned 0 issues, orch-go returned 1581)
- Dashboard shows "Following" toggle enabled with 35 ready issues from orch-go beads

### Tests Run
```bash
go test -v ./cmd/orch -run "TestHandleBeads|TestBeadsStatsCache" -count=1
# PASS: TestHandleBeadsMethodNotAllowed (0.00s)
# PASS: TestHandleBeadsWithProjectParam (0.01s)
# PASS: TestHandleBeadsReadyWithProjectParam (0.01s)
# PASS: TestBeadsStatsCacheProjectAwareness (0.00s)
# PASS: TestHandleBeadsReadyMethodNotAllowed (0.00s)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md` - Investigation file

### Decisions Made
- Decision: Use per-project cache entries (map keyed by project_dir) rather than global cache
  - Rationale: Allows concurrent dashboard users to view different projects without cache collisions
- Decision: Use CLIClient with WorkDir for non-default projects instead of relying on global beadsClient
  - Rationale: The global beadsClient is connected to orch-go's daemon; cross-project queries need CLI fallback

### Constraints Discovered
- CLI fallback for cross-project queries fails with "bd not found in PATH" in server context
  - This is a known issue (orch-go-loev7) - bd CLI is slow/broken in launchd environments
  - Impact: Cross-project beads queries may fail if bd isn't in server's PATH

### Externalized via `kn`
- No new kn entries needed - constraints already tracked in existing issues

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Status:** Active` (update to Complete before close)
- [x] Ready for `orch complete orch-go-tatzw`

### Follow-up Work (Already Tracked)
- orch-go-loev7: bd CLI slow in launchd/minimal env - impacts cross-project beads queries

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should the beads stats cache TTL be shorter (15s instead of 30s) when following orchestrator context? The context can change frequently.
- Should there be a visual indicator in the dashboard showing which project's beads are being displayed?

**What remains unclear:**
- How will this interact with multi-project focus configurations (e.g., when orchestrator is focused on multiple projects at once)

---

## Session Metadata

**Skill:** feature-impl
**Model:** Claude Opus
**Workspace:** `.orch/workspace/og-feat-dashboard-beads-follow-07jan-ce85/`
**Investigation:** `.kb/investigations/2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md`
**Beads:** `bd show orch-go-tatzw`
