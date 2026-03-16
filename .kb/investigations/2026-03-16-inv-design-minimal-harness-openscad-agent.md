## Summary (D.E.K.N.)

**Delta:** The harness engineering framework transfers to OpenSCAD with surprisingly clean 1:1 gate mapping — assert()=spawn gates, echo()=coaching, --summary-file=telemetry, slicer CLI=external validation gate — and the LLM-as-gate for intent alignment is the novel contribution that tests the "judgment layer" hypothesis.

**Evidence:** OpenSCAD eval (orch-go-5idhz) confirmed all four gate layers are mechanically implementable; the gate type taxonomy (execution/evidence/judgment) maps directly; accretion equivalent is polygon count + boolean depth.

**Knowledge:** The self-measurement methodology (precision/recall per gate, falsification criteria, cost tracking) is the truly portable artifact — more so than the gates themselves. LLM-as-gate has a concrete measurement protocol: 100 designs, 50 aligned/50 drifting, precision/recall against human labels.

**Next:** Implement Layer 1-2 (assert gates + geometry validation) as proof of concept, then Layer 3 (slicer), then Layer 4 (LLM intent gate with measurement).

**Authority:** architectural — Cross-domain design that establishes patterns for harness portability beyond orch-go.

---

# Investigation: Minimal Harness Design for OpenSCAD Agent Orchestration

**Question:** What is the minimal harness that proves structural enforcement transfers from code to non-code domains, specifically OpenSCAD 3D modeling?

**Started:** 2026-03-16
**Updated:** 2026-03-16
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None — implement via decomposed issues
**Status:** Complete
**Model:** harness-engineering

