# Design: Grammar-First Skill Architecture — Which 4 Behavioral Constraints Get Document Slots

**Date:** 2026-03-04
**Status:** Complete
**Beads:** orch-go-k3nu
**Depends on:** fabrication experiment (orch-go-wou2), injection-level experiment (orch-go-pkp2), simplify-orchestrator-skill (orch-go-w7xe), layered enforcement probe (Mar 2)

## TLDR

The 4 behavioral document slots must be **matched pairs**: each rule needs specific calibration knowledge alongside it to prevent the U-curve danger zone (5C: 20% correct on urgency vs bare 100%). The current v4 norms (delegation, filter, act-by-default, answer-the-question) are 75% correct but missing the most critical slot: the **undefined behavior handler**. Replace "answer the question asked" (which is a knowledge constraint wearing behavioral clothing) with an explicit boundary handler.

## The Problem

The fabrication experiment (wou2) proved that behavioral constraints without calibration context create a danger zone worse than no constraints at all. At 5 constraints, agents over-apply triage to urgent issues at 80% rate (vs 0% bare). The mechanism: process knowledge creates gravitational pull that overrides situational judgment.

**Budget (from behavioral-grammars model):**
- ~4 behavioral constraints (the rules, in prompt)
- ~50 knowledge constraints (calibration context, in prompt — survives dilution)
- Unlimited hard enforcement via hooks (31 constraints already moved)
- Unlimited coaching via hooks (28 constraints available)

**Design constraint:** Each behavioral slot costs ~10x more attention budget than a knowledge slot. Wasting one on something that could be knowledge-framed is an architectural mistake.

## Evidence-Based Slot Selection

### What the U-curve teaches

The 5C variant had these constraints: delegation, intent clarification, anti-sycophancy, phase reporting, no-bd-close. All five are pure behavioral prohibitions with zero calibration context. The result:

- `no-full-triage`: bare 100% correct → 5C **20%** correct (catastrophic regression)
- `acknowledges-limits`: bare 77% → 5C 70% (mild degradation)

