# OpenSCAD Harness: Self-Measurement Report

*March 16, 2026*

---

## 1. What This Harness Is

This is a measurement report for an enforcement harness applied to 3D modeling — specifically, OpenSCAD parametric design orchestrated by AI agents.

The harness lives at `.harness/openscad/` within orch-go, the multi-agent orchestration system. It was designed as a **portability test**: can the gate-based enforcement methodology that evolved over 3 months in a Go codebase transfer to a fundamentally different domain?

Code enforcement gates check syntax, compilation, and test results. OpenSCAD enforcement gates check parameter validity, geometric correctness, printability, and design intent. The question: does the same framework (hard gates → soft gates → measurement pairing) produce useful enforcement in both?

### Current Inventory

| Component | Files | Lines | Purpose |
|-----------|-------|-------|---------|
| Parts | 1 (`bracket.scad`) | 62 | Sample parametric design with assert gates |
| Validation library | 1 (`validate.scad`) | 63 | 7 reusable validation modules, 18 assert/GATE FAIL patterns |
| Geometry gate (L2) | 1 (`geometry-check.sh`) | 72 | CGAL manifold check, polygon budget, degenerate detection |
| Intent gate (L4) | 1 (`intent-check.sh`) | 148 | LLM-as-gate for design-spec alignment |
| Test suite | 1 (`test-gates.sh`) | 254 | 14 integration tests across Layer 1-2 |
| CLAUDE.md | 1 | 125 | Agent constraints and workflow |
| Skills | 2 (`design-part`, `iterate-design`) | — | Agent skill templates |
| **Total** | **8 files** | **724 lines** | — |

This is small. Intentionally. The orch-go harness evolved over 1,626 commits and 80K+ lines of code. The OpenSCAD harness is 724 lines, 1 sample part, and 0 production agents. Everything in this report should be read through that lens.

---

## 2. What We Enforce

### Layer 1: Parameter Validation (assert gates, in-process)

**Type:** Execution (hard)
**Mechanism:** `assert()` calls in `.scad` files, with `GATE FAIL:` prefix for grep-able error extraction
**Enforcement point:** During OpenSCAD render — exit code 1 on violation

The validation library (`lib/validate.scad`) provides 7 reusable modules:

| Module | What It Validates |
|--------|-------------------|
| `validate_range` | Parameter within min/max bounds |
| `validate_positive` | Dimension > 0 |
| `validate_wall_thickness` | Wall >= 0.8mm (FDM minimum) |
| `validate_fn` | $fn in 16-256 range |
| `validate_hole` | Hole diameter >= 1.0mm |
| `validate_interior` | Wall doesn't consume entire dimension |
| `validate_part` | Batch validation (calls all above) |

The sample bracket uses `validate_part()` as its entry point, covering: positive dimensions, wall thickness >= 0.8mm, $fn bounds 16-256, interior space exists, hole diameter >= 1.0mm, and fillet radius within wall_thickness bound.

**What these catch:** Degenerate parameters, negative dimensions, physically impossible constraints, parameters that would produce unprintable geometry.

**What these miss:** Valid parameters that produce bad geometry (self-intersection, non-manifold, excessive overhangs). That's Layer 2-3's job.

### Layer 2: Geometry Validation (post-render, shell script)

**Type:** Execution (hard)
**Mechanism:** `geometry-check.sh` wrapping OpenSCAD CGAL backend
**Enforcement point:** After render, before export

Five checks:
1. **CGAL render success** — exit code 0 (catches syntax errors, resource exhaustion)
2. **Manifold check** — grep for non-manifold warnings in CGAL output
3. **Polygon budget** — facet count from `--summary-file` JSON, block >200K, warn >50K
4. **Degenerate detection** — STL output < 200 bytes = empty/degenerate geometry
5. **Bounding box sanity** — optional dimension tolerance check via env vars

**What these catch:** Non-manifold geometry that Manifold backend silently repairs, polygon explosions from high $fn, empty renders, render failures.

**What these miss:** Printability (wall thickness in rendered mesh, overhang angles, support requirements). That requires a slicer — Layer 3.

### Layer 3: Printability Validation (slicer CLI)

**Status: Not implemented.** Design exists. PrusaSlicer CLI integration has not been tested.

### Layer 4: Intent Alignment (LLM-as-gate)

**Type:** Judgment (soft)
**Mechanism:** `intent-check.sh` calling Claude CLI with structured prompt
**Enforcement point:** After render, advisory only (fail-open)

Judges design against spec on 4 criteria:
1. Dimensional compliance (±5% tolerance)
2. Feature completeness
3. Constraint satisfaction (wall thickness, overhang, holes)
4. Purpose fit

