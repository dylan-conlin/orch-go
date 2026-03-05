# Spawning Orchestrator Sessions

Orchestrator sessions are spawnable entities, just like workers. You manage them the same way orchestrators manage workers.

---

## Goal Refinement Before Spawn (Core Responsibility)

**Before any spawn, translate strategic intent into actionable orchestrator goals.**

**The problem:** Vague goals cause orchestrator frame collapse. "Work on orch-go" → exploration → investigation → debugging (dropped two levels).

**The workflow:**

```
1. UNDERSTAND INTENT    →  What does Dylan actually want to accomplish?
2. CHECK PROJECT STATE  →  What's the current backlog, blockers, momentum?
3. IDENTIFY SPECIFIC GOAL  →  What CONCRETE outcome should this session produce?
4. PROPOSE WITH CRITERIA   →  Present goal + success criteria for confirmation
5. SPAWN ON CONFIRMATION   →  Only spawn after Dylan confirms
```

**What makes a good orchestrator goal:**

| Element | Example | Anti-Pattern |
|---------|---------|--------------|
| **Action verb** | "Ship", "Triage", "Complete" | "Work on", "Look at", "Explore" |
| **Concrete deliverable** | "auth feature merged to main" | "make progress on auth" |
| **Success criteria** | "all auth issues closed + integration tested" | "feel good about auth state" |
| **Scope boundary** | "auth feature only, defer rate-limiting" | no boundaries |

**Example goal refinement dialogue:**

```
Dylan: "Let's work on orch-go today"

Meta-orchestrator: "Checking orch-go backlog... I see 12 ready issues. 
The daemon reliability epic is 80% done with 3 remaining issues.
Recommend: 'Complete daemon reliability epic - close remaining 3 issues, 
run integration audit, push to main.'
Does that match your intent?"

Dylan: "Yes, let's do that"

Meta-orchestrator: *spawns orchestrator with refined goal*
```

**The discipline:** Never spawn an orchestrator with the exact words Dylan used if those words are vague. Your job is to refine them.

---

## The Spawn Command

```bash
orch spawn orchestrator "focus goal" --project <project>
```

## Model Requirement for Orchestrator Spawns (Critical)

Orchestrator spawns require a SOTA reasoning model. Do not spawn orchestrators on Sonnet-tier models.

**Approved model class examples:**
- Opus-tier reasoning models (for example, Opus 4.5)
- Gemini 3 Flash Preview
- GPT-5.2

**Default when uncertain:** Use Opus-tier reasoning.

**Recommended explicit spawn:**

```bash
orch spawn orchestrator "focus goal" --project <project> --model <sota_reasoning_model>
```

**Parameters:**
- `"focus goal"` - The strategic objective for this session (e.g., "Ship auth feature", "Triage orch-go backlog")
- `--project` - The project this orchestrator will focus on
- `--model` - Required for orchestrator spawns unless default is already set to a SOTA reasoning model

**What happens:**
1. Creates workspace at `.orch/workspace/orchestrator-{name}/`
2. Generates ORCHESTRATOR_CONTEXT.md (parallel to SPAWN_CONTEXT.md)
3. Opens tmux window (orchestrators are visible, not headless)
4. Loads orchestrator skill with session context
5. Gates completion on SESSION_HANDOFF.md production

---

## ORCHESTRATOR_CONTEXT.md

Like SPAWN_CONTEXT.md for workers, this provides session context:

```markdown
# Orchestrator Session Context

**Focus:** [The strategic goal]
**Project:** [Primary project]
**Started:** [Timestamp]
**Prior Handoff:** [Link to previous SESSION_HANDOFF.md if resuming]

## Current State
[Current `orch status` + `bd ready` output - active, stuck, ready]

## Session Scope
[Expected duration, checkpoint strategy]

## Authority
- Tactical decisions within focus: Orchestrator decides
- Strategic focus changes: Escalate to meta-orchestrator
- System evolution: Escalate to meta-orchestrator
```

---

## Visibility: Tmux, Not Headless

| Entity | Default Visibility | Why |
|--------|-------------------|-----|
| Workers | Headless (HTTP API) | High volume, automation-friendly |
| Orchestrators | Tmux window | Interactive, need observation |
| Meta-orchestrator | Direct conversation | Strategic, human-involved |

Orchestrator sessions appear in tmux so you can observe their progress, send guidance, and intervene if needed.

---

## Duration Expectations

| Worker | Orchestrator Session | Meta-Orchestrator Session |
|--------|---------------------|---------------------------|
| 1-4 hours | 2-8 hours (focus block) | Hours to days (strategic arc) |

Orchestrator sessions are longer than worker sessions. They may:
- Spawn multiple workers
- Wait for workers to complete
- Synthesize across workers
- Iterate until focus goal achieved

---

## When to Spawn a New Orchestrator Session

- New strategic focus (different goal)
- Different project (context switch)
- Prior session handed off and new work needed
- Context exhaustion (current session degraded)

**Don't spawn new session for:**
- Continuing same focus after interruption
- Minor scope adjustment within focus
- Just checking status
