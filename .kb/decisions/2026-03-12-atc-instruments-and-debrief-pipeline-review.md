# Decision: ATC Instruments and Debrief Pipeline — Architect Review

**Date:** 2026-03-12
**Status:** Accepted
**Deciders:** Architect review (orch-go-v8u1f)
**Extends:** `.kb/decisions/2026-02-28-atc-not-conductor-orchestrator-reframe.md`
**Prior Work:** `.kb/investigations/2026-02-28-inv-atc-lens-feature-audit.md`, `.kb/investigations/2026-03-05-inv-design-orchestrator-synthesis-comprehension.md`, `.kb/plans/2026-03-05-synthesis-as-comprehension.md`

## Decision

Accept the current ATC reframe and debrief pipeline as **structurally sound with two remaining gaps**: (1) completion summaries flooding the "What We Learned" section dilute comprehension signal, and (2) the debrief→briefing feedback loop (SYNTHESIS findings flowing back into future spawn context) remains manual. The monitoring reframe (conductor → ATC instrument naming) is low-priority cosmetic work that should not be invested in further — the architecture already behaves like ATC; renaming things won't change behavior.

## Assessment: What Was Built (Feb 28 – Mar 12)

### ATC Reframe (Feb 28)

The decision document and feature audit were thorough. The system was already ~70% ATC-shaped before the metaphor was named. The reframe's primary value was diagnostic — it identified the post-flight debrief as the biggest investment gap (Finding 3.1).

**Verdict: Complete.** The mental model is established. The audit's categorization (ATC/conductor/under-invested) is accurate. No further reframing work needed.

### Debrief Pipeline (Mar 5 – Mar 12)

All three phases from the synthesis-as-comprehension plan shipped:

| Phase | Status | What Shipped |
|-------|--------|-------------|
| Phase 1: Teach | Done | SYNTHESIZE definition replaced with Thread→Insight→Position. Template restructured ("What Changed" → "What We Learned" at top). |
| Phase 2: Prompt | Done | `orch debrief --learned` flag, `--quality` advisory heuristic, comprehension prompt after auto-population. `debrief.quality` events emitted. |
| Phase 3: Close loop | Done | `orch orient` surfaces last session's insight from most recent debrief's "What We Learned" section. |
| Batch synthesis | Done | `orch review synthesize` produces cross-agent synthesis (WHAT WE NOW KNOW / NEXT ACTIONS / OPEN QUESTIONS / CONNECTIONS). Shipped Mar 11. |

**Verdict: Infrastructure exists.** The "no infrastructure pull toward synthesis" gap identified in the investigation is now closed — there are CLI commands, templates, advisory gates, and feedback loops. The question shifts from "does infrastructure exist?" to "does it produce comprehension?"

## Assessment: What's Working

1. **Template restructure is effective.** The Mar 10 debrief's first 5 entries (lines 14-18) show genuine comprehension with connective language: "lost trust in the system's self-assessment after watching it escalate observations into theory into 'physics'", "each piece creates demand for the next." These are insights, not event logs.

2. **Advisory quality gate exists.** `--quality` detects empty_learned, action_verb_only, missing_connectives patterns. Events tracked in events.jsonl for longitudinal measurement.

3. **Cross-session continuity.** `orch orient` showing last session's insight closes the briefing↔debrief loop at the session level.

4. **Batch synthesis.** `orch review synthesize` produces cross-agent connections that individual SYNTHESIS.md files can't show. This is the "post-flight debrief across a fleet" that was entirely missing.

## Assessment: Remaining Gaps

### Gap 1: Completion summaries flood "What We Learned" (HIGH)

**Evidence:** The Mar 10 debrief has 5 genuine insights (lines 14-18) and 38 auto-populated completion summaries (lines 19-55). The completion reasons — "Extracted client.go (1449→1040)", "Fixed kb reflect --type stale" — are event-log entries that ended up in the comprehension section because `collectDebriefLearned()` merges `CollectWhatWeLearned(events)` (auto-generated from `agent.completed` events) with manually-authored `--learned` content.

**Impact:** The advisory quality gate becomes useless because the section always has content (auto-populated) and always has action verbs (completion reasons start with "Extracted", "Fixed", "Implemented"). The heuristic can't distinguish manual insights from auto-populated summaries.

**Root cause:** `debrief_cmd.go:262-282` — `collectDebriefLearned()` appends completion reasons from events to the "What We Learned" section. This was a design choice to avoid empty sections, but it undermines the section's purpose.

**Recommendation:** Separate auto-populated completion summaries from manual insights. Two approaches:

