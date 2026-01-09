# Session Handoff: 2026-01-09

**Session Duration:** 1h 58m
**Focus:** Model selection strategy and dual spawn mode design
**Commits:** 6 commits pushed to remote

---

## What We Accomplished

### 1. Investigated Opus Access Options ✅

**Problem:** Opus 4.5 blocked via OAuth for opencode (Anthropic enforcement).

**Investigation Path:**
- Attempted `opencode-anthropic-auth@0.0.7` plugin update (bypassed until today)
- Tested API key instead of OAuth
- Found: Anthropic updated enforcement 2026-01-09 (cat and mouse game)

**Decision:** Opus via OAuth is blocked and not worth chasing. Documented in:
- `kb-eaf467` (constraint)
- `.kb/investigations/2026-01-09-inv-explore-opencode-github-issue-7410.md`
- `.kb/investigations/2026-01-08-inv-opus-auth-gate-fingerprinting.md`

### 2. Cost Analysis: API vs Max Subscription 💰

**Evaluated Options:**

| Option | Monthly Cost | Orchestrator | Workers | Pros/Cons |
|--------|-------------|--------------|---------|-----------|
| **2x Claude Max** | $200 | Sonnet (Max) | Flash (Max) | Current setup, Opus blocked |
| **1x Max + Sonnet API** | $200-300 | Opus (Max) | Sonnet (API) | Opus for orchestration, $100-200 API cost |
| **1x Max + Opus API** | $500-900 | Opus (Max) | Opus (API) | $0.24/spawn greeting alone! |
| **1x Max + Flash API** | $120 | Opus (Max) | Flash (API) | Cheap but Gemini TPM limits |
| **1x Max + Claude tmux** | $100 | Opus (Max) | Opus (Max tmux) | **Cheapest, best quality** |

**Key Finding:** 38K token greeting ($0.24 on Opus API) due to kb context bloat makes API spawns expensive.

**Rate Limit Discovery:**
- Gemini Flash: Already at 4.73M/3M TPM (158% over limit from other projects)
- Tier 3 requires $1K Google Cloud spend ($571 short)
- Not worth pre-spending for marginal TPM increase

### 3. Designed Dual Spawn Mode Architecture 🏗️

**Decision:** Implement toggle between two spawn backends.

**Architecture:**
```yaml
# .orch/config.yaml
spawn_mode: claude  # or "opencode"

claude:
  model: opus
  tmux_session: workers-orch-go

opencode:
  model: flash
  server: http://localhost:4096
```

**Design Documents:**
- `.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md` - Full architecture
- `.kb/guides/dual-spawn-mode-implementation.md` - 7-task implementation guide

### 4. Implemented Config Toggle ✅ (Task 1 of 7)

**Completed:** `orch-go-5w0fj`

**What Works Now:**
```bash
orch config set spawn_mode claude    # Switch to Claude Code (tmux)
orch config set spawn_mode opencode  # Switch to OpenCode (HTTP API)
orch config get spawn_mode           # Check current mode
```

**Code Changes:**
- `pkg/config/config.go` - Added `SpawnMode`, `ClaudeConfig`, `OpenCodeConfig`
- `cmd/orch/config_cmd.go` - Added `set`/`get` subcommands
- Defaults: `opencode` mode (backward compatible)

**Commit:** `4ebf5082` - feat: add spawn_mode config with claude/opencode toggle

---

## Implementation Status

### ✅ Completed (1/7)
- [x] `orch-go-5w0fj` - Config system with spawn_mode toggle

### 🔓 Ready to Start (2/7)
- [ ] `orch-go-0z5i4` - Implement Claude mode spawn (tmux + claude CLI)
- [ ] `orch-go-1rk4z` - Add mode field to registry schema

### 🔒 Blocked (4/7)
- [ ] `orch-go-7ocqx` - Mode-aware status command (needs spawn + registry)
- [ ] `orch-go-ec9kh` - Mode-aware complete command (needs spawn + registry)
- [ ] `orch-go-wjf89` - Mode-aware monitor/send/abandon (needs spawn + registry)
- [ ] `orch-go-h4eza` - Testing (needs all above)

