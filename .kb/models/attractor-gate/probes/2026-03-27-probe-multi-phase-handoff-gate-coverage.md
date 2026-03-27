# Probe: Multi-Phase Handoff Gate — Coverage Gap in Phase-Count Verification

**Date:** 2026-03-27
**Model:** attractor-gate
**Status:** Complete
**Issue:** orch-go-w8jys

## Question

Does the attractor-gate model predict the failure mode where multi-phase architect designs pass the handoff gate with only one issue? And does the fix — counting issues against detected phases — constitute a gate improvement or an attractor?

## What I Tested

Reproduced the exact bug: created a SYNTHESIS.md with 3 phases (Phase 1, 2, 3), provided a handoff comment with only 1 issue ID, and ran the handoff gate. Pre-fix: gate passed. Post-fix: gate correctly fails with diagnostic showing "3 phases detected, 1 issue found."

Test cases:
- `TestVerifyArchitectHandoff_MultiPhase_SingleIssue`: 3 phases, 1 issue → FAIL (correct)
- `TestVerifyArchitectHandoff_MultiPhase_AllIssuesPresent`: 2 phases, 2 issues → PASS (correct)
- `TestVerifyArchitectHandoff_MultiPhase_OptOut`: 3 phases, explicit opt-out → PASS (correct)
- `TestVerifyArchitectHandoff_SinglePhase_Unchanged`: no phases, 1 issue → PASS (backward compat)
- `TestDetectPhases`: 10 cases covering Phase/Layer/Step/Stage indicators, case insensitivity, dedup

Additionally verified: `go test ./pkg/verify/` all pass, `go build ./...` clean, `go vet` clean.

## What I Observed

1. **The failure mode was predicted by the model.** The original handoff gate was a pure gate with no attractor: it checked a boolean condition ("does at least one issue exist?") without structural guidance toward the correct number of issues. This matches Claim 1: gate-only configurations fail. The gate existed, architects acknowledged it, but multi-phase work still got lost.

2. **The fix adds structural information to the gate** — parsing SYNTHESIS.md for Phase/Layer/Step/Stage indicators and requiring issue count >= phase count. This is not an attractor (doesn't embed coordination at design time) but makes the gate structurally aware rather than purely boolean. It's a gate refinement, not a gate→attractor transition.

3. **The opt-out path is critical.** Without it, the gate would be too rigid for advisory-only designs or designs where phases need orchestrator review before issuing. This matches the "gate must be passable by the gated" decision.

4. **Comment evidence counting**: the change from `hasHandoffIssueEvidence` (bool) to `countHandoffIssueEvidence` (int) closes the gap where architects could report "Created implementation issues: id1" in the comment and the gate would pass for a 3-phase design.

## Model Impact

**Confirms Claim 1 (attractor + gate together):** The original handoff gate was gate-only (boolean check), and it failed silently for multi-phase designs. The fix adds structural awareness (parsing phase indicators from SYNTHESIS.md), making the gate phase-count-aware. This isn't a full attractor (doesn't embed coordination at design time), but it demonstrates that gates need structural information to be effective — a boolean gate is insufficient when the structure it guards has dimensionality (how many phases).

**Extends Hypothesis 4 (intervention effectiveness hierarchy):** This probe suggests a sub-hierarchy within gates: **structurally-informed gates > boolean gates > advisory gates**. A boolean gate that checks "at least one" is qualitatively weaker than a gate that counts against a structural property. The 87 non-functional constraints in the model were all boolean/advisory — the multi-phase gate was boolean too, and it failed the same way.

**No contradiction** with any existing claims.
