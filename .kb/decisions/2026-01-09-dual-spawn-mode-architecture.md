# Decision: Dual Spawn Mode Architecture

**Date:** 2026-01-09
**Status:** Implemented (Extended Jan 20, 2026)
**Context:** Cost and rate limit constraints with Gemini API

**Note (Jan 20, 2026):** This decision has been extended to include a third backend: Docker. Docker provides Statsig fingerprint isolation as a "double escape hatch" for rate limit scenarios. See `.kb/investigations/2026-01-20-inv-design-claude-docker-backend-integration.md` for design rationale.

## Problem

Current architecture uses opencode HTTP API for spawns, requiring paid API access:
- Gemini Flash: TPM rate limits (4.73M/3M - already over)
- Sonnet: ~$100-200/month API costs
- Opus: Blocked via OAuth, $400-800/month via API

Single Claude Max subscription ($100/month) provides unlimited Opus but only works with `claude` CLI directly, not via opencode HTTP API.

## Decision

**Implement dual spawn mode system** allowing orchestrator to toggle between:
1. **Claude Code mode** - tmux + `claude` CLI (Max subscription, unlimited Opus)
2. **OpenCode mode** - HTTP API + dashboard (paid API, Flash/Sonnet/Opus)

## Architecture

### Config-Driven Mode Selection

```yaml
# .orch/config.yaml
spawn_mode: claude  # "claude" | "opencode"

claude:
  model: opus
  tmux_session: workers-orch-go

opencode:
  model: flash
  server: http://localhost:4096
```

### Mode Behaviors

| Command | Claude Mode | OpenCode Mode |
|---------|-------------|---------------|
| `spawn` | Create tmux window, launch `claude` | HTTP API session |
| `status` | Parse tmux windows | Query HTTP API |
| `complete` | Read tmux pane, verify artifacts | Query HTTP session |
| `monitor` | `tmux capture-pane` | SSE stream |
| `send` | `tmux send-keys` | HTTP POST |
| `abandon` | Kill tmux window | Delete session |

### Registry Schema

```json
{
  "agents": [{
    "id": "og-inv-task-123",
    "mode": "claude",           // Track backend
    "tmux_window": "inv-task",  // Claude mode field
    "session_id": null          // OpenCode mode field
  }]
}
```

### Easy Switching

```bash
# Switch modes globally
orch config set spawn_mode claude
orch config set spawn_mode opencode

# Override per-spawn
orch spawn --mode claude investigation "task"
orch spawn --mode opencode investigation "task"
```

## Tradeoffs

### Claude Mode

**Pros:**
- ✅ Unlimited Opus ($100/month flat)
- ✅ No rate limits
- ✅ Best quality for all work
- ✅ No API costs

**Cons:**
- ❌ No web dashboard
- ❌ Manual tmux window management
- ❌ Harder to spawn multiple agents in parallel
- ❌ No programmatic SSE monitoring

### OpenCode Mode

**Pros:**
- ✅ Web dashboard visibility
- ✅ Easy parallel spawning
- ✅ SSE event monitoring
- ✅ HTTP API automation

**Cons:**
- ❌ $100-300/month API costs
- ❌ Rate limits (TPM/RPM)
- ❌ Lower quality models (Flash/Sonnet)
- ❌ Opus blocked or very expensive

## Implementation Scope

### Core Changes

1. **Config system** (`pkg/config/`)
   - Add `spawn_mode` field
   - Mode-specific settings
   - Toggle command

2. **Spawn routing** (`pkg/spawn/`)
   - `claude.go` - tmux spawn implementation
   - `opencode.go` - existing HTTP spawn (rename from `context.go`)
   - Router based on mode

3. **Registry** (`pkg/registry/`)
   - Add `mode` field to agent schema
   - Track backend per agent

4. **Commands** (`cmd/orch/`)
   - Mode-aware routing in all commands
   - Fallback behavior for missing backend

5. **Status/Monitor** (`pkg/opencode/`, `pkg/tmux/`)
   - Unified status interface
   - Backend-specific implementations

### Compatibility

- **Default mode:** `opencode` (backward compatible)
- **Mixed agents:** Registry tracks per-agent mode
- **Graceful fallback:** Commands detect unavailable backend

## Success Criteria

- ✅ Can toggle between modes via config
- ✅ Existing opencode workflows still work
- ✅ Claude mode spawns work with tmux + `claude` CLI
- ✅ Status/complete/monitor work with both modes
- ✅ Can switch modes mid-project without breaking registry

## Cost Impact

| Mode | Monthly Cost | Use When |
|------|--------------|----------|
| Claude | $100 | Budget-constrained, Opus quality needed |
| OpenCode | $200-300 | Need dashboard, parallel spawning, or specific models |

## Alternatives Considered

1. **Claude-only (no opencode option)** - Rejected: Loses dashboard permanently
2. **OpenCode-only (pay for Opus API)** - Rejected: $400-800/month unsustainable
3. **Separate orch-claude and orch-opencode tools** - Rejected: Split ecosystem

## Related

- Investigation: `.kb/investigations/2026-01-09-debug-gemini-flash-rate-limiting.md`
- Investigation: `.kb/investigations/2026-01-20-inv-design-claude-docker-backend-integration.md` (Docker extension)
- Constraint: `kb-eaf467` (Opus blocked via OAuth)
- Decision: `kb-81f105` (Attempted plugin bypass)

---

## Extension: Docker Backend (Jan 20, 2026)

This decision has been extended to support a **third backend: Docker**.

### New Backend

| Mode | Purpose | When to Use |
|------|---------|-------------|
| Docker | Fresh Statsig fingerprint | Rate limit escape hatch |

### Architecture

```bash
orch spawn --backend docker investigation "task"
    ↓
Host tmux window created (same as claude mode)
    ↓
docker run claude-code-mcp ...
    ↓
Fresh fingerprint per spawn via ~/.claude-docker/
```

### Key Design Decisions

1. **Host tmux, not nested tmux** - Docker container runs Claude directly, observed via host tmux
2. **No dashboard visibility** - Escape hatch philosophy: trade convenience for independence
3. **One container per spawn** - Simplicity over optimization (rare usage expected)
4. **Explicit opt-in only** - `--backend docker` required (no auto-selection)

### Implementation References

- `pkg/spawn/docker.go` - SpawnDocker function
- `cmd/orch/backend.go` - Backend selection with docker support
- `~/.claude/docker-workaround/` - Docker image source
