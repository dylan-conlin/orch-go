# Model: Orchestration Cost Economics

**Domain:** Agent Orchestration / Model Selection / Cost Management
**Last Updated:** 2026-02-25
**Synthesized From:** 15 investigations, 25+ kb quick entries, decisions spanning Nov 2025 - Feb 2026, probes 2026-02-20 (stale references audit) and 2026-02-24 (account distribution design)

---

## Summary (30 seconds)

Agent orchestration cost is driven by three factors: **model pricing** (10-100x variance), **access restrictions** (fingerprinting, OAuth blocking), and **visibility** (lack of tracking caused $402 surprise spend). The Jan 2026 cost crisis revealed that headless spawning without cost visibility leads to runaway spend. As of Feb 2026, the **default spawn path is Claude backend + Max subscription** (Sonnet via Claude CLI), making the $200/mo flat rate the primary economic path — not just the escape hatch. The provider ecosystem spans 4 providers (Anthropic, Google, OpenAI/Codex, DeepSeek) with centralized config resolution (`pkg/spawn/resolve.go`) and model-aware backend routing. **Per-spawn account distribution** (Feb 20-21) enables capacity-aware routing across multiple Max accounts via `CLAUDE_CONFIG_DIR` injection, with the `Account` field tracked as a first-class resolved setting with provenance.

---

## The Cost Problem Timeline

| Date         | Event                                               | Impact                                             |
|--------------|-----------------------------------------------------|----------------------------------------------------|
| Dec 2025     | Gemini Flash (free) as default                      | $0/mo, hit 2,000 req/min TPM limit                 |
| Jan 9, 2026  | Switch to Sonnet API (pay-per-token)                | Unknown cost, no tracking                          |
| Jan 9, 2026  | Anthropic blocks OAuth for third-party tools        | Can't use Max subscription via OpenCode            |
| Jan 12, 2026 | Investigation identifies cost tracking need         | Never implemented                                  |
| Jan 13, 2026 | Cancel second Max subscription                      | $2,400/year savings                                |
| Jan 18, 2026 | Discover $402 spent in ~2 weeks                     | $70-80/day burn rate                               |
| Jan 18, 2026 | Switch back to Max subscription default             | $200/mo flat                                       |
| Jan 19, 2026 | Test confirms DeepSeek V3 function calling works    | New viable option                                  |
| Feb 2026     | Flash models banned for agent work                  | `validateModel()` gate in resolve.go               |
| Feb 2026     | OpenAI/Codex added as first-class provider          | 12 model aliases, OpenCode backend                 |
| Feb 2026     | Centralized `ResolvedSpawnSettings` with provenance | Multi-file resolver replaces monolithic backend.go |
| Feb 19, 2026 | Anthropic bans subscription OAuth in third-party tools | OpenCode + Anthropic = dead path without override |
| Feb 20, 2026 | Default backend changed to Claude (Sonnet default)  | Max subscription is now primary path (commit 21b543524) |
| Feb 20, 2026 | Account distribution Phase 1: schema + CLI + env injection | Per-spawn account selection via `CLAUDE_CONFIG_DIR` |
| Feb 21, 2026 | Account distribution Phase 2: capacity cache + heuristic routing | Automatic account routing based on remaining capacity |

---

## Model Pricing Comparison

| Model            | Input $/MTok | Output $/MTok | Access Method     | Notes                                              |
|------------------|--------------|---------------|-------------------|----------------------------------------------------|
| **DeepSeek V3**  | $0.25        | $0.38         | OpenCode API      | **10-65x cheaper, function calling works**         |
| DeepSeek R1      | $0.45        | $2.15         | OpenCode API      | Reasoning model, function calling experimental     |
| ~~Gemini Flash~~ | ~$0.10-0.30  | Variable      | ~~API/Free tier~~ | **BANNED for agent work** (`validateModel()` gate) |
| Claude Haiku     | $1.00        | $5.00         | Claude CLI (Max)  | Fast, lightweight                                  |
| Claude Sonnet    | $3.00        | $15.00        | Claude CLI (Max)  | **Default model** (claude-sonnet-4-5-20250929)     |
| Claude Opus      | $5.00        | $25.00        | Claude CLI (Max)  | Highest quality, requires Claude backend           |
| **Claude Max**   | $200/mo flat | -             | Claude CLI only   | Unlimited Sonnet + Opus. **Now the default path.** |
| OpenAI GPT-5     | Variable     | Variable      | OpenCode API      | First-class provider (12 aliases including Codex)  |

