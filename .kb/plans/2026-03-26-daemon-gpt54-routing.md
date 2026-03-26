## Summary (D.E.K.N.)

**Delta:** This plan turns GPT-5.4 from a manual benchmark result into a bounded daemon routing lane with explicit defaults, observability, and recovery.

**Evidence:** It is based on the Mar 26 benchmark, current daemon routing code, compliance-style config patterns, and historical GPT stall evidence refined for GPT-5.4-era prompts.

**Knowledge:** The key design move is to route by capability class and complexity, not by a universal provider preference.

**Next:** Implement the route object in `orch-go-ckddz`, then add config/observability in `orch-go-r7avo`, then wire bounded promotion in `orch-go-kdyh6` before closing `orch-go-xi8tc`.

---

# Plan: Daemon Gpt54 Routing

**Date:** 2026-03-26
**Status:** Ready
**Owner:** orch-go-3k1yo

**Extracted-From:** `.kb/investigations/2026-03-26-inv-design-daemon-route-tasks-gpt.md`
**Supersedes:** None
**Superseded-By:** None

---

## Objective

The daemon can make a legible model choice per issue: keep Opus for reasoning-heavy or high-complexity work, use GPT-5.4 only for eligible `feature-impl` overflow, and promote GPT failures to Opus once without loops. Humans can see the route and the reason from preview/review artifacts, and the behavior is covered by tests.

---

## Substrate Consulted

- **Models:** `.kb/models/daemon-autonomous-operation/model.md` - Current daemon model still describes skill-based routing and records GPT failure history.
- **Models:** `.kb/models/daemon-autonomous-operation/claims.yaml` - DAO-13 refines the GPT question: protocol compliance, silent-death frequency, and scope control matter more than prompt size now.
- **Decisions:** `kb context "daemon model routing"` surfaced the existing sparse explicit-model policy and the decision to let downstream resolution honor defaults when no explicit override exists.
- **Guides:** `.kb/guides/model-selection.md` - GPT-5.4 is the highest-context OpenAI option; Opus remains the default reasoning model.
- **Constraints:** Provenance, Session Amnesia, and defect classes 2/5/6/7 require one canonical route derivation plus bounded recovery.

---

## Decision Points

### Decision 1: What work is eligible for GPT-5.4 by default?

**Context:** GPT-5.4 now has positive benchmark evidence, but only for a narrow slice of work.

**Options:**
- **A: Bounded overflow lane** - GPT-5.4 only for `feature-impl` work labeled `effort:small` or `effort:medium`, with Opus staying default elsewhere. Pros: matches evidence, easy to explain, safer. Cons: leaves some GPT capacity unused.
- **B: Provider-wide GPT implementation default** - Route all `feature-impl` work to GPT-5.4. Pros: maximizes OpenAI path usage. Cons: overreaches current data and amplifies scope-control risk.

**Recommendation:** Option A because the current benchmark only clears GPT-5.4 for bounded implementation overflow, not default implementation or reasoning work.

**Status:** Decided

---

### Decision 2: Where should routing policy live?

**Context:** The daemon already has skill inference, route rewrites, and config resolution. The new policy must not create contradictory authority.

**Options:**
- **A: First-class daemon route object** - Add a route-policy step in daemon coordination and keep `pkg/spawn/resolve` as the lower-level backend/model resolver. Pros: one canonical daemon decision, fits current architecture. Cons: some refactoring.
- **B: Keep skill inference and add GPT special cases in `SpawnWork`** - Patch model choice near command execution. Pros: smaller diff. Cons: hidden policy, duplicate logic, harder observability.

**Recommendation:** Option A because the daemon already centralizes route rewriting in coordination code and the same pattern avoids defect class 5.

**Status:** Decided

---

### Decision 3: How should GPT failures recover?

**Context:** GPT-5.4 is viable enough to use, but not reliable enough to repeat blindly on every failure.

**Options:**
- **A: Promote once to Opus after classified GPT failure** - Retry via Opus when the failure matches empty execution / early silent death / repeat GPT failure. Pros: bounded, safe, observable. Cons: requires classification work.
- **B: Re-run GPT-5.4 with the same route** - Keep the cheaper lane until a human intervenes. Pros: simpler. Cons: duplicate-action and silent-loop risk.

**Recommendation:** Option A because the benchmark and retry design work justify one bounded recovery step, not indefinite optimism.

**Status:** Decided

---

## Phases

### Phase 1: Build the route object

**Goal:** Replace skill-only model inference with issue-aware route selection.
**Deliverables:** `orch-go-ckddz`, tests for skill/effort/fallback routing, route-reason field.
**Exit criteria:** Daemon can choose Opus vs GPT-5.4 from a single policy object and callers no longer infer model from skill alone.
**Depends on:** None.

### Phase 2: Add config and observability

**Goal:** Make routing legible and overrideable.
**Deliverables:** `orch-go-r7avo`, daemon config schema, preview/status/event output with route reason.
**Exit criteria:** Humans can explain why a model was chosen from daemon artifacts alone.
**Depends on:** Phase 1.

### Phase 3: Add bounded promotion

**Goal:** Recover safely when GPT routing fails in known ways.
**Deliverables:** `orch-go-kdyh6`, failure classification + one-step Opus promotion + tests.
**Exit criteria:** A classified GPT failure promotes once to Opus and never loops silently.
**Depends on:** Phase 1.

### Phase 4: Behavioral verification

**Goal:** Prove the full route behavior, not just unit-tested components.
**Deliverables:** `orch-go-xi8tc`, integration proof spanning route selection, observability, and promotion.
**Exit criteria:** End-to-end evidence shows reasoning work stays on Opus, eligible implementation work can route to GPT-5.4, and GPT failure promotion is visible and bounded.
**Depends on:** Phases 1-3.

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| GPT eligibility boundary | Benchmark evidence + thread policy + current router seam | Yes |
| Route-object placement | Current coordination code + compliance config pattern | Yes |
| GPT recovery policy | Retry design work + historical GPT failure modes | Yes |

**Overall readiness:** Ready to execute

---

## Structured Uncertainty

**What's tested:**
- ✅ GPT-5.4 is strong enough for bounded `feature-impl` overflow (`.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md`).
- ✅ Current daemon code has one obvious place to replace skill-only model inference (`pkg/daemon/skill_inference.go`, `pkg/daemon/coordination.go`).
- ✅ Existing daemon config already uses combo-first policy resolution that the router can mirror (`pkg/daemonconfig/compliance.go`).

**What's untested:**
- ⚠️ Real queue coverage of `effort:*` labels for eligible implementation work.
- ⚠️ Whether GPT failure classification needs more than `empty_execution` + repeat-failure signals on day one.

**What would change this plan:**
- If GPT-5.4 reasoning benchmarks land above default thresholds, the eligibility matrix should expand.
- If route reasons create too much operator noise, observability may need a compact default format.

---

## Success Criteria

- [ ] Reasoning-heavy and high-complexity work still route to Opus by default.
- [ ] Eligible `feature-impl` work can route to GPT-5.4 with a visible route reason.
- [ ] Classified GPT failure promotes once to Opus without duplicate spawns or hidden loops.
- [ ] `orch-go-xi8tc` closes with end-to-end evidence across selection, observability, and promotion.
