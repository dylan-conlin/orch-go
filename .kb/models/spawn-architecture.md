# Model: Spawn Architecture

**Domain:** Agent Spawning / Workspace Creation
**Last Updated:** 2026-01-29
**Synthesized From:** 36 investigations (Dec 2025 - Jan 2026) into spawn implementation, context generation, tier system, and triage friction

---

## Summary (30 seconds)

Spawn evolved through 5 phases from basic CLI integration to daemon-driven automation with triage friction. The architecture creates a workspace with SPAWN_CONTEXT.md embedding skill content + task description + kb context, then launches an OpenCode session. The tier system (light/full) determines whether SYNTHESIS.md is required at completion. Triage friction (`--bypass-triage` flag) intentionally makes manual spawns harder to encourage daemon-driven workflow.

---

## Core Mechanism

### The Spawn Flow

```
orch spawn <skill> "task"
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. SKILL RESOLUTION                                            │
│     Load ~/.claude/skills/{category}/{skill}/SKILL.md           │
│     Extract phases, constraints, requirements                   │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. BEADS ISSUE CREATION (unless --no-track)                    │
│     bd create "{task}" --type {inferred-from-skill}             │
│     Returns beads ID (e.g., orch-go-abc1)                       │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. KB CONTEXT GATHERING                                        │
│     kb context "{task keywords}"                                │
│     Finds relevant constraints, decisions, investigations       │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. BLOAT DETECTION                                             │
│     Extract file paths from task → check line counts           │
│     Warn if any files >800 lines (test files exempt)           │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. WORKSPACE CREATION                                          │
│     .orch/workspace/{name}/                                     │
│     ├── SPAWN_CONTEXT.md   (skill + task + context + bloat)    │
│     ├── .tier              (light/full)                        │
│     ├── .session_id        (OpenCode session ID)               │
│     ├── .beads_id          (beads issue ID)                    │
│     ├── .spawn_time        (timestamp)                         │
│     └── .spawn_mode        (headless/tmux/inline)              │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  6. OPENCODE SESSION                                            │
│     opencode run --model {model} --title "{name} [{beads-id}]"  │
│     Headless by default (HTTP API), --tmux for TUI             │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

**SPAWN_CONTEXT.md structure:**
```markdown
# TASK: {task description}

⚠️ BLOAT DETECTED: {warnings for files >800 lines, if any}

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

**Workspace metadata files:**
- `.tier` - "light" or "full" (determines SYNTHESIS.md requirement)
- `.session_id` - OpenCode session ID for `orch send`
- `.beads_id` - Issue tracking ID for `orch complete`
- `.spawn_time` - ISO timestamp for age calculations
- `.spawn_mode` - Which spawn mode was used

### State Transitions

**Normal spawn lifecycle:**

```
Command invoked (orch spawn)
    ↓
Skill loaded + beads issue created
    ↓
KB context gathered
    ↓
Workspace created (.orch/workspace/{name}/)
    ↓
SPAWN_CONTEXT.md generated
    ↓
OpenCode session created
    ↓
Registry entry added (Status: running)
    ↓
Agent starts working
```

**Cross-project spawn:**

```
cd ~/orchestrator-project
    ↓
orch spawn --workdir ~/target-project investigation "task"
    ↓
Workspace created in: ~/target-project/.orch/workspace/
Beads issue created in: ~/orchestrator-project/.beads/
Session directory: ~/orchestrator-project/ (BUG - should be target)
Agent works in: ~/target-project/
```

### Critical Invariants

1. **Workspace name = kebab-case task description** - Used for tmux window, directory name, session title
2. **Beads ID required for phase reporting** - `--no-track` creates untracked IDs that can't report to beads
3. **KB context uses --global flag** - Cross-repo constraints are essential
4. **Skill content stripped for --no-track** - Beads instructions removed when not tracking
5. **Session scoping is per-project** - `orch send` only works within same directory hash
6. **Token estimation at 4 chars/token** - Warning at 100k, error at 150k

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

**Phase 6: Context Quality & Observability (Jan 22-29, 2026)**
- Bloat detection warns agents about files >800 lines (Jan 24)
- Cross-project beads lookup failures documented as expected behavior (Jan 29)
- OpenCode native integration analysis - confirmed pragmatic bolt-on approach (Jan 28)
- Backend-dependent dedup coverage documented (Jan 22)

---

## References

**Key Investigations:**
- `2025-12-19-inv-spawn-agent-tmux-implementation.md` - Initial tmux implementation
- `2025-12-22-inv-flip-default-spawn-mode-headless.md` - Headless as default
- `2025-12-22-inv-implement-tiered-spawn-protocol.md` - Tier system design
- `2026-01-03-inv-spawn-dependency-gating-design.md` - Dependency checking
- `2026-01-06-inv-add-bypass-triage-friction-manual.md` - Triage friction
- `2026-01-22-inv-analyze-spawn-reliability-pattern-multiple.md` - Backend-dependent dedup coverage
- `2026-01-24-inv-spawn-time-bloat-context-injection.md` - Bloat detection implementation
- `2026-01-28-inv-investigate-opencode-native-agent-spawn.md` - OpenCode integration analysis
- `2026-01-29-inv-orch-spawn-shows-beads-lookup.md` - Cross-project beads behavior
- ...and 35+ others

**Decisions Informed by This Model:**
- `.kb/decisions/2026-01-21-strategic-first-gate-advisory-only.md` - Hotspot guidance remains visible but non-blocking.
- `.kb/decisions/2026-01-28-orchestrator-action-space-architectural-constraint.md` - Keeps orchestrator in meta-action space, pushes implementation to spawned workers.

**Related Models:**
- `.kb/models/context-injection.md` - How SPAWN_CONTEXT.md is assembled and injected
- `.kb/models/opencode-session-lifecycle.md` - How sessions work after spawn creates them
- `.kb/models/dashboard-agent-status.md` - How spawned agents' status is calculated

**Related Guides:**
- `.kb/guides/spawn.md` - How to use spawn command (procedural)
- `.kb/guides/daemon.md` - How daemon auto-spawns (procedural)

**Primary Evidence (Verify These):**
- `cmd/orch/spawn_cmd.go` - Main spawn command implementation (~800 lines)
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md generation (~400 lines)
- `pkg/spawn/config.go` - SpawnConfig struct and validation
- `pkg/skills/loader.go` - Skill discovery and loading
