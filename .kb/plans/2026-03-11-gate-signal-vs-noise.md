## Summary (D.E.K.N.)

**Delta:** Answer empirically whether each gate in the system is filtering signal (catching real problems) or generating noise (forcing routine bypasses that normalize override behavior).

**Evidence:** `orch stats` data from 201 spawns: 69.7% spawn gate bypass rate, self_review false positive repeated 10+ times, 66 hotspot bypasses with no reasons, only 6.1% pass verification first try. Measurement gap audit (Mar 11) identified survivorship bias in event architecture.

**Knowledge:** A gate that's routinely bypassed is worse than no gate — it normalizes overriding. The enforcement+measurement pairing principle means we can't evaluate gates without data on their decisions and outcomes. We now have the instrumentation (Phases 1-3 shipped today); this plan uses it.

**Next:** Phase 1 — classify every gate by its current signal-to-noise ratio using existing stats data.

---

# Plan: Gate Signal vs Noise

**Date:** 2026-03-11
**Status:** Active
**Owner:** Dylan + orchestrator

**Extracted-From:** Measurement gap audit investigation, `orch stats` walkthrough, "measurement as first-class harness layer" thread

---

## Objective

For every gate in the system (spawn gates, completion gates, pre-commit gates), determine whether it's filtering signal or generating noise. Gates that generate noise get fixed or removed. Gates that filter signal get measured for accuracy. The end state: every surviving gate has a known false positive rate and a justification for its cost.

---

## Substrate Consulted

- **Models:** harness-engineering (updated today — enforcement+measurement pairing), completion-verification (14-gate pipeline, gate type taxonomy)
- **Decisions:** "Compaction during last-mile is survivable" (today), "Three-layer hotspot enforcement" (Feb 26)
- **Data:** `orch stats` output (201 spawns, 198 completions, gate bypass/skip/fail rates), events.jsonl (4,825 events)
- **Constraints:** Gate decision events just shipped — historical data is limited. Must work with what we have + prospective collection.

---

## Decision Points

### Decision 1: What to do with noise gates

**Context:** When a gate is determined to be noise, we could fix it, downgrade it (hard→soft/advisory), or remove it entirely.

**Options:**
- **A: Fix first, remove if unfixable** — Attempt to calibrate (e.g., fix self_review false positives). If calibration fails after one attempt, remove. Preserves gate intent.
- **B: Remove immediately** — Noise gates are actively harmful (normalize bypassing). Don't spend effort fixing what shouldn't exist.
- **C: Downgrade to advisory** — Keep the check but make it non-blocking. Collects data without forcing bypasses.

**Recommendation:** A (fix first) for gates with clear false-positive patterns (self_review). C (downgrade) for gates where the signal is ambiguous. B (remove) only for gates with no plausible signal.

**Status:** Decided

### Decision 2: How to measure gate accuracy

**Context:** For signal gates, we need false positive and true positive rates. But we don't have labeled ground truth — we don't know which gate blocks were "correct."

**Options:**
- **A: Retrospective audit** — Sample 20-30 gate failures/bypasses, manually classify as correct/incorrect based on what the agent actually produced.
- **B: Prospective tracking** — Let gates run for 2-4 weeks with gate_decision events, then correlate blocks with outcomes (did redirected work succeed?).
- **C: Both** — Retrospective for immediate signal, prospective for ongoing accuracy.

**Recommendation:** C — retrospective gives us answers this week, prospective gives us the ongoing measurement surface.

**Status:** Decided

---

## Phases

### Phase 1: Gate census and classification

**Goal:** Enumerate every gate, classify as signal/noise/unknown based on existing data.
**Deliverables:**
- Gate inventory table: gate name, type (spawn/completion/pre-commit), bypass/skip/fail count, repeat-reason count, classification (signal/noise/unknown)
- For each noise-classified gate: specific false-positive pattern identified
**Exit criteria:** Every gate has a classification with evidence.
**Depends on:** Nothing — uses existing `orch stats` data.

### Phase 2: Fix noise gates

**Goal:** For each noise gate, either fix the false positive, downgrade to advisory, or remove.
**Deliverables:**
- self_review: fix `fmt.Print` false positive pattern
- hotspot bypass: add reason recording (66 with no reason = unmeasurable)
- verified/explain_back: evaluate whether these are noise or just orchestrator-input gates that need workflow change
- Any gate with >50% bypass rate gets a calibration issue
**Exit criteria:** No gate has >3 repeated identical bypass reasons. Bypass rate drops below 50%.
**Depends on:** Phase 1 (need classification before acting)

### Phase 3: Retrospective accuracy audit

**Goal:** For signal gates, determine false positive rate from historical data.
**Deliverables:**
- Sample 20-30 gate blocks/failures across gate types
- Manually classify each as: true positive (gate caught real problem), false positive (gate blocked good work), ambiguous
- Per-gate false positive rate estimate
**Exit criteria:** Each signal gate has an estimated false positive rate with confidence interval.
**Depends on:** Phase 1 (need to know which gates are signal)

### Phase 4: Prospective measurement baseline

**Goal:** Establish ongoing gate accuracy tracking using the instrumentation shipped today.
**Deliverables:**
- gate_decision events accumulating for 2+ weeks
- Gate effectiveness section in `orch stats` populated with real data
- Correlation query: blocked agents → redirected work → outcome
**Exit criteria:** Can answer "for agents that hit gate X, was the outcome better than ungated agents?" with statistical confidence.
**Depends on:** Phases 1-3 + 2-4 weeks of data accumulation. Aligns with Mar 24 accretion probe checkpoint.

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Noise gate response | Stats data, bypass reasons | Yes |
| Accuracy measurement | gate_decision events (new), historical stats | Yes (retrospective now, prospective needs time) |

**Overall readiness:** Ready to execute Phase 1 immediately.

---

## Structured Uncertainty

**What's tested:**
- ✅ 69.7% bypass rate from 201 spawns (stats data)
- ✅ self_review false positive pattern (10+ identical bypass reasons)
- ✅ 66 hotspot bypasses with no reason recorded
- ✅ Gate decision instrumentation is live (shipped today)

**What's untested:**
- ⚠️ Whether fixing self_review false positive changes overall bypass rate meaningfully
- ⚠️ Whether retrospective audit sample (20-30) is representative
- ⚠️ Whether 2-4 weeks is enough data for prospective accuracy measurement

**What would change this plan:**
- If Phase 1 reveals most gates are signal (low noise), skip Phase 2 and go straight to accuracy
- If retrospective audit shows >50% false positive rate across gates, consider whether the gate *philosophy* is wrong, not just calibration

---

## Success Criteria

- [ ] Every gate classified as signal/noise/unknown with evidence
- [ ] No gate has >3 repeated identical bypass reasons (noise eliminated)
- [ ] Spawn gate bypass rate below 50% (down from 69.7%)
- [ ] Each signal gate has a measured false positive rate
- [ ] Can answer "do gates improve agent quality?" with data, not belief
