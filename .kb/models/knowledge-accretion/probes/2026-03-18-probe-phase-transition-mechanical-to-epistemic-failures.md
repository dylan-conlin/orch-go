# Probe: Phase Transition — Mechanical Failures to Epistemic Failures

**Model:** knowledge-accretion
**Date:** 2026-03-18
**Status:** Complete

---

## Question

The knowledge-accretion model claims that measurement-improvement bias can make metrics look better without actual improvement (§4, Entropy Metrics, point 7: "Systems tracking their own health need to distinguish 'we got healthier' from 'we got better at measuring.'"). Does the current measurement infrastructure exhibit a more severe form of this — where ground truth mechanisms not only fail to detect false coherence but actively inflate confidence?

Three sub-questions:
1. **DIAGNOSTIC:** Where is the trust boundary? Which system outputs can be trusted vs. produce false confidence?
2. **STRUCTURAL:** What would Phase 2 (epistemic integrity) architecture look like vs Phase 1 (mechanical reliability)?
3. **MEASUREMENT:** How do you detect false coherence when the system itself produces the metrics?

---

## What I Tested

### Test 1: Ground Truth Inflation Effect

Measured the actual rework rate from events.jsonl and traced its effect through `GroundTruthAdjustedRate()`:

```bash
# Count reworks and completions from events.jsonl
cat ~/.orch/events.jsonl | python3 -c "..."

# Result:
# Completions: 817
# Reworks: 0
# Rework rate: 0.0%
```

Then traced the formula in `pkg/daemon/allocation.go:113-119`:
```go
func GroundTruthAdjustedRate(selfReported, reworkRate float64, hasReworkData bool) float64 {
    if !hasReworkData { return selfReported }
    groundTruthRate := 1.0 - reworkRate
    return (1-GroundTruthWeight)*selfReported + GroundTruthWeight*groundTruthRate
}
```

Key: `hasReworkData = sl.ReworkCount > 0 || sl.TotalCompletions >= MinSamplesForFullWeight (10)`.
Feature-impl has 330+ completions, so `hasReworkData=true` despite zero reworks.
With reworkRate=0.0: `adjustedRate = 0.7*0.757 + 0.3*1.0 = 0.830` — a +7.3pp inflation.

### Test 2: Merge Rate Tautology

Checked `orch orient` output and examined how merge rate is calculated in `pkg/orient/git_ground_truth.go`:

```bash
orch orient
# Output: "Merged: 596 (100%) | net lines: +380203"
```

The merge rate is 100% because all work commits directly to main — there's no PR workflow. The metric measures "did the agent commit" which is identical to "did the agent complete." The ground truth signal is tautological.

### Test 3: Trust Surface Mapping

Ran actual measurements across all monitoring subsystems:

```bash
orch kb orphans --stratified
# Orphan rate: 91.8% (1170/1275), positive-unlinked: 85.5%

orch kb audit decisions
# 43 of 79 accepted decisions (54.4%) have missing file references
```

Examined detector source code:
- `pkg/daemon/trigger_detectors_phase2.go` — model contradictions uses regex, no validation
- `pkg/daemon/detector_outcomes.go` — "useful rate" = completed/resolved, but "completed" is self-reported
- `pkg/kbmetrics/decision_audit.go` — checks file existence, not implementation correctness
- `pkg/findingdedup/findingdedup.go` — 0.20 Jaccard threshold not empirically validated

### Test 4: Orient Display as Self-Reinforcing Signal

```bash
orch orient
# "Completions: 134 | Abandonments: 0 | In-progress: 17"
# Zero abandonments with 134 completions
# All visible metrics are positive
```

---

## What I Observed

### Finding 1: Three Layers of False Confidence

The system produces three metrics that look like independent validation but are structurally dependent on the same self-reported signal:

| Layer | Metric | Appears to measure | Actually measures |
|-------|--------|-------------------|-------------------|
| **Self-reported** | 75.7% success rate | Agent work quality | "Agent said Phase: Complete" |
| **Ground truth adjusted** | 83.0% success rate | Quality verified by outcome data | Self-report inflated by +7.3pp because `reworkRate=0.0` treated as "perfect quality" |
| **Merge rate** | 100% | Work shipped to production | "Agent committed to main" (tautological — no PR workflow) |

