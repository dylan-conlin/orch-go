# Probe: Evidence Inventory — Orchestrator Skill Investigation Cluster (7 investigations, 4 probes)

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-12
**Status:** Complete

---

## Question

The orchestrator-session-lifecycle model has been updated through 40+ investigations and 18+ probes. The orchestrator skill investigation cluster (7 investigations, 4 probes from Feb 24 – Mar 11, 2026) contains the highest density of empirical findings. Which claims are MEASURED (test scores, line counts, concrete numbers), which are ANALYTICAL (logical deductions), and which are ASSUMED? Which claims have been independently verified across multiple sources?

---

## What I Tested

Read all 11 source documents in the cluster and extracted every empirical finding. Classified each by evidence type and cross-referenced for replication across sources.

**Sources read:**
1. `2026-02-24-design-orchestrator-skill-behavioral-compliance.md` (Investigation)
2. `2026-03-01-investigation-orchestrator-skill-behavioral-testing-baseline.md` (Investigation)
3. `2026-03-01-design-infrastructure-systematic-orchestrator-skill.md` (Investigation)
4. `2026-03-04-design-simplify-orchestrator-skill.md` (Investigation)
5. `2026-03-04-design-grammar-first-skill-architecture.md` (Investigation)
6. `2026-03-05-inv-design-orchestrator-skill-update-incorporating.md` (Investigation)
7. `2026-03-11-inv-task-orchestrator-skill-design-tension.md` (Investigation)
8. `2026-02-24-probe-orchestrator-skill-behavioral-compliance.md` (Probe)
9. `2026-03-01-probe-constraint-dilution-threshold.md` (Probe)
10. `2026-03-02-probe-emphasis-language-constraint-compliance.md` (Probe)
11. `2026-03-11-probe-orchestrator-skill-failure-mode-taxonomy.md` (Probe)

**Method:** Each claim extracted with source, evidence type classification, and replication status check.

---

## What I Observed

### Total findings: 67 claims across 4 themes

See SYNTHESIS.md for the full numbered inventory.

**Breakdown by evidence type:**
- MEASURED: 39 claims (58%) — test scores, line counts, word counts, command outputs
- ANALYTICAL: 26 claims (39%) — logical deductions from measured evidence
- ASSUMED: 2 claims (3%) — stated without direct evidence

**Breakdown by replication:**
- Multi-source (high-confidence): 14 claims appear in 2+ investigations
- Single-source: 53 claims

**Key high-confidence claims (appeared in 3+ sources):**
1. Knowledge transfers stick, behavioral constraints don't (5 sources)
2. Two-layer enforcement needed: prompt + infrastructure (5 sources)
3. Infrastructure enforcement > prompt enforcement (5 sources)
4. Identity compliance ≠ action compliance (3 sources)
5. Behavioral constraint budget ≤4 before dilution (3 sources)

**Key caveated claims:**
- Constraint dilution curve (1C→2C→5C→10C) carries REPLICATION FAILURE CAVEAT — specific threshold numbers unvalidated
- Emphasis language effect inherits dilution curve uncertainty
- All quantitative compliance numbers from single-turn --print mode; interactive sessions untested

---

## Model Impact

- [x] **Confirms** invariant: The model's 5 original failure modes (Frame Collapse, Self-Termination, Competing Instruction Hierarchy, Behavioral Constraint Dilution, State Derivation Disagreement) are well-supported — each traced to measured evidence in at least one investigation.

- [x] **Extends** model with: Evidence quality stratification. The model currently treats all findings uniformly. This inventory reveals a hierarchy: 14 high-confidence claims (multi-source), 25 well-measured single-source claims, and 2 assumed claims. The replication failure caveat on the dilution curve means the most cited quantitative thresholds (≤4 behavioral budget, dilution starts at 5) are directional hypotheses, not established facts.

- [x] **Extends** model with: Explicit identification of the cluster's 3 strongest empirical findings: (1) bare-parity testing as behavioral validation method (measured, replicated), (2) knowledge-vs-constraint divergence (measured across 3 experiments), (3) 82% skill size reduction without knowledge-transfer regression (measured, deployed).

---

## Notes

- Full numbered inventory of all 67 claims in SYNTHESIS.md at `.orch/workspace/og-inv-task-evidence-inventory-12mar-028b/SYNTHESIS.md`
- The dilution curve's replication failure is the single most important caveat — it affects claims #15-#24 in the inventory
- The cluster spans 16 days (Feb 24 – Mar 11) and represents the most intensive investigation period in the project's history
- Future work: re-run dilution experiments with clean isolation to validate or invalidate threshold numbers
