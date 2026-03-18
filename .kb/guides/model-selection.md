# Model Selection Guide

**Purpose:** Single authoritative reference for model selection, aliases, and provider architecture in orch-go. Synthesized from 10 investigations spanning Dec 20, 2025 - Jan 4, 2026.

**Last verified:** Feb 26, 2026

---

## Quick Reference

### Model Aliases

| Alias | Provider/Model | Use When |
|-------|---------------|----------|
| `opus` | anthropic/claude-opus-4-5-20251101 | Complex work, debugging, architecture (default) |
| `sonnet` | anthropic/claude-sonnet-4-5-20250929 | Simple edits, typo fixes, known simple scope |
| `haiku` | anthropic/claude-haiku | Routing, triage, simple classification |
| `flash` | google/gemini-2.5-flash | Cost-sensitive, alternative provider |
| `flash3` | google/gemini-3-flash-preview | Alternative Gemini 3 flash alias |
| `pro` | google/gemini-2.0-pro | Gemini with higher reasoning capability |

### Spawn Examples

```bash
# Default (Opus) - recommended for most work
orch spawn investigation "analyze auth system"

# Explicit model selection
orch spawn --model sonnet feature-impl "fix typo in README"
orch spawn --model flash investigation "analyze large codebase"

# Rate-limited escape hatch
orch spawn --model flash feature-impl "task" # Gemini, pay-per-token
```

---

## Architecture

### Responsibility Split

| Layer | Responsibility | Location |
|-------|---------------|----------|
| **Model Resolution** | Alias → provider/model mapping | pkg/model/model.go |
| **Account Management** | Claude Max OAuth, account switching | pkg/account/account.go |
| **Runtime Auth** | API auth at inference time | OpenCode (via auth.json) |
| **Session Creation** | Passing model to OpenCode | pkg/opencode/client.go |

**Key insight:** orch handles model selection and Claude Max accounts. OpenCode handles runtime auth. They handoff via `~/.local/share/opencode/auth.json` for Anthropic tokens.

### The Flow

