# Model: Coordination

**Created:** 2026-03-09
**Updated:** 2026-03-22
**Status:** Active
**Source:** Synthesized from 4 investigation(s) + 5 controlled experiments (139 trials) + external validation (6 independent sources) + 6 probe extensions (control theory, mechanism dimension, Align falsification, automated attractor discovery, attractor decay resilience, anticipatory placement)

## What This Is

A framework for multi-agent coordination in software engineering tasks. Describes when and why parallel agents produce individually correct work that cannot be merged, which coordination mechanisms prevent versus fail to prevent these failures, and how to implement coordination that works.

This document is organized by epistemic tier — how well each claim is supported:

1. **Battle-Tested Findings** — Directly observed and replicated in controlled experiments. These are engineering facts within the tested scenarios.
2. **Working Hypotheses** — Supported by experiments and literature review, but not independently reproduced. Useful frameworks that may need revision.
3. **Open Frontiers** — Extrapolations beyond tested scenarios, framework-level verdicts, and untested claims. Treat as directional, not authoritative.

The core insight, observed across 100 trials in same-file parallel-edit scenarios: **communication did not produce coordination.** Agents understood each other's work, discussed plans, and acknowledged potential conflicts — yet still produced unmergeable code. Only structural constraints (explicit placement instructions) prevented conflicts. This challenges the premise of frameworks like CrewAI, AutoGen, and LangGraph that assume agent-to-agent messaging solves coordination.

---

# Part I: Battle-Tested Findings

*Directly observed in controlled experiments. These claims are supported by quantitative evidence from 139+ trials across 5 experiments. Treat as reliable engineering guidance within the tested domain (same-file parallel edits, software engineering tasks, Claude models).*

## Core Claims

### Claim 1: Communication is insufficient for coordination in same-file parallel edits

In the tested same-file parallel-edit scenarios, giving agents awareness of each other's work — whether through context sharing or active messaging — did not reduce merge conflict rates. Agents acknowledged potential conflicts in their plans but did not change their behavior to avoid them.

**Test:** 4-condition experiment x N=10: no-coord, placement, context-share, messaging
**Evidence:** context-share 20/20 CONFLICT, messaging 20/20 CONFLICT (despite 18/20 trials with both plans written)
**Status:** Confirmed (p=1.0, N=80 trials, one experiment family, same-file same-repo tasks)

**Evidence quality:** Observed (single experiment family, N=80, same-file same-repo tasks; not yet tested across repos, languages, or task types).

### Claim 2: Structural placement prevents conflicts in tested scenarios

In the tested same-file scenarios, explicit non-overlapping insertion point instructions (e.g., "place after function X" vs "place after function Y") prevented merge conflicts across all trials, for both simple and complex tasks.

**Test:** Placement condition with specified insertion points
**Evidence:** placement 20/20 SUCCESS (clean merge + tests pass); attractor decay 9/9 SUCCESS with stale anchors (3 mutation types x 3 trials)
**Status:** Confirmed (N=29 across main experiment + decay experiment, same task family)

**Evidence quality:** Observed (N=29 across two experiment runs in the same task family; effectiveness at scale, with many agents, or across diverse codebases is untested).

**Attractor resilience (2026-03-22):** Placement attractors tolerate codebase mutations. Tested three mutation types with ORIGINAL (stale) placement prompts: function renames (agent adapted semantically), file reorganization (agent used secondary anchors), and addition of competing insertion points (agent followed literal attractor). 9/9 SUCCESS. The coordination value comes from **region separation** (agents assigned to different file regions), not from specific anchor function names. Agents compensate for stale anchors through semantic adaptation and anchor redundancy. See `probes/2026-03-22-probe-attractor-decay-degradation-curve.md`.

### Claim 3: Individual agent capability is not the bottleneck

Across 80 trials (160 agents total), all agents achieved 6/6 individual scores regardless of condition or task complexity. The failure was entirely in the coordination mechanism, not in individual agent performance.

**Test:** Score agents individually before attempting merge
**Evidence:** 160/160 agents scored 6/6 (100% individual success rate)
**Status:** Confirmed (N=160, single experiment family)

**Evidence quality:** Observed (160 agents across 80 trials in one experiment family; capability-coordination separation not tested with harder tasks where individual scores might vary).

### Claim 4: Coordination failure pattern is consistent across two tested task complexities

Both simple tasks (FormatBytes + FormatRate, ~40s each) and complex tasks (VisualWidth + FormatTable, 50-170s each) showed identical coordination patterns in the tested scenarios: conflicts without placement, success with placement.

