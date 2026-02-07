# Investigation: Infrastructure Root Cause Synthesis

**Date:** 2026-01-14
**Status:** Complete
**Trigger:** Weeks of recurring dashboard failures, exhaustion from circular debugging

## The Question

Why does the 3-service dashboard architecture keep failing despite multiple fix attempts?

## What We Found

### Three Distinct Failure Modes

**1. Orphan Process Problem**
- When overmind dies (crash, socket deleted, etc.), child processes survive
- `orch serve`, `bun`, `opencode` keep running on their ports
- Next `overmind start` fails: "address already in use"
- This is NOT a race condition - it's orphan management failure

**2. Socket File Fragility**
- Overmind uses `.overmind.sock` in working directory
- If deleted (cleanup, manual rm, confusion), overmind becomes unreachable
- `overmind status` fails, but processes keep running
- Creates orphans (failure mode #1)
- Evidence: 25+ stale sockets in /tmp/tmux-501/ from one day

**3. Auto-Start Race (in .zshrc)**
```bash
if [[ -f ~/Documents/personal/orch-go/Procfile ]] && ! overmind status &>/dev/null; then
    (cd ~/Documents/personal/orch-go && rm -f .overmind.sock && overmind start -D &>/dev/null &)
fi
```
- Multiple shells starting after reboot each run this check
- `overmind status` can return false during startup window
- Multiple overmind instances attempted → conflicts

### Why Previous Fixes Failed

| Attempt | Why It Failed |
|---------|---------------|
| launchd for each service | PATH issues, didn't solve orphan problem |
| launchd supervising overmind | Circular dependency, still had orphans |
| Socket cleanup in .zshrc | Only cleans socket, doesn't kill orphans |
| Documentation | Can't fix a process management problem |

### Why Other Projects Don't Have This Problem

Rails/Node `foreman start` "just works" because:
- Started manually from single terminal (no race)
- Fresh start each time (no orphans from prior runs)
- No auto-start from shell configs
- Dev stops services when done (clean state)

### Root Cause Summary

**Overmind + orphan processes + auto-start from shell = fragile system state**

The architecture (3 services via overmind) is sound. The failure is in lifecycle management:
- No cleanup of orphans before start
- Socket file can be accidentally deleted
- Auto-start isn't atomic

## Evidence

- 25+ overmind sockets in /tmp from Jan 14 alone
- `orch serve` (PID 97042) survived overmind death, blocked port 3348
- Manual cleanup + restart always works (proves architecture is fine)

## Recommendation

Fix lifecycle management, not architecture:
1. Pre-start: Kill processes on 3348/5188/4096
2. Pre-start: Clean stale socket
3. Start: Run overmind
4. Post-start: Verify actually running

This is what manual recovery does. Automate it.

## References

- `.kb/decisions/2026-01-14-infrastructure-complexity-justified.md` - Prior decision to keep architecture
- `.kb/decisions/2026-01-10-dev-vs-prod-architecture.md` - Dev vs prod separation
- `.kb/guides/dev-environment-setup.md` - Current (incomplete) guidance
