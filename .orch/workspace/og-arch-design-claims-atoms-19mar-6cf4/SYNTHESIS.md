# Session Synthesis

**Agent:** og-arch-design-claims-atoms-19mar-6cf4
**Issue:** orch-go-uq3s6
**Outcome:** success

---

## Plain-Language Summary

Designed the claims-as-atoms infrastructure that lets model claims drive inquiry through three consumption points. The design puts a `claims.yaml` file in each model directory as a machine-readable overlay (the model prose stays authoritative). Each claim has a confidence lifecycle (unconfirmed -> confirmed -> stale -> contested), domain tags for matching to recent work, and a falsification condition that the daemon uses to generate probe questions. Orient surfaces "edges" — cross-model tensions, stale claims in areas where we're actively working, and unconfirmed core claims — not a claim dump. The daemon generates at most 1 probe per cycle, only for claims in models that are actually being referenced. The completion pipeline updates claims only from probe results (not from feature work), preventing false-matching noise. Self-referential validation risk is handled by requiring evidence independence (can't confirm a claim using the same data the claim already cites), surfacing contradictions louder than confirmations, and auto-decaying confirmed claims to stale after 30 days.

Key discovery: the agent-trust-enforcement model already implements the target pattern (C1-C17 with status/evidence/falsification). This should be the reference implementation for bootstrapping the other 3 seed models.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key outcomes:
- Investigation file complete with 5 design forks navigated, schema defined, integration points specified
- 4-phase implementation sequence with clear boundaries
- Self-referential validation safeguards grounded in measurement-honesty invariants
- 3 blocking questions surfaced with recommendations

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-19-design-claims-atoms-infrastructure-model.md` — Full design investigation
- `.orch/workspace/og-arch-design-claims-atoms-19mar-6cf4/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-design-claims-atoms-19mar-6cf4/VERIFICATION_SPEC.yaml` — Verification spec

---

## Evidence (What Was Observed)

- Agent-trust-enforcement model has C1-C17 testable claims table — the exact pattern needed (status, evidence, falsification)
- Three existing claims systems serve different purposes: kbmetrics (bloat), kbgate (quality gating), publication ledger (external)
- Orient already scans model freshness and surfaces stale models — claims integration extends this
- Measurement-honesty invariant #5 (self-validating metrics) directly applies to circular probe validation risk
- Displacement thread warns: wrong knowledge in skill docs produced phantom constraints — same risk applies to wrong claims

---

## Architectural Choices

### Claims.yaml as overlay vs source of truth
- **What I chose:** Overlay — model.md prose remains authoritative, claims.yaml is a machine-readable index
- **What I rejected:** claims.yaml as source of truth (would require restructuring all 39 models)
- **Why:** Bootstrap constraint — can't require prose changes to existing models. Aligns with evidence hierarchy principle (code is truth, artifacts are hypotheses)
- **Risk accepted:** Divergence between prose and claims.yaml (mitigated by model_md_ref field + sync check)

### Probe-only claim updates vs automatic matching for all completions
- **What I chose:** Only probe completions update claims.yaml
- **What I rejected:** Automatic keyword matching for all completions
- **Why:** Automatic matching has high false positive rate. A feature agent touching gates area doesn't mean it tested gate claims.
- **Risk accepted:** Non-probe discoveries that contradict claims won't auto-update (mitigated by discovered work → beads issue flow)

### Contradiction asymmetry in orient output
- **What I chose:** Contradictions always surfaced, confirmations never surfaced
- **What I rejected:** Showing both confirmations and contradictions
- **Why:** Confirmation bias is the core self-referential risk. Surfacing confirmations creates a "feeling of progress" that suppresses questioning — the same displacement effect measurement-honesty documents.
- **Risk accepted:** Orchestrator may not know a claim was recently confirmed (mitigated by claims.yaml being readable directly)

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Claims.yaml cannot be source of truth without breaking 39 models — overlay is the only viable bootstrap path
- Self-referential validation is structurally equivalent to measurement-honesty's absent-signal trap — same mitigation patterns apply
- The agent-trust-enforcement model is already the closest implementation of the target pattern

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Implementation Issues

4 implementation phases, each a separate spawn:

1. **Schema + Bootstrap:** Define Go types in `pkg/claims/`, YAML parser, bootstrap 4 seed models
2. **Orient Integration:** Add `collectClaimEdges()` to orient_cmd.go
3. **Daemon Probe Generation:** Add `claimProbeGeneration` periodic task
4. **Completion Pipeline:** Add claim update logic to probe completion path

### MIGRATION_STATUS

```
MIGRATION_STATUS:
  designed: claims.yaml schema, orient integration, daemon probe generation, completion pipeline updates, self-referential validation safeguards
  implemented: none
  deployed: none
  remaining: all 4 implementation phases
```

---

## Unexplored Questions

- Whether domain_tags can reliably match claims to recent spawn activity (depends on tag quality and keyword overlap)
- Whether 30-day staleness threshold is correct — might need calibration after observing real probe generation rates
- How to handle model splits (when a model is decomposed, claims need to be redistributed)
- Whether the 39-model expansion should be gradual (add claims.yaml as models get updated) or batched

---

## Friction

No friction — smooth session. Background agents provided infrastructure context while I synthesized from direct reading.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-claims-atoms-19mar-6cf4/`
**Investigation:** `.kb/investigations/2026-03-19-design-claims-atoms-infrastructure-model.md`
**Beads:** `bd show orch-go-uq3s6`
