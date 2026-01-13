# Model: Model Access and Spawn Paths

**Domain:** Agent Spawning / Model Selection
**Last Updated:** 2026-01-12
**Synthesized From:** 2 investigations into Opus gate + 2 implementations of escape hatch pattern (Jan 8-11, 2026)

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
- **Flash 2.0** (`gemini-2.0-flash-exp`) - Default for headless spawns, fast, cheap
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
- Pay-per-token pricing (~$100-200/mo variable cost)
- **Constraint:** Cannot use Opus (fingerprinting blocks it)
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
Model: gemini-2.0-flash-exp (default)
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

**Jan 11, 2026:** Infrastructure work auto-detection added
- Keyword-based detection (opencode, spawn, daemon, registry, etc.)
- Auto-applies `--backend claude --tmux` at priority 2.5
- Prevents agents from killing themselves
- Makes escape hatch invisible for common cases

**Jan 12, 2026:** Model created from synthesis
- Recognized constraint has system-wide ripple effects
- Escape hatch pattern now embedded in spawn priority logic
- Cost/quality/reliability tradeoffs explicit
- Strategic questions about model usage surfaced

---

## References

**Investigations:**
- `.kb/investigations/2026-01-08-inv-opus-auth-gate-fingerprinting.md` - Initial discovery of auth gate, failed spoofing attempt, zombie agents
- `.kb/investigations/2026-01-10-inv-fix-dual-mode-spawn-bug.md` - Backend flag implementation and naming fix
- `.kb/investigations/2026-01-11-inv-add-infrastructure-work-detection-auto.md` - Keyword detection and auto-flag application

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
- Anthropic API: ~$100-200/mo variable (current usage, no Opus)
- Gemini API: Included in pay-per-token (default model)

**Failure evidence:**
- Zombie agents: orch-go-mo0ja, orch-go-pzi2i, orch-go-aoei0 (Jan 8)
- Header injection broke Gemini spawns (Jan 8, reverted)
- OpenCode crash killed infrastructure agent (Jan 11, before auto-detection)
