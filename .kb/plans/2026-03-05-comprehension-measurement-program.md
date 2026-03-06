## Summary (D.E.K.N.)

**Delta:** Establish a scientific measurement program that can distinguish agent comprehension from throughput — the gap the current detection engine cannot close.

**Evidence:** Behavioral grammars model (claims 1-7), orchestrator skill model (knowledge vs enforcement split), injection-level experiment (density > injection, intent-only limitation), fabrication U-curve (process over-application at intermediate density), existing scenario suite (7 compliance scenarios, 1 synthesis scenario), thread "Throughput completions vs comprehension completions."

**Knowledge:** Comprehension is a latent variable. You can't observe it directly, only through proxies. The current proxies (keyword presence) measure behavioral compliance, not cognitive orientation. But contrastive scenario design — situations where throughput and comprehension produce observably different outputs — can make the latent variable visible without requiring a new measurement paradigm.

**Next:** Write contradiction and red-herring scenarios (Phase 1). These work with the current detection engine and directly test the throughput-vs-comprehension distinction.

---

# Plan: Comprehension Measurement Program

**Date:** 2026-03-05
**Status:** Phase 2 complete, Phase 3 pending
**Owner:** Dylan

**Extracted-From:** Session analysis of orchestrator skill throughput-vs-comprehension problem (Mar 5, 2026)
**Thread:** "Throughput completions vs comprehension completions"

---

## Objective

Build a measurement system that can detect whether an orchestrator agent is thinking (connecting findings, detecting patterns, surfacing meaning) or draining a queue (spawn, complete, close, next). Success = automated scores that correlate with human judgment of comprehension quality at r > 0.6 on a calibration set.

---

## Substrate Consulted

