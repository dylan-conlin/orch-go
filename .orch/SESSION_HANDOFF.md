# Session Handoff - Dec 25, 2025

## Session Focus
Fixed dashboard phase display bug (agent cards showing null phases) and added Gemini 3 Flash model support.

## What We Fixed

### 1. Dashboard Phase Display Bug (Critical)
**Symptom:** Agent cards in web UI showed null for phase/task despite beads having comments.

**Root causes discovered:**
1. `ListOpenIssues()` used `bd list -s open -s in_progress -s blocked` - beads CLI only uses the LAST `-s` flag
2. `GetCommentsBatch()` spawned 300+ concurrent `bd` processes, overwhelming system and silently failing

**Fixes (commit `6d75de9`):**
- `pkg/verify/check.go`: Fetch all issues, filter in Go
- `pkg/verify/check.go`: Semaphore limiting to max 10 concurrent `bd` processes

### 2. Gemini 3 Flash Support
Added aliases to `pkg/model/model.go`:
```
flash3    → google/gemini-3-flash-preview
flash-3   → google/gemini-3-flash-preview
flash-3.0 → google/gemini-3-flash-preview
```

Usage: `orch spawn --model flash3 investigation "task"`
For orchestrator: `opencode -m google/gemini-3-flash-preview`

### 3. Beads Daemon Race Condition
Rebuilt `bd` binary with fix from beads repo (`aa401428`):
- `killDuplicateDaemons()` sanity check after acquiring flock
- Prevents process accumulation that caused CPU spikes

**Note:** Fix is on `fix-repo-empty-config` branch, not merged to main.

## Completed Agents

| Issue | Outcome |
|-------|---------|
| `bd-5dup` | ✅ Fixed - daemon race condition (rebuild bd applied) |
| `orch-go-6x36` | Refocused - was architecture investigation, now scoped to model aliases |

## Uncommitted Work

### beads repo (`~/Documents/personal/beads`)
- Branch: `fix-repo-empty-config` (13 commits ahead)
- Daemon fix committed but NOT merged to main
- Should merge or cherry-pick `aa401428` to main

## System State

- `orch serve` running with fixed binary
- `bd daemon` will auto-start with fixed binary on next command
- No active agents
- Work account: 1% usage (resets in ~7 days)

## Ready Work

```bash
bd ready | head -5
```

1. **[P1] `bd-5dup`** - Close it (fix done, just needs issue closure)
2. **[P2] `orch-go-6x36`** - Add DeepSeek/OpenRouter aliases (Gemini done)
3. **[P2] `orch-go-ndgj`** - Research: LLM psychosis article
4. **[P2] `orch-go-xe2j`** - Web-to-markdown MCP for research
5. **[P2] `orch-go-k0mg`** - Separate orch serve from orch servers

## Quick Resume

```bash
# Check status
orch status && bd ready | head -5

# Close daemon race issue
bd close bd-5dup --reason "Fixed: killDuplicateDaemons() sanity check added"

# Continue model aliases (add DeepSeek)
orch spawn feature-impl "add DeepSeek model aliases" --issue orch-go-6x36
```

## Key Learnings

1. **beads `-s` flag** - Only accepts ONE status, not multiple. Use `bd list --json | jq` for filtering.
2. **Concurrent bd processes** - Limit to ~10 max or system gets overwhelmed.
3. **launchd auto-restart** - `com.orch-go.serve` respawns serve; must rebuild before kill or old binary respawns.
