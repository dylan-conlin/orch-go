## Summary (D.E.K.N.)

**Delta:** This plan turns the benchmark-runner design into four implementation tracks with one integration gate.

**Evidence:** The plan is based on the benchmark runner investigation, current harness/spawn code boundaries, and the recent GPT-5.4 manual benchmark investigation.

**Knowledge:** The critical distinction is that benchmarking should orchestrate live runs while harness continues to analyze telemetry after the fact.

**Next:** Start with the CLI surface and suite schema so execution and reporting work share one declarative input.

---

# Plan: Benchmark Runner

**Date:** 2026-03-26
**Status:** Draft
**Owner:** orch-go

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** `.kb/investigations/2026-03-26-inv-design-benchmark-runner-command-orch.md`
**Supersedes:** [Path to plan this replaces, if applicable]
**Superseded-By:** [Path to plan that replaced this, if applicable]

---

## Objective

Ship a first version of `orch benchmark` that can run a suite of tracked model reliability cases, collect canonical per-case evidence, and emit a report that lets the orchestrator decide whether a model or backend path is viable without manual workspace archaeology.

---

## Substrate Consulted

> What existing knowledge informed this plan?

- **Models:** `orchestrator` kb context for actor model and worker-facing structured output expectations
- **Decisions:** `2026-02-28-atc-not-conductor-orchestrator-reframe.md` for orchestrator-facing ergonomics
- **Guides:** reliability-testing skill guidance for repeated-run validation and evidence capture
- **Constraints:** Harness namespace is already semantically occupied; benchmarking must avoid Multi-Backend Blindness and Contradictory Authority Signals

---

## Decision Points

> For each fork ahead, what are the options and which is recommended?

### Decision 1: Command placement

**Context:** The runner needs a home that fits its role without muddying existing command meaning.

**Options:**
- **A: Top-level `orch benchmark`** - Dedicated namespace for live benchmark execution. Pros: clear semantics, expandable subcommands, no harness overload. Cons: adds another root command.
- **B: `orch harness benchmark`** - Keeps measurement commands grouped. Pros: discoverable for people already using harness. Cons: conflates live orchestration with governance analytics.

**Recommendation:** Option A because current harness semantics are telemetry and governance oriented, while this workflow actively runs workers.

**Status:** Decided

---

### Decision 2: Execution model

**Context:** The command can either invent a benchmark-specific runner or layer over existing lifecycle primitives.

**Options:**
- **A: Thin runner over spawn/wait/rework** - Expand suite cases and reuse existing primitives. Pros: less duplication, aligned behavior, easier maintenance. Cons: some adapter code required.
- **B: Benchmark-specific lifecycle engine** - New runner handles everything internally. Pros: full local control. Cons: duplicates proven behavior and raises regression risk.

**Recommendation:** Option A because reuse keeps lifecycle behavior consistent and avoids a second orchestration path.

**Status:** Decided

---

### Decision 3: Artifact shape

**Context:** Benchmark conclusions need durable evidence and resumable state.

**Options:**
- **A: Dedicated benchmark run directory plus summary report** - One run folder contains suite input, case outputs, and verdict summary. Pros: comparable runs, resumable state, canonical evidence. Cons: introduces a new artifact surface.
- **B: Only beads comments and ad hoc investigations** - Rely on existing artifacts. Pros: no new storage concept. Cons: evidence stays scattered and hard to compare.

**Recommendation:** Option A because canonical run artifacts are the main value of making benchmarking first class.

**Status:** Decided

---

## Phases

> Execution phases with clear deliverables and exit criteria.

### Phase 1: CLI and suite schema

**Goal:** Give benchmark runs one declarative input surface.
**Deliverables:** `orch benchmark` root command, `run` subcommand, suite file schema, dry-run output, validation tests.
**Exit criteria:** A suite file can be parsed into benchmark cases and surfaced in dry-run mode.
**Depends on:** None.

### Phase 2: Execution engine

**Goal:** Run benchmark cases through existing orchestration primitives.
**Deliverables:** Benchmark package, case scheduler, serial execution default, wait/result capture, optional retry hooks.
**Exit criteria:** A suite can execute tracked cases and collect raw case outcomes.
**Depends on:** Phase 1.

### Phase 3: Artifact and report layer

**Goal:** Turn raw outcomes into canonical, reviewable benchmark evidence.
**Deliverables:** Run directory format, per-case artifacts, summary report, threshold evaluation, comparison-friendly metadata.
**Exit criteria:** A finished run produces a stable summary report and durable artifacts.
**Depends on:** Phase 2.

### Phase 4: Behavioral integration verification

**Goal:** Prove the whole command is trustworthy on repeated runs.
**Deliverables:** Integration issue completion, end-to-end test path, sample suite, repeatability evidence.
**Exit criteria:** Repeated suite runs produce stable verdict accounting without manual workspace inspection.
**Depends on:** Phases 1-3.

---

## Readiness Assessment

> Can we navigate each decision point ahead?

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Command placement | Current harness and spawn command surfaces | Yes |
| Execution model | Existing spawn loop and rework primitives | Yes |
| Artifact shape | Recent manual benchmark investigation pain points | Yes |

**Overall readiness:** Ready to execute

---

## Structured Uncertainty

**What's tested:**
- ✅ Existing lifecycle reuse is viable in principle because loop and reliability-testing contract tests pass

**What's untested:**
- ⚠️ The final benchmark artifact shape has not been exercised in implementation
- ⚠️ Parallel execution remains intentionally unvalidated

**What would change this plan:**
- If implementation proves spawn/wait/rework adapters too awkward, split execution behind a narrower facade before expanding scope
- If run artifacts prove too noisy, narrow version one to one summary file plus referenced workspace pointers

---

## Success Criteria

- [ ] `orch benchmark run --suite <file>` executes at least one multi-model suite end to end
- [ ] Benchmark output includes pass rate, duration, retries, commit evidence, and verification artifact presence per case
- [ ] Repeating the same suite yields comparable verdict accounting
- [ ] Integration issue `orch-go-00f3d` verifies behavioral trust, not just component existence