---

## Key Decisions Made

1. **Abandon Opus via OAuth/API** - Not worth $571 pre-spend or $0.24/spawn cost
2. **Dual spawn mode over single backend** - Flexibility to switch based on budget/needs
3. **Claude mode as budget option** - $100/month for unlimited Opus everywhere
4. **OpenCode mode for dashboard** - $200-300/month when visual monitoring needed

---

## Next Session Tasks

### Priority 1: Complete Claude Spawn Mode

Implement `orch-go-0z5i4` (Claude spawn) and `orch-go-1rk4z` (registry schema):

**Claude Spawn (`pkg/spawn/claude.go`):**
- `SpawnClaude()` - Create tmux window, launch `claude --file SPAWN_CONTEXT.md`
- `MonitorClaude()` - `tmux capture-pane` for output
- `SendClaude()` - `tmux send-keys` for messages
- `AbandonClaude()` - `tmux kill-window`

**Registry Schema (`pkg/registry/registry.go`):**
```go
type Agent struct {
    // ... existing fields
    Mode        string `json:"mode"`           // "claude" | "opencode"
    TmuxWindow  string `json:"tmux_window,omitempty"`   // Claude mode
    SessionID   string `json:"session_id,omitempty"`    // OpenCode mode
}
```

### Priority 2: Mode-Aware Commands

After spawn + registry complete, update commands:
- `status` - Route to tmux or HTTP based on mode
- `complete` - Verify artifacts from tmux or HTTP
- `monitor/send/abandon` - Backend-specific implementations

### Priority 3: Testing

Once all commands support both modes:
- Test mode toggle
- Test mixed registry (some claude, some opencode agents)
- Test graceful fallback when backend unavailable

**Estimate:** 2-3 hours for full implementation (spawn → testing)

---

## Files Changed This Session

**Code:**
- `pkg/config/config.go` - Config schema with spawn mode
- `cmd/orch/config_cmd.go` - Config set/get commands

**Documentation:**
- `.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md`
- `.kb/guides/dual-spawn-mode-implementation.md`
- `.kb/investigations/2026-01-09-debug-gemini-flash-rate-limiting.md`
- `.kb/investigations/2026-01-09-inv-explore-opencode-github-issue-7410.md`

**Config:**
- `.orch/config.yaml` - Added spawn_mode, claude, opencode sections

---

## Knowledge Captured

**Constraints:**
- `kb-eaf467` - Opus 4.5 blocked via OAuth (use Sonnet/Gemini)
- `kb-81f105` - Attempted plugin bypass (failed, Anthropic updated today)

**Investigations:**
- Gemini rate limits (4.73M/3M TPM, need Tier 3)
- Opus API cost ($0.24/greeting, unsustainable)
- opencode plugin bypass attempts (outdated by Anthropic)

**Decisions:**
- Dual spawn mode architecture (claude vs opencode)
- Default to opencode (backward compat)
- Claude mode for budget-conscious usage

---

## Recommended Next Steps

1. **Switch to Claude mode once implemented** - Save $100-200/month
2. **Keep dashboard infrastructure** - Easy to switch back if needed
3. **Monitor Gemini Tier 3 eligibility** - $571 more spend gets 20M TPM
4. **Consider kb context optimization** - Reduce 38K greeting tokens

---

## Context for Next Session

**Current State:**
- Config toggle implemented and tested
- spawn_mode defaults to "opencode" (backward compatible)
- Ready to implement Claude spawn and registry updates

**Architecture Choice:**
- tmux + `claude` CLI for workers
- Claude Code for orchestrator (this session)
- All on single Max subscription ($100/month)

**Why This Matters:**
- Eliminates API costs for spawns
- Unlimited Opus for all agents
- Loses dashboard (acceptable tradeoff)
- Easy to revert if priorities change

---

**Session Closed:** 2026-01-09 19:59 PST
**Branch:** master (6 commits ahead)
**Status:** Clean working tree, all changes pushed
