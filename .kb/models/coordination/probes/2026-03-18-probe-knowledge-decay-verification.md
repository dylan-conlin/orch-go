# Probe: Knowledge Decay Verification — Coordination Model

**Date:** 2026-03-18
**Model:** `.kb/models/coordination/model.md`
**Trigger:** 999d since last probe (daemon knowledge decay detector)
**Probe Type:** Verification (claim accuracy audit)

## Claims Verified

### Claim 1: Communication is insufficient for coordination — CONFIRMED CURRENT

**Model says:** Context sharing and messaging produce 0% improvement over no coordination. 60/60 conflict in non-placement conditions.

**Current evidence:**
- Experiment data intact at `experiments/coordination-demo/redesign/results/20260310-174045/analysis.md`
- Raw results confirm: context-share 20/20 CONFLICT, messaging 20/20 CONFLICT, no-coord 20/20 CONFLICT
- No new experiments have been run since 2026-03-10 (no additional result directories)
- The codebase does NOT implement any communication-based coordination — consistent with the model's recommendation

**Verdict:** Confirmed. Data intact, claim accurate, no contradicting evidence.

### Claim 2: Structural placement prevents conflicts completely — CONFIRMED CURRENT

**Model says:** Explicit non-overlapping insertion point instructions produce 100% success (20/20).

**Current evidence:**
- Experiment data confirms: placement 20/20 SUCCESS (clean merge + tests pass) for both simple and complex tasks
- In production, orch-go avoids the coordination problem entirely by assigning agents to separate issues (different files by design) rather than using within-file placement instructions
- The daemon's `coordination.go` implements structural routing (hotspot extraction, architect escalation, project interleaving) — all structural mechanisms, not communication-based
- `orch swarm` spawns agents on separate issues with concurrency control — never two agents on the same file

**Verdict:** Confirmed. The production system implicitly validates this by avoiding same-file parallel edits altogether.

### Claim 3: Individual agent capability is not the bottleneck — CONFIRMED CURRENT

**Model says:** 160/160 agents scored 6/6 regardless of condition.

**Current evidence:**
- `analysis.md` confirms: every condition × task × agent combination shows 6.0/6 avg and 10/10 perfect scores
- No contradicting findings in any subsequent investigations

**Verdict:** Confirmed. Data intact.

### Claim 4: Coordination failure is task-complexity-independent — CONFIRMED CURRENT

**Model says:** Simple and complex tasks show identical coordination patterns.

**Current evidence:**
- Analysis confirms identical results for both `simple` and `complex` task types across all 4 conditions
- Duration differs (complex tasks take longer) but coordination outcomes are identical

**Verdict:** Confirmed. Data intact.

## Implications Audit

### "Multi-agent frameworks are fundamentally flawed" — PARTIALLY STALE

**Model says:** CrewAI, AutoGen, LangGraph assume agent-to-agent messaging solves coordination and will produce merge conflicts.

**Current state:**
- The framework landscape has evolved since 2026-03-10:
  - Investigation `2026-03-01-inv-agent-framework-behavioral-constraints-landscape.md` (pre-model) surveyed 8 frameworks and found the same structural gap
  - Investigation `2026-03-11-inv-investigation-competitive-landscape-structured-knowledge.md` positioned orch-go's structural approach as categorically different
  - AutoGen is now in "maintenance mode" (~50K stars), succeeded by Microsoft Agent Framework
  - Claude Agent SDK and OpenAI Agents SDK have emerged as new entrants
- The claim's core logic remains sound: messaging-based coordination does not prevent same-file merge conflicts
- However, the specific framework names are partially outdated (AutoGen → deprecated)

**Recommendation:** Update "AutoGen" reference to note its deprecation and mention newer frameworks (Claude Agent SDK, OpenAI Agents SDK) in the model's implications section. The fundamental claim holds.

### "Effective multi-agent coordination requires structural constraints" — CONFIRMED BY PRODUCTION

**Model says:** The harness must assign non-overlapping work regions.

**Production implementation:**
- `daemon/coordination.go:RouteIssueForSpawn()` — routes issues to different skills/models, never two agents to same file region
- `daemon/coordination.go:PrioritizeIssues()` — interleaves by project to avoid same-project collisions
- `orch swarm` — separate issues, concurrency control, no shared file edits
- `pkg/plan/` — phase-based with `DependsOn` for sequential ordering
- Exploration mode (`--explore`) — decomposes into subproblems, uses judge-based synthesis rather than agent-to-agent coordination

All production multi-agent patterns use structural separation, not messaging. The model's recommendation is fully implemented.

## Open Questions Status

| Question | Status |
|----------|--------|
| Does placement work when agents exceed natural insertion points? | **Still open** — not tested |
| Can iterative messaging (multi-round) produce different results? | **Still open** — not tested |
| Does stronger coordination instruction change behavior? | **Still open** — not tested |
| At what granularity does structural placement become impractical? | **Still open** — not tested |

No open questions have been addressed since model creation.

## Boundary Check

The model explicitly scopes to same-file parallel editing. The production system avoids this scenario entirely by routing agents to different issues/files. This means:
- The model's experimental findings remain valid
- The model's practical impact is primarily in informing the design choice to avoid same-file parallelism
- The model does NOT cover cross-file coordination failures (which could be a separate investigation area)

## Summary

**Overall verdict: Model is current. All 4 core claims confirmed by intact experiment data.**

Minor staleness:
1. Framework names in implications partially outdated (AutoGen deprecated)
2. No open questions have been pursued since creation
3. Model could note that orch-go's production architecture validates claims by avoiding the problem entirely

**Recommended model update:** Minor — update framework references, add note about production validation.
