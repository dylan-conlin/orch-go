# Probe: Orchestrator Skill Model Construction — Synthesis from 3-Worker Parallel Investigation

**Model:** orchestrator-skill
**Date:** 2026-03-12
**Status:** Complete

---

## Question

Can the skill-specific content from the orchestrator-session-lifecycle model (44% of its 642 lines) be separated into a standalone model that properly qualifies evidence quality, incorporates all 5 identified gaps, and synthesizes the 13 failure modes — while maintaining correct boundary definitions with both session-lifecycle and behavioral-grammars models?

---

## What I Tested

Read and synthesized outputs from 3 parallel investigation workers:

1. **Evidence Inventory** (og-inv-task-evidence-inventory-12mar-028b): 67 claims extracted from 7 investigations and 4 probes, classified by evidence type (MEASURED/ANALYTICAL/ASSUMED) and replication status. 14 high-confidence multi-source claims identified.

2. **Contradiction Analysis** (og-inv-task-contradiction-analysis-12mar-72b0): 4 direct contradictions (2 blocking), 5 tension points, 2 temporal contradictions, 3 recommendation conflicts. Key finding: dilution curve thresholds cited as established in 4 downstream artifacts despite replication failure caveat.

3. **Gap Analysis** (og-inv-task-existing-model-12mar-241d): 44% of session-lifecycle model is skill-specific. 5 probes identified for migration. 5 gaps: failure mode #13, stance content type, measurement validity, CLAUDECODE blocker, concrete fixes.

Also read:
- `.kb/models/orchestrator-session-lifecycle/model.md` — structure reference, failure modes #1-#13
- `.kb/global/models/behavioral-grammars/model.md` — stance content type, measurement validity framework, general principles

---

## What I Observed

### Model Construction Decisions

1. **Constraint budget qualification**: Presented as "HYPOTHESIZED at ~2-4" with explicit replication failure caveat and downstream propagation failure warning. No downstream artifact should cite this number as established.

2. **Design tension reclassification**: The contradiction analysis found "accretion resolved" overstated (24% regrowth in 7 days). Reclassified from "resolved" to "managed by budget" — T-M2 in the managed category, not resolved. Final categorization: 3 fundamental, 4 managed (including the reclassified accretion tension), 2 live. This produces 9 tensions total as required.

3. **Stance as third content type**: Added alongside knowledge/behavioral with distinct transfer mechanism (epistemic orientation vs information addition vs action suppression). Referenced behavioral-grammars and skill-content-transfer models rather than duplicating evidence.

4. **CLAUDECODE named as specific blocker**: Called out in Structured Uncertainty as "the single largest technical blocker" — not buried in generic "testing pending" language.

5. **13 failure modes synthesized into 5 layers**: The original 12 mapped to 4 layers (prompt/infrastructure/structural/temporal). Mode #13 (architect design bypass) added as a 5th cross-cutting "knowledge-feedback" layer since it doesn't fit cleanly into the existing 4.

6. **5 probes copied (not moved)**: Session-lifecycle model still references them. Migration is copy-on-read — both models now have the probes, but new probes for skill-specific findings should target the orchestrator-skill model.

### Gap Closure Verification

| Gap | Status |
|-----|--------|
| Gap 1: Failure Mode #13 | ✅ Added as Mode #13 in layer E (Knowledge-Feedback) |
| Gap 2: Stance content type | ✅ Added as section "Three Content Types" with knowledge/stance/behavioral taxonomy |
| Gap 3: Measurement validity | ✅ Structured Uncertainty section with validated/directional/assumed/unknown tiers |
| Gap 4: CLAUDECODE blocker | ✅ Named in Structured Uncertainty and Design Tension T-L1 |
| Gap 5: Concrete fixes | ✅ "Actionable Fix Designs" section with 4 designs from probes |

---

## Model Impact

- [x] **Extends** orchestrator-session-lifecycle model: Skill-specific content now has a dedicated model. Session-lifecycle can slim its skill sections to pointers referencing this model.

- [x] **Confirms** behavioral-grammars model: The orchestrator skill is a specific application of general probabilistic constraint principles. The three content types (knowledge/stance/behavioral) confirmed as distinct transfer mechanisms. General theory stays in behavioral-grammars; specific application here.

- [x] **Extends** with: Evidence quality stratification. Claims weighted by multi-source measured > single-source measured > multi-source analytical > single-source analytical > assumed. The Structured Uncertainty section makes this explicit — no prior model had this.

---

## Notes

- Model is ~320 lines. Synthesizes understanding rather than dumping all 67 claims.
- The boundary rule (session-lifecycle = "what sessions do", orchestrator-skill = "how the skill shapes sessions") resolves the 44% overlap.
- The probe-to-downstream propagation failure (caveats not flowing to citing artifacts) is a systemic risk flagged in Claim 4 — this is a meta-finding about the knowledge system itself.
