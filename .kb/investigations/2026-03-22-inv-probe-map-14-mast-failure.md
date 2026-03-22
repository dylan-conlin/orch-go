## Summary (D.E.K.N.)

**Delta:** The Route→Actuator, Sequence→Reference, Throttle→Controller, Align→Sensor mapping is 64% clean (9/14 modes), with a systematic pattern in the mess: sensor components bleed into non-sensor primitives in 4 of 5 non-clean cases.

**Evidence:** Mapped all 14 MAST failure modes to control theory components. 11/14 (79%) involve sensors, converging with the open-loop thread's 14/16 (87.5%) from independent scope.

**Knowledge:** The mapping is structural homology, not isomorphism. Qualitative control theory insights transfer (sensor dominance, loop closure, cascade failures). Quantitative formal tools (stability analysis, optimal control) do not.

**Next:** Merge into coordination model. The control theory framing adds diagnostic value but not formal guarantees.

**Authority:** implementation - Extends existing model within established patterns

---

# Investigation: Map 14 MAST Failure Modes to Control Theory Components

**Question:** Does the hypothesized mapping (Route→Actuator, Sequence→Reference, Throttle→Controller, Align→Sensor) hold across all 14 MAST failure modes, making the mapping structural (inherits formal results) or analogical (useful for intuition only)?