**Test:** Run same 4 conditions on simple and complex task pairs
**Evidence:** Identical results across both task types in all conditions
**Status:** Observed in two task families (not sufficient to claim complexity-independence generally)

**Evidence quality:** Working-hypothesis (two task families in one repo; the pattern may not hold for qualitatively different task types such as cross-file refactors, schema changes, or multi-language tasks).

### Claim 5: Post-hoc self-checking (one tested gate design) does not prevent conflicts

In the tested gate design, giving agents a mandatory post-implementation verification step — "check your insertion point against the other agent's likely insertion point, revise if conflicting" — produced a 100% conflict rate, identical to uncoordinated agents. Agents performed the check, reported "no conflict expected," and kept their insertion point unchanged.

**Test:** Gate condition with context sharing + mandatory conflict check-and-revise step, N=20
**Evidence:** gate 20/20 CONFLICT (10/10 simple, 10/10 complex), all agents 6/6 individually
**Status:** Confirmed for this gate design (p=1.0, N=20 trials, 2026-03-22)

**Evidence quality:** Observed (N=20, one gate design; other gate designs such as multi-round negotiation, centralized merge-aware planners, or tool-assisted conflict detection are untested).

**Why this gate failed:** The self-check is itself subject to the same bias as the original insertion — agents evaluate "is my insertion point correct?" rather than "is my insertion point non-conflicting?" The semantically correct location beats the coordination-safe location at every decision point, including the verification step. This mirrors the orch-go production finding that blocking gates are bypassed 100% of the time (`.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md`).

## Directly Observed Patterns

**The compliance/coordination distinction was directly observed.** Across 80 trials, agents complied with individual task requirements (6/6 in all trials). They even complied with coordination instructions ("avoid conflicts"). But compliance was not coordination — understanding the goal did not produce the behavior.

**Evidence quality:** Observed (directly measured across 80 trials in one experiment family).

**Agents chose the "correct" location over the "non-conflicting" location.** In messaging trials, both agents chose to insert "after FormatDurationShort" because that's the semantically correct location per the task. Even when told about the other agent's identical plan, they did not deviate. The task instruction was stronger than the coordination instruction in 18/20 trials.

**Evidence quality:** Observed (18/20 messaging trials in one experiment family; generality to other task structures untested).

**Automated attractor discovery works.** A 2-phase experiment showed that structural coordination constraints can be generated automatically from failure data. Phase 1 collected collisions; Phase 2 injected auto-generated constraints and achieved 7/7 SUCCESS with zero human intervention. Only 1 collision was needed. See Key Experiment: Automated Attractor Discovery below.

**Evidence quality:** Observed (N=10, one task family; complex task types with multi-file ambiguous requirements untested).

**Anticipatory placement has a semantic congruence requirement.** LLM-generated placements from static analysis achieve 60% success (12/20): 100% for complex tasks (semantically congruent placements), 20% for simple tasks (adjacent/gravitational placements). The LLM exhibits the same gravitational bias as agents. Failure-data-driven discovery (100%) remains more reliable than static analysis (60%). See Key Experiment: Anticipatory Placement below.

**Evidence quality:** Observed (N=20, one task family; one model).

## Boundary Condition: N=1

When N=1 (single agent), all four coordination primitives are trivially satisfied:
- **Route:** Only one agent — no collision possible
- **Sequence:** Only one agent — ordering is implicit
- **Throttle:** Only one agent — velocity = verification bandwidth
- **Align:** Only one agent — model of correctness is self-consistent

This explains why autoresearch succeeds with radical simplicity — it eliminates coordination rather than solving it. The coordination framework's scope begins at N>1, where the action surface exceeds one agent's capacity.

**Implication:** The first architectural decision is not "which primitives to implement" but "can the work be structured so N=1 per work region?" If yes, coordination is free. orch-go's daemon implements this: each agent gets its own issue, avoiding coordination entirely. The 80-trial experiment shows what happens when N>1 without structural separation in same-file scenarios — 100% conflict rate.

**Evidence quality:** Observed (N=1 degenerate case directly observed; N>1 boundary validated across 100 trials).

---

# Part II: Working Hypotheses

*Supported by experiments and literature review but not independently reproduced. These are useful frameworks for reasoning about coordination, with evidence from one experiment family (N=139) plus analysis of 6 external frameworks. They may need revision as more evidence accumulates.*

## Four Coordination Primitives

The experimental findings suggest four structural primitives for multi-agent coordination. Literature review of 6 external frameworks (2026-03-22) is consistent with these primitives applying beyond orch-go, though independent reproduction in other contexts is needed to confirm generality.

