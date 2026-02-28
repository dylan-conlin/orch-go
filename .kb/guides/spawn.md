# How Spawn Works

**Purpose:** Single authoritative reference for how `orch spawn` creates and configures agents. Read this before debugging spawn issues.

**Last verified:** Feb 26, 2026

---

## The Flow

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
│  2. SETTINGS RESOLUTION (pkg/spawn/resolve.go)                  │
│     Resolve backend, model, spawn mode, tier from:              │
│     CLI flags → project config → user config → heuristics       │
│     Model-aware routing: Anthropic → Claude CLI, others → OC    │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. PREFLIGHT GATES (all hard gates — fail-fast)                │
│     Triage bypass → Concurrency → Rate limit →                  │
│     Verification → Hotspot                                      │
│     Any gate failure aborts spawn before side effects            │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. BEADS ISSUE CREATION (unless --no-track)                    │
│     bd create "{task}" --type {inferred-from-skill}             │
│     Returns beads ID → transitions to in_progress               │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. KB CONTEXT GATHERING                                        │
│     kb context "{task keywords}"                                │
│     Finds relevant constraints, decisions, investigations       │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  6. ATOMIC SPAWN PHASE 1 (with rollback)                        │
│     .orch/workspace/{name}/                                     │
│     ├── SPAWN_CONTEXT.md, AGENT_MANIFEST.json                  │
│     ├── .tier, .spawn_time, .spawn_mode, .beads_id             │
│     Failure → rollback (delete workspace + beads label)         │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  7. ATOMIC SPAWN PHASE 2 (per-backend session creation)         │
│     Claude backend: tmux window + claude CLI (no OC session)    │
│     OpenCode headless: HTTP API session + send prompt           │
│     OpenCode tmux: tmux window + opencode attach                │
│     Inline: opencode run in current terminal                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## Spawn Modes

| Mode | Flag | Behavior | Use When |
|------|------|----------|----------|
| **Headless** (default) | none | HTTP API, returns immediately | Automation, batch work, daemon |
| **Tmux** | `--tmux` | Creates tmux window with TUI | Visual monitoring, debugging |
| **Inline** | `--inline` | Runs in current terminal, blocking | Quick tests, debugging |

**Headless is the default** because:
- No TUI overhead
- Returns immediately (non-blocking)
- Works with daemon automation
- Session still accessible via `orch status`, `orch send`

---

## Backend Routing (Feb 2026)

Model-aware backend routing automatically selects the correct backend based on the model's provider. This became mandatory when Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026).

### Routing Rules

| Model Provider | Backend | Mechanism |
|----------------|---------|-----------|
| **Anthropic** (opus, sonnet, haiku) | Claude CLI | Tmux window + `claude` CLI. Cannot run headless. |
| **Google** (flash, pro) | OpenCode | HTTP API (headless) or tmux TUI |
| **OpenAI** (codex, gpt-5) | OpenCode | HTTP API (headless) or tmux TUI |

**Key constraint:** Anthropic models must never be spawned via OpenCode backend. The Claude CLI backend always forces tmux mode (headless is impossible for Claude CLI).

### Resolution Priority

Settings are resolved with a layered priority system (`pkg/spawn/resolve.go`):

1. **CLI flags** (highest) — `--backend claude` is a hard override, skips model-aware routing
2. **Project config** (`.orch/config.yaml`)
3. **User config** (`~/.orch/config.yaml`)
4. **Heuristics** — model-provider routing lives here
5. **Defaults** (lowest) — backend defaults to `claude`, model defaults to `opus`

**Override:** `allow_anthropic_opencode: true` in user config disables the Anthropic → Claude enforcement (for testing only).

### Daemon Model Inference

The daemon infers model from skill type before spawning:

| Skill Category | Model | Reasoning |
|----------------|-------|-----------|
| investigation, architect, systematic-debugging, codebase-audit, research | opus | Deep reasoning |
| feature-impl, issue-creation | sonnet | Implementation |
| All others | sonnet (default) | Safe default |