**Status: Implemented but untested at scale.** The script exists, the structured output format is defined, the fail-open behavior works. But no measurement dataset exists — precision and recall are unknown.

---

## 3. The Numbers

### 3.1 Test Results (Layer 1-2)

The test suite (`test-gates.sh`) runs 14 integration tests against real OpenSCAD CLI:

| Layer | Tests | Passed | Failed |
|-------|-------|--------|--------|
| Layer 1 (assert gates) | 9 | 8 | 1 |
| Layer 2 (geometry validation) | 5 | 5 | 0 |
| **Total** | **14** | **13** | **1** |

**The 1 failure is revealing.** Test 9 ("thin-but-valid wall emits warning") sets `wall_thickness=1.0` and expects the bracket to render with a warning. Instead, it fails with:

```
GATE FAIL: fillet_radius=2 > 1 maximum
```

This is correct behavior: `validate_range("fillet_radius", fillet_radius, 0, min(wall_thickness, 5))` sets max to `min(1.0, 5) = 1.0`, and the default `fillet_radius=2` exceeds it. The gate is catching a real constraint violation — the test expectation was wrong, not the gate.

**What this tells us:** The parameter validation gates interact. Changing one parameter (`wall_thickness`) changes the valid range of another (`fillet_radius`). This is analogous to orch-go's finding that gates compose in non-obvious ways. The test failed because the test didn't account for inter-parameter dependencies, not because the enforcement was wrong.

### 3.2 Gate Coverage (What Is Actually Checked)

| Gate | Scope | Automated? | Coverage |
|------|-------|------------|----------|
| Parameter asserts (L1) | Every render | Yes (in-process) | 100% of renders |
| Geometry validation (L2) | Manual invocation | Requires running `geometry-check.sh` | 0% (no automated pipeline) |
| Printability (L3) | Not implemented | No | 0% |
| Intent alignment (L4) | Manual invocation | Requires running `intent-check.sh` | 0% |
| Accretion (source) | Not implemented | No | 0% |
| Accretion (output) | In geometry-check.sh | Yes (when L2 runs) | Same as L2 |

**Only Layer 1 runs automatically.** Every other layer requires manual invocation. There is no completion pipeline, no daemon, no automated gate sequencing. In orch-go terms: enforcement exists but no orchestration exists to ensure it runs.

This is the most important finding in this report. The gates are implemented. They are not wired.

### 3.3 Harness Size Comparison

| Metric | orch-go (3 months) | OpenSCAD (day 1) | Ratio |
|--------|---------------------|-------------------|-------|
| Total harness lines | ~80,000+ | 724 | ~110:1 |
| Gate scripts/code | ~15 gates, ~3,000 lines | 4 layers, ~283 lines | ~11:1 |
| Test coverage | 14 gate tests + unit tests | 14 integration tests | 1:1 (tests) |
| Parts/code under enforcement | ~80K lines Go | 62 lines .scad | ~1,300:1 |
| Production agent runs | 449 spawns in 8 days | 0 | ∞ |
| Event telemetry | 15,037 events | 0 events | ∞ |
| Gate fire data | 30 days continuous | 14 test invocations | — |

The comparison is not flattering to the OpenSCAD harness, and shouldn't be. orch-go's measurement infrastructure exists because it learned the hard way that unmeasured gates are theater. The OpenSCAD harness doesn't have production data because no production agents have run.

---

## 4. What We'd Like to Claim But Can't Yet

### "The gate taxonomy is domain-general"

The architect investigation (orch-go-gv8lv) asserts that execution/evidence/judgment gate types map 1:1 to OpenSCAD. The Layer 1-2 test results support this: assert gates are execution gates (exit code 0/1), geometry checks are evidence gates (parse JSON metrics), and the intent gate is a judgment gate (LLM verdict).

But: 13 tests on 1 sample part is not validation. The orch-go self-measurement report draws from 449 spawns and 15,037 events. We have 14 test invocations and 0 production runs.

**Status: Structurally plausible, empirically untested.**

### "LLM-as-gate escapes the closed-loop problem"

The argument: orch-go rejected LLM-as-judge for code review because the reviewing LLM has the same blind spots as the generating LLM. The OpenSCAD intent gate has a structural difference — it judges rendered geometric output (STL metadata, PNG, dimensions) against textual intent, not code against code. Different input modalities may escape the shared-blind-spot problem.

But: this is a structural argument, not an empirical one. The measurement protocol (100 design-spec pairs, 50 aligned, 50 drifting) has been designed but not executed. Precision and recall are unknown.

**Status: Hypothesis with measurement protocol designed. Zero data.**