### Cost Equivalence Points

At **$200/mo Max subscription breakeven:**
- DeepSeek V3: ~317M input tokens OR ~526M output tokens
- Sonnet: ~67M input tokens OR ~13M output tokens
- Opus: ~40M input tokens OR ~8M output tokens

**Key insight:** At Jan 18 burn rate ($70-80/day with Sonnet), Max subscription is 10x cheaper. But DeepSeek V3 could be even cheaper if function calling holds up.

---

## Internal Credit System (Reverse Engineered Jan 2026)

**Source:** she-llac.com/claude-limits - obtained via reverse engineering SSE responses and Stern-Brocot tree algorithm on float precision artifacts. Data may become stale if Anthropic patches the leak.

### Credit Formula

Anthropic uses an internal credit system to meter subscription usage:

```
credits_used = ceil(input_tokens × input_rate + output_tokens × output_rate)
```

**Per-model credit rates:**

| Model  | Input Rate    | Output Rate   |
|--------|---------------|---------------|
| Haiku  | 2/15 (0.133)  | 10/15 (0.667) |
| Sonnet | 6/15 (0.4)    | 30/15 (2.0)   |
| Opus   | 10/15 (0.667) | 50/15 (3.333) |

### Actual Limits vs Marketing Claims

| Plan           | Marketing | 5-Hour Session | Weekly Limit | Actual Ratio                   |
|----------------|-----------|----------------|--------------|--------------------------------|
| Pro ($20)      | 1×        | 550,000        | 5,000,000    | baseline                       |
| Max 5× ($100)  | 5×        | 3,300,000      | 41,666,700   | **6× session, 8.33× weekly**   |
| Max 20× ($200) | 20×       | 11,000,000     | 83,333,300   | **20× session, 16.67× weekly** |

**Key insight:** Max 5× actually **overdelivers** (6× session, 8.33× weekly) while Max 20× **underdelivers** on weekly (16.67× instead of 20×). The 5× plan may offer better value per dollar.

### Cache Pricing: Subscription Advantage

| Operation            | API Cost          | Subscription Cost   |
|----------------------|-------------------|---------------------|
| Cache read           | 10% of input rate | **FREE**            |
| Cache write (5-min)  | 1.25× input rate  | Regular input price |
| Cache write (1-hour) | 2× input rate     | Regular input price |

**This is massive for agentic work:** Our tool-heavy orchestration (dozens of tool calls per agent turn) generates heavy cache hits. Free cache reads mean warm-cache scenarios deliver up to **36× value over API**.

### Value Multipliers (API Equivalent)

| Plan    | Monthly Cost | API Equivalent Value | Multiplier |
|---------|--------------|----------------------|------------|
| Pro     | $20          | $163                 | 8.1×       |
| Max 5×  | $100         | $1,354               | 13.5×      |
| Max 20× | $200         | $2,708               | 13.5×      |

**Validation:** Our Jan 18 discovery showed $70-80/day API burn rate (~$2,100-2,400/mo). At 13.5× value multiplier, our $200/mo Max subscription delivers ~$2,708 equivalent API value - right at our burn rate. External validation of cost decision.

---

## Access Restrictions

### Anthropic Fingerprinting (Jan 2026)

**What's blocked:**
- Third-party tools (OpenCode, Cursor) using Max subscription OAuth
- Opus 4.5 via any non-Claude-Code path

**How it works:**
- TLS fingerprinting (JA3)
- HTTP/2 frame characteristics
- Tool name patterns (lowercase vs PascalCase+_tool)
- OAuth scope requirements

**Evidence:** All community workarounds failed within 6 hours. Anthropic iterates faster than bypasses can stabilize.

**Source:** `.kb/investigations/archived/2026-01-08-inv-opus-auth-gate-fingerprinting.md`, `.kb/investigations/archived/2026-01-09-inv-anthropic-oauth-community-workarounds.md`

#### Failed Bypass Timeline (Jan 8-9, 2026)

| Time     | Attempt                              | Result                                               |
|----------|--------------------------------------|------------------------------------------------------|
| Jan 8 PM | Opus 4.5 fingerprint spoofing        | Failed - "authorized for use with Claude Code only"  |
| Jan 8 PM | Direct header injection              | Failed - sophisticated detection, broke other models |
| Jan 9 AM | opencode-anthropic-auth@0.0.7 plugin | Worked briefly, then blocked                         |
| Jan 9 PM | Community coordination attempts      | All failed within hours                              |

