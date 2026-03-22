# Probe: Problem-Surface Constraints — The Harness-Engineering Blind Spot

**Model:** harness-engineering
**Date:** 2026-03-22
**Status:** Complete
**claim:** HE-01 through HE-13 (all claims)
**verdict:** extends

---

## Question

Does the harness-engineering model have any claims about problem-surface constraints — narrowing what agents work on so enforcement machinery isn't needed? Or are all 13 claims exclusively about behavioral enforcement (gates, hooks, rules for how agents work)?

Secondary: Are there existing orch-go patterns that already use problem constraints without naming them?

---

## What I Tested

**1. Full model text search for problem-surface constraint language:**

```bash
grep -i "problem.*(constraint|surface|scope|narrow|simplif)" .kb/models/harness-engineering/model.md
# Result: 0 matches

grep -i "constrain.*(solution|problem|space|surface|scope)" .kb/models/harness-engineering/model.md
# Result: 4 matches — all about "constraining the solution space" (Fowler quote)
# or "constraining agent behavior." Zero about constraining the PROBLEM.

grep -i "light|tier.*light|single.file|domain.harness|narrow.*scope" .kb/models/harness-engineering/model.md
# Result: 0 matches
```

**2. Claims.yaml exhaustive review (HE-01 through HE-13):**

Categorized each claim by what it constrains:

| Claim | Topic | Constrains Problem? | Constrains Behavior? |
|-------|-------|---------------------|---------------------|
| HE-01 | Hard vs soft harness | No | Yes — enforcement types |
| HE-02 | Convention without gate = violated | No | Yes — enforcement coverage |
| HE-03 | Agent failure = harness failure | No | Yes — blame attribution |
| HE-04 | Extraction without routing = pump | No | Yes — extraction mechanism |
| HE-05 | Prevention > Detection > Rejection | No | Yes — enforcement timing |
| HE-06 | Mutable hard harness = soft harness | No | Yes — enforcement durability |
| HE-07 | Enforcement without measurement = theological | No | Yes — enforcement observability |
| HE-08 | Stronger models need more coordination gates | No | Yes — gate scaling |
| HE-09 | Skill content maps to harness types | No | Yes — content taxonomy |
| HE-10 | Gates work through signaling not blocking | No | Yes — gate mechanism |
| HE-11 | Gate calibration death spiral | No | Yes — gate precision |
| HE-12 | Compliance vs coordination failure | No | Yes — failure taxonomy |
| HE-13 | Both attractors AND gates required | No | Yes — enforcement pairing |

**Score: 0/13 claims about problem-surface constraints. 13/13 about behavioral enforcement.**

**3. Codebase search for existing problem-constraint patterns:**

Searched `pkg/spawn/`, `pkg/daemon/`, `skills/src/`, `cmd/orch/`, `.harness/` for patterns that narrow what agents work on rather than constraining how they work.

**4. autoresearch comparison (from investigation .kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md):**

autoresearch: 1 file, 1 metric, 5-min runs, keep/discard via git. Zero gates, zero hooks, zero governance. 48k stars.

---

## What I Observed

### Finding 1: The model has zero problem-surface claims

All 13 claims, all 5 "Constraints" section entries, all 8 "Why This Fails" entries, and all 4 design principles are about behavioral enforcement — how to constrain agent behavior after the problem is already defined. The word "problem" appears only in the context of "problem files" (hotspots) and "coordination problem."

The Fowler quote (model.md line 142) says "constraining the solution space" but was interpreted exclusively as "add more gates." The model's response to this quote was: build 14 gates, 12 hooks, 87 behavioral constraints, a 4-layer gate stack. The alternative interpretation — "simplify the problem so gates aren't needed" — is absent.

### Finding 2: orch-go already HAS 7 problem-constraint patterns — unnamed

| Pattern | What It Constrains | Files | Eliminates What? |
|---------|-------------------|-------|------------------|
| **Spawn tiers** (light/full) | Synthesis requirements | `pkg/spawn/config.go:17-47` | SYNTHESIS.md gate for light-tier |
| **Verification levels** (V0-V3) | Which completion gates fire | `pkg/spawn/verify_level.go:6-55` | V0 eliminates ~11 of 14 gates |
| **--explore decomposition** | Task scope via parallel narrowing | `cmd/orch/spawn_cmd.go:243-264` | Scope-creep governance |
| **Daemon skill routing** | Skill assignment based on hotspot | `pkg/daemon/coordination.go:40-98` | Post-hoc accretion enforcement |
| **Hotspot extraction** | Pre-spawn blocking + extraction | `pkg/daemon/coordination.go:47-77` | Feature work on bloated files |
| **Domain harness** (OpenSCAD) | Valid parameter/geometry space | `.harness/openscad/CLAUDE.md:124-132` | Behavioral review of designs |
| **Issue type scoping** | Min verification by issue type | `pkg/spawn/verify_level.go:46-55` | Gate overhead for simple tasks |

