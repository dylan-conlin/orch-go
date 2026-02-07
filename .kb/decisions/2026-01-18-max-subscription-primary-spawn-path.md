---
status: active
blocks:
  - keywords:
      - switch to api billing
      - change default backend to opencode
      - remove max subscription
---

## Summary (D.E.K.N.)

**Delta:** Switch from OpenCode API (pay-per-token) to Claude CLI (Max subscription) as default spawn backend.

**Evidence:** $402 API spend in ~2 weeks without cost visibility. Daily burn ramped to $70-80/day. Max subscription at $200/mo flat is 10x cheaper at current usage volume.

**Knowledge:** Headless spawning ergonomics enabled higher spawn volume than anticipated. Cost tracking was identified as needed (Jan 12) but never implemented. Without visibility, spend spiraled.

**Next:** Change default backend to claude. Implement API cost tracking before returning to pay-per-token path. Add second Max subscription if/when hitting limits.

---

# Decision: Max Subscription as Primary Spawn Path

**Date:** 2026-01-18
**Status:** Accepted

**Related-To:**
- `.kb/decisions/2026-01-13-cancel-second-claude-max-subscription.md` - Prior decision that assumed lighter consumption
- `.kb/investigations/2026-01-12-inv-sonnet-cost-tracking-requirements.md` - Identified need for tracking (never implemented)
- `.kb/models/model-access-spawn-paths.md` - Documents dual spawn architecture
- Issue `orch-go-be9z9` - Implementation of default change
- Issue `orch-go-z6gll` - API cost tracking (required before returning to API path)

---

## Context

**What happened:**
- Jan 9: Switched from Gemini Flash (free, TPM limited) to Sonnet API (pay-per-token)
- Jan 12: Investigation identified need for cost tracking - never implemented
- Jan 13: Cancelled second Max subscription assuming "lighter consumption"
- Jan 18: Discovered $402 spent in ~2 weeks, ramping to $70-80/day

**Root cause:** Headless spawning made it frictionless to spawn high volumes. Without cost visibility, no feedback loop existed to moderate usage.

**Trajectory comparison:**
- API path at current rate: ~$2,100-2,400/month
- Max subscription: $200/month flat (unlimited Sonnet + Opus)

---

## Decision

**Chosen:** Claude CLI (Max subscription) as default spawn backend

**Changes:**
1. Default backend: `claude` (was `opencode`)
2. Daemon spawns: Also use Claude CLI (creates tmux windows)
3. API path: Opt-in only via `--backend opencode`
4. Cost tracking: Required before returning to API path

**Trade-offs accepted:**
- Lose headless ergonomics (all spawns create tmux windows)
- Dashboard visibility reduced (Claude CLI sessions not in OpenCode)
- Lower concurrency ceiling (tmux-based vs headless)
- Gain: Predictable $200/mo cost

---

## Structured Uncertainty

**What's tested:**
- ✅ Max subscription provides unlimited Sonnet + Opus via Claude CLI
- ✅ Current API spend is unsustainable ($70-80/day)
- ✅ Claude CLI spawning works for both manual and daemon

**What's untested:**
- ⚠️ Whether daemon-driven Claude CLI spawns scale well (tmux window management)
- ⚠️ When Max subscription limits will be hit with new volume
- ⚠️ Impact on workflow of losing headless ergonomics

**What would change this:**
- API cost tracking implemented + budget alerts working → could return to hybrid model
- Hitting Max subscription limits frequently → add second subscription
- Anthropic changes Max pricing or limits

---

## Consequences

**Positive:**
- Predictable monthly cost ($200 vs $2,100+)
- Access to Opus when needed (same subscription)
- Forces intentional spawning (tmux friction)

**Negative:**
- Lose headless spawning ergonomics
- Dashboard visibility reduced
- May hit subscription limits with heavy usage

**Mitigations:**
- Track spawn volume to anticipate limit issues
- Add second Max subscription if hitting limits
- Implement API cost tracking for future hybrid use
