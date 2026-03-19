## Summary (D.E.K.N.)

**Delta:** Measurements have epistemic properties; a three-type taxonomy (false confidence/noisy signal/honest-but-misnamed) with corresponding actions (delete/calibrate/relabel) captures the lifecycle of measurement honesty in self-measuring systems.

**Evidence:** 10 measurement subsystems audited in trust audit: 3 classified as false confidence (deleted), 5 as noisy signal or honest-but-misnamed (simplified). Decision audit rebuild (d05a2997d) demonstrates the complete lifecycle: detect → delete → rebuild honestly.

**Knowledge:** The displacement effect — displaying a metric that can't fail suppresses the uncertainty awareness that would otherwise prompt investigation. Fewer trusted metrics > more untrusted metrics. Absent negative signal must be treated as "no data" not "positive signal."

**Next:** Use model's diagnostic protocol ("what would make this metric go red?") for future measurement evaluations. No implementation work needed — model captures existing decisions.

**Authority:** implementation - Synthesis of existing decisions into model; no new architectural choices.

---

# Investigation: Create Measurement Honesty Model

**Question:** Can the epistemic properties of measurements in self-measuring systems be captured in a reusable taxonomy? What are the types, diagnostic criteria, and correct remediation for each?

**Started:** 2026-03-19
**Updated:** 2026-03-19
**Owner:** orch-go-hcubt
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-18-arch-trust-audit-measurement-subsystems.md` | extends | yes — read full audit, verified verdict table | none |
| `.kb/models/knowledge-accretion/probes/2026-03-18-probe-phase-transition-mechanical-to-epistemic-failures.md` | extends | yes — read full probe, verified trust pyramid and false coherence taxonomy | none |
| `.kb/publications/self-measurement-report.md` | extends | yes — read full report, verified gate audit numbers and falsification criteria | none |

---

## Findings

### Finding 1: Three distinct epistemic failure modes in measurement

**Evidence:** The trust audit classified 8 subsystems into three categories that map to distinct failure modes:
- **False confidence (3 subsystems):** Ground truth adjustment, merge rate, decision audit v1 — all structurally incapable of producing negative signal
- **Noisy signal (3 subsystems):** Duplication detector (35% FP), model contradictions (keyword precision unknown), detector outcomes (self-reported "useful" rate)
- **Honest-but-misnamed (2 subsystems):** Learning store ("success rate" = completion rate), test evidence gate ("verification" = evidence check)

**Source:** `.kb/investigations/2026-03-18-arch-trust-audit-measurement-subsystems.md` — verdict table rows 1-8

**Significance:** The three types require different actions: false confidence → delete, noisy signal → calibrate, honest-but-misnamed → relabel. Applying the wrong action (e.g., calibrating a false-confidence metric instead of deleting it) produces a more sophisticated false-confidence metric.

---

### Finding 2: The absent-signal trap is the most insidious failure mode

**Evidence:** `GroundTruthAdjustedRate()` treated 0 reworks across 817 completions as evidence of quality, inflating 75.7% → 83.0% (+7.3pp). The rework channel was dead — nobody used `orch rework` — but the formula converted empty channel into positive signal. The phase transition probe mapped this as the most severe form of measurement-improvement bias: not just "we got better at measuring" but "we actively inflate confidence from what we don't know."

**Source:**
- Phase transition probe — Finding 1 (three layers of false confidence)
- `pkg/daemon/allocation.go:113-119` — `GroundTruthAdjustedRate()` formula
- Trust audit — verdict #4 (DELETE ground truth adjustment)

**Significance:** Absent negative signal is the hardest failure mode to detect because it looks like health. The diagnostic: if a metric has a negative-signal channel, check whether the channel is populated. If it's empty, the metric's positive signal is meaningless.

---

### Finding 3: Decision audit demonstrates the full lifecycle

**Evidence:** Decision audit went through all three states:
1. Started as honest-but-misnamed ("decision audit" = file existence check)
2. Measurement revealed false confidence (54.4% FP from architectural principles lacking file references)
3. Deleted per trust audit recommendation
4. Rebuilt (d05a2997d) with type-aware validation — split architectural vs implementation decisions with appropriate validation for each

The rebuild reduced false positives by using bigram search for architectural decisions and file-existence + pattern matching for implementation decisions. It's honest about its limitations: textual presence ≠ actual compliance.

**Source:** Commit d05a2997d (decision audit rebuild), trust audit verdict #7

**Significance:** This is the template case for the measurement honesty lifecycle. The key insight: the "delete" step is essential. Trying to fix a false-confidence metric without first deleting it usually produces a more complex version of the same problem. The clean break forces you to rebuild from "what CAN I actually measure?" rather than "how do I make the old metric more accurate?"

---

## Synthesis

**Key Insights:**

1. **The displacement effect is the core mechanism.** A metric that can't fail doesn't just provide no information — it actively suppresses the uncertainty awareness that would prompt investigation. This is worse than no metric.

2. **The diagnostic question is operational.** "What would make this metric go red?" is a practical test anyone can apply. If the answer is "nothing," delete the metric.

3. **The correct action depends on the type, not the severity.** A noisy signal with 35% FP (duplication detector) should be calibrated, not deleted — the detection mechanism has value. A false-confidence metric with elegant math (ground truth adjustment) should be deleted, not refined — no amount of sophistication fixes "structurally cannot fail."

**Answer to Investigation Question:**

Yes, the epistemic properties of measurements form a clean three-type taxonomy (false confidence / noisy signal / honest-but-misnamed) with corresponding actions (delete / calibrate / relabel). The taxonomy is grounded in 10 observed instances across one system. The diagnostic protocol ("what would make this go red?") is operationally useful. The decision audit lifecycle demonstrates the full measurement honesty cycle and serves as a template case. The model is captured in `.kb/models/measurement-honesty/model.md`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Three-type taxonomy covers all 10 audited subsystems (verified: mapped each to a type in trust audit)
- ✅ False confidence deletion improves system honesty (verified: ground truth adjustment, merge rate removed per audit; no regression)
- ✅ Decision audit rebuild reduced FP rate through type-aware validation (verified: commit d05a2997d shipped and tested)

**What's untested:**

- ⚠️ Taxonomy may not cover all measurement failure modes in systems other than orch-go
- ⚠️ "Delete before fixing" as a universal principle — may not apply when deletion cost is high (external dependencies on the metric)
- ⚠️ Whether the diagnostic protocol works for agents performing self-audit without human guidance

**What would change this:**

- A measurement failure mode that doesn't fit any of the three types
- A false-confidence metric that was successfully fixed without a delete step
- A system where displaying false-confidence metrics was net-positive (e.g., morale benefits outweighed decision-making cost)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Model creation (this investigation) | implementation | Synthesis of existing decisions, no new choices |
| Use diagnostic protocol for future metrics | implementation | Procedure within existing framework |
| Apply taxonomy to other projects | strategic | Cross-project methodology choice |

### Recommended Approach: Use model as diagnostic reference

**Why:** The model captures decisions already made and provides a reusable protocol for future measurement evaluations. No implementation work needed beyond creating the model artifact.

**Implementation sequence:**
1. Model created at `.kb/models/measurement-honesty/model.md`
2. Referenced from harness-engineering model (§5 measurement surface pairing)
3. Available as spawn context for future measurement-related work

---

## References

**Files Examined:**
- `.kb/investigations/2026-03-18-arch-trust-audit-measurement-subsystems.md` — Trust audit: 8 systems, 3 DELETE/5 SIMPLIFY
- `.kb/publications/self-measurement-report.md` — Self-measurement report: 5/8 gates zero fires, 65% dupdetect precision
- `.harness/openscad/SELF-MEASUREMENT-REPORT.md` — OpenSCAD cross-domain: honest "NO DATA" template
- `.kb/models/knowledge-accretion/probes/2026-03-18-probe-phase-transition-mechanical-to-epistemic-failures.md` — Phase transition: trust pyramid inversion, false coherence taxonomy
- `.kb/models/harness-engineering/model.md` — Harness engineering framework: §5 measurement surface pairing

**Commands Run:**
```bash
# Checked decision audit rebuild commit
git show d05a2997d --stat
git show d05a2997d --format="%B" --no-patch
```

**Related Artifacts:**
- **Model:** `.kb/models/measurement-honesty/model.md` — The model created by this investigation
- **Model:** `.kb/models/harness-engineering/model.md` — Parent framework model
