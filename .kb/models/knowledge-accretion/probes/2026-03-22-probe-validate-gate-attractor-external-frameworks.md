# Probe: Validate Gate/Attractor Mechanism Distinction Against External Frameworks

**Model:** knowledge-accretion
**Date:** 2026-03-22
**Status:** Complete
**claim:** KA-05 (intervention effectiveness hierarchy: structural attractors > signaling > blocking gates > advisory gates > metrics-only)
**verdict:** extends

---

## Question

The intervention effectiveness audit (2026-03-20) established that structural attractors outperform gates in orch-go: orphan rate 94.7%→52.0% from attractor-based model/probe directories, while every blocking gate was bypassed 100%. Is this pattern orch-go-specific, or does it hold in external multi-agent frameworks?

Hypothesis: systems that implement coordination via attractors (structural destinations) work; systems that implement coordination via gates (runtime checking/blocking) fail. The McEntire hierarchy→swarm→pipeline degradation should track with decreasing attractor usage and increasing gate usage.

---

## What I Tested

Re-analyzed the external framework evidence from `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md` through the gate/attractor lens, using precise definitions:

**Gate (runtime checking):** Coordination that depends on a runtime decision or check. An LLM, conditional branch, or validation step decides what happens next during execution. Can fail because the decider makes the wrong choice, or can be bypassed because agents route around the check.

**Attractor (structural destination):** Coordination embedded in the system's shape at design time. The structure itself routes work — no runtime decision required. Cannot be "bypassed" because there is no alternative path; the structure IS the coordination.

These definitions generalize the orch-go-specific terms: orch-go gates = pre-commit checks, spawn validation; orch-go attractors = `.kb/models/*/probes/` directories, model template priming. The external analysis uses the same distinction at a higher level of abstraction.

For each framework, examined:
1. How its coordination mechanisms work (from documented behavior)
2. Whether each mechanism is gate-type or attractor-type
3. Whether it succeeds or fails
4. The correlation between mechanism type and outcome

---

## What I Observed

### Framework Classification

#### 1. CrewAI (Fails — broken Route)

**Mechanism type: GATE**

CrewAI's hierarchical process delegates tasks via a manager LLM that decides at runtime which worker gets which task. This is a gate: an LLM checks the task description and makes a routing decision. Evidence of failure: "manager agents cannot delegate to worker agents" (GitHub #4783), "manager delegates to wrong agent" (#3179), manager executes tasks sequentially instead of routing.

The proposed fix (Sarkar, TDS Nov 2025) is telling: add explicit step-wise routing instructions — which converts the broken runtime gate into a weak attractor by embedding the route in the task definition itself (structural, not runtime).

**No attractors present.** No structural mechanism naturally pulls tasks to the correct agent.

#### 2. LangGraph (Fails — broken Throttle)

**Mechanism type: GATE**

LangGraph coordinates via a directed graph with conditional edges — at each node, a condition is evaluated and the next node is selected. These are gates: runtime checks at branching points. Evidence of failure: "deeply nested conditional branches or highly interconnected nodes can experience significant slowdowns." Teams "dedicate excessive time to managing orchestration rather than delivering core business value."

**Partial attractor (overridden by gates).** The graph topology is structural and could function as an attractor — defined paths that agents follow. But the dynamic conditional branching makes coordination fundamentally gate-based. The graph shape is flexible (configurable at runtime), not fixed (embedded in structure). LangGraph's graph is more like a configurable gate network than a structural attractor.

#### 3. OpenAI Agents SDK (Fails — broken Sequence)

**Mechanism type: GATE**

Agent handoffs are mediated through task outputs — each handoff is a gate where the receiving agent checks what the previous one produced. "Lacks built-in checkpointing for long-running workflows." "Coarse-grained error handling." No structural mechanism sequences the work; ordering depends on runtime output passing.

**No attractors present.** No structural destination pulls work into correct order.

#### 4. Claude Agent SDK (Fails — broken Route + Align)

**Mechanism type: ABSENT (neither gate nor attractor)**

Cross-machine agents are "completely blind to each other." The SDK provides no coordination mechanism at all — no gates checking correctness, no attractors pulling toward correct patterns. "Every interface change, every schema decision, every API contract must be manually relayed by the developer." The human developer serves as a manual gate, checking and relaying information between agents.

