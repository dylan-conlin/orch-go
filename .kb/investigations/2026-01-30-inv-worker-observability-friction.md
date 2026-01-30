# Investigation: Worker Observability Friction

**Question:** What friction exists in spawning, monitoring, and completing workers?

**Started:** 2026-01-30
**Updated:** 2026-01-30
**Owner:** Dylan + Claude
**Phase:** Complete
**Status:** Complete

---

## Problem Framing

During a specs-platform session, we attempted to release 3 issues to the daemon for parallel processing. The experience revealed multiple friction points in the worker lifecycle.

---

## Findings

### Finding 1: Visibility is Poor

**Evidence:** Workers spawn headless. To determine if they're running, stuck, or dead requires:
- Checking `~/.orch/events.jsonl` for spawn/complete events
- Inspecting workspace directories for STATUS.md or SPAWN_CONTEXT.md
- Querying the OpenCode API for session status
- Running `orch status --all` and grepping for specific beads IDs

No single command shows "here's what's happening with my workers."

**Example session:**
```bash
# Had to run all of these to understand state:
cat ~/.orch/events.jsonl | grep specs-platform | grep "session.spawned" | tail -5
cat ~/.orch/events.jsonl | grep specs-platform | grep "agent.completed" | tail -5
orch status --all 2>/dev/null | grep -E "specs-platform-(38|10\.1|19)"
ls -la .orch/workspace/sp-feat-fix-test-critical-30jan-5cd6/
bd show specs-platform-38
```

**Significance:** Orchestrator (human or Claude) can't quickly assess swarm health.

---

### Finding 2: Beads Status Decoupled from Session State

**Evidence:** `bd show specs-platform-38` shows `Status: in_progress` but this doesn't indicate:
- Whether an agent is actively working
- Whether the session died/abandoned
- How long it's been running
- What phase it's in

The beads status is set when spawned but not updated if the session fails silently.

**Significance:** `in_progress` is ambiguous - could mean "actively working" or "zombie."

---

### Finding 3: Daemon Spawn Workflow Has Multiple Gates

**Evidence:** Releasing issues to daemon required 3 fix-and-push cycles:

| Attempt | Rejection Reason | Fix Required |
|---------|------------------|--------------|
| 1 | `status is in_progress (already being worked on)` | Reset to `open` |
| 2 | `missing label 'triage:ready'` | Add label |
| 3 | `blocked by dependencies: specs-platform-10 (open)` | Remove parent dep |

Each gate is reasonable in isolation, but the feedback loop is slow (edit → commit → push → daemon poll → check preview).

**Related bug filed:** `orch-go-21070` - Daemon rejects in_progress issues even when no session exists.

**Significance:** Releasing work to daemon is not a single action.

---

### Finding 4: No Progress Feedback During Execution

**Evidence:** Workers spawned 30+ minutes ago. No way to know:
- Current phase (Planning? Implementation? Validation?)
- Percentage complete
- Blockers encountered
- Estimated time remaining

The only feedback is eventual completion or silence.

**Significance:** Can't distinguish "working hard" from "stuck" from "dead."

---

### Finding 5: Task Review Happens After Spawn, Not Before

**Evidence:** `specs-platform-19` (sheet export) was spawned before Dylan could review what it would do. He had to pause it mid-flight after seeing the phase comments:

```
Phase: Design - Architecture: 1) Extend GoogleSheetsClient for write operations...
Phase: Implementation - Starting with GoogleSheetsClient write methods
```

The task was premature - admin-ETL coexistence wasn't even implemented yet.

**Significance:** Spawn is fire-and-forget. No "here's what I'm about to do, proceed?" gate.

---

### Finding 6: Epic Children Auto-Depend on Epic

**Evidence:** Creating tasks with `--parent specs-platform-10` automatically added a dependency on the epic. This blocked spawning because the epic was open.

```
specs-platform-10.1: blocked by dependencies: specs-platform-10 (open)
```

Had to manually remove dependencies:
```bash
bd dep remove specs-platform-10.1 specs-platform-10
bd dep remove specs-platform-10.2 specs-platform-10
bd dep remove specs-platform-10.3 specs-platform-10
```

**Significance:** Parent != dependency. Epic is a container, not a blocker.

---

## Synthesis

**Core issue:** Spawning is easy, observing is hard.

The daemon efficiently processes queues and spawns workers, but the human/orchestrator experience degrades once work is in flight:

1. **Pre-spawn:** Multiple gates with slow feedback loops
2. **During execution:** No progress visibility
3. **Post-spawn:** Status is stale, completion detection requires spelunking

---

## Recommendations

### Short-term (UX improvements)

1. **`orch workers` command** - Show all active workers with beads ID, phase, runtime, last activity
2. **`bd show` enhancement** - Include session status (active/idle/dead) when in_progress
3. **`orch daemon preview --verbose`** - Show all gates and why each issue passes/fails upfront

### Medium-term (workflow changes)

4. **Spawn confirmation for P3+** - "About to spawn specs-platform-19 (Sheet export). Proceed? [y/n]"
5. **Parent vs dependency distinction** - `--parent` should set hierarchy, not block
6. **Phase comments visible in `orch status`** - Already partially there, but not for headless

### Long-term (architecture)

7. **Heartbeat/progress protocol** - Workers report phase transitions, daemon tracks liveness
8. **Dead session detection** - If session exits without Phase: Complete, mark as failed
9. **Dashboard real-time view** - WebSocket updates for worker progress

---

## Issues to Create

| Issue | Priority | Summary |
|-------|----------|---------|
| orch-go-21070 | P2 | Daemon rejects in_progress without checking session (already filed) |
| NEW | P2 | `--parent` creates blocking dependency, should be hierarchy only |
| NEW | P3 | Add `orch workers` command for active worker visibility |
| NEW | P3 | Dead session detection - mark abandoned workers as failed |

---

## References

- Session: specs-platform orchestrator session 2026-01-30
- Events: `~/.orch/events.jsonl`
- Workspaces: `.orch/workspace/sp-feat-*`
