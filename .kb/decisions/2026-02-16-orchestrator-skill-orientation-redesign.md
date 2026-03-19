# Decision: Orchestrator Skill Orientation Redesign

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** 
**Supersedes:** ~/Documents/personal/orch-go/.kb/investigations/2026-02-16-design-orchestrator-skill-orientation-redesign.md
**Superseded-By:** 


**Date:** 2026-02-16
**Status:** Accepted
**Context:** Synthesized from investigation analyzing orchestrator skill organizing principle and completion review disconnect

## Summary

The orchestrator skill should be reorganized around Dylan's orientation moments (spawn, during work, completion, session boundaries) instead of orchestrator actions (COMPREHEND → TRIAGE → SYNTHESIZE). Adds ORIENTATION_FRAME capture at spawn time and three-layer reconnection at completion time.

## The Problem

The orchestrator skill (~640 lines) is organized around what the orchestrator *does* rather than what it *achieves*. Each section was added in response to real crises and is locally correct, but they don't compose into a coherent experience. Key symptoms:

1. "Dylan's Reality" section is at line 464 (past halfway) — buried
2. Completion review starts from SYNTHESIS.md (agent-centric) not spawn-time motivation (Dylan-centric)
3. Session start protocol is buried in Meta-Orchestrator Interface
4. Spawn section focuses on mechanics without capturing "why Dylan cares"

The core insight: "why Dylan cares" **decays between spawn and completion**, and nothing in the system preserves it.

## The Decision

### Hybrid Reorganization: Four Orientation Moments

Top-level sections reorganized from action-based to moment-based:

```
1. IDENTITY: Orientation Keeper (ORIENT → DELEGATE → RECONNECT)
2. AT SPAWN TIME: Establish the Frame Together
3. DURING WORK: Maintain Narrative Continuity
4. AT COMPLETION: Three-Layer Reconnection
5. AT SESSION BOUNDARIES: "Previously On..."
6. HARD CONSTRAINTS (Inviolable)
7. REFERENCE
```

### ORIENTATION_FRAME at Spawn Time

Before spawning, the orchestrator captures why Dylan cares:
- Written in Dylan's terms, not technical terms
- Recorded in SPAWN_CONTEXT.md under `ORIENTATION_FRAME:` heading
- Recorded in beads via `bd comment <id> "FRAME: <why Dylan cares>"`
- Survives to completion time for reconnection

### Three-Layer Completion (within Gate 1)

Gate 1 (comprehension) restructured as three cumulative layers:

1. **Frame Reconnection** — Read spawn-time FRAME, reconnect Dylan: "Remember [problem in Dylan's words]?"
2. **Resolution** — Present what happened via SYNTHESIS.md: "[Problem] is [resolved/not]. Here's what changed."
3. **Contextual Placement** — Place within active threads: "This was N of M for [thread]. Remaining: [...]"

Gate 2 (behavioral) is preserved unchanged.

### All Hard Constraints Preserved

All five load-bearing patterns (ABSOLUTE DELEGATION RULE, Filter before presenting, Surface decision prerequisites, Pressure Over Compensation, Mode Declaration Protocol) map into the new structure without loss.

## Why This Design

### Principle: Progressive Disclosure

Lead with what matters (Dylan's orientation), details follow. Most orchestrator interactions hit 1-2 moments, not all 4.

### Principle: Surfacing Over Browsing

Surface "what to do NOW" based on the current moment, don't require navigation to the right section.

### Principle: Session Amnesia

State must externalize to files. ORIENTATION_FRAME captures the "why" at spawn time when motivation is fresh, not at completion time when it must be reconstructed.

### Trade-offs Accepted

1. **COMPREHEND → TRIAGE → SYNTHESIZE demoted** — From organizing principle to mechanism description. May reduce clarity about "what the orchestrator is."
2. **Added spawn step** — FRAME capture adds a step to spawn workflow (risk: ignored under pressure)
3. **More structured completion** — Three layers add ceremony; light completions abbreviate

## Implementation Status

**Partially implemented:**
- `ORIENTATION_FRAME` field added to `pkg/spawn/context.go` SPAWN_CONTEXT.md template
- Full skill reorganization not yet applied to `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/`

**Remaining:**
- Rewrite orchestrator skill template with four-moment structure
- Add `orch spawn --frame` CLI flag (Phase 3)
- Update orchestrator session lifecycle model with orientation state dimension

## Blocking Questions (for Dylan)

1. Should the identity statement change from "Strategic Comprehender" to "Orientation Keeper"?
2. How much implementation friction is acceptable for frame capture at spawn time? (soft prompt vs required field vs blocking gate)

## Evidence

- **Investigation:** `.kb/investigations/2026-02-16-design-orchestrator-skill-orientation-redesign.md`
- **Current skill:** `~/.claude/skills/meta/orchestrator/SKILL.md` (640 lines)
- **Skill template:** `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template`
- **Model:** `.kb/models/orchestrator-session-lifecycle/model.md`

## Related Decisions

- `2026-02-14-verifiability-first-hard-constraint.md` — Two-gate verification (preserved unchanged)
- `2026-01-04-orchestrator-session-lifecycle.md` — Session lifecycle (extended with orientation state)

## Auto-Linked Investigations

- .kb/investigations/archived/2026-02-16-design-orchestrator-skill-orientation-redesign.md
- .kb/investigations/2026-02-28-design-orientation-frame-gate-friction-biggest.md
- .kb/investigations/archived/2026-01-13-inv-add-orchestrator-skill-decision-tree.md
