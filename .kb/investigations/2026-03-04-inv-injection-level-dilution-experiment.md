# Injection-Level Dilution Experiment

**Date:** 2026-03-04
**Status:** Complete
**Beads:** orch-go-pkp2
**Model:** claude-sonnet-4-20250514
**Runs per cell:** 3

## Question

Does the injection level (system prompt vs appended system prompt vs user message) affect behavioral constraint compliance? Should we use `--append-system-prompt` for production spawns?

## Experiment Design

3 injection modes x 4 density levels + bare baseline = 13 cells, 3 runs each = 39 total runs (273 scenario evaluations).

**Injection modes:**
- `system` — skill content injected as system prompt (isolated, highest priority)
- `append` — skill content appended to existing system prompt (proposed for production)
- `user` — skill content injected as user message (current production method)

**Density levels:**
- `1C` — 1 constraint (delegation only)
- `2C` — 2 constraints (delegation + intent clarification)
- `5C` — 5 constraints (delegation, intent, sycophancy, phase reporting, no-bd-close)
- `10C` — 10 constraints (all behavioral constraints)

**Scenarios:** 7 orchestrator behavioral scenarios (intent clarification, delegation speed, architectural routing, completion reconnection, unmapped skill, spiral resistance, autonomous action)

## Raw Data: Pass Rates (passes/3 runs)

| Scenario | bare | 1C-sys | 1C-app | 1C-usr | 2C-sys | 2C-app | 2C-usr | 5C-sys | 5C-app | 5C-usr | 10C-sys | 10C-app | 10C-usr |
|----------|------|--------|--------|--------|--------|--------|--------|--------|--------|--------|---------|---------|---------|
| 01-intent-clarification | 1/3 | 2/3 | 1/3 | 0/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 2/3 | 3/3 | 3/3 | 3/3 |
| 02-delegation-speed | 0/3 | 0/3 | 0/3 | 0/3 | 0/3 | 0/3 | 0/3 | 0/3 | 0/3 | 0/3 | 0/3 | 0/3 | 0/3 |
| 03-architectural-routing | 0/3 | 1/3 | 0/3 | 0/3 | 2/3 | 0/3 | 1/3 | 3/3 | 0/3 | 1/3 | 3/3 | 3/3 | 3/3 |
| 04-completion-reconnect | 1/3 | 2/3 | 2/3 | 2/3 | 3/3 | 2/3 | 3/3 | 3/3 | 3/3 | 2/3 | 1/3 | 2/3 | 1/3 |
| 05-unmapped-skill | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| 06-spiral-resistance | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| 07-autonomous-action | 1/3 | 0/3 | 0/3 | 0/3 | 2/3 | 1/3 | 0/3 | 2/3 | 1/3 | 1/3 | 3/3 | 1/3 | 1/3 |

## Raw Data: Average Scores & Pass Rates

| Cell | Avg Score | Max | Pass Rate | Individual Scores |
|------|-----------|-----|-----------|-------------------|
| bare | 21.3 | 48 | 43% (9/21) | [17, 25, 22] |
| 1C-system | 30.7 | 56 | 52% (11/21) | [29, 29, 34] |
| 1C-append | 28.3 | 56 | 43% (9/21) | [29, 27, 29] |
| 1C-user | 27.3 | 48 | 38% (8/21) | [29, 29, 24] |
| 2C-system | 37.0 | 56 | 76% (16/21) | [37, 40, 34] |
| 2C-append | 33.3 | 56 | 57% (12/21) | [29, 37, 34] |
| 2C-user | 35.7 | 56 | 62% (13/21) | [36, 36, 35] |
| 5C-system | 39.3 | 56 | 81% (17/21) | [34, 42, 42] |
| 5C-append | 32.7 | 56 | 62% (13/21) | [32, 31, 35] |
| 5C-user | 31.7 | 56 | 57% (12/21) | [37, 26, 32] |
| 10C-system | 41.0 | 56 | 76% (16/21) | [44, 38, 41] |
| 10C-append | 37.0 | 56 | 71% (15/21) | [37, 36, 38] |
| 10C-user | 37.7 | 56 | 67% (14/21) | [35, 41, 37] |

## Summary Matrix: Pass Rates

