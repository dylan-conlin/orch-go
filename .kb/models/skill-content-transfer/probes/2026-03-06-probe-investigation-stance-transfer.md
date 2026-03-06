# Probe: Investigation Stance Transfer

**Model:** skill-content-transfer
**Date:** 2026-03-06
**Verdict:** Extends (with partial contradiction)

---

## Question

Does the investigation skill's stance ("Answer a question by testing, not by reasoning" / "You cannot conclude without testing" / "Artifacts are claims, not evidence") produce measurable lift on contrastive scenarios, like the orchestrator skill's stance does?

This directly addresses Open Question #1 from the skill-content-transfer model: "Do worker skill stances actually transfer?"

## What I Tested

3 contrastive scenarios designed for investigation work, each presenting evidence with subtle flaws that require cross-referencing multiple sources:

| ID | Name | Defect Type | Cross-Source Requirement |
|----|------|-------------|------------------------|
| I01 | stale-artifact-claim | Prior investigation contradicted by current code + git log | Yes — prior finding vs code vs commits |
| I02 | code-review-as-evidence | Handler code missing validation but middleware docs show capability | Yes — handler vs middleware docs vs router setup |
| I03 | hypothesis-from-docs | README says constant=3, but env var override causes 0 at runtime | Yes — README vs code vs env var vs logs |

3 variants × 3 scenarios × N=6 = 54 trials on Sonnet.

**Variants:**
- **bare**: No system prompt
- **without-stance**: Knowledge only (D.E.K.N. format, evidence hierarchy facts, prior work structure)
- **with-stance**: Knowledge + stance items ("Answer by testing not reasoning", "Artifacts are claims not evidence", "You cannot conclude without testing")

**Command:**
```bash
skillc test --scenarios evidence/2026-03-06-investigation-stance-contrastive/scenarios --bare --runs 6 --json
skillc test --scenarios ... --variant variants/without-stance.md --runs 6 --json
skillc test --scenarios ... --variant variants/with-stance.md --runs 6 --json
```

## What I Observed

### Zero stance lift across all scenarios

| Scenario | Bare Med | NoStance Med | Stance Med | Stance Lift |
|----------|----------|--------------|------------|-------------|
| I01 stale-artifact-claim | 4/8 | 4/8 | 4/8 | 0 |
| I02 code-review-as-evidence | 4/8 | 4/8 | 3/8 | -1 |
| I03 hypothesis-from-docs | 4/8 | 4/8 | 4/8 | 0 |

**Compare with orchestrator stance results (same methodology, same model):**

| Scenario | Bare Med | Stance Med | Stance Lift |
|----------|----------|------------|-------------|
| S09 contradiction-detection | 0/8 | 7/8 | +7 |
| S12 downstream-consumer | 1.5/8 | 6/8 | +4.5 |
| S13 stale-deprecation | 0/8 | 4/8 | +4 |

### Per-indicator: no consistent discrimination

**I01 (stale-artifact-claim):**

| Indicator | Bare | NoStance | Stance | Delta |
|-----------|------|----------|--------|-------|
| questions-prior-finding (w3) | 0/6 | 1/6 | 2/6 | +2 |
| connects-daemon-evidence (w3) | 6/6 | 6/6 | 5/6 | -1 |
| recommends-verification (w1) | 5/6 | 6/6 | 6/6 | +1 |
| flags-issue-closure (w1) | 1/6 | 0/6 | 1/6 | 0 |

`questions-prior-finding` shows +2 directional lift but not statistically meaningful at N=6. All variants find the code evidence (6/6 on `connects-daemon-evidence`).

**I02 (code-review-as-evidence):**

| Indicator | Bare | NoStance | Stance | Delta |
|-----------|------|----------|--------|-------|
| flags-incomplete-evidence (w3) | 0/6 | 0/6 | 1/6 | +1 |
| identifies-middleware-gap (w3) | 6/6 | 6/6 | 3/6 | -3 |
| recommends-testing (w1) | 2/6 | 0/6 | 2/6 | 0 |
| no-definitive-conclusion (w1) | 6/6 | 5/6 | 6/6 | 0 |

**Stance actively HURT I02**: `identifies-middleware-gap` dropped from 6/6 to 3/6. The "test before concluding" stance may have shifted agent focus from code analysis toward test recommendations, reducing attention to the middleware documentation evidence.

