# Session Handoff - 2026-01-08 (Evening)

## Session Focus
Continued agent visibility work + fixed critical spawn bug discovered via dead agent investigation.

## Key Accomplishments

| Feature | Status | Notes |
|---------|--------|-------|
| **Dashboard activity fix** | ✅ Done | Fixed "Starting up..." always showing - API sends string, frontend expected object |
| **Dead agent investigation** | ✅ Done | ok-9ph0 was orphaned due to invalid beads_id - led to root cause discovery |
| **Spawn validation fix** | ✅ Done | `resolveShortBeadsID` now fails if issue doesn't exist (was silently returning invalid ID) |

## The Bug We Fixed

**Problem:** `orch spawn --issue ok-9ph0` succeeded even though `ok-9ph0` was never a beads issue.

**Root cause:** `resolveShortBeadsID()` returned the invalid ID with just a warning instead of an error.

**Impact:** Agents could spawn with beads_ids that don't exist, making them impossible to properly close. They'd complete work, report "Task complete!", but have nowhere to close the issue - appearing as "dead" orphans.

**Fix:** Now returns error with helpful cross-project hints:
```
beads issue 'kb-cli-xyz123' not found

Hint: Issue 'kb-cli-xyz123' may belong to a different project.
If the issue is in 'kb-cli', try:
  cd ~/Documents/personal/kb-cli && orch complete kb-cli-xyz123
```

## Key Insight from This Session

The deeper question "why not require every spawn has a beads issue?" led us to discover the system *intends* this but had a bug in enforcement. The lenient error handling in `resolveShortBeadsID` was the gap.

**Agent lifecycle understanding gained:**
- Every spawn should have a beads issue (unless `--no-track`)
- If `--issue X` is provided but X doesn't exist, spawn should FAIL, not proceed with invalid ID
- Orphaned agents happen when this invariant is violated

## Files Changed This Session

- `web/src/lib/stores/agents.ts` - Transform API string current_activity to object format
- `cmd/orch/shared.go` - resolveShortBeadsID now returns error when issue not found
- `cmd/orch/main_test.go` - Updated test expectations for new strict behavior

## Git Status
- All changes committed and pushed to origin/master
- Working tree clean

## Next Session Should
1. **Consider daemon auto-cleanup** - Now that we understand the lifecycle, implement daemon closing sessions that report "Phase: Complete" + idle
2. **Watch for** - Any other places where invalid beads_ids might slip through
3. **Completion rate** - Should improve now that orphans won't be created

## Resume Commands
```bash
cd ~/Documents/personal/orch-go
orch status
orch stats  # Should show improved completion rate over time
```

## Key Investigations Referenced
- `.kb/investigations/2026-01-08-inv-25-28-agents-not-completing.md` - Why completion rate looked low
- `.kb/investigations/2026-01-08-inv-restore-dead-agent-detection-surfacing.md` - Dead agent detection design
