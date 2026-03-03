# Decision: Coherence Over Patches Principle

**Date:** 2026-01-04
**Status:** Accepted
**Context:** Dashboard status logic demonstrated that locally correct patches can produce globally incoherent code

## Decision

Add "Coherence Over Patches" as an LLM-First principle: When fixes accumulate in the same area, escalate to architect before the next patch.

## Context

The dashboard agent status logic in `serve_agents.go` became a canonical example of patch accumulation:

**The symptoms:**
- 10+ conditions scattered across 350+ lines
- SYNTHESIS.md check duplicated in two places (lines 862-868 and 909-930)
- Line 609 optimization caused idle agents to skip Phase: Complete checks
- 37% of commits in the 2-week period were "fix:" commits
- 454 investigations in .kb/, many clustered on dashboard/status topics

**Each fix was correct:**
- Agent A added Phase: Complete handling
- Agent B added the line 609 optimization for CPU
- Agent C added a second SYNTHESIS.md check for edge cases
- Agent D added beads issue closed detection

**The result was incoherent:**
- The optimization at line 609 inverted priority (checked session activity before Phase: Complete)
- Duplicate checks revealed missing model
- Investigation after investigation on the same topic

**Resolution required architect:**
- Designed "Priority Cascade Model" with explicit priority order
- Single `determineAgentStatus()` function replaced 10+ scattered conditions
- Removed the optimization that caused bugs

## Rationale

**Why existing principles didn't prevent this:**

| Principle | What It Enforced | Why It Wasn't Enough |
|-----------|------------------|---------------------|
| Verification Bottleneck | Human verifies each change | Each fix verified correctly - the design was wrong |
| Gate Over Remind | Can't skip completion | Agents completed properly |
| Evidence Hierarchy | Code is truth | Agents verified their own code changes |

The gap: No principle said "stop fixing and redesign." Each patch was locally validated but globally destructive.

**Why this is an LLM-First principle (not just system design):**

This isn't specific to our orchestration architecture. It's a fundamental pattern when multiple LLM agents modify shared code:
- Agents optimize locally (each fix makes sense)
- No agent has visibility into the full patch history
- Conditions accumulate because each agent adds their case
- The coherent model that would prevent all the edge cases never gets designed

**The math of patch accumulation:**

```
1 fix: bug fixed
3 fixes: pattern emerging
5 fixes: structural issue
10+ fixes: missing coherent model
```

Each additional fix has increasing probability of conflicting with prior fixes. The cost of the 10th fix exceeds the cost of redesigning after the 3rd.

## Tooling Built

Implemented `orch hotspot` command that detects areas needing architect intervention:

**Signals detected:**
- Git history: 5+ fix commits to same file in 4 weeks
- Investigation clustering: 3+ investigations on same topic (via kb reflect)

**Integration points:**
- `orch hotspot` CLI command for manual checks
- `orch spawn` integration: warns when spawning to hotspot area
- `orch daemon preview` integration: flags issues targeting hotspots

**The warning:**
```
⚠️  HOTSPOT DETECTED: serve_agents.go
    8 fix commits in last 28 days
    Recommend: orch spawn architect "design status model"
```

## Relationship to Other Principles

**Verification Bottleneck** and **Coherence Over Patches** are both stopping principles, but at different levels:

| Principle | Question | Action |
|-----------|----------|--------|
| Verification Bottleneck | "Did a human see this working?" | Slow down |
| Coherence Over Patches | "Should we be patching at all?" | Stop and redesign |

You can have perfect verification of each patch while the area descends into chaos. Verification ensures each step is correct; coherence ensures you're taking the right steps.

**Connection to Evolve by Distinction:**
Coherence Over Patches is the recognition that you're conflating "fixing bugs" with "improving the system." The distinction: bugs need fixes, design gaps need design.

## Evidence

- `orch-go/.kb/investigations/2026-01-04-design-dashboard-agent-status-model.md` - The canonical example
- `orch-go/.kb/investigations/2026-01-04-design-patch-density-architect-escalation.md` - The detection mechanism design
- `orch-go/.kb/investigations/2026-01-04-inv-implement-orch-hotspot-cli-command.md` - Tooling implementation
- `orch-go/.kb/investigations/2026-01-04-inv-integrate-hotspot-detection-into-orch.md` - Spawn integration
- `orch-go/.kb/investigations/2026-01-04-inv-add-hotspot-warnings-orch-daemon.md` - Daemon integration
