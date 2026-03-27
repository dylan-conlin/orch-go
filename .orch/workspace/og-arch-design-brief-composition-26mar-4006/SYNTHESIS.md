# Session Synthesis

**Agent:** og-arch-design-brief-composition-26mar-4006
**Issue:** orch-go-c5ha1
**Duration:** 2026-03-26
**Outcome:** success

---

## TLDR

Designed the brief composition layer — the missing step between individual brief atoms and thread-level understanding. Composition is an orchestrator-session act (not daemon automation), triggered at session start when 5+ unprocessed briefs accumulate, producing digest artifacts with draft thread proposals. The design explicitly prevents the 6th instance of the system conflating "I processed this" with "I understood this."

---

## Plain-Language Summary

Agents produce briefs when they finish work — half-page artifacts capturing what they learned, built, or decided. These briefs accumulate in `.kb/briefs/`. The problem is that the most valuable insights live in the relationships BETWEEN briefs, not in any single one. Today's evidence: 8 briefs independently discovered the same identity gap, 5 share an epistemic dishonesty pattern, but no brief names these cross-cutting patterns.

The composition layer sits between brief generation and human reading. When 5+ briefs accumulate and Dylan starts a session, the orchestrator clusters them, harvests their unresolved questions, and produces a "digest" — a document that says "here are the patterns I see, here are the open questions, and here are some thread proposals." The digest is explicitly labeled as unverified clustering, not understanding. Dylan engages with the clusters in conversation, decides what becomes thread-level knowledge, and the human judgment step is preserved.

The design is grounded in the signal-to-design loop model (composition is Stage 3 — clustering), constrained by Understanding Through Engagement (synthesis can't be automated away), and warned by the epistemic dishonesty thread (clustering ≠ understanding).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-design-brief-composition-layer.md` — Full architect investigation with 6 findings, 5 fork navigations, 3-phase implementation recommendation
- `.kb/global/models/signal-to-design-loop/probes/2026-03-26-probe-brief-composition-as-clustering-stage-instance.md` — Probe confirming composition as Instance 6 of the loop, extending the clustering stage description
- `.orch/workspace/og-arch-design-brief-composition-26mar-4006/VERIFICATION_SPEC.yaml`
- `.orch/workspace/og-arch-design-brief-composition-26mar-4006/SYNTHESIS.md`
- `.orch/workspace/og-arch-design-brief-composition-26mar-4006/BRIEF.md`

### Files Modified
- `.kb/global/models/signal-to-design-loop.md` — Added Instance 6 (Brief Composition), extended clustering stage "Works When" column, new evolution entry

### Commits
- Pending — will commit all artifacts together

---

## Evidence (What Was Observed)

- 61 briefs exist in `.kb/briefs/`, totaling 943 lines (~15 lines average per brief)
- 4 cross-cutting clusters identified manually in prior orchestrator session: identity gap (8), epistemic dishonesty (5), model-routing (6), production-exceeding-comprehension (3)
- Comprehension queue threshold is 5 (`DefaultComprehensionThreshold` in `pkg/daemon/comprehension_queue.go:34`)
- Orient command already collects threads, models, ready work, health at session start (`cmd/orch/orient_cmd.go`)
- Signal-to-design loop model has 5 existing instances, composition fits as Instance 6
- Tension sections cluster better than Frame sections — verified by comparing keyword overlap across cluster members

---

## Architectural Choices

### Choice 1: Orchestrator-session composition, not daemon automation
- **What I chose:** Composition happens in orchestrator session with Dylan present
- **What I rejected:** Daemon second pass (automated), dedicated composition agent (spawned)
- **Why:** Understanding Through Engagement principle — synthesis requires human vantage. Daemon has no context about Dylan's thinking. A composition agent removes the reactive moment where insight happens.
- **Risk accepted:** Composition doesn't happen between sessions; clusters may be stale by next session

### Choice 2: Digest as new artifact type, not direct thread modification
- **What I chose:** Digest in `.kb/digests/` with draft thread proposals (proposals, not actions)
- **What I rejected:** Direct thread writes (closed-loop risk), cluster summaries (false comprehension)
- **Why:** Epistemic honesty — clustering ≠ understanding. Comprehension artifacts thread: provoke, don't replace.
- **Risk accepted:** Extra artifact type adds complexity; digests may go unread

### Choice 3: Composition does NOT drain the comprehension queue
- **What I chose:** Briefs retain their comprehension state; digest is a lens on the queue, not a replacement
- **What I rejected:** Having composition transition briefs to "processed" (would conflate clustering with comprehension)
- **Why:** Would be the 6th instance of the system treating "I processed this" as "I understood this"
- **Risk accepted:** Queue stays full; spawning stays throttled until Dylan actually reads briefs

---

## Knowledge (What Was Learned)

### New Findings
- **Tension sections cluster better than Frame sections.** Unresolved questions converge when briefs share an underlying gap; narratives diverge because each brief has different context. This extends the signal-to-design loop model's clustering resolution hierarchy for structured signal types.
- **The Stage 3/4 boundary is a design parameter.** The signal-to-design loop model treats clustering and synthesis as sequential stages. The composition design reveals that how much of Stage 4 to include in automation is a choice — a function of trust in the automated actor and value of human participation.

### Decisions Made
- Composition actor: orchestrator session (not daemon, not dedicated agent)
- Trigger: accumulation threshold (5+ unprocessed briefs) at session start
- Output: digest artifact with epistemic label, cluster rationale, tension harvest, draft thread proposals
- Queue interaction: composition does not modify comprehension queue state
- Closed-loop mitigation: three-layer epistemic labeling

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

Key outcomes:
- All 5 design forks navigated with substrate-traced recommendations
- Probe confirms composition as Instance 6 of signal-to-design loop
- Model updated with new instance and extended clustering description
- 4 composition claims defined with verification methods

---

## Next (What Should Happen)

**Recommendation:** close — design complete, implementation issues to be created

### Implementation Phases (3 issues recommended)

1. **Phase 1:** `orch compose` CLI + digest artifact format — `cmd/orch/compose_cmd.go`, `.orch/templates/DIGEST.md`, brief parser
2. **Phase 2:** Orient integration — session-start hook calls compose when threshold met, injects digest into context
3. **Phase 3:** Thread proposal acceptance flow — `orch compose --accept` command, tension orphan surfacing

---

## Unexplored Questions

- **Keyword vs semantic clustering:** Started with keyword overlap; may need LLM-powered semantic clustering if false positives are too high. Requires API call (cost/latency trade-off).
- **Brief metadata tags:** Adding a `relates_to:` field to the brief template would improve clustering resolution from lexical proximity to explicit tag. But adds capture friction.
- **Digest UI presentation:** Should digests have a dashboard route, or is orient-only sufficient?

---

## Friction

Friction: none — smooth session. Substrate (principles, models, threads) was well-organized and coherent; design forks resolved cleanly against it.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-brief-composition-26mar-4006/`
**Investigation:** `.kb/investigations/2026-03-26-design-brief-composition-layer.md`
**Beads:** `bd show orch-go-c5ha1`
