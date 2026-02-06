# Model Selection Guide

**Purpose:** Authoritative reference for model selection in orch-go, reflecting the post-Jan-9-2026 reality where Anthropic blocks third-party OAuth access.

**Last verified:** Jan 21, 2026

---

## Quick Reference

### Current Reality (Jan 2026)

| Model            | Access Method                      | Cost                   | Best For                                                                                                        |
| ---------------- | ---------------------------------- | ---------------------- | --------------------------------------------------------------------------------------------------------------- |
| **Opus 4.6**     | Claude CLI only (Max subscription) | $200/mo flat           | Orchestration, complex reasoning, architecture                                                                  |
| **Sonnet 4.5**   | API or Claude CLI                  | $3/$15/MTok or $200/mo | General work, feature implementation                                                                            |
| **DeepSeek V3**  | API (OpenCode)                     | $0.25/$0.38/MTok       | Cost-sensitive work, standard investigations                                                                    |
| **Gemini Flash** | API (OpenCode)                     | Free tier available    | Large context (>200K), but 2K req/min limit                                                                     |
| **GPT-5.2**      | OpenCode (ChatGPT Pro)             | $200/mo flat           | Worker tasks + **interactive orchestrator-assist** (human-in-loop); **unsuitable for autonomous orchestration** |

### Key Constraints

1. **Opus requires Claude CLI** - Anthropic fingerprinting blocks API access since Jan 9
2. **GPT-5.2 unsuitable for autonomous orchestration** - Role boundary collapse, reactive gate handling, excessive deliberation. _Allowed_ for interactive orchestrator-assist with human supervision (see 2026-01-30 decision)
3. **Gemini Flash has TPM limits** - 2,000 req/min blocks tool-heavy agents
4. **DeepSeek V3 function calling works** - Confirmed Jan 19, despite "unstable" warning in docs

---

## Spawn Examples

```bash
# Default: Claude CLI + Opus (primary path since Jan 18)
orch spawn investigation "analyze auth system"

# Explicit API path (opt-in, pay-per-token)
orch spawn --backend opencode --model sonnet feature-impl "add logout button"

# Cost-optimized (DeepSeek V3)
orch spawn --backend opencode --model deepseek investigation "explore codebase"

# Rate limit escape (fresh fingerprint)
orch spawn --backend docker investigation "explore X"

# OpenAI (worker tasks only, NOT orchestration)
orch spawn --backend opencode --model gpt-5.2 feature-impl "simple edit"
```

---

## Architecture: Triple Spawn Paths

Model selection is now coupled to spawn backend due to Anthropic's OAuth blocking.

| Backend                  | Models Available              | Cost               | Use When                                      |
| ------------------------ | ----------------------------- | ------------------ | --------------------------------------------- |
| **Claude CLI** (default) | Opus, Sonnet (Max)            | $200/mo flat       | Primary work, orchestration, quality-critical |
| **OpenCode API**         | Sonnet, DeepSeek, Gemini, GPT | Pay-per-token      | Cost tracking needed, headless batch work     |
| **Docker**               | Opus, Sonnet (Max)            | $200/mo + overhead | Rate limit escape (fresh fingerprint)         |

**Why Claude CLI is default (Jan 18 decision):**

- API costs hit $70-80/day ($2,100-2,400/mo projected)
- Max subscription is 10x cheaper at that usage level
- Opus quality available only via CLI

---

## Model Selection by Task

### Orchestration / Meta-Work

**Autonomous orchestration - Required: Opus 4.6 via Claude CLI**

Autonomous orchestration requires:

