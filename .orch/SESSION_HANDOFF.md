# Session Handoff - Dec 26, 2025 (Morning)

## Session Focus
Started: "Clear completion backlog - fix verification criteria to stop false positives"
Evolved: Dashboard stability, daemon fixes, cross-project visibility, account management

## What We Accomplished

### 1. Dashboard Fixes (5 issues)
- **Gold border flashing** - Added debounced processing state clear
- **Tooltip hydration error** - Fixed with bits-ui child snippet pattern
- **Active/Complete mismatch** - Phase: Complete agents now properly marked completed
- **Jostling sort** - Skip is_processing in stable sort
- **Load tests** - 13 Playwright tests for 50+ agents

### 2. Daemon Fixes (2 critical)
- **Stale capacity count** - Added `Pool.Reconcile()` that syncs with OpenCode on each poll
- **bd show JSON parsing** - Already fixed in prior commit, daemon needed restart

**Overnight run failed** (0/14 issues) due to these bugs. Now fixed.

### 3. Cross-Project Visibility
- **Architect design complete** - Use OpenCode session.Directory for dynamic project discovery
- **Implementation in progress** (`orch-go-702d`) - Multi-project workspace aggregation

### 4. Account Management Issues Found
- **OAuth scope bug** (`orch-go-h35c` in progress) - `orch account add` missing `user:inference` scope
- **Workaround:** Use `opencode auth login` instead of `orch account add`
- **Auto-switch not implemented** - Created `orch-go-1qwt` for future

### 5. Designs Completed
- **Daemon UI visibility** - File-based status + /api/daemon endpoint (features created)
- **Max subscription arbitrage** - Don't build, violates ToS. Use API credits for colleagues.
- **Snap evaluation** - Complete MVP, ready for visual verification integration

## Decisions Made

| Decision | Reason |
|----------|--------|
| Use snap for simple screenshots | Playwright MCP costs ~5-10k tokens for tool defs |
| Don't build Max proxy | Account sharing violates Anthropic ToS |
| Cross-project uses session.Directory | Dynamic discovery, no static registry needed |

## Agents Still Running

| Agent | Task | Account |
|-------|------|---------|
| `orch-go-h35c` | OAuth scope fix | work |
| `orch-go-702d` | Cross-project visibility | work |

## Current State

```
Open:        53
In Progress: 14
Ready:       52
Closed:      500
```

**Commits today:** 15 (+8277/-337 lines)

## Gaps / Friction Noticed

1. **Account switching broken** - Wrong OAuth scopes. Use `opencode auth login` workaround.
2. **No auto-switch** - Hit rate limit, agent stalled. Manual switch required.
3. **Untracked agents linger** - No clean way to remove idle untracked agents from status.
4. **Dashboard cross-project** - Shows "Waiting for activity" for other project's agents.

## Resume Instructions

```bash
# Check running agents
orch status

# Complete if done
orch complete orch-go-h35c
orch complete orch-go-702d

# Test daemon overnight run
orch daemon run

# Check account status
orch usage
orch account list
```

## Daemon Queue (triage:ready)

High priority features waiting:
- `orch-go-4kjf` - Daemon writes status file
- `orch-go-6su0` - /api/daemon endpoint
- `orch-go-r3op` - Dashboard daemon indicator
- `orch-go-1qwt` - Auto-switch accounts on low capacity
- `orch-go-a85k` - Snap CLI integration for visual verification

## Usage

- **Work account:** 34% weekly, 99% session (active)
- **Personal account:** 16% weekly, 61% session (was rate limited)

## Git State

- All work committed locally on `master`
- 136 uncommitted changes (staged?)
- Check with `git status`