### Gotchas

- **`--mode` is NOT `--backend`:** `--mode` controls implementation approach (tdd/direct/verification-first). Using `--mode claude` or `--mode opencode` is rejected with a helpful error pointing to `--backend`.
- **`opencode attach` has no `--model` flag:** For tmux spawns with non-default models, the session is pre-created via HTTP API (which accepts model), then attached by session ID.
- **Claude backend + headless = impossible:** Claude CLI requires a terminal. If backend resolves to `claude` and mode resolves to `headless`, mode is auto-overridden to `tmux` (source: `claude-backend-requires-tmux`).

### Failed Approaches

- **gpt-5 alias for headless OpenCode spawn** — attempted, ran into issues with OpenCode's model routing.
- **tmux spawn with OpenCode backend + --model flag** — `opencode attach` doesn't support `--model`; pre-create session via HTTP API instead.

---

## Spawn Gates (Preflight Checks)

**All spawn gates are hard gates (fail-fast).** They abort spawn before any side effects (no beads issue created, no workspace, no session). This applies to both manual spawns and daemon-driven spawns.

### Gate Ordering

Gates run in this order (all must pass):

| Gate | What it checks | Override |
|------|---------------|----------|
| **Triage bypass** | Manual spawns have `--bypass-triage` | `--bypass-triage` flag |
| **Concurrency** | Active agent count < max (default 5) | `--max-agents 0` |
| **Rate limit** | Account usage < 95% | `ORCH_USAGE_BLOCK_THRESHOLD=100` |
| **Verification** | No unverified Tier 1 work exists | `--bypass-verification` + `--bypass-reason` |
| **Hotspot** | Target files not CRITICAL (>1500 lines) for blocking skills | `--force-hotspot` + `--architect-ref` |

### Hotspot Gate Details

The hotspot gate only blocks specific skills that modify code:

**Blocked skills:** `feature-impl`, `systematic-debugging`

**Exempt skills:** investigation, architect, codebase-audit, capture-knowledge, and all others (strategic/read-only skills).

**Auto-bypass:** Before hard blocking, the gate checks for a prior closed architect review of the critical files. If a verified architect review exists, the gate auto-bypasses.

**Context injection:** When hotspot files are detected (even if not blocking), the SPAWN_CONTEXT.md includes a hotspot warning with **all matched hotspot files** (`MatchedFiles`), not just the critical ones (`CriticalFiles` >1500 lines). This gives agents full awareness of the hotspot area.

### Why Hard Gates Matter

The Feb 14, 2026 duplicate spawn incident caused 10 agents to spawn because an `UpdateBeadsStatus` failure was logged as a warning and spawn continued. **Rule:** If a spawn prerequisite fails, return an error or skip the issue. Never log a warning and spawn anyway.

**Pre-spawn auditing:** Before spawning agents to fix reported problems, verify the premise against actual code first. Four issues flagged as "broken" all had functional code — a quick grep/build check would have prevented wasted audit capacity.

---

## Triage Bypass (Manual Spawn Friction)

**Manual spawns require `--bypass-triage` flag.**

The preferred workflow is daemon-driven triage:
1. Create issue: `bd create "task" --type task -l triage:ready`
2. Daemon auto-spawns: `orch daemon run`

Manual spawn is for exceptions only:
- Single urgent item requiring immediate attention
- Complex/ambiguous task needing custom context
- Skill selection requires orchestrator judgment

This friction is intentional - it encourages the scalable daemon-driven workflow over ad-hoc spawning.

**Example:**
```bash
# Preferred: daemon-driven
bd create "investigate auth" --type investigation -l triage:ready
orch daemon run

# Exception: manual spawn (requires bypass)
orch spawn --bypass-triage investigation "urgent exploration"
```

---

## Rate Limit Monitoring

Spawn performs proactive rate limit monitoring before creating agents.

### Thresholds

