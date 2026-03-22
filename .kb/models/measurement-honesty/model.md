# Model: Measurement Honesty

**Domain:** Epistemic Properties of Self-Measurement Systems
**Last Updated:** 2026-03-21
**Validation Status:** WORKING HYPOTHESIS — derived from one system (orch-go) over 3 months of measurement infrastructure development. The taxonomy is grounded in 10 subsystems audited, 3 deleted, 5 simplified. Not yet tested in a second system.
**Synthesized From:**
- `.kb/investigations/2026-03-18-arch-trust-audit-measurement-subsystems.md` — 8-system trust audit: 3 DELETE, 5 SIMPLIFY, 0 GROUND
- `.kb/publications/self-measurement-report.md` — First self-measurement report: 5/8 gates zero fires, 65% dupdetect precision, "enforcement without measurement is theological"
- `.harness/openscad/SELF-MEASUREMENT-REPORT.md` — OpenSCAD cross-domain report: all 5 falsification criteria returned NO DATA
- `.kb/models/knowledge-accretion/probes/2026-03-18-probe-phase-transition-mechanical-to-epistemic-failures.md` — Phase transition probe: 3 layers of false confidence, trust pyramid is inverted
- Commit `d05a2997d` — Decision audit rebuild: deleted at 54% FP, rebuilt with type-aware validation (template case for false-confidence → delete → rebuild cycle)

---

## Summary (30 seconds)

