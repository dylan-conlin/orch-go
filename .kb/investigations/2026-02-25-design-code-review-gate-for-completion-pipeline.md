# Design: Code Review Gate for Agent Completion Pipeline

**Date:** 2026-02-25
**Phase:** Complete
**Status:** Complete
**Beads:** orch-go-1247
**Type:** Architect investigation with recommendation

---

## Design Question

Should the agent completion pipeline include a code review gate — an independent agent that reviews the git diff before the orchestrator approves completion?

## Problem Framing

### The Observation

The orchestrator's completion review is a knowledge management ceremony, not a code quality gate. The current flow:

1. Worker agent writes code, runs tests, commits, reports Phase: Complete
2. `orch complete` runs 14 automated gates (build, test evidence, accretion, git diff, etc.)
3. Orchestrator provides explain-back (Gate 1: comprehension) and --verified (Gate 2: behavioral)
4. Issue closed

**Nobody reads the diff.** The orchestrator can't (frame guard blocks code files). Dylan doesn't (strategic comprehender role). The worker self-reports via SYNTHESIS.md. Automated gates check STRUCTURE (files exist, tests claimed, build compiles) but not SUBSTANCE (is the logic correct? secure? well-architected?).

### The Sharp Boundary

The completion-verification model documents a "sharp boundary at execution":
- **Above the boundary** (verified by execution): `go build ./...` — the only gate that actually runs code
- **Below the boundary** (verified by evidence claims): test output patterns, SYNTHESIS.md claims, visual verification evidence

Everything between "agent says tests pass" and "the binary compiles" is unverified territory.

### Success Criteria

A good answer will:
1. Address the real gap (not just the symptom)
2. Be consistent with existing substrate (principles, decisions, models)
3. Avoid the historical failure mode (gate proliferation → `--force` ceremony → theater)
4. Respect the verification bottleneck (human verification bandwidth is the rate limiter)

### Constraints

- Must not violate "phased adversarial verification over post-completion review" decision
- Must not create closed loops (agent verifying agent without human provenance)
- Must fit V0-V3 level system or justify a new level
- Must not slow the common case (most completions are fine)

### Scope

- IN: Completion pipeline gates, execution-time verification, selective review mechanisms
- OUT: CI/CD integration, deployment gates, pre-commit hooks (separate concern)

---

## Exploration

### Fork 1: Should code review exist in the completion pipeline at all?

**Options:**
- A: Yes, universal code review gate (every completion)
- B: Yes, selective code review (triggered by risk signals)
- C: No code review gate, but expand execution-based gates
- D: No changes — current pipeline is sufficient

**Substrate says:**

- **Principle (Provenance):** "Every conclusion must trace to something outside the conversation." Agent reviewing agent code is a closed loop. Code review findings are agent artifacts (hypotheses about code), not primary evidence (execution output).

- **Principle (Evidence Hierarchy):** "Code is truth. Artifacts are hypotheses." Code review produces MORE hypotheses, not more truth. Only execution produces truth.

- **Decision ("Phased adversarial verification over post-completion review"):** Post-completion review doesn't prevent validation loops. Only detects after agent committed to flawed conclusions. This decision DIRECTLY CONTRADICTS adding code review at completion time.

- **Decision (V0-V3 Levels):** "Historical failure mode: gates proliferate → flag combos needed → `--force` becomes default → theater." Adding a code review gate risks this exact pattern.

- **Decision (Verifiability-First Two-Gate):** Gate 1 (comprehension) and Gate 2 (behavioral) are HUMAN gates. Adding agent code review creates a third type — agent judgment — that breaks the clean human/machine separation.

**Recommendation:** Option C — expand execution-based gates. The gap is real, but agent code review is the wrong fix. See Fork 4 for what to expand.

**Trade-off accepted:** Logic errors, edge cases, and security vulnerabilities in code that compiles and passes tests will NOT be caught by automated gates. This is accepted because: (1) agent review wouldn't reliably catch them either (same model, same blind spots), (2) the explain-back gate forces human comprehension of WHAT was built, and (3) behavioral verification forces human observation of WHETHER it works.

