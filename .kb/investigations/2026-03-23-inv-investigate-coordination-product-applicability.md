## Summary (D.E.K.N.)

**Delta:** Orch-go's coordination evidence identifies three product-viable multi-agent patterns (region-routed parallelism, modification-anchored concurrent edits, contract-mediated composition), but the highest-value pattern — genuine composition where agents produce outputs that must integrate — has a critical evidence gap at the composition-verification layer.

**Evidence:** 329 trials across 8 experiments, coordination model (`.kb/models/coordination/model.md`), compositional correctness gap evidence from 3 domains (SE, sheet metal DFM, LED gate stack), modification-task self-coordination (40/40 SUCCESS), structural placement (29/29 at N=2), scaling degradation analysis (N=4,6).

**Knowledge:** The Reddit poster's binary framing (single-agent works, multi-agent fails) is incomplete — there's a middle ground where structural coordination enables reliable multi-agent products. But the interesting products aren't "multi-agent" in the way frameworks sell them (agents talking to agents). They're structurally decomposed systems where the coordination is baked into the problem shape, not bolted on through messaging.

**Next:** Strategic decision for Dylan — which product archetype to pursue first. The "constraint-first domain processor" (autoresearch pattern applied to new domains) has the strongest evidence base. The "contract-mediated composition" pattern is higher-value but needs the composition-verification gap closed.

**Authority:** strategic - Product direction is a value judgment about which market to address, not a technical decision.

---

# Investigation: Can Orch-Go Coordination Findings Unlock Reliable Multi-Agent Products Beyond Single-Transform?

**Question:** What problem types genuinely require multi-agent composition, which orch-go mechanisms enable them, and what would a structurally-coordinated product look like?

