# Session Synthesis

**Agent:** og-inv-bd-sync-hangs-17feb-8bab
**Issue:** orch-go-1008
**Duration:** ~45 minutes
**Outcome:** success

---

## Plain-Language Summary

`bd sync` had a second deadlock path that wasn't covered by the recent fix (c2af5a82). When `bd sync` runs, it acquires an exclusive file lock on `.beads/jsonl.lock`, then calls `git commit`. The git commit triggers the pre-commit hook, which calls `bd sync --flush-only`, which tries to acquire the same lock the parent already holds - classic deadlock. The fix adds `--no-verify` to all internal `bd sync` git commits, since the pre-commit hook's flush is redundant during sync (the export was already done). This is a beads bug, not orch-go, but it blocked all beads operations in any project with hooks installed.

## Verification Contract

- **Reproduction**: `timeout 15 bd sync` in orch-go with unstaged changes
- **Before fix**: Hangs indefinitely at "Committing changes to git..."
- **After fix**: Fails fast with `error: cannot pull with rebase: You have unstaged changes` (exit code 1, ~1 second)
- **Tests**: `TestBuildGitCommitArgs_ConfigOptions`, `TestGitCommitBeadsDir_PathspecDoesNotCommitOtherStagedFiles`, `TestImportFromJSONL_NoDbMode_NoDeadlock` - all PASS

---

## TLDR

Fixed a second `bd sync` deadlock: the pre-commit hook calls `bd sync --flush-only` which blocks on the same flock the parent `bd sync` holds. Fix: `--no-verify` on internal git commits. Also killed ~30 zombie `bd` processes and cleaned up stale git lock files.

---

## Delta (What Changed)

### Files Modified
- `~/Documents/personal/beads/cmd/bd/sync_git.go` - Added `--no-verify` to `buildGitCommitArgs()` to prevent pre-commit hook deadlock
- `~/Documents/personal/beads/cmd/bd/sync_helpers_more_test.go` - Updated test to expect `--no-verify` in commit args

### Files Created
- `.kb/models/beads-integration-architecture/probes/2026-02-17-bd-sync-precommit-hook-deadlock.md` - Detailed probe documenting the deadlock

### Cleanup
- Removed stale `.git/next-index-14337.lock` (Jan 22) and `.git/next-index-34125.lock` (Feb 16)
- Killed ~30 zombie `bd` processes (sync, comments, list, ready, stats, show)

---

## Evidence (What Was Observed)

- `bd version` confirmed patched binary (v0.41.0, f9c1bc9f) — the first deadlock fix IS present
- `ps -eo pid,ppid,pgid,stat,lstart,command | grep "bd "` showed 28+ hanging processes including `bd sync` from 12+ hours ago
- `lsof -p 34059` confirmed hanging `bd sync` held `.beads/jsonl.lock` with open pipes to non-existent children
- `timeout 5 git pull --rebase --dry-run` fails fast (exit 128) — NOT a git hang
- `timeout 10 bd sync` reproduced the hang at "Committing changes to git..."
- `ps` during hang showed new `bd sync --flush-only` child spawned by pre-commit hook
- After fix: `timeout 15 bd sync` completes in ~1 second with proper error
- After fix: `orch status` completes instantly (previously timed out)

### Tests Run
```bash
cd ~/Documents/personal/beads && go test -run "TestGitCommitBeadsDir_PathspecDoesNotCommitOtherStagedFiles|TestBuildGitCommitArgs|TestImportFromJSONL_NoDbMode_NoDeadlock" ./cmd/bd/ -v -timeout 30s
# PASS: all 3 tests passing (0.160s)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/beads-integration-architecture/probes/2026-02-17-bd-sync-precommit-hook-deadlock.md` - Second deadlock path documentation

### Decisions Made
- Decision: Use `--no-verify` on ALL internal bd git commits (not just sync) because `buildGitCommitArgs()` is only called from internal bd operations where the pre-commit hook's flush is always redundant

### Constraints Discovered
- `bd sync` + pre-commit hook is a circular dependency: sync → git commit → hook → flush → lock → deadlock
- Zombie `bd` processes accumulate: each deadlock creates 2+ orphaned processes that never release locks
- The first deadlock fix (c2af5a82) only covered the `importFromJSONL()` path, not the git commit path

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] Root cause identified and reproduced
- [x] Fix applied and tested in beads repo
- [x] Reproduction verified: `bd sync` no longer deadlocks
- [x] `orch status` no longer times out
- [x] Probe file complete
- [ ] Beads fix needs its own commit (in beads repo, not orch-go)

**Note:** The beads repo change needs to be committed separately. This issue's deliverable is the investigation + fix.

---

## Unexplored Questions

- **Process accumulation mitigation**: Should `bd` have a timeout on flock acquisition instead of blocking forever? A 30-second timeout with error message would prevent zombie accumulation.
- **Cross-project impact**: This deadlock affects ANY project with beads hooks in noDb mode, not just orch-go. Price-watch was also affected.
- **Hook architecture**: Should pre-commit hooks check `BD_SYNC_IN_PROGRESS` env var instead of relying on `--no-verify`? The current fix works but is somewhat opaque.

---

## Session Metadata

**Skill:** investigation (probe mode)
**Workspace:** `.orch/workspace/og-inv-bd-sync-hangs-17feb-8bab/`
**Probe:** `.kb/models/beads-integration-architecture/probes/2026-02-17-bd-sync-precommit-hook-deadlock.md`
**Beads:** `bd show orch-go-1008`
