# Decision: Models as Understanding Artifacts

**Date:** 2026-01-12
**Status:** Accepted
**Context:** Strategic orchestrator shift revealed synthesis had no home

## Decision

Create `.kb/models/` as distinct artifact type for synthesized understanding. Models are where orchestrators externalize the mental models they build through direct engagement with investigations.

## Problem

The strategic orchestrator shift (Jan 7, 2026) established that orchestrators synthesize findings rather than delegating understanding to spawned agents. But we had no artifact type for what synthesis produces.

**The gap:**
- Investigations = probes (temporal, narrow questions)
- Decisions = choices (point-in-time)
- Guides = procedures (how to do X)
- Epics = work coordination (task lists)

**Where does synthesized understanding live?**

Current state had understanding scattered:
- Some in Epic Model template (conflated with work coordination)
- Some in guides (conflated with procedures)
- Some in orchestrator's head (not externalized)

## The Model Artifact Type

**Purpose:** Externalize synthesized understanding of how system components work, why they fail, and what constraints exist.

**Boundary test:**

| Question | Artifact Type |
|----------|---------------|
| "How does X work?" | Model (`.kb/models/`) |
| "How do I do X?" | Guide (`.kb/guides/`) |
| "What did we choose and why?" | Decision (`.kb/decisions/`) |
| "What work remains?" | Epic (beads) |
| "Does X behave this way?" | Investigation (`.kb/investigations/`) |

**Key distinction:** Models are descriptive (how system IS), guides are prescriptive (how to DO).

## What This Unlocks

Models create **surface area for questions** by making implicit constraints explicit.

**Example from first model** (`dashboard-agent-status.md`):

Before model:
- "Dashboard is confusing, agents show dead when they're done" (symptom)
- Constraint buried: OpenCode doesn't expose session state

After model:
- Constraint explicit in "Constraints" section
- Enables strategic question: "Should we add that endpoint to OpenCode?"

**The value:** Not just organization, but making understanding queryable and discussable.

## Provenance Chain Architecture

Models are **nodes in provenance chains, not endpoints.**

```
Primary evidence (code, tests, behavior)
    ↓ (referenced in)
Investigations (probe findings)
    ↓ (synthesized into)
Models (understanding)
    ↓ (inform)
Decisions (choices)
    ↓ (create)
Guides, Epics (downstream work)
```

Models must reference investigations (via "Synthesized From" section).
Investigations must reference code (via "Files Examined" section).
Chain terminates in observable reality.

## Relationship to Strategic Orchestrator Model

This decision completes the architecture established in Jan 7 decisions:

**Strategic Orchestrator Model said:**
- Orchestrators synthesize (don't delegate understanding)
- Daemon coordinates (spawn mechanics)
- Synthesis is engagement (not spawnable)

**But didn't say:** Where does synthesis output go?

**This decision says:** Synthesis produces models in `.kb/models/`

## Relationship to Principles

### Understanding Through Engagement (new principle)

Models are the artifact type that "Understanding Through Engagement" produces.

**The principle:** You can spawn work to gather facts, but synthesis into coherent models requires cross-agent context that only orchestrator has.

**The artifact:** Models are where that synthesis lives.

### Evidence Hierarchy (existing principle)

Models are secondary evidence (like investigations and decisions). They must trace to primary evidence (code).

Models that don't reference code are closed loops - violates Provenance principle.

## Epic Model Template Status

The Epic Model template (`~/.orch/templates/epic-model.md`) tried to bundle:
- Process scaffold (how to approach complex problems)
- Understanding artifact (the model you're building)
- Work tracking (sessions, probes, tasks)

Now that models are distinct, we should re-evaluate this template:
- Extract process guide → `.kb/guides/complex-problem-solving.md`
- Extract understanding → `.kb/models/{domain}.md`
- Simplify epics → beads issues that reference models

**Decision deferred:** How to handle Epic Model template. Will revisit after models prove themselves in practice.

## Migration Path

**Phase 1: Prove the premise** (Jan 12, 2026)
- ✅ Create `.kb/models/` directory
- ✅ Create first model: `dashboard-agent-status.md` (synthesized 11 investigations)
- ✅ Test: Does it create surface area for questions? (Yes - Dylan: "exactly what I was thinking!")

**Phase 2: Capture decision** (this document)

**Phase 3: Minimal infrastructure**
- Create model template (based on dashboard model structure)
- Add "Understanding Through Engagement" to principles.md
- Update orchestrator skill (synthesis → create model)

**Phase 4: Selective migration**
- Only move artifacts that are clearly models ("how X works")
- Leave procedural guides alone ("how to do X")
- Don't force migration - let models accumulate organically

## Boundary Examples (Sharp Edges)

| Artifact | Type | Why |
|----------|------|-----|
| "How dashboard calculates agent status" | **Model** | Descriptive mechanism |
| "How to debug completion verification" | **Guide** | Procedural steps |
| "Why we chose Priority Cascade" | **Decision** | Choice at point in time |
| "Best practices for spawning" | **Guide** (probably) | Procedural knowledge |
| "How spawn lifecycle works" | **Model** | Mechanism understanding |

**Edge case:** If guide needs to explain mechanism (not just procedure), it should reference a model for context.

Example: "How to debug completion" guide should link to "agent-lifecycle" model for understanding.

## Success Criteria

**How we'll know models work:**

1. **Orchestrators create models** after synthesizing 3+ investigations
2. **Dylan asks sharper questions** because constraints are explicit
3. **Decisions reference models** for context (provenance chain works)
4. **Duplicate investigations decrease** (model answers the question)
5. **Epic readiness increases** (model = understanding achieved)

**If models don't get created:** Process isn't working, need to revisit.

**If models get created but not referenced:** Discoverability problem, need to fix.

## Origin

Conversation with Dylan, Jan 12, 2026. Started from "Understanding Through Engagement" principle discussion, recognized synthesis needs artifact home. Created first model (`dashboard-agent-status.md`) as proof-of-concept. Dylan's reaction: "brings clarity to what has been opaque... this is invaluable."

## Related Decisions

- `2026-01-07-strategic-orchestrator-model.md` - Orchestrators synthesize (this completes it)
- `2026-01-07-synthesis-is-strategic-orchestrator-work.md` - Synthesis not spawnable
- `2025-11-28-evolve-by-distinction.md` - This decision makes explicit: models ≠ guides

## Open Questions

1. **Should Epic Model template be split?** Process guide + model template + simple epics?
2. **Should models have their own `kb` subcommands?** (`kb create model`, `kb list models`)
3. **How do we prevent model drift?** (Model says X, code does Y - who catches this?)
4. **When do investigations promote to models?** (3+ investigations? Pattern emerges? Orchestrator judgment?)

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-14-design-spawn-context-model-inclusion.md
