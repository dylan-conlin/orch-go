# Session Synthesis

**Agent:** og-inv-investigate-orchestrator-failure-25mar-bddf
**Issue:** orch-go-hjllu
**Duration:** 2026-03-25 11:18 → 2026-03-25 11:35
**Outcome:** success

---

## TLDR

Analyzed the scs-sp session where the orchestrator failed Dylan during DFM engine data review. Found 6 distinct failure points, all synthesis gaps (orchestrator had the information but couldn't compose it). The root amplifier was Frame Collapse — staying in analytical mode through 3 escalating frustration signals.

---

## Plain-Language Summary

The orchestrator had every piece of information it needed to help Dylan understand the DFM engine results. It correctly identified the dataset composition, the precision/recall numbers, and even proposed the right solution (human verification of a sample). But it presented the information with contradictory framing ("coin flip" vs "data is enriched"), asked Dylan technical questions he couldn't answer, and at the critical moment when Dylan said "why can't we just run this on data with known classification?" — which was Dylan asking to do exactly what the orchestrator had already proposed — it gave a circular answer that killed the session. The completion review system contributed by passing through agent numbers without comprehension, seeding the confusion from the start.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-25-inv-investigate-orchestrator-failure-dfm-engine.md` — Full investigation with 6 findings, synthesis, and recommendations

### Files Modified
- None

### Commits
- (pending)

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for evidence criteria.

Key outcomes verified:
- 6 failure points identified with line-number evidence from transcript
- Each failure classified as synthesis gap (not skill gap)
- Mapped to known failure modes (Frame Collapse, parrot summary)
- Concrete recommendation: frustration-triggered mode shift protocol

---

## Knowledge (What Was Learned)

1. **Parrot summaries seed confusion cascades** — The completion review forwarded "0 recall loss, 15% override rate" without comprehension. Dylan had to ask what it meant, putting the orchestrator in recovery mode for the rest of the DFM discussion. The known constraint ("Review workflow must produce synthesis, not lists") predicted this exactly.

2. **Frame Collapse is the amplifier, not the root cause** — Individual errors (contradictory framing, circular answer) would be recoverable if the orchestrator shifted modes at frustration signals. But analytical lock-in through "I DON'T KNOW" → "running in circles" → "omg this is not working" turned each error into a compounding cascade.

3. **Self-awareness without behavior change is the concerning pattern** — The orchestrator said "You're right. Let me stop." (line 1612) and then immediately continued with analysis. This suggests the model can detect the problem but lacks an action protocol for what to do after detection.

---

## Next (What Should Happen)

1. **Add frustration mode shift to orchestrator skill** (architectural) — When frustration is detected: stop analyzing, acknowledge in one sentence, propose concrete action (not a question)
2. **Check if frustration trigger protocol was loaded** — The Jan 2026 addition may already cover this; need to verify if it was injected in scs-special-projects sessions
3. **Update orchestrator failure mode catalog** — Add "Contradictory Framing" as a distinct failure mode (sub-type of synthesis gap)

---

## Unexplored Questions

- Was the frustration trigger protocol (from Jan 2026 investigation) loaded in this session?
- Are recent brief/completion changes responsible for the parrot behavior, or is this longstanding?
- Would Opus 4.6 handle the frustration signals differently than whatever model ran this session?