- **Route** — Agents don't collide: work is assigned to non-overlapping regions. orch-go implements this via structural placement, file-level routing, and issue-level separation. Literature observation: CrewAI's core failure involves broken routing (GitHub #4783); DeepMind found centralized routing reduces error amplification from 17.2x to 4.4x.

**Evidence quality:** Working-hypothesis (one experiment family N=80 + literature mapping from 2 frameworks; not independently reproduced).

- **Sequence** — Work happens in the right order. orch-go implements this via spawn-implement-verify pipeline and daemon triage ordering. Literature observation: McEntire's pipeline architecture (broken sequence) achieved 0% success. MAST FM-1.3, FM-1.5, FM-2.1.

**Evidence quality:** Working-hypothesis (literature mapping from McEntire + MAST failure mode analysis; no controlled experiment isolating sequence as a variable).

- **Throttle** — Velocity does not exceed verification bandwidth. orch-go implements this via accretion gates, completion review, and spawn rate limiting. Literature observation: Anthropic reported 15x token consumption in multi-agent setups; McEntire's pipeline consumed $50 budget on planning alone.

**Evidence quality:** Working-hypothesis (literature mapping + orch-go production observation; no controlled experiment isolating throttle as a variable).

- **Align** — Agents share a current, accurate model of what correct means. orch-go implements this via skills, CLAUDE.md, governance hooks, and shared knowledge base. Literature observation: 50% of MAST failures (7/14 modes) involved alignment. Align appears to be the most neglected primitive across reviewed frameworks.

**Evidence quality:** Working-hypothesis (MAST failure mode analysis + orch-go production observation; no controlled experiment isolating align as a variable).

**Degenerate case:** When N=1 (single agent), all four primitives are trivially satisfied. This explains why autoresearch succeeds with radical simplicity — it eliminates coordination rather than solving it.

**Quantitative relationship (from literature):** Success degrades monotonically with missing primitives in McEntire's data:
- McEntire: 100% (single/0 missing) -> 64% (hierarchical/~1.5 missing) -> 32% (swarm/~3 missing) -> 0% (pipeline/~4 missing)
- DeepMind: 17.2x error amplification (independent/no primitives) -> 4.4x (centralized/+Route+Sequence)

### Align as Validity Condition

**Key insight:** Align appears to be the highest-leverage primitive and a validity condition for the other three. Route/Sequence/Throttle can mechanically operate without Align (messages get delivered, steps get ordered, velocity gets limited), but their coordination value degrades proportionally to Align quality. McEntire's data is consistent: partial Align correlates with 64% success, zero Align with 0%. This is a multiplier relationship, not a substrate dependency — 5 cases show Route/Sequence/Throttle mechanically holding while Align is broken (see probes/2026-03-22-probe-falsify-align-as-substrate.md). External evidence is consistent with: agents communicating while maintaining divergent models of correctness is a dominant failure pattern.

**Evidence quality:** Working-hypothesis (MAST failure mode analysis + falsification probe; the multiplier model is a qualitative description, not a quantitatively validated equation).

### Control Theory Mapping (Structural Homology)

The four primitives map onto control theory components. The mapping is structural homology (qualitative insights transfer), not isomorphism (quantitative formal tools do not transfer).

- **Route -> Actuator** — Directs work. Good primary match. FM-2.5 has sensor bleed but primary is actuator.

**Evidence quality:** Working-hypothesis (structural analogy from MAST failure mode mapping; useful interpretive lens, not formal validation).

- **Sequence -> Reference signal** — Target trajectory. Weak mapping (1/3 clean). Sequence failures in MAST data involve sensing position on trajectory as much as the reference itself.

**Evidence quality:** Working-hypothesis (structural analogy; weak mapping quality limits interpretive value).

- **Throttle -> Controller** — Regulates rate. Weak mapping (single example, messy). Premature termination involves both controller and sensor aspects.

**Evidence quality:** Working-hypothesis (structural analogy; single messy example limits confidence).

- **Align -> Sensor** — Observes correctness. Strong mapping (6/7 clean). One exception: FM-2.6 (reasoning-action mismatch) maps to actuator, not sensor.

**Evidence quality:** Working-hypothesis (structural analogy with strongest mapping quality of the four; 6/7 clean matches in MAST data).

**Sensor bleed pattern:** 4 of 5 non-clean mappings share a structure: sensor components appear in failures mapped to non-sensor primitives. Control theory explains this: when the sensor fails, other components appear to fail because they depend on feedback. This gives theoretical grounding to Align's importance — sensors are load-bearing for other components.

**Sensor involvement across MAST modes:** 11/14 (79%) involve sensors. This converges with the open-loop thread's independent finding that 14/16 (87.5%) of orch-go system failures involve missing sensors. Two independent scopes, same structural finding: most observed failures are observation failures.

**What transfers from control theory:**
- Sensor dominance: removing the sensor (opening the loop) is the most catastrophic failure
- Loop closure: every effective system needs closed feedback
- Sensor cascade: many "actuator" or "controller" failures are caused by bad sensor data

**What does NOT transfer:** Stability analysis (Bode/Nyquist), optimal control (LQR/MPC), formal controllability/observability — these require quantitative transfer functions that agent coordination doesn't have.

**Evidence:** Probe 2026-03-22, mapping all 14 MAST failure modes to control components. See `.kb/models/coordination/probes/2026-03-22-probe-control-theory-component-mapping.md`.

## Mechanism Dimension: Gates vs Attractors

The four primitives describe WHAT coordination requires. The mechanism dimension describes HOW to implement it. Two mechanism types exist, and they are not interchangeable:

**Gate (runtime checking):** Coordination that depends on a runtime decision or check. An LLM, conditional branch, or validation step decides what happens next during execution. Can fail because the decider makes the wrong choice, or can be bypassed because agents route around the check.

**Attractor (structural destination):** Coordination embedded in the system's shape at design time. The structure itself routes work — no runtime decision required. Cannot be "bypassed" because there is no alternative path; the structure IS the coordination.

**In the tested scenarios, effective coordination came from structural constraints, not communication.** The harness (orchestrator) assigned non-overlapping work regions, framing coordination as a scheduling/allocation problem rather than a communication problem. **Production validation (2026-03-18):** orch-go's daemon, swarm, and exploration mode implement structural separation — routing agents to different issues/files rather than coordinating within shared files. The production architecture is consistent with this claim.

**Evidence quality:** Working-hypothesis (supported by experiments N=100 + production architecture design; not tested across diverse coordination scenarios or alternative approaches).

**Why gates failed in our experiments:** They required correct runtime decisions by LLMs. Each decision point was a failure opportunity. CrewAI's manager must correctly route tasks. LangGraph's edges must correctly evaluate state. In our 20-trial gate condition (Claim 5), agents performed self-checks but did not change behavior — the semantically correct answer beat the coordination-correct answer at every decision point.

**Why attractors worked in our experiments:** Coordination decisions were made at design time. Anthropic's lead agent defines work regions before subagents start. orch-go's `.kb/models/*/probes/` directory reduces orphans not by checking at commit time (gate — bypassed 100%) but by making the model directory the natural destination for probe output (attractor — orphan rate halved). Attractor placement can be *discovered automatically* from failure data (see Key Experiment: Automated Attractor Discovery) — the system parses collision diffs, identifies gravitational insertion points, and generates non-overlapping constraints with zero human intervention. Gate logic, by contrast, requires human judgment about what runtime checks to perform.

**Anticipatory placement limitation (2026-03-22):** LLM-generated placements from static analysis alone (no failure data) achieve 60% success (12/20). The LLM exhibits the same gravitational bias as agents — it picks adjacent functions as "different" without understanding git merge proximity. Success depends on **semantic congruence**: when the assigned location is the natural home for the new code (e.g., VisualWidth after StripANSI), agents follow it (100% complex). When arbitrary (e.g., FormatRate after FormatDuration), agents revert to the gravitational point (20% simple). Failure-data-driven discovery (100%) remains more reliable than static analysis (60%). See `probes/2026-03-22-probe-anticipatory-placement-static-analysis.md`.

**Attractor resilience (2026-03-22):** Attractors tolerate stale anchors. 9/9 trials succeeded with original placement prompts after codebase mutations (renames, file reorganization, competing insertion points). Agents adapt through two mechanisms: (1) **semantic adaptation** — when an anchor function is renamed, agents find the semantically equivalent function; (2) **anchor redundancy** — placement instructions with multiple anchors ("after X, BEFORE Y") survive losing one anchor. The load-bearing property is **region separation** (agents assigned to different file regions), not anchor name accuracy. See `probes/2026-03-22-probe-attractor-decay-degradation-curve.md`.

### Per-Primitive Mechanism Recommendations

**Practical design principle:** Based on our experiments and literature review, attractors appear better suited for the heavy-load primitives (Route, Align) where agents must get coordination right consistently. Gates appear acceptable for lighter primitives (Throttle, Sequence) that tolerate occasional failure.

- **Route** — Recommended mechanism: attractor (structural assignment). In reviewed literature, runtime LLM routing decisions failed (CrewAI #4783).

**Evidence quality:** Working-hypothesis (one experiment family + literature mapping; recommendation based on pattern, not controlled comparison of mechanisms per primitive).

- **Align** — Recommended mechanism: attractor (output formats, shared knowledge). In McEntire's data, correctness checks in pipeline architecture produced 87% false rejections.

**Evidence quality:** Working-hypothesis (literature mapping + production experience; no controlled experiment comparing gate vs attractor for alignment specifically).

- **Throttle** — Gate acceptable (rate limits, scaling rules). Binary check with low ambiguity.

**Evidence quality:** Working-hypothesis (design reasoning from gate/attractor framework; no controlled experiment testing gate-based throttle failure rates).

- **Sequence** — Gate acceptable (dependency ordering). Ordering is mechanically enforceable.

**Evidence quality:** Working-hypothesis (design reasoning from gate/attractor framework; no controlled experiment testing gate-based sequence failure rates).

**Evidence:** Probe 2026-03-22, gate/attractor validation across 6 frameworks. See `.kb/models/knowledge-accretion/probes/2026-03-22-probe-validate-gate-attractor-external-frameworks.md`. Also: orch-go production data from `.kb/models/knowledge-accretion/probes/2026-03-20-probe-intervention-effectiveness-audit.md`.

## Implications (Extrapolated)

1. **In the tested scenarios and across 6 reviewed frameworks, messaging-based coordination did not produce coordination outcomes.** CrewAI, LangGraph, Claude Agent SDK, OpenAI Agents SDK, and similar systems that assume agents can coordinate through communication showed coordination failures in the reviewed literature. In our experiments (N=80), communication did not fail — the agents communicated effectively. The coordination still failed. Whether richer messaging protocols or stronger planning could overcome this is an open question. (Note: AutoGen, originally cited here, entered maintenance mode by Mar 2026, succeeded by Microsoft Agent Framework.)

**Evidence quality:** Working-hypothesis (extrapolated from same-file experiment N=80 + literature review of 6 frameworks; not directly reproduced across diverse coordination scenarios).

---

# Part III: Open Frontiers

*Extrapolations beyond tested scenarios, framework-level assessments based on literature review (not reproduction), and untested claims. Treat as directional signals, not established findings.*

## Framework Assessments (Literature Review Only)

**Evidence from 6 reviewed external frameworks:** In the reviewed literature, gate-based coordination correlated with failure and attractor-based coordination correlated with success across all 6 frameworks examined. These assessments are based on documentation and published reports, not independent reproduction.

- **CrewAI** — Does not work. Primary mechanism: gate (manager LLM routing).

**Evidence quality:** Working-hypothesis (literature review of framework documentation and GitHub issues; not reproduced in this repo).

- **LangGraph** — Does not work. Primary mechanism: gate (conditional graph edges).

**Evidence quality:** Working-hypothesis (literature review; not reproduced in this repo).

- **OpenAI Agents SDK** — Does not work. Primary mechanism: gate (output-mediated handoffs).

**Evidence quality:** Working-hypothesis (literature review; not reproduced in this repo).

- **Claude Agent SDK** — Does not work. Primary mechanism: absent (human manual coordination).

**Evidence quality:** Working-hypothesis (literature review; not reproduced in this repo).

- **Anthropic production** — Works. Primary mechanism: attractor-dominant (task regions + output formats).

**Evidence quality:** Working-hypothesis (literature review of Anthropic's published multi-agent patterns; not independently verified).

- **autoresearch** — Works. Primary mechanism: pure attractor (N=1 structural constraint).

**Evidence quality:** Working-hypothesis (literature review; N=1 eliminates coordination rather than solving it, so "attractor" is a degenerate case).

**McEntire degradation tracks the gate/attractor gradient (from literature):**
- Single agent (pure attractor): 100%
- Hierarchical (attractor + gates): 64%
- Swarm (pure gates): 32%
- Pipeline (maximum gates): 0%

## Open Questions

### Answered

- ~~Does a stronger coordination instruction ("you MUST choose a different insertion point than the other agent") change behavior?~~ **Answered 2026-03-22:** No. Gate condition with mandatory check-and-revise step: 20/20 CONFLICT. Even explicit self-verification doesn't overcome semantic-correctness bias.
- ~~Is Align a substrate (other primitives can't function without it)?~~ **Answered 2026-03-22:** No — "substrate" overclaims. 5 cases show Route/Sequence/Throttle mechanically holding while Align is broken. Align is a multiplier/validity condition with proportional (not binary) impact. See `probes/2026-03-22-probe-falsify-align-as-substrate.md`.
- ~~Does the control theory mapping (Route->Actuator, Sequence->Reference, Throttle->Controller, Align->Sensor) hold?~~ **Answered 2026-03-22:** Structural homology, not isomorphism. 64% clean mapping with systematic sensor bleed pattern. Qualitative insights transfer; formal tools do not. See `probes/2026-03-22-probe-control-theory-component-mapping.md`.
- ~~Does the gate/attractor mechanism distinction generalize beyond orch-go?~~ **Answered 2026-03-22:** Literature review of 6 frameworks is consistent with the distinction. Gate-based coordination correlated with failure, attractor-based with success. McEntire degradation tracks gate/attractor gradient monotonically. See `.kb/models/knowledge-accretion/probes/2026-03-22-probe-validate-gate-attractor-external-frameworks.md`.
- ~~Can structural attractors be discovered automatically from collision patterns?~~ **Answered 2026-03-22:** Yes. 2-phase experiment: system parsed conflict diffs, identified gravitational insertion point, generated non-overlapping constraints, achieved 7/7 success with zero human intervention. 1 collision is sufficient for effective constraint generation. See `probes/2026-03-22-probe-automated-attractor-discovery.md`.

### Open

- Does placement work when the number of agents exceeds the number of natural insertion points?
- Can iterative messaging (multi-round negotiation) produce different results than single-shot plan exchange?
- At what task granularity does structural placement become impractical?
- Should Align decompose into sub-primitives? **Evidence supports yes (2026-03-22):** 80-trial messaging condition shows task alignment defeating coordination alignment in 18/20 trials. At minimum three sub-components: task alignment (agent understands its own task), state alignment (agent knows system state), coordination alignment (agent adjusts behavior for multi-agent context). Needs formal decomposition and separate testing.
- Do the four primitives have ordering dependencies? (Must Route precede Sequence?)
- How do primitives interact with task type? (DeepMind found coordination strategy is task-dependent — financial reasoning favors centralized, web navigation favors decentralized)
- Do the primitives apply to non-LLM multi-agent systems? (robotics, distributed computing, human organizations)
- Does the multiplier model (Coordination_value = Route x Sequence x Throttle x Align) hold quantitatively, or is the interaction more complex? McEntire's 64% hierarchical result is consistent, but no controlled experiment isolates each factor.
- ~~Can attractor-based coordination degrade? (What happens when structural destinations become stale or misaligned with evolving requirements?)~~ **Answered 2026-03-22:** Not for incremental codebase changes. 9/9 SUCCESS with stale attractors across renames, file reorganization, and competing insertion points. Agents adapt through semantic resolution and anchor redundancy. Region separation (not anchor accuracy) is the load-bearing property. Untested: wholesale restructuring. See `probes/2026-03-22-probe-attractor-decay-degradation-curve.md`.
- Does automated attractor discovery work for complex tasks? (Multi-file, ambiguous requirements may produce collision patterns harder to parse or requiring more nuanced constraint generation.)
- What is the minimum number of natural insertion points needed per agent? (Automated discovery relies on function boundaries as candidate points. With 6 functions and 2 agents, alternatives were plentiful. What about 6+ agents?)
- Would a stronger placement model (Opus vs Haiku) produce better anticipatory placements? The simple-task failures are arguably a reasoning failure — the LLM picks adjacent functions as "different" without understanding git merge proximity. A model with better spatial reasoning might avoid this.
- Can anticipatory placement be improved by injecting git-merge-specific knowledge into the placement prompt? (e.g., "adjacent functions produce conflicts even if they are different functions")

---

# Appendix: Experiments and Evidence

## Evidence Summary

| Date | Source | Finding |
|------|--------|---------|
| 2026-03-09 | Pilot (N=1) | Both Haiku and Opus produce 6/6 individually, 100% merge conflict |
| 2026-03-09 | N=10 FormatBytes | 100% conflict rate at N=10, Fisher's exact p=1.0 |
| 2026-03-09 | Complex task (N=1) | 4-file conflicts including add/add type, semantic conflicts from design divergence |
| 2026-03-10 | 4-condition experiment (N=80) | Placement 20/20 success, all other conditions 60/60 conflict, 160/160 individual 6/6 |
| 2026-03-18 | Decay verification probe | All 4 claims confirmed current. Experiment data intact. Framework references updated (AutoGen -> deprecated). Production architecture validates structural approach. |
| 2026-03-22 | External framework validation probe | All 4 claims confirmed as general (not orch-go-specific). 14 MAST failure modes map to 4 primitives. McEntire experiment shows monotonic degradation. DeepMind scaling paper confirms centralized coordination reduces error amplification. Align identified as dominant/neglected primitive (50% of failures). |
| 2026-03-22 | Control theory component mapping probe | Primitives map to control components (Route->Actuator, Sequence->Reference, Throttle->Controller, Align->Sensor) with 64% clean mapping. Sensor bleed pattern: 11/14 MAST modes involve sensors (79%), converging with open-loop thread's 87.5%. Structural homology, not isomorphism. |
| 2026-03-22 | Align-as-substrate falsification probe | "Substrate" overclaims — 5 cases show Route/Sequence/Throttle mechanically holding while Align is broken (MAST FM-1.1, McEntire hierarchical 64%, launchd post-mortem, orch-go competing instructions, stale knowledge cascades). Align is a multiplier/validity condition with proportional (not binary) impact. "Meta-primitive" language replaced. |
| 2026-03-22 | Gate condition experiment (N=20) | Post-hoc self-checking gate produces 100% conflict rate (20/20). Agents perform the check, report no conflict, keep identical insertion points. Gates are subject to the same semantic-correctness bias as the original decision. All 40 agents scored 6/6 individually. |
| 2026-03-22 | Gate/attractor external validation probe | 6/6 external frameworks show perfect correlation: gate-based coordination fails (CrewAI, LangGraph, OpenAI Agents SDK), attractor-based works (Anthropic production, autoresearch). McEntire degradation tracks gate/attractor gradient: pure attractor 100% -> attractor+gates 64% -> pure gates 32% -> max gates 0%. |
| 2026-03-22 | Automated attractor discovery experiment (N=10) | System automatically discovered effective placement constraints from 2 observed collisions. Phase 1: 2/3 CONFLICT (gravitational point: FormatDurationShort). Phase 2: 7/7 SUCCESS with auto-generated constraints. Zero human intervention in constraint generation. Closed loop validated: observe failures -> extract constraints -> inject constraints -> prevent failures. |
| 2026-03-22 | Attractor decay experiment (N=9) | Stale attractors do NOT degrade coordination. 9/9 SUCCESS across 3 mutation types (rename, reorganize, add alternatives) with original stale placement prompts. Agents adapt through semantic resolution and anchor redundancy. Region separation is the load-bearing property, not anchor accuracy. Hypothesis of cliff-edge failure disproved. |
| 2026-03-22 | Anticipatory placement experiment (N=20) | LLM-generated placements achieve 60% overall (12/20): 100% for complex tasks (semantically congruent placements), 20% for simple tasks (adjacent/gravitational placements). Static analysis without failure data is insufficient — the placement LLM exhibits the same gravitational bias as agents. Semantic congruence between task and placement determines agent compliance. |

## Key Experiment: 4-Condition Redesign (2026-03-10)

**Design:** 4 conditions x 2 task types (simple + complex) x N=10 = 80 trials, 160 agent invocations
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

## Key Experiment: Gate Condition Extension (2026-03-22)

**Design:** 1 condition (gate) x 2 task types (simple + complex) x N=10 = 20 trials, 40 agent invocations
**Model:** claude-haiku-4-5 (both agents)
**Tasks:** Same as 4-condition experiment

| Condition | Mechanism | Simple | Complex |
|-----------|-----------|:------:|:-------:|
| gate | Context sharing + mandatory post-implementation conflict check-and-revise | 10/10 CONFLICT | 10/10 CONFLICT |

**Gate prompt:** After implementing, agents were required to: (1) review their insertion points, (2) predict the other agent's insertion points, (3) check for overlap, (4) revise if conflicting. Despite this explicit check-and-revise protocol, all 20 trials produced conflicts.

**Duration comparison:** Gate condition agents averaged 105s (simple) and 144-251s (complex), comparable to context-share condition, suggesting agents didn't spend significant extra time on the verification step.

**Results:** `experiments/coordination-demo/redesign/results/20260322-124035/`

## Key Experiment: Automated Attractor Discovery (2026-03-22)

**Design:** 2-phase experiment. Phase 1: 3 trials (no attractors, collect collisions). Phase 2: 7 trials (auto-generated attractors). 10 trials total, 20 agent invocations.
**Model:** claude-haiku-4-5 (both agents)
**Tasks:** Agent A (FormatBytes) + Agent B (FormatRate) — same as simple condition in 4-condition experiment

| Phase | Mechanism | Result |
|-------|-----------|--------|
| Phase 1 (no attractors) | No placement instructions | 2/3 CONFLICT (1 trial invalid — worktree race) |
| Phase 2 (auto-generated) | Constraints generated from Phase 1 collision patterns | **7/7 SUCCESS** |

**How it works:** The system parses Phase 1 conflict diffs to extract hunk start lines (`@@ -92 @@`), maps them to function boundaries via `grep -n '^func '`, identifies `FormatDurationShort` as the "gravitational function" (where both agents independently insert), scans for alternative insertion points, and assigns Agent B to `StripANSI` (maximally separated). Constraint text is injected into prompts with zero human editing.

**Key finding:** Structural coordination is not just effective but *automatable*. Only 1 collision was needed — a single conflict diff provides the hunk header and function boundary needed to generate effective constraints. The closed loop works: observe failures -> extract collision points -> generate non-overlapping constraints -> inject into prompts -> prevent failures.

**Implication for attractors:** This strengthens the attractor mechanism. Not only do attractors work better than gates (in our experiments), but attractor placement can be *discovered automatically* from failure data. Gate logic requires human judgment about what to check; attractor placement can be derived algorithmically from the structure of observed failures.

**Results:** `experiments/coordination-demo/redesign/results/attractor-20260322-154220/`

## Key Experiment: Anticipatory Placement (2026-03-22)

**Design:** 1 condition (anticipatory) x 2 task types x N=10 = 20 trials, 40 agent invocations + 20 placement LLM calls
**Model:** claude-haiku-4-5 (placement model, both agents)
**Tasks:** Same as 4-condition experiment
**Question:** Can an LLM predict non-overlapping insertion points from static code analysis alone, without any failure data?

| Condition | Mechanism | Simple | Complex |
|-----------|-----------|:------:|:-------:|
| anticipatory | LLM-generated placement from code + task analysis | **2/10 SUCCESS** | **10/10 SUCCESS** |

**Corrected from raw results:** Raw experiment reported 0/20 SUCCESS because 12 trials were falsely classified as BUILD_FAIL due to stale `.go` files in experiment results directories. Manual replay of 3 trials confirmed all BUILD_FAIL merges were actually clean + passing.

**Why complex succeeds (100%):** The LLM picked StripANSI for VisualWidth's placement 9/10 times because VisualWidth *uses* StripANSI. Semantic congruence between task and placement -> agents follow the instruction. FormatTable got FormatDurationShort (semantically neutral but distant from StripANSI). Both placements produce non-overlapping diffs.

**Why simple fails (80%):** FormatBytes and FormatRate have no semantic relationship to any existing function. The LLM picks FormatDuration/FormatDurationShort as "different" functions 8/10 times — they ARE different functions but are adjacent (13 lines apart), and agents override the "after FormatDuration" instruction because the base prompt says "after FormatDurationShort." The gravitational pull to end-of-file dominates.

**Key finding: Semantic congruence requirement.** LLM-generated placements work when the assigned location is the natural semantic home for the new code. When placement is arbitrary relative to the task's dependencies, agents revert to the gravitational insertion point. This means static-analysis-based attractor generation requires understanding task-code semantic relationships, not just spatial separation.

**Comparison across attractor generation methods:**

| Method | Success Rate | Requires Failure Data? | Key Advantage |
|--------|-------------|----------------------|---------------|
| Human-designed | 20/20 (100%) | No | Understands both space and semantics |
| Failure-data-driven (Exp B) | 7/7 (100%) | Yes (1 collision) | Identifies actual collision point |
| Anticipatory/static (Exp C) | 12/20 (60%) | No | Zero-shot, no prior failures needed |

**Results:** `experiments/coordination-demo/redesign/results/20260322-162206/`

---

## Boundaries

**What this model covers:**
- Multi-agent coordination failure and prevention in software engineering tasks
- Four coordination primitives (Route, Sequence, Throttle, Align) proposed from 100 trials + literature review of 6 external sources
- Mechanism dimension: gate (runtime) vs attractor (structural) implementation
- Control theory structural homology (qualitative insights, not formal tools)
- Scope boundary: N>1 (single agent trivially satisfies all primitives)

**What this model does NOT cover:**
- Cross-file coordination (agents editing entirely different files)
- Sequential execution patterns (one agent after another)
- Human-in-the-loop coordination
- Non-git merge strategies (e.g., semantic merge tools)
- Tasks where agents naturally edit different regions
- Quantitative control theory (stability analysis, optimal control, formal observability)

---

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