**Conclusion:** Cat-and-mouse is not a viable strategy. Anthropic's detection is actively maintained.

### Rate Limits vs Usage Quota (Critical Distinction)

**⚠️ There are TWO distinct limit types, and they behave differently:**

| Limit Type              | Scope                        | Docker Bypass? | Resets                           |
|-------------------------|------------------------------|----------------|----------------------------------|
| **Request-rate limits** | Device (Statsig fingerprint) | ✅ YES         | Immediately with new fingerprint |
| **Weekly usage quota**  | Account                      | ❌ NO          | Weekly (Sunday)                  |

**Request-rate limits:** Per-device throttling based on Statsig fingerprint. Fresh Docker container = fresh fingerprint = no throttling.

**Weekly usage quota:** Account-level weekly allocation (e.g., "97% used"). Tied to account, not device. Docker escape hatch does NOT bypass this.

**Test evidence (Jan 20-21):**
- Wiped `~/.claude-docker/`, logged in as gmail account
- Usage charged to gmail (2%→3%) while sendcutsend stayed at 94-95%
- Copying Statsig fingerprint did NOT bypass 97% quota

**When to use Docker escape hatch:**
- ✅ Request-rate limited (getting throttled, need to spawn immediately)
- ❌ Weekly quota exhausted (must wait for reset or switch accounts)

**Source:** `kb-e3e0a8`, `kb-c3dbe7`, `.kb/investigations/2026-01-21-inv-synthesize-anthropic-blocking-kb-quick.md`

### Cross-Account Rate Limit Bug

**Problem:** Device fingerprinting causes one account's rate limit to affect other accounts on same device.

