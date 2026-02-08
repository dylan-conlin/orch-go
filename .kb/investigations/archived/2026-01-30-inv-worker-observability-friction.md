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

### Finding 1: Cross-Project Workers Invisible in orch status

**Evidence:** Workers spawned for specs-platform don't appear in `orch status` at all when run from orch-go (or any other project).

```bash
$ orch status --all | grep specs-platform
# (no output)
```

Yet events.jsonl confirms they were spawned:
```bash
$ cat ~/.orch/events.jsonl | grep specs-platform | grep "session.spawned" | tail -3
{"type":"session.spawned","session_id":"ses_3eeeb8508ffe...","data":{"beads_id":"specs-platform-38",...}}
{"type":"session.spawned","session_id":"ses_3eeeb27b3ffe...","data":{"beads_id":"specs-platform-19",...}}
{"type":"session.spawned","session_id":"ses_3eedefcdaffe...","data":{"beads_id":"specs-platform-10.1",...}}
```

`orch status` only shows workers from the current project (orch-go). Cross-project workers are invisible.

**Significance:** If you spawn workers for project A while in project B, you can't see them. This is the primary visibility gap.

---

### Finding 2: Even Within Project, Visibility Requires Spelunking

**Evidence:** To determine if workers are running, stuck, or dead requires:
- Checking `~/.orch/events.jsonl` for spawn/complete events
- Inspecting workspace directories for STATUS.md or SPAWN_CONTEXT.md
- Querying the OpenCode API for session status
- Running `orch status --all` and grepping for specific beads IDs

**Example session:**
```bash
# Had to run all of these to understand state:
cat ~/.orch/events.jsonl | grep specs-platform | grep "session.spawned" | tail -5
cat ~/.orch/events.jsonl | grep specs-platform | grep "agent.completed" | tail -5
ls -la .orch/workspace/sp-feat-fix-test-critical-30jan-5cd6/
bd show specs-platform-38
curl -s "http://127.0.0.1:4096/sessions/ses_3eeeb8508ffe..."
```

**Significance:** No single command shows "here's what's happening with my workers."

---

### Finding 3: Beads Status Decoupled from Session State

**Evidence:** `bd show specs-platform-38` shows `Status: in_progress` but this doesn't indicate:
- Whether an agent is actively working
- Whether the session died/abandoned
- How long it's been running
- What phase it's in

The beads status is set when spawned but not updated if the session fails silently.

**Significance:** `in_progress` is ambiguous - could mean "actively working" or "zombie."

---

### Finding 4: Daemon Spawn Workflow Has Multiple Gates

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

### Finding 5: No Progress Feedback During Execution

**Evidence:** Workers spawned 30+ minutes ago. No way to know:
- Current phase (Planning? Implementation? Validation?)
- Percentage complete
- Blockers encountered
- Estimated time remaining

The only feedback is eventual completion or silence.

**Significance:** Can't distinguish "working hard" from "stuck" from "dead."

---

### Finding 6: Task Review Happens After Spawn, Not Before

**Evidence:** `specs-platform-19` (sheet export) was spawned before Dylan could review what it would do. He had to pause it mid-flight after seeing the phase comments:

```
Phase: Design - Architecture: 1) Extend GoogleSheetsClient for write operations...
Phase: Implementation - Starting with GoogleSheetsClient write methods
```

The task was premature - admin-ETL coexistence wasn't even implemented yet.

**Significance:** Spawn is fire-and-forget. No "here's what I'm about to do, proceed?" gate.

---

### Finding 7: Epic Children Auto-Depend on Epic

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

1. **`orch status` cross-project visibility** - Show workers from ALL projects, not just current. This is the primary gap.
2. **`orch workers` command** - Show all active workers with beads ID, phase, runtime, last activity
3. **`bd show` enhancement** - Include session status (active/idle/dead) when in_progress
4. **`orch daemon preview --verbose`** - Show all gates and why each issue passes/fails upfront

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
| NEW | P1 | `orch status` doesn't show cross-project workers - primary visibility gap |
| NEW | P2 | `--parent` creates blocking dependency, should be hierarchy only |
| NEW | P3 | Add `orch workers` command for active worker visibility |
| NEW | P3 | Dead session detection - mark abandoned workers as failed |

---

## References

- Session: specs-platform orchestrator session 2026-01-30
- Events: `~/.orch/events.jsonl`
- Workspaces: `.orch/workspace/sp-feat-*`
