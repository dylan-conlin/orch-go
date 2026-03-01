# Model: Model Access and Spawn Paths

**Domain:** Agent Spawning / Model Selection
**Last Updated:** 2026-02-28
**Synthesized From:** 5 investigations (Opus gate, Gemini TPM limits, community workarounds, cost tracking, escape hatch implementations) spanning Jan 8-12, 2026. Updated Feb 2026-27-28 via drift probes and model drift agent.

---

## Summary (30 seconds)

Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026), making **Claude CLI the default backend** for Anthropic models (was previously the "escape hatch"). The architecture now uses **model-aware backend routing**: Anthropic models → Claude CLI (tmux), non-Anthropic models (Google, OpenAI, DeepSeek) → OpenCode API (headless). Account routing is capacity-aware with primary/spillover accounts and a health threshold (>20%). Infrastructure independence — originally the escape hatch's key benefit — is now the default behavior since Claude CLI is the primary backend.

---

## Core Mechanism

### Available Models and Access Methods

**Anthropic Models (default, via Claude CLI):**
- **Opus 4.5** (`claude-opus-4-5-20251101`) - Highest quality, Max subscription
- **Sonnet 4.5** (`claude-sonnet-4-5-20250929`) - Default model, balanced quality/speed
- **Haiku** - Fast, lower cost

**Gemini Models (via OpenCode API):**
- **Flash** - BLOCKED at resolve layer (`validateModel` returns error for any flash model)
- **Pro** - Higher quality Gemini option

**OpenAI Models (via OpenCode API):**
- **GPT-4o**, **GPT-4o-mini** - General purpose
- **GPT-5.x** (alias: gpt-5 → gpt-5.2) - Latest generation
- **o3**, **o3-mini** - Reasoning models
- **Codex** models - Whitelisted via Codex plugin

**DeepSeek Models (via OpenCode API):**
- **deepseek-chat**, **deepseek-reasoner** (alias: "reasoning")

### Access Patterns

**Pattern 1: Claude CLI (Default for Anthropic Models)**
```
User → orch spawn → Claude CLI (tmux) → Anthropic API (Max subscription fingerprint)
```

**Characteristics:**
- Tmux window (visual progress, survives server restarts)
- Opus 4.5 access with Max subscription
- Flat $200/mo (unlimited usage)
- **Default since Feb 19, 2026** (Anthropic banned subscription OAuth in third-party tools)
- **Independence:** Does not depend on OpenCode server
- Dashboard visibility limited (tmux-only agents need reconciliation)

**Pattern 2: OpenCode API (For Non-Anthropic Models)**
```
User → orch spawn --model gpt-5 → OpenCode HTTP API (localhost:4096) → OpenAI/Google/DeepSeek API
```

**Characteristics:**
- Headless (no UI, returns immediately)
- High concurrency (5+ agents simultaneously)
- Dashboard visibility via SSE
- Pay-per-token pricing
- **Required for:** Google, OpenAI, DeepSeek models (Claude CLI can only run Anthropic)
- **Dependency:** OpenCode server must be running
- **Anthropic models blocked** unless `allow_anthropic_opencode: true` in user config

### Key Components

**Backend Selection Priority (pkg/spawn/resolve.go:resolveBackend):**
```
1. CLI --backend flag (highest priority)
2. Model-derived requirement (anthropic → claude; openai/google/deepseek → opencode)
3. Project config spawn_mode
4. User config backend
5. Infrastructure heuristic → claude (advisory when overridden)
6. Default: claude (since default model is Anthropic Sonnet)
```

Note: Infrastructure detection is **advisory** — when higher-priority settings
(CLI, model requirement, project/user config) specify a different backend,
infrastructure detection only emits a warning instead of overriding.

**Model-aware backend routing (post-resolve, Decision: kb-2d62ef):**
After initial backend resolution, if backend was NOT from CLI, the model's provider
overrides the backend: Anthropic → claude, non-Anthropic → opencode.
This is the primary routing mechanism since the Anthropic OAuth ban.

**Additional derived behavior:**
- `--backend claude` implies tmux spawn mode (auto-switch from headless)
- Anthropic models on opencode blocked by default (override: `allow_anthropic_opencode: true` in user config)
- Flash models blocked entirely at resolve layer
- When CLI says `--backend claude` but model is non-Anthropic (from lower precedence), model auto-resolves to Sonnet

