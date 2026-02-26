# Model: Spawn Architecture

**Domain:** Agent Spawning / Workspace Creation
**Last Updated:** 2026-02-25
**Synthesized From:** 36 investigations (Dec 2025 - Jan 2026) into spawn implementation, context generation, tier system, and triage friction. Updated Feb 2026 via drift probes and model drift agent.

---

## Summary (30 seconds)

Spawn evolved through 7 phases from basic CLI integration to a modular, gate-driven architecture with capacity-aware account routing. The architecture creates a workspace with SPAWN_CONTEXT.md embedding skill content + task description + kb context, then launches a session via two-phase atomic spawn with rollback. Spawn settings are resolved via `pkg/spawn/resolve.go` with 6-level precedence and per-setting provenance tracking. The spawn pipeline is split across three packages: `pkg/spawn/` (config, resolution, context generation), `pkg/spawn/gates/` (pre-spawn validation), `pkg/spawn/backends/` (backend abstraction), and `pkg/orch/` (pipeline orchestration and mode dispatch). The tier system (light/full) determines whether SYNTHESIS.md is required at completion. Claude CLI is the default backend since Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026).

---

## Core Mechanism

### The Spawn Flow

```
orch spawn <skill> "task"
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. SETTINGS RESOLUTION (pkg/spawn/resolve.go)                  │
│     Resolve backend, model, tier, spawn mode, MCP, mode, etc.  │
│     Precedence: CLI > beads labels > project config >           │
│                 user config > heuristics > defaults              │
│     Each setting tracked with SettingSource provenance          │
│     Model-aware backend routing: Anthropic→claude, others→OC   │
│     Account routing: capacity-aware primary/spillover           │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. SKILL RESOLUTION (pkg/skills/loader.go)                     │
│     Load ~/.claude/skills/{category}/{skill}/SKILL.md           │
│     Load dependencies (e.g., worker-base)                       │
│     Extract phases, constraints, requirements                   │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. SPAWN GATES (pkg/spawn/gates/)                              │
│     Hotspot check: block spawns targeting CRITICAL files        │
│     Triage gate: require --bypass-triage for manual spawns      │
│     Rate limit gate: check account capacity                     │
│     Concurrency gate: limit concurrent agents                   │
│     Verification gate: check verification requirements          │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. BEADS ISSUE CREATION (unless --no-track)                    │
│     bd create "{task}" --type {inferred-from-skill}             │
│     Returns beads ID (e.g., orch-go-abc1)                       │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. KB CONTEXT GATHERING (pkg/spawn/kbcontext.go)               │
│     kb context "{task keywords}" --global                       │
│     Finds relevant constraints, decisions, investigations       │
│     Gap analysis scores context quality (pkg/spawn/gap.go)      │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  6. ATOMIC SPAWN PHASE 1 (pkg/spawn/atomic.go)                  │
│     Tag beads issue with orch:agent label (via beads socket)    │
│     Create workspace + AGENT_MANIFEST.json + dotfiles           │
│     (Rollback all writes on failure)                            │
│                                                                  │
│     .orch/workspace/{name}/                                     │
│     ├── SPAWN_CONTEXT.md      (skill + task + context)          │
│     ├── AGENT_MANIFEST.json   (canonical agent identity)        │
│     ├── .tier                 (light/full)                      │
│     ├── .beads_id             (beads issue ID)                  │
│     ├── .spawn_time           (timestamp)                       │
│     └── .spawn_mode           (headless/tmux/claude)            │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  7. MODE DISPATCH (pkg/orch/spawn_modes.go)                     │
│     Claude:   SpawnClaude() → tmux + Claude CLI                 │
│     Headless: OpenCode HTTP API (startHeadlessSession)          │
│     Tmux:     OpenCode TUI in tmux window                       │
│     Inline:   OpenCode TUI blocking in current terminal         │
│     Default: Claude backend → tmux; OpenCode backend → headless │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  8. ATOMIC SPAWN PHASE 2 (pkg/spawn/atomic.go)                  │
│     Write .session_id                                           │
│     Update AGENT_MANIFEST.json with session ID                  │
│     (Best-effort: session already running)                      │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

**SPAWN_CONTEXT.md structure:**
```markdown
# TASK: {task description}

# SKILL CONTEXT:
{full SKILL.md content embedded}

# BEADS ISSUE:
{issue details if --issue provided}

# KB CONTEXT:
{relevant constraints, decisions, investigations}

# DELIVERABLES:
- {workspace}/SYNTHESIS.md (if full tier)
- Git commits with changes
```

**Workspace metadata (canonical: AGENT_MANIFEST.json):**
- `AGENT_MANIFEST.json` - Canonical source of agent identity and spawn-time context
- Read path: `ReadAgentManifestWithFallback()` → OpenCode session metadata → AGENT_MANIFEST.json → dotfiles (legacy)

**Legacy dotfiles (still written for backward compatibility):**
- `.tier` - "light" or "full" (determines SYNTHESIS.md requirement)
- `.session_id` - OpenCode session ID for `orch send`
- `.beads_id` - Issue tracking ID for `orch complete`
- `.spawn_time` - Timestamp for age calculations
- `.spawn_mode` - Which spawn backend was used

### State Transitions

**Normal spawn lifecycle:**

```
Command invoked (orch spawn)
    ↓
