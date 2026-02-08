# Model: Model Access and Spawn Paths

**Domain:** Agent Spawning / Model Selection
**Last Updated:** 2026-01-28
**Synthesized From:** 8 investigations + ~70 kb quick entries (Opus gate, Gemini TPM limits, community workarounds, cost tracking, escape hatch implementations, Docker backend design, Docker container constraints, API cost discovery, GPT-5.2 orchestration test) spanning Jan 8-22, 2026

---

## Summary (30 seconds)

Anthropic restricts Opus 4.5 access via fingerprinting that blocks API usage but allows Claude Code CLI with Max subscription. After discovering API costs hit $70-80/day ($2,100-2,400/mo projected), the **recommended primary path** became Claude CLI + Opus ($200/mo flat). The **triple spawn architecture** provides three backends: Claude CLI (Opus, tmux, quality + cost-effective), OpenCode API (Sonnet/DeepSeek, headless, cost tracking), and Docker (fresh fingerprint, rate limit bypass). Default backend is configured via `~/.orch/config.yaml` → `backend:` field. GPT-5.2 tested Jan 21 and deemed unsuitable for orchestration (role boundary collapse, reactive gate handling). Model choice encodes cost, quality, and reliability requirements.

---

## Core Mechanism

### Available Models and Access Methods

**Anthropic Models:**
- **Opus 4.5** (`claude-opus-4-5-20251101`) - Highest quality, primary model for orchestration
- **Sonnet 4.5** (`claude-sonnet-4-5-20250929`) - Balanced quality/speed, API fallback
- **Haiku** - Fast, lower cost

**DeepSeek Models:**
- **DeepSeek V3** (`deepseek/deepseek-chat`) - Cost-effective ($0.25/$0.38/MTok), function calling confirmed working Jan 19

**Gemini Models:**
- **Flash 3** (`gemini-3-flash-preview`) - Fast, cheap, but 2,000 req/min TPM limit (Paid Tier 2)
- **Pro** - Higher quality Gemini option

**OpenAI Models:**
- **GPT-5.2** - Available via ChatGPT Pro subscription, **unsuitable for orchestration** (Jan 21 decision)
- **GPT-5.2 Codex** - Optimized for agentic coding, worker tasks only

### Access Patterns

**Pattern 1: Claude CLI (Recommended Primary)**
```
User → orch spawn → claude CLI → Anthropic API (with fingerprint)
```

**Characteristics:**
- Tmux window (visual progress)
- Opus 4.5 access (Max subscription required)
- Flat $200/mo unlimited - **10x cheaper than API at heavy usage**
- Best model quality for orchestration
- **Trade-off:** No dashboard visibility, manual lifecycle
- **Independence:** Doesn't depend on OpenCode server

**Why this became primary (Jan 18):**
- API costs discovered: $402 in ~2 weeks, ramping to $70-80/day
- Projected API spend: $2,100-2,400/mo vs $200/mo flat
- Opus quality only available via CLI (fingerprinting blocks API)

**Pattern 2: OpenCode API (Secondary Path - Opt-in)**
```
User → orch spawn --backend opencode → OpenCode HTTP API (localhost:4096) → Anthropic/DeepSeek/Gemini API
```

**Characteristics:**
- Headless (no UI, returns immediately)
- High concurrency (5+ agents simultaneously)
- Dashboard visibility via SSE
- Pay-per-token pricing (cost tracking recommended)
- **Constraint:** Cannot use Opus (fingerprinting blocks it)
- **Constraint:** Gemini Flash has 2,000 req/min TPM limit (tool-heavy agents hit it)
- **Dependency:** OpenCode server must be running

**When to use API path:**
- Cost tracking/metering needed
- DeepSeek V3 for cost-sensitive work ($0.25/$0.38/MTok)
- Dashboard visibility required
- Headless batch processing

**Pattern 3: Docker (Double Escape Hatch)**
```
User → orch spawn --backend docker → host tmux window → docker run claude-code-mcp → claude CLI (fresh fingerprint)
```

**Characteristics:**
- Host tmux window running Docker container
- Fresh Statsig fingerprint per spawn (rate limit isolation)
- Uses `~/.claude-docker/` for config (separate from host `~/.claude/`)
- Same lifecycle as claude mode (status, complete, abandon via tmux)
- **Independence:** Bypasses host fingerprint rate limits (device-level throttling)
- **Trade-off:** No dashboard visibility, ~2-5s container startup overhead
- **Prerequisite:** Docker image `claude-code-mcp` built from `~/.claude/docker-workaround/`

**Environment Constraints:**
- `BEADS_NO_DAEMON=1` auto-set (Unix sockets fail with "chmod: invalid argument" over mounts)
- Container PATH includes `/usr/local/go/bin` for auto-rebuild
- Real configs (CLAUDE.md, settings.json, skills/, hooks/) mounted read-only after base `~/.claude-docker/` overlay