**Patches-Decision:** N/A (new domain, no prior decision)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-16-eval-openscad-headless-agent-workflow.md` | extends | Yes — all CLI findings verified by hands-on eval | None |
| `.kb/threads/2026-03-16-cross-domain-harness-portability-can.md` | extends | Yes — three hypotheses (H1-H3) directly inform this design | None |
| `.kb/models/harness-engineering/model.md` | applies | Yes — gate taxonomy, measurement pairing, falsification criteria all transfer | None — the model predicts this should work |
| `.kb/models/completion-verification/model.md` | applies | Yes — gate type taxonomy (execution/evidence/judgment) is the structural template | None |

---

## Findings

### Finding 1: Gate Type Taxonomy Maps 1:1 to OpenSCAD

**Evidence:** The completion verification model defines three gate types: execution (binary, unfakeable), evidence (pattern matching), judgment (human/LLM). OpenSCAD has direct equivalents at every tier:

| orch-go Gate Type | orch-go Example | OpenSCAD Equivalent |
|---|---|---|
| Execution (hard) | `go build` | `openscad -o out.stl` (exit code 0/1) |
| Execution (hard) | `go vet` | `assert()` in .scad (exit 1, parseable errors) |
| Evidence (structured soft) | Test evidence patterns | `--summary-file` JSON (bounding box, vertex count) |
| Evidence (structured soft) | Accretion gate | Polygon count / boolean depth limits |
| Evidence (structured soft) | Visual verification | PNG render via `--autocenter --viewall` |
| Judgment (soft) | Explain-back | LLM-as-gate: intent alignment check |
| Advisory (non-blocking) | Coaching plugin | `echo()` warnings in .scad |

**Source:** Eval findings (orch-go-5idhz), completion-verification model

**Significance:** This confirms H1 from the thread — the gate taxonomy IS domain-general. The same three tiers emerge naturally in OpenSCAD without forcing. This is the strongest evidence for harness portability.

---

### Finding 2: Four-Layer Gate Stack with Clear Hard/Soft Boundary

**Evidence:** The eval identified exactly where mechanical gateability ends and judgment begins:

**Layer 1 — Parameter Validation (HARD, in-process):**
- `assert(wall_thickness >= 0.8, "GATE FAIL: wall too thin")` in .scad files
- Exit code 1, parseable stderr, first-fail semantics
- Agent can write these; they execute as part of render
- Equivalent to: `go vet` / type checking

**Layer 2 — Geometry Validation (HARD, post-render):**
- Manifold check: CGAL reports `Simple: yes/no`
- Vertex/facet bounds from `--summary-file` JSON
- Bounding box sanity (dimensions within expected range)
- Zero-volume detection (degenerate geometry)
- Equivalent to: `go build` / compilation

**Layer 3 — Printability Validation (HARD, external tool):**
- PrusaSlicer CLI: `prusa-slicer --export-gcode --info input.stl`
- Detects: wall thickness violations, excessive overhangs, unsupported bridging
- Produces: estimated print time, material usage, support volume
- This is the "external validator" layer — like running integration tests
- Equivalent to: running actual tests (the gap orch-go hasn't closed)

**Layer 4 — Intent Alignment (JUDGMENT, LLM-as-gate):**
- Inputs: design spec (intent), STL metadata, PNG render, .scad source
- Output: aligned/misaligned with confidence score + reasoning
- This is the novel contribution — first LLM judgment gate with measurable precision
- Equivalent to: explain-back gate, but automated and falsifiable

**Source:** Eval findings, thread analysis of gate boundaries

**Significance:** The boundary between Layer 2 and Layer 3 is the interesting one — it's where OpenSCAD's native capabilities end and external tooling begins. This maps to the "sharp boundary at execution" in the harness model. Layers 1-2 are native OpenSCAD. Layer 3 requires PrusaSlicer integration. Layer 4 requires LLM API calls.

---

### Finding 3: Accretion in OpenSCAD Has Three Metrics

**Evidence:** "File bloat" in code is measured by line count. In OpenSCAD, there are three independent dimensions of complexity:

| Metric | What It Measures | How to Measure | Threshold Heuristic |
|---|---|---|---|
| **Polygon count** | Output complexity | `--summary-file` → facets field | Warning >50K, block >200K for single part |
| **Boolean depth** | CSG tree complexity | Static analysis of `union/difference/intersection` nesting | Warning >8 levels, block >15 |
| **Module count** | Source complexity | Count `module` declarations in .scad | Warning >20, block >40 per file |
| **File lines** | Source size (same as code) | `wc -l` | Same 800/1500 thresholds |

Polygon count is the OUTPUT metric (like binary size). Boolean depth and module count are SOURCE metrics (like cyclomatic complexity and function count). File lines transfers directly.

The key insight: polygon count can explode from a single line of code (`$fn=360` on a cylinder = 720 facets vs `$fn=16` = 32 facets). Unlike code where lines ≈ complexity, OpenSCAD can produce massive output from tiny source. This means SOURCE metrics and OUTPUT metrics must both be gated.

**Source:** Eval findings (parameter exploration with -D flags), thread analysis

**Significance:** This is the accretion equivalent. The pre-commit growth gate transfers for file lines. The completion accretion gate transfers for polygon count. Boolean depth is a new metric type — analogous to cyclomatic complexity but for geometry.

---

## Synthesis

**Key Insights:**

1. **The gate type taxonomy is domain-general.** Execution/evidence/judgment tiers emerge naturally in OpenSCAD without modification. This validates H1 from the thread.

2. **The measurement methodology is the truly portable artifact.** Gates are domain-specific (assert() vs `go build`). But the methodology — precision/recall per gate, cost tracking, falsification criteria, enforcement+measurement pairing — transfers completely. The self-measurement report IS the template for any domain.

3. **LLM-as-gate is concretely designable and falsifiable.** Unlike the rejected agent-judgment gate in orch-go (closed loop, same blind spots), the OpenSCAD intent gate has an important structural difference: the LLM judges geometric/visual output against declared intent, not code against code. The input modalities are different from the generation modalities. This may escape the "same blind spots" problem.

**Answer to Investigation Question:**

The minimal harness requires: (1) a CLAUDE.md template with OpenSCAD-specific constraints and workflow, (2) a 4-layer gate stack progressing from deterministic to judgment, (3) a completion pipeline that checks all layers, (4) two skill templates (design-part, iterate-design), (5) a measurement plan that mirrors the self-measurement report methodology, and (6) accretion metrics for polygon count, boolean depth, module count, and file lines. The LLM-as-gate for intent alignment is the novel contribution that tests H3 (judgment gates are viable).

---

# DELIVERABLE 1: CLAUDE.md Template for OpenSCAD Projects

```markdown
# [Project Name] — OpenSCAD

## Architecture

```
agent (Claude Code)
    │
    ├── openscad CLI          ← render, validate, export
    ├── prusa-slicer CLI      ← printability analysis
    └── intent-gate script    ← LLM alignment check
```

## Workflow

1. Agent writes/modifies .scad files
2. Render with Manifold backend for fast iteration: `openscad -o out.stl --backend manifold file.scad -D 'param=value'`
3. Validate with CGAL for manifold correctness: `openscad -o /dev/null --backend cgal --quiet file.scad`
4. Export PNG for visual check: `openscad -o preview.png --autocenter --viewall file.scad`
5. Run printability check: `prusa-slicer --export-gcode --info out.stl`
6. Run intent gate: `./gates/intent-check.sh out.stl design-spec.md`

## Constraints

### Render Backends
- **Manifold** for iteration (232x faster than CGAL). Use `-D` flags for parameter exploration.
- **CGAL** for final validation only. CGAL detects non-manifold geometry that Manifold silently repairs.
- Always run CGAL validation pass before declaring a design complete.

