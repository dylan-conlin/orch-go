# Model: Coordination

**Created:** 2026-03-09
**Updated:** 2026-03-18
**Status:** Active
**Source:** Synthesized from 4 investigation(s) + 1 controlled experiment (80 trials)

## What This Is

A model of multi-agent coordination failure in software engineering tasks. Describes when and why parallel agents produce individually correct work that cannot be merged, and which coordination mechanisms prevent versus fail to prevent these failures.

The core insight: **communication does not produce coordination.** Agents can understand each other's work, discuss plans, and acknowledge potential conflicts — yet still produce unmergeable code. Only structural constraints (explicit placement instructions) prevent conflicts. This challenges the premise of frameworks like CrewAI, AutoGen, and LangGraph that assume agent-to-agent messaging solves coordination.

---

## Core Claims (Testable)

### Claim 1: Communication is insufficient for coordination

Giving agents awareness of each other's work — whether through context sharing or active messaging — does not reduce merge conflict rates. Agents acknowledge potential conflicts in their plans but do not change their behavior to avoid them.

**Test:** 4-condition experiment × N=10: no-coord, placement, context-share, messaging
**Evidence:** context-share 20/20 CONFLICT, messaging 20/20 CONFLICT (despite 18/20 trials with both plans written)
**Status:** Confirmed (p=1.0, 80 trials)

### Claim 2: Structural placement prevents conflicts completely

Explicit, non-overlapping insertion point instructions (e.g., "place after function X" vs "place after function Y") prevent merge conflicts with 100% reliability, for both simple and complex tasks.

**Test:** Placement condition with specified insertion points
**Evidence:** placement 20/20 SUCCESS (clean merge + tests pass)
**Status:** Confirmed (p=1.0, 20 trials)

### Claim 3: Individual agent capability is not the bottleneck

All agents (160 total across 80 trials) achieved 6/6 individual scores regardless of condition or task complexity. The failure is entirely in the coordination mechanism, not in individual agent performance.

**Test:** Score agents individually before attempting merge
**Evidence:** 160/160 agents scored 6/6 (100% individual success rate)
**Status:** Confirmed

### Claim 4: Coordination failure is task-complexity-independent

Both simple tasks (FormatBytes + FormatRate, ~40s each) and complex tasks (VisualWidth + FormatTable, 50-170s each) show identical coordination patterns: 100% conflict without placement, 100% success with placement.

**Test:** Run same 4 conditions on simple and complex task pairs
**Evidence:** Identical results across both task types in all conditions
**Status:** Confirmed

---

## Implications

1. **Multi-agent frameworks that rely on messaging for coordination are fundamentally flawed.** CrewAI, LangGraph, Claude Agent SDK, OpenAI Agents SDK, and similar systems that assume agents can coordinate through communication will produce merge conflicts when agents modify the same files. The communication doesn't fail — the agents communicate perfectly. The coordination still fails. (Note: AutoGen, originally cited here, entered maintenance mode by Mar 2026, succeeded by Microsoft Agent Framework. The claim applies equally to its successor.)

2. **Effective multi-agent coordination requires structural constraints.** The harness (orchestrator) must assign non-overlapping work regions. This is a scheduling/allocation problem, not a communication problem. **Production validation (2026-03-18):** orch-go's daemon, swarm, and exploration mode all implement structural separation — routing agents to different issues/files rather than coordinating within shared files. The production architecture validates this claim by design.

3. **The compliance/coordination distinction is real.** Agents comply perfectly with individual task requirements (6/6 in all trials). They even comply with coordination instructions ("avoid conflicts"). But compliance is not coordination — understanding the goal doesn't produce the behavior.

4. **Agents choose the "correct" location over the "non-conflicting" location.** In messaging trials, both agents consistently chose to insert "after FormatDurationShort" because that's the semantically correct location per the task. Even when told about the other agent's identical plan, they didn't deviate. The task instruction is stronger than the coordination instruction.

---

## Boundaries

**What this model covers:**
- Same-file parallel editing coordination
- Git merge conflicts as coordination failure signal
- Communication vs structural coordination mechanisms
- Agent behavior under coordination instructions

**What this model does NOT cover:**
- Cross-file coordination (agents editing entirely different files)
- Sequential execution patterns (one agent after another)
- Human-in-the-loop coordination
- Non-git merge strategies (e.g., semantic merge tools)
- Tasks where agents naturally edit different regions

---

## Evidence