### "Accretion transfers to 3D modeling"

The accretion boundaries (300/600 line warning/block, 50K/200K polygon warning/block, 15/30 module warning/block, 8/15 boolean depth warning/block) are heuristic — calibrated from orch-go experience with code, adapted by rough analogy to OpenSCAD.

But: no OpenSCAD projects have hit these thresholds. No agents have been observed creating progressively larger .scad files. The accretion phenomenon in orch-go emerged from 30+ agents adding features independently over months. We have 1 hand-authored bracket.

**Status: Thresholds defined, calibration data nonexistent.**

### "The harness engineering framework is portable"

The thread (`.kb/threads/2026-03-16-cross-domain-harness-portability-can.md`) framed three hypotheses:
- H1: Gate taxonomy is domain-general
- H2: Enforcement+measurement pairing transfers
- H3: LLM judgment gates are viable with measurement

All three are structurally plausible. None are empirically validated. The OpenSCAD harness proves the framework can be *instantiated* in a new domain. It does not prove the instantiation *works*.

**Status: Framework instantiated. Working unknown.**

---

## 5. What the Numbers Falsify

### 5.1 Falsification Criteria (adapted from orch-go)

| Criterion | Measurement | Kill Threshold | Current Status |
|-----------|-------------|----------------|----------------|
| Gates are ceremony | Gate fire rate in production | Fire rate <5% across all gates for 30d | **NO DATA** — no production agents |
| Gates are irrelevant | Gate block rate | Zero blocks in 30d of production use | **NO DATA** — no production agents |
| Soft harness is inert | Intent gate precision | Precision <65% (not better than coin flip) | **NO DATA** — measurement dataset not built |
| Framework is anecdotal | Benefit in OpenSCAD domain | No measurable quality improvement vs no-harness baseline | **NO DATA** — no baseline, no comparison |
| Accretion thresholds wrong | Calibration against real parts | >50% of legitimate designs blocked by thresholds | **NO DATA** — 1 design, below all thresholds |

**Every falsification criterion returns NO DATA.** This is the honest state of the harness. The enforcement mechanisms exist. The measurement infrastructure to evaluate them does not.

### 5.2 What Test 9 Tells Us

The single test failure is the most empirically interesting finding. It demonstrates:

1. **Parameter interaction complexity** — Reducing `wall_thickness` from 3→1.0 changes `fillet_radius` max from 3→1.0, making the default fillet invalid. Multi-parameter constraints create non-obvious failure surfaces.

2. **Test design reveals gate behavior** — The test assumed `wall_thickness=1.0` was the only change being tested. The gate caught a second-order effect. This mirrors orch-go's finding that gate interaction effects are unmeasured.

3. **The gate was right, the expectation was wrong** — A well-calibrated gate that blocks unexpectedly is doing its job. The human expectation was incomplete, not the enforcement.

---

## 6. Comparison with orch-go Methodology

### What transferred (methodology)

| Methodology Element | orch-go | OpenSCAD |
|---------------------|---------|----------|
| Gate type taxonomy (execution/evidence/judgment) | 3 types, 15+ gates | 3 types, 4 layers |
| Enforcement+measurement pairing | Every gate has a measurement surface | Designed but not implemented |
| Falsification criteria with thresholds | 4 criteria, 1 falsified, 2 insufficient data | 5 criteria, all NO DATA |
| Honest reporting of gaps | 22/37 gates with zero data | 4/4 layers lacking production data |
| Self-measurement report format | Sections 1-6, appendices | Same structure |

### What hasn't transferred (operational infrastructure)

| Infrastructure | orch-go | OpenSCAD |
|----------------|---------|----------|
| Event telemetry (`events.jsonl`) | 15,037 events | None |
| Automated completion pipeline | 18 gates, sequenced | None |
| Daemon orchestration | OODA cycle, auto-spawn | None |
| Harness audit command | `orch harness audit` | None |
| Accretion measurement | `accretion.delta` events | None |
| Gate decision logging | `spawn.gate_decision` events | None |

The methodology transferred. The infrastructure did not. This is the gap between "designing a harness" and "operating a harness."

### The orch-go self-measurement report's key insight applies here

> "Enforcement without measurement is theological. You believe the gate works. You have no evidence."

The OpenSCAD harness is currently theological. The gates exist. The tests prove they fire correctly in controlled conditions. But no production agent has ever been constrained by them. We have enforcement without operation, and therefore enforcement without measurement.

---

## 7. What Would Make This Empirical

### Phase 1: Run agents through the harness