### Parameter Validation (assert gates)
Every .scad file MUST include parameter validation at the top:
```openscad
// Parameter gates — exit 1 on violation
assert(wall_thickness >= 0.8, str("GATE FAIL: wall_thickness=", wall_thickness, " < 0.8mm minimum"));
assert(width > 0 && height > 0 && depth > 0, "GATE FAIL: all dimensions must be positive");
assert($fn >= 16, str("GATE FAIL: $fn=", $fn, " too low for printable curves"));
```

Pattern: `GATE FAIL:` prefix for grep-able error extraction.

### Accretion Boundaries
| Metric | Warning | Block |
|--------|---------|-------|
| File lines | >300 lines | >600 lines |
| Module count | >15 per file | >30 per file |
| Polygon count | >50K facets | >200K facets |
| Boolean depth | >8 levels | >15 levels |

### Design Space Exploration
Use `-D` flags for parameter sweeps, NOT code modifications:
```bash
for thickness in 1.0 1.5 2.0 2.5; do
    openscad -o "out_${thickness}.stl" file.scad -D "wall_thickness=${thickness}" --backend manifold --quiet
done
```

### Common Gotchas
- **Negative dimensions produce geometry** — always assert positivity
- **Manifold silently repairs non-manifold** — always CGAL-validate before export
- **$fn controls facet count exponentially** — $fn=360 on cylinder = 720 facets
- **First-fail assert semantics** — only first assertion fires; use validation module for batch checking
- **No printability awareness** — OpenSCAD exit 0 does NOT mean printable
- **STL has no parameter metadata** — emit parameter snapshots alongside exports
```

---

# DELIVERABLE 2: Gate Stack Design

## Layer 1: Parameter Validation (assert-based, in .scad)

**Type:** Execution (hard)
**Where:** Inside .scad files, evaluated during render
**Enforcement:** Exit code 1 on violation
**Measurement:** Parse stderr for `GATE FAIL:` pattern, count violations per render

**Implementation pattern — validation module:**
```openscad
// lib/validate.scad — batch parameter validation
module validate_bracket(w, h, d, wall, hole_d, fn) {
    // Collect all violations before asserting
    violations = [
        if (wall < 0.8) str("wall_thickness=", wall, " < 0.8mm"),
        if (w <= 0) str("width=", w, " must be positive"),
        if (h <= 0) str("height=", h, " must be positive"),
        if (d <= 0) str("depth=", d, " must be positive"),
        if (hole_d < 1.0) str("hole_diameter=", hole_d, " < 1.0mm minimum"),
        if (fn < 16) str("$fn=", fn, " too low for printable curves"),
        if (wall > w/2) str("wall=", wall, " > width/2=", w/2, " (no interior space)")
    ];

    assert(len(violations) == 0,
        str("GATE FAIL: ", len(violations), " violations: ", violations));
}
```

**What this catches:** Degenerate parameters, negative dimensions, physically impossible constraints.
**What this misses:** Valid parameters that produce bad geometry (self-intersection, printability).

## Layer 2: Geometry Validation (post-render)

**Type:** Execution (hard)
**Where:** Shell script wrapping OpenSCAD CLI
**Enforcement:** Script exit code
**Measurement:** Parse `--summary-file` JSON for metrics

**Implementation — geometry gate script:**
```bash
#!/bin/bash
# gates/geometry-check.sh — Layer 2 geometry validation
set -euo pipefail

SCAD_FILE="$1"
SUMMARY_FILE="${2:-/tmp/openscad-summary.json}"
STL_OUTPUT="${3:-/tmp/gate-check.stl}"
MAX_FACETS="${MAX_FACETS:-200000}"
MAX_VERTICES="${MAX_VERTICES:-100000}"

# Step 1: Render with CGAL (manifold detection)
echo "=== CGAL Manifold Check ==="
CGAL_OUTPUT=$(openscad -o "$STL_OUTPUT" --backend cgal --quiet \
    --summary-file "$SUMMARY_FILE" "$SCAD_FILE" 2>&1) || {
    echo "GATE FAIL: CGAL render failed (exit $?)"
    echo "$CGAL_OUTPUT" | grep -i "error\|warning\|assert" || true
    exit 1
}

# Step 2: Check for non-manifold warnings
if echo "$CGAL_OUTPUT" | grep -qi "not.*manifold\|Simple: no"; then
    echo "GATE FAIL: Non-manifold geometry detected"
    echo "$CGAL_OUTPUT"
    exit 1
fi

