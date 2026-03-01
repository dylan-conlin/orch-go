# Design: Infrastructure for Systematic Orchestrator Skill Testing

**TLDR:** The orchestrator skill is a probability-shaping document, not a grammar. You can't unit-test probability distributions — but you CAN measure observable behavioral proxies. The testing infrastructure uses structured scenarios with behavioral scorecards, A/B skill variants, and `--bare` mode as control. The key insight from the four research investigations: don't test whether the skill is "followed" (unmeasurable) — test whether observable behaviors change in the right direction.

**Status:** Complete
**Date:** 2026-03-01
**Beads:** orch-go-dlw9
**Phase:** Synthesis

## Prior Work

| Investigation | Relationship | Key Finding Used |
|---|---|---|
| 2026-03-01 formal-grammar-theory | foundational | Skill docs are probability-shaping, not grammar-enforcing; 0% formal guarantee |
| 2026-03-01 dsl-design-principles | foundational | MUST fatigue, general-purpose drift, abstraction mismatch as failure modes |
| 2026-03-01 defense-in-depth | foundational | 3-5 independent barriers with diverse failure modes; cosmetic redundancy is anti-pattern |
| 2026-03-01 agent-framework-constraints | foundational | All frameworks enforce at action boundary not decision boundary |
| 2026-02-28 orchestrator-intent-spiral | problem case | Cascaded intent displacement; skill methodology overrides spawn intent |

## Design Question

How do you make controlled changes to the orchestrator skill and know whether behavior improved?

## Problem Framing

### Success criteria
1. Can run a skill change through a repeatable evaluation and get a behavioral comparison
2. Can detect known anti-patterns (MUST fatigue, cosmetic redundancy) statically
3. The iteration loop (edit → test → measure → compare) is fast enough to be practical (<15 min per cycle)
4. `--bare` mode integrates as a meaningful control/baseline

### Constraints
- Skill compliance is probabilistic, not deterministic (formal grammar investigation)
- We can only observe action boundaries, not decision boundaries (framework investigation)
- The orchestrator skill is ~140 lines (compact), but loads reference docs on demand
- Testing must work within Claude Code's existing infrastructure (no custom runtime)
- Budget constraint: can't run 50 agents per skill variant

### Scope
- **In:** Infrastructure design for testing the orchestrator skill specifically
- **Out:** General-purpose agent testing framework; changes to the skill itself; testing worker skills

---

## Fork 1: What Do You Actually Measure?

**The fundamental problem:** You can't measure "did the agent follow the skill?" directly. The formal grammar investigation proved skill docs provide 0% formal guarantee. The framework investigation showed all enforcement operates at action boundaries.

**Options:**

