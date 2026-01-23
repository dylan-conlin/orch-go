# Model: Current Model Stack

**Domain:** Agent Spawning / Model Selection
**Last Updated:** 2026-01-23
**Authoritative For:** What models and backends orch-go uses TODAY

---

## Summary (30 seconds)

**Orchestrator:** Claude Code CLI on macOS (Opus) - Dylan's primary Max account
**Workers:** Docker Claude Code (Opus) - second Max account, fresh fingerprint isolation
**Fallback:** OpenCode API with Sonnet - pay-per-token, rarely used

Dylan orchestrates directly from Claude Code on macOS. Workers spawn into Docker containers via `orch spawn --backend docker`, using a second Max account for rate limit isolation.

This document is the authoritative "current state" reference. Cite THIS document when describing orch-go's model stack, not individual historical decisions.

---

## Current Stack (as of Jan 2026)

| Role | Model/Backend | Account | Cost | Notes |
|------|---------------|---------|------|-------|
| **Orchestrator** | Claude Code CLI (macOS) | Max #1 | $200/mo | This conversation - Dylan orchestrates here |
| **Workers** | Docker Claude Code | Max #2 | $200/mo | Fresh fingerprint, rate limit isolation |
| **Fallback** | OpenCode API + Sonnet | API key | Pay-per-token | Rarely used - no Opus access |

### Current Operational Setup

**Why this setup:**
- Orchestrator needs macOS access (launchctl, make, Docker CLI)
- Workers benefit from fingerprint isolation (rate limits are per-device)
- Two Max accounts = $400/mo total, but unlimited Opus for both orchestration and workers
- OpenCode coaching plugin NOT exercised (orchestrator is in Claude Code, not OpenCode)

### Typical Commands

```bash
# Dylan orchestrates from Claude Code (this conversation)
# Workers spawn into Docker:
orch spawn --backend docker feature-impl "task"

# Daemon auto-spawns with docker backend (configured in ~/.orch/config.yaml)
bd create "task" --type task -l triage:ready  # Daemon picks up and spawns docker

# Fallback to native Claude CLI (rare - when Docker is problematic)
orch spawn --backend claude feature-impl "task"

# OpenCode API (rarely used)
orch spawn --backend opencode --model sonnet feature-impl "task"
```

---

## Decision Trail

This stack evolved through several decisions. **Only the most recent decision is current policy.**

| Date | Decision | Status | Key Change |
|------|----------|--------|------------|
| Jan 9, 2026 | `2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` | **Superseded** | Gemini Flash primary (abandoned due to TPM limits → Sonnet API) |
| Jan 18, 2026 | `2026-01-18-max-subscription-primary-spawn-path.md` | **Current** | Claude Max via CLI primary (API costs unsustainable) |

### Why the Stack Changed

1. **Jan 9:** Anthropic blocked OAuth → switched to Gemini Flash (free) + Sonnet API (fallback)
2. **Jan 9-18:** API costs spiraled ($402 in ~2 weeks, $70-80/day) without visibility
3. **Jan 18:** Switched to Claude Max via CLI ($200/mo flat) as primary to control costs

---

## When to Use Each Path

### Use Docker (`--backend docker`) - DEFAULT for Workers:
- All daemon-spawned work (configured default)
- Rate limit isolation via second Max account
- Workers don't need macOS host access
- Fresh Statsig fingerprint per container

### Use Native Claude CLI (`--backend claude`) When:
- Worker needs macOS host access (rare)
- Docker is having issues (memory, crashes)
- Debugging spawn infrastructure itself

### Use OpenCode API (`--backend opencode`) When:
- Need dashboard visibility for specific work
- Testing OpenCode-specific features
- **Note:** No Opus access via API (fingerprinting blocks it)

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
5. **Orchestrator needs macOS host access** - launchctl, make, Docker CLI can't run from Docker sandbox
6. **Docker workers can't spawn other Docker workers** - Sandbox doesn't have Docker installed
7. **OpenCode coaching plugin not exercised** - Orchestrator is in Claude Code, not OpenCode TUI
8. **Two Max accounts = rate limit isolation** - Device-level throttling is per-fingerprint, weekly quota is per-account

---

## Known Friction Points (Jan 2026)

**Docker spawn reliability:**
- Memory: Colima 12GB, containers 6GB (raised Jan 22)
- SIGKILL crashes were OOM - monitoring for recurrence
- Container startup adds ~2-5s overhead

**Daemon issues:**
- Fixed: Daemon now iterates through all candidates when one fails dedup/completion check (Jan 23)
- Open: Status-based spawn dedup to prevent duplicates (orch-go-wq3mz)
- Open: Synthesis completion recognition to prevent false spawns (orch-go-qu8fj)

**Docker ↔ macOS friction (orch-go-m3f8b):**
- Docker workers can't run: launchctl, make (macOS binary), Docker CLI
- Workaround: Copy-paste commands to orchestrator session
- Question: Should we expose limited host commands?

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
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Why Claude Max via CLI is primary

**Superseded Decisions:**
- `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` - Historical: Gemini Flash primary

**Architecture Details:**
- `.kb/models/model-access-spawn-paths.md` - Full triple spawn architecture, constraints, failure modes

**Implementation:**
- `cmd/orch/backend.go` - Backend selection logic
- `CLAUDE.md` lines 87-138 - Triple spawn mode documentation
