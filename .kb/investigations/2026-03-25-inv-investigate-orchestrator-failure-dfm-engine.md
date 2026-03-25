## Summary (D.E.K.N.)

**Delta:** The orchestrator failed Dylan 6 times during DFM engine data review — every failure was a synthesis gap (had the information, couldn't compose it), not a skill gap. The root cause: the orchestrator stayed in analytical mode through 3 escalating frustration signals, and its contradictory framing of the data ("coin flip" + "data is enriched with conflicts") created the confusion Dylan experienced.

**Evidence:** Transcript analysis of 1665-line session. Orchestrator parroted "0 recall loss, 15% override rate" without understanding it (line 1406). Presented contradictory framings that confused Dylan (lines 1514 vs 1520). At line 1653, Dylan asked "why can't we just run this on data with known classification?" — asking to CREATE a clean test set — and the orchestrator gave a circular answer about needing known answers.

**Knowledge:** The orchestrator's completion review (brief/SYNTHESIS parrot) seeded the initial confusion. The known constraint "Review workflow must produce synthesis, not lists" predicted this failure exactly. The orchestrator also hit the known "Frame Collapse" failure mode — staying in analytical frame through frustration escalation instead of shifting to problem-solving mode.

**Next:** Two recommendations: (1) architectural — add "frustration detection → mode shift" to orchestrator skill guidance, (2) strategic — decide whether completion review should present agent numbers at all vs always re-derive from source data.

**Authority:** architectural — Cross-component fix touching orchestrator skill and completion review flow

---

# Investigation: Orchestrator Failure — DFM Engine Data Confusion

**Question:** Why did the orchestrator fail to help Dylan reason about DFM engine evaluation data, and is this a skill gap (doesn't know ML evaluation) or a synthesis gap (had information but couldn't compose it)?

**Started:** 2026-03-25
**Updated:** 2026-03-25
**Owner:** orch-go-hjllu
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/orchestrator-skill/probes/2026-03-11-probe-orchestrator-skill-failure-mode-taxonomy.md | extends | yes | - |
| .kb/investigations/2026-02-28-investigation-orchestrator-intent-spiral.md | extends | yes (by pattern match) | - |

**Relationship types:** extends (adds new evidence to existing failure mode catalog)

---

## Findings

### Finding 1: Parrot Summary — Completion Review Failed to Produce Synthesis

**Evidence:** At line 1406, the orchestrator presents scs-sp-0z3 completion with: "With LLM layer: 0 recall loss, 15% override rate (on 20-part sample)." Dylan immediately asks "what does this mean" (line 1422). The orchestrator admits "I'm just parroting the agent's summary without actually understanding what those numbers mean in context" (line 1425).

After reading the actual data, the orchestrator provides a reasonable explanation (lines 1432-1452). But the initial presentation was lifted from the agent's SYNTHESIS.md without comprehension.

**Source:** Transcript lines 1406-1452

**Significance:** This is the exact failure the known constraint predicts: "Review workflow must produce synthesis, not lists — Dylan repeatedly asks 'what's ready to review' and gets bare numbered results without comprehension." The brief/completion system passed numbers through without meaning. This seeded all subsequent confusion — if the orchestrator had understood the data before presenting, it could have framed it correctly from the start.

---

### Finding 2: Contradictory Framing Created Dylan's Confusion

**Evidence:** In a single response (lines 1508-1535), the orchestrator presents two incompatible framings:

- "48% precision is a coin flip" (line 1514)
- "this ground truth dataset is built from recut data — parts that already had problems. It's heavily enriched with conflicts" (lines 1520-1522)

Dylan's response: "i feel like i'm being told that our system is a coin flip but then i'm being told that the data is all recuts. i'm lost" (lines 1543-1545).

The orchestrator's framing was internally contradictory. If the data is enriched with conflicts, then 48% precision has a specific interpretation: it means the engine flags too many *clean parts*, not that it's randomly guessing. The "coin flip" metaphor was misleading because it implies random behavior — the engine's behavior is systematic (it over-flags), not random.

**Source:** Transcript lines 1508-1545

**Significance:** This is the central comprehension failure. The orchestrator had both pieces of information but composed them in a way that confused rather than clarified. It presented the data enrichment as a "caveat" instead of as the core context for interpreting the precision number.

**What should have happened:** "The engine catches 82% of real issues (good). But on this test set — which has more conflicts than normal production — it flags 62% of parts that shipped fine as conflicts. In production, where most parts are clean, the false positive rate would be lower, but we don't have a production measurement."

---

### Finding 3: Orchestrator Asked Dylan a Question He Couldn't Answer

**Evidence:** After a detailed breakdown of the dataset (33 CONFLICT, 47 CLEAR, 95 AMBIGUOUS), the orchestrator asks: "What's your instinct on the ambiguous bucket — is the classification between geometric/process/ambiguous trustworthy, or is that where the real work needs to happen?" (lines 1583-1585)

Dylan's response: "I DON'T KNOW" (line 1589).

**Source:** Transcript lines 1554-1589

**Significance:** The orchestrator routed a technical data quality question to Dylan instead of proposing how to answer it. Dylan is a strategist/operator, not a data engineer debugging classification schemes. The orchestrator should have recognized this was its job to investigate (or propose investigating), not Dylan's to assess by instinct.

**What should have happened:** "The 95 ambiguous parts are the core problem — they're more than half the data and excluded from scoring. To resolve this, someone needs to review a sample of the actual part geometry. Want me to create a task for that?"

---

### Finding 4: Three Frustration Escalations — Zero Mode Shifts

**Evidence:** Dylan signaled escalating frustration three times:

1. **"I DON'T KNOW"** (line 1589) — First frustration signal. Orchestrator responds with more analysis: "we built a scoring pipeline before we had clean labels" (line 1594). Correct analysis, wrong mode.

2. **"i feel like we're running in circles"** (line 1610) — Meta-frustration. Orchestrator says "You're right. Let me stop." then immediately asks another analytical question: "What's the actual thing you're trying to accomplish with Matt?" (lines 1612-1615). Self-awareness without behavior change.

3. **"omg this is not working"** (line 1664) — Session-ending frustration. This is the last line of the transcript. The orchestrator's previous response (lines 1655-1662) had given a circular answer that provoked this.

At no point did the orchestrator shift out of analytical mode. Each escalation was met with more analysis or more questions.

**Source:** Transcript lines 1589, 1610, 1664

**Significance:** This maps to the known "Frame Collapse" failure mode from the orchestrator skill failure taxonomy. The orchestrator collapses into a single frame (analytical) and cannot shift to another (problem-solving, emotional acknowledgment, or action-oriented). The frustration trigger protocol (added Jan 2026, per investigation 2026-01-18) was either not loaded or not effective.

---

### Finding 5: Circular Answer at the Critical Moment

**Evidence:** Dylan asks: "why can't we just run this on data with known classification?" (line 1653)

Orchestrator responds: "Because we'd need parts where we already know the answer — 'this part has a geometric hw+bend conflict' or 'this part is clean.'" (lines 1655-1657)

Dylan was asking: "Why don't we CREATE a dataset with known, clean labels?" The orchestrator heard: "Where can we FIND pre-existing labeled data?"

The orchestrator's own analysis, two responses earlier, had proposed exactly what Dylan was asking about: "Take 30-50 parts the engine flags as CONFLICT, have someone who knows the parts look at the actual geometry and say 'yes this is a real conflict' or 'no it's not.' That gives you real precision in an afternoon." (lines 1636-1641, path A)

So the orchestrator proposed the right solution, then failed to recognize Dylan asking for exactly that solution in simpler terms.

**Source:** Transcript lines 1636-1641 vs 1653-1662

**Significance:** This is the most damaging failure. Dylan's question was the breakthrough moment — he was saying "let's just do the thing." The orchestrator had already proposed the thing. But it interpreted Dylan's plain-language restatement as a different question and answered it circularly. This turned a potential resolution into the session-ending frustration at line 1664.

---

### Finding 6: Harness False Positive Contributed to Session Friction

**Evidence:** At line 1282, `orch status` reports scs-sp-0z3 as UNRESPONSIVE. The orchestrator proposes abandoning and re-releasing (line 1291). Dylan corrects: "you just ran into a harness error. 0z3 is running fine" (line 1296).

**Source:** Transcript lines 1282-1300

**Significance:** This isn't directly about the DFM data confusion, but it shows the orchestrator trusting harness signals over investigation — the exact pattern the constraint "Agent failures are harness failures until proven otherwise" was designed to prevent. This minor failure contributed to session friction (Dylan had to correct the orchestrator) and may have eroded trust that affected the later data discussion.

---

## Synthesis

**Key Insights:**

1. **Synthesis gap, not skill gap** — The orchestrator correctly identified the dataset composition (33/47/95 split), correctly calculated TP/FP/FN/TN, correctly identified the false positive problem, and even proposed human verification as the right path forward. Every piece of information needed to help Dylan was available. The failure was in composing and presenting it: contradictory framing, asking questions instead of proposing actions, and failing to recognize Dylan restating the solution.

2. **Completion review seeded the confusion cascade** — The parrot summary at line 1406 ("0 recall loss, 15% override rate") set the tone. When Dylan had to ask what it meant, the orchestrator was already in recovery mode — explaining after the fact rather than framing coherently from the start. If the completion review had produced synthesis instead of forwarding numbers, the entire DFM data discussion might have started from a clearer foundation.

3. **Frame Collapse is the amplifier** — Individual comprehension errors (contradictory framing, circular answer) would be recoverable if the orchestrator shifted modes at frustration signals. But staying in analytical mode through all three escalations meant each error compounded instead of being interrupted and corrected. The orchestrator's self-awareness ("You're right. Let me stop." at line 1612) without actual behavior change is the most concerning pattern — it demonstrates the model can detect the problem but not act on the detection.

**Answer to Investigation Question:**

The orchestrator's failure was a **synthesis gap**. It had all the information needed to help Dylan but couldn't compose it coherently, couldn't shift out of analytical mode when frustrated signals demanded it, and at the critical moment (line 1653) couldn't map Dylan's plain-language question to the solution it had already proposed. The brief/completion system contributed by forwarding undigested numbers that seeded the confusion. The orchestrator skill's frame collapse failure mode — staying in one cognitive mode despite escalation — amplified every individual error into a session-ending breakdown.

---

## Structured Uncertainty

**What's tested:**

- ✅ The orchestrator had the correct data — verified by comparing transcript analysis (lines 1554-1576) to actual dataset description
- ✅ Contradictory framing confused Dylan — verified by Dylan's explicit statement at lines 1543-1545
- ✅ Three frustration escalations received analytical responses, not mode shifts — verified line-by-line at 1589, 1610, 1664
- ✅ Line 1653 question was about CREATING labeled data, not FINDING it — verified by context (orchestrator had proposed this 13 lines earlier)

**What's untested:**

- ⚠️ Whether the frustration trigger protocol was loaded in this session (would need to check skill injection for scs-special-projects)
- ⚠️ Whether recent brief/completion changes specifically caused the parrot behavior vs this being a longstanding pattern
- ⚠️ Whether a different model (e.g., Opus 4.6 vs Opus 4.5) would have handled the frustration signals differently

**What would change this:**

- If the frustration trigger protocol was loaded and explicitly ignored, this is a compliance failure, not a skill gap
- If the completion review system was recently changed to include more raw numbers, the parrot behavior might be a regression rather than a baseline

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add frustration-triggered mode shift to orchestrator skill | architectural | Cross-component: changes orchestrator behavior across all projects |
| Completion review should re-derive meaning from source data | strategic | Changes the fundamental completion review workflow for Dylan |
| Add "contradictory framing" to orchestrator failure mode catalog | implementation | Extends existing taxonomy within established model |

### Recommended Approach: Frustration Mode Shift Protocol

**Add a concrete behavior trigger to the orchestrator skill:** When Dylan signals frustration (caps, "I don't know", "running in circles", "not working"), the orchestrator MUST:
1. Stop analyzing
2. Acknowledge the friction in one sentence
3. Propose a concrete next action (not a question, not a framework)

**Why this approach:**
- The orchestrator already detects frustration ("You're right. Let me stop.") but doesn't change behavior
- The fix is behavioral, not informational — the skill needs an explicit action sequence, not more awareness
- This directly addresses the Frame Collapse failure mode identified in the taxonomy

**Trade-offs accepted:**
- May cause premature action-taking when the user is frustrated but still exploring — acceptable because the current failure mode (continued analysis through frustration) is worse

### Alternative Approaches Considered

**Option B: Re-derive numbers during completion review (always read source data)**
- **Pros:** Prevents parrot summaries, forces comprehension before presentation
- **Cons:** Adds significant time to completion review, may be redundant when agents write good SYNTHESIS.md
- **When to use instead:** If parrot behavior is the primary failure mode across many sessions (not just this one)

**Option C: Add ML evaluation reasoning to orchestrator skill**
- **Pros:** Would prevent the contradictory framing around precision/recall
- **Cons:** Domain-specific knowledge shouldn't live in a general orchestrator skill — this was a synthesis failure, not a knowledge failure
- **When to use instead:** If ML evaluation work becomes a regular pattern (not a one-off)

**Rationale for recommendation:** The frustration mode shift addresses the amplifier (Frame Collapse) rather than individual errors. Individual errors (contradictory framing, circular answer) are hard to prevent via skill guidance — they're emergent. But the failure to shift modes when Dylan signals frustration IS preventable with explicit behavioral triggers.

---

### Implementation Details

**What to implement first:**
- Add frustration detection → mode shift sequence to orchestrator skill (3-step protocol above)
- Add "Contradictory Framing" as failure mode to orchestrator-session-lifecycle model

**Things to watch out for:**
- ⚠️ The frustration trigger protocol from Jan 2026 (inv-update-orchestrator-skill-add-frustration) may already address this — need to check what it does vs what happened here
- ⚠️ Adding more behavioral rules to the skill risks the known "Behavioral Constraint Dilution" failure mode

**Areas needing further investigation:**
- Was the frustration trigger protocol loaded in this scs-special-projects session? If yes, why did it fail?
- Is the parrot behavior a regression from recent completion/brief changes?

**Success criteria:**
- ✅ At next frustration signal, orchestrator stops analysis and proposes concrete action
- ✅ Completion review presents meaning, not forwarded numbers
- ✅ This pattern does not repeat in next 5 orchestrator sessions

---

## References

**Files Examined:**
- Transcript: `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/2026-03-25-111539-lets-begin-with-orch-review.txt` — Full 1665-line session transcript
- `.kb/models/orchestrator-skill/probes/2026-03-11-probe-orchestrator-skill-failure-mode-taxonomy.md` — Failure mode catalog for cross-reference

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-28-investigation-orchestrator-intent-spiral.md` — Prior Frame Collapse / intent displacement evidence
- **Investigation:** `.kb/investigations/2026-01-18-inv-update-orchestrator-skill-add-frustration.md` — Frustration trigger protocol addition
- **Constraint:** "Review workflow must produce synthesis, not lists" — Predicts Finding 1 exactly
- **Constraint:** "Agent failures are harness failures until proven otherwise" — Violated at Finding 6

---

## Investigation History

**2026-03-25 11:20:** Investigation started
- Initial question: Why did the orchestrator fail to help Dylan reason about DFM engine data?
- Context: Spawned from scs-sp session where Dylan escalated frustration 3 times

**2026-03-25 11:25:** Full transcript analyzed (1665 lines)
- Identified 6 distinct failure points across the session
- Confirmed synthesis gap (not skill gap) — orchestrator had correct data throughout

**2026-03-25 11:30:** Investigation completed
- Status: Complete
- Key outcome: Synthesis gap amplified by Frame Collapse — orchestrator had the right information and the right solution but couldn't compose it for Dylan or shift modes at frustration signals
