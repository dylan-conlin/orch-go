# Model: Current Model Stack

**Domain:** Agent Spawning / Model Selection
**Last Updated:** 2026-01-26
**Authoritative For:** What models and backends orch-go uses TODAY

---

## Summary (30 seconds)

**Orchestrator:** Claude Code CLI on macOS (Opus) - Dylan's primary Max account
**Workers:** Claude Code CLI on macOS (Opus) - same Max account (original "escape hatch")
**Fallback:** OpenCode API with Sonnet - pay-per-token, rarely used

Dylan orchestrates directly from Claude Code on macOS. Workers also spawn via native Claude Code CLI using `orch spawn --backend claude`. The Docker backend is disabled (second Max account cancelled).

**Under investigation:**
- `CLAUDE_CONFIG_DIR` fingerprint isolation as simpler alternative to Docker (`orch-go-20922`)
- OpenCode stealth mode for Max OAuth access (`orch-go-20925`) - could restore dashboard visibility + Max subscription from spawned agents

**New development (Jan 26):** OAuth stealth mode IS viable - pi-ai demonstrates stable Max subscription access by mimicking Claude Code's identity markers. See `2026-01-26-claude-max-oauth-stealth-mode-viable.md`.

This document is the authoritative "current state" reference. Cite THIS document when describing orch-go's model stack, not individual historical decisions.

---

## Current Stack (as of Jan 26, 2026)

| Role | Model/Backend | Account | Cost | Notes |
|------|---------------|---------|------|-------|
| **Orchestrator** | Claude Code CLI (macOS) | Max #1 | $200/mo | This conversation - Dylan orchestrates here |
| **Workers** | Claude Code CLI (macOS) | Max #1 | (same) | Native CLI spawn, shared account |
| **Fallback** | OpenCode API + Sonnet | API key | Pay-per-token | Rarely used - no Opus access |

### Current Operational Setup

**Why this setup:**
- Orchestrator needs macOS access (launchctl, make, Docker CLI)
- Second Max account was cancelled → Docker backend no longer viable
- Workers share single Max account ($200/mo total)
- OpenCode coaching plugin NOT exercised (orchestrator is in Claude Code, not OpenCode)

**Constraint:** All workers share same fingerprint (statsig stable_id) → subject to request-rate throttling when concurrent.

### Typical Commands

```bash
# Dylan orchestrates from Claude Code (this conversation)
# Workers spawn via native Claude CLI:
orch spawn --backend claude feature-impl "task"

# Daemon auto-spawns with claude backend (configured in ~/.orch/config.yaml)
bd create "task" --type task -l triage:ready  # Daemon picks up and spawns

# OpenCode API (rarely used)
orch spawn --backend opencode --model sonnet feature-impl "task"
```

### Potential Future: CLAUDE_CONFIG_DIR Isolation

Investigation `orch-go-20910` found that fingerprint isolation doesn't require Docker - simply setting `CLAUDE_CONFIG_DIR` to a fresh directory creates new statsig identity. This is how sneakpeek achieves variant isolation.

**Testing in `orch-go-20922`:** If successful, could add `--backend config-dir` that:
- Creates fresh `~/.claude-spawn-{beads-id}` per spawn
- No Docker overhead (~2-5s container startup eliminated)
- Request-rate isolation without second Max account

### Potential Future: OpenCode Stealth Mode

Investigation `2026-01-26-inv-analyze-pi-ai-anthropic-oauth.md` found that OAuth access IS viable by mimicking Claude Code's identity markers. pi-ai has been doing this stably for months.

**Implementation in `orch-go-20925`:** Modify Dylan's OpenCode fork to:
- Detect OAuth tokens (`sk-ant-oat` prefix) and activate stealth mode
- Set headers: `user-agent: claude-cli/{version}`, `x-app: cli`, etc.
- Include Claude Code identity system prompt
- Optional: tool name normalization to PascalCase

**Benefits if successful:**
- Dashboard visibility restored (OpenCode spawns visible again)
- Max subscription usable from spawned agents
- Leverages existing orch-go/OpenCode integration
- Could become default backend again, displacing native CLI

---

## Decision Trail

This stack evolved through several decisions. **Only the most recent decision is current policy.**

| Date | Decision | Status | Key Change |
|------|----------|--------|------------|
| Jan 9, 2026 | `2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` | **Superseded** | Gemini Flash primary (abandoned due to TPM limits → Sonnet API) |
| Jan 18, 2026 | `2026-01-18-max-subscription-primary-spawn-path.md` | **Superseded** | Claude Max via CLI primary, Docker workers with second Max account |
| Jan 26, 2026 | `2026-01-26-claude-max-oauth-stealth-mode-viable.md` | **Active** | OAuth stealth mode viable; OpenCode can use Max subscriptions |
| Jan 26, 2026 | (stack change - no formal decision yet) | **Current** | Single Max account, native CLI workers, Docker disabled |

### Why the Stack Changed

1. **Jan 9:** Anthropic blocked OAuth → switched to Gemini Flash (free) + Sonnet API (fallback)
2. **Jan 9-18:** API costs spiraled ($402 in ~2 weeks, $70-80/day) without visibility
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