- Gate anticipation (synthesize flags upfront, not learn by hitting)
- Role boundary maintenance (delegate, don't collapse to worker)
- Failure adaptation (change strategy, not repeat)
- Confident execution (minimal deliberation)

**GPT-5.2 tested and failed for autonomous use** (Jan 21):

- 3 spawn attempts for multi-gate scenario
- Role boundary collapse (started debugging instead of delegating)
- 6+ identical timeout failures without strategy change
- 200+ second thinking blocks

**Interactive orchestrator-assist - GPT-5.2 allowed with human supervision** (Jan 30):

GPT-5.2 may be used for orchestrator-assist when a human is actively supervising and can intervene:

- Requires strict tool gating (spawn, close, push require approval)
- Human provides gate anticipation and strategic direction
- Human redirects when role boundaries blur
- Use when cost optimization or multi-model comparison is valuable

```bash
# Interactive orchestrator-assist with GPT-5.2 (human-in-loop required)
orch spawn --backend opencode --model gpt-5.2 --interactive orchestrator "coordinate feature work"
```

**Never use GPT-5.2 for:**

- Daemon orchestration (background services)
- Autonomous orchestrator spawns (unattended operation)
- Default orchestration mode

### Complex Reasoning / Architecture

**Recommended: Opus via Claude CLI**

```bash
orch spawn architect "design auth system"
orch spawn systematic-debugging "root cause analysis"
```

### Standard Investigations / Feature Work

**Options:**

- Opus (quality, $200/mo flat) - `orch spawn investigation "task"`
- DeepSeek V3 (cost, $0.25/$0.38/MTok) - `orch spawn --backend opencode --model deepseek investigation "task"`

DeepSeek V3 confirmed working for standard orchestration (Jan 19 test: 3 minutes, 62K tokens, successful completion with tool calls).

### Simple Edits / Known Scope

**Recommended: Sonnet**

```bash
orch spawn --model sonnet feature-impl "fix typo in README"
```

### Large Context (>200K tokens)

**Recommended: Gemini Flash** (but watch TPM limits)

```bash
orch spawn --backend opencode --model flash investigation "analyze large codebase"
```

**Warning:** Tool-heavy agents (35+ calls/sec) hit 2,000 req/min limit. Use Sonnet if hitting rate limits.

---

## Cost Economics

### The Jan 18 Discovery

Switched from free Gemini to paid Sonnet on Jan 9 with no cost tracking:

- **$402 spent in ~2 weeks** without awareness
- Ramping to **$70-80/day** ($2,100-2,400/mo projected)
- Max subscription at $200/mo is **10x cheaper**

### Breakeven Analysis

At Max subscription cost ($200/mo):

| Model       | Breakeven Usage                           |
| ----------- | ----------------------------------------- |
| DeepSeek V3 | ~317M input tokens OR ~526M output tokens |
| Sonnet      | ~67M input tokens OR ~13M output tokens   |
| Opus API    | ~40M input tokens OR ~8M output tokens    |

**Implication:** Heavy usage → Max subscription. Light/metered usage → API with cost tracking.

### Current Recommendation

1. **Primary:** Claude CLI + Max subscription (predictable $200/mo)
2. **Cost-sensitive:** DeepSeek V3 via API ($0.25/$0.38/MTok)
3. **Never:** Sonnet API without cost tracking (learned the hard way)

---

## Model Aliases

| Alias           | Provider/Model                       |
| --------------- | ------------------------------------ |
| `opus`          | anthropic/claude-opus-4-6            |
| `sonnet`        | anthropic/claude-sonnet-4-5-20250929 |
| `haiku`         | anthropic/claude-haiku               |
| `flash`         | google/gemini-2.5-flash              |
| `flash3`        | google/gemini-3-flash-preview        |
| `pro`           | google/gemini-2.0-pro                |
| `deepseek`      | deepseek/deepseek-chat               |
| `gpt5`, `gpt-5` | openai/gpt-5-20251215                |
| `o3`            | openai/o3                            |

---

## Rate Limit Handling

### Claude Max Rate Limits

1. **Primary:** Wait for reset (resets at 6am local)
2. **Secondary:** Switch account (if multiple Max subscriptions)
   ```bash
   orch account switch work
   ```
3. **Escape hatch:** Docker backend (fresh Statsig fingerprint)
   ```bash
   orch spawn --backend docker investigation "task"
   ```

**Note:** Docker fingerprint isolation bypasses device-level rate throttling, NOT weekly usage quota. The weekly quota is account-level.

### Gemini TPM Limits

Gemini Flash Paid Tier 2: 2,000 req/min

Tool-heavy agents (investigation, systematic-debugging) can hit this with a single agent. Solutions:

1. Use Sonnet instead
2. Apply for Tier 3 (20,000 req/min)
3. Accept retry delays (not recommended)

---

## Debugging

### "Model ignored"

```bash
# Check what orch passes
orch spawn --model opus investigation "test" 2>&1 | grep -i model
```

### "Opus auth rejected"

Opus requires Claude CLI backend:

```bash
# Wrong (will fail)
orch spawn --backend opencode --model opus investigation "task"

# Right
orch spawn --backend claude --model opus investigation "task"
# Or just (claude is default backend)
orch spawn investigation "task"
```

### "Rate limited"

```bash
# Check account status
orch account list

# Check usage
orch usage

# Switch accounts or use Docker escape hatch
orch account switch work
# or
orch spawn --backend docker investigation "task"
```

---

## References

### Decisions

- `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` - Anthropic blocking response
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Switch to Claude CLI default
- `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md` - GPT-5.2 autonomous orchestration findings
- `.kb/decisions/2026-01-30-gpt-interactive-orchestrator-assist-allowed.md` - GPT-5.2 allowed for interactive/human-in-loop

### Models

- `.kb/models/model-access-spawn-paths.md` - Detailed spawn path mechanics
- `.kb/models/orchestration-cost-economics.md` - Full cost analysis

### Investigations

- `.kb/investigations/2026-01-09-inv-anthropic-oauth-community-workarounds.md` - Why workarounds failed
- `.kb/investigations/2026-01-19-inv-test-deepseek-v3-function-calling.md` - DeepSeek V3 validation
- `.kb/investigations/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md` - GPT-5.2 analysis

### Benchmarks

- `.kb/benchmarks/2026-01-28-logout-fix-6-model-comparison.md` - 6 models on debugging task (Codex, DeepSeek passed; Opus, Sonnet, GPT, Gemini failed)
