## Summary (D.E.K.N.)

**Delta:** Add `--explore` spawn mode that enables parallel decomposition with judge synthesis for investigation and architect skills — analysis only, no code writes.

**Evidence:** Cursor First Proof Challenge (Mar 2026) — general-purpose harness solved research-grade math via decompose/parallelize/verify/iterate. Architect design: `.kb/investigations/2026-03-11-design-exploration-mode-decompose-parallelize-verify.md`. Thread: `.kb/threads/2026-03-11-exploration-mode-decompose-parallelize-verify.md`.

**Knowledge:** Cursor optimizes for compliance on bounded tasks (exploration harness). We optimize for coordination on living systems (maintenance harness). These aren't contradictory — they're sequential: exploration is upstream of enforcement. Explore freely in isolation, then existing gates filter promotion.

**Next:** Phase 1 — `--explore` flag and exploration orchestrator skill.

---

# Plan: Exploration Mode

**Date:** 2026-03-11
**Status:** Active
**Owner:** Dylan

**Extracted-From:** `.kb/investigations/2026-03-11-design-exploration-mode-decompose-parallelize-verify.md` (orch-go-fauck)

---

## Objective

Investigation and architect agents can decompose hard questions into parallel subproblems, judge sub-findings for quality, and synthesize compositional understanding — producing better analysis than single-agent single-pass for complex questions. No code writes in v1.

---

## Substrate Consulted

- **Models:** harness-engineering (maintenance vs exploration harness distinction), skill-content-transfer (soft harness dilution informs why prompts alone aren't enough for our context)
- **Decisions:** Architect design doc (orch-go-fauck) — spawn mode not skill, spawned orchestrator pattern, multi-dimensional judge verdicts
- **Constraints:** Claude Max rate limits bound breadth to 3 workers default. No code writes in v1.

---

## Decision Points

### Decision 1: Spawn mode vs new skill

**Context:** Is exploration a new skill or a modification of spawn?

**Recommendation:** Spawn mode (`--explore` flag). Skills own domain behavior, spawn owns execution topology. Decided in architect design.

**Status:** Decided

### Decision 2: Judge model selection

**Context:** Should the judge use a different model than workers to catch blind spots?

**Options:**
- **A: Same model** — Simpler, no cross-model complexity. Cons: same blind spots.
- **B: Different model** — Better coverage. Cons: adds model routing to v1.

**Recommendation:** A (same model) for v1. Cross-model judging is a Phase 3 experiment.

**Status:** Decided

### Decision 3: Iteration depth

**Context:** Should v1 support judge-triggered re-exploration (depth > 1)?

**Options:**
- **A: Single pass (depth=1)** — Simpler, still valuable as parallel investigation. Cons: misses Cursor's key insight (iteration found what single-pass couldn't).
- **B: Iterative (depth=N)** — Closer to Cursor's result. Cons: complex, expensive, hard to bound.

**Recommendation:** A for v1. B is Phase 3.

**Status:** Decided

---

## Phases

### Phase 1: Core spawn machinery

**Goal:** `orch spawn --explore investigation "question"` works end-to-end
**Deliverables:**
- `--explore` flag in spawn command
- Exploration orchestrator skill (decompose/fan-out/fan-in/judge/synthesize loop)
- Decomposition and synthesis prompt templates
- Cost bounding (breadth limit, rate-limit awareness)
**Exit criteria:** Can spawn an exploration investigation that decomposes into 3 workers, judges findings, and produces a unified analysis document.
**Depends on:** Nothing

### Phase 2: Judge skill

**Goal:** Structured quality evaluation of sub-findings
**Deliverables:**
- `exploration-judge` skill with grounding/consistency/coverage/relevance/actionability dimensions
- Structured verdict output (YAML)
- Contested findings surfaced as the most valuable output
**Exit criteria:** Judge produces structured verdicts that the synthesizer uses to weight findings.
**Depends on:** Phase 1

### Phase 3: Observability and measurement

**Goal:** Exploration mode has measurement surfaces per harness-engineering model
**Deliverables:**
- Dashboard integration showing exploration tree
- Events: exploration.decomposed, exploration.judged, exploration.synthesized
- Metrics: subproblem quality, judge agreement, synthesis coherence
**Exit criteria:** Can see an exploration run in the dashboard and evaluate its quality from events.
**Depends on:** Phase 1, Phase 2

### Phase 4: Iteration (future)

**Goal:** Judge-triggered re-exploration for gaps
**Deliverables:**
- `--explore-depth N` flag
- Judge gap detection triggers additional subproblems
- Cross-model judge experiment
**Exit criteria:** Exploration can iterate until judge finds no remaining gaps or depth limit hit.
**Depends on:** Phase 2, Phase 3

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Spawn mode vs skill | Architect design decided | Yes |
| Judge model | Deferred to Phase 3 | Yes (default: same model) |
| Iteration depth | Deferred to Phase 4 | Yes (default: single pass) |

**Overall readiness:** Ready to execute Phase 1-2.

---

## Structured Uncertainty

**What's tested:**
- Spawned orchestrator pattern works (existing infrastructure)
- Parallel spawn with fan-in works (orch spawn + orch wait)
- Investigation/architect skills produce analysis without code writes

**What's untested:**
- Whether decomposition quality is good enough to be useful (will the decomposer identify genuinely independent subproblems?)
- Whether judge verdicts improve synthesis quality vs just concatenating sub-findings
- Whether 3-breadth is sufficient or exploration needs more parallelism to show value

**What would change this plan:**
- If decomposition quality is poor, may need structured decomposition templates per domain rather than a generic approach
- If Claude Max rate limits make even breadth-3 impractical, may need to serialize workers instead of parallelize

---

## Success Criteria

- [ ] `orch spawn --explore investigation "question"` produces analysis that's measurably better than single-agent investigation on the same question
- [ ] Exploration runs complete within rate-limit budget (no account exhaustion)
- [ ] Judge identifies at least one contested finding or gap per exploration run
- [ ] Zero code writes from exploration agents (isolation holds)
