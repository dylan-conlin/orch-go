# Session Handoff - 22 Dec 2025 (night)

## TLDR

Both P1 reliability bugs resolved. Phantom agents fixed (status now accurate). Workers stalling investigated (root cause: Claude API not responding, recommend adding stall detection).

---

## What Shipped

### Commits
| Commit | Description |
|--------|-------------|
| `0ba0104` | Filter phantom agents - require parseable beadsID |
| `5d8c187` | Mark stalled tmux agents as phantom + fix concurrency check |
| `d4b8fc8` | Investigation: workers stall during Build phase |

### Issues Closed
| Issue | Type | Resolution |
|-------|------|------------|
| orch-go-c4fh | bug | Fixed - filter by beadsID, mark stalled tmux as phantom |
| orch-go-d039 | bug | Investigated - stall = busy without message events >5min |

---

## Key Findings

### Phantom Agents (Fixed)
**Root causes:**
1. Tmux service windows (docker, frontend, logs) in `workers-*` sessions counted as agents
2. OpenCode sessions without beadsID passing idle filter
3. Tmux windows with dead OpenCode sessions not marked phantom

**Fix:** Only count windows/sessions with parseable beadsID; mark tmux-only as "stalled"

### Workers Stalling (Investigated)
**Finding:** "Build" is OpenCode's agent mode, not a compile phase. Stall = Claude API request sent but no tokens streaming back.

**Causes (likely order):**
1. Rate limiting (429) - not surfaced to user
2. API hang - request sent, no response
3. Network issues
4. OpenCode internal issues

**Recommendation:** Add stall detection to SSE monitor - alert when `session.status: busy` for >5min without `message.part` events

---

## System State

**Account usage:** 13% (resets in 6d 19h)

**Status:** 0 active, 7 phantom (cross-project agents from skillc/price-watch)

**Ready queue (all P2):**
```
orch-go-xwh    Dashboard UI/UX iteration
orch-go-bdd    Epic: Headless Swarm
orch-go-bdd.3  Daemon concurrency control
orch-go-36b    Dashboard agent visibility
orch-go-vut1   Model flexibility phase 2
```

---

## Quick Start Next Session

```bash
orch status
bd ready

# P2 work available - pick based on priority
orch spawn feature-impl "task" --issue <id>
```
