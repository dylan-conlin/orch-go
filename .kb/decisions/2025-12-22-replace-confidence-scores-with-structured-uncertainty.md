# Decision: Replace Confidence Scores with Structured Uncertainty

**Date:** 2025-12-22
**Status:** Accepted
**Deciders:** Dylan, Orchestrator

## Context

Investigation templates have included confidence percentages (e.g., "Confidence: High (85%)") since the early orchestration system. The intent was calibration - helping readers assess reliability.

## Problem

Evidence shows confidence scores are harmful, not helpful:

1. **Documented backfire:** Nov 2025 Codex case study - 5 investigations claimed High/Very High confidence (70-95%), all reached wrong conclusions about root cause
2. **No discriminative value:** Current orch-go investigations show 90% clustering at 85-95% confidence - the metric provides no signal
3. **Fundamentally uncalibratable:** LLMs optimize for sounding confident, not being accurate. Asking for confidence percentages is asking the wrong question.

## Decision

**Remove percentage-based confidence scores. Replace with structured uncertainty.**

| Remove | Replace With |
|--------|--------------|
| `**Confidence:** High (85%)` | (delete entirely) |
| `**What's certain:**` | `**What's tested:**` (requires evidence of actual test) |
| `**What's uncertain:**` | `**What's untested:**` (explicit hypotheses without validation) |
| - | `**What would change this:**` (falsifiability criteria) |

## Rationale

- "Here's what I don't know" is more actionable than "I'm 70% confident"
- Evidence-over-estimation: if agent can't point to something concrete, that's the low confidence signal
- Forces enumeration of tested vs untested claims rather than vibes-based percentages
- Aligns with "test before concluding" investigation discipline

## Consequences

- All investigation templates need updating (orch-knowledge, skill sources)
- D.E.K.N. summary format may need adjustment (currently has Confidence line)
- Historical investigations retain old format (not worth backfilling)

## References

- `.kb/investigations/2025-12-22-inv-trace-confidence-score-effectiveness.md`
- Codex epistemic debt case study (orch-knowledge, Nov 2025)
- Investigation skill SKILL.md acknowledges "old investigation system produced confident wrong conclusions"