The ground truth adjustment is the most insidious: it converts the absence of a negative signal (nobody uses `orch rework`) into a positive signal (everything must be fine). The formula treats rework_rate=0 as evidence rather than absence of evidence.

### Finding 2: The Trust Pyramid Is Inverted

Mapping actual trust level across all measurement subsystems:

| Subsystem | Claimed Signal | Actual Trust | Why |
|-----------|---------------|-------------|-----|
| `go build` (Gate 10) | "Code compiles" | **HIGH** | Execution-based, unfakeable |
| Git commit existence | "Work product exists" | **HIGH** | Verifiable fact |
| Test evidence gate | "Tests were run" | **MEDIUM** | Pattern-matches claims, can't verify execution |
| Orphan rate | "Knowledge is disconnected" | **MEDIUM** | Counts correctly, but natural baseline unknown |
| Learning store | "Skill success rates" | **LOW** | Circular: measures self with own events |
| Ground truth adjustment | "Outcome-verified quality" | **FALSE** | Inflates confidence from absent signal |
| Merge rate | "Work ships to production" | **FALSE** | Tautological: commit=completion in single-branch flow |
| Detector outcomes | "Detectors create useful work" | **LOW** | "Useful" = "not abandoned" = self-reported |
| Decision audit | "Decisions are implemented" | **LOW** | File existence ≠ implementation correctness |
| Model contradictions | "Models are internally consistent" | **LOW** | Regex-based, no validation of detection accuracy |

The system has 2 HIGH-trust signals, 2 MEDIUM-trust signals, and 6 LOW-to-FALSE signals. The dashboard prominently displays the LOW/FALSE signals (completion rate, merge rate, success rate) while the HIGH signals (build pass, commit existence) are invisible infrastructure.

### Finding 3: Phase 1 vs Phase 2 Failure Mode Taxonomy

**Phase 1 (mechanical):** System breaks visibly. Build fails. Agent crashes. Dashboard shows error. Feedback is immediate and self-correcting. The system is well-equipped for this — 14 verification gates, coaching plugins, daemon recovery.

**Phase 2 (epistemic):** System runs smoothly but produces false confidence. Evidence:
- 134 completions/24h at "100% merge rate" — but 91.8% investigation orphan rate underneath
- 75.7% self-reported success inflated to 83.0% by ground truth adjustment
- 43/79 decisions with missing references (54.4%) detected but not acted on
- Zero abandonments, zero reworks, zero failed completions — a suspiciously clean signal

**The key asymmetry:** Phase 1 failures trigger Phase 1 infrastructure (gates, alerts, recovery). Phase 2 failures are invisible TO the same infrastructure because the infrastructure measures Phase 1 properties (existence, compilation, syntax) not Phase 2 properties (correctness, impact, value).

### Finding 4: The Self-Referential Measurement Problem

The system exhibits three circular measurement patterns:

1. **Rework circle:** Rework events feed ground truth → ground truth adjusts allocation → allocation determines what spawns → spawns determine completion → completion determines "success" → 0 reworks so everything looks good → ground truth inflates further.

2. **Detector outcome circle:** Detector creates issue → agent works issue → agent self-reports completion → detector counts "completed" as "useful" → detector budget maintained → detector creates more issues.

3. **Orphan detection circle:** System detects orphan rate → spawns investigation to reduce orphans → investigation becomes orphan → orphan rate increases → system detects higher rate → spawns more investigations.

The only break in circularity is human intervention (`orch rework`, `orch abandon`, manual review). But the data shows this intervention channel is structurally unused (0 reworks, 0 abandonments in 24h window with 134 completions).

### Finding 5: False Coherence Pattern Taxonomy (Observed in This System)

