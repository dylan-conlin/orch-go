# Probe: Code Review Gate Design for Completion Pipeline

**Date:** 2026-02-25
**Status:** Complete
**Beads:** orch-go-1247
**Model:** completion-verification

## Question

The completion pipeline has a "sharp boundary at execution" — everything above "tests actually pass" is verified (build compiles), everything below is not (test evidence is pattern-matched claims, SYNTHESIS.md is self-reported). Should a code review gate close this boundary by having an independent agent review the diff before completion?

**Model claims being tested:**
1. Gates are "structurally independent but functionally level-selective" — would a code review gate fit the level system?
2. The "only unfakeable gate" is build (`go build ./...`) — would code review add a second unfakeable gate or just another fakeable one?
3. V3 (Behavioral) is the highest level, adding visual/explain-back/behavioral gates — is code review a V4, or something orthogonal?

## What I Tested

### Test 1: Substrate Consultation

Reviewed all relevant principles, models, decisions, and prior investigations:

**Principles checked:**
- **Provenance:** "Every conclusion must trace to something outside the conversation." Agent reviewing agent code is a closed loop unless it produces something human-verifiable. Code review findings would be agent artifacts (hypotheses), not primary evidence.
- **Evidence Hierarchy:** "Code is truth. Artifacts are hypotheses." Code review produces MORE artifacts, not more truth. Only execution produces truth (tests, build).
- **Gate Over Remind:** If code quality matters, gate it. But WHAT do you gate on? Agent opinion? That's a reminder with gate dressing.
- **Verification Bottleneck:** "System cannot change faster than human can verify behavior." Code review by agent is NOT human verification. It doesn't increase Dylan's verification bandwidth.

**Decisions checked:**
- **"Use phased adversarial verification over post-completion review"** — DIRECTLY CONTRADICTS code review at completion. This decision says verification during execution prevents flawed conclusions; post-completion review only detects after the fact.
- **Verifiability-First (Two-Gate):** Gate 1 (comprehension) and Gate 2 (behavioral) are HUMAN gates. Adding agent review would create a third type — agent opinion — that breaks the clean human/machine separation.
- **V0-V3 Levels:** Designed to REDUCE gate proliferation, not add more. "Historical failure mode: gates proliferate → flag combos needed → `--force` becomes default → theater."

**Investigations checked:**
- Synthesis (Feb 19-24): "The system keeps discovering that prompt-level constraints fail under pressure, and the fix is always the same — add infrastructure enforcement alongside the prompt." But code review is NOT infrastructure enforcement — it's agent judgment, which is what prompts are.
- Rework Loop Design: Documents the dead-end after Block/Failed escalation but doesn't suggest review as the fix.

### Test 2: Failure Mode Analysis

What does "nobody reads the diff" actually fail to catch?

| Failure Mode | Current Coverage | Gap? |
|---|---|---|
| Build doesn't compile | Build gate (`go build`) | NO — fully covered |
| Tests don't pass | Test evidence gate (pattern-matched) | PARTIAL — checks for evidence format, not actual pass/fail |
| Logic error in passing code | None | YES |
| Security vulnerability | None (unless build fails) | YES |
| Architectural violation | Accretion gate (file growth), coupling hotspot (advisory) | PARTIAL |
| Edge case missed | None | YES |
| Code doesn't match SYNTHESIS | Git diff gate (filename match) | PARTIAL — checks files, not semantics |

**Actual gap:** Logic errors, security issues, and edge cases in code that compiles and passes tests. These are real. The question is whether agent code review closes them.

### Test 3: Would Agent Code Review Close the Gap?

Analyzed what a code review agent would actually do:
1. Read the git diff (files changed since spawn)
2. Check for common issues (error handling, security, coupling, edge cases)
3. Report findings

**Problems with this approach:**
- **Closed loop:** Same model family, similar blind spots. If the writing agent missed an edge case, what makes us think the reviewing agent catches it?
- **No ground truth:** Code review produces opinions, not verifiable outcomes. Unlike `go build` (binary pass/fail) or test evidence (concrete output), code review findings are judgments.
- **False confidence:** "An agent reviewed it" could REDUCE human scrutiny, making things worse.
- **Post-completion timing:** Violates "phased adversarial verification" decision. By the time review happens, the agent has already committed to its conclusions.
- **Cost/speed:** Opus-level agent reading a diff is expensive. For every completion. Most diffs will be fine.

### Test 4: What Works Instead?

Analyzed execution-time alternatives:

| Alternative | Type | Catches | Cost |
|---|---|---|---|
| `go vet` (expand build gate) | Execution | Unused vars, unreachable code, format errors, common bugs | ~2s, zero tokens |
| `staticcheck` | Execution | Hundreds of bug patterns, performance issues | ~5s, zero tokens |
| Pre-commit linting | Execution-time | Style, formatting, import order | ~1s, zero tokens |
| Selective review (hotspot-triggered) | Advisory | Complex problems in known-fragile areas | Agent tokens, but only when warranted |

**Key insight:** Expanding the build gate to include `go vet` is strictly better than agent code review because:
1. It executes (produces truth, not hypothesis)
2. It's fast and free (no token cost)
3. It catches real bugs (not opinions)
4. It's deterministic (same code = same result)
5. It fits the existing V2 level (Evidence) without adding a new level

## What I Observed

**The "nobody reads the diff" framing contains a hidden assumption: that reading the diff is valuable.** In the context of this system:

1. Dylan doesn't read code (by choice — strategic comprehender role)
2. The orchestrator can't read code (by design — frame guard)
3. Agent code review is a closed loop (same model, same blind spots)

The completion pipeline isn't broken because nobody reads the diff. The completion pipeline is correctly designed for a system where the human verifies BEHAVIOR, not CODE. The explain-back gate forces comprehension. The behavioral gate forces observation. The test/build gates force mechanical correctness.

**What IS broken:** The build gate only runs `go build`. It should also run `go vet` (catches real bugs, zero cost, execution-based). This is a genuine gap — mechanical verification that's available and not being used.

**The "sharp boundary at execution"** is the right observation from the model, but the fix isn't to add another non-execution gate (code review). The fix is to expand what executes. `go vet`, `staticcheck`, and potentially `go test -race` are all execution-based verification that would catch real bugs without the closed-loop problem of agent review.

## Model Impact

**Confirms:** Model claim that build is "the only unfakeable gate." Code review would be fakeable (agent opinion, not execution output). The right response is to add MORE unfakeable gates (vet, staticcheck) not more fakeable ones.

**Confirms:** Model claim about "sharp boundary at execution." The boundary exists, it's real, and it's the right place to push — but by expanding execution, not by adding judgment.

**Extends:** The model should document the distinction between **execution-based gates** (build, vet, test) and **evidence-based gates** (test evidence pattern, SYNTHESIS review, explain-back). Execution-based gates produce provenance. Evidence-based gates check for provenance. Code review would be a third type — **judgment-based** — that produces neither provenance nor checks for it.

**Contradicts nothing.** The model's architecture is correct. The completion pipeline is correctly positioned as a knowledge management ceremony with execution-based mechanical checks. Adding code review would mix concerns.

## Investigation Reference

Full design: `.kb/investigations/2026-02-25-design-code-review-gate-for-completion-pipeline.md`