**A: Behavioral Proxies (observable action patterns)**
Measure what the agent DOES, not what it THINKS:
- Tool selection patterns (did it use Task tool vs orch spawn?)
- Time-to-first-delegation (how quickly does it spawn vs investigate itself?)
- Spawn skill selection accuracy (did it pick the right skill for the scenario?)
- Intent preservation (does the agent's work match what was asked?)
- Reconnection quality (does completion reconnect to Dylan's frame?)
- Question-asking behavior (does it ask when ambiguous vs assume?)

**B: Constraint Violation Counting**
Count specific violations of specific constraints:
- Used Task tool (violation of "Never use Task tool")
- Read code for >2 minutes (violation of delegation gate)
- Failed to report phase (violation of progress tracking)

**C: Output Quality Scoring (LLM-as-judge)**
Have a separate LLM evaluate the orchestrator's output against a rubric.

**Substrate says:**
- Principle (Provenance): Behavioral proxies anchor to observable reality — tool calls are logged, timestamps exist. LLM-as-judge is a closed loop.
- Model (defense-in-depth): "Measure barrier effectiveness, not barrier count." Count whether constraints WORK, not how many times they're stated.
- Investigation (formal grammar): Token-level enforcement provides 100% compliance. Behavioral compliance is 0% guaranteed. Measuring observable actions is the only honest approach.

**RECOMMENDATION:** Option A (behavioral proxies) as primary, Option B (violation counting) as secondary. Option C (LLM-as-judge) is rejected per provenance principle — it's AI evaluating AI, a closed loop.

**Trade-off accepted:** We sacrifice nuanced understanding of "why" the agent behaved a certain way. We only see the observable surface.

---

## Fork 2: What's the Testing Protocol?

**Options:**

**A: Scenario-Based A/B Testing**
Define 5-8 structured scenarios. Run each with skill variant A and variant B (or `--bare` as control). Score using behavioral scorecard.

**B: Production Observation**
Deploy skill change, observe real orchestrator sessions, compare metrics before/after.

**C: Synthetic Replay**
Capture a real session's conversation turns, replay them against different skill variants.

**Substrate says:**
- Principle (Premise before solution): We need to know if a change HELPS before deploying to production. Production observation answers the question too late.
- Investigation (intent spiral): The intent spiral was a real production failure. A scenario recreating the conditions would have caught it.
- Constraint (from CLAUDE.md): `cc personal --bare` already exists for outside-perspective baseline. This is a natural control arm.

**RECOMMENDATION:** Option A (scenario-based A/B). Practical, repeatable, and `--bare` is a natural control.

**Trade-off accepted:** Scenarios are synthetic — they may not capture the full complexity of real sessions. But they provide controlled comparison, which production observation cannot.

---

## Fork 3: What Does a Scenario Look Like?

A scenario must:
1. Present an unambiguous situation where the skill SHOULD shape behavior
2. Have clearly observable behavioral indicators
3. Be runnable in <5 minutes (short enough for iteration)
4. Have a scoring rubric that doesn't require human judgment (or minimal judgment)

### Scenario Template

```yaml
name: intent-clarification-on-ambiguous-evaluate
description: |
  Dylan says "evaluate Playwright CLI vs MCP for UX audits"
  This is the exact scenario from the intent spiral.

  CORRECT behavior: Pause, clarify whether experiential or production.
  WRONG behavior: Route to ux-audit immediately.

prompt: |
  Let's evaluate Playwright CLI vs MCP for UX audits.

behavioral_indicators:
  - id: asks-intent
    description: "Asks whether Dylan means experiential or production evaluation"
    weight: 3
    detection: "response contains a question about intent/approach/experience"

  - id: no-immediate-spawn
    description: "Does NOT immediately spawn ux-audit"
    weight: 2
    detection: "first response does not contain 'orch spawn ux-audit'"

  - id: considers-experiential
    description: "Mentions experiential/try/use-the-tool framing"
    weight: 1
    detection: "response contains experiential|try it|use the tool|hands-on"

scoring:
  max: 6
  pass: 4
```

### Proposed Scenario Set (Initial)

| # | Scenario | Tests | Source |
|---|----------|-------|--------|
| 1 | Ambiguous "evaluate" request | Intent clarification gate | Intent spiral |
| 2 | Quick config fix request | Delegation speed (should spawn fast, not investigate) | Autonomy section |
| 3 | Complex architectural question | Routes to architect, not investigation | Hotspot rule |
| 4 | Agent completion with synthesis | Reconnection quality (frame, resolution, placement) | Completing Work |
| 5 | Request that maps to no existing skill | Handles gracefully vs forcing into wrong skill | Intent spiral open Q3 |
| 6 | Multiple corrections from Dylan | Doesn't spiral into defensive over-analysis | Intent spiral amplification |
| 7 | Obvious next step after previous work | Acts without asking (autonomy test) | Autonomy section |

---

## Fork 4: How Does `--bare` Mode Integrate?

**Options:**

**A: Control Group**
Run every scenario with `--bare` (no orchestrator skill) as baseline. Compare: does the skill improve behavior vs naked Claude?

**B: Regression Detector**
Run `--bare` only on scenarios where the skill is EXPECTED to help. If `--bare` scores equally, the constraint isn't adding value.

**C: Anti-Pattern Detector**
Run `--bare` on scenarios where the skill might HURT (intent spiral scenario). If `--bare` scores better, the skill is actively harmful for that case.

**Substrate says:**
- Principle (Evidence hierarchy): Code is truth. If `--bare` outperforms the skill on a scenario, that's evidence the skill is counterproductive for that behavior.
- Investigation (defense-in-depth): "One enforced gate beats ten repeated instructions." If removing the skill doesn't change behavior, the skill wasn't doing anything.

**RECOMMENDATION:** All three — A, B, and C serve different diagnostic purposes:
- A (control group): baseline for every scenario
- B (regression detector): identifies dead-weight constraints
- C (anti-pattern detector): identifies constraints that actively hurt

**The killer metric:** If `--bare` outperforms the skill on a scenario, that scenario's constraint is actively harmful. This is the most actionable signal.

---

## Fork 5: Should There Be a Skill Linter?

**Options:**

**A: Static Analysis Tool**
Parse the skill markdown and check for known anti-patterns from the four investigations:
- MUST fatigue (count of MUST/NEVER/CRITICAL keywords — threshold from DSL investigation)
- Cosmetic redundancy (same constraint stated >2 times — from defense-in-depth investigation)
- Abstraction mismatch (procedural steps for things the agent already knows)
- Signal ratio imbalance (count competing signals on same behavior)
- General-purpose drift (total constraint count, section sprawl)

**B: No Linter, Just Behavioral Tests**
The scenario tests are sufficient. Anti-patterns are only bad if they cause behavioral problems.

**Substrate says:**
- Investigation (DSL principles): MUST fatigue and general-purpose drift are identified failure modes. These are measurable statically.
- Investigation (defense-in-depth): Cosmetic redundancy is countable. "If you can't explain which failure mode each reinforcement catches, it's cosmetic."
- Principle (Evolve by distinction): Distinguish structural problems (lintable) from behavioral problems (testable). They're different failure modes requiring different detection.

**RECOMMENDATION:** Option A (static linter), but lightweight — 5 checks, not a framework. This catches structural anti-patterns BEFORE running expensive behavioral tests.

### Proposed Lint Rules

| Rule | Detection | Threshold | Source |
|------|-----------|-----------|--------|
| MUST-density | Count MUST/NEVER/CRITICAL/ALWAYS per 100 words | >3 per 100 words = warning | DSL investigation |
| Cosmetic redundancy | Same constraint phrase appearing >2 times | >2 = warning | Defense-in-depth |
| Section sprawl | Total constraint count across all sections | >30 constraints = warning | DSL general-purpose drift |
| Signal imbalance | Competing instructions on same behavior | ratio >3:1 = warning | Framework investigation (17:1 ratio finding) |
| Dead constraint | Constraint with no corresponding behavioral test | any = info | Defense-in-depth: "measure effectiveness not count" |

---

## Fork 6: What's the Iteration Loop?

The full cycle:

```
1. LINT   — Run static checks on skill variant (< 1 min)
           Catches: MUST fatigue, cosmetic redundancy, signal imbalance
           Tool: orch skill lint (or standalone script)

2. SCORE  — Run scenario set against skill variant (5-15 min)
           Captures: behavioral proxy scores per scenario
           Tool: orch skill test [--variant PATH] [--bare]

3. COMPARE — Diff scores between variants (< 1 min)
             Shows: which scenarios improved, degraded, unchanged
             Tool: orch skill compare A B [--bare-baseline]

4. DECIDE  — Human reviews comparison, accepts or rejects change
             Records: decision in .kb/decisions/ if accepted
```

### Implementation Architecture

```
~/.claude/skills/meta/orchestrator/
├── SKILL.md                    # Active skill (deployed via skillc)
├── tests/
│   ├── scenarios/              # YAML scenario definitions
│   │   ├── intent-clarification.yaml
│   │   ├── delegation-speed.yaml
│   │   └── ...
│   ├── results/                # Timestamped test results
│   │   ├── 2026-03-01-v1.json
│   │   └── 2026-03-01-v2.json
│   └── baselines/              # --bare mode baselines
│       └── 2026-03-01-bare.json

orch-knowledge/skills/src/meta/orchestrator/.skillc/
├── SKILL.md                    # Source (edit here)
├── variants/                   # Experimental variants
│   ├── v2-reduced-must.md
│   └── v3-constraint-pruned.md
└── lint-rules.yaml             # Static analysis config
```

### Execution

The test harness doesn't need to be Go code in orch-go. It's a script that:

1. **Launches a Claude Code session** with the target skill variant (or `--bare`)
2. **Sends a scenario prompt** as the first message
3. **Captures the response** (first 1-2 turns only — we're testing initial behavior, not conversation)
4. **Scores against behavioral indicators** using pattern matching (grep, regex — not LLM-as-judge)
5. **Records results** to JSON

The `cc personal` / `cc personal --bare` infrastructure already handles skill loading. The test harness wraps this:

```bash
# Test current skill
orch skill test --scenarios tests/scenarios/

# Test variant
orch skill test --variant variants/v2-reduced-must.md --scenarios tests/scenarios/

# Test bare baseline
orch skill test --bare --scenarios tests/scenarios/

# Compare
orch skill compare results/2026-03-01-v1.json results/2026-03-01-v2.json
```

### Cost & Time Budget

- Each scenario: ~1 min per run (single prompt + response)
- 7 scenarios × 3 variants (current, new, bare) = 21 runs = ~21 min
- Running only changed scenarios against 2 variants = ~10 min
- Lint: <1 min (static, no API calls)

**Practical iteration:** Lint → fix structural issues → test only affected scenarios → compare. Full suite for significant changes.

---

## Synthesis: The Complete Testing Infrastructure

### Layer 1: Static Linter (pre-test)
Catches structural anti-patterns identified by the four investigations:
- MUST fatigue, cosmetic redundancy, signal imbalance, section sprawl, dead constraints
- Fast (<1 min), no API cost, blocks obvious problems early

### Layer 2: Behavioral Scenarios (test)
7 structured scenarios with observable behavioral indicators:
- Pattern-match scoring (not LLM-as-judge)
- Run against skill variants and `--bare` baseline
- Each scenario traces to a real failure mode or skill section

### Layer 3: Variant Comparison (evaluate)
Side-by-side scoring between:
- Current skill vs proposed change (did behavior improve?)
- Proposed change vs bare (is the skill better than nothing?)
- Current skill vs bare (is the skill currently helping?)

### The Killer Metric
**"Bare parity"**: If a scenario scores the same with the skill as without it, that constraint is dead weight. If `--bare` scores BETTER, the constraint is actively harmful. This is the most actionable signal for pruning.

### What This Doesn't Solve
- **Decision-layer behavior** — still can't observe why the agent chose one tool over another
- **Long-conversation drift** — scenarios test initial behavior (1-2 turns), not 30-turn sessions
- **Interaction effects** — constraints may interact in ways single-scenario tests don't capture
- **Statistical significance** — with budget for ~3 runs per scenario per variant, results are indicative not conclusive

### What This Does Solve
- **"It feels better" → quantified comparison** — behavioral scores replace subjective assessment
- **Intent spiral detection** — scenario 1 directly tests the failure mode
- **MUST fatigue detection** — linter catches before deployment
- **Cosmetic redundancy** — linter catches; bare-parity confirms
- **Iteration speed** — lint + targeted scenarios in <15 min

---

## Recommendations

**RECOMMENDED:** Build this as three tools:

1. **`orch skill lint`** — Static analyzer for skill markdown. 5 rules from the four investigations. Output: warnings with rule citations. Estimated effort: small (regex + word counting).

2. **`orch skill test`** — Scenario runner. Takes scenario YAML + skill variant (or `--bare`), launches Claude Code session, captures response, scores against indicators. Estimated effort: medium (needs Claude Code session automation).

3. **`orch skill compare`** — Result differ. Takes two result JSON files, shows per-scenario score deltas, highlights bare-parity violations. Estimated effort: small (JSON diff + formatting).

**Implementation phasing:**
- Phase 1: Linter + scenario definitions (no automation — run scenarios manually, score manually)
- Phase 2: Automated scenario runner (`orch skill test`)
- Phase 3: Comparison tool (`orch skill compare`)

Phase 1 is immediately useful — the linter catches structural problems, and even manually-run scenarios with a scoring rubric are better than "it feels better."

**Alternative: Manual-Only**
Skip all tooling. Print a scoring rubric, run scenarios by hand, record scores in a spreadsheet. Pros: zero engineering effort. Cons: friction means it won't happen consistently.

**When this recommendation would change:** If the orchestrator skill stabilizes and changes become infrequent, the tooling investment isn't justified. This infrastructure is worth building only if Dylan is entering a phase of systematic skill iteration (which the task description confirms).

---

## Implementation-Ready Output

### File targets
- `cmd/orch/skill_lint_cmd.go` — Static linter command
- `pkg/skill/lint.go` — Lint rule implementations
- `pkg/skill/lint_test.go` — Lint rule tests
- Scenario YAML files in skill source directory
- (Phase 2) `cmd/orch/skill_test_cmd.go` — Test runner
- (Phase 2) `pkg/skill/test.go` — Scenario execution + scoring
- (Phase 3) `cmd/orch/skill_compare_cmd.go` — Comparison tool

### Acceptance criteria
- [ ] `orch skill lint SKILL.md` outputs warnings for MUST fatigue, cosmetic redundancy, signal imbalance
- [ ] Running current orchestrator skill through linter produces actionable output
- [ ] 7 scenario YAML files exist with behavioral indicators
- [ ] Manual execution of a scenario produces a scorable result
- [ ] (Phase 2) `orch skill test` automates scenario execution
- [ ] (Phase 3) `orch skill compare` shows per-scenario deltas

### Out of scope
- Testing worker skills (different problem, different infrastructure)
- Changes to the orchestrator skill itself (this designs the testing infrastructure)
- Statistical rigor (this is practitioner tooling, not academic research)
- LLM-as-judge scoring (rejected per provenance principle)