**Account Routing (pkg/spawn/resolve.go:resolveAccount):**
```
1. CLI --account flag (highest priority)
2. Heuristic: capacity-aware routing (when CapacityFetcher set)
   - Check primary accounts first (sorted deterministically)
   - If any primary >20% capacity on both limits → use it
   - Else check spillover accounts
   - Else fallback to first primary (fail-open)
3. Default: first primary account
```

**Infrastructure Work Detection (cmd/orch/spawn_cmd.go:isInfrastructureWork):**
- Scans task description + beads issue for keywords
- Keywords (22): "opencode", "orch-go", "pkg/spawn", "pkg/opencode", "pkg/verify", "pkg/state", "cmd/orch", "spawn_cmd.go", "serve.go", "status.go", "main.go", "dashboard", "agent-card", "agents.ts", "daemon.ts", "skillc", "skill.yaml", "SPAWN_CONTEXT", "spawn system", "spawn logic", "spawn template", "orchestration infrastructure", "orchestration system"
- Auto-applies `--backend claude` (which implies tmux) when no higher-priority setting overrides
- Prevents agents from killing themselves (e.g., restarting OpenCode during spawn)

### State Transitions

**Normal spawn (Anthropic model, default):**
```
orch spawn feature-impl "task"
    ↓
Settings resolved via pkg/spawn/resolve.go
    ↓
Model: anthropic/claude-sonnet-4-5-20250929 (default)
Backend: claude (derived from model provider)
    ↓
Claude CLI in tmux window
    ↓
Account: capacity-aware routing (primary → spillover → first primary)
```

**Non-Anthropic model spawn:**
```
orch spawn --model gpt-5 feature-impl "task"
    ↓
Model: openai/gpt-5.2 (CLI flag, alias resolved)
Backend: opencode (derived from model provider)
    ↓
Headless session via OpenCode HTTP API
    ↓
Dashboard visibility via SSE
```

**Infrastructure spawn (auto-detected, advisory):**
```
orch spawn systematic-debugging "fix opencode server crash"
    ↓
Keyword detected: "opencode" (cmd/orch/spawn_cmd.go)
    ↓
No higher-priority backend setting → auto-apply: backend=claude
    ↓
claude backend implies tmux (derived)
    ↓
Tmux session via Claude CLI
    ↓
Survives OpenCode server restart
```

**Explicit backend + model override:**
```
orch spawn --backend claude --model opus architect "complex design"
    ↓
Backend: claude (explicit CLI flag)
    ↓
Model: opus (Max subscription)
    ↓
Tmux implied by claude backend
    ↓
Tmux session, highest quality
```

### Critical Invariants

1. **Never spawn OpenCode infrastructure work without --backend claude --tmux**
   - Violation: Agent kills itself mid-execution when server restarts
   - Now auto-detected: infrastructure keywords trigger `--backend claude` which implies tmux

2. **Infrastructure detection is advisory, not overriding (changed Feb 2026)**
   - Runs at priority 5 (below CLI, model requirement, project config, user config)
   - When higher-priority setting present, emits warning instead of overriding
   - Ensures explicit user choices are always respected

3. **Anthropic models blocked on OpenCode by default**
   - API requests to Anthropic models on opencode return error
   - Override: `allow_anthropic_opencode: true` in user config (`~/.orch/config.yaml`)
   - Opus specifically requires Claude CLI backend (fingerprinting blocks API)

4. **Claude CLI provides true independence**
   - Claude CLI binary ≠ OpenCode server
   - Tmux session persists across service restarts
   - Different authentication path (Max subscription OAuth)

5. **Flash models are blocked entirely (added Feb 2026)**
   - `validateModel()` returns error for any flash model
   - Supersedes the Gemini Flash TPM limit constraint — no workaround needed

---

## Why This Fails

### Failure Mode 1: Zombie Agents (Jan 8, 2026)

**Symptom:** Agents tracked in registry but never actually ran

**Root Cause:** Spawning with `--model opus` before understanding auth gate
- orch created registry entry
- OpenCode session created
- Anthropic rejected API request (fingerprinting)
- Agent hung in "running" state
- Consumed concurrency slot without doing work

