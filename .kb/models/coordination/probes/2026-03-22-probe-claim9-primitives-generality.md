# Probe: Claim 9 — Four Primitives Generality

**Model:** coordination
**Date:** 2026-03-22
**Status:** Complete
**claim:** COORD-09
**verdict:** scopes

---

## Question

The model claims four coordination primitives (Route, Sequence, Throttle, Align) are "general to any multi-agent system." The epistemic audit flagged this as overclaimed — the primitives come from one experiment family + literature review of 6 LLM frameworks. Are these primitives truly general, or domain-specific to multi-agent software engineering with merge-based integration?

**Falsification condition:** (a) Find a coordination failure from an established domain that doesn't map to any of the four primitives, OR (b) find a working multi-agent system that explicitly omits one, OR (c) find an established taxonomy that uses a fundamentally different decomposition.

---

## What I Tested

Mapped Route/Sequence/Throttle/Align against four established coordination taxonomies:

1. **Malone & Crowston (1994)** — "Coordination Theory" from organizational science (canonical taxonomy)
2. **Mintzberg (1979/1993)** — Five coordination mechanisms from "The Structuring of Organizations"
3. **MAST/Cemri et al. (2025)** — 14 failure modes from 150+ MAS tasks across 5 frameworks (already in model)
4. **Distributed systems** — CAP theorem, consensus protocols, Kubernetes scheduling

Sources: Web search for Malone & Crowston coordination theory, Mintzberg five coordination mechanisms, MAST paper (arxiv.org/html/2503.13657v1). MAST mapping already in model from prior probe.

---

## What I Observed

### Mapping 1: Malone & Crowston (1994) — Coordination = Managing Dependencies

Malone & Crowston define coordination as "managing dependencies between activities." Their dependency types:

| Malone & Crowston Dependency | Maps to Primitive? | Quality |
|---|---|---|
| **Shared resource** (multiple activities use same resource) | Route | Clean — Route prevents collision on shared resources |
| **Producer/consumer** (one activity produces what another needs) | Sequence | Clean — ordering dependency |
| **Simultaneity** (activities must happen at same time) | Sequence (inverse) | Partial — anti-ordering, not addressed directly |
| **Task-subtask** (decomposing work into parts) | Route (partially) | Partial — Route assigns parts but doesn't define HOW to decompose |
| **Usability** (outputs must be in correct form for consumers) | Align | Clean — shared model of correctness |

**Finding:** 4/5 Malone & Crowston dependency types map to the four primitives. "Simultaneity" is a weak fit (inverse of Sequence). However, Malone & Crowston also describe **coordination mechanisms** (how you manage dependencies): notification, tracking, managerial decision-making. These are mechanisms (like gate/attractor), not requirements. The model's distinction between "what" (primitives) and "how" (mechanisms) is consistent with Malone & Crowston's framework.

**Gap found:** Malone & Crowston include **task decomposition** as a coordination problem. This is upstream of Route — before you can route work, you must decide what the work units are. The four primitives assume work is already decomposed into agent-assignable units. Decomposition is a missing prerequisite.

### Mapping 2: Mintzberg (1979) — Five Coordination Mechanisms

Mintzberg describes HOW organizations coordinate, not WHAT must be coordinated:

| Mintzberg Mechanism | Relationship to Primitives |
|---|---|
| **Mutual adjustment** (informal communication) | Mechanism for Route/Align — tested and failed in experiments (messaging condition) |
| **Direct supervision** (centralized coordinator) | Mechanism for Route/Sequence — resembles the orchestrator pattern |
| **Standardization of work processes** | Mechanism for Sequence — prescribe the process steps |
| **Standardization of outputs** | Mechanism for Align — define what "correct" looks like |
| **Standardization of skills** | Mechanism for Align — pre-runtime alignment via shared training/prompt engineering |

**Finding:** Mintzberg's taxonomy is complementary, not competing — it describes mechanisms (how) while Route/Sequence/Throttle/Align describe requirements (what). This is a strength: the four primitives sit at a different abstraction level than Mintzberg and don't contradict it. However, Mintzberg's taxonomy also implies that Throttle has no corresponding mechanism (none of his five specifically address rate/velocity control). This may mean Throttle is domain-specific to computational agents (physical organizations throttle via resource scarcity, not explicit rate limits).

### Mapping 3: MAST Failure Modes (already in model)

The model already maps all 14 MAST failure modes to the four primitives + control theory. 50% involve Align (sensor), 79% involve some sensor component. This mapping is the strongest external evidence for the primitives.

