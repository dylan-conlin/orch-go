# Handoff: System Self-Reflection Epic (orch-go-ws4z)

**Created:** 2025-12-21
**Purpose:** Deep context for orchestrator session focusing on ws4z epic

---

## Epic Overview

**ID:** `orch-go-ws4z`
**Title:** System Self-Reflection - Temporal Pattern Awareness

**The Core Question:** The system addresses session amnesia but lacks project-lifetime memory. Artifacts accumulate without awareness of their relationships, evolution, or potential for synthesis. Humans must notice patterns manually.

**Goal:** Enable the system to surface temporal patterns, citation networks, and synthesis candidates autonomously.

**Success Criteria:**
- System can surface "you have 5 investigations about X with no synthesis" without human noticing first
- Citation/reference counting enables emergent hierarchy (load-bearing artifacts visible)
- Temporal drift detection identifies when decisions no longer match practice
- Chronicle artifact captures decision evolution narratives

---

## Genesis: How This Epic Was Born

This epic emerged from **post-synthesis reflection** after completing `orch-go-4kwt` (Amnesia-Resilient Artifact Architecture).

**The arc:**
1. `orch-go-4kwt` asked: "What artifacts exist and how do they relate?"
2. Six investigations answered this, synthesized into minimal artifact taxonomy
3. Dylan's reflection surfaced deeper questions that the taxonomy couldn't answer
4. Those questions became `orch-go-ws4z`

**Key insight from synthesis session:** The most valuable output came from interactive follow-up ("what else do you see?"), not autonomous completion. This pattern itself became issue `orch-go-4kwt.8` (now reparented to ws4z).

---

## Children (7 Issues)

| ID | Title | Type | Description |
|----|-------|------|-------------|
| `orch-go-ws4z.4` | Design: kb reflect command | feature | Command that surfaces synthesis/deprecation candidates |
| `orch-go-ws4z.6` | Design: Self-reflection protocol | feature | Synthesize all investigations into coherent protocol |
| `orch-go-ws4z.7` | Citation mechanisms and reference counting | task | How artifacts track citations to each other |
| `orch-go-ws4z.8` | Temporal signals for autonomous reflection | task | What signals trigger system to suggest reflection |
| `orch-go-ws4z.9` | Chronicle artifact type | task | New artifact capturing decision evolution narratives |
| `orch-go-ws4z.10` | Questioning inherited constraints | task | When/how to question constraints like local-first |
| `orch-go-4kwt.8` | Reflection checkpoint pattern | task | Pause-before-complete for richer synthesis (reparented) |

**Suggested order:**
1. `.7` (citations) and `.8` (temporal signals) - foundational investigations
2. `.9` (chronicle) and `.10` (constraints) - exploratory investigations  
3. `4kwt.8` (reflection checkpoint) - protocol enhancement
4. `.4` (kb reflect command) - depends on above findings
5. `.6` (self-reflection protocol) - synthesis of all above

---

## Foundational Context

### The Artifact Taxonomy (from 4kwt)

**5 Essential Artifacts:**
| Artifact | Location | Purpose |
|----------|----------|---------|
| SPAWN_CONTEXT.md | `.orch/workspace/{name}/` | Agent initialization |
| SYNTHESIS.md | `.orch/workspace/{name}/` | Session outcome (D.E.K.N.) |
| Investigation | `.kb/investigations/` | Deep research |
| Decision | `.kb/decisions/` | Architectural choices |
| Beads Comments | `.beads/` | Phase tracking |

**3 Supplementary Artifacts:**
- SESSION_HANDOFF.md - Cross-session orchestrator context
- FAILURE_REPORT.md - Context when agents fail
- kn entries - Quick decisions, constraints, tried/failed

**Three-Tier Temporal Model:**
- **Ephemeral (session-bound):** `.orch/workspace/`
- **Persistent (project-lifetime):** `.kb/`
- **Operational (work-in-progress):** `.beads/`

**D.E.K.N. Universal Handoff Structure:**
- **Delta:** What changed/was discovered
- **Evidence:** How we know (primary sources)
- **Knowledge:** What it means (insights, constraints)
- **Next:** What should happen (recommendation)

### Key Findings from 4kwt Investigations

**Knowledge Promotion Paths (.2):**
- 4 documented paths: kn→kb, investigation→decision, investigation→guide, constraint→principle
- CLI support exists: `kb promote`, `kb publish`
- Low promotion rate (39 kn → 1 kb decision) is intentional curation
- **Gap for ws4z:** No mechanism to surface "ready for promotion" candidates

