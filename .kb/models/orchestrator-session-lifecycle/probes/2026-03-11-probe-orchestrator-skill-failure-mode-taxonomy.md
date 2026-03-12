# Probe: Orchestrator Skill Failure Mode Taxonomy — Complete Catalog from 6 Investigations

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-11
**Status:** Complete

---

## Question

The model documents 5 failure modes in its "Why This Fails" section: Frame Collapse, Self-Termination, Competing Instruction Hierarchy, Behavioral Constraint Dilution, and State Derivation Disagreement. Are these 5 modes the complete taxonomy, or do the 6 source investigations (Jan-Mar 2026) reveal additional distinct failure modes not yet represented in the model? Which failures were the PRIMARY drivers of the skill's evolution?

---

## What I Tested

Read and cross-referenced all 6 investigations specified in the task, plus 7 related probes from the orchestrator-session-lifecycle model's probes directory:

**Investigations read:**
1. `2026-01-18-inv-update-orchestrator-skill-add-frustration.md` — Frustration trigger protocol addition
2. `2026-02-24-design-orchestrator-skill-behavioral-compliance.md` — Identity vs action compliance gap (17:1 signal ratio)
3. `2026-03-01-design-infrastructure-systematic-orchestrator-skill.md` — Testing infrastructure design (behavioral scenarios, linting)
4. `2026-03-04-design-simplify-orchestrator-skill.md` — Skill simplification from 2,368→448 lines
5. `2026-03-05-inv-design-orchestrator-skill-update-incorporating.md` — 72-commit infrastructure delta, 13 stale edits
6. `2026-02-28 evidence/orchestrator-intent-spiral/orchestrator-skill-snapshot.md` — Skill state at time of intent spiral failure

**Additional sources cross-referenced:**
- `2026-02-28-investigation-orchestrator-intent-spiral.md` — Cascaded intent displacement (the canonical intent spiral)
- Probe: `2026-02-24-probe-orchestrator-skill-behavioral-compliance.md`
- Probe: `2026-03-01-probe-constraint-dilution-threshold.md`
- Probe: `2026-03-02-probe-emphasis-language-constraint-compliance.md`
- Probe: `2026-02-16-orchestrator-skill-orientation-redesign.md`
- Probe: `2026-03-01-probe-agent-framework-behavioral-constraints-landscape.md`
- Probe: `2026-02-25-probe-orchestrator-skill-cross-project-injection-failure.md`
- Probe: `2026-02-18-orchestrator-skill-cli-staleness-audit.md`
- Probe: `2026-02-27-probe-communication-breakdown-postmortem-3-sessions.md`
- The model itself: `orchestrator-session-lifecycle/model.md`

---

## What I Observed

### Finding 1: The model's 5 failure modes are confirmed but incomplete

The model's "Why This Fails" section accurately documents 5 failure modes. However, the 6 investigations reveal **7 additional distinct failure modes** not captured in the model's current taxonomy, totaling 12 failure modes across 4 categories (prompt-level, infrastructure-level, structural, temporal).

### Finding 2: Three failure modes drove the Jan→Mar 2026 skill evolution

The PRIMARY drivers were:
1. **Behavioral Constraint Dilution** (Feb-Mar 2026 probes) — Drove the 2,368→448 line simplification
2. **Competing Instruction Hierarchy** (Feb 2026 investigation) — Drove the two-layer enforcement strategy (hooks + skill restructuring)
3. **Cascaded Intent Displacement** (Feb 28 intent spiral) — Drove the addition of intent clarification gates, `--intent` flag consideration, and experiential vs production distinction

These three together explain: why the skill shrank 82% (dilution), why hooks now enforce behavioral constraints (competing hierarchy), and why the routing table was extended with intent types (intent displacement).

### Finding 3: The 12 failure modes cluster into 4 layers

| Layer | Count | Description |
|-------|-------|-------------|
| Prompt-level | 4 | Failures in skill content or instruction processing |
| Infrastructure-level | 3 | Failures in injection, deployment, or enforcement mechanisms |
| Structural | 3 | Failures in skill architecture or design patterns |
| Temporal | 2 | Failures that emerge over time or across sessions |

### Finding 4: 6 of 12 failure modes have been resolved or mitigated

| Status | Count | Examples |
|--------|-------|---------|
| Resolved by hooks | 3 | Task tool use, bd close, code reading |
| Resolved by simplification | 2 | Constraint dilution, MUST fatigue |
| Resolved by template fix | 1 | Self-termination |
| Open / fundamental | 4 | Intent displacement, frame collapse, cross-project injection, knowledge surfacing gap |
| Partially mitigated | 2 | Competing instruction hierarchy, CLI staleness |

---

## Model Impact

- [x] **Extends** model with: 7 additional failure modes not in the current "Why This Fails" section — Cascaded Intent Displacement, Skill Content Staleness, MUST Fatigue / Constraint Overhead, Error-Correction Feedback Loop, Cross-Project Injection Failure, Debugging Paralysis (Frame Guard Liability), and Orientation Decay. These should be merged into the model's taxonomy, organized by layer.

- [x] **Confirms** invariant: The existing 5 failure modes (Frame Collapse, Self-Termination, Competing Instruction Hierarchy, Behavioral Constraint Dilution, State Derivation Disagreement) are accurately described and correctly categorized.

- [x] **Extends** model with: Primary driver identification — three failure modes (Dilution, Competing Hierarchy, Intent Displacement) explain the majority of the Jan→Mar 2026 skill evolution trajectory. The model's Evolution section documents WHAT changed but not which failure modes DROVE each phase. Linking failure modes to evolution phases would make the model more explanatory.

---

## Notes

- The full taxonomy with evidence chains is in the SYNTHESIS.md at `.orch/workspace/og-inv-task-orchestrator-skill-11mar-6c43/SYNTHESIS.md`
- Several failure modes interact: Constraint Dilution amplifies Competing Instruction Hierarchy (more constraints = weaker signal per constraint). Error-Correction Feedback Loop amplifies Intent Displacement (corrections drive deeper into wrong methodology).
- The replication failure caveat on the dilution curve (noted in both constraint-dilution and emphasis-language probes) means specific threshold numbers are directional, not established.