**Status:** Known bug since March 2025 (GitHub #630), unfixed.

**Workaround:** Docker container (confirmed working for rate limits, not for quota).

**Source:** `~/.kb/investigations/2025-11-30-claude-code-cross-account-rate-limit-bug.md`

---

## DeepSeek V3: Viable Primary Option

### Previous Understanding (Jan 18)

> "The current version of the deepseek-chat model's Function Calling capability is unstable, which may result in looped calls or empty responses." - DeepSeek API docs

**Recommendation at the time:** Do not use for agentic work.

### Current Understanding (Jan 19)

**Test performed:** Spawned DeepSeek V3 agent via OpenCode API with investigation skill.

**Tool calls executed successfully:**
- Read (3 files)
- Grep (17 matches found)
- Bash (ran go test)
- Write (created investigation file)
- kb quick constrain (externalized knowledge)

**Result:** 3 minutes, 62K tokens, successful completion with SYNTHESIS.md.

**Conclusion:** DeepSeek V3 function calling is stable enough for standard orchestration tasks. The "unstable" warning may be outdated or apply only to edge cases.

### Revised Recommendation

| Workload                            | Recommended Model     | Cost             |
|-------------------------------------|-----------------------|------------------|
| Standard investigation/feature work | DeepSeek V3           | $0.25/$0.38/MTok |
| Complex reasoning, architecture     | Claude Opus (via Max) | $200/mo flat     |
| Tool-heavy bursts (>2000 req/min)   | Sonnet API or Max     | Variable         |
| Cost-insensitive, highest quality   | Claude Opus           | $5/$25/MTok      |

**Source:** `.orch/workspace/og-inv-test-deepseek-v3-19jan-25d3/SYNTHESIS.md`

---

## Cost Visibility Gap

### The Problem

Switched from free Gemini to paid Sonnet on Jan 9 with **no cost tracking**:
- Dashboard shows Max subscription usage (OAuth)
- Dashboard does NOT show API token usage
- No per-spawn cost visibility
- No budget alerts

**Result:** $402 spent in ~2 weeks without awareness, ramping to $70-80/day.

### Current Status (Feb 2026): Substantially Mitigated

**Token counting exists** (`cmd/orch/tokens.go`, 336 lines): `orch tokens` shows per-session and aggregate token counts (input, output, cache read, reasoning). Supports `--all` (include completed), `--json`, and per-session detail views. Queries `pkg/opencode.TokenStats` from OpenCode API.

**Account capacity tracking exists** (`pkg/account/account.go`): `ShouldAutoSwitch()` checks capacity thresholds (`FiveHourRemaining`, `SevenDayRemaining` percentages) and capacity-aware spawn routing automatically avoids exhausted accounts.

**Still missing:**
- Dollar-cost calculation from token counts (model-specific pricing)
- Per-spawn cost aggregation
- Budget alerts (80%/95%/100% thresholds)

**Largely mitigated by architecture changes:**
1. Default path is Max subscription (flat $200/mo) — no pay-per-token surprise
2. Per-spawn account distribution prevents single-account quota exhaustion
3. Token counting provides session-level visibility for Max credit consumption
4. The gap only matters when using non-Max paths (DeepSeek, OpenAI API)

**Source:** `.kb/investigations/archived/2026-01-12-inv-sonnet-cost-tracking-requirements.md`, `.kb/investigations/2026-01-18-inv-add-api-cost-tracking-widget.md`

---

## Spawn Path Economics

### Current Architecture (Feb 2026)

The dual spawn architecture has been refactored from monolithic `backend.go` into a centralized resolver system (`pkg/spawn/resolve.go`) with model-aware backend routing and per-spawn account distribution.

| Path                  | Backend                                   | Models                             | Cost               | Use When                                   |
|-----------------------|-------------------------------------------|------------------------------------|--------------------|--------------------------------------------|
| **Primary (default)** | Claude CLI                                | Sonnet, Opus, Haiku (Anthropic)    | $200/mo flat (Max) | All Anthropic model work                   |
| **Alternative**       | OpenCode API                              | DeepSeek, OpenAI/Codex, Gemini Pro | Pay-per-token      | Non-Anthropic providers, cost optimization |
| **Override**          | OpenCode API + `allow_anthropic_opencode` | Any                                | Pay-per-token      | Explicit user config override              |

**Key change from Jan 2026:** The primary/escape-hatch distinction has inverted. Claude backend + Max is now the *default*, not the escape hatch. The infrastructure escape hatch still exists (`InfrastructureDetected` → auto-select Claude backend) but is now redundant since Claude is already the default.

### Account Distribution (Feb 2026)

Per-spawn account selection enables capacity-aware routing across multiple Max accounts. This addresses the weekly quota exhaustion problem — when one account nears its limit, spawns automatically route to accounts with remaining capacity.

**Account schema** (`~/.orch/accounts.yaml`):

| Field      | Purpose                                          | Example                    |
|------------|--------------------------------------------------|----------------------------|
| `email`    | Account identifier                               | `dylan@sendcutsend.com`    |
| `tier`     | Subscription tier (affects quota)                 | `20x`, `5x`               |
| `role`     | Routing priority                                  | `primary`, `spillover`     |
| `config_dir` | Claude CLI config directory for account isolation | `~/.claude-personal`     |

**How it works:**

1. `ResolvedSpawnSettings` includes an `Account` field with provenance tracking
2. Resolution precedence: CLI flag (`--account`) > capacity-aware heuristic > default (first primary account)
3. Heuristic checks: `FiveHourRemaining` and `SevenDayRemaining` percentages (healthy ≥ 20%)
4. `BuildClaudeLaunchCommand()` in `pkg/spawn/claude.go` injects `CLAUDE_CONFIG_DIR` and unsets `CLAUDE_CODE_OAUTH_TOKEN` when using a non-default account
5. Two independent auth mechanisms: OpenCode OAuth (`~/.local/share/opencode/auth.json`, global) vs Claude CLI config dir (`CLAUDE_CONFIG_DIR`, per-spawn)

**Routing strategy:** Work-first (primary accounts), personal-spillover (fallback when primary exhausted).

**Source:** `pkg/spawn/resolve.go` (account resolution), `pkg/spawn/claude.go` (env var injection), `pkg/account/account.go` (capacity checking)

### Economic Decision Tree

```
What provider is the model?
  → Anthropic: Claude backend (default, Max subscription)
  → OpenAI/Codex: OpenCode backend (auto-routed)
  → DeepSeek: OpenCode backend (auto-routed)
  → Google: OpenCode backend (auto-routed, Flash BANNED)

Which account? (Claude backend only)
  → Auto: capacity-aware heuristic routes to healthiest account
  → --account work: force specific account (CLI override)
  → Primary accounts checked first, spillover accounts as fallback
  → Healthy threshold: ≥20% remaining on both 5-hour and 7-day limits

Need Anthropic model via OpenCode backend?
  → Set allow_anthropic_opencode: true in ~/.orch/user.yaml
  → Warning: bypasses fingerprinting protection, may fail

Is cost the primary constraint?
  → YES: Use DeepSeek V3 ($0.25/$0.38/MTok) via OpenCode backend
  → NO: Use Sonnet via Max subscription (default)

Is highest quality needed?
  → YES: Use Opus via Max subscription (Claude backend)
  → NO: Use Sonnet (default) or DeepSeek V3
```

**Source:** `pkg/spawn/resolve.go` (centralized resolver), `.kb/models/model-access-spawn-paths/model.md`

### Config Resolution System (Feb 2026)

Spawn settings are now resolved via `ResolvedSpawnSettings` with full provenance tracking. Each setting (backend, model, tier, spawn mode, MCP, mode, validation, **account**) records its source:

**Precedence (highest to lowest):**
1. CLI flags (`--backend`, `--model`, etc.)
2. Beads labels (e.g., `needs:playwright`)
3. Project config (`.orch/config.yaml` — per-backend model, spawn mode)
4. User config (`~/.orch/user.yaml` — default model, backend, tier)
5. Heuristics (infrastructure detection, task scope inference, skill defaults)
6. Defaults (Claude backend, Sonnet model, headless mode)

**Model-aware backend routing:** When backend is not explicitly set via CLI, the model's provider determines the backend automatically. Anthropic models → Claude backend. Non-Anthropic models → OpenCode backend. This generalizes the dual-spawn architecture into provider-based routing.

**Implementation:** `pkg/spawn/resolve.go` (resolver), `pkg/spawn/claude.go` (Claude backend), `pkg/spawn/opencode_mcp.go` (OpenCode backend)

---

## Alternatives Evaluated

### OpenAI/Codex (First-Class Provider, Feb 2026)

**Status:** Promoted from "backup alternative" to first-class provider in the model alias system.

- 12 model aliases: `gpt`, `gpt4o`, `gpt-4o`, `gpt4o-mini`, `gpt-5`, `gpt5-latest`, `gpt-5-mini`, `o3`, `o3-mini`, `codex`, `codex-mini`, `codex-max`, `codex-latest`, `codex-5.1`, `codex-5.2`
- Routed to OpenCode backend automatically via `modelBackendRequirement()`
- Codex CLI available (terminal agent like Claude Code, GPT Pro OAuth path)
- ChatGPT Pro ($200/mo) — no API access included (same as Anthropic Max)

**Source:** `pkg/model/model.go` (aliases), `pkg/spawn/resolve.go` (routing), `.kb/investigations/archived/2026-01-18-inv-research-compare-openai-chatgpt-pro-anthropic-max.md`

### OpenCode Zen (Cooperative Buying Pool)

- Claims breakeven pricing via volume pooling
- Sustainability uncertain (no financial transparency)
- Not recommended until funding model proven

### OpenCode Black ($200/mo)

- Emergency response to Jan 9 Anthropic block
- Routes through "enterprise gateway" to bypass restrictions
- Temporary cat-and-mouse, not sustainable
- Treat as industry drama, not strategic option

**Source:** `.kb/investigations/archived/2026-01-13-research-opencode-zen-black-architecture-economics.md`

---

## Strategic Questions Answered

### Q: Is Sonnet API cheaper than Max subscription?

**A:** At normal usage, no. Jan 18 showed $70-80/day burn rate with Sonnet API → $2,100-2,400/mo projected. Max at $200/mo is 10x cheaper.

### Q: Should we use DeepSeek for orchestration?

**A:** Yes, now that function calling is confirmed working. DeepSeek V3 at $0.25/$0.38/MTok is viable for most orchestration work. Reserve Opus for complex reasoning.

### Q: When to use escape hatch (Max subscription)?

**A:** Max subscription via Claude backend is now the **default**, not the escape hatch. The infrastructure escape hatch (`InfrastructureDetected` flag) still exists but is redundant since Claude is already the default backend. The real question is now "when to use non-Anthropic providers?" — answer: when cost optimization matters or when you need OpenAI/Codex-specific capabilities.

### Q: Is second Max subscription needed?

**A:** No. Cancelled Jan 13. Opus gate forced lighter consumption patterns. Can re-subscribe in 5 minutes if needed.

### Q: How does multi-account routing work economically?

**A:** Two Max accounts (work 20×, personal 5×) provide $300/mo total capacity. The spawn resolver routes based on remaining capacity — primary accounts first, spillover when exhausted. This extends effective weekly quota by ~40% (5× account adds 41.6M weekly credits on top of work account's 83.3M). The `CLAUDE_CONFIG_DIR` env var injection in `pkg/spawn/claude.go` enables per-spawn account isolation without global state switching.

---

## Constraints

### C1: Anthropic Protects Claude Code Revenue

Fingerprinting blocks third-party tools from Max subscription. This is intentional product strategy, not a bug. Expect continued enforcement. The `allow_anthropic_opencode: true` user config option provides an explicit escape hatch, but Anthropic models on OpenCode backend are blocked by default in code (`validateModelCompatibility()`).

### C2: Cost Visibility Required Before API Usage

The $402 surprise proves: never use pay-per-token without cost tracking. Implement tracking before returning to Sonnet API as default.

### C3: Model Quality vs Task Complexity

**Benchmark caveat:** SWE-bench scores (R1: 49.2% vs Opus: 80.9%) measure complex multi-file coding, not general orchestration. Most orchestration tasks (investigation, search, file reading) don't require SWE-bench-level capability.

**Practical guidance:**
- Standard investigation/search → DeepSeek V3 sufficient (tested Jan 19)
- Complex multi-file refactoring → Prefer Claude
- Architecture decisions → Prefer Opus (reasoning quality matters)

### C4: Gemini Flash Banned for Agent Work

Originally blocked by 2,000 req/min TPM limit. Now **explicitly banned** in code via `validateModel()` in `pkg/spawn/resolve.go` — any Flash model (Google provider + "flash" in model ID) returns a hard error. This is enforced at spawn resolution time, not just a recommendation.

---

## References

### Decisions
- `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md`
- `.kb/decisions/2026-01-13-cancel-second-claude-max-subscription.md`
- ~~`.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md`~~ (DELETED — decision absorbed into code patterns in `pkg/spawn/resolve.go`)

### Investigations
- `.kb/investigations/archived/2026-01-28-inv-download-analyze-https-she-llac.md` - Credit formula, free cache reads, value multipliers
- `.kb/investigations/archived/2026-01-08-inv-opus-auth-gate-fingerprinting.md`
- `.kb/investigations/archived/2026-01-09-inv-anthropic-oauth-community-workarounds.md`
- `.kb/investigations/archived/2026-01-12-inv-sonnet-cost-tracking-requirements.md`
- `.kb/investigations/2026-01-18-inv-add-api-cost-tracking-widget.md`
- `.kb/investigations/archived/2026-01-18-research-compare-deepseek-models-anthropic-models.md`
- `.kb/investigations/2026-01-21-inv-synthesize-anthropic-blocking-kb-quick.md` - kb quick entries synthesis
- `~/.kb/investigations/2025-11-30-claude-code-cross-account-rate-limit-bug.md`

### Models & Guides
- `.kb/models/model-access-spawn-paths/model.md`
- `.kb/guides/model-selection.md`

### Test Evidence
- `.orch/workspace/og-inv-test-deepseek-v3-19jan-25d3/SYNTHESIS.md` - DeepSeek V3 function calling test

### Probes
- `.kb/models/orchestration-cost-economics/probes/2026-02-20-model-drift-stale-references-audit.md` — Identified 3 deleted references, stale spawn path economics, expanded provider ecosystem
- `.kb/models/orchestration-cost-economics/probes/2026-02-24-probe-automatic-account-distribution-design.md` — Verified per-spawn account infrastructure gaps, designed CLAUDE_CONFIG_DIR injection

**Primary Evidence (Verify These):**
- Anthropic billing dashboard - Actual spend history showing $402 in ~2 weeks
- `~/.local/share/opencode/auth.json` - Auth token storage for OpenCode backend (NOT used by Claude CLI backend)
- DeepSeek API documentation - Current pricing ($0.25/$0.38/MTok) and function calling status
- `pkg/spawn/resolve.go` - Centralized spawn resolver with provenance tracking (8 settings including Account)
- `pkg/spawn/claude.go` - Claude CLI launch command with CLAUDE_CONFIG_DIR account injection
- `pkg/account/account.go` - Account schema (Tier, Role, ConfigDir), capacity checking, auto-switch
- `pkg/model/model.go` - Model alias ecosystem (4 providers, 30+ aliases)
- `cmd/orch/tokens.go` - Token counting implementation (input, output, cache read, reasoning per session)
- she-llac.com credit formula reverse engineering - Internal credit system documentation