**Rate Limit Clarification:**
- Docker bypasses **request-rate throttling** (per-device limits)
- Docker does NOT bypass **weekly usage quota** (account-level, e.g., "97% used")
- Tested: Wiped ~/.claude-docker/, logged in fresh - usage charged to correct account

### Key Components

**Backend Selection Priority (cmd/orch/backend.go):**
```
1. Explicit --backend flag (claude, opencode, or docker)
2. --opus flag (implies claude backend)
3. Project config (.orch/config.yaml spawn_mode)
4. Global config (~/.orch/config.yaml backend)  ← CHECK THIS FOR CURRENT DEFAULT
5. Code default: opencode (fallback if no config)
```

**Current default:** See `~/.orch/config.yaml` → `backend:` field

**Note:** Infrastructure work detection is advisory-only (warns, doesn't override). Docker backend must be explicitly requested via `--backend docker`.

**Infrastructure Work Detection:**
- Scans task description + beads issue for keywords
- Keywords: "opencode", "spawn", "daemon", "registry", "orch serve", "overmind", "dashboard"
- Auto-applies `--backend claude --tmux` flags
- Prevents agents from killing themselves (e.g., restarting OpenCode during spawn)

### State Transitions

**Normal spawn (uses configured default):**
```
orch spawn feature-impl "task"
    ↓
Backend: from ~/.orch/config.yaml (or code default: opencode)
    ↓
Model: depends on backend (opus for claude, configurable for opencode)
    ↓
Session type depends on backend
```

**API spawn (opt-in for cost tracking/headless):**
```
orch spawn --backend opencode --model sonnet feature-impl "task"
    ↓
Backend: opencode (explicit)
    ↓
Model: sonnet (or deepseek for cost-sensitive)
    ↓
Headless session via HTTP API
    ↓
Dashboard visibility, pay-per-token
```

**Infrastructure spawn (auto-detected):**
```
orch spawn systematic-debugging "fix opencode server crash"
    ↓
Keyword detected: "opencode"
    ↓
Advisory warning (doesn't override, since claude is already default)
    ↓
Tmux session via Claude CLI
    ↓
Survives OpenCode server restart
```

**Docker escape hatch (rate limit bypass):**
```
orch spawn --backend docker investigation "explore X"
    ↓
Backend: docker (explicit override)
    ↓
Host tmux window created
    ↓
docker run claude-code-mcp
    ↓
Fresh Statsig fingerprint, rate limit isolated
```

### Critical Invariants

1. **Never spawn OpenCode infrastructure work without --backend claude --tmux**
   - Violation: Agent kills itself mid-execution when server restarts

2. **Infrastructure detection is advisory-only**
   - Warns when critical infrastructure keywords detected
   - Never auto-overrides backend selection
   - User must explicitly choose escape hatch

3. **Opus only accessible via Claude CLI backend**
   - API requests to Opus fail with auth error
   - Fingerprinting checks more than headers (TLS, HTTP/2 frames, ordering)

4. **Escape hatch provides true independence**
   - Claude CLI binary ≠ OpenCode server
   - Tmux session persists across service restarts
   - Different authentication path (Max subscription OAuth)

5. **Docker backend requires explicit opt-in**
   - Must use `--backend docker` flag
   - Docker image `claude-code-mcp` must be pre-built
   - Uses separate config directory (`~/.claude-docker/`) for fingerprint isolation

6. **Docker containers have environment constraints**
   - `BEADS_NO_DAEMON=1` must be set (Unix sockets don't work over Docker mounts)
   - Container PATH must include `/usr/local/go/bin` for auto-rebuild
   - Real configs (CLAUDE.md, settings.json, skills/, hooks/) mounted read-only to override `~/.claude-docker/` overlay

7. **Weekly usage quota is account-level, not device-level**
   - Docker fingerprint isolation bypasses device-level rate throttling only
   - Weekly usage quota (e.g., "97% used") is tied to account, not fingerprint
   - Copying statsig fingerprint to Docker doesn't bypass weekly limits

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
2. Using Claude CLI with Max subscription (current escape hatch)
3. Accepting Sonnet/Flash as primary models

**Strategic question enabled:** "Is Opus quality worth $200/mo flat cost vs pay-per-token Sonnet/Flash?"

**This enables:** Anthropic to differentiate Claude Code from API access (product strategy)
**This constrains:** Cannot use Opus via API without Max subscription + Claude CLI

### Constraint 2: Critical Paths Need Independence

**Why escape hatch exists:**

When building infrastructure the primary path depends on, failure cascades:
- Fixing OpenCode → spawned via OpenCode → fix restarts server → agent dies → fix incomplete
- Debugging spawn system → spawned via spawn system → meta-circular trap

**Architectural principle:** Critical paths require secondary mechanisms that don't depend on what can fail

**Trade-offs accepted:**
- Escape hatch has less automation (no dashboard)
- Lower concurrency (manual tmux tracking)
- Flat cost model (Max subscription)

**When this matters:**
- Building/fixing orch-go spawn system
- Debugging OpenCode server crashes
- Dashboard/monitoring infrastructure work
- Daemon implementation

**This enables:** Infrastructure work to complete even when primary path fails
**This constrains:** Must maintain two spawn paths (complexity cost)

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

### Constraint 4: Cost Model Determines Primary Path

**The Jan 18 Discovery:**
- API costs hit $402 in ~2 weeks without awareness
- Ramping to $70-80/day ($2,100-2,400/mo projected)
- Max subscription at $200/mo is **10x cheaper** at heavy usage

**Claude CLI (Max subscription) - NOW PRIMARY:**
- Flat $200/mo unlimited
- Best model quality (Opus)
- **Free cache reads** (vs 10% on API) - massive for tool-heavy agentic work
- Trade-off: No dashboard visibility, tmux management

**OpenCode API (pay-per-token) - NOW SECONDARY:**
- Variable cost scales with usage
- Dashboard visibility, headless operation
- DeepSeek V3: $0.25/$0.38/MTok (cost-effective option)
- Sonnet: $3/$15/MTok (quality fallback)

**Strategic question answered:** "Should we shift more work to Max subscription?"

**Current answer (Jan 18):** YES - Claude CLI is now default. API path is opt-in for cost tracking, headless batch work, or specific model needs.

**This enables:** Predictable $200/mo cost with highest model quality
**This constrains:** Lose dashboard visibility for primary path (accepted trade-off)

### Constraint 5: Gemini Flash TPM Limits Block Tool-Heavy Agents

**Why we can't use Gemini Flash as default:**

Google imposes 2,000 requests/minute limit on Gemini Flash 3 (Paid Tier 2):
- Tool-heavy agents (investigation, systematic-debugging) make 35+ tool calls/second
- Each tool use (Read, Grep, Bash, etc.) = one API request
- Single agent can hit 2,000 req/min limit
- Retry logic slows spawns to crawl

**Evidence:** Investigation `2026-01-09-debug-gemini-flash-rate-limiting.md`

**Implication:** Forced switch to Sonnet on Jan 9, 2026
- Lost "free" model (Gemini via AI Studio)
- Gained reliability (no rate limit throttling)
- Lost cost visibility (no tracking of Sonnet spend)
- Created new constraint: unknown budget trajectory

**Workarounds attempted:**
1. Apply for Tier 3 (20,000 req/min) - Status: Pending
2. Use Sonnet instead - Status: Current solution
3. Tolerate retry delays - Status: Unacceptable for production

**Strategic question enabled:** "Should we invest in Tier 3 or accept Sonnet costs?"

**This enables:** Google to manage API load across users
**This constrains:** Cannot use Gemini Flash for tool-heavy agents without Tier 3

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

### Constraint 8: GPT-5.2 Unsuitable for Orchestration

**Why we can't use GPT-5.2 for orchestrators:**

Tested GPT-5.2 as orchestrator on Jan 21 (session ses_4207). Five critical anti-patterns emerged:

| Pattern | GPT-5.2 Behavior | Expected (Opus) Behavior |
|---------|-----------------|-------------------------|
| Gate handling | Reactive (hit → fix → repeat) | Anticipatory (synthesize all flags upfront) |
| Role boundaries | Collapses to worker mode | Maintains supervision boundary |
| Deliberation | Excessive (200+ second thinking blocks) | Confident, decision-focused |
| Failure recovery | Repeats same pattern (6+ identical failures) | Adapts strategy |
| Instruction synthesis | Literal, sequential | Contextual, synthesized |

**Evidence:**
- 3 spawn attempts required for multi-gate scenario (--bypass-triage, then strategic-first)
- Role boundary collapse: After spawning architect agent, GPT started debugging Docker directly
- 6+ timeout failures without strategy adaptation
- 200+ second thinking blocks revealing internal uncertainty

**Implication:** GPT-5.2 may work for constrained worker tasks but is structurally unsuited for orchestration. This isn't a prompting issue - it's a fundamental behavioral difference.

**This enables:** Clear model selection guidance (Opus for orchestration)
**This constrains:** Cannot use GPT-5.2/ChatGPT Pro subscription for orchestrator agents

**Reference:** `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md`

### Constraint 9: GPT-5.2 Also Unsuitable for Workers (Jan 24, 2026)

**Why we can't use GPT-5.2 for workers either:**

Tested GPT-5.2 and GPT-5.2-codex as workers on Jan 24. Both failed:

| Model | Result | Failure Mode |
|-------|--------|--------------|
| gpt-5.2 | Task completed, then hallucinated | Created 565-line quantum physics simulation after finishing actual task |
| gpt-5.2-codex | Went idle repeatedly | Couldn't navigate spawn context, needed constant nudging |

**Root cause:** Our 662-line spawn context evolved to fit Claude's instruction-following style. It's not that GPT is worse - our prompts are in "Claude dialect."

Evidence: GPT-5.2 works well in Codex CLI (OpenAI-tuned prompts) but fails in our system.

**Implication:** The entire orchestration system is Claude-coupled. Accepted as design constraint rather than problem to solve.

**This enables:** Full optimization for Claude without multi-model compromises
**This constrains:** All workers must be Claude models (Opus, Sonnet, Haiku)

**Reference:** `.kb/decisions/2026-01-24-claude-specific-orchestration-accepted.md`

---

## References

**Evolution history:** `.kb/models/spawn-system-evolution.md` - Timeline of spawn system changes (Jan 8-24, 2026)

**Investigations:**
- `.kb/investigations/archived/2026-01-08-inv-opus-auth-gate-fingerprinting.md` - Initial discovery of auth gate, failed spoofing attempt, zombie agents
- `.kb/investigations/2026-01-09-debug-gemini-flash-rate-limiting.md` - Gemini Flash TPM limits (2,000 req/min), forced switch to Sonnet
- `.kb/investigations/archived/2026-01-09-inv-anthropic-oauth-community-workarounds.md` - Community bypass attempts, cat-and-mouse dynamics, 474+ GitHub comments
- `.kb/investigations/2026-01-10-inv-fix-dual-mode-spawn-bug.md` - Backend flag implementation and naming fix
- `.kb/investigations/2026-01-11-inv-add-infrastructure-work-detection-auto.md` - Keyword detection and auto-flag application
- `.kb/investigations/archived/2026-01-12-inv-sonnet-cost-tracking-requirements.md` - Cost visibility gap, tracking requirements, strategic questions blocked
- `.kb/investigations/2026-01-19-inv-test-deepseek-v3-function-calling.md` - DeepSeek V3 function calling validation
- `.kb/investigations/archived/2026-01-20-inv-design-claude-docker-backend-integration.md` - Docker backend design, host tmux pattern, fingerprint isolation
- `.kb/investigations/archived/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md` - GPT-5.2 orchestration test, anti-patterns discovered

**Decisions informed by this model:**
- Triple spawn architecture (primary Claude CLI + API secondary + Docker escape hatch)
- Claude CLI + Opus as default (Jan 18 cost discovery)
- Never spawn infrastructure work via OpenCode (advisory warning)
- Opus access requires Max subscription + Claude CLI
- Docker provides fingerprint isolation for rate limit scenarios
- GPT-5.2 unsuitable for orchestration (Jan 21)
- GPT-5.2 unsuitable for workers, Claude-specific orchestration accepted (Jan 24)

**Decisions:**
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Claude CLI became default after API cost discovery
- `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md` - GPT-5.2 unsuitable for orchestration
- `.kb/decisions/2026-01-24-claude-specific-orchestration-accepted.md` - Accept Claude-specific system, don't invest in multi-model

**Related models:**
- `.kb/models/dashboard-agent-status.md` - How status is calculated (relates to session state constraint)

**Related guides:**
- `.kb/guides/model-selection.md` - Authoritative model selection reference (quick lookup)

**Primary Evidence (Verify These):**
- `cmd/orch/backend.go:resolveBackend()` - Backend selection priority cascade
- `cmd/orch/backend.go:addInfrastructureWarning()` - Advisory infrastructure detection
- `pkg/spawn/docker.go:SpawnDocker()` - Docker backend implementation
- `CLAUDE.md` lines 130-170 - Triple spawn mode documentation
- `~/.claude/skills/meta/orchestrator/SKILL.md` line 625 - "Why escape hatch exists" section

**Cost evidence:**
- Claude Max: $200/mo flat (unlimited Opus via CLI, free cache reads) - **NOW PRIMARY**
- Anthropic API: $402 spent in ~2 weeks (Jan 9-18), ramping to $70-80/day before switch
- DeepSeek V3 API: $0.25/$0.38/MTok (cost-effective secondary option)
- Gemini API: Free via AI Studio (but 2,000 req/min limit hit)
- ChatGPT Pro: $200/mo flat (GPT-5.2, but unsuitable for orchestration)

**Failure evidence:**
- Zombie agents: orch-go-mo0ja, orch-go-pzi2i, orch-go-aoei0 (Jan 8)
- Header injection broke Gemini spawns (Jan 8, reverted)
- OpenCode crash killed infrastructure agent (Jan 11, before auto-detection)
