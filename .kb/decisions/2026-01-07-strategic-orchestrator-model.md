---
status: active
blocks:
  - keywords:
      - meta-orchestrator role
      - orchestrator synthesis
      - spawn architect to think
      - delegate understanding
---

# Decision: Strategic Orchestrator Model

**Date:** 2026-01-07
**Status:** Accepted
**Context:** Rethinking orchestrator role after meta-orchestrator experiment

## Decision

Collapse meta-orchestrator and orchestrator into a single **strategic orchestrator** role. The strategic orchestrator's job is **comprehension**, not coordination. Coordination is the daemon's job.

## Context

The meta-orchestrator experiment (Jan 4-6, 2026) was valuable but possibly premature:

- It produced real insights: "Perspective is Structural", "Escalation is Information Flow"
- But it may not need to be a permanent separate role
- The key insight was about *perspective*, not about needing two orchestration layers

Meanwhile, a pattern emerged: orchestrators were being used as spawn-dispatchers, outsourcing *understanding* to architects and design-sessions. The system was optimized for **throughput** when what's needed is **understanding**.

## The Model

### What Changed

| Aspect | Old Model | Strategic Model |
|--------|-----------|-----------------|
| Orchestrator's job | "What should we spawn next?" | "What do we need to understand?" |
| Coordination | Orchestrator decides what/when | Daemon handles (triage:ready → spawn) |
| Synthesis | Spawned work (architect, design-session) | Orchestrator work (direct engagement) |
| Epic readiness | Task list complete | Model complete (understanding achieved) |
| Hierarchy | Worker → Orchestrator → Meta-Orchestrator → Dylan | Worker → Strategic Orchestrator → Dylan |

### The Work Division

| Work Type | Who Does It | Why |
|-----------|-------------|-----|
| Investigation (discovering facts) | Worker agent | Requires codebase exploration |
| Implementation (writing code) | Worker agent | Requires file editing |
| Synthesis (combining findings) | Strategic orchestrator | Requires cross-agent context |
| Understanding (building models) | Strategic orchestrator | Requires engagement, not delegation |
| Coordination (what to spawn when) | Daemon | Already automated |

### Epic Readiness = Model Completeness

An epic is "ready" not when tasks are listed, but when you can explain:

- What problem we're solving (not symptoms)
- Why previous approaches failed
- What the key constraints are
- Where the risks live
- What "done" looks like

**Test:** Can you write a 1-page document that would let a fresh agent implement without confusion? If yes, you understand it. If no, you're still probing.

### The Probe Model

Each issue in an epic is a **probe into understanding**. You're not collecting tasks - you're collecting evidence until the model coheres. The epic is ready when the orchestrator holds a coherent model, not when a list exists.

## What This Rejects

- Spawning architects to "think for me"
- Design-sessions as outsourced understanding
- Synthesis as spawnable work
- Orchestrator as spawn-dispatcher
- Meta-orchestrator as permanent role

## What This Embraces

- Direct orchestrator engagement with knowledge
- Daemon handling coordination mechanics
- Synthesis as orchestrator core competency
- Understanding as the goal, not task completion
- Dylan as the perspective check (not a meta-orchestrator layer)

## Relationship to Other Decisions

- **Synthesis is Strategic Orchestrator Work** (2026-01-07): Specific instance of this model
- **Meta-Orchestrator Frame Shift** (2026-01-04): The experiment that revealed these patterns
- **Perspective is Structural** (principles.md): The hierarchy insight that survives without a separate meta-orchestrator role

## Implementation

The orchestrator skill needs updates:

1. Remove "spawn architect to think" patterns
2. Add synthesis as explicit orchestrator work
3. Clarify daemon relationship (daemon coordinates, orchestrator comprehends)
4. Add epic readiness gate (UNDERSTANDING.md or equivalent)

## Open Questions

- How should reflection surface opportunities? (Dashboard? `orch status`? Session start prompt?)
- What triggers orchestrator synthesis? (On demand? Threshold? Part of epic composition?)
- Does Dylan need a mechanism to catch strategic orchestrator dropping into tactical mode?

## Origin

Session with Dylan, 2026-01-07. Started with "maybe I don't need meta-orchestrator and orchestrators - maybe just one strategic orchestrator." Evolved through exploring the duplicate synthesis issue, which revealed the deeper question about what synthesis is and who should do it.

---

## Refinement: Context-Scoping as Irreducible Function (2026-01-19)

**Context:** While building the Decidability Graph model, questioned whether workers could answer "framing questions" if given the right context.

**Discovery:** The hierarchy isn't about reasoning capability - workers can do any kind of reasoning (factual, design, even framing) IF they have the right context loaded. The irreducible orchestrator function is **deciding what context to load**, not performing synthesis itself.

### What This Refines

| Original Model | Refined Understanding |
|----------------|----------------------|
| Orchestrator *does* synthesis | Orchestrator *scopes* what gets synthesized |
| Synthesis is not spawnable | Synthesis is spawnable once scoped |
| "Spawn architect to think" is wrong because architect can't think strategically | "Spawn architect to think" is wrong because it abdicates the scoping decision |
| Authority is role-based | Authority is context-scoping-based |

### The Authority Chain (Refined)

- **Daemon:** Executes pre-scoped work (context already defined by spawn)
- **Orchestrator:** Scopes context for workers (decides what frames/knowledge to load)
- **Dylan:** Scopes context for orchestrator (or overrides scoping decisions)

### What Remains True

The original decision's core insight holds: "spawn architect to think for me" is still an anti-pattern. The refinement clarifies *why* - not because workers can't think, but because the orchestrator is abdicating its irreducible function (scoping).

### What This Enables

A "frame-evaluator" worker is now conceivable: spawn with multiple frames as context, instruction to evaluate from outside. The orchestrator's contribution is *deciding those were the relevant frames to compare*.

### What Remains Irreducibly Human (Dylan)

- Overriding scoping decisions ("you're looking at the wrong thing")
- Value judgments that determine which frames matter
- Accountability for where the system points its attention

**Reference:** `.kb/models/decidability-graph.md` - "The Irreducible Function: Context Scoping" section
**Quick decision:** `kb-227b01`
