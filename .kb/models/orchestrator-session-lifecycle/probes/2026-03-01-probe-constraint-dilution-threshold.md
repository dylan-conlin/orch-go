# Probe: Constraint Dilution Threshold — Does 3-Form Survive Competition?

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-01
**Status:** Complete

---

## Question

The redundancy saturation investigation (aj58) found that 3 structurally diverse forms achieve ceiling compliance (8/8, zero variance) for both knowledge and behavioral constraints — but only tested with a SINGLE constraint active in isolation.

**Claim under test:** The aj58 investigation's Finding 4 hypothesizes that "the total number of constraints in a skill document is bounded by the attention budget" and estimates 15-25 constraints as the ceiling. The complexity investigation (xm5q) found that full skill documents (50+ constraints) produce 0% delegation on complex tasks.

**Specific question:** At what constraint count does 3-form compliance drop below ceiling? Does constraint competition create a dilution curve, or is there a sharp threshold?

---

## What I Tested

**Test Design:** 6 variants with increasing constraint density, all using 3-form structural diversity (table + checklist + anti-patterns):

| Variant | Constraints | Word Count | Contains |
|---------|------------|------------|----------|
| Bare | 0 | 0 | Nothing |
| 1C-D | 1 | 196 | Delegation only |
| 1C-I | 1 | 241 | Intent only |
| 2C | 2 | 427 | Delegation + Intent |
| 5C | 5 | 971 | Both + 3 fillers |
| 10C | 10 | 1800 | Both + 8 fillers |

**Filler constraints:** Anti-sycophancy, phase reporting, no bd close, architect routing, session close protocol, beads tracking, context loading, tool restriction.

```bash
# 36 total test runs (6 variants × 3 runs × 2 scenarios)
skillc test --scenarios scenarios/ --variant variants/<name>.md --model sonnet --runs 3 --json --transcripts transcripts/
skillc test --scenarios scenarios/ --bare --model sonnet --runs 3 --json --transcripts transcripts/
```

---

## What I Observed

**Delegation Probe (behavioral constraint):**

| Variant | Scores | Median | proposes-delegation |
|---------|--------|--------|---------------------|
| Bare | [0, 5, 5] | 5/8 | 0/2 |
| 1C-D | [8, 8, 8] | **8/8** | **3/3** |
| 2C | [8, 8, 8] | **8/8** | **3/3** |
| 5C | [3, 8, 8] | 8/8 | 2/3 |
| 10C | [5, 5, 5] | 5/8 | **0/3** |

**Intent Probe (knowledge constraint):**

| Variant | Scores | Median | asks-clarification |
|---------|--------|--------|-------------------|
| Bare | [3, 6, 3] | 3/8 | 1/3 |
| 1C-I | [8, 6, 8] | **8/8** | 3/3 |
| 2C | [8, 6, 3] | 6/8 | 2/3 |
| 5C | [5, 8, 8] | **8/8** | 2/3 |
| 10C | [6, 8, 6] | 6/8 | 3/3 |

**Key observation:** At 10 constraints, the delegation constraint's active behavioral indicator (proposes-delegation) drops to 0/3 — identical to bare. The model avoids code reading and frames as delegation (passive) but never proposes spawning an agent (active). This is exactly the bare-level performance pattern.

---

## Model Impact

- [x] **Contradicts** invariant: aj58's claim that "the correct response to bare-parity violations is to isolate the constraint and express it in 3 structurally diverse forms." This only works in isolation or with 1-2 companion constraints. At 5+ competing constraints, variance returns. At 10, behavioral constraints regress to bare parity.

- [x] **Extends** model with: The attention budget hypothesis from aj58 Finding 4 is confirmed, but the budget is MUCH smaller than estimated (15-25). For behavioral constraints, the effective budget is ~2-4 co-resident constraints. Knowledge constraints have a higher budget (functional at 10). This implies the model needs a constraint type taxonomy: knowledge (prompt-safe, high budget) vs behavioral (infrastructure-required, low budget).

- [x] **Confirms** invariant: The baseline investigation's empirical finding that behavioral constraints fail in full skill documents. The mechanism (dilution) is now quantified: degradation starts at 5 constraints, reaches bare parity at 10, and the production skill (50+) is far beyond the threshold.

---

## Notes

- Prior work (aj58): 3-form = [8,8,8] for both constraints in isolation — confirmed as control
- Prior work (xm5q): Full skill (50+ constraints) = 0% delegation on complex tasks — confirmed as endpoint
- This probe fills the gap: dilution is gradual (not a cliff), behavioral threshold is 2-5, knowledge threshold is higher
- The aj58 recommendation to "apply 3-form to critical constraints" needs qualification: only viable if skill has ≤4 total behavioral constraints
- Full investigation: .kb/investigations/2026-03-01-inv-test-constraint-dilution-threshold.md

---

## ⚠️ Replication Failure Caveat (2026-03-04)

**The dilution curve (3/3→3/3→2/3→0/3) did not replicate under clean isolation (orch-go-zola).** Specific threshold numbers (behavioral budget ~2-4, degradation starts at 5, bare parity at 10) are unvalidated. The opus "confirmation" (identical curve shape) was noise matching noise at N=3 — two small-sample experiments agreeing does not constitute validation. All quantitative claims in this probe should be treated as directional hypotheses, not established findings.