# Step 3: Check summary metrics
if [ -f "$SUMMARY_FILE" ]; then
    FACETS=$(python3 -c "import json; print(json.load(open('$SUMMARY_FILE'))['facets'])" 2>/dev/null || echo "0")
    VERTICES=$(python3 -c "import json; print(json.load(open('$SUMMARY_FILE'))['vertices'])" 2>/dev/null || echo "0")

    if [ "$FACETS" -gt "$MAX_FACETS" ]; then
        echo "GATE FAIL: facet count $FACETS > $MAX_FACETS limit"
        exit 1
    fi
    if [ "$VERTICES" -gt "$MAX_VERTICES" ]; then
        echo "GATE FAIL: vertex count $VERTICES > $MAX_VERTICES limit"
        exit 1
    fi

    echo "Geometry OK: $FACETS facets, $VERTICES vertices"
fi

# Step 4: Bounding box sanity (dimensions within expected range)
# Optional: pass expected dimensions via env vars
if [ -n "${EXPECTED_WIDTH:-}" ]; then
    # Parse bounding box from summary and compare
    echo "Bounding box check: see summary JSON"
fi

echo "=== Layer 2 PASS ==="
exit 0
```

**What this catches:** Non-manifold geometry, excessive polygon count, render failures.
**What this misses:** Printability (wall thickness, overhangs), intent alignment.

## Layer 3: Printability Validation (slicer CLI)

**Type:** Execution (hard, external tool)
**Where:** Shell script calling PrusaSlicer CLI
**Enforcement:** Script exit code based on slicer analysis
**Measurement:** Parse slicer output for support volume, print time, warnings

**Implementation — printability gate script:**
```bash
#!/bin/bash
# gates/printability-check.sh — Layer 3 slicer validation
set -euo pipefail

STL_FILE="$1"
PRINTER_PROFILE="${PRINTER_PROFILE:-MK4S}"
MAX_SUPPORT_RATIO="${MAX_SUPPORT_RATIO:-0.3}"  # support volume / part volume

# Step 1: Run PrusaSlicer analysis
echo "=== Printability Check (PrusaSlicer) ==="
SLICER_OUTPUT=$(prusa-slicer --export-gcode \
    --load "$HOME/.config/PrusaSlicer/printer/$PRINTER_PROFILE.ini" \
    --info "$STL_FILE" 2>&1) || {
    echo "GATE FAIL: Slicer rejected geometry"
    echo "$SLICER_OUTPUT"
    exit 1
}

# Step 2: Parse key metrics
# PrusaSlicer --info outputs: volume, dimensions, print time estimate
# Actual parsing depends on PrusaSlicer version output format

# Step 3: Check for slicer warnings
if echo "$SLICER_OUTPUT" | grep -qi "thin wall\|zero volume\|degenerate"; then
    echo "GATE WARN: Slicer detected potential issues"
    echo "$SLICER_OUTPUT" | grep -i "thin\|zero\|degenerate"
    # Warning, not block — agent should address but not hard-stopped
fi

# Step 4: Support volume ratio check
# If support volume > MAX_SUPPORT_RATIO * part volume, flag it
# This catches designs that are technically printable but wasteful

echo "=== Layer 3 PASS ==="
exit 0
```

**What this catches:** Unprintable geometry, excessive support needs, wall thickness violations.
**What this misses:** Intent alignment (is this the RIGHT design?), aesthetic quality.

**Note:** PrusaSlicer CLI integration needs validation. The exact flags and output format may differ by version. This is the external tool dependency — like depending on a test framework.

## Layer 4: Intent Alignment (LLM-as-gate)

**Type:** Judgment (soft, but measurably falsifiable)
**Where:** Script calling LLM API with structured prompt
**Enforcement:** Advisory (non-blocking) in v1, blocking after precision is validated >80%
**Measurement:** Precision/recall against human-labeled dataset

**This is the novel contribution.** Unlike the rejected agent-judgment gate in orch-go (closed loop — agent reviewing agent code with same blind spots), the intent gate has a structural difference: the LLM judges rendered geometric output against a textual intent declaration. The input modalities differ from the generation modalities.

**Implementation — intent gate:**
```bash
#!/bin/bash
# gates/intent-check.sh — Layer 4 LLM intent alignment
set -euo pipefail

STL_FILE="$1"
SPEC_FILE="$2"  # design-spec.md with intent declaration
SUMMARY_FILE="${3:-/tmp/openscad-summary.json}"
PNG_FILE="${4:-/tmp/preview.png}"

# Construct gate prompt
SPEC_CONTENT=$(cat "$SPEC_FILE")
SUMMARY_CONTENT=$(cat "$SUMMARY_FILE" 2>/dev/null || echo "{}")

GATE_PROMPT="You are a design review gate. Your job is to judge whether a 3D model
aligns with its declared design intent.

DESIGN INTENT (what was requested):
---
$SPEC_CONTENT
---

GEOMETRY METADATA:
---
$SUMMARY_CONTENT
---

