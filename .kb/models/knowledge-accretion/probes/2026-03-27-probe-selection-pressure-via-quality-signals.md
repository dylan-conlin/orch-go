# Probe: Selection Pressure via Quality Signals — Can Automated Filtering Complement Gates?

**Model:** knowledge-accretion
**Date:** 2026-03-27
**Status:** Complete
**claim:** Effectiveness hierarchy, coordination taxonomy, HyperAgents extension (evolutionary selection)
**verdict:** extends

---

## Question

The knowledge accretion model's effectiveness hierarchy says: structural attractors > signaling mechanisms > blocking gates > advisory > metrics-only. HyperAgents extended this with "evolutionary selection" as a structural attractor. Can orch-go incorporate a lighter form of selection pressure — automated quality signals on syntheses — that sits between structural attractors and advisory gates? Does this mechanism fit the model, or is it a new category?

Specifically: the HyperAgents probe (2026-03-26) raised follow-up question #1: "Could orch-go incorporate selection pressure? E.g., when 3+ agents have investigated the same question, use automated evaluation to select the best synthesis rather than expecting each agent to check prior work." This probe tests that question through design analysis.

---

## What I Tested

1. **Audited existing synthesis pipeline** for available quality signals:
   - `verify.ParseSynthesis()` extracts: TLDR, Delta, Evidence, Knowledge, Next, UnexploredQuestions, ArchitecturalChoices, Recommendation, NextActions
   - `debrief.CheckQuality()` provides: connective language detection (17 phrases), action-verb detection (20 prefixes)
   - `complete_brief.go` maps synthesis → brief with Frame/Resolution/Tension structure
   - `comprehension_queue.go` orders by mod-time only — no quality dimension

2. **Identified 6 mechanically-detectable quality signals** from parsed Synthesis struct:
   - `structural_completeness` (0-4 D.E.K.N. sections populated)
   - `evidence_specificity` (file paths, test output cited in Evidence/Delta)
   - `model_connection` (references to .kb/models/ in Knowledge)
   - `connective_reasoning` (causal language in Knowledge/TLDR — reuses existing detection)
   - `tension_quality` (UnexploredQuestions populated with actual questions)
   - `insight_vs_report` (Knowledge lines that contain insights, not action logs)

3. **Analyzed signal placement against model claims:**
   - Gate lifecycle arc: "designed → measured → found inert or high-FP → downgraded/removed"
   - Effectiveness hierarchy: structural attractors > signaling > blocking gates
   - HyperAgents ablation: selection without archive fails; gates without selection show little improvement
   - "formula-shaped sentences" warning against uncalibrated numeric scores

4. **Examined thread 'every-spawn-composes-knowledge':**
   - Finding #1: "The best briefs compose knowledge by correcting false beliefs. The weakest report accurately but change nothing."
   - Finding #2: "Frame quality determines composition quality."
   - Finding #3: "Tension sections are orphaned knowledge seeds."
   - These findings align with signal taxonomy: `connective_reasoning` and `tension_quality` detect exactly the difference between composing-knowledge briefs and report briefs.

5. **Cross-referenced ranking/attention layer boundary probe:**
   - Layer 1 (substrate ordering): exists, mod-time sort
   - Layer 2 (method-expressing ordering): MISSING — thread grouping, tension surfacing
   - Layer 3 (learned ranking): correctly held-back
   - Quality signals are Layer 2 — they express the method's values (evidence, connection, insight) through ordering.

---

## What I Observed

### 1. Quality signals are a new position in the effectiveness hierarchy

The existing hierarchy: structural attractors > signaling mechanisms > blocking gates > advisory > metrics-only.

Quality signals don't fit cleanly into any existing category:
- They're NOT structural attractors (they don't route contributions at write time)
- They're NOT signaling mechanisms in the existing sense (which trigger daemon responses)
- They're NOT blocking gates (advisory, don't prevent completion)
- They're MORE than advisory (they affect ordering, which affects attention allocation)
- They're MORE than metrics-only (they're embedded in artifacts, not dashboards)

**New category: "attention routing."** Quality signals are metadata embedded in artifacts that route reader attention without blocking production. They sit between signaling mechanisms (which trigger automated responses) and advisory gates (which display warnings). The key difference from advisory: advisories are shown once at completion time and forgotten. Quality signals persist in the artifact and affect every subsequent read.

This maps to what HyperAgents calls "performance-weighted parent selection" — but applied to reading, not reproduction. Syntheses with more quality signals get read first (selection pressure on attention), not syntheses that "reproduce" (no population dynamics in orch-go).