| Date | Source | Finding |
|------|--------|---------|
| 2026-03-09 | Pilot (N=1) | Both Haiku and Opus produce 6/6 individually, 100% merge conflict |
| 2026-03-09 | N=10 FormatBytes | 100% conflict rate at N=10, Fisher's exact p=1.0 |
| 2026-03-09 | Complex task (N=1) | 4-file conflicts including add/add type, semantic conflicts from design divergence |
| 2026-03-10 | 4-condition experiment (N=80) | Placement 20/20 success, all other conditions 60/60 conflict, 160/160 individual 6/6 |
| 2026-03-18 | Decay verification probe | All 4 claims confirmed current. Experiment data intact. Framework references updated (AutoGen → deprecated). Production architecture validates structural approach. |

---

## Key Experiment: 4-Condition Redesign (2026-03-10)

**Design:** 4 conditions × 2 task types (simple + complex) × N=10 = 80 trials, 160 agent invocations
**Model:** claude-haiku-4-5 (both agents)
**Tasks:** Agent A (FormatBytes/VisualWidth) + Agent B (FormatRate/FormatTable) modifying same files

| Condition | Mechanism | Simple | Complex |
|-----------|-----------|:------:|:-------:|
| no-coord | No mention of other agent | 10/10 CONFLICT | 10/10 CONFLICT |
| placement | Explicit non-overlapping insertion points | **10/10 SUCCESS** | **10/10 SUCCESS** |
| context-share | Each agent given other's full task description | 10/10 CONFLICT | 10/10 CONFLICT |
| messaging | Shared file-based coordination + task awareness | 10/10 CONFLICT | 10/10 CONFLICT |

**Messaging observation:** 18/20 messaging trials had BOTH agents write coordination plans. Plans consistently acknowledged "no conflicts expected" while choosing identical insertion points ("after FormatDurationShort"). Agents understand the conflict but don't resolve it.

**Results:** `experiments/coordination-demo/redesign/results/20260310-174045/`

---

## Open Questions

- Does placement work when the number of agents exceeds the number of natural insertion points?
- Can iterative messaging (multi-round negotiation) produce different results than single-shot plan exchange?
- Does a stronger coordination instruction ("you MUST choose a different insertion point than the other agent") change behavior?
- At what task granularity does structural placement become impractical?

## Source Investigations

### 2026-03-09-inv-coordination-failure-controlled-demo-same.md

**Delta:** Coordination failures when two agents implement the same feature are dominated by structural factors (same insertion points), not model capability — both Haiku and Opus scored 6/6 individually but produced 100% merge conflict rate.
**Evidence:** Pilot experiment: identical task (FormatBytes) given to Haiku (49s, 34 test cases) and Opus (63s, 24 test cases) in isolated worktrees; both achieved perfect individual scores; merge produced CONFLICT in both modified files (display.go, display_test.go); both independently generated identical commit messages.
**Knowledge:** Coordination failure is a protocol problem, not a capability problem — even the most capable model cannot avoid conflicts without coordination infrastructure (file-level locking, insertion-point reservation, or sequential execution).

---

### 2026-03-09-inv-coordination-demo-n10-formatbytes.md

**Delta:** At N=10, coordination failure rate is 100% for both Haiku and Opus, confirming the pilot finding that merge conflicts are structural (same insertion points in git), not capability-dependent.
**Evidence:** 20 agent runs (10 per model): both scored 6/6 individually in all trials; all 10 trial pairs produced merge conflicts; Fisher's exact test p=1.0; duration difference not significant (haiku 39.1s vs opus 44.0s, t=1.103, p>0.05).
**Knowledge:** For well-defined, unambiguous tasks, model capability does not affect coordination failure rate. The failure is entirely structural.

---

### 2026-03-09-inv-coordination-demo-complex-ambiguous.md

**Delta:** Complex/ambiguous multi-file task reveals capability differences in ambiguity resolution (Opus anticipates Unicode edge cases, produces stronger alignment tests) while coordination failure remains 100% structural — now with two conflict types: content conflicts AND add/add conflicts for new files.
**Evidence:** N=1 experiment: identical 4-file task given to Haiku and Opus; both 10/10; merge produces CONFLICT in all 4 files.
**Knowledge:** Binary compliance scoring cannot distinguish model capability. Coordination failure extends to new file creation.

---

### 2026-03-10 4-Condition Experiment

**Delta:** Communication (context sharing, active messaging) produces zero improvement over no coordination. Only structural placement instructions prevent conflicts. 80 trials, 160 agents, all individually 6/6.
**Evidence:** See Key Experiment section above.
**Knowledge:** Coordination is a structural problem, not a communication problem. Agents comply with coordination instructions without producing coordination outcomes.
