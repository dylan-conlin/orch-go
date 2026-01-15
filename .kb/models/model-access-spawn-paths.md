# Model: Model Access and Spawn Paths

**Domain:** Agent Spawning / Model Selection
**Last Updated:** 2026-01-12
**Synthesized From:** 5 investigations (Opus gate, Gemini TPM limits, community workarounds, cost tracking, escape hatch implementations) spanning Jan 8-12, 2026

---

## Summary (30 seconds)

Anthropic restricts Opus 4.5 access via fingerprinting that blocks API usage but allows Claude Code CLI with Max subscription. This constraint forced a **dual spawn architecture**: primary path (OpenCode API + Sonnet/Flash, headless, high concurrency) and escape hatch (Claude CLI + Opus, tmux, crash-resistant). The escape hatch exists because critical infrastructure work (fixing the spawn system itself) can't depend on what might fail. Model choice now encodes reliability requirements, not just quality preferences.

---

## Core Mechanism

### Available Models and Access Methods

**Anthropic Models:**
- **Opus 4.5** (`claude-opus-4-5-20251101`) - Highest quality, restricted access
- **Sonnet 4.5** (`claude-sonnet-4-5-20250929`) - Balanced quality/speed
- **Haiku** - Fast, lower cost

**Gemini Models:**
- **Flash 3** (`gemini-3-flash-preview`) - Fast, cheap, but 2,000 req/min TPM limit (Paid Tier 2)
- **Pro** - Higher quality Gemini option

### Access Patterns

**Pattern 1: OpenCode API (Primary Path)**
```
User → orch spawn → OpenCode HTTP API (localhost:4096) → Anthropic/Gemini API
```

**Characteristics:**
- Headless (no UI, returns immediately)
- High concurrency (5+ agents simultaneously)
- Dashboard visibility via SSE
- Pay-per-token pricing (unknown current spend, switched to Sonnet Jan 9)
- **Constraint:** Cannot use Opus (fingerprinting blocks it)
- **Constraint:** Gemini Flash has 2,000 req/min TPM limit (tool-heavy agents hit it)
- **Dependency:** OpenCode server must be running

**Pattern 2: Claude CLI (Escape Hatch)**
```
User → orch spawn --backend claude → claude CLI fork → Anthropic API (with fingerprint)
```

**Characteristics:**
- Tmux window (visual progress)
- Lower concurrency (manual tracking)
- Opus 4.5 access (Max subscription required)
- Flat $200/mo (unlimited usage)
- **Independence:** Survives OpenCode server crashes
- **Trade-off:** No dashboard visibility, manual lifecycle

### Key Components

**Backend Selection Priority (pkg/spawn/config.go):**
```
1. Explicit --backend flag (highest priority)
2. Auto-apply for infrastructure work (keywords detected)
   → Sets --backend claude --tmux automatically
3. Model-based auto-selection
   → opus/sonnet → opencode
   → flash/pro → opencode
4. Default: opencode
```

**Infrastructure Work Detection:**
- Scans task description + beads issue for keywords
- Keywords: "opencode", "spawn", "daemon", "registry", "orch serve", "overmind", "dashboard"
- Auto-applies `--backend claude --tmux` flags
- Prevents agents from killing themselves (e.g., restarting OpenCode during spawn)

### State Transitions

**Normal spawn (OpenCode):**
```
orch spawn feature-impl "task"
    ↓
Backend: opencode (default)
    ↓
Model: claude-sonnet-4-5 (default since Jan 9, was gemini-3-flash before TPM limits)
    ↓
Headless session via HTTP API
    ↓
Dashboard visibility
```

**Infrastructure spawn (auto-detected):**
```
orch spawn systematic-debugging "fix opencode server crash"
    ↓
Keyword detected: "opencode"
    ↓
Auto-apply: --backend claude --tmux
    ↓
Tmux session via Claude CLI
    ↓
Survives OpenCode server restart
```

**Explicit escape hatch:**
```
orch spawn --backend claude --model opus architect "complex design"
    ↓
Backend: claude (explicit override)
    ↓
Model: opus (Max subscription)
    ↓
Tmux session, highest quality
```

### Critical Invariants

1. **Never spawn OpenCode infrastructure work without --backend claude --tmux**
   - Violation: Agent kills itself mid-execution when server restarts

2. **Infrastructure detection runs before model auto-selection**
   - Priority 2.5 (between explicit flags and model-based selection)
   - Ensures auto-apply happens even without explicit --backend

3. **Opus only accessible via Claude CLI backend**
   - API requests to Opus fail with auth error
   - Fingerprinting checks more than headers (TLS, HTTP/2 frames, ordering)

4. **Escape hatch provides true independence**
   - Claude CLI binary ≠ OpenCode server
   - Tmux session persists across service restarts
   - Different authentication path (Max subscription OAuth)

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

**Strategic question enabled:** "Should we shift more work to escape hatch to optimize cost?"

**Current answer:** No - headless primary path provides better ergonomics for most work. Reserve escape hatch for critical infrastructure.

**This enables:** Cost-effective high-concurrency spawning via API path
**This constrains:** Escape hatch limited to critical work due to ergonomic overhead

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
- Dual spawn architecture (primary + escape hatch)
- Never spawn infrastructure work via OpenCode
- Opus access requires Max subscription + Claude CLI
- Infrastructure detection auto-applies escape hatch flags

**Related models:**
- `.kb/models/dashboard-agent-status.md` - How status is calculated (relates to session state constraint)

**Primary Evidence (Verify These):**
- `pkg/spawn/config.go:selectBackend()` - Backend selection priority cascade
- `pkg/spawn/config.go:detectInfrastructureWork()` - Keyword detection logic
- `CLAUDE.md` lines 130-170 - Dual spawn mode documentation
- `~/.claude/skills/meta/orchestrator/SKILL.md` line 625 - "Why escape hatch exists" section

**Cost evidence:**
- Claude Max: $200/mo flat (unlimited Opus via CLI)
- Anthropic API: Unknown current spend (switched to Sonnet Jan 9, no tracking)
- Gemini API: Free via AI Studio (but 2,000 req/min limit hit)
- Gemini Tier 3: Pending (20,000 req/min, would enable Flash as primary)

**Failure evidence:**
- Zombie agents: orch-go-mo0ja, orch-go-pzi2i, orch-go-aoei0 (Jan 8)
- Header injection broke Gemini spawns (Jan 8, reverted)
- OpenCode crash killed infrastructure agent (Jan 11, before auto-detection)
