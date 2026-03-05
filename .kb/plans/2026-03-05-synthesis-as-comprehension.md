## Summary (D.E.K.N.)

**Delta:** Orchestrator debriefs produce comprehension (Thread→Insight→Position) instead of event logs, with infrastructure that pulls toward synthesis the way daemon/bd ready/labels pull toward triage.

**Evidence:** Synthesis-as-comprehension investigation (2026-03-05): SYNTHESIZE has 1 line vs TRIAGE's 80 lines of infrastructure. 3 actual debriefs are pure event logs. Behavioral grammars Claim 3: situational pull overwhelms static reinforcement.

**Knowledge:** Comprehension requires three cognitive moves (Thread→Insight→Position) that parallel Three-Layer Reconnection at session scope. The debrief template is the highest-leverage change — it shapes behavior more than skill text.

**Next:** All phases implemented. Observe over 5+ sessions whether debriefs produce comprehension vs event logs. Check `debrief.quality` events in `orch stats` for longitudinal signal.

---

# Plan: Synthesis As Comprehension

**Date:** 2026-03-05
**Status:** Implemented (all 3 phases delivered 2026-03-05, observing for behavioral effect)
**Owner:** dylanconlin

**Extracted-From:** `.kb/investigations/2026-03-05-inv-design-orchestrator-synthesis-comprehension.md`

---

## Objective

Orchestrator sessions end with comprehension artifacts — insights connecting agent findings to meaning — not event logs listing what happened. The debrief becomes the synthesis moment, with infrastructure support matching what triage already has.

---

## Substrate Consulted

- **Models:** `behavioral-grammars` (Claim 3: infrastructure > instruction, Claim 5: self-detection failure), `orchestrator-session-lifecycle` (session phases, debrief patterns)
- **Decisions:** None superseded — this is new infrastructure
- **Guides:** None directly applicable
- **Constraints:** Skill constraint density limits (~+9 lines net for skill changes). Advisory gates only — no blocking at session end (competing checkpoint pressure).

---

## Decision Points

### Decision 1: Advisory vs blocking gate on debrief quality

**Context:** Should the summary-detection heuristic block session end or just warn?

**Options:**
- **A: Advisory (warning)** — Detect summary patterns, emit warning, don't block. Pros: no competing pressure with session checkpoints, collects data on failure rate. Cons: can be ignored.
- **B: Blocking gate** — Require comprehension signals before session end. Pros: enforces compliance. Cons: creates competing pressures, gets gamed with filler connectives.

**Recommendation:** A (Advisory) because behavioral grammars says measure the failure mode before designing enforcement. Need data on how often summaries occur before blocking is justified.

**Status:** Decided

---

## Phases

### Phase 1: Teach — Skill text + template restructure

**Goal:** Give orchestrators the cognitive moves for synthesis and a template that demands them
**Deliverables:**
- orch-go-7ldp4: Replace SYNTHESIZE definition with Thread→Insight→Position, add session-level synthesis subsection
- orch-go-msgjr: Rename "What Changed" → "What We Learned" (top position), add T→I→P coaching comments, reframe "What's Next" as strategic direction
**Exit criteria:** skillc test scores don't degrade; first debrief using new template contains connective language
**Depends on:** Nothing — ready to start

### Phase 2: Prompt — orch debrief enhancement

**Goal:** Make `orch debrief` actively prompt for comprehension and detect summary patterns
**Deliverables:**
- orch-go-d5rdt: Add `--learned` flag for "What We Learned" section, add `--quality` advisory heuristic (flags event-log patterns, missing connectives), print comprehension prompt after auto-populating facts
**Measurement:** `--quality` emits `debrief.quality` event to `events.jsonl` (pass/fail + triggered patterns). Enables `orch stats` to show comprehension rate over time.
**Exit criteria:** `orch debrief --quality` correctly flags pure event logs and passes comprehension paragraphs; events emitted to events.jsonl
**Depends on:** Phase 1 (template structure must match code expectations)

### Phase 3: Close the loop — orch orient feedback

**Goal:** Surface prior session's insight at session start so comprehension threads persist across sessions
**Deliverables:**
- orch-go-o79iw: Add "Last session's insight" section to `orch orient` reading most recent debrief's "What We Learned"
**Exit criteria:** `orch orient` shows prior comprehension; orchestrator references it in session work
**Depends on:** Phase 1 (template must have "What We Learned" section to read)

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Advisory vs blocking gate | behavioral-grammars Claim 3+5 | Yes — decided advisory |
| Skill text net change | Investigation measured +9 lines | Yes — within budget |
| Template structure | Investigation has exact template | Yes — copy-paste ready |

**Overall readiness:** Ready to execute

---

## Structured Uncertainty

**What's tested:**
- ✅ Current debriefs are pure event logs (verified across 3 sessions)
- ✅ SYNTHESIZE has no infrastructure pull (verified: 1 line vs 80 lines)
- ✅ Summary patterns are regex-detectable (verified: structural analysis of bad vs good examples)

**What's untested:**
- ⚠️ Whether teaching T→I→P actually changes synthesis behavior (needs skillc test + 3+ sessions observation)
- ⚠️ Whether restructured template produces comprehension or just longer summaries
- ⚠️ Whether advisory heuristic has acceptable false positive rate (<30%)

**What would change this plan:**
- If skillc test shows zero behavioral effect from Phase 1, shift entirely to infrastructure (skip skill text, focus on template + tool)
- If template produces "creative summaries" (longer event logs with filler connectives), escalate to conversational gate (Option C from investigation)

---

## Success Criteria

- [x] skillc test scores for orchestrator skill don't degrade after Phase 1 edits (7ldp4: 41/56 vs 38/56 baseline, +3)
- [x] Advisory gate correctly flags pure event logs as summaries (d5rdt: --quality detects empty_learned, action_verb_only, missing_connectives)
- [x] `debrief.quality` events tracked in events.jsonl (d5rdt: emits debrief.quality event)
- [x] `orch orient` surfaces prior session's insight (o79iw: FormatLastSessionInsight reads WhatWeLearned)
- [ ] Debrief "What We Learned" sections contain connective language ≥50% of the time across 5+ sessions (OBSERVING)