**Finding:** MAST confirms that the four primitives cover the LLM multi-agent domain well. But MAST is also specific to LLM multi-agent systems — it doesn't extend to robotics, distributed computing, or human organizations.

### Mapping 4: Distributed Systems (CAP + Consensus + Scheduling)

| Distributed Systems Concern | Maps to Primitive? | Quality |
|---|---|---|
| **Consensus** (agreement on shared state) | Align | Clean |
| **Leader election** (who coordinates) | Route | Clean |
| **Partitioning/sharding** (dividing data/work) | Route | Clean |
| **Ordering** (total/partial order of operations) | Sequence | Clean |
| **Rate limiting** (backpressure, flow control) | Throttle | Clean |
| **Fault tolerance/availability** | NONE | Gap — recovery from coordination failure |
| **Consistency model choice** (eventual vs. strong) | Partially Align | Partial — a design meta-choice, not a primitive |

**Finding:** 5/7 distributed systems concerns map cleanly. But two important distributed systems concepts have no primitive equivalent:

1. **Fault tolerance / Recovery** — What happens AFTER coordination fails? The four primitives are about PREVENTING failures, not recovering from them. Distributed systems invest heavily in recovery (retries, quorum fallbacks, rollback). The model's experiments have 100% conflict rates in failed conditions — there's no recovery mechanism.

2. **Consistency model choice** — In distributed systems, you choose HOW MUCH coordination you need (strong consistency = full Align, eventual = relaxed Align). This meta-choice isn't a primitive but determines which primitives are required.

### Synthesis: What the four primitives miss

Three coordination concerns found across established taxonomies that DON'T map to Route/Sequence/Throttle/Align:

1. **Decomposition** (Malone & Crowston) — dividing work into coordinatable units. Prerequisite to Route. The model assumes work is pre-decomposed.

2. **Recovery** (Distributed systems) — what happens when coordination fails. The model focuses on prevention, not recovery. In practice, some coordination failures are inevitable and systems need graceful degradation.

3. **Meta-coordination** (Mintzberg, distributed systems) — choosing which coordination strategy to apply. When do you use Route vs Sequence? When do you relax Align? This is coordination ABOUT coordination.

---

## Model Impact

- [x] **Scopes** claim: "general to any multi-agent system" → "general to multi-agent LLM software engineering systems with merge-based integration"

**Specific changes needed in model.md:**

1. Section title "Four Coordination Primitives (Generalized Framework)" should become "Four Coordination Primitives (Multi-Agent SE Framework)" — removing "Generalized"
2. The qualifying text already says "Literature review of 6 external frameworks is consistent with these primitives applying beyond orch-go" — this should add: "Mapping against Malone & Crowston and Mintzberg shows the primitives are consistent with but narrower than established coordination theory. They address coordination requirements for parallel work integration but do not cover decomposition, recovery, or meta-coordination."
3. Evidence tier for the primitives should stay "Working-hypothesis" but qualifying details should note: "proposed from one experiment family + literature review of 6 LLM frameworks; consistent with Malone & Crowston and Mintzberg taxonomies at the requirements level; missing decomposition, recovery, and meta-coordination primitives found in broader coordination theory"
4. Open questions should add: "Should decomposition (pre-Route) and recovery (post-failure) be added as primitives, or are they out of scope?"

**What the primitives DO well:**
- Clean mapping to 4/5 Malone & Crowston dependency types
- Complementary (not contradictory) to Mintzberg's mechanisms
- Cover 14/14 MAST failure modes in the LLM domain
- Clean mapping to 5/7 distributed systems concerns

**What "general to any multi-agent system" overclaims:**
- Missing decomposition (how to divide work)
- Missing recovery (how to handle coordination failure)
- Missing meta-coordination (choosing the right strategy)
- Throttle may be domain-specific (computational agents, not physical organizations)
- Evidence is from LLM multi-agent systems only — no robotics, no human teams

---

## Notes

The four primitives are a useful and well-structured taxonomy for their domain. The problem is the scope claim, not the primitives themselves. "General to multi-agent software engineering with merge-based integration" is supportable from the evidence. "General to any multi-agent system" is not.

The most interesting gap is **recovery**: orch-go's production system handles this implicitly (if an agent fails, the issue goes back to the daemon queue), but the model doesn't formalize it. In distributed systems, recovery is often the most important coordination primitive because you can't guarantee other primitives always hold.
