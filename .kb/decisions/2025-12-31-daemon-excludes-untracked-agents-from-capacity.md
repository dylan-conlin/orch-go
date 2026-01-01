# Decision: Daemon Excludes Untracked Agents from Capacity Count

**Status:** Accepted
**Date:** 2025-12-31
**Context:** Daemon capacity management for autonomous agent spawning

---

## Summary

The daemon's capacity counting excludes agents spawned with `--no-track` (untracked agents) from the active count. Only tracked agents (those spawned by the daemon or with explicit beads issues) count against the daemon's capacity limit.

## Context

During December 2025, the daemon implementation went through several iterations of capacity counting bugs:
1. Initially counted all OpenCode sessions regardless of recency
2. Added 30-minute recency filter (still overcounted)
3. Added untracked agent filter (this decision)
4. Added closed issue filter (final piece)

The core insight was that untracked agents (`orch spawn --no-track`) are explicitly opted out of daemon management. They represent ad-hoc exploratory work that shouldn't consume daemon capacity slots.

## Decision

**Daemon capacity counts only tracked agents with open beads issues.**

Implementation:
- Extract beads ID from session title using `[beads-id]` pattern
- Check if beads ID contains `-untracked-` substring
- Skip sessions with untracked beads IDs when computing active count
- Skip sessions with closed beads issues when computing active count

## Rationale

1. **Explicit opt-out should be respected** - The `--no-track` flag exists precisely so users can spawn agents without affecting daemon capacity

2. **Prevents capacity starvation** - Without this filter, overnight exploratory work could consume all daemon slots

3. **Aligns with daemon purpose** - The daemon manages tracked work from the beads backlog; untracked work is outside its scope

4. **Consistent with orch review behavior** - The `orch review` command already identifies untracked agents for display purposes using the same `-untracked-` pattern

## Consequences

**Positive:**
- Daemon capacity accurately reflects managed work
- Users can freely spawn untracked agents without blocking daemon
- Clear separation between ad-hoc and managed work

**Negative:**
- Slight complexity in capacity calculation (title parsing, beads ID extraction)
- Sessions without beads IDs in title default to "tracked" (conservative behavior)

**Neutral:**
- Untracked agents still visible in `orch status`, just not counted for daemon capacity

## Supersedes

This decision consolidates findings from:
- `.kb/investigations/2025-12-26-inv-daemon-capacity-count-stale-after.md` - Added untracked filtering
- Multiple capacity counting investigations that incrementally fixed the issue

## References

- `pkg/daemon/daemon.go:extractBeadsIDFromSessionTitle()` - Title parsing
- `pkg/daemon/daemon.go:isUntrackedBeadsID()` - Untracked detection
- `pkg/daemon/daemon.go:DefaultActiveCount()` - Capacity counting
