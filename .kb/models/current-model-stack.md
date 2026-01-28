# Model: Current Model Stack

**Domain:** Agent Spawning / Model Selection
**Last Updated:** 2026-01-28
**Authoritative For:** What models and backends orch-go uses TODAY

---

## Summary (30 seconds)

**Orchestrator:** Claude Code CLI on macOS (Opus 4.5) - Dylan's primary Max account
**Workers:** OpenCode API with OAuth stealth mode (Opus/Sonnet) - same Max account, dashboard visibility
**Fallback:** Claude Code CLI (native) - for infrastructure work or when OpenCode server is unstable

Dylan orchestrates directly from Claude Code on macOS. Workers spawn via OpenCode API using OAuth stealth mode (implemented Jan 28). This restores dashboard visibility while using the Max subscription.

**Stealth mode implemented (Jan 28):** Full pi-ai parity achieved - see `2026-01-26-claude-max-oauth-stealth-mode-viable.md`. Commit `77e60ac7e` in OpenCode fork.

**Still under investigation:**
- `CLAUDE_CONFIG_DIR` fingerprint isolation for request-rate isolation (`orch-go-20922`)

This document is the authoritative "current state" reference. Cite THIS document when describing orch-go's model stack, not individual historical decisions.

---

## Current Stack (as of Jan 28, 2026)

| Role | Model/Backend | Account | Cost | Notes |
|------|---------------|---------|------|-------|
| **Orchestrator** | Claude Code CLI (macOS) | Max #1 | $200/mo | This conversation - Dylan orchestrates here |
| **Workers** | OpenCode API + OAuth stealth | Max #1 | (same) | Dashboard visibility, Opus/Sonnet via OAuth |
| **Fallback** | Claude Code CLI (native) | Max #1 | (same) | Infrastructure work, escape hatch |

### Current Operational Setup

**Why this setup:**
- Orchestrator needs macOS access (launchctl, make, Docker CLI)
- Workers use OpenCode API with OAuth stealth mode for dashboard visibility
- Single Max account ($200/mo total) shared across orchestrator and workers
- OpenCode server must be started WITHOUT `ANTHROPIC_API_KEY` to use OAuth

**Constraint:** All workers share same fingerprint (statsig stable_id) → subject to request-rate throttling when concurrent.

### Typical Commands

```bash
# Dylan orchestrates from Claude Code (this conversation)

# Workers spawn via OpenCode API with OAuth (default)
orch spawn --backend opencode --model anthropic/claude-opus-4-5-20251101 feature-impl "task"

# Or let daemon auto-spawn (uses opencode backend from config)
bd create "task" --type task -l triage:ready

# Native CLI fallback for infrastructure work
orch spawn --backend claude --tmux feature-impl "fix opencode server"
```

### Server Setup for OAuth

**Critical:** The OpenCode server must be started without `ANTHROPIC_API_KEY`:

```bash
# Start dashboard services with OAuth
ANTHROPIC_API_KEY="" orch-dashboard start

# Or manually restart OpenCode server
pkill -f "opencode serve"
ANTHROPIC_API_KEY="" opencode serve --port 4096
```

If `ANTHROPIC_API_KEY` is set, OpenCode uses that instead of OAuth tokens.

### OpenCode Stealth Mode (Implemented Jan 28)

Full pi-ai parity achieved in commit `77e60ac7e`:
- User-Agent: `claude-cli/2.1.15 (external, cli)`
- System prompt: `"You are Claude Code, Anthropic's official CLI for Claude."`
- Headers: `x-app: cli`, `anthropic-dangerous-direct-browser-access: true`, `anthropic-beta: claude-code-20250219,oauth-2025-04-20,...`
- SDK: Uses `authToken` instead of `apiKey` for OAuth tokens

**Verified working:**
- Sonnet 4.5 via OAuth ✅
- Opus 4.5 via OAuth ✅

### Potential Future: CLAUDE_CONFIG_DIR Isolation

Investigation `orch-go-20910` found that fingerprint isolation doesn't require Docker. Testing in `orch-go-20922` to add `--backend config-dir` for request-rate isolation without Docker overhead.

---

## Decision Trail

This stack evolved through several decisions. **Only the most recent decision is current policy.**

| Date | Decision | Status | Key Change |
|------|----------|--------|------------|
| Jan 9, 2026 | `2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` | **Superseded** | Gemini Flash primary (abandoned due to TPM limits → Sonnet API) |
| Jan 18, 2026 | `2026-01-18-max-subscription-primary-spawn-path.md` | **Superseded** | Claude Max via CLI primary, Docker workers with second Max account |
| Jan 26, 2026 | `2026-01-26-claude-max-oauth-stealth-mode-viable.md` | **Active** | OAuth stealth mode viable; OpenCode can use Max subscriptions |
| Jan 28, 2026 | (stack change) | **Current** | OpenCode with OAuth stealth as primary worker backend |

