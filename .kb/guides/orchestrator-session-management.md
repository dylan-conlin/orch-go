# Orchestrator Session Management

**Purpose:** Single authoritative reference for orchestrator sessions, spawnable orchestrators, and the meta-orchestrator architecture. Read this before debugging orchestrator lifecycle issues.

**Last verified:** 2026-01-07

**Synthesized from:** 40 investigations on orchestrator topics (Dec 21, 2025 - Jan 7, 2026)

---

## Overview

This guide covers the orchestrator session lifecycle - from spawning orchestrator sessions to completion, including the three-tier hierarchy (meta-orchestrator -> orchestrator -> worker), session boundaries, completion verification, and the SESSION_HANDOFF.md artifact pattern.

**When to consult this guide:**
- Understanding when orchestrators should be spawned vs interactive
- Debugging completion failures for orchestrator sessions
- Understanding the meta-orchestrator role and frame shift
- Fixing frame collapse or level confusion issues

---

## Architecture

```
                    ┌─────────────────────────┐
                    │   Meta-Orchestrator     │
                    │   (Dylan - human)       │
                    │   Strategic decisions   │
                    └────────────┬────────────┘
                                 │ spawns/completes
                                 ▼
                    ┌─────────────────────────┐
                    │    Orchestrator         │
                    │    (Claude agent)       │
                    │    Strategic comprehension │
                    │    SESSION_HANDOFF.md   │
                    └────────────┬────────────┘
                                 │ spawns/completes
                                 ▼
                    ┌─────────────────────────┐
                    │    Worker               │
                    │    (Claude agent)       │
                    │    Implementation       │
                    │    SYNTHESIS.md         │
                    └─────────────────────────┘
```

**Key insight:** Each level is completed by the level above. Workers don't self-terminate; orchestrators complete them. Orchestrators don't self-terminate; the meta-orchestrator (Dylan or spawned meta-orchestrator) completes them.

---

## How It Works

### Session Types and Boundaries

**What:** Three distinct session types exist with different boundary patterns.

**Key insight:** Worker boundaries are protocol-driven (Phase: Complete + /exit). Orchestrator boundaries are state-driven (SESSION_HANDOFF.md + wait). Cross-session boundaries are manual (SESSION_HANDOFF.md for continuity).

| Session Type | Boundary Trigger | Handoff Mechanism | Artifact |
|--------------|------------------|-------------------|----------|
| Worker | `bd comment "Phase: Complete"` + `/exit` | SPAWN_CONTEXT.md → SYNTHESIS.md | SYNTHESIS.md |
| Orchestrator | SESSION_HANDOFF.md created + wait | ORCHESTRATOR_CONTEXT.md → SESSION_HANDOFF.md | SESSION_HANDOFF.md |
| Cross-session | End of working day/session | Manual reflection | SESSION_HANDOFF.md |

### Orchestrator Spawn Infrastructure

**What:** Orchestrator-type skills (skill-type: policy) receive ORCHESTRATOR_CONTEXT.md instead of SPAWN_CONTEXT.md.

**Key insight:** Orchestrators are detected via skill metadata, not explicit flags. Skills with `skill-type: policy` or `skill-type: orchestrator` trigger orchestrator context generation.

| Aspect | Worker | Orchestrator |
|--------|--------|--------------|
| Context File | SPAWN_CONTEXT.md | ORCHESTRATOR_CONTEXT.md |
| Completion Artifact | SYNTHESIS.md | SESSION_HANDOFF.md |
| Beads Tracking | Required | Skipped |
| Default Spawn Mode | Headless | Tmux (visible) |
| Completion Signal | `Phase: Complete` + `/exit` | SESSION_HANDOFF.md + wait |
| Workspace Prefix | og-work-*, og-feat-*, etc. | og-orch-* |

### Session Registry

**What:** `~/.orch/sessions.json` tracks active orchestrator sessions without beads.

**Key insight:** Orchestrators aren't issues being worked on - they're interactive sessions. Beads tracks work items; the session registry tracks conversations.

| Field | Purpose |
|-------|---------|
| workspace_name | Unique identifier |
| session_id | OpenCode session ID |
| spawn_time | When session started |
| project_dir | Project the session is managing |
| status | active/completed/abandoned |