```
orch spawn --model opus ...
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. MODEL RESOLUTION (pkg/model)                                 │
│     model.Resolve("opus") → {Provider: "anthropic",             │
│                               ModelID: "claude-opus-4-5-20251101"}│
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. SPAWN CONFIG                                                 │
│     spawn.Config.Model = resolvedModel.Format()                 │
│     → "anthropic/claude-opus-4-5-20251101"                      │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. OPENCODE INVOCATION                                          │
│     Headless: HTTP API POST /session (model in body)            │
│     Inline: opencode run --model {model} ...                    │
│     Tmux: opencode attach --model {model} ...                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Default Model Behavior

**Default is Opus** - When no `--model` flag is provided:

```go
// In pkg/model/model.go
var DefaultModel = ModelSpec{
    Provider: "anthropic",
    ModelID:  "claude-opus-4-5-20251101",
}
```

**Why Opus is default:**
- Best reasoning capability for orchestration work
- Covered by Claude Max subscription (no per-token cost)
- Orchestrator skill guidance recommends Opus for complex work
- Matches user expectation for high-quality agents

**Historical note:** Default was briefly Gemini 3 Flash during development, causing confusion. Changed to Opus to align with orchestrator guidance (Dec 2025).

---

## Model-Aware Backend Routing (Feb 2026)

Models auto-route to the correct backend based on provider:

| Provider | Backend | Spawn Mode |
|----------|---------|------------|
| Anthropic (opus, sonnet, haiku) | Claude CLI | Tmux (always) |
| Google (flash, pro) | OpenCode | Headless or tmux |
| OpenAI (codex, gpt-5) | OpenCode | Headless or tmux |

**Constraint:** Anthropic models must never be spawned via OpenCode backend — Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026). The `--backend` CLI flag overrides auto-routing.

### Model Capability Requirements

Not all models can follow the orch worker agent protocol:

| Model | Compatibility | Notes |
|-------|--------------|-------|
| Claude Opus / Sonnet | Reliable | Primary models for all work |
| GPT-5.2-codex | **Unreliable** | Hallucinated constraints, excessive token use, failed session close (3/3 agents stalled) |
| gpt-4o | **Incompatible** | Spawns but never starts working — can't handle agentic workflows |
| Gemini Flash | Usable for simple tasks | Cost-effective for large context but lower reasoning |

### Daemon Model Inference

The daemon infers model from skill type: opus for deep-reasoning skills (investigation, architect, debugging), sonnet for implementation skills (feature-impl, issue-creation). Model-drift issues use a **spawn-count threshold** (3 stale spawns) rather than time-based, measuring actual impact.

---

## Spawn Mode Model Passing

All three spawn modes correctly pass the `--model` flag:

| Mode | How Model is Passed | Verified |
|------|---------------------|----------|
| **Headless** | HTTP POST /session with model field | Yes |
| **Inline** | `opencode run --model {model}` CLI flag | Yes |
| **Tmux** | `opencode attach --model {model}` CLI flag | Yes |

**Historical bug:** Early headless mode (HTTP API) ignored model parameter. Fixed by adding model field to CreateSessionRequest and ensuring API passes it through (Dec 22, 2025).

---

## Model Selection Strategy

### When to Use Each Model

| Skill | Recommended Model | Why |
|-------|------------------|-----|
| `investigation` | Opus (default) | Understanding codebase requires depth |
| `architect` | Opus (default) | Design decisions require tradeoff analysis |
| `systematic-debugging` | Opus (default) | Root cause analysis requires reasoning |
| `codebase-audit` | Opus (default) | Comprehensive review requires thoroughness |
| `feature-impl` (complex) | Opus (default) | Multi-step implementation needs context |
| `feature-impl` (simple) | Sonnet | Single-file typo fixes, simple edits |

**The test:** Before downgrading to Sonnet, ask: "Would I trust a quick summary or do I need thorough analysis?"

### Rate-Limiting Escalation

When Claude Max hits rate limits:

1. **Primary:** Switch Claude Max account
   ```bash
   orch account switch work  # Second Max account
   ```

2. **Secondary:** Use Gemini (pay-per-token)
   ```bash
   orch spawn --model flash feature-impl "task"
   ```

**Account management:**
- `orch account list` - Show saved accounts
- `orch account switch <name>` - Switch to different Max account
- Accounts stored in `~/.orch/accounts.yaml`

---

## Multi-Provider Architecture

### Anthropic (Claude)

- **Auth:** OAuth via Claude Max subscription
- **Token management:** orch handles refresh, writes to OpenCode's auth.json
- **Account switching:** orch manages multiple Max accounts for capacity

### Google (Gemini)

- **Auth:** API key (no OAuth)
- **Token management:** OpenCode handles via its own config
- **No orch account management needed** - Simple API key

### Future Providers (OpenRouter, DeepSeek)

- **Expected pattern:** API key based (like Gemini)
- **orch responsibility:** Add model aliases to pkg/model
- **OpenCode responsibility:** Handle API key auth
- **No orch account management needed** - They're not OAuth providers

---

## Cost Considerations (Late 2025 Pricing)

### Claude API Pricing

| Model | Input | Output | Notes |
|-------|-------|--------|-------|
| Opus 4.5 | $5.00/MTok | $25.00/MTok | Highest capability |
| Sonnet 4.5 | $3.00/MTok | $15.00/MTok | ≤200K tokens |
| Sonnet 4.5 (>200K) | $6.00/MTok | $22.50/MTok | Context cliff (API pricing only) |
| Haiku 4.5 | $1.00/MTok | $5.00/MTok | Triage/routing |

**Key insight:** Sonnet's API price doubles at 200K tokens ("context cliff"). Claude Code now defaults to 1M context windows, so this only affects direct API usage.

### Claude Max Subscriptions

| Plan | Price | Usage |
|------|-------|-------|
| Pro | $20/mo | Basic usage |
| Max 5x | $100/mo | 5x Pro (~100 turns/day break-even) |
| Max 20x | ~$400/mo | 20x Pro (~400 turns/day break-even) |

**Recommendation:** Max 5x for power users. API for automated/cached workflows.

### Gemini Pricing

| Model | Input | Output |
|-------|-------|--------|
| Flash 2.0 | ~$0.10-0.30/MTok | Variable |
| Pro 2.0 | ~$1.25-2.00/MTok | Variable |

**Gemini advantage:** Much cheaper for large context work.

---

## Model Arbitrage Pattern

From the investigations, a three-tier arbitrage strategy emerged:

| Tier | Purpose | Recommended Model | Cost |
|------|---------|-------------------|------|
| **1. Routing** | Intent detection, triage | Haiku or Llama 4 Scout | <$0.30/MTok |
| **2. Execution** | General tasks, coding | Gemini Flash or DeepSeek | <$0.30/MTok |
| **3. Reasoning** | Complex planning, debugging | Opus or DeepSeek R1 | Higher |

**For orch-go:** We default to Opus (Tier 3) because orchestration work is complex reasoning work. Tier 1/2 routing could be future optimization.

---

## Debugging Model Issues

### "Wrong model used"

**Check resolution:**
```bash
# In Go tests
go test ./pkg/model -v -run TestResolve
```

**Check what orch passes:**
```bash
# Look at spawn output
orch spawn --model opus investigation "test" 2>&1 | grep -i model
```

### "Model ignored in headless mode"

**Historical bug** (fixed Dec 2025). If you see this:
1. Update orch-go (fix is in pkg/opencode/client.go)
2. Verify model field in CreateSessionRequest

### "Rate limited, need different model"

```bash
# Check current account
orch account list