The 10C variant added: architect routing, session close, beads tracking, context loading, tool preference. Still pure behavioral, still no calibration. Result: 30% on urgency (slight recovery because more constraints dilute each other, reducing any single constraint's gravitational pull).

The current v4 (448 lines, ~47 knowledge items + 4 behavioral norms) scores 70% on urgency. **The knowledge items are doing the calibration work.** The routing tables, stall triage, completion lifecycle — these are the context that teaches the agent WHEN process applies and when it doesn't.

### The matched-pair principle

A behavioral constraint is a rule: "do X" / "don't do Y". A calibration context is knowledge that bounds the rule: "X applies when A, not when B." Without the calibration, the agent has a hammer and everything looks like a nail.

**The 5C failure:** "Delegate code work" (rule) + nothing about urgency/triviality (no calibration) = agent delegates production fire response to a multi-day investigation pipeline.

## The 4 Slots

### Slot 1: Delegation + calibration

**Rule:** "You never implement. Code-level understanding is investigation work — spawn it."

**Why behavioral (not knowledge):** Delegation fights the model's strongest prior — the desire to solve problems directly. This is counter-instinctual, the hardest category. It MUST be behavioral.

**Calibration context (knowledge constraints that must co-reside):**
- The three jobs (COMPREHEND, TRIAGE, SYNTHESIZE) — defines what orchestrator DOES do
- The 5-minute context gathering rule — defines the boundary
- Tool action space — explicit list of what orchestrator reads
- Fast path surface table — shows that some situations map to immediate action (urgent → `bd create -l triage:ready`), not investigation
- Stall triage failure modes — prevents over-investigating stalled agents

**Why this calibration set:** The fast path table is the critical piece. It teaches the agent that "Dylan reports symptom" → immediate issue creation, not "let me spawn an investigation." Without this, delegation over-applies to trivial/urgent work (the 5C failure).

**Minimum knowledge items for calibration:** ~8 (three jobs, 5-min rule, tool action space, fast path table entries for urgency/completion/frustration)

### Slot 2: Undefined behavior handler

**Rule:** "When a situation doesn't map to your routing tables or known patterns, say so. Don't force-fit into the nearest process."

**Why this slot exists:** The 5C experiment's most damaging failure was force-fitting (80% of agents proposed triage for an urgent production issue). The agent had process knowledge (triage exists) but no explicit instruction for "what to do when the situation doesn't fit." Without this, the nearest process becomes a gravitational attractor.

**Why behavioral (not knowledge):** This fights the model's completion prior — the drive to always provide a structured response. Saying "I don't have a pattern for this" feels like failure to the model. It must be an explicit behavioral norm.

**Calibration context (knowledge constraints):**
- Skill selection decision tree — defines what IS mapped
- Intent clarification patterns — shows the shape of ambiguity resolution
- Epic model phases (probing/forming/ready) — teaches that "unknown" is a valid state, not a failure state

**Why this calibration set:** The decision tree defines the boundary of known patterns. The epic model phases normalize the "I don't know yet" state. Together they give the agent permission to stop at the edge of its routing, rather than pushing past it.

**Minimum knowledge items for calibration:** ~5 (skill decision tree, intent clarification table, epic model phases, "strategic question" fast path entry)

**This is the slot the current v4 is missing.** The v4 has "answer the question asked" which is actually a knowledge constraint (investigation findings ≠ design decisions) disguised as behavioral. It can be reformulated as knowledge and freed to make room for undefined behavior handling.

### Slot 3: Filter before presenting (act-by-default)

**Rule:** "Only present options you'd recommend. If obvious next step, act without asking."

**Why combined:** These are two sides of the same coin — reducing decision fatigue for Dylan. "Filter" handles the case where you present options. "Act" handles the case where no options are needed. Combining them into one slot saves budget.

**Why behavioral (not knowledge):** Both fight the model's "be helpful by showing all options" prior and the "ask permission before acting" prior. These are moderate-strength priors (not as strong as delegation), but they directly shape every interaction.

**Calibration context (knowledge constraints):**
- Completion workflow by work type — teaches what auto-completes without asking
- Session start/end protocols — scripted sequences where "just do it" is correct
- Priority emergence pattern — "priority crystallizes through dialogue" teaches when to ACT vs when to SURFACE
- Signal prefixes — Dylan explicitly signals when perspective shift is needed

**Why this calibration set:** The completion workflows and session protocols teach the agent the situations where acting without asking is expected. Signal prefixes teach when Dylan explicitly shifts mode, overriding default act-by-default. Without these, "act by default" either over-applies (agent acts on things that need discussion) or under-applies (agent asks about things that should be automatic).

**Minimum knowledge items for calibration:** ~6 (completion workflow table, session start protocol, session end protocol, signal prefixes table, mode declarations)

### Slot 4: Pressure over compensation

**Rule:** "When a system fails to surface knowledge, don't compensate by pasting the knowledge yourself. Note the gap. Let the system failure create pressure to improve."

**Why behavioral (not knowledge):** This fights the strongest helper prior — "I can fix this right now by just providing the information." Compensation feels immediately helpful but prevents systemic improvement. This is counter-instinctual.

**Calibration context (knowledge constraints):**
- "The test: Am I helping the system, or preventing it from learning?"
- Knowledge capture commands (kb quick decide/tried/constrain) — shows the ACTION for gap-noting
- Frustration protocol — teaches that frustration about system gaps is signal, not noise
- Observability architecture — teaches that Dylan sees the system through the dashboard, not through compensating orchestrators

**Why this calibration set:** The most dangerous failure mode is compensation feeling indistinguishable from helpfulness. The test question + knowledge capture commands give the agent a specific alternative action (note the gap via kb quick) instead of just saying "don't help." The frustration protocol teaches that the discomfort of NOT compensating is productive.

**Minimum knowledge items for calibration:** ~4 (test question, kb quick commands, frustration protocol pointer, observability architecture note)

## What Changes from v4

| Current v4 Slot | Proposed | Rationale |
|---|---|---|
| 1. Delegation | 1. Delegation (kept) | Strongest prior, most evidence |
| 2. Filter before presenting | 3. Filter + act-by-default (combined, kept) | Two sides of same coin |
| 3. Act by default | 3. (merged above) | Freed a slot |
| 4. Answer the question asked | **REMOVED** → reformulated as knowledge | Was knowledge wearing behavioral clothing |
| — | **2. Undefined behavior handler (NEW)** | Most critical missing piece per U-curve |
| — | **4. Pressure over compensation (NEW)** | Fights strongest helper prior |

### Why "answer the question asked" becomes knowledge

"Investigation findings ≠ design decision" is a fact about the work, not a suppression of a model prior. It can be stated as knowledge: "In this system, investigation produces evidence. Architect produces decisions. Don't collapse them." This survives dilution as a knowledge constraint and frees the behavioral slot for something that actually fights a prior.

### Why "anti-sycophancy" doesn't get a slot

The 5C experiment included anti-sycophancy. It showed zero signal — performance on frustration handling was identical across bare/5C/10C/current. This suggests either: (a) the model's sycophancy prior is weak enough that bare Claude handles it fine, or (b) the constraint form doesn't fire during the relevant cognitive phase. Either way, it's not worth a behavioral slot. If it becomes a problem, it's a coaching hook candidate (detect apology → inject "state your position").

## Minimum Calibration Budget

| Slot | Behavioral | Knowledge items | Total |
|---|---|---|---|
| 1. Delegation | 1 | ~8 | 9 |
| 2. Undefined behavior handler | 1 | ~5 | 6 |
| 3. Filter + act-by-default | 1 | ~6 | 7 |
| 4. Pressure over compensation | 1 | ~4 | 5 |
| **Total** | **4** | **~23** | **~27** |

This is within budget (4 behavioral + 50 knowledge) and leaves ~27 knowledge slots for the remaining v4 content (routing tables, vocabulary, tool ecosystem, etc.).

## Testing Recommendation

The critical validation is whether the undefined behavior handler prevents the U-curve regression. Create a variant:

- **4B-calibrated:** 4 behavioral norms (proposed) + their ~23 calibration items + remaining knowledge items
- Test against: bare, 5C (synthetic, no calibration), current v4

**Key scenario:** urgency-vs-process. If 4B-calibrated scores ≥70% (matching current) on `no-full-triage` while maintaining bare parity on `acknowledges-limits`, the architecture is validated.

**Secondary test:** Create a **4B-uncalibrated** variant (4 behavioral norms, NO knowledge items). If this scores <50% on urgency-vs-process, the matched-pair principle is confirmed — behavioral constraints alone are insufficient.

## Relationship to Existing Work

- **v4 skill (orch-go-w7xe):** This design REFINES v4, doesn't replace it. v4 already has the right structure (4 norms + knowledge). The change is slot reallocation: drop "answer the question asked," add "undefined behavior handler" and "pressure over compensation."
- **Hook infrastructure:** The 6 active hooks remain unchanged. They handle hard-behavioral constraints. This design is about the 4 judgment-behavioral slots that stay in prompt.
- **skillc test:** The validation method is skillc test with the proposed variant files.

## Implementation Sequence (for architect)

1. Reformulate "answer the question asked" as knowledge (add to routing/completion section)
2. Write undefined behavior handler norm + calibration context
3. Write pressure over compensation norm + calibration context
4. Merge filter + act-by-default into single combined norm
5. Create skillc test variant, run against bare and current v4
6. If validated, update v4 template in `.skillc/`