### Session Registry Status Updates (Jan 2026 fix)

**What:** Registry status now updates on completion and abandonment.

**Key insight:** `orch complete` updates status to "completed", `orch abandon` updates to "abandoned" - sessions are preserved for history, not removed.

### Interactive vs Spawned Workspaces

**What:** Two workspace models exist for orchestrators.

| Type | Workspace Location | Artifact | Created By |
|------|-------------------|----------|------------|
| Spawned | `.orch/workspace/{name}/` | SESSION_HANDOFF.md | `orch spawn orchestrator` |
| Interactive | `~/.orch/session/{date}/` | SESSION_CONTEXT.md | `orch session start` |

**Key insight:** Interactive sessions have lighter workspace structure. Spawned orchestrators get full workspaces with pre-filled templates.

---

## Checkpoint Discipline (Jan 2026)

**Purpose:** Prevent quality degradation from context exhaustion via session duration warnings.

**Thresholds:**
| Duration | Level | Action |
|----------|-------|--------|
| < 2h | ok | Continue |
| 2-3h | warning | Consider checkpoint |
| 3-4h | strong | Strongly recommend checkpoint |
| > 4h | exceeded | Session should have checkpointed |

**Usage:** `orch session status` shows checkpoint level with visual indicators and actionable guidance.

**Why these thresholds:** Prior evidence showed 5h sessions with partial outcomes. Duration is a practical proxy for context usage (maps roughly to token consumption).

---

## Key Concepts

| Concept | Definition | Why It Matters |
|---------|------------|----------------|
| **Frame Shift** | The transition from orchestrator to meta-orchestrator is a change in perspective, not just hierarchy | Dylan operates from the meta frame; Claude instances operate as orchestrators within that frame |
| **Level Collapse** | When an agent drops levels and does work below its station | Meta-orchestrator doing worker work wastes the strategic vantage point |
| **Session vs Issue** | Sessions are conversations; issues are work items | Orchestrators have sessions, workers have issues |
| **Tier** | The workspace tier (light, full, orchestrator) determining verification rules | Orchestrator tier skips beads checks, verifies SESSION_HANDOFF.md |
| **Interactive vs Spawned** | Interactive orchestrators use `orch session start/end`; spawned are completed by level above | Spawned orchestrators wait, they don't self-terminate |

---

## Common Problems

### "Spawned orchestrator tried to run orch session end"

**Cause:** ORCHESTRATOR_CONTEXT.md template contradicted the hierarchical completion model.

**Fix:** Spawned orchestrators should write SESSION_HANDOFF.md and WAIT. The level above runs `orch complete`. Template was fixed in Jan 2026.

**NOT the fix:** Adding more guardrails to the skill - the template framing sets behavioral mode.

### "Orchestrator doing worker-level work (frame collapse)"

**Cause:** Vague goals cause exploration mode which leads to investigation which leads to debugging. Framing cues override skill instructions.

**Fix options:**
1. Provide specific goals with action verbs, concrete deliverables, and success criteria
2. Add "Goal Refinement Before Spawn" - ask agent to state understanding before proceeding
3. Use the WHICH vs HOW test: meta decides WHICH focus, orchestrator decides HOW to execute

**NOT the fix:** Adding more ABSOLUTE DELEGATION RULE warnings - the agent already knows, but framing is stronger.

**Detection (Jan 2026):** Frame collapse requires EXTERNAL detection - orchestrators can't see their own frame collapse. Multi-layer approach:
1. **Skill guidance** with explicit time check: "If editing code for >15 minutes, you've frame collapsed"
2. **SESSION_HANDOFF.md** "Frame Collapse Check" section prompting reflection
3. **OpenCode plugin** potential: Track Edit tool usage on code files vs orchestration artifacts
4. **Meta-orchestrator review** of handoffs looking for "Manual fixes" sections

**Key trigger:** Failure-to-implementation pattern - after agents fail, orchestrator tries to "just fix it" instead of trying different spawn strategy.

### "orch complete fails for orchestrator sessions with bd show errors"

**Cause:** Complete command was checking workspace directory before registry, causing orchestrator workspace names to be treated as beads IDs.