**Session Boundaries (.3):**
- Worker: Phase Complete → /exit (strictly enforced)
- Orchestrator: Context full → session-transition skill (state-detected)
- Cross-session: SESSION_HANDOFF.md (manual)
- **Gap for ws4z:** Synthesis is post-hoc, not progressive

**Multi-Agent Synthesis (.5):**
- Workspace isolation prevents conflicts
- SYNTHESIS.md + orch review enables orchestrator synthesis
- 0 merge conflicts in 100+ commits
- **Gap for ws4z:** No cross-agent awareness (each agent is isolated)

---

## Key Questions This Epic Must Answer

1. **Citation Mechanisms (.7):**
   - What's the minimal citation mechanism? (frontmatter links? content parsing?)
   - How to track inbound links ("what cites this decision?")?
   - How to surface load-bearing artifacts (high citation count = foundational)?

2. **Temporal Signals (.8):**
   - Which signals have highest value/noise ratio?
   - Candidates: temporal density, repeated constraints, citation convergence, contradictions, staleness, concept drift
   - What's the trigger mechanism? (hook? daemon? command?)

3. **Chronicle Artifact (.9):**
   - Is Chronicle a new artifact type or a view over existing artifacts?
   - What's the structure? Timeline? Narrative? Graph?
   - Who creates it? Orchestrator synthesis? Automated from git history?

4. **Questioning Constraints (.10):**
   - What signals indicate a constraint may no longer serve?
   - How to distinguish "constraint is wrong" from "we're misapplying it"?
   - Should constraints have expiration dates or review triggers?

5. **Reflection Checkpoint (4kwt.8):**
   - Skill-level: Add reflection phase before completion?
   - Spawn-level: --interactive flag for pause-before-complete?
   - Artifact-level: Require "unexplored questions" section in SYNTHESIS.md?

6. **kb reflect Command (.4):**
   - `kb reflect --type synthesis` - investigations needing synthesis
   - `kb reflect --type stale` - decisions with no recent citations
   - `kb reflect --type drift` - practice diverged from decision
   - `kb reflect --type promote` - kn entries ready for kb promotion

7. **Self-Reflection Protocol (.6):**
   - How does the system develop institutional memory that transcends any single session?
   - Integration points: hooks, daemon, commands
   - Success metrics for "system is self-aware"

---

## Relevant Artifacts

### Decisions
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - **READ THIS FIRST**

### Investigations (from 4kwt - foundation for ws4z)
- `.kb/investigations/2025-12-21-design-deep-pattern-analysis-orchestration-artifacts.md` - Original pattern analysis
- `.kb/investigations/2025-12-21-design-minimal-artifact-taxonomy.md` - Synthesis investigation
- `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md` - How knowledge flows
- `.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md` - Session boundary patterns
- `.kb/investigations/2025-12-21-inv-beads-kb-workspace-relationships-how.md` - Three-layer architecture
- `.kb/investigations/2025-12-21-inv-multi-agent-synthesis-when-multiple.md` - Multi-agent coordination
- `.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md` - Failure capture gaps

### Synthesis
- `.orch/workspace/og-arch-synthesize-findings-investigations-21dec/SYNTHESIS.md` - 4kwt.7 synthesis that spawned ws4z

---

## Current System State

- **Focus:** System stability and hardening
- **Active work:** `orch-go-pe5d` (registry removal) - Phase 2 in progress
- **ws4z status:** All 7 children are ready to spawn

---

## Recommended Approach

1. **Start with investigations** - `.7` (citations) and `.8` (temporal signals) can run in parallel
2. **Let findings inform design** - Don't design `kb reflect` until citation/signal investigations complete
3. **Synthesis last** - `.6` (self-reflection protocol) synthesizes everything
4. **Use the pattern you're designing** - Apply reflection checkpoint pattern to your own work

---

## Commands

```bash
# View epic
bd show orch-go-ws4z

# View all children
bd show orch-go-ws4z.4
bd show orch-go-ws4z.6
bd show orch-go-ws4z.7
bd show orch-go-ws4z.8
bd show orch-go-ws4z.9
bd show orch-go-ws4z.10
bd show orch-go-4kwt.8

# Spawn investigations
orch spawn investigation "Citation mechanisms - how artifacts track references to each other" --issue orch-go-ws4z.7
orch spawn investigation "Temporal signals - what triggers autonomous reflection" --issue orch-go-ws4z.8

# Get context on related topics
kb context "artifact taxonomy"
kb context "knowledge promotion"
kb context "session boundaries"
```