These patterns are doing problem-constraint work but the model doesn't recognize them as such. They're scattered across 4 packages with no unifying principle.

### Finding 3: The Fowler quote has two valid interpretations

> "constraining the solution space" — the opposite of what most expect from AI coding

**Interpretation A (current model):** Constrain the *behavior* of agents — add gates, hooks, rules that prevent wrong paths. This produces the 14-gate completion pipeline, 12 deny hooks, 4-layer enforcement stack.

**Interpretation B (missing from model):** Constrain the *problem* agents work on — narrow scope, single-file focus, scalar metrics, fixed budgets — so behavioral enforcement is unnecessary. This produces autoresearch: 1 file, 1 metric, zero governance.

The model adopted Interpretation A exclusively. Interpretation B is what autoresearch proves works.

### Finding 4: Where problem constraints could reduce gate count

| Current Workflow | Gates Required | With Problem Constraint | Gates Eliminated |
|-----------------|---------------|------------------------|------------------|
| Investigation on open-ended question | V1 (artifacts + synthesis) | --explore → 3 narrow probes | Scope-creep governance |
| Feature-impl on hotspot file | V2 + accretion + hotspot gate | Daemon routes to architect first | Accretion gate, hotspot gate |
| OpenSCAD design iteration | Domain 5-layer + general 14-gate | Domain harness with tight constraints | ~10 general gates irrelevant |
| Issue-creation task | V0 (acknowledge only) | Already constrained — light tier | 11 of 14 gates already skipped |
| Optimization task (scalar metric) | V2 + synthesis + explain-back | autoresearch pattern (1 metric, keep/discard) | All governance except build gate |

The V0 verification level is already an implicit problem constraint — "this task is simple enough that most gates don't apply." The model doesn't frame this as a problem constraint; it frames it as gate selection.

---

## Model Impact

- [x] **Extends** model with: A structural blind spot — the entire model covers behavioral constraints (13/13 claims) with zero coverage of problem-surface constraints. The model already USES problem constraints (7 patterns found) without recognizing them as a distinct enforcement strategy. Proposed new claim HE-14: problem-surface constraint design can eliminate behavioral enforcement machinery.

**Proposed claim HE-14:**

```
text: "Problem-surface constraints (narrowing what agents work on) can eliminate behavioral
  enforcement (gates/hooks constraining how agents work). orch-go already uses 7 unnamed
  problem-constraint patterns (spawn tiers, verification levels, skill routing, --explore
  decomposition, hotspot extraction, domain harnesses, issue type scoping). autoresearch
  proves the extreme case: total problem constraint, zero behavioral enforcement."
type: mechanism
scope: bounded
confidence: confirmed
priority: core
evidence:
  - source: "13/13 HE claims about behavioral enforcement, 0/13 about problem constraints — structural blind spot"
    date: "2026-03-22"
    verdict: extends
  - source: "7 existing orch-go patterns that use problem constraints without naming them"
    date: "2026-03-22"
    verdict: extends
  - source: "autoresearch (48k stars): 1 file, 1 metric, zero gates — problem constraint eliminates governance"
    date: "2026-03-22"
    verdict: extends
  - source: "V0 verification already eliminates ~11/14 gates via implicit problem constraint"
    date: "2026-03-22"
    verdict: confirms
falsifies_if: "A tightly-constrained problem surface (single file, scalar metric, fixed budget)
  that still requires behavioral enforcement to produce correct outputs (N>20 runs)"
```

---

## Notes

**The relationship to HE-01 (hard > soft):** Problem-surface constraints are a third category — neither hard harness (infrastructure enforcement) nor soft harness (context/instructions). They work by making enforcement unnecessary rather than making it harder or softer. HE-01's taxonomy has a gap: hard/soft is about enforcement TYPE, but doesn't address whether enforcement is NEEDED.

**The autoresearch comparison is asymmetric:** autoresearch works because its problem is naturally narrow (optimize one file toward one metric). Most orch-go tasks are open-ended and multi-file. The insight isn't "be like autoresearch" — it's "before adding gates, ask whether the problem can be narrowed enough to not need them." The 7 existing patterns prove orch-go already does this pragmatically.

**The Fowler quote's dual interpretation suggests a design principle:** When you encounter a governance failure, two responses are available: (1) add enforcement (gate, hook, rule), or (2) narrow the problem so enforcement isn't needed (decompose, route, scope, tier). The model currently only recommends (1).
