## Summary (D.E.K.N.)

**Delta:** This plan converts GPT-5.4 zero-token silent deaths from an unclassified failure into a bounded, observable recovery path.

**Evidence:** It is based on the March 26 benchmark retry evidence plus verified code-path analysis of spawn retry, OpenCode status, lifecycle orphan detection, and status telemetry.

**Knowledge:** The key design insight is that orch needs an OpenCode-specific empty-execution classifier, not a generic retry-on-zero-tokens rule.

**Next:** Implement the classifier in `orch-go-o9k80`, then wire retry policy in `orch-go-phbzy`, then expose evidence in `orch-go-bfzfc` before closing the integration proof issue `orch-go-6k6o8`.

---

# Plan: Gpt54 Empty Execution Retry

**Date:** 2026-03-26
**Status:** Ready
**Owner:** orch-go-7iw5a

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** `.kb/investigations/2026-03-26-inv-design-retry-strategy-gpt-silent.md`
**Supersedes:** None
**Superseded-By:** None

---

## Objective

orch can detect when an OpenCode worker accepted work but produced an empty execution, retry that exact failure once, and then surface repeat failures for review without creating duplicate work or hiding the reason from operators.

---

## Substrate Consulted

> What existing knowledge informed this plan?

- **Models:** [Which models were consulted and what they said]
- **Models:** `.kb/models/opencode-session-lifecycle/model.md` - `busy/retry/idle` are the authoritative OpenCode session states; persisted session existence is not liveness.
- **Decisions:** `kb context "retry strategy gpt silent death openai opencode"` surfaced the decision that OpenCode status is queried via HTTP and sessions persist on disk.
- **Guides:** Worker/architect protocol from SPAWN_CONTEXT plus issue-handoff requirements for multi-component designs.
- **Constraints:** Provenance, Session Amnesia, and backend-specific semantics; do not generalize OpenCode token rules to Claude/tmux workers.

---

## Decision Points

> For each fork ahead, what are the options and which is recommended?

### Decision 1: What should trigger retry?

**Context:** Zero tokens alone are too weak, but manual rerun evidence shows a recoverable transient exists.

**Options:**
- **A: Empty-execution fingerprint** - require prompt acceptance, active-to-idle transition, zero tokens, no assistant output, and no landed artifacts. Pros: precise, evidence-based, safer against duplicate work. Cons: more implementation effort.
- **B: Retry on zero tokens alone** - retry any zero-token session. Pros: simple. Cons: brittle, backend-specific, risks false positives.

**Recommendation:** Option A because it matches the observed failure while respecting defect classes 6 and 7.

**Status:** Decided

---

### Decision 2: How much automatic retry is safe?

**Context:** A recoverable transient justifies automation, but repeated retries would hide regressions and create loops.

**Options:**
- **A: One-shot retry then escalate** - exactly one automatic retry for `empty_execution`, then review on repeat failure. Pros: bounded, observable, low loop risk. Cons: may leave some recoverable second transients for humans.
- **B: Retry until success or existing generic budget** - reuse broader retry budgets. Pros: less new machinery. Cons: conflates unrelated failure classes and increases duplicate-action risk.

**Recommendation:** Option A because the benchmark proves recoverability once, not indefinite reliability.

**Status:** Decided

---

## Phases

> Execution phases with clear deliverables and exit criteria.

### Phase 1: Classify terminal outcomes

**Goal:** Create a trustworthy OpenCode terminal-outcome classifier.
**Deliverables:** `orch-go-o9k80`, tests for success vs `empty_execution` vs deterministic failure.
**Exit criteria:** Retry logic can consume a single canonical classification object instead of ad hoc token checks.
**Depends on:** None.

### Phase 2: Wire bounded recovery

**Goal:** Retry classified empty executions once and escalate on repeat.
**Deliverables:** `orch-go-phbzy`, retry metadata, escalation behavior.
**Exit criteria:** First failure retries automatically, second failure surfaces for review, no loops.
**Depends on:** Phase 1.

### Phase 3: Make recovery legible

**Goal:** Expose why orch retried and what happened.
**Deliverables:** `orch-go-bfzfc`, operator-facing status/review/event output; `orch-go-6k6o8` integration proof.
**Exit criteria:** Humans can verify retry cause and outcome from status/review artifacts alone.
**Depends on:** Phases 1-2.

---

## Readiness Assessment

> Can we navigate each decision point ahead?

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Retry fingerprint | Benchmark evidence + spawn/lifecycle code | Yes |
| Retry budget | Benchmark evidence + defect taxonomy | Yes |

**Overall readiness:** Ready to execute

---

## Structured Uncertainty

**What's tested:**
- ✅ One GPT-5.4 investigation silently died then succeeded on rerun (`.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md`).
- ✅ Current lifecycle uses `SessionExists`, which treats persisted sessions as alive (`pkg/opencode/client.go`, `pkg/agent/lifecycle_impl.go`).

**What's untested:**
- ⚠️ Population frequency of `empty_execution` beyond the benchmark sample.
- ⚠️ Whether OpenCode can provide a stronger server-side termination reason later.

**What would change this plan:**
- If partial-work sessions can present as zero-token exits, the classifier must become stricter before enabling retry.
- If larger GPT-5.4 runs show repeated empty-execution bursts, retry may need to become opt-in rather than default.

---

## Success Criteria

- [ ] `empty_execution` is a first-class classified terminal outcome with automated tests.
- [ ] A matched first failure retries once and a matched second failure escalates without loops.
- [ ] Status/review/events expose retry reason and outcome clearly enough for human verification.