| Level | Usage % | Behavior |
|-------|---------|----------|
| **Normal** | < 80% | Spawn proceeds normally |
| **Warning** | 80-95% | Warning displayed, spawn proceeds |
| **Critical** | ≥ 95% | Attempts auto-switch, blocks if no alternate account |

### Auto-Switch Behavior

At critical threshold (95%):
1. Checks for alternate accounts with more headroom
2. If found, automatically switches and continues
3. If not found, **blocks spawn** with guidance

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `ORCH_USAGE_WARN_THRESHOLD` | Warning threshold % | 80 |
| `ORCH_USAGE_BLOCK_THRESHOLD` | Blocking threshold % | 95 |
| `ORCH_AUTO_SWITCH_DISABLED` | Disable auto-switch (`1` or `true`) | false |

### Override

To bypass rate limit blocking (not recommended):
```bash
ORCH_USAGE_BLOCK_THRESHOLD=100 orch spawn --bypass-triage ...
```

---

## Duplicate Prevention

Spawn checks for existing work on the same issue before creating new agents.

### What Gets Checked

When spawning with `--issue`:

1. **Closed issue:** Spawn blocked - "issue is already closed"
2. **In-progress with active agent:** Spawn blocked - use `orch send` or `orch abandon`
3. **In-progress with stale session:** Warning, spawn proceeds (session >30m idle)
4. **Phase: Complete reported:** Spawn blocked - run `orch complete` first

### Phase: Complete Detection

When respawning an `in_progress` issue without an active session, spawn checks `verify.IsPhaseComplete(beadsID)`. If a recent "Phase: Complete" comment exists, spawn is **hard blocked** — you must run `orch complete` first. This prevents duplicate spawn loops where an agent has finished but hasn't been formally completed.

### Beads ID Consistency

`spawn.ValidateBeadsIDConsistency()` checks if the task text mentions a same-project beads ID that differs from the `--issue` flag. This is a **soft warning only** (prints to stderr, does not block). It catches copy-paste errors where the task description references one issue but `--issue` tracks a different one.

### Active Agent Detection

An agent is considered "active" if:
- Session exists in OpenCode
- Last activity < 30 minutes ago
- Has parseable beads ID in title (orch-spawned, not manual)

**Stale sessions** (>30m inactive) are logged but don't block respawning.

### Concurrency Limit

By default, limits to 5 concurrent active agents. Configure via:
- `--max-agents <n>` flag
- `ORCH_MAX_AGENTS` environment variable
- Set to 0 to disable (not recommended)

Active count excludes:
- Stale sessions (>30m inactive)
- Non-orch sessions (manual OpenCode sessions)
- Completed agents (Phase: Complete reported)

---

## Key Flags

### Required Flags

| Flag | Purpose |
|------|---------|
| `--bypass-triage` | **Required for manual spawns.** Acknowledges bypassing daemon-driven triage workflow. |

### Core Flags