### Why the Stack Changed

1. **Jan 9:** Anthropic blocked OAuth → switched to Gemini Flash (free) + Sonnet API (fallback)
2. **Jan 9-18:** API costs spiraled ($402 in ~2 weeks, $70-80/day) without visibility
3. **Jan 18:** Switched to Claude Max via CLI ($200/mo flat) as primary
4. **Jan 26:** Discovered pi-ai's stealth mode approach - mimics Claude Code identity
5. **Jan 28:** Implemented full pi-ai parity in OpenCode fork → OAuth works again
3. **Jan 18:** Switched to Claude Max via CLI ($200/mo flat) as primary, Docker workers with second Max account
4. **Jan 26:** Second Max account cancelled → Docker backend disabled, workers use native CLI on single account

---

## When to Use Each Path

### Use Native Claude CLI (`--backend claude`) - DEFAULT for Workers:
- All daemon-spawned work (configured default)
- Workers have macOS host access (can run make, launchctl, etc.)
- No container overhead

### Use OpenCode API (`--backend opencode`) When:
- Need dashboard visibility for specific work
- Testing OpenCode-specific features
- **Note:** No Opus access via API (fingerprinting blocks it)

### Use Docker (`--backend docker`) - DISABLED:
- Second Max account cancelled
- Docker backend no longer operational
- Kept in codebase for potential future use with CLAUDE_CONFIG_DIR approach

### Orchestrator (Claude Code on macOS):
- Always native macOS Claude Code (this conversation)
- Has access to: launchctl, make, Docker CLI, tmux
- Uses Max account #1
- NOT in OpenCode, so coaching plugin doesn't apply

---

## Constraints (Why This Stack)

1. **Anthropic fingerprinting blocks Opus via API** - Only accessible through Claude CLI with Max subscription
2. **Pay-per-token costs spiral without visibility** - No cost tracking implemented, led to $402 surprise
3. **Gemini Flash TPM limits (2,000 req/min)** - Tool-heavy agents hit limits, forced switch away
4. **Dashboard visibility only via OpenCode** - Claude CLI spawns don't appear in dashboard
5. **Orchestrator needs macOS host access** - launchctl, make, Docker CLI can't run from containers
6. **Single Max account** - Workers share orchestrator's fingerprint, subject to request-rate throttling
7. **OpenCode coaching plugin not exercised** - Orchestrator is in Claude Code, not OpenCode TUI

---

## Known Friction Points (Jan 2026)

**Single-account rate limiting:**
- All workers share same statsig fingerprint
- Concurrent spawns may hit request-rate throttling
- Weekly quota is account-level (unaffected)
- **Potential fix:** `CLAUDE_CONFIG_DIR` isolation (`orch-go-20922`)

**Daemon issues:**
- Fixed: Daemon now iterates through all candidates when one fails dedup/completion check (Jan 23)
- Open: Status-based spawn dedup to prevent duplicates (orch-go-wq3mz)
- Open: Synthesis completion recognition to prevent false spawns (orch-go-qu8fj)

**No dashboard visibility for workers:**
- Claude CLI spawns don't appear in orch dashboard
- Status tracking relies on beads comments + workspace artifacts
- Native swarm mode (sneakpeek) could provide alternative visibility

---

## Update Instructions

When the model stack changes:

1. **Create new decision** in `.kb/decisions/` with new policy
2. **Mark old decisions as superseded** - Add `Superseded-By:` field and note at top
3. **Update THIS document** - Change current stack table and decision trail
4. **Update model-access-spawn-paths.md** - If architectural implications change

**Trigger for update:** Any change to default backend, primary model, or cost model.

---

## References

**Current Policy:**
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Why Claude Max via CLI is primary (partially superseded - Docker disabled)

**Superseded Decisions:**
- `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` - Historical: Gemini Flash primary

**Architecture Details:**
- `.kb/models/model-access-spawn-paths.md` - Full triple spawn architecture, constraints, failure modes

**Implementation:**
- `cmd/orch/backend.go` - Backend selection logic
- `CLAUDE.md` lines 87-138 - Triple spawn mode documentation

**Under Investigation:**
- `orch-go-20922` - Test CLAUDE_CONFIG_DIR fingerprint isolation for worker spawns
- `.kb/investigations/2026-01-25-inv-investigate-claude-code-native-swarm.md` - Native swarm internals, fingerprint isolation via CLAUDE_CONFIG_DIR
- `orch-go-20903` - Strategic question: Should we adopt native Claude orchestration over orch-ecosystem?
