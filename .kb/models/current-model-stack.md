# Model: Current Model Stack

**Domain:** Agent Spawning / Model Selection
**Last Updated:** 2026-01-22
**Authoritative For:** What models and backends orch-go uses TODAY

---

## Summary (30 seconds)

**Primary:** Claude Max via CLI (`--backend claude`) - unlimited Sonnet/Opus at $200/mo flat
**Fallback:** OpenCode API with Sonnet (`--backend opencode`) - pay-per-token, opt-in only
**Escape Hatch:** Docker (`--backend docker`) - fresh fingerprint for rate limit bypass

This document is the authoritative "current state" reference. Cite THIS document when describing orch-go's model stack, not individual historical decisions.

---

## Current Stack (as of Jan 2026)

| Role | Model/Backend | Cost | When to Use |
|------|---------------|------|-------------|
| **Primary** | Claude CLI + Max subscription | $200/mo flat | Default for all spawns |
| **Fallback** | OpenCode API + Sonnet | Pay-per-token | Explicit `--backend opencode` |
| **Escape Hatch** | Docker + Claude CLI | $200/mo (shared) | Rate limit bypass: `--backend docker` |

### Default Backend

```bash
# Default: uses Claude CLI (Max subscription)
orch spawn feature-impl "task"

# Explicit API path (when needed)
orch spawn --backend opencode feature-impl "task"

# Fresh fingerprint for rate limits
orch spawn --backend docker feature-impl "task"
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

### Use Default (Claude CLI) When:
- Normal agent spawning
- Need Opus quality (`--model opus`)
- Cost predictability matters
- Workflow doesn't require dashboard visibility

### Use OpenCode API (`--backend opencode`) When:
- Need headless operation (no tmux windows)
- Need dashboard visibility
- High concurrency (5+ agents)
- Cost tracking is implemented and monitored

### Use Docker (`--backend docker`) When:
- Hitting device-level rate limits
- Need fresh Statsig fingerprint
- Testing rate limit isolation

---

## Constraints (Why This Stack)

1. **Anthropic fingerprinting blocks Opus via API** - Only accessible through Claude CLI with Max subscription
2. **Pay-per-token costs spiral without visibility** - No cost tracking implemented, led to $402 surprise
3. **Gemini Flash TPM limits (2,000 req/min)** - Tool-heavy agents hit limits, forced switch away
4. **Dashboard visibility only via OpenCode** - Claude CLI spawns don't appear in dashboard

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