**Fix:** Registry-first lookup. Complete now checks `~/.orch/sessions.json` first, then workspace, then beads. Fixed in Jan 2026.

### "Workspace name collision overwrote SESSION_HANDOFF.md"

**Cause:** Workspace names were deterministic (proj-skill-slug-date) with no uniqueness suffix.

**Fix:** Added 4-character random hex suffix to workspace names (e.g., og-orch-goal-05jan-a1b2).

### "Orchestrator skill loads for workers despite audience:orchestrator"

**Cause:** Session-context plugin checked ORCH_WORKER at plugin init (once) instead of in config hook (per-session).

**Fix:** Moved ORCH_WORKER check from plugin init to config hook for per-session filtering.

### "Tmux-spawned orchestrators don't capture .session_id"

**Cause:** Tmux spawns used standalone OpenCode mode (embedded server), sessions not visible via shared API.

**Fix:** Switch to attach mode with `--dir` flag. Sessions now registered with shared server at localhost:4096.

**Note:** Additional issue discovered - `FindRecentSession` matches by title, not workspace. May need `--title` flag or directory+time matching.

### "orch stats shows 0%/16.7% completion for orchestrators"

**This is BY DESIGN, not a bug.** Orchestrators are classified as `CoordinationSkill`:
```go
var coordinationSkills = map[string]bool{
    "orchestrator":      true,
    "meta-orchestrator": true,
}
```

**Key insight:** Orchestrators run until context exhaustion or session interruption, not until "Phase: Complete". The stats correlation also had a bug (used beads_id instead of workspace for matching), but even with fix, completion rate would be low by design.

**Recommendation:** Separate coordination skills from task skills in stats display to avoid misleading metrics.

### "Interactive sessions don't create workspaces"

**Cause:** `orch session start` only writes to `~/.orch/session.json`, doesn't create workspace directories.

**Fix:** Enhanced `orch session start` to create `~/.orch/session/{date}/` with SESSION_HANDOFF.md template.

**Key insight:** Two parallel session models exist - "tracked spawns" (full workspaces) and "lightweight sessions" (minimal). The gap was interactive sessions losing context on exit.

---

## Key Decisions (from investigations)

These are settled. Don't re-investigate:

- **Orchestrators ARE structurally spawnable** - SESSION_CONTEXT.md ↔ SPAWN_CONTEXT.md, SESSION_HANDOFF.md ↔ SYNTHESIS.md. The "not spawnable" perception was false.
- **Meta-orchestrator IS Dylan** (initially) - No automation needed. Dylan makes strategic decisions, spawns orchestrator sessions, reviews handoffs.
- **Three-tier hierarchy is descriptive, not prescriptive** - Dylan already operates as meta-orchestrator. Making it explicit adds infrastructure, not concepts.
- **Beads tracking inappropriate for orchestrators** - Orchestrators manage sessions, not issues. Session registry replaces beads for orchestrators.
- **Tmux default for orchestrator spawns** - Orchestrators need visibility for interactive work; workers default to headless.
- **CLI for orchestrator/scripts, MCP for agent-internal use** - Orchestrators use CLI commands, not MCP tools.
- **Signal ratio matters in skill documents** - 4:1 ask-vs-act ratio overwhelms specific "act silently" guidance. Rebalancing needed.
- **Interactive orchestrators serve legitimate functions** (Jan 2026) - NOT compensation for daemon gaps. Serve: (1) goal refinement through conversation, (2) real-time frame correction, (3) synthesis of worker results. Daemon automates dispatch, orchestrators provide direction and synthesis.
- **Checkpoint discipline via visibility** (Jan 2026) - 2h/3h/4h thresholds enforced via `orch session status` warnings, not hard blocks. Respects orchestrator judgment while surfacing risk.

---

## What Lives Where

