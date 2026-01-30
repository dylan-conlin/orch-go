# How Spawn Works

**Purpose:** Single authoritative reference for how `orch spawn` creates and configures agents. Read this before debugging spawn issues.

**Last verified:** Jan 29, 2026

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

## Spawn Modes (UI)

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

## Backend Architecture (Triple Spawn)

orch supports three backends for redundancy and different use cases.

| Backend | Flag | CLI Used | Cost Model | Dashboard | Use When |
|---------|------|----------|------------|-----------|----------|
| **Claude** (default) | `--backend claude` | `claude` CLI | $200/mo Max | No | Primary work, Opus quality |
| **OpenCode** | `--backend opencode` | OpenCode API | Pay-per-token | Yes | Cost tracking, headless batch |
| **Docker** | `--backend docker` | Docker + `claude` | $200/mo Max | No | Rate limit escape (fresh fingerprint) |

### Backend Selection Priority

When spawning, backend is determined by (in order):

1. **Explicit `--backend` flag** (claude, opencode, or docker)
2. **`--opus` flag** (implies claude backend)
3. **Project config** (`.orch/config.yaml spawn_mode`)
4. **Global config** (`~/.orch/config.yaml backend`)
5. **Default:** claude

**Example priority cascade:**
```bash
# Explicit flag wins
orch spawn --backend opencode investigation "task"   # → OpenCode

# --opus implies claude
orch spawn --opus investigation "task"               # → Claude

# Project config (if .orch/config.yaml has spawn_mode: opencode)
orch spawn investigation "task"                      # → OpenCode

# Default (no config)
orch spawn investigation "task"                      # → Claude
```

### Primary Path: Claude CLI

```bash
orch spawn investigation "analyze X"
```

**Characteristics:**
- Uses `claude` CLI with Max subscription
- Opus access (highest quality model)
- Flat $200/mo cost
- Tmux window for visual monitoring

**Use for:** Most work, orchestration, architecture, complex reasoning

### OpenCode API Path

```bash
orch spawn --backend opencode --model sonnet feature-impl "task"
```

**Characteristics:**
- HTTP API to OpenCode server (localhost:4096)
- Dashboard visibility
- Pay-per-token pricing
- Model choice: Sonnet, DeepSeek, Gemini

**Use for:** Cost tracking, batch automation, when dashboard visibility needed

**Constraint:** Cannot use Opus via OpenCode (Anthropic fingerprinting blocks it)

**Integration depth:** orch-go uses pragmatic bolt-on approach with `ORCH_WORKER=1` environment variable (converted to `x-opencode-env-ORCH_WORKER` header). OpenCode has comprehensive native agent support (Agent.Info with mode, Session.Info with parentID, task tool for spawning), but orch-go intentionally uses external orchestration model for:
- Decoupling from OpenCode's agent model (runtime portability)
- Cross-project orchestration (OpenCode sessions are project-scoped)
- Full control over worker lifecycle

**Alternative:** Could use OpenCode's native session hierarchy (parentID) for better UI visibility. See `.kb/investigations/2026-01-28-inv-investigate-opencode-native-agent-spawn.md` for integration options analysis.

### Docker Escape Hatch

```bash
orch spawn --backend docker investigation "task"
```

**Characteristics:**
- Host tmux window runs Docker container
- Fresh Statsig fingerprint per spawn
- Uses `~/.claude-docker/` (separate from host `~/.claude/`)
- Same lifecycle commands as claude mode (status, complete, abandon)

**Use for:** Rate limit bypass when host fingerprint is throttled