**Neither gate nor attractor.** The coordination gap is total. When humans fill it, they use gate-like behavior (checking, relaying).

#### 5. Anthropic Production System (Works)

**Mechanism type: ATTRACTOR-DOMINANT (with gate support)**

The lead-agent + subagents pattern implements coordination primarily through attractors:

| Primitive | Implementation | Type | Why |
|-----------|---------------|------|-----|
| **Route** | Lead agent defines non-overlapping work regions with detailed task descriptions | **Attractor** | The task definition IS the structural destination — subagents are pulled toward their assigned scope by the shape of their instructions |
| **Align** | Explicit output formats specified per subagent | **Attractor** | Expected output shape pulls agents toward correct results at design time |
| **Throttle** | Scaling rules (limits on subagent count, after initial 50-agent explosion) | **Gate** | Runtime limit that checks count before spawning |
| **Sequence** | Parallel execution with dependency ordering | **Gate** | Ordering enforced via dependency checks |

**Key observation:** Route and Align (the dominant primitives — responsible for 9/14 MAST failure modes) are implemented as attractors. Throttle and Sequence (the supporting primitives) are implemented as gates. The system works because the heavy-load coordination is structural, while only the lighter coordination tasks use runtime checking.

#### 6. autoresearch (Works via N=1)

**Mechanism type: PURE ATTRACTOR**

The single file, single metric, single agent constraint is the ultimate structural attractor. There is no coordination to fail because the structure makes collision impossible. No gates exist because there's nothing to gate. The system's shape IS the coordination.

This is the degenerate case that proves the principle: when the structure fully determines the work, no runtime checking is needed, and success is 100%.

### McEntire Degradation: Gate/Attractor Gradient

| Architecture | Success | Coordination Type | Analysis |
|---|---|---|---|
| Single agent | **100%** | **Pure attractor** | N=1 = structure fully determines scope. No runtime decisions needed. |
| Hierarchical | **64%** | **Gate-dominant with structural support** | Manager makes runtime routing decisions (gate), but the hierarchy itself is structural (manager above workers = weak attractor). 64% success = the weak structural component partially compensates for gate failures. |
| Swarm | **32%** | **Pure gate** | Peer-to-peer negotiation, voting, checking. No structural hierarchy to provide even weak attractor effects. All coordination is runtime. |
| Pipeline | **0%** | **Pure gate (maximum checking)** | Sequential validation at every handoff. Most gates of any architecture. Agents rejected 87% of submissions with zero factual basis — gates firing incorrectly. Budget consumed on planning checks. |

**The gradient is clear:** Success drops as the gate/attractor ratio increases. Single agent (pure attractor) → hierarchical (attractor with gates) → swarm (pure gates) → pipeline (maximum gates). The pipeline's 0% is particularly damning: it has the MOST coordination gates of any architecture, and the WORST outcome. More checking = worse results.

### Correlation Summary

| Framework | Works? | Primary Mechanism | Gate/Attractor |
|-----------|--------|-------------------|----------------|
| CrewAI | No | Manager LLM routing decisions | **Gate** |
| LangGraph | No | Conditional graph edges | **Gate** |
| OpenAI Agents SDK | No | Output-mediated handoffs | **Gate** |
| Claude Agent SDK | No | None (human manual) | **Absent** |
| Anthropic production | **Yes** | Detailed task regions + output formats | **Attractor-dominant** |
| autoresearch | **Yes** | N=1 structural constraint | **Pure attractor** |

**Perfect correlation:** Every failing system uses gate-based coordination (or none). Every working system uses attractor-based coordination. 6/6 frameworks match the prediction.

### Why This Correlation Holds

The probe reveals a deeper mechanism than just "gates bad, attractors good":

**Gates fail because they require correct runtime decisions by LLMs.** The manager in CrewAI must correctly decide which worker gets each task. The conditional edges in LangGraph must correctly evaluate state. The handoffs in OpenAI Agents SDK must correctly transfer context. Each decision point is a failure opportunity, and LLMs make wrong decisions at measurable rates.

**Attractors succeed because the coordination decision is made at design time.** Anthropic's lead agent defines work regions before subagents start. autoresearch constrains to N=1 before any execution begins. The structure embeds the coordination — no LLM runtime decision required.

