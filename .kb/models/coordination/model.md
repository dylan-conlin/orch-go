# Model: Coordination

**Created:** 2026-03-09
**Updated:** 2026-03-22
**Status:** Active
**Source:** Synthesized from 4 investigation(s) + 1 controlled experiment (80 trials) + external validation (6 independent sources)

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

## Four Coordination Primitives (Generalized Framework)

The experimental findings generalize into four structural primitives required for multi-agent coordination. External validation (2026-03-22) confirms these are general to any multi-agent system, not specific to orch-go.

| Primitive | What It Does | orch-go Implementation | External Validation |
|-----------|-------------|----------------------|---------------------|
| **Route** | Agents don't collide — work is assigned to non-overlapping regions | Structural placement, file-level routing, issue-level separation | CrewAI's core failure is broken routing (GitHub #4783). DeepMind: centralized routing reduces error amplification from 17.2x to 4.4x |
| **Sequence** | Work happens in the right order | Spawn → implement → verify pipeline, daemon triage ordering | McEntire: pipeline architecture (broken sequence) achieved 0% success. MAST FM-1.3, FM-1.5, FM-2.1 |
| **Throttle** | Velocity doesn't exceed verification bandwidth | Accretion gates, completion review, spawn rate limiting | Anthropic: 15x token consumption in multi-agent. McEntire: pipeline consumed $50 budget on planning alone |
| **Align** | Agents share a current, accurate model of what correct means | Skills, CLAUDE.md, governance hooks, shared knowledge base | 50% of MAST failures (7/14 modes). Most neglected primitive across all frameworks |

**Key insight:** Align is the highest-leverage primitive and the validity condition for the other three. Route/Sequence/Throttle can mechanically operate without Align (messages get delivered, steps get ordered, velocity gets limited), but their coordination value degrades proportionally to Align quality. McEntire confirms: partial Align → 64% success, zero Align → 0%. This is a multiplier relationship, not a substrate dependency. External evidence confirms: agents communicating perfectly while maintaining divergent models of correctness is the dominant failure pattern. (Updated 2026-03-22: "meta-primitive"/"substrate" language replaced after falsification probe found 5 cases where Route/Sequence/Throttle mechanically hold while Align is broken — see probes/2026-03-22-probe-falsify-align-as-substrate.md.)

**Degenerate case:** When N=1 (single agent), all four primitives are trivially satisfied. This explains why autoresearch succeeds with radical simplicity — it eliminates coordination rather than solving it.

**Quantitative relationship:** Success degrades monotonically with missing primitives:
- McEntire: 100% (single/0 missing) → 64% (hierarchical/~1.5 missing) → 32% (swarm/~3 missing) → 0% (pipeline/~4 missing)
- DeepMind: 17.2x error amplification (independent/no primitives) → 4.4x (centralized/+Route+Sequence)

### Control Theory Mapping (Structural Homology)

The four primitives map onto control theory components. The mapping is structural homology (qualitative insights transfer) not isomorphism (quantitative formal tools do not transfer).

| Primitive | Control Component | Mapping Quality | Notes |
|-----------|------------------|-----------------|-------|
| **Route** | Actuator (directs work) | Good (primary match) | FM-2.5 has sensor bleed but primary is actuator |
| **Sequence** | Reference signal (target trajectory) | Weak (1/3 clean) | Sequence failures almost always involve sensing position on trajectory |
| **Throttle** | Controller (regulates rate) | Weak (single example, messy) | Premature termination involves both controller and sensor |
| **Align** | Sensor (observes correctness) | Strong (6/7 clean) | One exception: FM-2.6 (reasoning-action mismatch) maps to actuator, not sensor |

**Sensor bleed pattern:** 4 of 5 non-clean mappings share a structure: sensor components appear in failures mapped to non-sensor primitives. Control theory explains this: when the sensor fails, other components appear to fail because they depend on feedback. This gives theoretical grounding to "Align is the meta-primitive" — sensors are load-bearing for all other components.

**Sensor involvement across all MAST modes:** 11/14 (79%) involve sensors. This converges with the open-loop thread's independent finding that 14/16 (87.5%) of orch-go system failures are missing sensors. Two independent scopes, same structural finding: most failures are observation failures.

**What transfers from control theory:**
- Sensor dominance: removing the sensor (opening the loop) is the most catastrophic failure
- Loop closure: every effective system needs closed feedback
- Sensor cascade: many "actuator" or "controller" failures are caused by bad sensor data

**What does NOT transfer:** Stability analysis (Bode/Nyquist), optimal control (LQR/MPC), formal controllability/observability — these require quantitative transfer functions that agent coordination doesn't have.

**Evidence:** Probe 2026-03-22, mapping all 14 MAST failure modes to control components. See `.kb/models/coordination/probes/2026-03-22-probe-control-theory-component-mapping.md`.

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
| 2026-03-22 | External framework validation probe | All 4 claims confirmed as general (not orch-go-specific). 14 MAST failure modes map to 4 primitives. McEntire experiment shows monotonic degradation. DeepMind scaling paper confirms centralized coordination reduces error amplification. Align identified as dominant/neglected primitive (50% of failures). |
| 2026-03-22 | Control theory component mapping probe | Primitives map to control components (Route→Actuator, Sequence→Reference, Throttle→Controller, Align→Sensor) with 64% clean mapping. Sensor bleed pattern: 11/14 MAST modes involve sensors (79%), converging with open-loop thread's 87.5%. Structural homology, not isomorphism. |
| 2026-03-22 | Align-as-substrate falsification probe | "Substrate" overclaims — 5 cases show Route/Sequence/Throttle mechanically holding while Align is broken (MAST FM-1.1, McEntire hierarchical 64%, launchd post-mortem, orch-go competing instructions, stale knowledge cascades). Align is a multiplier/validity condition with proportional (not binary) impact. "Meta-primitive" language replaced. |

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
- Should Align decompose into sub-primitives? (It covers 50% of external failures — may be "state alignment" + "goal alignment"). **New evidence (2026-03-22):** 80-trial messaging condition shows task alignment defeating coordination alignment in 18/20 trials — agents agreed on the "correct" insertion point (task Align intact) while failing to coordinate (coordination Align broken). At minimum three sub-components: task alignment, state alignment, coordination alignment.
- Do the four primitives have ordering dependencies? (Must Route precede Sequence?)
- How do primitives interact with task type? (DeepMind found coordination strategy is task-dependent — financial reasoning favors centralized, web navigation favors decentralized)
- Do the primitives apply to non-LLM multi-agent systems? (robotics, distributed computing, human organizations)

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