**Critical constraints:**
- Docker image `claude-code-mcp` must be pre-built
- `BEADS_NO_DAEMON=1` set automatically (Unix sockets don't work over Docker mounts)
- Container PATH must include `/usr/local/go/bin` for auto-rebuild
- ~2-5s startup overhead per spawn

**Important:** Docker bypasses **device-level rate throttling** only. The weekly usage quota is **account-level** and cannot be bypassed with fingerprint isolation.

### Infrastructure Work Detection

When spawning work that mentions "opencode", "spawn", "daemon", "registry", "orch serve", "overmind", or "dashboard", orch **warns** that claude+tmux is recommended but does NOT auto-override.

**Why:** Infrastructure work can kill its own execution path (e.g., restarting OpenCode server kills OpenCode-spawned agents).

```bash
# System warns about infrastructure keywords
orch spawn --bypass-triage investigation "fix opencode server crash"
# Warning: Infrastructure keywords detected. Consider --backend claude --tmux

# Explicitly use escape hatch for critical work
orch spawn --bypass-triage --backend claude --tmux investigation "fix opencode"
```

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

## Bloat Detection

Spawn performs automatic bloat detection when generating SPAWN_CONTEXT.md.

### How It Works

1. **File path extraction** - Parses task description for file references (e.g., `pkg/spawn/context.go`)
2. **Line counting** - Checks each mentioned file against 800-line threshold
3. **Warning injection** - Adds warnings to SPAWN_CONTEXT.md for bloated files
4. **Test file exemption** - Skips files matching `*_test.go`, `*.test.ts`, etc.

### Example Warning

```markdown
⚠️ BLOAT DETECTED:

The following files mentioned in your task exceed the 800-line coherence threshold:

- pkg/spawn/context.go (1,247 lines)

Before modifying these files, consider extraction. See .kb/guides/code-extraction-patterns.md.
```

### Why This Matters

- **Coherence** - Files over 800 lines are hard to understand and modify
- **Agent success** - Agents struggle with large files (multiple concerns, deep nesting)
- **Technical debt** - Bloat indicates missing abstractions

**Related:** `orch hotspot` shows all files over threshold across project.

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

### Cross-Project Beads Lookup Behavior

When spawning, orch checks ALL active sessions for concurrency limiting, including sessions from other projects:

```bash
# Example: Sessions from multiple projects
og-feat-something [orch-go-21012]      # orch-go project
sp-feat-something [specs-platform-36]  # specs-platform project
```

**Expected behavior:** When spawning from orch-go, beads lookups for `specs-platform-36` will fail with "issue not found". This is **normal and expected** - the issue exists in a different project's beads database.

**Why this happens:**
- OpenCode sessions persist across projects
- Concurrency limiting checks all sessions
- Cross-project beads IDs won't exist in current project's database

**Impact:** You may see warnings like "beads lookup failed for specs-platform-36" - these are informational, not errors. The spawn continues normally.

---

## Key Decisions (from kb quick)

### Spawn Mechanics
- **Headless is default** - `--tmux` is opt-in for visual monitoring
- **SPAWN_CONTEXT.md is redundant** - generated from beads + kb + skill + template
- **Tiered spawn** - `.tier` file controls SYNTHESIS.md requirement
- **Fire-and-forget** - tmux spawn doesn't capture session ID, use `orch status` to find it
- **Triage bypass required** - manual spawns need `--bypass-triage` to encourage daemon workflow

### Backend Architecture
- **Claude CLI is default backend** - Opus access, $200/mo flat (Jan 18, 2026 decision)
- **Only two viable API paths** - claude+opus or opencode+sonnet (Opus blocked via API)
- **Docker provides fingerprint isolation** - for device-level rate limit bypass only
- **Weekly quota is account-level** - Docker cannot bypass weekly usage limits
- **Infrastructure detection is advisory** - warns but doesn't auto-override backend

### Safety & Limits
- **Proactive rate limits** - warn at 80%, block at 95% with auto-switch attempt
- **Duplicate prevention** - checks for active agents before respawning same issue
- **Dedup coverage is backend-dependent** - OpenCode has session dedup; Claude CLI and Docker rely on status update + Phase: Complete check only (lighter protection)
- **Agents limited to 3 iterations** - without human review to prevent runaway loops
- **Abandon after service crashes** - stale sessions don't reconnect, need re-triage
- **Don't spawn multiple agents for same file** - causes merge conflicts

---

## Debugging Checklist

Before spawning an investigation about spawn issues:

1. **Check kb:** `kb context "spawn"`
2. **Check this doc:** You're reading it
3. **Check skill exists:** `ls ~/.claude/skills/`
4. **Check beads:** `bd show <id>` if using `--issue`
5. **Check workspace:** `ls .orch/workspace/` for generated files
6. **Check backend:** Is the right backend being used? (`orch spawn --help`)

### Backend-Specific Debugging

**"Opus auth rejected"**
- Opus requires Claude CLI backend, not OpenCode API
- Fix: Use `--backend claude` (or remove `--backend opencode`)

**"Docker spawn failing"**
- Check image exists: `docker images | grep claude-code-mcp`
- Check PATH in container includes `/usr/local/go/bin`
- Verify BEADS_NO_DAEMON=1 is being set

**"Rate limited but Docker shows 0%"**
- Weekly quota is account-level, not device-level
- Docker only bypasses request-rate throttling, not weekly limits
- Solution: Wait for reset or switch accounts

**"Agent killed mid-work"**
- If working on infrastructure (opencode, spawn, daemon), agent may have killed its own execution path
- Solution: Use `--backend claude --tmux` for infrastructure work

If those don't answer your question, then investigate. But update this doc with what you learn.

---

## Related Documentation

### Guides
- **Model Selection:** `.kb/guides/model-selection.md` - Which model for which task
- **Triple Spawn Implementation:** `.kb/guides/dual-spawn-mode-implementation.md` - Implementation details
- **Daemon Guide:** `.kb/guides/daemon.md` - Autonomous spawning via daemon

### Models
- **Spawn Architecture:** `.kb/models/spawn-architecture.md` - Architectural overview and evolution
- **Context Injection:** `.kb/models/context-injection.md` - How SPAWN_CONTEXT.md is assembled
- **Model Access & Spawn Paths:** `.kb/models/model-access-spawn-paths.md` - Detailed mechanics

### Recent Investigations (Jan 2026)
- **Bloat Detection:** `.kb/investigations/2026-01-24-inv-spawn-time-bloat-context-injection.md` - Implementation of spawn-time bloat warnings
- **Cross-Project Beads:** `.kb/investigations/2026-01-29-inv-orch-spawn-shows-beads-lookup.md` - Expected cross-project lookup failures
- **OpenCode Integration:** `.kb/investigations/2026-01-28-inv-investigate-opencode-native-agent-spawn.md` - Analysis of native vs bolt-on integration
- **Reliability Patterns:** `.kb/investigations/2026-01-22-inv-analyze-spawn-reliability-pattern-multiple.md` - Backend-dependent dedup coverage