| Pattern | Instance | Detection Possible? |
|---------|----------|-------------------|
| **Metric inflation** | Ground truth adjustment inflates 75.7%→83.0% from absent rework data | Yes — check if negative channel is populated |
| **Tautological validation** | 100% merge rate because single-branch workflow | Yes — check if metric measures something distinct from what it validates |
| **Existence ≠ correctness** | Decision audit checks if files exist, not if they implement the decision | Partial — requires semantic analysis |
| **Activity ≠ value** | 134 completions/24h but 91.8% orphan rate | Yes — compare activity metrics vs impact metrics |
| **Smooth operation as evidence** | 0 abandonments, 0 failures, 0 reworks treated as system health | Yes — check if absence of negative signal is structurally caused |
| **Self-referential calibration** | Detector "useful rate" based on self-reported completion | Hard — requires external ground truth |
| **Measurement maturity masking** | New metrics (orphan rate, decision audit) reveal problems that existed all along but were invisible | Awareness only — distinguishing "worse" from "newly visible" |
| **Infrastructure-outcome conflation** | Building ground truth system treated as achieving ground truth | Awareness only — distinguish "shipped feature" from "feature works" |

---

## Model Impact

- [x] **Extends** model with: A specific failure mode the model doesn't cover — **false ground truth**. The model discusses measurement-improvement bias (§4.7: metrics improving from better measurement, not better reality). But the ground truth adjustment is MORE severe: it actively inflates confidence from the absence of a negative signal. The model should add a new failure mode: "Ground truth mechanisms that use unused channels produce false positive signal, converting 'nobody uses this feedback channel' into 'everything is correct.'"

- [x] **Confirms** invariant: Gate deficit (§3). The knowledge-accretion model claims "every convention without a gate will eventually be violated" and lists 6 ungated knowledge transitions. This probe confirms the deficit extends to the measurement layer itself — ground truth, merge rate, and detector outcomes are all "ungated" in the sense that no external validation checks their output.

- [x] **Extends** model with: **Phase transition taxonomy**. The model describes accretion dynamics (things degrading from correct contributions) but doesn't distinguish between visible degradation (Phase 1, mechanical) and invisible degradation (Phase 2, epistemic). The system's monitoring infrastructure is optimized for Phase 1 — it detects broken builds, missing files, crashed agents. Phase 2 failures (false coherence, inflated metrics, tautological validation) are structurally invisible to the same infrastructure. This is a new organizing concept: the system must evolve its measurement surface from "does it exist/compile/run?" to "did it actually improve anything?"

- [x] **Extends** model with: **Architectural patterns for Phase 2**. Three patterns emerged:
  1. **External ground truth injection:** Metrics that can't be produced by the measured system. Git merge rate was the right idea but is tautological in single-branch flow. Needs: PR workflow, code review, or human quality sampling.
  2. **Negative signal channels must be populated to be meaningful:** A rework_rate of 0 should trigger a "channel health" warning, not be treated as "perfect quality." The system should monitor whether its feedback channels are actually being used.
  3. **Divergence detection between metric layers:** When self-reported success (75.7%), ground-truth-adjusted (83.0%), and merge rate (100%) all point upward but impact metrics (orphan rate 91.8%, stale decisions 54.4%) point downward, the system should flag the divergence rather than displaying only the positive metrics.

---

## Notes

### What Would Phase 2 Architecture Look Like?

**Principle:** Every metric must have an independent validation channel that isn't produced by the system being measured.

| Current Metric | Phase 2 Validation | Difficulty |
|---------------|-------------------|-----------|
| Completion rate | Human quality sampling (random N per week) | Low |
| Rework rate | Auto-detect when completed work gets re-addressed (follow-up issues on same topic) | Medium |
| Merge rate | PR workflow with review gate (architectural change) | High |
| Orphan rate | Track whether linked investigations actually influenced decisions | Medium |
| Detector outcomes | Track whether detector-spawned work changed behavior, not just closed | Hard |
| Decision audit | Semantic check (does code implement decision intent, not just reference it?) | Hard |

### The Rework Rate Problem

The ground truth weight of 0.3 was designed with the assumption that rework data would exist. The `hasReworkData` check (line 137) treats `TotalCompletions >= 10` as sufficient evidence of rework data — but it's evidence of completion volume, not rework volume. The fix: `hasReworkData` should require `ReworkCount > 0` OR an explicit "no rework needed" signal, not just high completion volume.

### Comparison to External Systems

In traditional software engineering, the equivalent of this problem is "code coverage theater" — high coverage percentage that tests implementation details rather than behavior. The fix was mutation testing (external validation). The orch-go equivalent would be "metric mutation" — deliberately introducing known-bad completions and checking whether the measurement system detects them.