| Flag | Purpose |
|------|---------|
| `--issue <id>` | Spawn for existing beads issue (don't create new) |
| `--no-track` | Skip beads issue creation (ad-hoc work) |
| `--model <alias>` | Model selection: opus, sonnet, haiku, flash, pro. Triggers model-aware backend routing. |
| `--backend <name>` | Force backend: `claude` or `opencode`. Overrides model-aware routing. |
| `--mcp <server>` | Add MCP server (e.g., `--mcp playwright`) |
| `--workdir <path>` | Run agent in different directory |

### Mode Flags

| Flag | Purpose |
|------|---------|
| `--tmux` | Use tmux TUI mode instead of headless |
| `--inline` | Run in current terminal, blocking with TUI |
| `--attach` | Spawn in tmux and attach immediately (implies `--tmux`) |
| `--headless` | Run headless via HTTP API (redundant - this is the default for OpenCode backend) |

**Note:** `--mode` controls implementation approach (tdd/direct), NOT backend. See [Backend Routing](#backend-routing-feb-2026).

### Tier Flags

| Flag | Purpose |
|------|---------|
| `--light` | Light tier: skips SYNTHESIS.md requirement on completion |
| `--full` | Full tier: requires SYNTHESIS.md for knowledge externalization |

Default tier is determined by skill:
- **Full tier:** investigation, architect, research, codebase-audit, design-session, systematic-debugging
- **Light tier:** feature-impl, reliability-testing, issue-creation

### Feature-impl Configuration Flags

| Flag | Purpose |
|------|---------|
| `--phases <list>` | Comma-separated phases (e.g., `implementation,validation`) |
| `--mode <mode>` | Implementation mode: `tdd` (default) or `direct` |
| `--validation <level>` | Validation level: `none`, `tests` (default), `smoke-test` |

### Safety Flags

| Flag | Purpose |
|------|---------|
| `--max-agents <n>` | Maximum concurrent agents (default 5, 0 to disable) |
| `--auto-init` | Auto-initialize .orch and .beads if missing |
| `--force` | Override safety checks (blocking dependencies, existing workspace) |
| `--force-hotspot` | Override hotspot gate (requires `--architect-ref`) |
| `--architect-ref <id>` | Beads ID of prior architect review (required with `--force-hotspot`) |
| `--bypass-verification` | Override verification gate (requires `--bypass-reason`) |
| `--bypass-reason <text>` | Reason for bypassing verification gate |

### Context Quality Flags

| Flag | Purpose |
|------|---------|
| `--skip-artifact-check` | Bypass pre-spawn kb context check |
| `--gate-on-gap` | Block spawn if context quality is too low (score < threshold) |
| `--skip-gap-gate` | Explicitly bypass gap gating (documents conscious decision) |
| `--gap-threshold <n>` | Custom gap quality threshold (default 20) |

---

## Two-Phase Atomic Spawn (Feb 2026)

Spawn executes in two phases with rollback on failure:

**Phase 1 (common):** Beads issue creation + workspace setup. Rolls back (deletes issue + workspace) on failure. All four backends share this phase.

**Phase 2 (per-backend):** Session creation varies by backend:

| Backend | Phase 2 behavior |
|---------|-----------------|
| **opencode** (headless) | Creates OpenCode session via HTTP API, sends prompt |
| **opencode** (tmux) | Creates tmux window, runs `opencode attach` |
| **claude** | Creates tmux window, runs `claude` CLI — **no OpenCode session** |
| **inline** | Runs `opencode run` in current terminal |

**Claude backend note:** `runSpawnClaude` never calls `AtomicSpawnPhase2`, so the manifest has no `session_id`. This is by design — Claude CLI is the primary backend and doesn't depend on OpenCode.

### Non-Default Models in Tmux Mode

`opencode attach` CLI doesn't support a `--model` flag. For tmux spawns with non-default models, pre-create the OpenCode session via HTTP API (which accepts model), then attach by session ID. This avoids modifying the OpenCode fork.

### Worker Detection Chain

Worker sessions are detected through a four-stage chain: HTTP header → session metadata → plugin hook → `detectWorkerSession`. Verified operational Feb 2026 (stress test: 50+ tool calls emitted correct `context_usage` worker metric, zero orchestrator metrics).

---

## Beads Integration at Spawn

### Issue Creation & Status Transition

When spawn creates a beads issue (or uses `--issue`):
1. Issue is created (or existing issue is validated)
2. Issue is transitioned to `in_progress` status immediately
3. Issue is assigned to the workspace name

**Why transition at spawn time:** Dashboard discovery depends on status being `open` or `in_progress`. Without the transition, auto-created issues would be invisible to the dashboard until the agent's first phase report.

### ORIENTATION_FRAME

`ORIENTATION_FRAME` content belongs in beads comments only, not in SPAWN_CONTEXT.md. The SPAWN_CONTEXT template should not include orientation frame content — it's ephemeral context that's captured in the beads comment history instead.

### kb context is Automatic

`orch spawn` runs `kb context` automatically via `pkg/spawn/kbcontext.go`. The `spawn_without_context` metric was killed because spawning without context is impossible through the normal path.

---

## What Gets Generated

### SPAWN_CONTEXT.md

The agent's instruction file. Contains:
- Task description
- Skill content (full SKILL.md embedded)
- KB context (relevant constraints, decisions)
- Beads ID for phase reporting
- Workspace path for artifacts
- Authority levels (what agent can decide vs escalate)

**Authority levels:** See `.kb/guides/decision-authority.md` for detailed criteria on when agents should decide vs escalate. The SPAWN_CONTEXT.md includes a summary, but the full guide provides the decision tree and examples.

**Key insight:** SPAWN_CONTEXT.md is 100% generated from beads + kb + skill + template. If you need to change what agents receive, change the sources, not the output.

### Workspace Files

| File | Purpose |
|------|---------|
| `.tier` | "light" or "full" - controls SYNTHESIS.md requirement |
| `.session_id` | OpenCode session ID (absent for Claude backend) |
| `.spawn_time` | Timestamp for filtering constraint matches |
| `.spawn_mode` | Backend identifier for send/monitor dispatch routing |
| `.beads_id` | Beads issue ID for this agent |
| `SPAWN_CONTEXT.md` | Agent instructions |
| `AGENT_MANIFEST.json` | Machine-readable agent metadata |
| `SYNTHESIS.md` | Agent creates this before completion (full tier only) |

---

## Skill → Issue Type Mapping

Spawn infers beads issue type from skill:

| Skill | Issue Type |
|-------|------------|
| `investigation` | investigation |
| `systematic-debugging` | bug |
| `feature-impl` | feature |
| `codebase-audit` | task |
| `architect` | task |
| others | task |

This matters because daemon uses issue type to infer skill when auto-spawning.

---

## Common Problems

### "Spawn hangs on kb context"

**Cause:** Some kb queries are slow or hang.

**Fix:** Use `--skip-artifact-check` to bypass kb context gathering.

### "bd comment fails with 'issue not found'"

**Cause:** Using `--no-track` creates placeholder beads IDs (e.g., `orch-go-untracked-*`) that don't exist in the database.

**This is expected.** Untracked spawns can't report phases via beads. The agent should still create artifacts in the workspace.

### "Agent doesn't have the context it needs"

**Cause:** KB context didn't find relevant entries, or skill doesn't include needed guidance.

**Fix:** 
1. Check what `kb context "{keywords}"` returns
2. Add missing knowledge via `kn decide/constrain/tried`
3. Update skill if guidance is missing

### "Wrong skill loaded"

**Cause:** Skill name typo or skill doesn't exist.

**Check:** `ls ~/.claude/skills/` to see available skills.

### "--mode claude" rejected

**Cause:** `--mode` controls implementation approach (tdd/direct/verification-first), not backend selection. Users confuse it with `--backend`.

**Fix:** Use `--backend claude` or `--backend opencode` for backend selection. The error message explains this.

### "Hotspot gate blocked my spawn"

**Cause:** Target files exceed 1500 lines (CRITICAL hotspot) and skill is `feature-impl` or `systematic-debugging`.

**Fix options:**
1. Use `--force-hotspot` + `--architect-ref <issue-id>` (requires prior closed architect review)
2. Run an `architect` skill first to review the hotspot area (architect is exempt)
3. Extract the large file first to bring it under threshold

### "Spawn re-spawned an issue that was already done"

**Cause:** Agent reported Phase: Complete but `orch complete` wasn't run. Without formal completion, the issue stays `in_progress` and can be re-spawned.

**Fix:** Run `orch complete <agent-id>` to verify and close the issue. The spawn system checks for Phase: Complete comments and blocks re-spawning when found.

---

## Cross-Project Spawns

Use `--workdir` to spawn agents in different repos:

```bash
# Spawn in kb-cli repo from orch-go
orch spawn feature-impl "add feature" --workdir ~/Documents/personal/kb-cli
```

**What happens:**
- Agent runs in target directory
- Beads issue created in CURRENT directory (orchestrator's repo)
- Workspace created in target's `.orch/workspace/`

**Gotcha:** `bd comment` from the agent uses target directory, but issue is in orchestrator's repo. This can cause "issue not found" errors. Use `--no-track` for cross-repo work, or manually track.

---

## Key Decisions (from kn)

- **Model-aware backend routing** - Anthropic models → Claude CLI, others → OpenCode. Default model is opus, default backend is claude. (Feb 2026)
- **All spawn gates are hard (fail-fast)** - No soft warnings. Gate failure aborts before side effects.
- **Hotspot gate uses skill allowlist** - Only `feature-impl` and `systematic-debugging` are blocked; exempt skills aren't enumerated.
- **Two-phase atomic spawn** - Phase 1 (beads + workspace) with rollback, Phase 2 (per-backend session) best-effort.
- **Headless is default** - `--tmux` is opt-in (but Claude backend always forces tmux)
- **SPAWN_CONTEXT.md is redundant** - generated from beads + kb + skill + template
- **ORIENTATION_FRAME in beads only** - not embedded in SPAWN_CONTEXT.md
- **Tiered spawn** - `.tier` file controls SYNTHESIS.md requirement
- **Fire-and-forget** - tmux spawn doesn't capture session ID, use `orch status` to find it
- **Triage bypass required** - manual spawns need `--bypass-triage` to encourage daemon workflow
- **Proactive rate limits** - warn at 80%, block at 95% with auto-switch attempt
- **Duplicate prevention** - checks for active agents, Phase: Complete, and beads ID consistency before respawning
- **Issues transition to in_progress at spawn** - dashboard discovery depends on this status
- **Accretion gravity via structural extraction** - create attractor packages, agents naturally route there (proven by probe: agent found `pkg/spawn/gates/` without being told)

---

## Debugging Checklist

Before spawning an investigation about spawn issues:

1. **Check kb:** `kb context "spawn"`
2. **Check this doc:** You're reading it
3. **Check skill exists:** `ls ~/.claude/skills/`
4. **Check beads:** `bd show <id>` if using `--issue`
5. **Check workspace:** `ls .orch/workspace/` for generated files

If those don't answer your question, then investigate. But update this doc with what you learn.

---

## Code Organization

Key files for spawn system developers:

| File | Lines | Purpose |
|------|-------|---------|
| `cmd/orch/spawn_cmd.go` | ~882 | Main spawn command, all flags, `isInfrastructureWork` |
| `pkg/spawn/resolve.go` | ~661 | Centralized settings resolution (backend, model, mode, tier) |
| `pkg/orch/extraction.go` | ~1551 | Preflight checks, beads setup, context writing (**CRITICAL hotspot**) |
| `pkg/orch/spawn_modes.go` | ~529 | Dispatch + per-backend runners (headless, tmux, claude, inline) |
| `pkg/orch/spawn_helpers.go` | ~149 | Formatting and utility helpers |
| `pkg/spawn/atomic.go` | ~113 | Two-phase atomic spawn with rollback |
| `pkg/spawn/config.go` | ~485 | SpawnConfig struct, workspace naming |
| `pkg/spawn/claude.go` | ~172 | Claude CLI tmux launch |
| `pkg/spawn/gates/` | | Gate implementations (hotspot, verification, triage, concurrency, rate limit) |
| `pkg/spawn/backends/` | | Extracted backend implementations (in-progress, not yet wired to main dispatch) |
| `pkg/orch/flags.go` | ~46 | `ValidateMode` — rejects `claude`/`opencode` as `--mode` values |

**Note:** `extraction.go` exceeds the 1500-line CRITICAL threshold. The P0 extraction (Feb 2026) already split spawn modes and helpers out, reducing it from 2077 to ~1551 lines. Further extraction needed.