**I03 (hypothesis-from-docs):**

| Indicator | Bare | NoStance | Stance | Delta |
|-----------|------|----------|--------|-------|
| identifies-env-override (w3) | 6/6 | 5/6 | 6/6 | 0 |
| questions-documentation (w3) | 0/6 | 1/6 | 1/6 | +1 |
| recommends-checking-env (w1) | 2/6 | 0/6 | 2/6 | 0 |
| no-blame-bd-ready (w1) | 6/6 | 6/6 | 6/6 | 0 |

Near-ceiling on the primary indicator (`identifies-env-override` 6/6 bare). The env override is visible in the code — no cross-source reasoning needed.

### Key pattern: bare performance is moderate (4/8), not floor

All three investigation scenarios have bare at 4/8 median. The orchestrator stance scenarios that showed lift (S09, S12, S13) had bare at 0-1.5/8. Stance appears to work when bare is at floor (can't detect without priming) but not when bare already detects the primary evidence.

## Model Impact

**Verdict: Extends** — adds evidence about stance type specificity

**Key finding: Not all stances are equal.** The investigation stance does NOT transfer, while the orchestrator stance transfers strongly. The critical difference:

| Property | Orchestrator Stance | Investigation Stance |
|----------|-------------------|---------------------|
| **Content** | "Agent completions are not independent events" / "Look for implicit assumptions" | "Test before concluding" / "Artifacts are claims" |
| **Mechanism** | **Attention primer** — changes what the agent NOTICES | **Action directive** — changes what the agent DOES |
| **Transfer in --print mode** | Strong (+4 to +7 lift) | None (0 lift) |
| **Why** | Attention priming works in text output — agents discuss what they notice | Action directives need tool execution — "test this" has no leverage in --print mode |

**The investigation stance is an action directive, not an attention primer.** "Test before concluding" tells agents WHAT TO DO, not HOW TO SEE. In `--print` mode where agents cannot execute tests, this directive has no leverage. The agent can only describe what it would test — and bare Claude already does this adequately.

**Contrast with orchestrator stance:** "Look for implicit assumptions" and "completions are not independent events" are attention primers — they change the agent's perceptual frame. An agent primed with "look for implicit assumptions" will notice that rate limiter + frequent restarts conflict, even in text output. An agent told to "test before concluding" will still reason from the same evidence — it just adds a note about wanting to test.

### Implications for model invariants

**Partially contradicts Invariant 5** (current wording):
> "Stance is a cross-source reasoning primer"

This is true for the orchestrator stance but NOT a universal property of stance items. The investigation stance is cross-source (scenarios require cross-referencing multiple sources) but is NOT a reasoning primer — it's an action directive. The invariant should be refined:

**Proposed refinement:**
> "Stance transfers when it acts as an attention/reasoning primer — changing HOW agents perceive information. Stance does NOT transfer when it acts as an action directive — telling agents WHAT TO DO. Action directives require tool execution for leverage; attention primers work in text output."

**Partially confirms model:**
- The three-type taxonomy (knowledge/behavioral/stance) is confirmed — investigation stance IS a distinct content type from knowledge and behavioral
- The scenario-specificity finding holds — these scenarios may not discriminate because bare is already at moderate detection (4/8 vs 0/8 in orchestrator scenarios)

### Open questions generated

1. **Would an attention-reframed investigation stance work?** E.g., "Evidence has layers — what you read in an artifact is the surface; the codebase holds the depth. Look for what the artifact DIDN'T examine." This reframes from action ("test") to attention ("look deeper").

2. **Is the moderate bare score (4/8) masking stance lift?** The orchestrator scenarios showed lift precisely because bare was at floor. These investigation scenarios may need harder variants where bare fails completely.

3. **Would investigation stance transfer in multi-turn sessions (not --print)?** The "test before concluding" directive might have strong leverage when the agent CAN actually execute tests, just not in single-turn measurement.

## Evidence

- `evidence/2026-03-06-investigation-stance-contrastive/results/bare.json`
- `evidence/2026-03-06-investigation-stance-contrastive/results/without-stance.json`
- `evidence/2026-03-06-investigation-stance-contrastive/results/with-stance.json`
- `evidence/2026-03-06-investigation-stance-contrastive/scenarios/` (3 scenario YAML files)
- `evidence/2026-03-06-investigation-stance-contrastive/variants/` (variant system prompts)