**Started:** 2026-03-23
**Updated:** 2026-03-23
**Owner:** orch-go-ad3vx
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/coordination/model.md` | extends | yes (read all claims, cross-referenced evidence) | none |
| `.kb/threads/2026-03-22-constraint-first-design-orchestration-wrong.md` | extends | yes | none |
| `.kb/threads/2026-03-23-constrain-shape-probe-three-layer.md` | extends | yes | none |
| `.kb/threads/2026-03-22-beads-atom-problem-work-molecules.md` | extends | yes | none |
| `.kb/models/harness-engineering/model.md` (§compositional correctness gap) | extends | yes (read lines 304-329) | none |

---

## Findings

### Finding 1: Four problem categories with distinct coordination requirements

The coordination evidence reveals that "multi-agent" is not a single problem — it's at least four structurally different problems with different coordination needs:

| Category | Structure | Coordination Cost | Evidence |
|----------|-----------|-------------------|----------|
| **A: Parallelizable-independent** | Each agent works on a separate output. No integration beyond aggregation. | Zero (N=1 per region) | orch-go daemon production, autoresearch (N=1 by design) |
| **B: Modification-anchored** | Multiple agents modify different parts of the same artifact | Zero (self-coordinating) | 40/40 SUCCESS including 10/10 no-coord (modification experiment) |
| **C: Additive-convergent** | Multiple agents add new content to the same artifact with gravitational insertion points | Moderate (placement needed) | 29/29 SUCCESS with placement, 0/60 without (4-condition experiment) |
| **D: Genuinely integrative** | Agents produce qualitatively different outputs that must compose into something neither could produce alone | Unknown (untested) | Compositional correctness gap observed in 3 domains but no coordination experiment |

**Evidence:** Coordination model 329 trials, modification probe (N=40), scaling probe (N=130 agent invocations).

**Source:** `.kb/models/coordination/model.md` lines 1-120, modification probe, scaling probe

**Significance:** The Reddit poster's successful products (email→CRM, resume→structured, FAQ→answer) are all Category A — parallelizable-independent. Their failed products used agent-to-agent communication to solve Category C/D problems. The poster's conclusion ("multi-agent is over-engineering") is correct for Category A but incorrect as a universal claim. Categories B and C are reliably solvable with structural mechanisms. Category D is the unsolved frontier.

---

### Finding 2: The Reddit poster's failures match the exact failure modes in the coordination model

The poster described using CrewAI and LangGraph with agent-to-agent communication. The coordination model has specific evidence for why these fail:

- **CrewAI**: Gate-based (manager LLM routing). Model assessment: "Does not work." Evidence: GitHub #4783 routing failures, literature review.
- **LangGraph**: Gate-based (conditional graph edges). Model assessment: "Does not work." Same gate failure mechanism.
- **Communication-based coordination**: 0/40 SUCCESS without merge education, 6/20 SUCCESS (30%) with merge education. Five distinct failure patterns persist even with correct knowledge.

The poster's experience is predicted by the model. Their successful products avoided coordination entirely (Category A), and their failed products used communication-based coordination (gates, not attractors) for problems that required structural solutions.

**Evidence:** Coordination model Part III framework assessments, merge-educated messaging probe (N=20), 4-condition experiment (N=80)

**Source:** `.kb/models/coordination/model.md` lines 241-278

**Significance:** This isn't a "multi-agent doesn't work" finding — it's a "communication-based multi-agent doesn't work" finding. The distinction matters for product design.

---

### Finding 3: Three product archetypes are viable with current evidence

**Archetype 1: Constraint-First Domain Processor (Category A, strong evidence)**

The autoresearch pattern: constrain the problem surface until N=1 per work unit, then parallelize horizontally. Not "multi-agent coordination" but "single-agent at scale."

- **Mechanism:** Problem decomposition eliminates coordination need entirely (N=1 boundary condition)
- **Evidence strength:** Strong. autoresearch (48k stars, 16 days). orch-go daemon production (each agent gets own issue).
- **Example products:** Parallel code review (each file reviewed independently), batch document processing, parallel test generation, security audit per module
- **Ceiling:** Can only solve problems where quality doesn't depend on cross-unit coherence. Cannot build a cathedral this way — only bricks.

**Archetype 2: Region-Routed Concurrent System (Categories B+C, strong evidence)**

Multiple agents work on the same codebase/artifact with structural separation. The orchestrator assigns non-overlapping regions, agents work in parallel, outputs merge cleanly.

- **Mechanism:** Structural placement (attractors) for additive work, natural anchoring for modification work
- **Evidence strength:** Strong for N=2 (29/29 placement, 40/40 modification). Degrades at N>2 with limited insertion points (70% pairwise at N=4, 67% at N=6).
- **Example products:** Parallel feature implementation across a codebase, multi-section document authoring with assigned sections, concurrent configuration editing
- **Ceiling:** Requires insertion-points >= agents for additive work. Import/shared-resource conflicts add a second failure dimension. Works best when work regions are naturally separable.

**Archetype 3: Contract-Mediated Composition (Category D, theoretical)**

An architect agent produces a structural contract (interface definition, schema, component boundaries), then implementation agents build to the contract. The contract IS the attractor — it constrains each agent's output surface so integration succeeds.

- **Mechanism:** The contract replaces communication. Instead of agents talking to each other, they each implement against a shared structural specification. This is the Anthropic production pattern (lead agent defines work regions before subagents start).
- **Evidence strength:** Weak for the composition layer. Strong evidence that the individual pieces work (placement, modification anchoring, attractor resilience). NO evidence for the composition verification step — who checks that the contract was satisfied and parts integrate correctly?
- **Example products:** Full-stack feature development (backend/frontend/migration from shared API contract), multi-modal content creation (research/visualization/writing from shared outline), microservice implementation from shared protobuf/OpenAPI spec
- **Ceiling:** The compositional correctness gap. Individually correct implementations may not compose into a correct whole. The LED gate stack evidence (valid geometry, non-functional routing) and the daemon.go evidence (30 correct commits, structural degradation) show this is a real failure mode.

**Evidence:** Coordination model claims 1-5, compositional correctness gap (harness engineering model lines 304-329), constrain-shape-probe thread, beads atom thread

**Source:** Multiple (cited inline above)

**Significance:** Archetype 1 is proven but incremental (it's "parallel single-agent," not truly multi-agent). Archetype 2 is proven for specific problem structures. Archetype 3 is the highest-value pattern but has the largest evidence gap. A product team choosing between these is choosing between "safe and limited" vs "ambitious and uncertain."

---

### Finding 4: The composition-verification gap is the critical unsolved problem

The constrain-shape-probe thread (2026-03-23) identifies three layers for composed systems: **constrain** (gates), **shape** (attractors), **probe** (composition verification). The coordination model has strong evidence for layers 1 and 2 but almost no evidence for layer 3.

The compositional correctness gap manifests identically across domains:

| Domain | Components Valid | Composition Fails | What No Gate Checks |
|--------|-----------------|-------------------|---------------------|
| Multi-agent SE | Each commit compiles, passes review | 30 commits = +892 lines, 6 duplicated concerns | Cross-agent coherence |
| Sheet metal DFM | Each operation passes DFM rules | Bend line crosses hardware location | Inter-operation interference |
| LED gate stack | Valid geometry, manifold, printable | Disconnected LED channels | Connectivity/routing |
| Contract-mediated products | Each agent implements to spec | Integration fails at boundaries | Cross-component composition |

The thread concludes: "this is a vocabulary contribution and design checklist, not a discovery." The checklist: (1) name the emergent property as a claim, (2) write it as explicit specification, (3) build test apparatus. Step 3 is standard integration testing. Steps 1-2 are where systems fail.

**Evidence:** Compositional correctness gap evidence from harness engineering model, LED probe (~150 renders), SCS AI Part Builder probe, constrain-shape-probe thread

**Source:** `.kb/models/harness-engineering/model.md` lines 304-329, `.kb/threads/2026-03-23-constrain-shape-probe-three-layer.md`

**Significance:** For Archetype 3 (the highest-value product pattern), you need to solve composition verification — not just structural coordination. The coordination model tells you how to prevent merge conflicts. It doesn't tell you how to verify that independently-correct components produce a correct whole. This is the gap between "parallel coding tool" and "reliable multi-agent system."

---

### Finding 5: Evidence is narrower than it appears

The 329 trials are impressive but concentrated:

| Dimension | What's Tested | What's Not |
|-----------|--------------|------------|
| **Domain** | Software engineering (Go code) | Content creation, data analysis, design, hardware, legal, finance |
| **Model family** | Claude (primarily Haiku 4.5) | GPT-4, Gemini, open-source models |
| **Scale** | N=2 (strong), N=4,6 (degrading) | N=10+ (real production systems) |
| **Integration** | Git merge (textual) | API composition, database writes, UI rendering, multi-format output |
| **Task type** | Additive same-file, modification same-file | Cross-file, cross-repo, cross-language, qualitatively different outputs |
| **Failure mode** | Merge conflicts (binary: clean/conflict) | Semantic conflicts, quality degradation, coherence drift |

The fundamental mechanisms likely transfer (structural placement is domain-agnostic, attractor vs gate is a design principle, not a code pattern). But the specific numbers (100% placement success, 0% communication success) are scoped to same-file additive Go code with Claude Haiku.

**Evidence:** Coordination model evidence quality annotations throughout (every claim has explicit evidence quality tags)

**Source:** `.kb/models/coordination/model.md` — evidence quality fields on every claim

**Significance:** A product team building on these findings should treat the mechanisms as directional guidance and the specific numbers as upper/lower bounds within the tested domain. Cross-domain transfer of the principles (structure > communication, attractors > gates) is plausible but unvalidated.

---

## Synthesis

**Key Insights:**

1. **The binary framing is wrong — the useful question is "what type of multi-agent?"** The Reddit poster's experience (single works, multi fails) is correct for communication-based multi-agent (Category A vs C/D with gates). It's incorrect as a universal claim because structural multi-agent (Categories B/C with attractors) is reliably solvable. The industry consensus is stuck because the dominant frameworks (CrewAI, LangGraph, OpenAI Agents SDK) all implement gate-based coordination, which the evidence shows fails at 70-100% rates.

2. **The highest-value product pattern requires solving composition, not just coordination.** Coordination (preventing merge conflicts) is necessary but not sufficient. The compositional correctness gap — where individually valid components compose into non-functional wholes — is the barrier to Archetype 3 (contract-mediated composition). This gap exists in SE (daemon.go), manufacturing (DFM), and physical design (LED routing). Closing it requires named composition claims + test apparatus, per the constrain-shape-probe framework.

3. **The most defensible product insight is the "coordination tax" inversion.** Every multi-agent framework markets itself as enabling coordination. The evidence says: the best coordination is the one you eliminate. Products that constrain problem surfaces (autoresearch), decompose into naturally-separable regions (orch-go daemon), or choose task types that self-coordinate (modification anchoring) dramatically outperform products that try to coordinate through communication. This is counter-intuitive and counter-narrative — it says the winning multi-agent strategy is to make each agent as independent as possible.

**Answer to Investigation Question:**

Yes, orch-go's coordination findings can unlock multi-agent products beyond single-transform — but not the way the industry imagines. The evidence supports three viable patterns:

1. **Constraint-first parallelism** (proven, incremental) — autoresearch pattern applied to new domains. Not truly "multi-agent" but scales single-agent horizontally.
2. **Region-routed concurrent work** (proven for N=2, degrades at scale) — structural placement for additive tasks, natural anchoring for modification tasks. Genuine multi-agent within tested bounds.
3. **Contract-mediated composition** (theoretical, highest value) — architect agent produces structural contract, implementation agents build to contract. Requires solving the composition-verification gap.

The critical gap for productization is NOT coordination (that's solved for the tested cases) — it's composition verification. The open question is: once agents produce individually correct outputs, how do you verify the composition works? This is domain-specific (integration testing for code, connectivity analysis for physical design, coherence checking for content) and cannot be solved with a general-purpose framework.

---

## Structured Uncertainty

**What's tested:**

- Structural placement prevents merge conflicts at N=2 (29/29 SUCCESS across 2 task types)
- Communication-based coordination fails at 70-100% for additive same-file tasks (100/120 CONFLICT)
- Modification tasks self-coordinate without any mechanism (40/40 SUCCESS)
- Automated attractor discovery works from observed collision patterns (7/7 SUCCESS after 2 collisions)
- Gate-based self-checking is as ineffective as no coordination (20/20 CONFLICT)

**What's untested:**

- Whether structural placement works outside software engineering (content, design, data analysis)
- Whether contract-mediated composition can be verified automatically
- Whether these mechanisms work with non-Claude models
- Whether scaling to N>6 agents is viable for any product scenario
- Whether the coordination principles apply to non-merge integration (API composition, multi-format output)

**What would change this:**

- If structural placement failed at 50%+ rate in a non-SE domain, the domain transfer claim weakens significantly
- If a communication-based approach achieved >80% success in any tested scenario, the "attractors > gates" principle needs revision
- If composition verification proves impossible to automate for a specific product domain, Archetype 3 is not viable for that domain

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Choose product archetype to pursue | strategic | Irreversible market/direction decision, resource commitment |
| Design composition-verification experiments | architectural | Cross-domain, multiple valid approaches |
| Package coordination findings for external communication | strategic | Public narrative, career/brand implications |

### Recommended Approach: Staged Product Validation

**Start with Archetype 1 (constraint-first parallelism) in a non-SE domain to validate cross-domain transfer, while designing composition-verification experiments for Archetype 3.**

**Why this approach:**
- Archetype 1 has the strongest evidence and lowest risk — validates the cross-domain transfer hypothesis with minimal investment
- Composition-verification experiment design can happen in parallel without blocking product work
- If cross-domain transfer fails, you learn early and cheaply

**Trade-offs accepted:**
- Archetype 1 products are incremental, not revolutionary — may not differentiate from existing tools
- Archetype 3 remains theoretical until composition verification is solved

**Implementation sequence:**
1. Pick a non-SE domain (content generation, data analysis, or security audit) and implement the autoresearch pattern — this validates whether the coordination principles transfer
2. Design a composition-verification experiment: architect agent produces a contract, two implementation agents build to it, measure whether integration succeeds without human intervention
3. Based on results, decide whether to pursue Archetype 2 (region-routed, proven but limited) or Archetype 3 (contract-mediated, ambitious but uncertain)

### Alternative Approaches Considered

**Option B: Go directly to Archetype 3 (contract-mediated composition)**
- **Pros:** Highest value, most differentiated
- **Cons:** Largest evidence gap, composition verification unsolved, could waste significant time
- **When to use instead:** If there's a specific product opportunity with clear demand where composition verification is tractable (e.g., full-stack features with well-defined API contracts)

**Option C: Productize the coordination framework itself (sell the methodology)**
- **Pros:** Directly leverages the 329-trial evidence base, doesn't require solving composition
- **Cons:** Framework market is saturated (CrewAI, LangGraph, etc.), "our framework is better because structure > communication" is a hard sell without production evidence from multiple domains
- **When to use instead:** If the blog post / publication strategy gains traction and creates demand for the methodology

**Rationale for recommendation:** Option A provides the fastest feedback loop on the cross-domain transfer hypothesis while keeping the higher-value options open. It's the minimum-risk path to validating whether the findings generalize.

---

### Things to watch out for:

- The coordination model's numbers (100% placement success, 0% communication success) are upper/lower bounds in the tested domain — expect degradation when crossing domain boundaries
- "Structural placement" in non-SE domains may look very different from file-region assignment — the principle transfers but the implementation must be domain-specific
- The "coordination tax inversion" insight (best coordination is the one you eliminate) may be a hard sell to customers who bought into the agent-swarm narrative

### Areas needing further investigation:

- Cross-domain transfer experiment: does structural placement work for content creation? Data analysis? Design?
- Composition-verification experiment: can the constrain-shape-probe pattern be automated?
- Model family testing: do the coordination principles hold for GPT-4, Gemini, open-source?
- Scale testing: what happens at N=10+ with structural placement in production-scale codebases with many natural insertion points?

### Success criteria:

- Cross-domain transfer validated: structural placement achieves >80% success in at least one non-SE domain
- At least one product archetype demonstrated end-to-end with real users
- Composition-verification gap addressed (at least a prototype for one domain)

---

## References

**Files Examined:**
- `.kb/models/coordination/model.md` — Full coordination model (329 trials, 8 experiments, 4 primitives)
- `.kb/models/harness-engineering/model.md` lines 304-329 — Compositional correctness gap
- `.kb/threads/2026-03-23-constrain-shape-probe-three-layer.md` — Three-layer verification model
- `.kb/threads/2026-03-22-constraint-first-design-orchestration-wrong.md` — Constraint-first design
- `.kb/threads/2026-03-22-beads-atom-problem-work-molecules.md` — Work molecules / composition
- `.kb/threads/2026-03-20-open-loop-systems-unifying-pattern.md` — Sensor gap pattern
- `.kb/models/domain-harness-architecture/model.md` — Cross-domain enforcement layering
- `.kb/models/coordination/probes/2026-03-23-probe-modification-task-experiment.md` — Modification self-coordination
- `.kb/models/coordination/probes/2026-03-22-probe-automated-attractor-discovery.md` — Automated attractor discovery
- `.kb/models/coordination/probes/2026-03-23-probe-agent-scaling-limited-insertion-points.md` — Scaling degradation

**Related Artifacts:**
- **Model:** `.kb/models/coordination/model.md` — Primary evidence source
- **Thread:** `.kb/threads/2026-03-23-constrain-shape-probe-three-layer.md` — Composition verification framework
- **Thread:** `.kb/threads/2026-03-22-constraint-first-design-orchestration-wrong.md` — Constraint-first design principle

---

## Investigation History

**2026-03-23:** Investigation started
- Initial question: Can orch-go coordination findings unlock reliable multi-agent products beyond single-transform?
- Context: Reddit post (25+ agents built) captures industry consensus that multi-agent is over-engineering. Orch-go has 329 trials of evidence that may unlock a middle ground.

**2026-03-23:** Analysis complete
- Read full coordination model (329 trials), all referenced probes and threads
- Identified 4 problem categories, 3 product archetypes
- Key finding: composition verification (not coordination) is the critical gap for highest-value products
- Status: Complete