### Fork 2: What makes code review a closed loop?

**The argument FOR code review:** A fresh agent brings a different perspective, like a human code reviewer.

**Why this doesn't hold for AI agents:**

1. **Same model family.** Both writer and reviewer are Claude instances. They share training data, reasoning patterns, and blind spots. When a human reviews code, they bring DIFFERENT knowledge (domain expertise, past bug patterns, personal experience). When Claude reviews Claude's code, it brings the same knowledge.

2. **No accumulated context.** Human reviewers carry institutional memory — "we tried this approach last year and it broke under load." Agent reviewers start from scratch each time (Session Amnesia). The spawn context provides some institutional memory via kb context, but this is the same context the writing agent had.

3. **Unfalsifiable findings.** Human code review produces discussion (back-and-forth, clarification, alternative proposals). Agent code review produces a report. If the report says "this looks fine," the orchestrator (who can't read code) has no way to evaluate whether the review was thorough. This is exactly the "agent confirmed this" anti-pattern (Provenance principle).

4. **No provenance chain.** When `go build` says PASS, that's verifiable. When `go vet` reports an issue, you can reproduce it. When an agent says "I reviewed the code and found no issues," that traces to... the agent's judgment. Closed loop.

### Fork 3: What is the actual gap?

**What the pipeline catches today:**

| What | Gate | Type |
|---|---|---|
| Build compiles | Build gate (`go build ./...`) | Execution ✓ |
| Test evidence exists | Test evidence gate (pattern match) | Evidence claim |
| Files match SYNTHESIS claims | Git diff gate | Structural check |
| File growth controlled | Accretion gate | Metric check |
| Human understands what was built | Explain-back (Gate 1) | Human gate ✓ |
| Human observed behavior | Behavioral (Gate 2) | Human gate ✓ |

**What the pipeline misses:**

| What | Severity | Would agent review catch it? |
|---|---|---|
| Logic error in code that compiles and passes tests | Medium | Maybe — same blind spots as writer |
| Security vulnerability | High | Maybe — but static analysis tools are better |
| Architectural violation | Medium | Partially — accretion/coupling gates partially cover |
| Missing edge case handling | Low-Medium | Unlikely — if writer missed it, reviewer likely will too |
| Dead code / unused imports | Low | `go vet` catches this deterministically |

**Key insight:** For every gap, there's either (a) a deterministic tool that's better than agent review, or (b) no reliable way to catch it with current AI.

### Fork 4: What should expand instead?

**Options:**
- A: Expand build gate to include `go vet`
- B: Add `staticcheck` to build gate
- C: Add test execution to completion (actually run tests, not just check evidence)
- D: Selective spawned review for hotspot-flagged completions

**Substrate says:**
- **Principle (Evidence Hierarchy):** Execution-based verification is primary evidence. `go vet` findings are as reliable as `go build` findings — deterministic, reproducible, verifiable.
- **Model (completion-verification):** "The only unfakeable gate is build." We can add more unfakeable gates by expanding what executes.

**Recommendation:** A + D (expand build gate AND add selective review mechanism)

- **A (go vet):** Zero-cost, zero-token, deterministic. Catches unused variables, unreachable code, incorrect format strings, suspicious constructs. Fits naturally into the existing build gate at V2.
- **D (selective review):** When the coupling hotspot system or accretion gate flags high-risk areas, the completion output includes an advisory warning. The orchestrator can then choose to spawn an investigation to review the code — using existing infrastructure, triggered by orchestrator judgment, not automated gate.

**Rejected:**
- B (staticcheck): More opinionated, may produce false positives, requires external tool installation. Consider as future enhancement after `go vet` proves its value.
- C (test execution): Running the full test suite at completion time is expensive (may take minutes), and the build gate already proves compilation. If tests fail, the agent should have caught it during execution. If they didn't, agent code review wouldn't catch it either.

### Fork 5: How should the orchestrator consume hotspot warnings?

**Options:**
- A: Block completion for hotspot-area changes
- B: Advisory warning in completion output
- C: Automatic spawning of review investigation

**Substrate says:**
- **Decision (V0-V3):** "Spawn-only immutability." Verification level set at spawn, not changed at completion. Blocking based on completion-time discovery violates this.
- **Decision (V0-V3):** "Warn, don't elevate." When unexpected conditions are found at completion, warn — don't block. The orchestrator learns to declare higher levels upfront.

**Recommendation:** Option B — advisory warning. The completion output includes a warning like:

```
⚠ HOTSPOT: Changes touch cmd/orch/complete_cmd.go (coupling-cluster score: 180 CRITICAL)
  Consider: orch spawn investigation "review changes from orch-go-XXXX"
```

The orchestrator decides whether to act on it. This preserves:
- Spawn-only immutability (no new gates at completion)
- Orchestrator judgment (human-in-the-loop for review decision)
- Existing infrastructure (investigation skill, spawn command)

---

## Synthesis

### Core Recommendation

**Do NOT add a code review gate to the completion pipeline.**

The completion pipeline is correctly designed as a knowledge management ceremony with execution-based mechanical checks. The "nobody reads the diff" observation is accurate but the proposed solution (agent code review) creates worse problems than it solves:

1. **Closed loop** (Provenance violation): Agent reviewing agent without human-verifiable output
2. **Post-completion timing** (contradicts "phased adversarial verification" decision): Review at completion doesn't prevent flawed conclusions, only detects them
3. **Gate proliferation risk** (V0-V3 lesson): Adding another gate pushes toward `--force` as happy path
4. **False confidence**: "An agent reviewed it" could reduce human scrutiny
5. **Cost without proportionate benefit**: Opus-level token spend per completion for marginal quality gain

### Instead, Do Two Things

#### 1. Expand the build gate to include `go vet`

Add `go vet ./...` to the existing build verification gate (`pkg/verify/build_verification.go`). This is:
- **Execution-based** (produces provenance, not opinion)
- **Zero-cost** (no token spend, ~2 seconds)
- **Deterministic** (same code = same result)
- **Catches real bugs** (unused variables, unreachable code, format string mismatches, suspicious constructs)
- **Already at V2** (fits the Evidence level without new infrastructure)

#### 2. Add hotspot-triggered advisory warnings at completion

When `orch complete` detects changes in hotspot areas (coupling clusters, bloat hotspots, fix-density hotspots), emit an advisory warning with a suggested follow-up command. The orchestrator decides whether to act.

This is:
- **Advisory, not blocking** (preserves spawn-only immutability)
- **Orchestrator-judgment-driven** (human in the loop for review decision)
- **Uses existing infrastructure** (investigation skill, spawn command)
- **Targeted** (only fires for high-risk areas, not every completion)

### The Three-Type Gate Taxonomy

This investigation reveals a useful taxonomy for the model:

| Gate Type | Examples | Produces | Provenance? |
|---|---|---|---|
| **Execution-based** | Build, go vet, tests (if we ran them) | Binary pass/fail | ✓ Verifiable |
| **Evidence-based** | Test evidence patterns, git diff, SYNTHESIS | Pattern match against claims | Partial |
| **Judgment-based** | Code review, explain-back, behavioral | Opinion/comprehension | Human only |

The completion pipeline should consist of:
- Execution-based gates for mechanical correctness (machine-verifiable)
- Evidence-based gates for completeness claims (anti-theater detection)
- Judgment-based gates for human comprehension (explain-back, behavioral)

**Agent code review** would be a fourth type — agent judgment — which has no provenance chain. It produces opinion that neither the orchestrator nor Dylan can independently verify.

---

## Recommendations

⭐ **RECOMMENDED:** Expand execution-based gates + selective advisory warnings

**Why:**
- Closes the real gap (mechanical verification tools not being used) without the closed-loop problem
- Consistent with all substrate: Provenance, Evidence Hierarchy, phased adversarial verification, V0-V3 levels
- Minimal implementation effort (~20 lines to add `go vet` to build gate)
- Zero ongoing cost (no token spend)

**Trade-off:** Logic errors, security vulnerabilities, and edge cases that pass `go build` AND `go vet` will NOT be caught automatically. This is accepted because: (a) agent review wouldn't reliably catch them either, (b) explain-back forces human comprehension, (c) behavioral gate forces human observation, and (d) targeted review is available as an escape hatch via `orch spawn investigation`.

**Expected outcome:**
- `go vet` catches 10-30% of the "compiles but has issues" class of bugs (based on typical go vet output in Go projects)
- Zero additional completion time (vet runs in ~2s alongside build)
- Hotspot warnings provide actionable signal for the 5-10% of completions in fragile areas

**Alternative: Universal agent code review gate**
- **Pros:** Fresh perspective on every diff, catches some issues humans would catch
- **Cons:** Closed loop (Provenance violation), post-completion timing (contradicts existing decision), expensive (opus per completion), gate proliferation risk, false confidence
- **When to choose:** If evidence emerges that execution-based gates + human review are insufficient AND a new model family (different blind spots) becomes available for review

**Alternative: No changes**
- **Pros:** No implementation work, no new complexity
- **Cons:** Leaves the `go vet` gap — free verification not being used
- **When to choose:** If `go vet` produces too many false positives in the codebase (test first)

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the completion review gap question directly
- Future agents proposing code review should see this analysis

**Suggested blocks keywords:**
- "code review"
- "completion pipeline"
- "diff review"
- "agent review"
- "go vet"
- "build gate"

---

## Implementation-Ready Output

### Phase 1: Expand build gate (low effort, high value)

**File targets:**
- `pkg/verify/build_verification.go` — add `go vet ./...` alongside `go build ./...`
- `pkg/verify/build_verification_test.go` — add test for vet integration
- `.kb/guides/completion-gates.md` — update gate 9 documentation

**Acceptance criteria:**
- `go vet ./...` runs during `VerifyBuildForCompletion()` when Go files changed
- Vet failures are reported as warnings (not errors) initially to assess noise level
- After confidence period: promote to errors if false positive rate <5%

### Phase 2: Hotspot advisory warnings (medium effort)

**File targets:**
- `cmd/orch/complete_cmd.go` — add hotspot check after verification passes
- `pkg/verify/hotspot_advisory.go` — new file, reads hotspot data and matches against diff
- `.kb/guides/completion-gates.md` — document advisory mechanism

**Acceptance criteria:**
- `orch complete` checks if changed files appear in hotspot areas
- Advisory warning printed with suggested follow-up command
- Warning does NOT block completion
- Warning includes hotspot type (coupling, bloat, fix-density) and severity score

### Out of scope
- Agent code review agent/skill
- Automatic spawning of review investigations
- CI/CD integration
- Pre-commit hooks (separate concern, valuable but different)

---

## Discovered Work

| Issue | Type | Description |
|---|---|---|
| Expand build gate with `go vet` | feature | Add `go vet ./...` to build verification, initially as warning |
| Add hotspot advisory warnings to `orch complete` | feature | Warn when completion touches hotspot areas |
| Update completion-verification model | task | Add three-type gate taxonomy (execution/evidence/judgment) |

---

## Source Investigations and Evidence

- Completion-verification model: `.kb/models/completion-verification/model.md`
- V0-V3 decision: `.kb/decisions/2026-02-20-verification-levels-v0-v3.md`
- Verifiability-first decision: `.kb/decisions/2026-02-14-verifiability-first-hard-constraint.md`
- Phased adversarial verification decision: Prior knowledge (from kb context)
- Enforcement synthesis: `.kb/investigations/2026-02-24-synthesis-enforcement-accretion-verification-design-burst.md`
- Three code paths probe: `.kb/models/completion-verification/probes/2026-02-16-probe-three-code-paths-verification-state.md`
- Verification levels probe: `.kb/models/completion-verification/probes/2026-02-20-probe-verification-levels-design.md`
- Rework loop probe: `.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md`
- Frame guard: `~/.orch/hooks/gate-orchestrator-code-access.py`
- Build verification: `pkg/verify/build_verification.go`
- Completion gates guide: `.kb/guides/completion-gates.md`
