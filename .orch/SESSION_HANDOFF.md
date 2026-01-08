# Session Handoff - 2026-01-08 (Late Morning)

## Session Focus
Synthesis of Jan 8 investigations + fixing observation infrastructure bugs.

## Key Accomplishments

| Item | Status | Notes |
|------|--------|-------|
| **Decision: Observation Infrastructure Principle** | Created | `.kb/decisions/2026-01-08-observation-infrastructure-principle.md` |
| **Fix: SSE clearing current_activity** | Fixed | Frontend was erasing activity on idle transition, showing "Starting up..." |
| **Fix: Beads RPC failure** | Fixed | Stray `orch-go.db` file caused daemon health check failure |
| **Completed agents** | 2 closed | `orch-go-tuofe` (debugging), `orch-go-18t3i` (epic child inference) |

## The Observation Principle

Synthesized 11 investigations from today into a key principle:

> **"If the system can't observe it, the system can't manage it."**

Observation infrastructure is load-bearing. Gaps create false signals (agents appear "dead" when actually complete).

**Five gaps identified and addressed:**
1. Events not emitted (bd close bypass) - fixed earlier
2. Events double-counted (stats) - fixed earlier  
3. State not surfaced (dead/stalled) - dead restored, stalled designed
4. Progress signals missing (activity) - **fixed this session**
5. RPC failures silent (beads) - **diagnosed this session**

## Bugs Fixed This Session

### 1. SSE Clearing Activity Bug
**Symptom:** Agents showed "Starting up..." even after completing work
**Root cause:** `handleSSEEvent` for `session.status` idle was setting `current_activity: undefined`
**Fix:** Keep `current_activity` when going idle, only clear `is_processing`
**File:** `web/src/lib/stores/agents.ts:698-708`

### 2. Beads RPC Silent Failure
**Symptom:** Agent `orch-go-tuofe` showed as "dead" despite having Phase: Complete in beads
**Root cause:** Empty `orch-go.db` file in `.beads/` caused daemon to fail health check with "multiple database files found"
**Fix:** Remove stray database file
**Constraint recorded:** `kn-e3e9c6` - beads daemon fails silently with multiple .db files

## Files Changed This Session

- `.kb/decisions/2026-01-08-observation-infrastructure-principle.md` (created)
- `web/src/lib/stores/agents.ts` - Don't clear activity on idle
- `.kn/entries.jsonl` - Constraint about beads multi-db failure

## Git Status
- All changes committed and pushed to origin/master
- Working tree clean (except build artifacts)

## Outstanding Work

| Item | Status | Notes |
|------|--------|-------|
| Stalled agent detection | Designed | 15-min threshold, pending implementation |
| Epic child inference | Merged | Agent completed this session |
| Dashboard performance epic | Open | `orch-go-8s2kl` |

## Resume Commands
```bash
cd ~/Documents/personal/orch-go
orch status
bd ready | head -5
```

## Key Learning
When dashboard shows unexpected state (dead vs complete), trace the observation pipeline:
1. Is beads daemon healthy? (`bd daemon health`)
2. Are comments being fetched? (test RPC connection)
3. Is phase parsing working? (check commentsMap population)
4. Is Priority Cascade running? (dead < phaseComplete < issueClosed)

Silent failures in any step cause the cascade to fall through to session heartbeat status.
