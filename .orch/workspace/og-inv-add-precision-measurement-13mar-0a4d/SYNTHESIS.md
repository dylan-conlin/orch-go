# Session Synthesis

**Agent:** og-inv-add-precision-measurement-13mar-0a4d
**Issue:** orch-go-5d2gy
**Outcome:** success

---

## TLDR

Retrospective precision audit of the duplication detector found **64.6% precision** (35% false positive rate) across 259 match occurrences in 67 completion events — contradicting the assumed 0% FP. Root causes: structural coincidence (40% of FPs — same Go pattern, different domain), different semantics (47% — similar AST shape, different logic), self-match bug (7% — same function in both modified/corpus partitions), and opposite operations (7% — lock/unlock sharing boilerplate). Model updated with quantified precision data and FP taxonomy.

---

## Plain-Language Summary

The duplication detector runs on every agent completion and flags function pairs that look structurally similar. It was assumed to have zero false positives — every detection was treated as real code duplication worth extracting. I classified every detection from the event log (259 matches across 54 unique function pairs) by reading both functions in each pair. Result: only 65% of detections are real duplication. The other 35% are functions that happen to look similar to the AST fingerprinter but aren't actually extractable — things like `Logger.Log` and `WriteCheckpoint` which both write JSONL files but serve completely different purposes, or `findZombieIssues` and `findPhantomIssues` which have similar structure but different logic.

This confirms the harness engineering model's invariant that "enforcement without measurement is theological" — we believed the detector was precise, but measurement shows it generates noise at nearly 1:2 ratio with signal.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification checklist.

Key outcomes:
- Probe file: `.kb/models/harness-engineering/probes/2026-03-13-probe-duplication-detector-precision-measurement.md`
- Model updated: `.kb/models/harness-engineering/model.md` (invariant 7, Layer 2 status, Section 7 evidence)
- Precision: 64.6% (TP=164, FP=90, borderline=5, total=259)

---

## Delta (What Changed)

### Files Created
- `.kb/models/harness-engineering/probes/2026-03-13-probe-duplication-detector-precision-measurement.md` — Probe with full precision analysis
- `.orch/workspace/og-inv-add-precision-measurement-13mar-0a4d/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-add-precision-measurement-13mar-0a4d/VERIFICATION_SPEC.yaml` — Verification contract

### Files Modified
- `.kb/models/harness-engineering/model.md` — Updated invariant 7, Layer 2 status, and Section 7 evidence with precision data

---

## Evidence (What Was Observed)

- 67 `duplication.detected` events in `~/.orch/events.jsonl` containing 259 total match occurrences across 54 unique function pairs
- Manual classification of all 54 pairs by reading both functions: 33 TP, 15 FP, 6 borderline
- Occurrence-weighted precision: 164 TP / (164 + 90) = 64.6%
- Unique pair precision: 33 / (33 + 15) = 68.8%
- Largest single FP source: `(Logger).Log ↔ WriteCheckpoint` at 23 occurrences (25.6% of all FP)
- Self-match bug: `inferSkillFromBeadsIssue` matched against itself (same function in both modified/corpus partitions)
- Existing `.dupdetectignore` covers only `(Logger).Log*` patterns — does not suppress the largest FP pair because allowlist requires BOTH sides to match the same pattern

---

## Architectural Choices

No architectural choices — this was a measurement/investigation task.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/harness-engineering/probes/2026-03-13-probe-duplication-detector-precision-measurement.md` — Full precision audit with FP taxonomy

### Constraints Discovered
- AST fingerprinting at 0.85 threshold produces ~35% false positives in a 80K+ line Go codebase
- The allowlist mechanism (`.dupdetectignore`) requires both sides of a pair to match the same pattern — this design prevents suppressing structural-coincidence FPs where function names differ
- `CheckModifiedFilesProject` has a self-match bug where the same function appears in both modified and corpus partitions

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Follow-up Issues

1. **Expand `.dupdetectignore`** — Add patterns for the 15 known FP pairs. Would raise precision from 65% to ~95% for currently-known pairs. Low-effort (no code change).

2. **Fix self-match bug** — Deduplicate functions in `CheckModifiedFilesProject` before comparison. The same function should not appear in both modified and corpus partitions.

3. **Add `duplication.suppressed` event** — When allowlist suppresses a pair, log it. This creates passive ongoing precision measurement: `precision = detected / (detected + suppressed)`.

---

## Unexplored Questions

- **Recall measurement:** This probe measured precision (what % of detections are real) but not recall (what % of real duplications are detected). Are there significant duplications the 0.85 threshold misses?
- **Threshold sensitivity:** Would raising to 0.90 eliminate most "different semantics" FPs while keeping the TP pairs (which tend to be 92%+ similarity)?
- **Beads RPC pattern:** The largest TP cluster (35+ occurrences) is beads RPC-with-CLI-fallback duplication. This is a strong extraction signal — has it been actioned?

---

## Friction

Friction: none — data was readily available in events.jsonl and code was straightforward to read.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-add-precision-measurement-13mar-0a4d/`
**Probe:** `.kb/models/harness-engineering/probes/2026-03-13-probe-duplication-detector-precision-measurement.md`
**Beads:** `bd show orch-go-5d2gy`
