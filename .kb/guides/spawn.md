# How Spawn Works

**Purpose:** Single authoritative reference for how `orch spawn` creates and configures agents. Read this before debugging spawn issues.

**Last verified:** Jan 7, 2026

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
│  4. WORKSPACE CREATION                                          │
│     .orch/workspace/{name}/                                     │
│     ├── SPAWN_CONTEXT.md   (skill + task + context)            │
│     ├── .tier              (light/full)                        │
│     ├── .session_id        (OpenCode session ID)               │
│     └── .spawn_time        (timestamp)                         │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. OPENCODE SESSION                                            │
│     opencode run --model {model} --title "{name} [{beads-id}]"  │
│     Headless by default (HTTP API), --tmux for TUI             │
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
| `--model <alias>` | Model selection: opus, sonnet, haiku, flash, pro |
| `--mcp <server>` | Add MCP server (e.g., `--mcp playwright`) |
| `--workdir <path>` | Run agent in different directory |

### Mode Flags

| Flag | Purpose |
|------|---------|
| `--tmux` | Use tmux TUI mode instead of headless |
| `--inline` | Run in current terminal, blocking with TUI |
| `--attach` | Spawn in tmux and attach immediately (implies `--tmux`) |
| `--headless` | Run headless via HTTP API (redundant - this is the default) |

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

**Claude backend note:** `runSpawnClaude` never calls `AtomicSpawnPhase2`, so the manifest has no `session_id`. This is by design — Claude CLI is the escape hatch that doesn't depend on OpenCode.

### Non-Default Models in Tmux Mode

`opencode attach` CLI doesn't support a `--model` flag. For tmux spawns with non-default models, pre-create the OpenCode session via HTTP API (which accepts model), then attach by session ID. This avoids modifying the OpenCode fork.

### Worker Detection Chain

Worker sessions are detected through a four-stage chain: HTTP header → session metadata → plugin hook → `detectWorkerSession`. Verified operational Feb 2026 (stress test: 50+ tool calls emitted correct `context_usage` worker metric, zero orchestrator metrics).

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
| `.session_id` | OpenCode session ID for API calls |
| `.spawn_time` | Timestamp for filtering constraint matches |
| `SPAWN_CONTEXT.md` | Agent instructions |
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

- **Headless is default** - `--tmux` is opt-in
- **SPAWN_CONTEXT.md is redundant** - generated from beads + kb + skill + template
- **Tiered spawn** - `.tier` file controls SYNTHESIS.md requirement
- **Fire-and-forget** - tmux spawn doesn't capture session ID, use `orch status` to find it
- **Triage bypass required** - manual spawns need `--bypass-triage` to encourage daemon workflow
- **Proactive rate limits** - warn at 80%, block at 95% with auto-switch attempt
- **Duplicate prevention** - checks for active agents before respawning same issue

---

## Debugging Checklist

Before spawning an investigation about spawn issues:

1. **Check kb:** `kb context "spawn"`
2. **Check this doc:** You're reading it
3. **Check skill exists:** `ls ~/.claude/skills/`
4. **Check beads:** `bd show <id>` if using `--issue`
5. **Check workspace:** `ls .orch/workspace/` for generated files

If those don't answer your question, then investigate. But update this doc with what you learn.
