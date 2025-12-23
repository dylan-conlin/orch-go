# Session Handoff - 22 Dec 2025 (late night)

## TLDR

Fixed dashboard to show real-time agent activity. Headless spawn mode now works reliably. Discovered and worked around policy skill loading bug (orchestrator skill was bloating worker context).

---

## What Shipped

### Dashboard Fixes
- Added `idle` status for sessions without recent activity (>10min)
- Filter stale sessions (>6h) from display
- Fixed duplicate key error in Svelte each block
- Real-time agent activity display working

### Headless Spawn Mode
- Flipped default from tmux to headless (orch-go-9e15.1)
- Updated CLAUDE.md and help text
- Model flexibility for headless spawns

### Bug Workaround
- **Policy skill loading bug** (orch-go-v2cz): Orchestrator skill (1224 lines) was loading for ALL sessions including workers, causing token limit errors
- **Workaround**: Moved `~/.claude/skills/policy/orchestrator` to `~/.claude/skills/meta/orchestrator`
- Proper fix needs OpenCode to respect `audience: orchestrator` field

---

## Account State

```
work: 28% weekly, 25% 5-hour (resets 3h)
```

---

## Open Issues

| Issue | Description | Status |
|-------|-------------|--------|
| orch-go-v2cz | Policy skills loading for workers despite audience:orchestrator | Open - needs OpenCode fix |
| orch-go-9e15.3 | Update orchestrator skill for headless default | Blocked - needs orch-knowledge repo |

---

## Cleaned Up

- Killed stale tmux worker sessions (workers-beads-ui-svelte, workers-kb-cli, workers-price-watch, workers-skillc, workers-test-spawn-auto-init)
- Cleaned 14 stale windows from workers-orch-go
- Active agents: 6 (down from 33)

---

## Quick Start Next Session

```bash
# Check status
orch status
orch usage

# Dashboard
~/bin/orch serve &
cd ~/Documents/personal/orch-go/web && bun run dev

# Ready queue
bd ready
```
