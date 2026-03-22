# Probe: Control Theory Component Mapping — 14 MAST Failure Modes

**Model:** coordination
**Date:** 2026-03-22
**Status:** Complete
**claim:** N/A (extending model with new theoretical framing, not testing existing claim)
**verdict:** extends

---

## Question

Does the hypothesized mapping (Route→Actuator, Sequence→Reference signal, Throttle→Controller, Align→Sensor) hold across all 14 MAST failure modes? If so, the coordination framework inherits control theory's formal results. If not, the mapping is analogy, not structure.

---

## What I Tested

Systematic mapping of each MAST failure mode (FM-1.1 through FM-3.3) to control theory components using control loop semantics:
- **Reference signal (r)**: desired output/setpoint
- **Controller (C)**: compares error, decides corrective action
- **Actuator (A)**: executes the decision
- **Sensor (S)**: measures actual output, feeds back

For each mode: identified the primary control component failure, checked whether it matches the hypothesized primitive→component mapping, and noted any cross-component bleed.

Source: 14 failure modes from MAST taxonomy (Cemri et al., Berkeley, NeurIPS 2025, 1642 traces, kappa=0.88), already mapped to primitives in `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md`.

---

## What I Observed

### Full Mapping Table

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

### Summary Statistics

- **Clean (1:1 primitive→component)**: 9/14 (64%)
- **Messy (sensor bleed into non-sensor primitive)**: 4/14 (29%)
- **Cross-mapping (primitive→unexpected component)**: 1/14 (7%)

### The Sensor Bleed Pattern

4 of 5 non-clean mappings have the same structure: the sensor component appears in failures mapped to non-sensor primitives.

- FM-1.3 (Sequence/Reference): Agent can't observe what steps are completed → needs sensor
- FM-2.1 (Sequence/Reference): Agent loses knowledge of position on trajectory → needs sensor
- FM-2.5 (Route/Actuator): Route delivered info, but receiver didn't sense/process it → needs sensor
- FM-3.1 (Throttle/Controller): Controller stops early because it can't sense incompleteness → needs sensor

This is exactly what control theory predicts: when the sensor fails, OTHER components appear to fail. A controller can't regulate without sensor feedback. An actuator can't stay on target without sensor correction. Sensor failure cascades.

### The One Cross-Mapping

FM-2.6 (Reasoning-action mismatch) is mapped to Align (sensor hypothesis), but is actually an actuator failure. The controller (reasoning) computes the right action; the actuator (action execution) produces a different output. This is a textbook controller-actuator coupling failure in control theory. It breaks the 1:1 mapping: not all Align failures are sensor failures.

### Sensor Involvement Tally

- Direct sensor failures (Align→Sensor): 7 modes
- Sensor bleed into non-Align failures: 4 modes
- **Total with sensor involvement: 11/14 (79%)**
- Pure non-sensor failures: 3/14 — FM-1.2 (pure Actuator), FM-1.5 (pure Reference), FM-2.6 (Align→Actuator crossover)

### Connection to Open-Loop Thread

The open-loop thread found 14/16 system failures are missing sensors (87.5%). This probe finds 11/14 MAST failures involve sensors (79%). These numbers converge from independent scopes:

- Open-loop thread: orch-go internal systems (behavioral accretion, stale decisions, gate bypass, etc.)
- MAST mapping: academic multi-agent failure taxonomy (1642 traces, 7 frameworks)

Same finding in different vocabularies: **most failures are observation failures.** Systems act without verifying consequences.

---

## Model Impact

- [ ] **Confirms** invariant: N/A
- [ ] **Contradicts** invariant: N/A
- [x] **Extends** model with: control theory component mapping — structural homology (not isomorphism) between primitives and control components

### What transfers from control theory:
1. **Sensor dominance**: Removing the sensor (opening the loop) is the most catastrophic failure. Maps to: Align is the most common failure mode (7/14 direct, 11/14 with bleed).
2. **Loop closure principle**: Every effective system needs a closed feedback loop. Maps to: frameworks implementing Align succeed more.
3. **Sensor cascade**: Many "actuator" or "controller" failures are caused by bad sensor data. Maps to: non-Align failures often have Align components (sensor bleed).

### What does NOT transfer:
1. **Stability analysis** (Bode plots, Nyquist criteria) — requires quantitative transfer functions. Primitive mapping is categorical.
2. **Optimal control** (LQR, MPC) — requires mathematical plant models. Multi-agent coordination doesn't have these.
3. **Formal controllability/observability** — definitions don't map to agent primitives.

### Per-primitive mapping quality:

| Primitive | Component | Quality | Notes |
|-----------|-----------|---------|-------|
| Align | Sensor | Strong (6/7 clean) | One exception: FM-2.6 maps to Actuator |
| Route | Actuator | Good (1/2 clean) | FM-2.5 has sensor bleed but primary is Actuator |
| Sequence | Reference | Weak (1/3 clean) | 2/3 have sensor bleed — need to sense position on trajectory |
| Throttle | Controller | Weak (0/1 clean) | Single example has sensor bleed |

### Verdict

**Structural homology, not isomorphism.** The mapping is 64% clean, with a systematic pattern in the mess (sensor bleed). The qualitative insights from control theory transfer and are useful for diagnostic framing. The quantitative formal tools do not transfer. The framework inherits control theory's *intuitions* (sensor dominance, loop closure, cascade failures) but not its *theorems* (stability criteria, optimal control, formal observability).

---

## Notes

- The Align→Sensor mapping is the strongest and most useful. It gives theoretical grounding to the observation that Align is the "meta-primitive."
- The Sequence→Reference mapping is the weakest. Sequence failures almost always involve a sensor component (agent needs to observe where it is on the trajectory), suggesting Sequence may be a composite of Reference + Sensor.
- FM-2.6 (reasoning-action mismatch) is the single most interesting case: it's an Align primitive failure that maps to an Actuator component. This means "shared model of correctness" (Align) sometimes fails not because sensing is wrong, but because execution doesn't match intent. The Align primitive may be broader than just "sensor."
- The 79% sensor involvement rate (this probe) converging with the 87.5% sensor rate (open-loop thread) from independent scopes is strong evidence that sensor poverty is the dominant structural problem in multi-agent systems.