This is the same mechanism as orch-go's finding: the `.kb/models/*/probes/` directory reduces orphans not by checking at commit time (gate) but by making the model directory the natural destination for probe output (attractor). The pre-commit accretion gate was bypassed 100%, but the directory structure cut orphan rates in half — same distinction, different scale.

### What the External Pattern Actually Is

The gate/attractor distinction captures the external pattern, but it's a special case of a more fundamental division: **structural coordination (design-time) vs runtime coordination (execution-time).**

- Structural: decisions embedded in system shape before execution starts
- Runtime: decisions made by agents during execution

Every external working system uses structural coordination. Every failing system uses runtime coordination. The gate/attractor vocabulary from orch-go maps onto this cleanly: attractors ARE structural coordination; gates ARE runtime coordination.

---

## Model Impact

- [x] **Extends** model with: External validation of the intervention effectiveness hierarchy. The hierarchy (structural attractors > signaling > blocking gates > advisory gates > metrics-only) is not orch-go-specific. External evidence from 6 independent multi-agent frameworks shows perfect correlation: attractor-based coordination works (2/2), gate-based coordination fails (3/3), absent coordination fails (1/1). McEntire's controlled experiment shows monotonic success degradation as the gate/attractor ratio increases.

- [x] **Extends** model with: The gate/attractor distinction generalizes to a structural-vs-runtime coordination principle. Gates fail because they require correct LLM runtime decisions; attractors succeed because coordination decisions are embedded in system structure at design time. This is the same mechanism in orch-go (directory structure > pre-commit checks) and in external frameworks (task region definitions > manager LLM routing).

- [x] **Extends** model with: The Anthropic production system validates a mixed strategy — attractors for the heavy-load primitives (Route, Align) and gates for the lighter primitives (Throttle, Sequence). This suggests a practical design principle: use attractors for coordination that agents must get right consistently (routing, alignment), and gates for coordination that tolerates occasional failure (rate limiting, ordering).

---

## Notes

### Strength of Evidence

The 6/6 correlation is striking but has caveats:

1. **Selection bias.** The external framework investigation selected frameworks that were documented to fail or succeed. A systematic survey might find exceptions — frameworks with gate-based coordination that work, or attractor-based coordination that fails.

2. **Confound: coordination complexity.** Systems with more coordination requirements (more agents, more shared state) tend to use more gates AND fail more. The failure might correlate with coordination complexity rather than mechanism type. However, McEntire's experiment controls for this — all 4 architectures attempt the same 28 tasks. The only variable is architecture type. And the gate/attractor gradient still holds.

3. **Definitional flexibility.** Classifying a mechanism as "gate" or "attractor" requires judgment. LangGraph's graph topology could be argued as an attractor (structural path) or a gate (conditional branching). The classification used here is based on whether the coordination decision happens at design time (attractor) or runtime (gate), which is more precise than pure gate/attractor vocabulary.

### Connection to McEntire's "Dysmemic Pressure"

McEntire observed that pipeline agents' "communication becomes statistically independent of reality" — agents pass messages back and forth that have no relationship to the actual task state. This is what maximum gate-based coordination looks like in practice: every handoff is a checking point, but the checks themselves are wrong. Gates without structural grounding degenerate into ceremony.

This maps exactly to orch-go's finding: the self-review gate had 79% false positives and 0 true positives. The accretion gate was bypassed 100%. Gates that check without structural grounding become noise. McEntire's pipeline agents are running the multi-agent equivalent of orch-go's failed pre-commit hooks — lots of checking, no coordination.

### Design Implication

For orch-go's coordination model: the four primitives (Route, Sequence, Throttle, Align) are necessary, but HOW they're implemented matters as much as WHETHER they're implemented. Gate-based implementations of the primitives will fail. Attractor-based implementations will succeed. The external evidence suggests:

- **Route:** Implement as attractor (structural assignment) not gate (LLM routing decisions)
- **Align:** Implement as attractor (output format definitions, shared knowledge) not gate (correctness checks)
- **Throttle:** Gate implementation is acceptable (rate limits, scaling rules)
- **Sequence:** Gate implementation is acceptable (dependency ordering)
