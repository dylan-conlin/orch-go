# Session Handoff

**Orchestrator Session:** og-work-understand-meta-orchestration-04jan
**Goal:** Understand the meta-orchestration system
**Duration:** 2026-01-04 17:34 → ~17:55
**Outcome:** accomplished

---

## Summary

Successfully analyzed the meta-orchestration system architecture. The system implements a three-tier hierarchy (meta-orchestrator → orchestrator → worker) where each level manages the level below it, with distinct artifacts, responsibilities, and completion protocols.

---

## Key Findings: How the Pieces Fit Together

### 1. The Three-Tier Hierarchy

| Level | What It Manages | Artifact | Scope |
|-------|-----------------|----------|-------|
| **Worker** | Code/tasks | SYNTHESIS.md | Single issue |
| **Orchestrator** | Workers | SESSION_HANDOFF.md | Single project, single focus |
| **Meta-Orchestrator** | Orchestrator sessions | Cross-session synthesis | Cross-project, multi-session |

**Frame Shift Insight:** Each level thinks ABOUT the level below, not AS the level below:
- Workers think AS implementers
- Orchestrators think ABOUT workers (patterns, delegation, synthesis)
- Meta-orchestrators think ABOUT orchestrators (strategic focus, system evolution)

### 2. Skill Inheritance System

**Location:** Skills live at `~/.claude/skills/`
- `meta/orchestrator/SKILL.md` - The orchestrator skill (~1,900 lines)
- `meta/meta-orchestrator/SKILL.md` - The meta-orchestrator skill (~627 lines)

**Dependency mechanism:** Via skill frontmatter:
```yaml
dependencies:
  - orchestrator
```

When spawning, `skills.LoadSkillWithDependencies()` resolves the dependency chain and concatenates skill content. The meta-orchestrator inherits the full orchestrator skill.

### 3. Orchestrator Session Plugin

**Location:** `plugins/orchestrator-session.ts`

**Two responsibilities:**
1. **Config hook:** Injects orchestrator skill into instructions at session start
2. **Event hook:** Auto-starts `orch session start` on `session.created` event

**Worker detection:** Plugin skips processing for workers by checking:
- `ORCH_WORKER=1` environment variable
- `SPAWN_CONTEXT.md` existence
- Path containing `.orch/workspace/`

### 4. Spawnable Orchestrators

**Detection:** Skill type in frontmatter triggers orchestrator handling:
```yaml
skill-type: policy  # or "orchestrator"
```

**In spawn_cmd.go (line 574):**
```go
isOrchestrator = metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"
```

**Differences from worker spawns:**

| Aspect | Worker | Orchestrator |
|--------|--------|--------------|
| Context file | SPAWN_CONTEXT.md | ORCHESTRATOR_CONTEXT.md |
| Completion artifact | SYNTHESIS.md | SESSION_HANDOFF.md |
| Exit command | `/exit` | `orch session end` |
| Default spawn mode | Headless | Tmux (visible) |
| Beads tracking | Per-issue | Per-session (no issue tracking) |
| Marker file | `.tier` | `.orchestrator` |

**ORCHESTRATOR_CONTEXT.md template:** `pkg/spawn/orchestrator_context.go` - Contains session goal, authority levels, escalation triggers, and completion protocol.

### 5. The WHICH vs HOW Distinction

**Meta-orchestrator makes WHICH decisions:**
- Which project gets attention?
- Which epic is priority?
- When to shift focus?

**Orchestrator makes HOW decisions:**
- Which skill for this issue?
- How to verify completion?
- When to spawn vs wait?

**The test:** "Is this about direction or execution?"

### 6. Key Guardrails

**Meta-orchestrator guardrails:**
- ⚠️ Don't Micromanage (let orchestrators operate autonomously)
- ⚠️ Don't Compensate (pressure over compensation - let gaps surface)
- ⚠️ Don't Bottleneck (review outcomes, not individual actions)
- ⚠️ Don't Skip Handoff Review (SESSION_HANDOFF.md is your learning loop)
- ⚠️ Don't Drop Levels (manage one level down, not two)

### 7. Architecture Recommendation (from prior investigation)

The Jan 4 architect session concluded: **Don't build new meta-orchestrator infrastructure.** Instead, incrementally enhance existing `orch session` with:
1. Verification gate (`--require-handoff` flag)
2. Dashboard visibility (`/api/session` endpoint)
3. Pattern analysis (`kb reflect --type orchestrator`)

The meta-orchestrator role is currently filled by Dylan interactively.

---

## Architectural Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     META-ORCHESTRATOR                           │
│   • Strategic focus (which project/epic)                        │
│   • Reviews SESSION_HANDOFF.md from orchestrator sessions       │
│   • System evolution decisions                                  │
│   Currently: Dylan (interactive), spawnable via meta-orch skill │
└──────────────────────────┬──────────────────────────────────────┘
                           │ spawns/reviews
┌──────────────────────────▼──────────────────────────────────────┐
│                      ORCHESTRATOR                               │
│   • Spawned with `orch spawn meta-orchestrator "goal"`          │
│   • Gets ORCHESTRATOR_CONTEXT.md (not SPAWN_CONTEXT.md)         │
│   • Skill loaded: orchestrator + meta-orchestrator (merged)     │
│   • Produces SESSION_HANDOFF.md                                 │
│   • Ends with `orch session end` (not /exit)                    │
└──────────────────────────┬──────────────────────────────────────┘
                           │ spawns via `orch spawn`
┌──────────────────────────▼──────────────────────────────────────┐
│                        WORKER                                   │
│   • Spawned with `orch spawn <skill> "task"`                    │
│   • Gets SPAWN_CONTEXT.md with skill embedded                   │
│   • Produces SYNTHESIS.md                                       │
│   • Ends with `/exit`                                           │
└─────────────────────────────────────────────────────────────────┘
```

---

## Work Completed

### Context Gathered
- Read meta-orchestrator skill (627 lines) at `~/.claude/skills/meta/meta-orchestrator/SKILL.md`
- Read orchestrator skill (inherited) at `~/.claude/skills/meta/orchestrator/SKILL.md`
- Read orchestrator-session plugin at `plugins/orchestrator-session.ts`
- Read prior architect synthesis from `og-arch-meta-orchestrator-architecture-04jan`
- Read spawn infrastructure: `cmd/orch/spawn_cmd.go`, `pkg/spawn/orchestrator_context.go`, `pkg/spawn/config.go`

### No Agents Spawned
This was a synthesis session - all work done through reading and analysis.

---

## Active Work

No active agents or pending work from this session.

---

## Recommendations for Next Session

**Immediate Priority:** None - this was an understanding session.

**Follow-up opportunities:**
1. The three features from the architect session are still not implemented:
   - Verification gate (`--require-handoff` on `orch session end`)
   - Dashboard visibility for orchestrator sessions
   - `kb reflect --type orchestrator` for pattern detection

**Key insight to remember:** The meta-orchestrator skill is already implemented and spawnable. The gap is in verification and automation, not in the spawn mechanism itself.

---

## Context to Remember

- **Skill type "policy" or "orchestrator"** triggers orchestrator handling in spawn
- **Skills inherit via `dependencies:` frontmatter** - loaded by `LoadSkillWithDependencies()`
- **Plugin system works via config and event hooks** - orthogonal to skill system
- **Frame shift is the key concept** - each level thinks ABOUT the level below

---

## Session Metadata

**Skill:** meta-orchestrator
**Workspace:** `.orch/workspace/og-work-understand-meta-orchestration-04jan/`