- **Models:** Behavioral grammars (claims 1-7, especially claim 5: grammars can't detect their own failures), orchestrator skill model (knowledge transfer vs enforcement)
- **Decisions:** Stance items are non-removable (kb-3f85c9), skill changes require --runs 3 (kb-3a8b09), deploy gated on behavioral baseline (orch-go-jxpe7, closed)
- **Guides:** Experiment skill procedure (hypothesis → design → trials → analysis → structured uncertainty)
- **Constraints:** Detection engine supports only `contains X|Y|Z` and `does not contain X|Y|Z`. No AND logic, length checks, or semantic evaluation. LLM-as-judge creates closed evaluation loop (orchestrator skill model constraint).

---

## Decision Points

### Decision 1: Scorer extension scope

**Context:** Contrastive scenarios (Phase 1) work within the current grammar. But calibration (Phase 2) may reveal that keyword proxies don't correlate with comprehension regardless of scenario design. If so, we need scorer extensions.

**Options:**
- **A: Minimal extensions (length + AND)** — Add `response length > N` and `contains-all X|Y|Z`. Low effort, modest measurement improvement. Pros: ships fast, backwards compatible. Cons: still keyword-level, may not correlate.
- **B: Structural analysis** — Add cross-reference density, novel vocabulary detection, relational language scoring. High effort, potentially transformative. Pros: measures connection-making directly. Cons: complex, may over-engineer before knowing if it matters.
- **C: LLM-as-judge (constrained)** — Use a different model family (e.g., Gemini) to rate responses on a rubric. Pros: directly measures what we care about. Cons: violates current "no LLM-as-judge" constraint, introduces model-dependency, expensive.

**Recommendation:** A first, gate on calibration results. If keyword proxies show r < 0.4 with human ratings, escalate to B. C is a last resort — only if the fundamental premise (contrastive scenarios make comprehension visible through keywords) is falsified.

**Status:** Partially resolved by Phase 2 calibration

**Phase 2 update (Mar 6):** Calibration shows the bottleneck is indicator vocabulary per-scenario, not grammar expressiveness. Overall rho=0.637 passes gate — the keyword proxy approach works. But S11 (rho=0.141) shows indicators are broken for absence-detection scenarios. The problem isn't that we need AND logic or length checks (Option A) or structural analysis (Option B) — it's that S11's indicator vocabulary doesn't capture what humans judge as comprehension for that scenario type. **Priority shifts from scorer extensions to per-scenario indicator redesign.** S09/S13 indicators are validated and need no changes. S11 indicators need complete vocabulary redesign before any scorer extensions would matter.

### Decision 2: Multi-turn testing approach

**Context:** Single-turn `--print` mode measures intent, not action. `--test-mode full` enables tool execution but is slower and more expensive. Multi-turn comprehension decay (Phase 3) may require either multi-turn scenario support in skillc or a custom harness.

**Options:**
- **A: Extend skillc test for multi-turn with state** — Add scenario support for N-turn conversations with intermediate completions injected. Pros: reusable, integrates with existing infrastructure. Cons: significant skillc engineering.
- **B: Custom experiment harness** — One-off script that runs a multi-turn conversation, injecting completions at turns 1, 3, 5, 7. Pros: fast to build, tailored to this experiment. Cons: throwaway work.

**Recommendation:** Defer until Phase 1-2 validate the contrastive approach. If contrastive scenarios work in single-turn, multi-turn may not be needed.

**Status:** Deferred

---

## Phases

### Phase 1: Contrastive Scenario Authoring

**Goal:** Write scenarios where throughput and comprehension produce observably different outputs, testable with current detection engine.

**Deliverables:**
- Scenario 09: Contradiction (two agents with conflicting findings) — `orch-go-rahs1`
- Scenario 10: Red herring (obvious action + subtle signal) — `orch-go-h7vka`
- Scenario 08 validated (synthesis-after-completions already written) — `orch-go-8ugfj`

**Exit criteria:**
- All 3 scenarios pass basic sanity: bare baseline ≠ skill variant on at least one indicator (N=3 runs)
- At least one scenario discriminates comprehension from throughput (skill > bare by ≥ 2 indicators)

**Depends on:** Nothing — works with current infrastructure

**Beads:** orch-go-rahs1 (closed), orch-go-h7vka (closed), orch-go-8ugfj (closed)

**Result:** Phase 1 complete. Scenarios 08 and 09v2 are discriminating. Scenario 10 hits ceiling — needs redesign (signal spanning multiple completions). Key finding: scenario 09v2 discriminates stance from knowledge (first empirical evidence). Investigation: `.kb/investigations/2026-03-05-inv-experiment-comprehension-calibration-contrastive-scenarios.md`

### Phase 2: Human Calibration

**Goal:** Validate whether automated scores correlate with human judgment of comprehension quality.

**Deliverables:**
- 20-30 responses to synthesis-requiring scenarios (generated across variants: bare, skill-without-stance, skill-with-stance)
- Dylan's blind 1-5 comprehension ratings for each response
- Correlation analysis: human rating vs automated indicator scores
- Decision: which automated proxies to keep, which are noise

**Exit criteria:**
- At least one automated proxy correlates with human ratings at r > 0.6
- OR: no proxies correlate, triggering Decision 1 escalation to option B

**Depends on:** Phase 1 (scenarios must exist to generate responses)

**Beads:** orch-go-54y23

**Result:** Phase 2 complete. 24 blind-rated responses across 4 scenarios × 3 variants. Overall Spearman rho=0.637 (p=0.0001) passes r>0.6 gate — automated proxies correlate with human judgment. Per-scenario: S09=0.980, S13=0.894, S12=0.747 (all validated), S11=0.141 (broken — indicators uncorrelated with human judgment, need vocabulary redesign). Scorer vocabulary bias toward skill-enhanced responses: 3/4 biggest disagreements on bare variants. Variant means: bare=2.0, without-stance=4.2, with-stance=4.1 — stance lift is scenario-specific, not universal. Evidence: `evidence/2026-03-06-human-calibration/`

### Phase 3: Scorer Extensions (conditional)

**Goal:** Extend detection grammar based on calibration findings.

**Deliverables:**
- Response-length detection (`response length > N`) — `orch-go-osad3`
- AND logic (`response contains-all X|Y|Z`) — `orch-go-co965`
- Possibly: structural extensions if calibration shows keyword proxies insufficient

**Exit criteria:**
- Extended scorer produces higher correlation with human ratings than keyword-only scorer
- OR: extensions don't help, meaning the contrastive scenario design is doing the heavy lifting (good outcome — means the scenarios are the innovation, not the scorer)

**Depends on:** Phase 2 calibration results informing what to build

**Beads:** orch-go-osad3, orch-go-co965

### Phase 4: Multi-turn Comprehension Decay (conditional)

**Goal:** Test whether synthesis degrades across session turns.

**Deliverables:**
- Multi-turn experiment: inject completions at turns 1, 3, 5, 7, measure synthesis quality at each
- Finding: does stance degrade, hold steady, or vary with something else?

**Exit criteria:**
- Clear signal on decay vs stability
- OR: multi-turn measurement doesn't add information beyond single-turn (good — simplifies the program)

**Depends on:** Phases 1-2 validated, multi-turn testing approach decided (Decision 2)

**Beads:** orch-go-77mle

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Scorer extension scope | Calibration data collected (Phase 2 complete) | Partially resolved — bottleneck is indicator vocabulary, not grammar |
| Multi-turn approach | Contrastive scenario validation complete (Phase 1-2 done) | Ready to decide |

**Overall readiness:** Phases 1-2 complete. Phase 3 (scorer extensions) re-scoped: priority is S11 indicator vocabulary redesign, not grammar extensions. Phase 4 (multi-turn) ready to plan.

---

## Structured Uncertainty

**What's tested:**
- Keyword detection measures behavioral compliance (extensive evidence from injection-level, fabrication experiments)
- Stance items affect agent orientation (throughput-vs-comprehension observed Mar 5, restored to v4 baseline)
- Single-turn `--print` measures intent, not action (identity-action gap, Feb 24 probe)
- Automated proxies correlate with human comprehension ratings at rho=0.637 overall (Phase 2, Mar 6, N=24)
- Per-scenario indicator validity varies dramatically: S09=0.980, S13=0.894, S12=0.747, S11=0.141
- Scorer vocabulary bias toward skill-enhanced responses (3/4 biggest disagreements on bare variants)
- Stance lift is scenario-specific, not universal (aggregate without-stance=4.2, with-stance=4.1)

**What's untested:**
- Whether redesigned S11 indicators can achieve acceptable correlation with human judgment
- Whether comprehension degrades across turns or is stable once primed
- Whether scorer extensions improve discrimination beyond better scenario design (Phase 2 suggests bottleneck is indicator vocabulary, not grammar)

**What would change this plan:**
- If contrastive scenarios show bare = skill on comprehension indicators → the detection grammar fundamentally can't measure this, need LLM-as-judge or structural analysis
- If human calibration shows r < 0.3 for all automated proxies → keyword-level measurement is a dead end for comprehension
- If stance items don't affect contrastive scenario scores → the throughput-vs-comprehension problem isn't about stance (something else is driving it)

---

## Success Criteria

- [ ] 3+ contrastive scenarios that discriminate comprehension from throughput (skill > bare)
- [ ] At least one automated proxy correlating with human comprehension ratings at r > 0.6
- [ ] Skill regression test suite that catches throughput-machine degradation before deploy
- [ ] Clear decision on scorer extension scope (minimal vs structural vs LLM-as-judge)