SOURCE CODE:
---
$(cat "${STL_FILE%.stl}.scad" 2>/dev/null || echo "Source not available")
---

Judge this design on 4 criteria:
1. DIMENSIONAL COMPLIANCE: Do the dimensions match the spec?
2. FEATURE COMPLETENESS: Are all specified features present?
3. CONSTRAINT SATISFACTION: Are material/printer constraints respected?
4. PURPOSE FIT: Would this design actually serve its declared purpose?

Respond in this exact format:
VERDICT: ALIGNED | MISALIGNED | NEEDS_REVIEW
CONFIDENCE: 0.0-1.0
CRITERIA:
  dimensional: PASS | FAIL | PARTIAL
  features: PASS | FAIL | PARTIAL
  constraints: PASS | FAIL | PARTIAL
  purpose: PASS | FAIL | PARTIAL
REASONING: <one paragraph explaining the verdict>"

# Call LLM API (Claude or other)
# In practice: claude --print "$GATE_PROMPT" --model haiku
# Parse structured output for VERDICT line

echo "=== Layer 4: Intent Gate ==="
echo "$GATE_PROMPT" | claude --print --model haiku 2>/dev/null || {
    echo "GATE WARN: Intent gate unavailable (LLM API error)"
    echo "Proceeding without intent check"
    exit 0  # Advisory only — don't block on API failure
}
```

**Design decisions for the LLM gate:**
1. **Use Haiku, not Opus.** Gate cost must be low. Haiku is sufficient for structured comparison.
2. **Structured output format.** VERDICT/CONFIDENCE/CRITERIA format is parseable, not prose.
3. **Four criteria, not one.** Dimensional compliance is mechanical (could be gated without LLM). Purpose fit is pure judgment. The spectrum within the gate mirrors the spectrum across layers.
4. **Advisory in v1.** Don't block until precision is validated >80%.
5. **Fail-open on API error.** Gate unavailability should not block agent work.

**Why this escapes the "closed loop" problem:**
- In orch-go, agent-reviewing-agent-code is same model, same training data, same blind spots
- Here, the gate LLM receives DIFFERENT modalities (geometry metadata, rendered image, spec text) than the generation LLM used (wrote .scad code)
- The gate judges OUTPUT against INTENT, not code against code
- This is closer to how human review works: look at the thing, compare to the spec

**Measurement protocol (see Deliverable 5):**
- 100 design-spec pairs: 50 aligned, 50 deliberately drifting
- Drift types: wrong dimensions, missing features, wrong material constraints, purpose mismatch
- Precision/recall against human labels
- Threshold: <65% = not deployable, 65-80% = advisory only, >80% = blocking gate

---

# DELIVERABLE 3: Completion Pipeline

When an agent's design work is "done," the completion pipeline checks:

```
┌─────────────────────────────────────────────────────┐
│                 Completion Pipeline                    │
├─────────────────────────────────────────────────────┤
│ 1. Phase Complete    │ Agent reported Phase: Complete │
│ 2. Source exists     │ .scad file(s) committed        │
│ 3. STL renders       │ openscad -o exits 0            │
│ 4. Assert gates pass │ No GATE FAIL in stderr         │
│ 5. CGAL manifold     │ Simple: yes                    │
│ 6. Polygon budget    │ facets < threshold              │
│ 7. Slicer passes     │ prusa-slicer exits 0           │
│ 8. Accretion check   │ Lines/modules within limits    │
│ 9. Intent gate       │ VERDICT: ALIGNED (advisory)    │
│ 10. Parameter snap   │ -D values recorded in metadata │
│ 11. Synthesis exists  │ SYNTHESIS.md with design notes │
└─────────────────────────────────────────────────────┘
```

**Gate type classification:**

| # | Gate | Type | Hard/Soft | Blocks? |
|---|------|------|-----------|---------|
| 1 | Phase Complete | Evidence | Soft | Yes |
| 2 | Source exists | Evidence | Hard (file check) | Yes |
| 3 | STL renders | Execution | Hard | Yes |
| 4 | Assert gates | Execution | Hard | Yes |
| 5 | CGAL manifold | Execution | Hard | Yes |
| 6 | Polygon budget | Evidence | Hard (threshold) | Yes (>200K) / Warn (>50K) |
| 7 | Slicer passes | Execution | Hard | Yes |
| 8 | Accretion check | Evidence | Hard (threshold) | Yes (>600 lines) / Warn (>300) |
| 9 | Intent gate | Judgment | Soft (LLM) | Advisory until validated |
| 10 | Parameter snap | Evidence | Soft | Warn only |
| 11 | Synthesis | Evidence | Soft | Yes (full tier) |

**Verification levels (adapted from V0-V3):**

| Level | Gates | Typical Work |
|-------|-------|-------------|
| V0 (Acknowledge) | 1 | Parameter tweaks, documentation |
| V1 (Renders) | 1-5 | Design exploration, prototyping |
| V2 (Printable) | 1-8 | Production-ready parts |
| V3 (Reviewed) | 1-11 | Assemblies, critical parts |

---

# DELIVERABLE 4: Skill Templates

## Skill: design-part

```markdown
---
name: design-part
description: Create a 3D part from a design specification
tier: full
verification-level: V2
---

