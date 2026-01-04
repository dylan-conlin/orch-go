# How Spawn Works

**Purpose:** Single authoritative reference for how `orch spawn` creates and configures agents. Read this before debugging spawn issues.

**Last verified:** Jan 4, 2026

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

## Key Flags

| Flag | Purpose |
|------|---------|
| `--issue <id>` | Spawn for existing beads issue (don't create new) |
| `--no-track` | Skip beads issue creation (ad-hoc work) |
| `--model <alias>` | Model selection: opus, sonnet, flash, pro |
| `--mcp <server>` | Add MCP server (e.g., `--mcp playwright`) |
| `--workdir <path>` | Run agent in different directory |
| `--tmux` | Use tmux TUI mode instead of headless |

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

---

## Debugging Checklist

Before spawning an investigation about spawn issues:

1. **Check kb:** `kb context "spawn"`
2. **Check this doc:** You're reading it
3. **Check skill exists:** `ls ~/.claude/skills/`
4. **Check beads:** `bd show <id>` if using `--issue`
5. **Check workspace:** `ls .orch/workspace/` for generated files

If those don't answer your question, then investigate. But update this doc with what you learn.