Settings resolved (backend, model, tier, spawn mode)
    ↓
Skill loaded + beads issue created
    ↓
KB context gathered
    ↓
Atomic Phase 1: Tag beads + write workspace (rollback on failure)
    ↓
Session created (OpenCode API or Claude CLI)
    ↓
Atomic Phase 2: Write session ID + update manifest
    ↓
Agent starts working
```

**Cross-project spawn (fixed Feb 25, 2026):**

```
cd ~/orchestrator-project
    ↓
orch spawn --workdir ~/target-project investigation "task"
    ↓
Workspace created in: ~/target-project/.orch/workspace/
Beads DefaultDir set to: ~/orchestrator-project/.beads/
projectDir threaded through kb context for correct resolution
Agent works in: ~/target-project/
```

### Critical Invariants

1. **Workspace name = kebab-case task description** - Used for tmux window, directory name, session title
2. **Beads ID required for phase reporting** - `--no-track` creates untracked IDs that can't report to beads
3. **KB context uses --global flag** - Cross-repo constraints are essential
4. **Skill content stripped for --no-track** - Beads instructions removed when not tracking
5. **Session scoping is per-project** - `orch send` only works within same directory hash
6. **Token estimation at 4 chars/token** - Warning at 100k, error at 150k
7. **Model-aware backend routing** - Backend determined by model provider unless CLI overrides (Decision: kb-2d62ef)
8. **Claude backend implies tmux** - Claude CLI physically requires tmux window; headless + claude auto-switches to tmux
9. **Account routing is capacity-aware** - Primary accounts checked first; spillover activated when primaries exhausted (>20% threshold)

---

## Why This Fails

### Failure Mode 1: Cross-Project Spawn Sets Wrong Session Directory

**Symptom:** `orch spawn --workdir /other/project` creates session with orchestrator's directory

**Root cause:** Session directory is set from spawn caller's CWD, not `--workdir` target

**Why it happens:**
- OpenCode infers directory from process CWD
- `--workdir` changes agent's working directory, not spawn process CWD
- Session gets orchestrator directory, beads issue in orchestrator project

**Impact:**
- Sessions unfindable via directory filtering
- Cross-project work tracking is split

**Fix needed:** Pass explicit directory to OpenCode session creation

### Failure Mode 2: Token Limit Exceeded on Spawn

**Symptom:** Spawn fails with "context too large" error

**Root cause:** SPAWN_CONTEXT.md exceeds 150k token limit

**Why it happens:**
- Skill content (~10-40k tokens)
- KB context can be large (30-50k tokens)
- Task description minimal
- Estimation: 4 chars/token

**Fix (Dec 22):** Warning at 100k tokens, hard error at 150k with guidance

### Failure Mode 3: Daemon Spawns Blocked Issues

**Symptom:** Daemon spawns issue that has blockers

**Root cause:** Dependency checking missing in triage workflow

**Why it happens:**
- `bd ready` returns issues without blockers
- Daemon spawns from `triage:ready` label (doesn't check dependencies)
- Race condition: issue labeled before dependencies checked

**Fix (Jan 3):** Dependency gating with `--force` override flag

---

## Constraints

### Why Can't We Infer Skill from Task Description?

**Constraint:** Natural language is ambiguous - "fix bug" could be systematic-debugging or feature-impl

**Implication:** Must explicitly specify skill in spawn command

**Workaround:** Daemon infers skill from beads issue type

**This enables:** Precise skill selection for complex tasks
**This constrains:** Manual spawns require explicit skill argument

### Why Require --bypass-triage for Manual Spawns?

**Constraint:** Design choice to make manual spawns intentionally harder

**Implication:** Friction encourages daemon-driven workflow

**Workaround:** Use `--bypass-triage` flag for urgent/exceptional spawns

**This enables:** Scalable automation via daemon
**This constrains:** Ad-hoc spawning is discouraged

### Why Two-Tier System Instead of Always Requiring SYNTHESIS.md?

**Constraint:** Light work (bug fixes, small features) doesn't need full synthesis

**Implication:** Tier determines completion requirements

**Workaround:** Skills set default tier, `--tier` flag overrides

**This enables:** Appropriate documentation for work complexity
**This constrains:** Must decide tier at spawn (can't change mid-flight)

---

## Evolution

**Phase 1: Initial Implementation (Dec 19, 2025)**
- CLI command structure with Cobra
- Skill loading from `~/.claude/skills/`
- SPAWN_CONTEXT.md template generation
- Beads integration for tracking

**Phase 2: Tmux Visual Mode (Dec 20-21, 2025)**
- Per-project workers sessions (`workers-orch-go`)
- Window naming with skill emojis
- `opencode attach` for TUI + API dual access
- Readiness detection via pane content polling

**Phase 3: Headless Default (Dec 22, 2025)**
- Flipped default from tmux to headless (HTTP API)
- `--tmux` became opt-in
- Enabled daemon automation
- SSE monitoring via `orch monitor`

**Phase 4: Tiered Completion (Dec 22, 2025)**
- Light tier for implementation (no SYNTHESIS.md required)
- Full tier for knowledge work (SYNTHESIS.md required)
- Skill-based defaults
- `.tier` file in workspace

**Phase 5: Triage Friction (Jan 3-6, 2026)**
- Dependency gating (`--force` to override)
- `--bypass-triage` flag to discourage manual spawns
- Daemon-driven workflow as default
- Event logging for bypass analysis

**Phase 6: Atomic Spawn + Resolved Settings (Jan-Feb 2026)**
- Registry removed; AGENT_MANIFEST.json replaces dotfiles as canonical metadata
- `pkg/spawn/resolve.go` centralizes all settings resolution with provenance
- Two-phase atomic spawn with rollback on failure (`pkg/spawn/atomic.go`)
- `--backend claude` implies tmux spawn mode (derived setting)
- Flash models blocked entirely at resolve layer
- Context file variants: SPAWN_CONTEXT.md, ORCHESTRATOR_CONTEXT.md, META_ORCHESTRATOR_CONTEXT.md
- Hotspot gating blocks spawns targeting CRITICAL files (>1500 lines)

**Phase 7: Modular Extraction + Account Distribution (Feb 2026)**
- Extracted `pkg/orch/spawn_modes.go` + `pkg/orch/spawn_helpers.go` from `extraction.go` (-644 lines)
- New `pkg/spawn/gates/` subdirectory: hotspot, triage, ratelimit, concurrency, verification gates
- New `pkg/spawn/backends/` subdirectory: backend interface + common/headless/inline/tmux implementations
- Account distribution with capacity-aware routing (3 phases: schema+CLI+env → cache+heuristic → logging)
- `resolveAccount()` routes between primary/spillover accounts based on capacity fetcher
- Cross-project spawn fixes: `beads.DefaultDir` set correctly, `projectDir` threaded through kb context
- Bug-type issues now route to `systematic-debugging` skill (was `architect`)
- `--force-hotspot` requires `--architect-ref` with verified closed architect issue
- `--disallowedTools` enforcement + PreToolUse hook for `bd close` gating
- Claude CLI became default backend (Anthropic banned subscription OAuth in third-party tools Feb 19)
- Pre-create session for tmux spawns with non-default models
- GPT-5 alias remapped to `gpt-5.2` to prevent zombie sessions

---

## References

**Key Investigations:**
- `2025-12-19-inv-spawn-agent-tmux-implementation.md` - Initial tmux implementation
- `2025-12-22-inv-flip-default-spawn-mode-headless.md` - Headless as default
- `2025-12-22-inv-implement-tiered-spawn-protocol.md` - Tier system design
- `2026-01-03-inv-spawn-dependency-gating-design.md` - Dependency checking
- `2026-01-06-inv-add-bypass-triage-friction-manual.md` - Triage friction
- ...and 31 others

**Decisions Informed by This Model:**
- Headless default (enables daemon automation)
- Tier system (appropriate docs for work complexity)
- Triage friction (encourage daemon workflow)
- KB context gathering (prevent duplicate work)

**Related Models:**
- `.kb/models/model-access-spawn-paths/model.md` - Model selection, backend routing, escape hatch
- `.kb/models/agent-lifecycle-state-model/model.md` - How spawned agents' status is calculated

**Related Guides:**
- `.kb/guides/spawn.md` - How to use spawn command (procedural)
- `.kb/guides/daemon.md` - How daemon auto-spawns (procedural)

**Primary Evidence (Verify These):**
- `cmd/orch/spawn_cmd.go` - Main spawn command + infrastructure detection (~876 lines)
- `pkg/orch/extraction.go` - Spawn pipeline types and functions (~1437 lines)
- `pkg/orch/spawn_modes.go` - Mode dispatch: inline/headless/tmux/claude (~529 lines)
- `pkg/orch/spawn_helpers.go` - Helper utilities for spawn pipeline (~148 lines)
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md generation (~1200 lines)
- `pkg/spawn/kbcontext.go` - KB context gathering and formatting (~1100 lines)
- `pkg/spawn/config.go` - Config struct, tier defaults, skill mappings (~460 lines)
- `pkg/spawn/resolve.go` - Settings resolution with 6-level precedence, account routing (~580 lines)
- `pkg/spawn/atomic.go` - Two-phase atomic spawn with rollback (~114 lines)
- `pkg/spawn/claude.go` - Claude CLI backend (tmux spawn, MCP wiring) (~172 lines)
- `pkg/spawn/gap.go` - Context gap analysis and quality scoring
- `pkg/spawn/session.go` - Session management, AGENT_MANIFEST.json read/write
- `pkg/spawn/gates/` - Pre-spawn validation gates (hotspot, triage, ratelimit, concurrency, verification)
- `pkg/spawn/backends/` - Backend abstraction layer (backend interface, common, headless, inline, tmux)
- `pkg/skills/loader.go` - Skill discovery, loading, dependency composition