**Examples:**
- orch-go-mo0ja, orch-go-pzi2i, orch-go-aoei0, orch-go-gd1gd, orch-go-lwc3o

**Fix:** Never use `--model opus` without `--backend claude`

### Failure Mode 2: Header Injection Conflicts (Jan 8, 2026)

**Symptom:** Gemini Flash spawns hung after attempting Opus bypass

**Root Cause:** Injected Claude Code headers (`x-app: cli`, `anthropic-version`, etc.) into OpenCode's Anthropic provider
- Bypassed Opus gate (didn't work)
- Broke Gemini spawns (headers conflicted with Bun fetch/SDK)
- System-wide impact from localized change

**Lesson:** Fingerprinting is more sophisticated than headers alone

### Failure Mode 3: Infrastructure Work Kills Itself

**Symptom:** Agent fixing OpenCode server crashes mid-execution

**Root Cause:** Agent spawned via OpenCode API, agent's fix restarts OpenCode server, agent's session killed

**Solution:** Infrastructure work detection auto-applies `--backend claude --tmux`

**Why auto-detection matters:**
- Humans forget to add flags manually
- Task description might not mention "opencode" explicitly
- Keyword scan catches common patterns
- Escape hatch becomes invisible safety net

---

## Constraints

### Constraint 1: Opus Auth Gate Fingerprinting

**Why we can't "just use Opus via API":**

Anthropic's auth gate checks multiple fingerprints:
- HTTP headers (User-Agent, x-app, anthropic-version, anthropic-beta)
- TLS fingerprint (JA3 - TLS client hello characteristics)
- HTTP/2 frame fingerprinting (frame ordering, priorities)
- Header ordering (not just presence)

**Evidence:** Injecting all known headers still resulted in rejection (inv-2026-01-08)

**Implication:** Bypassing the gate requires either:
1. Proxying through actual Claude Code binary (complex)
2. Using Claude CLI with Max subscription (current default backend)
3. Accepting Sonnet/Flash as primary models

**Strategic question enabled:** "Is Opus quality worth $200/mo flat cost vs pay-per-token Sonnet/Flash?"

**This enables:** Anthropic to differentiate Claude Code from API access (product strategy)
**This constrains:** Cannot use Opus via API without Max subscription + Claude CLI

### Constraint 2: Critical Paths Need Independence

**Why backend independence matters:**

When building infrastructure, failure cascades if the build depends on that infrastructure:
- Fixing OpenCode → spawned via OpenCode → fix restarts server → agent dies → fix incomplete
- Debugging spawn system → spawned via spawn system → meta-circular trap

**Architectural principle:** Critical paths require mechanisms that don't depend on what can fail

**Current state (Feb 2026):** Claude CLI became the default backend, so infrastructure independence is now the default rather than an opt-in escape hatch. Non-Anthropic model work still uses OpenCode API.

**When this especially matters:**
- Building/fixing orch-go spawn system
- Debugging OpenCode server crashes
- Dashboard/monitoring infrastructure work
- Daemon implementation

**This enables:** Infrastructure work completes even when OpenCode fails (by default)
**This constrains:** Must maintain two spawn backends (complexity cost)

### Constraint 3: OpenCode Doesn't Expose Session State

**Why dashboard shows "wrong" status sometimes:**

OpenCode HTTP API provides:
- `/sessions` - List sessions with IDs
- `/sessions/:id` - Get session metadata
- `/sessions/:id/messages` - Get message history

**Missing:** Session execution state (running/idle/waiting/complete)

**Implication:** Dashboard must infer status from message history
- Parse recent messages for "Phase: Complete"
- Check registry state
- Verify session existence
- Priority cascade can show "running" when actually complete

**Related Model:** `.kb/models/dashboard-agent-status.md` - Status calculation mechanism

**This enables:** Simple OpenCode API without internal state exposure
**This constrains:** Dashboard must infer status from indirect signals

### Constraint 4: Cost Model Determines Concurrency

**OpenCode API (pay-per-token):**
- Variable cost scales with usage
- Natural limit from budget
- Currently ~$100-200/mo (Dylan's usage)
- Can spawn 5+ agents concurrently (only paying for active work)

**Claude CLI (Max subscription):**
- Flat $200/mo unlimited
- Could spawn unlimited agents
- But: No dashboard visibility
- But: Manual tmux management doesn't scale

**Strategic question enabled:** "What's the cost-optimal split between Claude CLI and OpenCode API?"

**Current answer (Feb 2026):** Claude CLI is the default ($200/mo flat for unlimited Anthropic models). OpenCode API used only for non-Anthropic models (pay-per-token).

**This enables:** Predictable cost for most work via Max subscription
**This constrains:** Non-Anthropic model work still incurs per-token costs

### Constraint 5: Gemini Flash Blocked Entirely (Updated Feb 2026)

**Previous state (Jan 2026):** Flash had 2,000 req/min TPM limit making it unreliable for agents.

**Current state (Feb 2026):** Flash models are **blocked at the resolve layer**. `validateModel()` in `pkg/spawn/resolve.go` returns an error for any flash model. No workaround available or needed.

**Why the change:** Flash TPM limits caused enough reliability problems that it was easier to block Flash entirely than to manage rate limiting. Sonnet is the default model.

**This supersedes:** Tier 3 application, rate limit workarounds, retry tolerance questions.

**This enables:** Clean error messages instead of subtle rate limiting failures
**This constrains:** Cannot use any Flash model for agent work

### Constraint 6: Community Workarounds are Fragile Cat-and-Mouse

**Why we don't bypass Opus gate:**

Community discovered workarounds for Anthropic's OAuth blocking:
- Tool name renaming (`bash` → `Bash_tool`)
- Official plugin (opencode-anthropic-auth@0.0.7)
- Rotating suffix (TTL-based hourly changes)

**All workarounds failed within hours:**
- Official plugin: Worked 6 hours, then re-blocked
- Tool renaming: Requires source edits on every OpenCode update
- Rotating suffix: Most resilient but highest maintenance burden

**Evidence:** Investigation `2026-01-09-inv-anthropic-oauth-community-workarounds.md` (474+ GitHub comments)

**Anthropic's fingerprinting:**
- Tool names (lowercase vs PascalCase + `_tool`)
- OAuth scope (`user:sessions:claude_code`)
- User-Agent patterns
- TLS fingerprints (JA3)
- HTTP/2 frame characteristics

**Implication:** Anthropic iterates faster than community can stabilize workarounds

**Community response:** 119+ users canceled Max subscriptions, migrated to alternative models

**Strategic decision:** Abandon workarounds, use Sonnet API as fallback, Gemini as primary when Tier 3 available

**This enables:** Stable, maintenance-free spawn system without workaround churn
**This constrains:** Cannot use Opus via API regardless of community discoveries

### Constraint 7: Cost Tracking Missing for Sonnet Usage

**Why we don't know current spend:**

Switched from free Gemini to paid Sonnet on Jan 9, 2026. No cost tracking implemented:
- Dashboard shows Max subscription usage (OAuth)
- Dashboard does NOT show API token usage (pay-per-token)
- No alerts when approaching budget limits
- Unknown daily burn rate

**Evidence:** Investigation `2026-01-12-inv-sonnet-cost-tracking-requirements.md`

**Strategic questions blocked:**
1. "Is Sonnet cheaper than Max subscription ($200/mo)?"
2. "Should we invest in Max for unlimited Opus?"
3. "Which spawn types consume most budget?"
4. "Are we approaching monthly limits?"

**Implication:** Can't make data-driven decisions about model selection without cost visibility

**Solutions available:**
1. Anthropic Usage API (`/v1/billing/cost`) - Daily cost data
2. Local token counting - Per-spawn granularity
3. Hybrid approach (recommended) - Both for strategic + tactical decisions

**Status:** Tracking not implemented, costs unknown since Jan 9

**This enables:** Simple setup without external API integrations
**This constrains:** Cannot make data-driven model selection decisions

---

## Evolution

**Jan 8, 2026:** Opus auth gate discovered
- Attempted to spawn with `--model opus`
- Received auth rejection: "This credential is only authorized for use with Claude Code"
- Created zombie agents (tracked but never ran)
- Investigated fingerprinting mechanism (TLS, HTTP/2, headers)

**Jan 8, 2026:** Header injection attempt failed
- Injected all known Claude Code headers into OpenCode provider
- Opus still rejected
- Gemini spawns started hanging (header conflicts)
- Abandoned spoofing approach

**Jan 10, 2026:** Dual spawn architecture emerged
- Implemented `--backend` flag for explicit path selection
- Documented escape hatch pattern in CLAUDE.md
- Opus becomes Max-subscription-only model
- Primary path remains OpenCode API with Sonnet/Flash

**Jan 10, 2026:** Backend flag bug fixed
- Decision doc examples used `--mode`, code used `--backend`
- Flag was being ignored
- Fixed naming, verified priority order

**Jan 9, 2026:** Gemini Flash TPM limits hit
- Single investigation agent hitting 2,000 req/min limit
- Tool-heavy spawns (35+ calls/sec) trigger rate limiting
- OpenCode retry logic causes 3-30s delays per request
- Forced immediate switch to Sonnet for reliability

**Jan 9, 2026:** Community Opus workarounds research
- Discovered community had found tool name fingerprinting mechanism
- Official plugin (0.0.7) released, then re-blocked within 6 hours
- 474+ GitHub comments documenting cat-and-mouse game
- Strategic decision: abandon workarounds, accept Sonnet/Gemini split

**Jan 11, 2026:** Infrastructure work auto-detection added
- Keyword-based detection (opencode, spawn, daemon, registry, etc.)
- Auto-applies `--backend claude --tmux` at priority 2.5
- Prevents agents from killing themselves
- Makes escape hatch invisible for common cases

**Jan 12, 2026:** Cost tracking gap identified
- No visibility into Sonnet spend since Jan 9 switch
- Dashboard shows Max usage but not API token usage
- Strategic decisions blocked without cost data
- Investigation documented requirements for tracking implementation

**Jan 12, 2026:** Model created from synthesis
- Recognized constraint has system-wide ripple effects
- Escape hatch pattern now embedded in spawn priority logic
- Cost/quality/reliability tradeoffs explicit
- Strategic questions about model usage surfaced

**Jan-Feb 2026:** Backend resolution refactored
- `selectBackend()` and `detectInfrastructureWork()` removed from config.go
- Backend selection moved to `pkg/spawn/resolve.go:resolveBackend()` with 6-level precedence
- Infrastructure detection moved to `cmd/orch/spawn_cmd.go:isInfrastructureWork()`
- Infrastructure detection became advisory (warns instead of overriding when explicit settings present)
- Flash models blocked entirely at resolve layer (validateModel returns error)
- `--backend claude` now implies tmux spawn mode
- `allow_anthropic_opencode: true` user config override added
- Expanded infrastructure keywords from 8 to 22

**Feb 19, 2026:** Anthropic OAuth ban reshaped architecture
- Anthropic banned subscription OAuth in third-party tools (OpenCode uses OAuth)
- Default backend changed from `opencode` to `claude` (default model is Anthropic Sonnet)
- Model-aware backend routing became primary mechanism (Decision: kb-2d62ef)
- Claude CLI → primary path (was escape hatch); OpenCode → multi-model access path
- Anthropic models on OpenCode blocked by default (override available)

**Feb 20-25, 2026:** Account distribution + modular extraction
- Account routing with capacity-aware primary/spillover heuristic (3-phase implementation)
- `resolveAccount()` checks primary accounts first, then spillover, fail-open to first primary
- Bug-type issues route to `systematic-debugging` (was `architect`)
- GPT-5 alias remapped to `gpt-5.2` to prevent zombie sessions
- Pre-create session for tmux spawns with non-default models
- Cross-project spawn fixes: beads DefaultDir, projectDir through kb context
- `--force-hotspot` requires `--architect-ref` with verified closed architect issue
- `--disallowedTools` enforcement + PreToolUse hook for `bd close` gating

**Feb 25-27, 2026:** Cross-repo support + verification levels
- BEADS_DIR env var injection in Claude CLI spawns for cross-repo phase reporting
- Account isolation: unset `CLAUDE_CODE_OAUTH_TOKEN` + set `CLAUDE_CONFIG_DIR` for non-default accounts
- V0-V3 verification levels replace binary tier (model → claude backend is unchanged)
- Agreements gate added to spawn pipeline (non-blocking warning-only)
- GatherSpawnContext signature extended with `orientationFrame` parameter

**Feb 27-28, 2026:** Safety gates + environment isolation
- `--reason` flag required for safety-override flags (`--bypass-triage`, `--force-hotspot`, `--no-track`); min 10 chars, events.jsonl persistence
- Concurrency gate now counts only running agents (idle excluded) and includes tmux agents (Claude CLI backend)
- `--max-agents 0` means unlimited; flag sentinel changed to -1
- `CLAUDE_CONTEXT` env var explicitly set on all spawn paths (worker/orchestrator/meta-orchestrator)
  - Fixed bug where OpenCode backend spawns inherited parent CLAUDE_CONTEXT, triggering wrong hooks

---

## References

**Investigations:**
- `.kb/investigations/2026-01-08-inv-opus-auth-gate-fingerprinting.md` - Initial discovery of auth gate, failed spoofing attempt, zombie agents
- `.kb/investigations/2026-01-09-debug-gemini-flash-rate-limiting.md` - Gemini Flash TPM limits (2,000 req/min), forced switch to Sonnet
- `.kb/investigations/2026-01-09-inv-anthropic-oauth-community-workarounds.md` - Community bypass attempts, cat-and-mouse dynamics, 474+ GitHub comments
- `.kb/investigations/2026-01-10-inv-fix-dual-mode-spawn-bug.md` - Backend flag implementation and naming fix
- `.kb/investigations/2026-01-11-inv-add-infrastructure-work-detection-auto.md` - Keyword detection and auto-flag application
- `.kb/investigations/2026-01-12-inv-sonnet-cost-tracking-requirements.md` - Cost visibility gap, tracking requirements, strategic questions blocked

**Decisions informed by this model:**
- Dual spawn backends (Claude CLI primary + OpenCode multi-model)
- Infrastructure work uses Claude CLI by default (independent of OpenCode)
- Opus access requires Max subscription + Claude CLI
- Infrastructure detection auto-applies Claude CLI backend

**Related models:**
- `.kb/models/spawn-architecture/model.md` - Full spawn pipeline and workspace lifecycle
- `.kb/models/agent-lifecycle-state-model/model.md` - How status is calculated

**Primary Evidence (Verify These):**
- `pkg/spawn/resolve.go:resolveBackend()` - Backend selection 6-level precedence (~55 lines)
- `pkg/spawn/resolve.go:Resolve()` - Central settings resolution entry point (~110 lines)
- `pkg/spawn/resolve.go:resolveAccount()` - Capacity-aware account routing (~75 lines)
- `pkg/spawn/resolve.go:validateModel()` - Flash blocking, model compatibility
- `pkg/spawn/resolve.go:modelBackendRequirement()` - Model→backend mapping
- `cmd/orch/spawn_cmd.go:isInfrastructureWork()` - Keyword detection logic (22 keywords)
- `cmd/orch/spawn_cmd.go` - `--reason` flag validation for safety-override flags (~952 lines total)
- `pkg/orch/extraction.go:ResolveSpawnSettings()` - Resolve wrapper with logging (~1619 lines total)
- `pkg/orch/spawn_modes.go:DispatchSpawn()` - Mode routing (inline/headless/tmux/claude) (~530 lines)
- `pkg/spawn/claude.go:BuildClaudeLaunchCommand()` - Claude CLI with account isolation + BEADS_DIR injection (~165 lines)
- `pkg/spawn/config.go:ClaudeContext()` - CLAUDE_CONTEXT env var resolution (worker/orchestrator/meta-orchestrator)
- `pkg/spawn/gates/concurrency.go:CheckConcurrency()` - Concurrency gate with tmux agent counting (~198 lines)
- `pkg/model/model.go` - Model aliases and default model definition (~167 lines)
- `CLAUDE.md` - Dual spawn mode documentation

**Cost evidence:**
- Claude Max: $200/mo flat (unlimited Opus via CLI) - Now default path
- Anthropic API via OpenCode: Blocked by default (OAuth ban Feb 19, 2026)
- Non-Anthropic API: Pay-per-token via OpenCode (OpenAI, Google, DeepSeek)
- Flash models: Blocked entirely (not a cost issue, reliability issue)

**Failure evidence:**
- Zombie agents: orch-go-mo0ja, orch-go-pzi2i, orch-go-aoei0 (Jan 8)
- Header injection broke Gemini spawns (Jan 8, reverted)
- OpenCode crash killed infrastructure agent (Jan 11, before auto-detection)
- GPT-5 alias zombie sessions (Feb 2026, fixed by remapping to gpt-5.2)