# Switch accounts
orch account switch work

# Fall back to Gemini
orch spawn --model flash ...
```

---

## Implementation Details

### pkg/model/model.go

```go
// Key structures
type ModelSpec struct {
    Provider string
    ModelID  string
}

// Aliases map
var Aliases = map[string]ModelSpec{
    "opus":    {Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"},
    "sonnet":  {Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"},
    "haiku":   {Provider: "anthropic", ModelID: "claude-haiku"},
    "flash":   {Provider: "google", ModelID: "gemini-2.5-flash"},
    "flash3":  {Provider: "google", ModelID: "gemini-3-flash-preview"},
    "pro":     {Provider: "google", ModelID: "gemini-2.0-pro"},
    // ... more aliases
}

// Resolution
func Resolve(spec string) ModelSpec {
    if spec == "" {
        return DefaultModel  // Opus
    }
    spec = strings.ToLower(spec)  // Case-insensitive
    if alias, ok := Aliases[spec]; ok {
        return alias
    }
    // Parse provider/model format
    return parseProviderModel(spec)
}
```

### Adding New Aliases

1. Add to `Aliases` map in pkg/model/model.go
2. Add test case in pkg/model/model_test.go
3. No account management changes needed (API key providers)

---

## Source Investigations (Synthesized)

This guide consolidates findings from:

1. **2025-12-20-inv-investigate-model-flexibility-arbitrage-orch.md** - Initial model alias implementation
2. **2025-12-20-inv-research-gemini-model-arbitrage-alternatives.md** - Gemini/DeepSeek arbitrage research
3. **2025-12-20-research-model-arbitrage-api-vs-max.md** - API vs Max pricing analysis
4. **2025-12-21-inv-fix-buildspawncommand-pass-model-flag.md** - BuildSpawnCommand --model fix
5. **2025-12-21-inv-model-handling-conflicts-between-orch.md** - Model handling bugs root cause
6. **2025-12-22-inv-model-flexibility-phase-expand-model.md** - Headless mode model support
7. **2025-12-23-inv-model-selection-issue-architect-agent.md** - OpenCode API model fix
8. **2025-12-24-inv-model-provider-architecture-orch-vs.md** - Provider auth architecture
9. **2025-12-24-inv-test-gemini-flash-model-resolution.md** - Gemini alias verification
10. **2026-01-04-inv-implement-priority-cascade-model-dashboard.md** - Dashboard model display

**Key patterns across investigations:**
- Model selection is now consistent across all spawn modes (after Dec 2025 fixes)
- Opus is the correct default for orchestration work
- Multi-provider support is additive (just add aliases)
- Claude Max account management is Anthropic-specific complexity
