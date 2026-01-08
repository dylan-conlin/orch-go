# Session Synthesis

**Agent:** og-feat-dashboard-agents-filter-07jan-c354
**Issue:** orch-go-dwuk3
**Duration:** 2026-01-07T16:45 → 2026-01-07T17:15
**Outcome:** success

---

## TLDR

Fixed dashboard agents filter to correctly show agents spawned with `--workdir` to other projects. The early project filter was using `s.Directory` (OpenCode session directory, which is always orch-go due to --attach bug) instead of `agent.ProjectDir` (correctly populated from workspace cache).

---

## Delta (What Changed)

### Files Created
- `cmd/orch/serve_filter_test.go` - Unit tests for filterByProject, filterByTime, parseSinceParam, and parseProjectFilter functions

### Files Modified
- `cmd/orch/serve_agents.go` - Removed early project filter that used wrong directory; added comment explaining why project filter must be deferred to late filtering

### Commits
- (Pending commit for these changes)

---

## Evidence (What Was Observed)

- Line 345 used `filterByProject(s.Directory, projectFilterParam)` - s.Directory is session directory (orchestrator's cwd), not target project
- Line 727 correctly populates `agent.ProjectDir` from workspace cache's `beadsProjectDirs` lookup
- Line 894 correctly uses `filterByProject(agent.ProjectDir, projectFilterParam)` - this filter was already correct

### Tests Run
```bash
# All tests pass
go test ./...
# PASS: all packages

# Build succeeds
go build ./cmd/orch
# Success
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-dashboard-agents-filter-session-directory.md` - Documents the bug and fix

### Decisions Made
- Removed early project filter rather than trying to get correct project_dir early: simpler fix, late filter already correct

### Constraints Discovered
- For --workdir spawns, OpenCode session directory is the orchestrator's cwd, not the target project
- Correct project_dir is only available after workspace cache lookup (from SPAWN_CONTEXT.md)

### Externalized via `kn`
- (None - bug fix follows existing patterns)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-dwuk3`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Performance impact of removing early project filter (expected minimal, main costs are workspace cache and beads batch fetch)

**Areas worth exploring further:**
- Could extract project from workspace name pattern for early filtering if performance becomes an issue

**What remains unclear:**
- Actual behavior in live dashboard with --workdir spawns (would need manual testing)

---

## Session Metadata

**Skill:** feature-impl
**Model:** Claude
**Workspace:** `.orch/workspace/og-feat-dashboard-agents-filter-07jan-c354/`
**Investigation:** `.kb/investigations/2026-01-07-inv-dashboard-agents-filter-session-directory.md`
**Beads:** `bd show orch-go-dwuk3`