**Started:** 2026-03-22
**Updated:** 2026-03-22
**Owner:** probe agent (orch-go-fp57t)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** coordination

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md` | extends | Yes (14 MAST modes used as input) | None |
| `.kb/threads/2026-03-20-open-loop-systems-unifying-pattern.md` | confirms | Yes (sensor dominance converges) | None |
| `.kb/models/coordination/model.md` | extends | Yes (four primitives are the substrate) | None |

---

## Findings

### Finding 1: The full 14-mode mapping

| # | MAST Mode | Description | Primitive | Hyp. Component | Actual Primary | Clean? |
|---|-----------|-------------|-----------|----------------|----------------|--------|
| FM-1.1 | Disobey task spec | Align | Sensor | Sensor | Yes |
| FM-1.2 | Disobey role spec | Route | Actuator | Actuator | Yes |
| FM-1.3 | Step repetition | Sequence | Reference | Reference + Sensor | Messy |
| FM-1.4 | Loss of conversation history | Align | Sensor | Sensor | Yes |
| FM-1.5 | Unaware of termination | Sequence | Reference | Reference | Yes |
| FM-2.1 | Conversation reset | Sequence | Reference | Reference + Sensor | Messy |
| FM-2.2 | Fail to ask for clarification | Align | Sensor | Sensor | Yes |
| FM-2.3 | Task derailment | Align | Sensor | Sensor | Yes |
| FM-2.4 | Information withholding | Align | Sensor | Sensor | Yes |
| FM-2.5 | Ignored other agent's input | Route | Actuator | Actuator + Sensor | Messy |
| FM-2.6 | Reasoning-action mismatch | Align | Sensor | **Actuator** | **Cross** |
| FM-3.1 | Premature termination | Throttle | Controller | Controller + Sensor | Messy |
| FM-3.2 | No/incomplete verification | Align | Sensor | Sensor | Yes |
| FM-3.3 | Incorrect verification | Align | Sensor | Sensor (miscalibrated) | Yes |

**Source:** 14 MAST failure modes from `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md` (Finding 1), mapped through control theory semantics (reference signal, controller, actuator, sensor).

**Significance:** 9/14 (64%) map cleanly, 4/14 (29%) are messy with sensor bleed, 1/14 (7%) is a cross-mapping. The mapping is mostly clean but not perfectly 1:1.

---

### Finding 2: Sensor bleed — the systematic pattern in the mess

4 of 5 non-clean mappings share a common structure: the sensor component bleeds into failures mapped to non-sensor primitives.

- **FM-1.3** (Sequence→Reference): Agent can't observe which steps are already done → needs sensor
- **FM-2.1** (Sequence→Reference): Agent lost knowledge of position on trajectory → needs sensor
- **FM-2.5** (Route→Actuator): Route delivered info, but receiver didn't sense/process it → needs sensor
- **FM-3.1** (Throttle→Controller): Controller stops early because it can't sense incompleteness → needs sensor

This is exactly what control theory predicts: when the sensor fails, other components APPEAR to fail. A controller can't regulate without sensor feedback. An actuator can't stay on target without sensor correction.

**Source:** Control theory principle: sensor failure cascades to all other components because they depend on feedback.

**Significance:** The sensor bleed pattern means the 79% sensor involvement rate understates the importance — sensors are load-bearing for ALL four components. This gives theoretical grounding to "Align is the meta-primitive."

---

### Finding 3: FM-2.6 is the single genuine counterexample

FM-2.6 (reasoning-action mismatch) is mapped to Align (hypothesized sensor), but is actually an actuator failure. The agent's reasoning (controller) computes the correct action, but its execution (actuator) produces something different. This is a textbook controller-actuator coupling failure.

This breaks the 1:1 mapping: not all Align failures are sensor failures. The Align primitive is broader than "sensor" — it includes both sensing correctness (FM-1.1, FM-2.2, FM-3.2) AND executing in alignment with intent (FM-2.6).

**Source:** Control theory: reasoning-action mismatch = controller output doesn't match actuator output.

**Significance:** The Align primitive may be a composite of two control theory concepts: sensor (observing correctness) + actuator fidelity (executing as intended). If this is right, Align decomposition should separate "observation alignment" from "execution alignment."

---

### Finding 4: Convergence with open-loop thread

| Source | Scope | Sensor involvement | Rate |
|--------|-------|-------------------|------|
| Open-loop thread | orch-go internals (7 systems) | 14/16 open loops are missing sensors | 87.5% |
| This probe | MAST academic taxonomy (14 modes, 1642 traces) | 11/14 modes involve sensors | 79% |

Independent scopes, convergent finding. The ~80-88% sensor involvement rate from two unrelated analyses strengthens the claim that sensor poverty is the dominant structural problem in multi-agent systems.

**Source:** `.kb/threads/2026-03-20-open-loop-systems-unifying-pattern.md` (14/16 finding), this probe (11/14 finding).

**Significance:** These ARE the same finding in different vocabularies. "Most failures are observation failures" is the unified statement. The convergence from orch-go internals and academic multi-agent taxonomy makes this a robust cross-validated claim.

---

## Synthesis

**Key Insights:**

1. **The mapping is structural homology, not isomorphism** — 64% clean, 29% messy with sensor bleed, 7% cross-mapping. The framework inherits control theory's qualitative insights but not its quantitative formal tools.

2. **Sensor bleed validates "Align is the meta-primitive"** — Non-sensor failures almost always have sensor components. Control theory explains this: sensors are load-bearing for all other components. When the sensor fails, the controller, actuator, and reference tracking all degrade.

3. **Per-primitive mapping quality varies** — Align→Sensor is strong (6/7 clean). Route→Actuator is good (primary match). Sequence→Reference is weak (2/3 have sensor bleed). Throttle→Controller has only one example, messy.

4. **FM-2.6 suggests Align may need decomposition** — The Align primitive covers both "observing correctness" (sensor) and "executing as intended" (actuator fidelity). This connects to the model's open question about whether Align should decompose into sub-primitives.

**Answer to Investigation Question:**

The mapping is **mostly structural** (64% clean) with a **systematic pattern in the mess** (sensor bleed). It's not clean enough for formal results transfer (stability theorems, optimal control) but IS clean enough for diagnostic framing: "where is the failure?" → "which control component is missing?" The most useful inheritance is the sensor dominance principle — control theory provides theoretical grounding for why Align is the meta-primitive: without sensors, all other components fail.

---

## Structured Uncertainty

**What's tested:**

- Mapped all 14 MAST failure modes to control theory components (manual mapping with control theory semantics)
- Computed clean/messy/cross rates (9/4/1 out of 14)
- Identified sensor bleed as systematic pattern (4/5 non-clean cases)
- Compared to open-loop thread rate (79% vs 87.5%)

**What's untested:**

- Whether the sensor bleed pattern holds for the broader MAST dataset (1642 traces vs. 14 taxonomy modes)
- Whether FM-2.6's actuator interpretation is universally agreed (could be debated)
- Whether the Sequence→Reference mapping improves if Sequence is modeled as Reference + state estimator (composite)
- Whether non-LLM multi-agent systems show the same sensor dominance pattern

**What would change this:**

- Finding that sensor bleed doesn't hold in the full 1642-trace MAST dataset would weaken the cascade claim
- A control theory expert identifying a different primary component for any mode would revise the mapping
- Evidence that quantitative control tools CAN be applied (e.g., someone models agent error rates as transfer functions) would upgrade from homology to isomorphism

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add control theory framing to coordination model | implementation | Extends model within established patterns |
| Explore Align decomposition into sensor + actuator fidelity | architectural | Cross-component analysis |
| Use control theory vocabulary in publication framing | strategic | Publication positioning choice |

### Recommended Approach: Add control theory section to coordination model

**Why this approach:**
- The mapping is clean enough to provide diagnostic value
- Sensor dominance gives theoretical backing to "Align is the meta-primitive"
- The sensor bleed pattern explains WHY Align failures cascade

**Trade-offs accepted:**
- The mapping is approximate (64%), not exact — must be clearly labeled as homology
- Sequence→Reference is the weakest link and may need revision

---

## References

**Files Examined:**
- `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md` — 14 MAST failure mode mappings to primitives
- `.kb/models/coordination/model.md` — Four coordination primitives, experimental evidence
- `.kb/threads/2026-03-20-open-loop-systems-unifying-pattern.md` — Open-loop systems analysis, 14/16 sensor finding

**External Documentation:**
- [MAST: Why Do Multi-Agent LLM Systems Fail?](https://arxiv.org/abs/2503.13657) — Berkeley, NeurIPS 2025, 14 failure modes, 1642 traces

**Related Artifacts:**
- **Probe:** `.kb/models/coordination/probes/2026-03-22-probe-control-theory-component-mapping.md` — Detailed probe output
- **Model:** `.kb/models/coordination/model.md` — Parent model (updated with findings)

---

## Investigation History

**2026-03-22:** Investigation started
- Question: Does the Route→Actuator, Sequence→Reference, Throttle→Controller, Align→Sensor mapping hold across 14 MAST modes?
- Context: Spawned from coordination model probe to test control theory hypothesis from open-loop thread

**2026-03-22:** All 14 modes mapped
- Finding: 9/14 clean, 4/14 sensor bleed, 1/14 cross-mapping
- Key pattern: sensor components bleed into non-sensor primitives

**2026-03-22:** Investigation completed
- Status: Complete
- Key outcome: Structural homology (not isomorphism) — qualitative insights transfer, quantitative tools don't. 79% sensor involvement converges with open-loop thread's 87.5%.
