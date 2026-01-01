# Decision: Daemon Skips Failing Issues Per Spawn Cycle

**Status:** Accepted
**Date:** 2025-12-31
**Context:** Daemon queue processing behavior when spawn failures occur

---

## Summary

When a spawn fails (e.g., unfilled failure report, workspace conflict), the daemon skips that issue for the remainder of the current spawn cycle and tries the next eligible issue. Failed issues are retried on the next poll cycle (60 seconds later).

## Context

The daemon was discovered to have a blocking behavior where any spawn failure would halt the entire spawn loop:

```go
// BEFORE: Any failure breaks the loop
if !result.Processed {
    break  // Blocks ALL remaining issues
}
```

This meant that a single issue with an unfilled failure report could block the entire queue indefinitely, including issues from other projects that had no relationship to the failing issue.

## Decision

**Implement per-cycle skip tracking for failed spawns.**

Implementation:
- `NextIssueExcluding(skip map[string]bool)` - Returns next eligible issue not in skip set
- `OnceExcluding(skip map[string]bool)` - Attempts spawn, adds to skip set on failure
- Daemon loop maintains per-cycle skip map, cleared at start of each cycle
- Failed issues logged for visibility, but don't block other work

## Rationale

1. **Queue fairness** - One issue's blocker shouldn't penalize unrelated work

2. **Automatic retry** - Issues are only skipped for one cycle; they're retried after potential fixes are deployed

3. **Cross-project isolation** - Work from project A shouldn't be blocked by failure report gates in project B

4. **Debugging visibility** - Skipped issues are logged, making the behavior transparent

## Consequences

**Positive:**
- Daemon processes entire queue each cycle (capacity permitting)
- Issues with transient failures recover automatically
- Clear logs show which issues were skipped and why

**Negative:**
- Failed issues generate repeated skip logs until resolved
- Slight memory overhead for skip map (negligible)

**Neutral:**
- Root cause of spawn failure still needs human attention
- Skip tracking doesn't persist across daemon restarts

## Alternative Considered

**Mark issues as blocked in beads:**
- Would persist state across daemon restarts
- Would require human intervention to unblock
- Rejected because most spawn failures are transient (unfilled reports get filled)

## References

- `.kb/investigations/2025-12-30-inv-daemon-blocked-cross-project-failure.md` - Root cause investigation
- `pkg/daemon/daemon.go:NextIssueExcluding()` - Skip-aware issue selection
- `cmd/orch/daemon.go` - Per-cycle skip map in spawn loop