Spawn 10+ design tasks through the OpenSCAD harness with real agents. Track:
- Gate fire rate per layer
- False positive rate (gates that block valid work)
- Time per gate evaluation
- Whether agents follow CLAUDE.md constraints without hard enforcement

### Phase 2: Build the intent gate dataset

Construct 100 design-spec pairs (50 aligned, 50 drifting). Measure:
- Intent gate precision and recall
- Confidence calibration
- Cost per evaluation
- Drift type detection rates

### Phase 3: Calibrate accretion thresholds

Have agents build 20+ parts of increasing complexity. Measure:
- Polygon count distribution across real designs
- Boolean depth distribution
- Where thresholds trigger vs. where they should trigger
- Source-size to output-size relationship

### Phase 4: Compare with orch-go

With production data from both domains:
- Do the same gate types fire at similar rates?
- Does the hard/soft boundary fall in the same place?
- Does accretion manifest the same way (uncoordinated growth)?
- Does the measurement methodology produce the same kinds of insights?

---

## 8. Summary

The OpenSCAD harness is a structural proof that the orch-go enforcement framework can be instantiated in a non-code domain. The gate taxonomy maps cleanly. The 4-layer design (parameter → geometry → printability → intent) parallels the code domain (syntax → compilation → testing → review). The measurement methodology and falsification criteria transfer directly.

But "can be instantiated" is not "works." The orch-go self-measurement report drew its credibility from uncomfortable numbers: 5/8 dormant gates, 65% detector precision, 40% orphan rate. Those numbers required production operation and measurement infrastructure.

The OpenSCAD harness has neither. It is a hypothesis with 724 lines of supporting code and 13 passing tests. That's a starting point, not a conclusion.

**What we can claim:** The framework structure transfers. Gates can be written for 3D modeling constraints. The gate type taxonomy applies without forcing.

**What we can't claim:** The gates improve outcomes. The thresholds are calibrated. The LLM intent gate works. The framework is portable in any meaningful operational sense.

**What would kill this:** Production agents that ignore the harness constraints despite enforcement. Intent gate precision <65%. Accretion thresholds that block >50% of legitimate designs. Or, most simply: nobody ever runs agents through this harness, and it remains theological indefinitely.

---

## Appendix: Test Evidence

```
=== Layer 1: Assert Gate Tests ===
Test 1: Valid bracket renders              PASS
Test 2: Negative width triggers assert     PASS
Test 3: Zero height triggers assert        PASS
Test 4: Thin wall triggers assert          PASS
Test 5: Low fn triggers assert             PASS
Test 6: Wall > width/2 triggers assert     PASS
Test 7: Fillet > wall triggers assert      PASS
Test 8: Excessive fn triggers assert       PASS
Test 9: Thin-but-valid wall emits warning  FAIL (gate caught fillet_radius > wall_thickness)

=== Layer 2: Geometry Validation Tests ===
Test 10: Valid bracket passes Layer 2      PASS
Test 11: Assert failure propagates         PASS
Test 12: Low facet threshold triggers      PASS
Test 13: Degenerate geometry caught        PASS
Test 14: Bounding box dimension check      PASS

Results: 13/14 passed, 1 failed (inter-parameter constraint interaction)
```

## Appendix: Falsification Status

| Criterion | Test | Threshold | Status |
|-----------|------|-----------|--------|
| Gates are ceremony | Gate fire rate in production | <5% for 30d | **NO DATA** |
| Gates are irrelevant | Block rate | Zero blocks in 30d | **NO DATA** |
| Soft harness is inert | Intent gate precision | <65% | **NO DATA** |
| Framework is anecdotal | Quality improvement vs baseline | No measurable benefit | **NO DATA** |
| Accretion thresholds wrong | Legitimate design block rate | >50% blocked | **NO DATA** |

## Appendix: Data Sources

| Source | Description |
|--------|-------------|
| `.harness/openscad/test/test-gates.sh` | 14 integration tests, run March 16 |
| `.harness/openscad/parts/bracket.scad` | 1 sample part (62 lines) |
| `.harness/openscad/lib/validate.scad` | 7 validation modules (63 lines) |
| `.harness/openscad/gates/geometry-check.sh` | Layer 2 gate (72 lines) |
| `.harness/openscad/gates/intent-check.sh` | Layer 4 gate (148 lines) |
| `.kb/investigations/2026-03-16-inv-design-minimal-harness-openscad-agent.md` | Architect investigation |
| `.kb/investigations/2026-03-16-eval-openscad-headless-agent-workflow.md` | OpenSCAD eval |
| `.kb/publications/self-measurement-report.md` | orch-go self-measurement report (methodology source) |
| `.kb/models/harness-engineering/model.md` | Harness engineering framework |