### 2. The signal taxonomy maps to the every-spawn-composes-knowledge findings

The thread identified the quality spectrum: composing-knowledge (best) → correcting beliefs → reporting accurately → reporting without change (weakest).

The 6 signals detect exactly this spectrum:
- **composing-knowledge brief:** model_connection ✓, connective_reasoning ✓, tension_quality ✓ → 6/6 signals
- **correcting-beliefs brief:** evidence_specificity ✓, connective_reasoning ✓, structural_completeness ✓ → 4-5/6
- **accurate-report brief:** structural_completeness ✓, evidence_specificity ✓ → 2-3/6
- **task-completion-only brief:** structural_completeness partial → 0-1/6

The signal count naturally stratifies briefs by composition quality. This isn't designed — it falls out of measuring what the method values (evidence, connection, insight, open questions).

### 3. HyperAgents' "handcrafted beats self-modified" applies here

The HyperAgents ablation found that the meta agent modifying its own selection mechanism "did not outperform the carefully handcrafted mechanism." This has a direct design implication: the 6-signal taxonomy should be handcrafted and stable, not adaptive.

If we built an auto-tuning ranker (adjusting signal weights from brief feedback), the HyperAgents evidence predicts it wouldn't outperform the handcrafted version. This is because:
- The signal taxonomy captures method values (evidence, connection, insight) — these are constitutive, not empirical
- Adaptive weighting optimizes for feedback patterns, which may drift from method values (Goodhart's Law)
- The handcrafted taxonomy is interpretable; adaptive weights are opaque

**Design recommendation confirmed:** Boolean signals with equal-weight `signal_count`, not a learned scoring function.

### 4. This closes the "selection pressure" gap identified in the HyperAgents probe

The HyperAgents probe (Section 5, "Selection Pressure as an Alternative to Gate-Based Coordination") identified three reasons orch-go can't use population-based selection:
1. No automated evaluation function for open-ended knowledge work
2. Everything committed to main stays (no pruning)
3. Human (Dylan) is the selection mechanism, creating a bottleneck

Quality signals address #1 and #3 partially:
- **#1:** The 6 signals ARE an automated evaluation function — not for correctness, but for composition quality. This is the automatable portion of verification (self-disconfirming knowledge thread).
- **#3:** Signals rank briefs so Dylan reads the highest-quality ones first. This doesn't remove the bottleneck but reduces its cost — fewer low-quality briefs consume attention time.
- **#2 remains:** Everything stays. Quality signals don't prune; they prioritize. This is consistent with the model's finding that orch-go needs gates AND attractors, not gates OR selection.

---

## Model Impact

- [x] **Extends** model with:
  1. **New effectiveness hierarchy position: "attention routing"** — metadata embedded in artifacts that routes reader attention without blocking production. Sits between signaling mechanisms (automated responses) and advisory gates (one-time warnings). Distinct because it persists in artifacts and affects every subsequent read, not just the moment of creation.
  2. **Quality signal taxonomy as partial selection pressure:** The 6 mechanically-detectable signals (structural completeness, evidence specificity, model connection, connective reasoning, tension quality, insight vs report) provide automated evaluation for composition quality — the automatable portion of the verification bottleneck. This doesn't replace human judgment but reduces its scope to compositional decisions.
  3. **Handcrafted-beats-adaptive as design principle:** HyperAgents' evidence that self-modified selection doesn't outperform handcrafted selection generalizes to signal taxonomy design: handcraft signals from method values, don't auto-tune from feedback. The signals capture what the method constitutively values, not what correlates with outcomes.

---

## Notes

- The signal taxonomy is NOT a gate — it doesn't block completion. The gate lifecycle arc (designed → measured → inert → removed) doesn't apply because signals are never blocking. They're pure attention-routing.
- The 6 signals are all derived from fields already parsed by `verify.ParseSynthesis()`. No new parsing infrastructure needed.
- `signal_count` (how many of 6 signals fired) is a crude but honest sort key. It treats all signals as equal weight, which is probably wrong — evidence_specificity likely matters more than structural_completeness. But equal weight is the honest prior when calibration data doesn't exist. The knowledge accretion model's "formula-shaped sentence" warning applies to premature weighting.
- If brief feedback (shallow/good) accumulates enough data, the correlation between signals and feedback could calibrate weights. But per the HyperAgents finding, this calibration may not outperform equal weights. Worth testing empirically if the data arrives.