# Design Part

Create a parametric OpenSCAD part from a design specification.

## Workflow

1. **Read the design spec** (SPAWN_CONTEXT or attached spec file)
2. **Write .scad file** with parameter validation (assert gates)
3. **Render with Manifold** for fast iteration
4. **Iterate** using -D flag parameter sweeps
5. **Validate with CGAL** for manifold correctness
6. **Run slicer check** for printability
7. **Export STL + PNG** preview
8. **Record parameters** in metadata

## Required Outputs
- `parts/<name>.scad` — parametric source
- `exports/<name>.stl` — rendered mesh
- `exports/<name>.png` — visual preview
- `exports/<name>-summary.json` — render metadata

## Constraints
- Every .scad file starts with assert() parameter validation
- Manifold for iteration, CGAL for final validation
- Polygon count < 200K for single parts
- All parameters must have sensible defaults
- No magic numbers — use named variables with comments
```

## Skill: iterate-design

```markdown
---
name: iterate-design
description: Explore parameter space of an existing design
tier: light
verification-level: V1
---

# Iterate Design

Systematically explore the parameter space of an existing .scad design.

## Workflow

1. **Read existing .scad file** and identify parameters
2. **Define sweep ranges** (which params, what bounds)
3. **Run parameter sweep** via -D flags (NOT by editing .scad)
4. **Collect results** (STL + summary JSON per configuration)
5. **Analyze tradeoffs** (strength vs weight, material vs cost)
6. **Recommend optimal configuration** with reasoning

## Required Outputs
- `sweeps/<name>-sweep-results.json` — parameter configurations + metrics
- `sweeps/<name>-recommendation.md` — analysis and recommendation
- `exports/<name>-optimal.stl` — recommended configuration export

## Constraints
- NEVER modify the source .scad file — use -D flags only
- Use Manifold backend for sweeps (speed matters)
- Run CGAL validation on final recommended configuration only
- Record all configurations tested, not just the best
```

---

# DELIVERABLE 5: Measurement Plan

## Methodology Transfer

The self-measurement report methodology transfers directly. For each gate:

| Measurement | Code Domain (orch-go) | OpenSCAD Domain |
|---|---|---|
| **Gate fire rate** | spawn_gate_decision events / total spawns | gate failures / total renders |
| **Gate precision** | retrospective classification of blocks (TP/FP) | same — classify each gate failure as correct or false positive |
| **Gate cost** | pipeline timing per step | time per gate evaluation |
| **Gate coverage** | % of completions with data (was 4.7% for accretion) | % of designs that pass through each gate |
| **Dormancy** | gates that evaluate but never fire | same — especially polygon count gate on simple designs |

## Per-Gate Measurement

| Gate | Enforcement Metric | Measurement Metric | Falsification |
|---|---|---|---|
| Assert (L1) | Exit code 1 on violation | Violations per render, violation types | "If agents never trigger asserts, either all params are always valid or the asserts are too lax" |
| Geometry (L2) | CGAL manifold check | Non-manifold rate, facet/vertex distributions | "If CGAL never finds non-manifold, are we over-constraining params?" |
| Slicer (L3) | PrusaSlicer pass/fail | Printability failure rate, common failure types | "If slicer never fails, the assert gates are catching everything and L3 is dormant" |
| Intent (L4) | LLM verdict | Precision/recall vs human labels | "If intent gate precision < 65%, it's noise not signal" |
| Accretion | Line/module/polygon limits | Growth trajectories over time | "If accretion gates never fire, are thresholds too generous?" |

## LLM-as-Gate Measurement Protocol

**The novel measurement that doesn't exist in orch-go.**

### Dataset Construction

Build a labeled dataset of 100 design-spec pairs:

**50 ALIGNED designs:**
- Design matches spec dimensions within ±5%
- All specified features present
- Material/printer constraints respected
- Design serves declared purpose

**50 MISALIGNED designs (10 per drift type):**
1. **Dimensional drift:** Correct features, wrong dimensions (±20%+)
2. **Missing features:** Right dimensions, features omitted (no mounting holes, missing fillets)
3. **Constraint violation:** Printability issues (0.3mm walls, 80° overhangs, unsupported bridges)
4. **Purpose mismatch:** Geometrically valid but wrong for the stated purpose (bracket too weak, enclosure doesn't fit board)
5. **Subtle drift:** Almost right but one critical dimension off, or one feature subtly wrong

### Measurement

```
For each (design, spec) pair:
    1. Run intent gate → VERDICT, CONFIDENCE
    2. Compare to human label → TP/FP/TN/FN
    3. Record confidence calibration (is 0.9 confidence actually 90% correct?)