Measurements have epistemic properties that determine whether they help or harm decision-making. A measurement structurally incapable of producing negative signal is worse than no measurement — it creates false confidence. The taxonomy: **false confidence** (measurement that cannot fail — delete it), **noisy signal** (measurement that produces signal with unknown precision — calibrate it), **honest-but-misnamed** (measurement that works but is labeled as something it doesn't measure — relabel it). The governing principle: fewer trusted metrics > more untrusted metrics. This model provides a diagnostic protocol for evaluating any measurement: ask "what would make this metric go red?" — if nothing can, the metric is producing false confidence and should be deleted rather than displayed.

---

## Core Mechanism

### The False Confidence Problem

A metric that always shows green is not evidence of health. It may be structurally incapable of showing red. The danger: humans and autonomous agents treat displayed metrics as evidence, adjusting confidence and decisions accordingly. A false-confidence metric doesn't just fail to inform — it actively misinforms by displacing the uncertainty it should represent with a green signal.

**The displacement effect:** When a dashboard shows "100% merge rate" and "83% quality score," the operator treats the system as measured. The presence of metrics suppresses the question "is this actually working?" — the very question the metrics were supposed to answer. Removing the metric would be better: at least then the operator knows they're flying blind.

**Evidence (orch-go, March 2026):** The orient display showed three mutually-reinforcing green metrics — 75.7% success rate, 83.0% ground-truth-adjusted rate, 100% merge rate. Underneath: 91.8% investigation orphan rate, 54.4% decision audit false positives, 0 reworks across 817 completions. The positive metrics created a coherent picture of health. The negative metrics, less prominently displayed, told a different story. The system was not lying — each metric measured what it claimed. But the ensemble produced false confidence because the visible metrics were structurally incapable of detecting the problems the less-visible metrics revealed.

### The Three-Type Taxonomy

Every measurement falls into one of three epistemic categories. The category determines the correct remediation:

| Type | Epistemic Property | What It Looks Like | Correct Action | Why This Action |
|------|-------------------|-------------------|----------------|-----------------|
| **False confidence** | Structurally cannot produce negative signal | Always green, feels reassuring, creates blind spots | **Delete** | Displaying a metric that can't fail is worse than no metric — it suppresses the uncertainty awareness that would otherwise prompt investigation |
| **Noisy signal** | Produces signal but precision is unknown or poor | Fires sometimes, but you can't tell true from false positives | **Calibrate** | The detection mechanism has value; it just needs precision measurement so operators can weight it correctly |
| **Honest-but-misnamed** | Correctly measures X but is labeled as measuring Y | Works fine, but decisions based on it assume it measures something different | **Relabel** | The metric is sound; the name creates confusion about what it tells you |

### How to Classify a Measurement

The diagnostic question: **"What would make this metric go red?"**

1. **If nothing can** → False confidence. Delete.
   - Example: Merge rate in single-branch workflow. All commits go to main. Merge rate is always 100%. Nothing can make it not-100%. It's measuring "did the agent commit" which is the same as "did the agent complete."
   - Example: Skill inference "success rate." The pipeline has exhaustive fallback (label → title → description → type), and the type field is present in 100% of issues (4,036/4,036). The metric structurally cannot fail — it measures "did a skill get assigned," not "was the right skill assigned."

2. **If something can, but you don't know the false positive rate** → Noisy signal. Calibrate.
   - Example: Duplication detector. It fires on ~89 events. But before measurement, assumed 0% FP rate. Actual: 35%. The signal exists but operators can't correctly weight it without knowing precision.

3. **If it accurately measures something, but the label implies it measures something else** → Honest-but-misnamed. Relabel.
   - Example: "Success rate" that actually measures completion rate (self-reported Phase: Complete). The measurement is accurate — agents do report completion at that rate. But "success" implies quality assessment, while the metric only measures self-reported completion.

### The Absent-Signal Trap

The most insidious failure mode: treating the absence of a negative signal as a positive signal.

**Mechanism:** System has a feedback channel (rework, abandonment, failure). Nobody uses the channel. Metric formula treats empty channel as "everything is fine" rather than "channel is dead."

**Evidence:** `GroundTruthAdjustedRate()` in `pkg/daemon/allocation.go` blended self-reported success (75.7%) with rework rate at 70/30 weight. With 0 reworks across 817 completions, the formula computed `0.7 × 0.757 + 0.3 × 1.0 = 0.830` — a +7.3 percentage point inflation. The absence of rework data was converted into evidence of quality.

**The fix principle:** Empty negative-signal channels should trigger "channel health" warnings, not contribute positive signal. `rework_rate = 0` should mean "we don't know" (insufficient data), not "everything is perfect."

---

## Instances (Observed in orch-go)

### False Confidence (Deleted)

| Metric | What It Claimed | Why It's False Confidence | Resolution |
|--------|----------------|--------------------------|------------|
| **Ground truth adjustment** | "Outcome-verified quality rate" | Inflates self-report by +7.3pp from absent rework data. Zero reworks = dead channel, not perfect quality. | Deleted — `GroundTruthAdjustedRate()` removed, replaced with direct blended rate |
| **Merge rate** | "Work ships to production" | Tautological in single-branch flow: commit = complete = merge. Cannot distinguish good from bad work. | Deleted from orient display. `NetLinesAdded` kept (git-grounded, actually informative) |
| **Decision audit (v1)** | "Decisions are implemented" | Checked file existence, not implementation correctness. 54.4% of decisions flagged as "missing" because they're architectural principles, not file-specific directives. | Deleted, then rebuilt with type-aware validation — split into architectural (check gate/hook/test reflection) vs implementation (check file existence + pattern matching) |

### Noisy Signal (Calibrated)

| Metric | What It Claimed | The Noise Problem | Calibration Applied |
|--------|----------------|-------------------|---------------------|
| **Duplication detector** | "These functions are duplicates" | 35% false positive rate (assumed 0% before measurement). 1 in 3 warnings is noise. Risk: operator learns to ignore all warnings. | Allowlist for known FP categories, precision tracked per run |
| **Model contradictions detector** | "Probe contradicts model" | Keyword matching ("contradict", "refute") with unknown precision. "Does not contradict" triggers false positive. | Priority lowered P2→P3, outcome tracking wired |
| **Detector outcomes** | "Detector creates useful work" | "Useful" = "completed" = self-reported. Budget adjustment code existed but wasn't wired. | Budget adjustment wired into trigger scan. Rename "useful rate" → "resolution rate" |

### Honest-but-Misnamed (Relabeled)

| Metric | Old Name | What It Actually Measures | New Name |
|--------|----------|--------------------------|----------|
| **Skill success rates** | "Success rate" | Self-reported completion per skill | "Completion rate" (display only — allocation code still uses `SuccessRate` field names and ±20% quality-proxy modulation, an active instance of §3 failure mode) |
| **Test evidence gate** | "Verification" | Pattern-matches agent claims for test output format | "Evidence check" (necessary-but-not-sufficient signal) |
| **Orphan rate** | "91.8% orphan rate" (displayed as problem signal) | Count of unlinked investigations — but 90%+ is the natural baseline, not a problem | Removed absolute rate from orient. Replaced with "new orphans per session" (actionable delta) |

---

## The Decision Audit as Template Case

The decision audit lifecycle is the canonical example of the full measurement honesty cycle. It went through all three states and demonstrates the correct response to each:

**Phase 1 — Honest-but-misnamed:** The original decision audit checked whether `.kb/decisions/` files referenced existing code paths. This is a valid operation — file existence IS checkable. But it was named "decision audit" implying it checks decision implementation, which requires semantic understanding, not file existence.

**Phase 2 — False confidence discovered:** Measurement revealed 54.4% of decisions flagged as "missing references." Investigation showed most were architectural principles ("use pattern X") that never referenced specific files. The metric was structurally generating false positives for an entire class of decisions.

**Phase 3 — Deleted:** The trust audit (March 18) recommended deletion. The metric created work (investigating "missing references") without catching real problems (decisions that were actually ignored). The automated decision audit was removed.

**Phase 4 — Rebuilt honestly:** The decision audit was rebuilt (commit `d05a2997d`) with type-aware validation:
- **Architectural principles** → checked for reflection in gates, hooks, tests, CLAUDE.md (bigram search, not file existence)
- **Implementation decisions** → file-existence checks on referenced paths and patterns

The rebuild addressed false confidence by splitting the measurement into two measurements with different validation appropriate to each type. The new version is honest about what it can and cannot check.

**The pattern:** Detect false confidence → delete the dishonest metric → understand what's actually measurable → rebuild with honest scope. Skipping the delete step (trying to "fix" a false-confidence metric) usually produces a more complex false-confidence metric.

---

## Critical Invariants

1. **A metric that cannot go red provides no information.** Displaying it is negative value — it suppresses uncertainty awareness. Deletion is the correct response.

2. **Absent negative signal ≠ positive signal.** Empty feedback channels (zero reworks, zero failures, zero complaints) indicate unused channels, not perfection. Formulas must distinguish "no data" from "positive data."

3. **Trust level must be visible alongside the metric.** A MEDIUM-trust metric displayed identically to a HIGH-trust metric inflates the ensemble's apparent trustworthiness. The trust pyramid in orch-go was inverted: 6 LOW/FALSE signals displayed prominently, 2 HIGH signals invisible.

4. **Precision must be measured before the metric is operational.** The duplication detector operated for weeks with an assumed 0% FP rate. Actual: 35%. Unmeasured precision means the metric's signal-to-noise ratio is unknown — and unknown SNR means the operator can't correctly weight the signal.

5. **Metrics that validate themselves are circular.** Self-reported completion feeding back into allocation scoring feeding back into what gets spawned feeding back into what gets completed. The only break in circularity is external ground truth (human review, external system validation).

6. **Delete before fixing.** Deleting a false-confidence metric is always safe. "Fixing" it risks producing a more sophisticated false-confidence metric. The correct sequence: delete → understand what's measurable → rebuild with honest scope (decision audit template case).

---

## Why This Fails

### 1. Deletion Is Politically Harder Than Fixing

**What happens:** A metric exists on a dashboard. Removing it feels like regression — "we had X metrics, now we have fewer." Stakeholders may interpret deletion as losing capability rather than gaining honesty.

**Evidence:** The trust audit recommended deleting 3 of 8 subsystems. Each deletion required explaining why fewer metrics is better than more metrics. The OpenSCAD self-measurement report demonstrated the inverse: honest "NO DATA" across 5 criteria was more credible than fabricated numbers.

**Mitigation:** Frame deletion as a quality improvement: "We measured 10 things. 3 were false confidence. We now measure 7 things honestly." The OpenSCAD report's approach — "every falsification criterion returns NO DATA" — is the template for honest reporting of unmeasured state.

### 2. Noisy Signals Become False Confidence Through Familiarity

**What happens:** A metric fires with unknown precision. Over time, operators learn to expect it and stop questioning its accuracy. The noisy signal becomes an unquestioned signal — functionally equivalent to false confidence.

**Evidence:** The duplication detector operated for weeks at 35% FP. Operators developed a bypass reflex. This is the gate calibration death spiral: too-noisy signal → bypass reflex → gate ignored → no enforcement.

**Mitigation:** Precision measurement on a schedule. If precision cannot be measured, the metric should display its unknown status: "Precision: unmeasured" alongside every reading.

### 3. Relabeling Is Insufficient When the Name Is Load-Bearing

**What happens:** A metric is renamed to honestly describe what it measures. But downstream systems (daemon allocation, orient display, decision logic) use the old semantics. Renaming the display doesn't fix the formula.

**Evidence:** Renaming "success rate" to "completion rate" is correct labeling. But the ±20% allocation modulation still treats completion rate as a quality proxy. The formula doesn't change because the label changes.

**Mitigation:** Relabeling must propagate to formulas, not just displays. If the formula assumes the metric measures quality, it must be adjusted to treat it as measuring throughput.

### 4. Rebuilds Inherit the Original Assumptions

**What happens:** A false-confidence metric is deleted and rebuilt. The rebuild uses different mechanics but the same unstated assumption: that the measurable proxy correlates with the target concept.

**Evidence:** Decision audit v1 assumed file-existence → implementation. Decision audit v2 uses bigram search for architectural decisions. The bigram approach is more precise, but still assumes textual presence → actual compliance. This is a better proxy, not ground truth.

**Mitigation:** Rebuilt metrics should explicitly state what they can and cannot detect. Decision audit v2 should be labeled "structural reflection check" not "decision compliance audit."

### 5. Two-Gap Independence: Measuring an Action Without Its Consequences

**What happens:** A system measures whether an enforcement action fired (deny count) but not what happened afterward (displacement). Closing the action-counting gap creates false confidence: "we blocked N times, therefore governance works" — while the downstream architectural cost remains invisible.

**Evidence:** Governance hooks have zero observability (F1, hook audit 2026-03-12). Even with perfect deny counting, a deny count without displacement tracking is structurally equivalent to the absent-signal trap (invariant #2). The system cannot distinguish "hooks prevented bad code" from "hooks displaced code to wrong locations." Agent scs-sp-8dm was blocked from `pkg/spawn/gates/concurrency.go` and placed the logic in `pkg/orch/spawn_preflight.go` instead — the enforcement succeeded, the architecture was violated.

**Mitigation:** Enforcement metrics must be paired: action-fired + consequence-measured. A deny count should not be displayed without a companion displacement signal, even if the displacement signal is labeled "unknown" or "unmeasured." This generalizes: any metric measuring a system action (gate fired, alert sent, agent blocked) creates false confidence if the downstream effect is invisible.

### 6. Structural Undetectability Ceiling

**What happens:** Some consequences of an enforcement action are structurally undetectable regardless of instrumentation quality. When an agent is denied and chooses an invisible workaround (completely different approach), no heuristic or commit audit can distinguish "displaced code" from "legitimate alternative." The detector's recall has an inherent ceiling < 100%.

**Evidence:** Governance hook denials have four outcomes: compliance (detectable via self-report), displacement (detectable via commit heuristics), abandonment (partially detectable), invisible workaround (structurally undetectable). Any displacement metric is a floor estimate, never a ceiling.

**Mitigation:** Label such metrics as floor estimates: "at least N displacements detected" rather than "N displacements occurred." Never display as "0 displacements = no displacement." This pattern applies wherever the measured concept has a component invisible to the measurement.

---

## Constraints

### Why Delete Rather Than Disable?

**Constraint:** A disabled metric still exists in code. Future developers or agents may re-enable it without understanding why it was disabled. Disabled metrics accumulate as technical debt and create confusion about what's actually measured.

**This enables:** Clear codebase — what exists is what runs
**This constrains:** No "just in case" metrics — either measure honestly or don't measure

### Why Can't We Ground Everything in External Truth?

**Constraint:** External ground truth (human sampling, PR review, external validation) requires architectural changes and ongoing human effort. In a single-developer system with autonomous agents, the human bandwidth for quality sampling is the bottleneck.

**This enables:** Honest assessment of what CAN be measured
**This constrains:** Many desirable metrics (code quality, decision correctness, investigation value) remain structurally unmeasurable without infrastructure changes

### Why Precision Before Operation?

**Constraint:** Operators naturally trust displayed metrics. A metric with unknown precision trains operators on a signal-to-noise ratio they can't perceive. By the time precision is measured, behavioral patterns (bypass reflex, false confidence) may already be established.

**This enables:** Operators can correctly weight signals from the start
**This constrains:** Cannot ship a new metric without a precision baseline — even if the baseline is "we measured 20 samples and found 15% FP"

---

## Relationship to Other Models

```
                    Measurement Honesty
                      (this model)
                          |
            +-------------+-------------+
            |             |             |
    Harness Engineering   |    Knowledge Accretion
    (§5: measurement      |    (§4: measurement-
     surface pairing —    |     improvement bias,
     enforcement needs    |     false coherence)
     measurement)         |
                          |
                  Completion Verification
                  (gate type taxonomy:
                   execution/evidence/judgment
                   maps to HIGH/MEDIUM/LOW trust)
```

- **Harness Engineering** (invariant #7): "Enforcement without measurement is theological; enforcement with measurement is empirical." This model extends that insight: measurement without epistemic classification is also theological — you have numbers but don't know what they mean.
- **Knowledge Accretion** (§4, point 7): "Systems tracking their own health need to distinguish 'we got healthier' from 'we got better at measuring.'" This model adds the third, more severe case: "we got worse at knowing what we don't know" (false confidence from absent negative signals).
- **Completion Verification** (gate types): Execution gates → HIGH trust (binary, unfakeable). Evidence gates → MEDIUM trust (pattern-matches claims). Judgment gates → variable trust (depends on judge). The gate type taxonomy maps directly to the trust hierarchy this model describes.

---

## Evolution

**2026-03-11:** Measurement audit revealed enforcement without measurement. 52% field gaps, 0 gate events, 111s invisible dupdetect cost, 4.7% accretion coverage. The harness engineering model added invariant #7: enforcement needs measurement. But the classification was binary: measured vs unmeasured.

**2026-03-13:** Duplication detector precision measured at 65% (35% FP). First instance of a metric transitioning from "unknown precision" to "known noisy signal." Revealed the assumed-0%-FP pattern.

**2026-03-16:** Self-measurement report published. "The credibility of any enforcement framework rests not on the gates that fire but on the willingness to report the ones that don't." First systematic honest reporting of measurement gaps.

**2026-03-18:** Phase transition probe mapped 10 measurement subsystems into trust levels (2 HIGH, 2 MEDIUM, 6 LOW/FALSE). Identified three layers of false confidence (self-reported → ground-truth adjusted → merge rate). The trust pyramid was inverted: LOW/FALSE signals displayed prominently, HIGH signals invisible.

**2026-03-18:** Trust audit applied the taxonomy: 3 DELETE (false confidence), 5 SIMPLIFY (noisy signal + honest-but-misnamed). Governing principle crystallized: "A metric you can't trust is worse than no metric."

**2026-03-19:** Decision audit rebuilt with type-aware validation (commit d05a2997d). Completed the template case: false confidence detected → deleted → rebuilt honestly. This model created to capture the taxonomy and diagnostic protocol.

**2026-03-19:** Governance displacement probe revealed two new failure modes. (1) **Two-gap independence:** measuring an action (hook deny count) without its consequence (code displacement) creates a new instance of false confidence — "we blocked N times" doesn't mean governance works. (2) **Structural undetectability ceiling:** some displacement outcomes (invisible workarounds) cannot be detected regardless of instrumentation, meaning displacement metrics are inherently floor estimates. Added §5 and §6 to "Why This Fails."

**2026-03-21:** Issue quality baseline probe confirmed invariants #1 and #2 in daemon skill inference. The spawn.skill_inferred pipeline (682 unique inferences, 641 daemon spawns) has a 100% "success rate" — but this is structurally guaranteed by exhaustive fallback chains (label → title → description → type), not evidence of correct routing. 69% of inferences fall through to the coarsest signal (type-based), and zero mechanism exists to measure whether inferred skills were correct. Added new false-confidence example (exhaustive fallback producing guaranteed positive results).

---

## References

**Investigations:**
- `.kb/investigations/2026-03-18-arch-trust-audit-measurement-subsystems.md` — 8-system trust audit, verdict table
- `.kb/investigations/2026-03-19-inv-create-measurement-honesty-model.md` — This model's creation investigation

**Probes:**
- `.kb/models/knowledge-accretion/probes/2026-03-18-probe-phase-transition-mechanical-to-epistemic-failures.md` — Trust pyramid, false coherence taxonomy, Phase 1 vs Phase 2 failure modes
- 2026-03-19: Knowledge Decay Verification — All 6 concrete claims confirmed. Found active instance of §3 failure mode: `SuccessRate` field names in allocation code still use old semantics despite display relabeling.
- 2026-03-19: Governance Displacement Measurement Design — Confirms invariants #2 and #4. Extends model with two-gap independence (action-fired vs consequence-measured), structural undetectability ceiling, and latency-honesty tradeoff in hook instrumentation. Proposes 3-phase measurement design for governance hooks.
- 2026-03-21: Issue Quality Baseline — Confirms invariants #1 and #2. Skill inference "success rate" is false confidence: exhaustive fallback guarantees 100% via type field (always present), while routing accuracy is unmeasured. 69% type-based fallback, 12% label, 12% title, 5% description. Corpus: 4,036 issues, 100% have type, 66% have description, 15% have labels.

**Publications:**
- `.kb/publications/self-measurement-report.md` — Orch-go self-measurement report (methodology and honest reporting template)
- `.harness/openscad/SELF-MEASUREMENT-REPORT.md` — OpenSCAD cross-domain report (honest "NO DATA" as template)

**Threads:**
- `.kb/threads/2026-03-11-measurement-as-first-class-harness.md` — Enforcement without measurement is theological

**Related Models:**
- `.kb/models/harness-engineering/model.md` — §5 (measurement surface pairing), invariant #7
- `.kb/models/knowledge-accretion/model.md` — §4 (measurement-improvement bias)
- `.kb/models/completion-verification/model.md` — Gate type taxonomy (execution/evidence/judgment → trust levels)

**Primary Evidence (Verify These):**
- `pkg/daemon/allocation.go` — `GroundTruthAdjustedRate()` (deleted per trust audit) and `BlendedSuccessRate()` (replacement)
- `pkg/kbmetrics/decision_audit.go` — Rebuilt with type-aware validation (commit d05a2997d)
- `pkg/orient/git_ground_truth.go` — Merge rate calculation (tautological in single-branch)
- `cmd/orch/orient_cmd.go` — Orient display (where metrics are shown to operator)
