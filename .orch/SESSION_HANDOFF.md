# Session Handoff - 2025-12-24 (Late Night)

## Theme: Observability Stack Fixed

Major stabilization session. The core orchestration tools now work reliably.

## What Was Fixed

| Issue | Before | After | Commit |
|-------|--------|-------|--------|
| Headless spawn model | Always sonnet | Correct model | 7598101 |
| orch status speed | 11s | 1.5s | 15603d8 |
| kb context speed | 50s | 72ms | cbf2d36 (kb-cli) |
| orch status detection | 0 active always | Shows running/idle | 561c493 |
| Session titles | No beads ID | Includes [beads-id] | 23650cc |

### Key Changes

**Headless spawn:** Switched from HTTP API to CLI subprocess (`opencode run --format json --model`). OpenCode API ignores model param - this is the workaround.

**orch status:** 
- Batched beads CLI calls (was N sequential calls)
- Parallelized comment fetching
- Uses messages endpoint to detect active vs idle (OpenCode API returns `status: null`)
- OpenCode agents now show "running/idle" instead of "phantom"

**kb context:** Added `--stale` flag to make stale detection opt-in. The stale check was calling `bd show` (~5s each) for every beads ID.

## Current State

### All Spawn Modes Working
```bash
orch spawn SKILL "task"           # Headless (default) ✅
orch spawn --tmux SKILL "task"    # Tmux window ✅
orch spawn --inline SKILL "task"  # Blocking TUI ✅
```

### Observability
- `orch status` - 1.5s, shows running/idle/phantom correctly
- `kb context` - 72ms (use `--stale` for stale warnings, adds ~50s)
- Dashboard at http://localhost:5188 - has known issues (see epic below)

### Binary Install Note
Two `kb` binaries exist: `~/bin/kb` and `~/go/bin/kb`. PATH uses `~/bin/` first. After `go install`, copy to `~/bin/`:
```bash
cp ~/go/bin/kb ~/bin/kb
```
Issue `orch-go-23fh` tracks standardizing this.

## Ready Queue

### Dashboard Bugs (Epic: orch-go-mhec)
Created from playwright audit - 4 ready, 1 needs review:
- `orch-go-mhec.1` - Status filter test expects 4 options, UI has 5
- `orch-go-mhec.2` - Duplicate 'Clear' button selector ambiguity
- `orch-go-mhec.3` - Race-condition tests hardcoded port
- `orch-go-mhec.4` - Agent grid uses index as key (stale data)
- `orch-go-mhec.5` - Svelte 5 runes standardization (triage:review)

### Other P2s
```bash
bd ready | head -10
```
- Template sync tasks (orch-go-hdrc, tsx5, rtym)
- kb extract/supersede commands (jgc1, p73c)
- orch clean messaging bug (i1cm)

## Skill Audit

Audited recent skill changes (Dec 20-23). **No degradation found.**
- SYNTHESIS compliance ~80%+ when required
- "Missing" synthesis files are intentional light-tier spawns
- Progressive disclosure reduced feature-impl 77% without quality loss

## Account Status
- personal: 1% used
- work: 22% used (resets in 6d 14h)

## Commands to Start

```bash
orch status                    # Verify observability works
bd ready | head -10            # Check queue
bd show orch-go-mhec           # Dashboard epic details
```

## Recent Commits
```
3460d69 investigation: audit recent skill changes (Dec 20-23)
561c493 fix: headless agents show running/idle instead of phantom
8e52211 feat: detect active sessions via messages endpoint
23650cc fix: include beads ID in OpenCode session titles
15603d8 perf: optimize orch status from 12s to 1s
7598101 fix: use CLI subprocess for headless spawns
```