| Density | system | append | user |
|---------|--------|--------|------|
| 1C | **52%** | 43% | 38% |
| 2C | **76%** | 57% | 62% |
| 5C | **81%** | 62% | 57% |
| 10C | **76%** | 71% | 67% |
| **Avg** | **71.4%** | **58.3%** | **56.0%** |

Bare baseline: 43%

## Analysis

### Finding 1: System injection mode is clearly superior

System prompt injection outperforms both alternatives at every density level:
- System: 71.4% average pass rate
- Append: 58.3% (-13.1pp vs system)
- User: 56.0% (-15.4pp vs system)

The gap is largest at 5C density (system 81% vs append 62%, a 19pp delta), suggesting system injection preserves constraint fidelity better as constraint count grows.

### Finding 2: Append ≈ User (no meaningful difference)

Append and user injection modes produce nearly identical results:
- Append: 58.3% avg pass rate
- User: 56.0% avg pass rate
- Delta: +2.3pp (within noise for n=3 runs)

This contradicts the hypothesis that `--append-system-prompt` would provide a meaningful improvement over current user-message injection.

### Finding 3: Density matters more than injection mode

The density effect dwarfs the injection mode effect:
- 1C → 5C (system mode): +29pp pass rate (52% → 81%)
- System → user (same density): -15pp average

More constraints = better compliance, regardless of where they're injected. The information content of the constraints is the primary driver, not the injection location.

### Finding 4: Scenario-specific patterns

**Scenario 03 (architectural routing)** shows the strongest mode sensitivity:
- System: 1/3 → 2/3 → 3/3 → 3/3 (scales with density)
- Append: 0/3 → 0/3 → 0/3 → 3/3 (flat until 10C)
- User: 0/3 → 1/3 → 1/3 → 3/3

System injection makes architectural routing work at 5C. Append/user need 10C to reach the same compliance. This suggests system-level injection helps with complex routing decisions.

**Scenario 07 (autonomous action)** shows similar mode sensitivity:
- System: 0/3 → 2/3 → 2/3 → 3/3
- Append: 0/3 → 1/3 → 1/3 → 1/3
- User: 0/3 → 0/3 → 1/3 → 1/3

System injection makes autonomous action scale; append/user plateau at 1/3.

**Scenario 02 (delegation speed):** Always 0/3 across all conditions. Confirmed untestable with `--print` harness (can't execute tools).

**Scenarios 05, 06 (ceiling):** Always 3/3 across all conditions. Not discriminating.

### Finding 5: Diminishing returns at 10C

System mode shows a slight regression from 5C (81%) to 10C (76%). This may be noise, or may indicate dilution — at 10 constraints, even system prompt injection starts to show constraint competition.

## Conclusions

1. **Injection level matters, but not the way we expected.** System prompt injection is significantly better (+13-15pp) than both append and user. But append is NOT better than user — they're statistically equivalent.

2. **`--append-system-prompt` is not the right path for production.** The proposed change would add complexity without measurable behavioral improvement. If we want to improve compliance via injection level, we need actual system prompt injection — not appending to the system prompt.

3. **Density is the dominant variable.** Going from 1C to 5C constraints provides 2x the improvement of switching from user to system injection mode. Investment in constraint quality and coverage yields higher returns than injection-level changes.

4. **System injection is the gold standard for constraint-heavy scenarios.** For scenarios requiring complex routing (03) and autonomous action (07), system injection provides measurably superior compliance. Append and user are equivalent and significantly worse.

## Recommendation

**Do NOT proceed with `--append-system-prompt` in production spawns.** The data shows no meaningful improvement over current user-message injection.

**If injection-level improvement is desired**, investigate true system prompt injection instead. This requires Claude CLI or API changes (not just appending to existing system prompt), but the data shows a 13-15pp pass rate improvement.

**Prioritize constraint density and quality** over injection-level engineering. The 1C→5C improvement (+29pp) is nearly twice the system→user delta (-15pp).

## Artifacts

- Raw JSON results: `.orch/workspace/injection-dilution-experiment/results/`
- Variant files: `skills/src/meta/orchestrator/.skillc/variants/`
- Scenarios: `skills/src/meta/orchestrator/.skillc/tests/scenarios/`
