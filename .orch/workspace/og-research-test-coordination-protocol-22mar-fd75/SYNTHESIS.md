# Session Synthesis

**Agent:** og-research-test-coordination-protocol-22mar-fd75
**Issue:** orch-go-nsb49
**Duration:** 2026-03-22
**Outcome:** success

---

## TLDR

Tested whether orch-go's four coordination primitives (Route, Sequence, Throttle, Align) are general to multi-agent coordination or orch-go-specific. They're general. Six independent external sources — MAST (Berkeley, 1642 traces), Google DeepMind (180 configs), McEntire's controlled experiment (28 tasks, 4 architectures), Anthropic's production multi-agent system, Getmaxim production patterns, and framework-specific failures (CrewAI, LangGraph, OpenAI/Claude Agent SDKs) — all converge on the same four structural requirements. No fifth primitive was needed. Align is the dominant failure mode (50% of MAST failures) and the most neglected by existing frameworks.

---

## Plain-Language Summary

Dylan hypothesized that orch-go's coordination architecture rests on four primitives: Route (agents don't collide), Sequence (work happens in order), Throttle (speed doesn't exceed verification capacity), and Align (agents share a model of what "correct" means). The question was whether these four are specific to orch-go or general to any multi-agent system.

I tested this by mapping documented failures from every major agent framework and three academic studies to the primitives. The mapping is clean — every failure fits one of the four, and no failure requires a fifth. The most striking evidence is McEntire's experiment: single-agent achieves 100%, hierarchical 64%, swarm 32%, and a pipeline with all four primitives broken achieves 0%. Success degrades exactly as the primitives predict.

The biggest finding for potential publication: **Align is the primitive nobody builds for.** Frameworks focus on Route (who does what) and Sequence (in what order). But 50% of documented failures are Align failures — agents working correctly by their own standards while building toward divergent definitions of "correct." This matches orch-go's core experimental finding that agents can communicate perfectly without coordinating.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md` - Full investigation with evidence mapping
- `.kb/models/coordination/probes/2026-03-22-external-framework-validation.md` - Probe documenting external validation

### Files Modified
- `.kb/models/coordination/model.md` - Added Four Coordination Primitives section, external validation evidence entry, new open questions

### Commits
- (pending)

---

## Evidence (What Was Observed)

- MAST taxonomy: 14 failure modes across 1642 traces map cleanly to 4 primitives. Align dominates (7/14 modes).
- McEntire experiment: Success rate tracks with primitive implementation (100% → 64% → 32% → 0%)
- Google DeepMind: Error amplification drops from 17.2x to 4.4x when adding Route+Sequence via centralized coordination
- CrewAI: Core failure is broken Route — manager can't delegate to correct worker (GitHub #4783)
- Anthropic: Production multi-agent system independently discovered and solved for all four primitives
- Getmaxim: Independently-derived 4 production failure categories map 1:1 to the 4 primitives
- autoresearch: Succeeds by constraining to N=1, trivially satisfying all primitives (degenerate case)

---

## Architectural Choices

No architectural choices — this was a research/validation task, no code changes.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md` - External validation of four coordination primitives
- `.kb/models/coordination/probes/2026-03-22-external-framework-validation.md` - Probe for coordination model

### Decisions Made
- The four primitives are general, not orch-go-specific — this is now a confirmed claim in the coordination model

### Constraints Discovered
- Align may need decomposition — it's the broadest category (50% of failures). Possible split: "state alignment" vs "goal alignment"
- Task-type dependency exists — DeepMind found financial reasoning favors centralized, web navigation favors decentralized

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, model updated)

### Follow-up (Strategic)

The finding that these primitives are general supports publishing them as a framework — not as orch-go marketing but as an independent contribution. The strongest angle is the Align insight: "communication doesn't produce coordination" backed by both orch-go's experiment (80 trials) and external evidence (MAST, McEntire, DeepMind).

This is a strategic decision for Dylan: product (build orch-go) vs contribution (publish the framework).

### If Close
- [x] All deliverables complete
- [x] Investigation file has Phase: Complete
- [x] Probe merged into model
- [x] Ready for `orch complete orch-go-nsb49`

---

## Unexplored Questions

- Whether Align should decompose into sub-primitives (state alignment vs goal alignment)
- Whether primitives have ordering dependencies (must Route precede Sequence?)
- How primitives interact with task type (DeepMind found strategy varies)
- Whether primitives apply to non-LLM multi-agent systems (robotics, distributed computing, human orgs)
- McEntire's "dysmemic pressure" concept and its relationship to Align drift over time

---

## Friction

Friction: tooling: WebFetch failed to extract article content from TDS (Towards Data Science) — articles are paywalled. Had to rely on search snippets and alternative sources for CrewAI manager-worker and multi-agent trap articles. Impact: ~5 min finding alternative sources.

---

## Session Metadata

**Skill:** research
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-research-test-coordination-protocol-22mar-fd75/`
**Investigation:** `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md`
**Beads:** `bd show orch-go-nsb49`