| Thing | Location | Purpose |
|-------|----------|---------|
| Session Registry | `~/.orch/sessions.json` | Track active orchestrator sessions |
| Orchestrator Context Template | `pkg/spawn/orchestrator_context.go` | Generate ORCHESTRATOR_CONTEXT.md |
| Session State | `pkg/session/session.go` | Session lifecycle management |
| Completion Verification | `pkg/verify/check.go` | Tier-aware verification |
| Orchestrator Skill | `~/.claude/skills/meta/orchestrator/SKILL.md` | Orchestrator guidance |
| Meta-Orchestrator Skill | `~/.claude/skills/meta/meta-orchestrator/SKILL.md` | Meta-orchestrator guidance |
| SESSION_HANDOFF Template | `~/.orch/templates/SESSION_HANDOFF.md` | Handoff artifact structure |

---

## Debugging Checklist

Before spawning an investigation about orchestrator session issues:

1. **Check kb:** `kb context "orchestrator session"`
2. **Check this guide:** You're reading it
3. **Check session registry:** `cat ~/.orch/sessions.json`
4. **Check workspace tier:** `cat .orch/workspace/<name>/.tier`
5. **Check for marker files:** `ls -la .orch/workspace/<name>/` (look for .orchestrator)

If those don't answer your question, then investigate. But update this guide with what you learn.

---

## The Orchestrator Autonomy Pattern

**When orchestrators should act silently (no announcement):**
- Complete agents at Phase: Complete
- Synthesize findings after completion
- Monitor spawned agents
- Check status periodically

**When orchestrators should propose-and-act (state intent):**
- Spawning agents
- Completing multiple agents in batch
- Checking on specific agents

**When orchestrators should ask (rare):**
- Multiple valid approaches with meaningful tradeoffs
- Scope/priority decisions
- Costly irreversible actions with unclear value

**Anti-pattern recognition:** "Want me to complete them?" is always wrong. "Option A it is" after presenting choices is wrong (wait for approval).

---

## The Spawn Improvement Loop

Meta-orchestrator's core workflow:

```
1. SPAWN with specific goal
2. OBSERVE agent behavior
3. REVIEW handoff (SESSION_HANDOFF.md or SYNTHESIS.md)
4. DIAGNOSE friction (what was harder than it should have been?)
5. IMPROVE next spawn (context, goal specificity, skill choice)
```

**Key insight:** The meta-orchestrator provides post-mortem perspective while the orchestrator is still working. This real-time diagnosis is the core value add.

---

## Verification by Tier

| Tier | Artifact Required | Beads Checks | Phase Reporting |
|------|-------------------|--------------|-----------------|
| light | None | Yes | Yes |
| full | SYNTHESIS.md | Yes | Yes |
| orchestrator | SESSION_HANDOFF.md | No | No |

**Implementation:** `pkg/verify/check.go:VerifyCompletionWithTier()` routes to tier-specific verification.

---

## Dashboard Context Following (Jan 2026)

**What:** Dashboard beads display follows orchestrator's current project context.

**How it works:**
1. API endpoints accept `project_dir` query parameter: `/api/beads?project_dir=/path/to/project`
2. Cache is per-project (keyed by directory) to support concurrent views
3. Frontend passes project_dir from orchestrator context to beads fetch calls
4. Reactive refetch on context change

**Troubleshooting:**
```
Dashboard slow/stale → orch doctor --fix → starts missing servers
Dashboard shows wrong project → check orchestrator context → tmux window may have stale pwd
```

**Quick reference:** `orch doctor --fix` resolves most dashboard issues by starting missing servers.

---

## References

- **Decision:** `.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md` - Frame shift concept
- **Decision:** `.kb/decisions/2026-01-04-orchestrator-session-lifecycle.md` - Hierarchical completion model
- **Investigation:** `.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md` - Session boundary patterns
- **Investigation:** `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md` - Spawnable infrastructure
- **Investigation:** `.kb/investigations/2026-01-04-inv-meta-orchestrator-level-collapse-spawned.md` - Frame collapse analysis
- **Source code:** `pkg/spawn/orchestrator_context.go` - Context generation
- **Source code:** `pkg/session/registry.go` - Session registry
- **Source code:** `cmd/orch/complete_cmd.go` - Completion flow

---

## History

- **2026-01-07:** Updated with 12 additional investigations covering: checkpoint discipline, frame collapse detection, stats correlation, dashboard context-following, session registry status updates, interactive vs spawned workspace differences
- **2026-01-06:** Created from synthesis of 28 orchestrator investigations