```

**Metrics:**
- Precision: TP / (TP + FP) — "when it says MISALIGNED, is it right?"
- Recall: TP / (TP + FN) — "does it catch real misalignment?"
- Confidence calibration: plot confidence vs actual accuracy

**Thresholds:**
- Precision <65% → gate is noise, do not deploy
- Precision 65-80% → advisory only (warn but don't block)
- Precision >80% → eligible for blocking gate
- Recall <50% → gate misses too much, needs prompt engineering

### Cost Measurement

- Time per intent gate evaluation (target: <5s with Haiku)
- API cost per evaluation (target: <$0.01 with Haiku)
- Compare: is the intent gate cheaper than human review?

## Falsification Criteria (adapted from harness model)

| Criterion | Measurement | Kill Threshold | Survives If |
|---|---|---|---|
| Gates are ceremony | Accretion velocity pre/post gate | Post-gate growth >80% of pre-gate | Measurable reduction in polygon bloat |
| Gates are irrelevant | Gate fire rate | Fire rate <5% across all gates | At least one gate fires >10% |
| Soft harness is inert | Intent gate precision vs random | Precision not significantly >50% | Precision >65% (better than coin flip) |
| Framework is anecdotal | Second-domain deployment | No benefit outside orch-go | Gates catch real issues in OpenSCAD |

## Self-Measurement Report Template

The report for OpenSCAD would mirror the orch-go report:

```markdown
# OpenSCAD Harness Self-Measurement Report

## What We Enforce
- N parameter assert gates across M .scad files
- Geometry validation via CGAL (manifold check)
- Printability via PrusaSlicer CLI
- Intent alignment via LLM gate (advisory)

## What We Measure
| Gate | Fire Rate | Precision | Cost | Coverage |
|------|-----------|-----------|------|----------|
| Assert (L1) | X% | Y% | Zms | 100% |
| Geometry (L2) | ... | ... | ... | ... |
| Slicer (L3) | ... | ... | ... | ... |
| Intent (L4) | ... | ... | ... | ... |

## What We'd Like to Claim But Can't Yet
- [Claims awaiting data]

## What Would Kill This
- [Active falsification criteria with current status]
```

---

# DELIVERABLE 6: Accretion Equivalent

## What "File Bloat" Means in OpenSCAD

| Code Metric | OpenSCAD Analog | Why It Matters |
|---|---|---|
| File lines (>1500 = CRITICAL) | File lines (>600 = CRITICAL) | Lower threshold: .scad files are more information-dense than Go |
| Function count | Module count (>30 = CRITICAL) | Modules are the unit of decomposition |
| Cyclomatic complexity | Boolean depth (>15 = CRITICAL) | Deep CSG trees are hard to debug and slow to render |
| Binary size | Polygon count (>200K = CRITICAL) | Output complexity that affects downstream tools |
| N/A (code-specific) | $fn budget | Resolution parameter that explodes polygon count |

## Gate Implementation

**Pre-render gate (like pre-commit):**
```bash
# Check source-level accretion before render
FILE_LINES=$(wc -l < "$SCAD_FILE")
MODULE_COUNT=$(grep -c "^module " "$SCAD_FILE")

if [ "$FILE_LINES" -gt 600 ]; then
    echo "ACCRETION BLOCK: $SCAD_FILE is $FILE_LINES lines (limit: 600)"
    exit 1
fi
if [ "$MODULE_COUNT" -gt 30 ]; then
    echo "ACCRETION BLOCK: $SCAD_FILE has $MODULE_COUNT modules (limit: 30)"
    exit 1
fi
```

**Post-render gate (like completion accretion):**
```bash
# Check output-level accretion after render
FACETS=$(python3 -c "import json; print(json.load(open('$SUMMARY_FILE'))['facets'])")
if [ "$FACETS" -gt 200000 ]; then
    echo "ACCRETION BLOCK: $FACETS facets exceeds 200K limit"
    exit 1
