## Summary (D.E.K.N.)

**Delta:** Produced a comprehensive self-measurement report synthesizing 15,037 events, gate audit data, and falsification verdicts into an honest examination of what the enforcement infrastructure actually does (vs. what we assumed).

**Evidence:** events.jsonl analysis, `orch harness report`, `orch harness audit`, `orch hotspot`, git log, 5 prior probes/investigations on gate precision and cost.

**Knowledge:** 5/8 spawn gates show zero fires in 30d; duplication detector is 65% precise (assumed 0% FP); 40%+ investigations are orphans; 22/37 total gates have zero production data. Enforcement without measurement was indeed theological.

**Next:** Close. Report is complete at `.kb/publications/self-measurement-report.md`.

**Authority:** implementation - Writing/analysis task within existing patterns, no architectural changes.

---

# Investigation: Write Self Measurement Report

**Question:** Can the system honestly examine its own enforcement mechanisms and produce a credible self-measurement report?

**Started:** 2026-03-16
**Updated:** 2026-03-16
**Owner:** research agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** harness-engineering

---

## Findings

### Finding 1: Gate dormancy is the dominant state

**Evidence:** 5/8 spawn gates show zero fires in 30d. Hotspot gate evaluates 244 spawns at ~300ms each, blocks zero. No CRITICAL files exist (largest: 1,040 lines vs 1,500 threshold).

**Source:** `orch harness audit`, `.kb/models/harness-engineering/probes/2026-03-13-probe-hotspot-gate-cost-precision-measurement.md`

**Significance:** Gates that evaluate but never fire are ceremony, not enforcement. The cost is real (~73s total), the enforcement value is zero.

### Finding 2: Duplication detector precision is 65%, not 100%

**Evidence:** Retrospective audit of 259 match occurrences: 164 TP, 90 FP, 5 borderline. FP root causes: different semantics (47%), structural coincidence (40%), self-match bug (7%), opposite operations (7%).

**Source:** `.kb/models/harness-engineering/probes/2026-03-13-probe-duplication-detector-precision-measurement.md`

**Significance:** 35% false positive rate produces alert fatigue, training operator to ignore warnings — the gate calibration death spiral.

### Finding 3: Investigation orphan rate is 40%+

**Evidence:** 1,213 investigation files, 727 completed = 486 orphans (40.1%).

**Source:** `find .kb/investigations/ -name "*.md" | wc -l`, `grep -l "Status: Complete" ...`

**Significance:** The system generates investigative debt faster than it retires it.

---

## Synthesis

**Key Insights:**

1. **Measurement changes the conversation** - Before instrumentation, gates were assumed to work. After instrumentation, 5/8 are dormant, 1 detector has 35% FP, and most gates have zero production data.

2. **Dormancy is a distinct failure mode** - Not the same as false positives or false negatives. A gate that evaluates correctly but can't fire because its triggering condition doesn't exist is technically correct but operationally useless.

3. **Honest self-reporting is the value proposition** - The report's credibility comes from showing warts, not successes.

**Answer to Investigation Question:** Yes, the system can examine itself honestly. The resulting report surfaces uncomfortable findings (dormant gates, low precision, high orphan rates) because the measurement infrastructure makes them visible.

---

## Structured Uncertainty

**What's tested:**
- ✅ Gate fire rates computed from events.jsonl (verified: 15,037 events)
- ✅ Duplication detector precision measured (verified: 259 match occurrences classified)
- ✅ Investigation orphan count (verified: file count vs status grep)
- ✅ Fix:feat ratio (verified: git log analysis, 256 fix / 414 feat = 0.62)

**What's untested:**
- ⚠️ Whether gate presence prevents problems (no counterfactual)
- ⚠️ Soft harness effectiveness (no controlled experiments)
- ⚠️ Cross-system generalizability (single system)

**What would change this:**
- Post-gate accretion data showing velocity change (checkpoint Mar 24)
- Controlled A/B experiments removing soft harness components
- Deployment on a second system with same measurement infrastructure

---

## References

**Files Examined:**
- `~/.orch/events.jsonl` - 15,037 events for lifecycle analysis
- `.kb/models/harness-engineering/model.md` - Theoretical framework
- `.kb/models/harness-engineering/probes/2026-03-13-*.md` - Precision measurement probes
- `.kb/investigations/2026-03-11-inv-gate-census-signal-noise-classification.md` - Gate census
- `.kb/threads/2026-03-11-measurement-as-first-class-harness.md` - Measurement thread

**Commands Run:**
```bash
orch harness report
orch harness audit
orch hotspot
kb context 'harness measurement gate enforcement'
git log --oneline --since="6 weeks ago" | wc -l
```

---

## Investigation History

**2026-03-16:** Investigation started
- Initial question: Can the system honestly examine its own enforcement mechanisms?
- Context: Task to write self-measurement report for publication

**2026-03-16:** Investigation completed
- Status: Complete
- Key outcome: Report produced at `.kb/publications/self-measurement-report.md` with raw numbers, falsification verdicts, and honest assessment of measurement gaps.
