<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch reconcile --fix --all --mode close` failed to close zombie issues due to three bugs: error return values ignored, missing `--force` flag, and bd CLI returning exit 0 on soft errors.

**Evidence:** Tested with real zombie issues - before fix, issues remained in_progress; after fix, issues correctly closed.

**Knowledge:** Zombie reconciliation requires `--force` because zombies inherently lack "Phase: Complete" comments (they were abandoned). CLI fallback is needed because RPC daemon may be unhealthy.

**Next:** Fix implemented and verified. No further action needed.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural)

---

# Investigation: orch reconcile --fix Doesn't Close Zombie Issues

**Question:** Why doesn't `orch reconcile --fix --all --mode close` actually close zombie beads issues?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Error Return Values Ignored

**Evidence:** In `runReconcileFix()`, the error return value from `applyFix()` was ignored in both `--all` mode and interactive mode:
```go
if reconcileFixAll {
    applyFix(z, reconcileFixMode, "")  // Error ignored!
    continue
}
```

**Source:** `cmd/orch/reconcile.go:262-264`

**Significance:** Even when close operations failed, the loop continued without reporting the failure to the user. This made failures invisible.

---

### Finding 2: Missing `--force` Flag

**Evidence:** The `bd close` command requires a "Phase: Complete" comment unless `--force` is used. Zombie issues are inherently incomplete (no agent working on them), so they typically don't have this comment.

**Source:** Tested with: `BEADS_NO_DAEMON=1 bd close orch-go-2rtc --reason "Zombie reconciled"` returned "Error: cannot close orch-go-2rtc: no 'Phase: Complete' comment found (use --force to override)"

**Significance:** The `applyFix` function was calling `bd close` without `--force`, which failed for most zombies. This is the root cause of the bug.

---

### Finding 3: bd CLI Returns Exit Code 0 on Soft Errors

**Evidence:** Running `bd close` on an issue without "Phase: Complete" prints an error message but returns exit code 0. The `FallbackClose` function only checks `err` (the exit code), not the output.

**Source:** Tested with: `bd close orch-go-2rtc --reason "test" ; echo "Exit code: $?"` - Output showed "Error: cannot close..." but exit code was 0.

**Significance:** This caused `FallbackClose` to report success even when the close failed. This is a bug in the beads CLI (should return non-zero exit on failure), but we can work around it by checking output.

---

## Synthesis

**Key Insights:**

1. **Silent failures cascade** - Three bugs combined: ignored errors, missing force flag, and exit code 0 on soft errors. Each alone might have been caught, but together they created a completely silent failure mode.

2. **Zombie reconciliation requires force** - Zombies are abandoned work. They won't have "Phase: Complete" comments. Using `--force` is semantically correct for this operation.

3. **RPC path was also failing** - The RPC daemon reported "database disk image is malformed" during health check, causing fallback to CLI. This didn't cause the bug but revealed it was consistently hitting the CLI path.

**Answer to Investigation Question:**

`orch reconcile --fix` wasn't closing zombies because it was calling `bd close` without `--force`, which fails on issues without "Phase: Complete" comments. The failure was invisible because: (1) error return values were ignored, (2) `bd close` returns exit 0 even on soft errors. The fix adds `--force` for zombie close operations and properly reports success/failure for each operation.

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch reconcile --fix --all --mode close` now correctly closes zombies (verified: ran against orch-go-2rtc, status changed to closed)
- ✅ Success/failure feedback is now displayed (verified: output shows "✓ Closed" or "✗ Failed" for each issue)
- ✅ Summary counts are now shown (verified: output shows "Reconcile complete: N succeeded, M failed")

**What's untested:**

- ⚠️ Interactive mode (tested only --all mode, but code path is the same)
- ⚠️ Reset mode (only tested close mode)
- ⚠️ Cross-platform (only tested on macOS)

**What would change this:**

- Finding would be wrong if `bd close --force` had unexpected side effects
- Finding would be wrong if zombie issues actually should require Phase: Complete

---

## Implementation Recommendations

### Recommended Approach ⭐

**Use --force for zombie close operations** - Zombies are abandoned work that inherently lack Phase: Complete comments.

**Why this approach:**
- Semantically correct - zombies are abandoned, force is appropriate
- Simple implementation - just add --force flag to CLI call
- Works reliably - bypasses daemon health issues by using CLI fallback

**Trade-offs accepted:**
- Bypasses the Phase: Complete safety check for zombie operations
- Acceptable because zombies are being cleaned up, not completed work

**Implementation sequence:**
1. Add proper error handling and user feedback in runReconcileFix
2. Use CLI fallback with --force for close operations
3. Check output for error patterns since bd CLI returns 0 on soft errors

### Alternative Approaches Considered

**Option B: Add Force field to RPC CloseArgs**
- **Pros:** Would work through daemon, more integrated
- **Cons:** Daemon was unhealthy (database corruption), requires beads changes
- **When to use instead:** When daemon stability improves

**Option C: Fix bd CLI exit codes**
- **Pros:** Fixes root cause in beads
- **Cons:** Requires beads repo change, doesn't fix force issue
- **When to use instead:** Should be done anyway as separate fix

---

## References

**Files Examined:**
- `cmd/orch/reconcile.go` - Main implementation of reconcile command
- `pkg/beads/client.go` - RPC client and FallbackClose implementation
- `pkg/beads/types.go` - CloseArgs struct definition

**Commands Run:**
```bash
# Test reconcile flow
orch reconcile --min-age 0 --project orch-go --fix --all --mode close

# Verify close behavior
bd close orch-go-2rtc --reason "test" ; echo "Exit code: $?"

# Test force close
bd close orch-go-2rtc --force --reason "test"
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-arch-orch-reconcile-fix-07jan-d2b1/`

---

## Investigation History

**2026-01-07 16:47:** Investigation started
- Initial question: Why doesn't orch reconcile --fix --all --mode close actually close zombie issues?
- Context: Previous session reported the bug, needed root cause analysis

**2026-01-07 17:05:** Found ignored error return values in runReconcileFix

**2026-01-07 17:15:** Discovered bd CLI returns exit 0 on soft errors

**2026-01-07 17:20:** Identified missing --force flag as root cause

**2026-01-07 17:32:** Investigation completed
- Status: Complete
- Key outcome: Fixed by adding --force, proper error handling, and output checking
