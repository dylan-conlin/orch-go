## Summary (D.E.K.N.)

**Delta:** Add Max 5x ($100/mo) for orchestrator overflow while keeping ChatGPT Pro for dual-vendor resilience. Total: $500/mo.

**Evidence:** Opus 4.5 degraded noticeably before 4.6 release (user-observed, Reddit exodus to GPT confirms). GPT-5.3-Codex performing well as spawn default (18+ agents/day, 5/5 verification tasks in 88 min). Marginlab tracks daily SWE-bench degradation for both providers. Cross-account fingerprint bug (GitHub #3857, 6+ months unfixed) makes 2x Max 20x impractical until CLAUDE_CONFIG_DIR isolation is wired.

**Knowledge:** Model reliability is a vendor risk, not a solved problem. Anthropic published a degradation postmortem (Sep 2025). Having both Claude and Codex as working spawn paths turned an accidental dual-vendor setup into genuine resilience when Opus 4.5 degraded. Single-vendor = single point of failure.

**Next:** Wire up CLAUDE_CONFIG_DIR isolation (orch-go-21511 in progress). Once proven, reassess whether to replace ChatGPT Pro with 2nd Max 20x (same $400/mo, all-Claude) or keep dual-vendor. Subscribe to Marginlab alerts for early warning.

---

# Decision: Subscription Strategy — Dual-Vendor Resilience

**Date:** 2026-02-10
**Status:** Accepted

**Related-To:**
- `.kb/models/multi-model-evaluation-feb2026.md` — Codex production evidence
- `.kb/archive/investigations/2026-01-22-inv-research-claude-max-fingerprinting-bug.md` — Cross-account fingerprint bug
- `.kb/decisions/2026-01-13-cancel-second-claude-max-subscription.md` — Prior decision to go single-account (superseded)
- `.kb/benchmarks/2026-01-28-logout-fix-6-model-comparison.md` — Multi-model quality comparison
- `orch-go-21511` — CLAUDE_CONFIG_DIR isolation implementation (in progress)

---

## Context

**Background:** Running 1x Claude Max 20x ($200/mo) + ChatGPT Pro ($200/mo) = $400/mo total. Claude at 93% weekly usage. GPT-5.3-Codex is default model for all spawns to preserve Claude capacity for orchestration.

**What triggered this decision:** Approaching Claude Max weekly limit mid-week. Considering adding a second Max subscription. During analysis, realized the dual-vendor setup provides critical resilience against model degradation — which Dylan experienced firsthand with Opus 4.5 before 4.6 released.

**Key insight:** Opus 4.5 degraded noticeably in the weeks before Opus 4.6 (Feb 5). Many Claude users migrated to GPT during this period (Reddit reports). Dylan's ChatGPT Pro subscription — originally for experimentation — became an accidental safety net. This resilience is worth preserving intentionally.

**Complication:** Adding a 2nd Claude Max account is blocked by Anthropic's Statsig fingerprinting bug (GitHub #3857) — one account's exhausted limits contaminate the other on the same machine. CLAUDE_CONFIG_DIR isolation (env var pointing to separate config dir) bypasses this without Docker, but isn't wired into orch spawn yet (orch-go-21511).

---

## Options Considered

### Option A: Add Max 5x, Keep ChatGPT Pro ($500/mo) ⭐ CHOSEN
- **Pros:**
  - 25x Claude for orchestation (solves immediate capacity crunch)
  - Codex continues handling spawns (proven, decent quality)
  - Dual-vendor resilience preserved
  - Fingerprint bug irrelevant (5x is orchestrator overflow on same account)
- **Cons:**
  - $100/mo more than status quo
  - Codex still has Claude-dialect friction (completion protocol compliance)

### Option B: Drop ChatGPT Pro, Add 2nd Max 20x ($400/mo)
- **Pros:**
  - Same cost as today
  - All-Claude stack, no dialect mismatch
  - 40x total Claude capacity
- **Cons:**
  - **Single vendor** — if Claude degrades again, no fallback
  - Requires CLAUDE_CONFIG_DIR wired (not done yet)
  - Fingerprint bug is real friction until fixed
  - Loses Codex as proven spawn path

### Option C: Status Quo ($400/mo)
- **Pros:** No change, no cost increase
- **Cons:** Hitting 93% weekly limits, constrains orchestrator usage

### Option D: Add 2nd Max 20x + Keep ChatGPT Pro ($600/mo)
- **Pros:** Maximum capacity + resilience
- **Cons:** $200/mo increase, still blocked by fingerprint bug

---

## Decision

**Chosen:** Option A — Add Max 5x ($100/mo), keep ChatGPT Pro

**Rationale:**

1. **Dual-vendor resilience is a feature, not an accident** — Opus 4.5 degradation proved this. Both providers released new models on the same day (Feb 5). Model reliability is unpredictable.
2. **Immediate need is orchestrator headroom** — The 93% weekly usage is from orchestrator sessions, not spawns. Max 5x adds 25% more Claude capacity specifically for that.
3. **Codex is working** — 18+ agents/day, feature-impl quality is good, main friction is completion protocol (Claude-dialect), not code quality.
4. **Fingerprint bug blocks 2nd Max 20x today** — CLAUDE_CONFIG_DIR isolation in progress but not proven yet. Don't commit $200/mo to an approach that requires unproven infrastructure.
5. **Reassess after CLAUDE_CONFIG_DIR lands** — If isolation works, the option to go 2x Max 20x + drop ChatGPT Pro ($400/mo, all-Claude) remains available.

**Trade-offs accepted:**
- $100/mo cost increase (from $400 to $500)
- Continued Claude-dialect friction with Codex spawns
- Not maximizing Claude capacity (25x vs potential 40x)

---

## Reassessment Triggers

- **CLAUDE_CONFIG_DIR isolation proven** → Reassess Option B (2x Max 20x, all-Claude)
- **Codex quality degrades** → Shift spawns back to Claude, may need 2x Max 20x
- **Claude degradation detected** (Marginlab alert) → Shift more work to Codex
- **New model from either provider** → Run orch eval before changing defaults
- **Quarterly review** → Check spend vs value, adjust tiers

---

## Supersedes

- `.kb/decisions/2026-01-13-cancel-second-claude-max-subscription.md` — That decision assumed lighter consumption. Usage is back up due to active orchestration workflow. The fingerprint bug analysis from that era remains valid.
