# Session Handoff - 2025-12-23 (Night)

## What Happened This Session

### Bugs Fixed
- **orch work defaults to headless** - Was spawning in tmux, now matches `orch spawn` default (commit f8c1707)

### Design Decisions Made
1. **OpenCode model selection is per-message, not per-session** (kn-142954)
   - Intentional design enabling mid-conversation model switching
   - Fix: pass model to message API, not session creation

2. **Servers status abstraction** (orch-go-lb61 → closed)
   - Keep tmux session abstraction for `orch servers`
   - `orch serve` is infrastructure, not project server - separate to dedicated command

3. **Dashboard progressive disclosure** (orch-go-crkh → closed)
   - Use Active/Recent/Archive sections (Active expanded, others collapsed)
   - Problem is presentation, not persistence

### Investigation Completed
**Token limit explosion** (orch-go-4gb9 → closed)
- Root causes: KB context bloat (60k+ tokens), double skill loading (37k tokens)
- Recommendations: ORCH_WORKER=1, KB token limits, pre-spawn warnings

### Issues Closed
| ID | Reason |
|----|--------|
| orch-go-g1cz | Completed (from previous session) |
| orch-go-lb61 | Architect: servers status recommendation |
| orch-go-crkh | Architect: dashboard filtering recommendation |
| orch-go-cyzr | Investigation: model selection root cause found |
| orch-go-iq2h | Failed: token limit, recreated as orch-go-qkbw |
| orch-go-9e15.3 | Completed: orchestrator skill already reflects headless default |
| orch-go-4gb9 | Investigation: token explosion root causes |

### Issues Created
| ID | Priority | Description |
|----|----------|-------------|
| orch-go-kive | P1 | Fix headless spawn model selection |
| orch-go-6tdr | P1 | orch status hangs/crashes (exit 137) |
| orch-go-y222 | P1 | Set ORCH_WORKER=1 in headless spawn |
| orch-go-k0mg | P2 | Separate orch serve from servers command |
| orch-go-iv07 | P2 | Dashboard progressive disclosure UI |
| orch-go-fwka | P2 | Design: source of truth for agent tracking |
| orch-go-qkbw | P2 | Research LLM landscape (recreated with correct skill) |
| orch-go-9k1l | P2 | Pre-spawn token estimation |
| orch-go-fqdt | P2 | Dashboard project filter |
| orch-go-bozf | P2 | KB context token limit |

### Cross-Repo Issues (triage:review)
- orch-go-jgc1: kb extract command (needs kb-cli)
- orch-go-p73c: kb supersede command (needs kb-cli)

## Session Reflection

### What Worked
- Discovery through use - found real bugs by using the system
- Architect agents produced clear recommendations
- Learning captured in the moment (per-message model selection)

### Friction Points
- Dashboard doesn't show what matters (can't tell active vs phantom vs idle)
- Source of truth confusion (`orch status` vs dashboard show different things)
- CLI broken (`orch status` and `orch complete` hanging)
- Token limit with no warning

### Theme
**Observability layer is fragile.** Spawn/daemon/complete work, but "see what's happening" tools have gaps.

## Next Session Priority

**P1 Bugs (blocking):**
1. **orch-go-6tdr** - Fix orch status/complete hanging (CLI unusable)
2. **orch-go-kive** - Fix model selection in headless spawn
3. **orch-go-y222** - Set ORCH_WORKER=1 to prevent skill double-loading
4. **orch-go-4ufh** - Fix orch wait with session ID

## Commands to Start

```bash
# CLI is broken - check via API
curl -s http://127.0.0.1:3348/api/agents | jq '[.[] | select(.status == "active")] | length'

# Check ready queue
bd ready | head -10

# Start with the blocking CLI bug
bd show orch-go-6tdr
```

## Account Status
- work: 2% used (resets in 6d 20h)
