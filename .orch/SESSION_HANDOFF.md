# Session Handoff - 22 Dec 2025 (evening)

## TLDR

**Phantom agents bug fixed** (`orch-go-c4fh` closed). Status now correctly shows 0 active when no agents running. Workers stalling investigation spawned (`orch-go-d039`). New issue discovered: session titles don't contain beadsID, so active agents may show as phantom until this is addressed.

---

## What Shipped This Session

### Commits
| Commit | Description |
|--------|-------------|
| `0ba0104` | Filter phantom agents - require parseable beadsID in tmux/OpenCode |
| `5d8c187` | Mark stalled tmux agents as phantom + fix concurrency check |

### Issues Closed
| Issue | Description |
|-------|-------------|
| orch-go-c4fh | Phantom agents in orch status - fixed with beadsID filtering |

---

## Root Cause Analysis: Phantom Agents

**Problem:** `orch status` showed 17 "active" agents when only 6 were real

**Three root causes found:**
1. **Tmux windows without beadsID** - Service windows (docker, frontend, logs) in `workers-*` sessions were counted
2. **OpenCode sessions without beadsID** - Manual sessions and test sessions passing idle filter
3. **Stalled tmux windows** - Tmux window exists but OpenCode session ended (cross-project agents)

**Fix:** 
- Only include windows/sessions with parseable beadsID (orch-spawned)
- Mark tmux windows without active OpenCode session as "phantom/stalled"
- Concurrency limit only counts sessions with beadsID

**Before:** 17 "active", blocked spawning
**After:** 0 active, 7 phantom (correctly classified)

---

## In-Progress Work

| Issue | Status | Agent |
|-------|--------|-------|
| orch-go-d039 | investigating | og-inv-workers-stall-during-22dec |

---

## New Issue Discovered

**Session titles don't contain beadsID** - When spawning, OpenCode auto-generates session title from first message (e.g., "Reading SPAWN_CONTEXT.md"). The beadsID is not embedded in the title, so `orch status` can't match active sessions to their beads issues.

**Impact:** Active agents may appear as phantom in status
**Workaround:** Check tmux directly or wait for workspace `.session_id` file approach

---

## System State

**Account usage:** 13% (resets in 6d 19h)

**Ready queue:**
```
1. orch-go-d039  [P1] Workers stalling at Build phase (IN PROGRESS - agent spawned)
2. orch-go-i1cm  [P2] Clean messaging fix  
3. orch-go-257f  [P2] Add kn init to orch init
4. orch-go-xwh   [P2] Dashboard UI/UX iteration
```

**Phantoms (cross-project, can ignore):**
- 6 from skillc project
- 1 from price-watch project

---

## Quick Start Next Session

```bash
# Check if worker stalling investigation completed
orch status
bd show orch-go-d039  # Check comments for phase

# If complete, review and close
orch complete orch-go-d039

# Otherwise, continue monitoring or check tmux
tmux capture-pane -t workers-orch-go:2 -p | tail -30
```