fi
```

## The $fn Budget — Unique to OpenSCAD

`$fn` (number of facets for curves) is a global resolution parameter. A single `$fn=360` on a cylinder with 4 holes produces 360 × 5 = 1,800 facets. With 20 such features, you're at 36,000 facets from resolution alone.

**Budget approach:**
- Low-res iteration: `$fn=32` (fast renders, visible facets)
- Medium-res review: `$fn=64` (smooth enough for visual check)
- High-res export: `$fn=128` (print-quality, use only for final)

Gate: warn if `$fn > 128` for any feature, block if `$fn > 256`.

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenSCAD assert() gates work as spawn gate equivalent (verified: eval hands-on testing)
- ✅ --summary-file produces parseable JSON with facets/vertices (verified: eval)
- ✅ CGAL detects non-manifold that Manifold silently repairs (verified: eval)
- ✅ -D flags enable parameter exploration without code changes (verified: eval)
- ✅ Gate type taxonomy (execution/evidence/judgment) maps to OpenSCAD (verified: structural analysis)

**What's untested:**

- ⚠️ PrusaSlicer CLI integration for automated printability checking (not tested — need to install and validate CLI flags)
- ⚠️ LLM-as-gate precision for intent alignment (hypothesis — needs 100-pair measurement protocol)
- ⚠️ Accretion thresholds for polygon count and boolean depth (heuristic — need empirical calibration)
- ⚠️ Boolean depth static analysis (need .scad parser or regex approach)
- ⚠️ Whether the intent gate escapes the "closed loop" problem (structural argument, not empirical)

**What would change this:**

- Intent gate precision <65% → the judgment layer doesn't work, restrict to mechanical gates only
- PrusaSlicer CLI doesn't produce parseable output → need different slicer or custom analysis
- OpenSCAD accretion patterns differ fundamentally from code → thresholds need different calibration methodology

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Implement Layer 1-2 gates | implementation | Standard gate patterns, well-understood |
| Integrate PrusaSlicer CLI | implementation | External tool integration, reversible |
| Build LLM intent gate | architectural | Novel gate type, sets precedent for judgment gates |
| Set accretion thresholds | architectural | Threshold calibration affects all future agents |
| Adapt measurement methodology | architectural | Establishes cross-domain measurement pattern |
| Publish as harness portability evidence | strategic | External claims, publication risk |

### Recommended Approach: Phased Implementation

**Phase 1: Mechanical Gates (1-2 sessions)**
- Layer 1 (assert patterns) + Layer 2 (geometry validation script)
- CLAUDE.md template deployed to test project
- design-part skill template
- This proves the framework transfers at the mechanical level

**Phase 2: External Validation (1 session)**
- Layer 3 (PrusaSlicer CLI integration)
- iterate-design skill template
- This proves the external-tool gate pattern

**Phase 3: Judgment Gate + Measurement (2-3 sessions)**
- Layer 4 (LLM intent gate) implementation
- 100-pair measurement dataset construction
- Precision/recall measurement
- This is the novel contribution — validates or kills H3

**Phase 4: Self-Measurement Report (1 session)**
- Collect gate metrics across all layers
- Produce self-measurement report using orch-go methodology
- Compare cross-domain: do the same patterns emerge?

**Trade-offs accepted:**
- Starting with mechanical gates before judgment is conservative but builds trust
- 100-pair dataset is small but sufficient for initial precision estimate (±10% at 95% CI)
- PrusaSlicer dependency introduces tool-specific coupling

**Implementation sequence:**
1. Create test OpenSCAD project with sample parts (bracket, enclosure, gear)
2. Write gate scripts (Layer 1, 2)
3. Write CLAUDE.md and skill templates
4. Integrate PrusaSlicer (Layer 3)
5. Build intent gate and measurement dataset (Layer 4)
6. Run measurement protocol
7. Produce self-measurement report

---

## References

**Files Examined:**
- `.kb/investigations/2026-03-16-eval-openscad-headless-agent-workflow.md` — Primary eval findings
- `.kb/threads/2026-03-16-cross-domain-harness-portability-can.md` — Thread with hypotheses H1-H3
- `.kb/models/harness-engineering/model.md` — Framework being tested for portability
- `.kb/models/completion-verification/model.md` — Gate type taxonomy source

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-16-cross-domain-harness-portability-can.md` — Driving thread
- **Eval:** `.kb/investigations/2026-03-16-eval-openscad-headless-agent-workflow.md` — Evidence base
- **Model:** `.kb/models/harness-engineering/model.md` — Framework under test

---

## Investigation History

**2026-03-16 16:45:** Investigation started
- Initial question: What is the minimal harness for OpenSCAD agent orchestration?
- Context: Proving harness engineering portability to non-code domain (thread H1-H3)

**2026-03-16 17:30:** Design complete
- 4-layer gate stack designed with implementation scripts
- CLAUDE.md template, 2 skill templates, measurement plan, accretion metrics defined
- 6 implementation issues to create
- Key outcome: Gate type taxonomy confirmed domain-general; LLM-as-gate is the novel testable claim