- **Option A (minimal):** Move auto-populated completion reasons to "What Happened" section where they belong. "What We Learned" becomes manual-only (`--learned` flag + thread entries). If it's empty, the quality gate fires — which is the intended behavior.
- **Option B (richer):** Keep completion reasons in "What We Learned" but visually separate them (e.g., under a "Completion Notes" sub-heading). Manual insights go under "Insights" sub-heading. Quality gate checks only the "Insights" portion.

**Recommendation: Option A.** Completion reasons are events (what happened), not insights (what we learned). Moving them to "What Happened" is the correct ATC interpretation: the flight data recorder captures events; the debrief captures understanding.

### Gap 2: No automated SYNTHESIS → spawn context feedback (MEDIUM)

**What exists:** SYNTHESIS.md → `orch review` (manual read) → orchestrator decides what to promote → manual `kb quick` or decision creation. Future agents get better context only if the orchestrator manually promotes findings.

**What's missing:** No automated path from "Agent X discovered constraint Y" → "Future agents spawned on related work see constraint Y." The debrief→orient loop works at session scope (orchestrator comprehension), but not at agent scope (spawn context enrichment).

**Impact:** This is the "briefing quality depends on debrief quality" cycle the ATC audit identified. Currently the cycle requires manual orchestrator intervention at every step.

**Recommendation:** This is a known gap but NOT urgent. The current manual process works at current scale (~5-15 agents/day). Automated promotion risks the "closed loop" problem the Mar 10 debrief identified — AI agents reinforcing their own conclusions without external validation. The manual step (orchestrator reviews and decides what to promote) is a feature, not a bug, at this stage.

**When to revisit:** If agent throughput exceeds ~30/day, or if the same constraint is re-discovered by 3+ agents (pattern detection in `orch reflect` already detects this via synthesis checkpoint advisory).

### Gap 3: Monitoring command reframe (LOW — DO NOT INVEST)

The ATC audit recommended reframing `orch tail` → "flight data recorder", `orch status` → "radar display", `orch question` → "distress signal" in documentation and help text. Also recommended deprecating `orch send`, `orch monitor`, `orch patterns`.

**Assessment:** This is cosmetic. The system's behavior is already ATC-shaped. Renaming commands or updating help text does not change how the system works. The behavioral shift (orchestrator stops micro-managing) comes from the skill text and decision authority guide, not from renaming `orch status` to `orch radar`.

**Deprecation candidates:**
- `orch send` — Already rarely used. Keep as escape hatch but don't invest.
- `orch monitor` — Daemon handles completion processing. SSE watching is niche debugging. Keep, don't invest.
- `orch patterns` — Lowest usage of any command. Candidate for removal in a future cleanup pass.

**Recommendation:** No action. The audit's categorization is correct and useful as a diagnostic lens, but acting on it (renaming, deprecating) has low ROI compared to fixing Gap 1.

## What This Changes

### Immediate Action (Gap 1 fix)

Move auto-populated completion reasons from "What We Learned" to "What Happened" in `collectDebriefLearned()`. This is a ~20-line change in `debrief_cmd.go` + corresponding update in `pkg/debrief/debrief.go`.

**Effect:** "What We Learned" becomes meaningful signal — either the orchestrator provides manual insights (via `--learned` or thread entries) or it's empty, which fires the quality gate. No more drowning 5 insights in 38 completion summaries.

### No Action

- Monitoring command reframing (cosmetic, low ROI)
- Automated SYNTHESIS → spawn context promotion (manual step is a feature at current scale)
- Command deprecation (`send`, `monitor`, `patterns` — keep as-is)

## Relationship to Existing Decisions

| Decision | How This Review Extends It |
|----------|---------------------------|
| ATC-Not-Conductor Reframe (2026-02-28) | Confirms the reframe is complete and the diagnostic value has been extracted. No further investment in the metaphor itself. |
| Synthesis-as-Comprehension Plan (2026-03-05) | All 3 phases shipped. Identifies Gap 1 (completion summary flooding) as the next iteration needed for the debrief to produce actual comprehension. |
| No Code Review Gate (2026-02-25) | Consistent — this review does not propose adding review gates. The quality gate remains advisory. |
| Five Design Principles for Automation Legibility (2026-03-08) | The "quiet for normal, color for anomalies" principle from High-Performance HMI supports not investing in monitoring reframing — the dashboard already follows this pattern. |

## Success Criteria

- [ ] Gap 1 fix shipped: completion reasons moved from "What We Learned" to "What Happened"
- [ ] Quality gate fires on debriefs with no manual insights (empty "What We Learned")
- [ ] Next 3 debriefs show cleaner separation between insights and events
- [ ] No regressions in `orch debrief --quality` heuristic accuracy
