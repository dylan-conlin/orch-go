---
title: "OpenSCAD Headless Agent Workflow — Experiential Evaluation"
date: 2026-03-16
type: experiential-eval
status: complete
beads_id: orch-go-5idhz
---

# OpenSCAD Headless Agent Workflow — Experiential Evaluation

## What I Did

Built a parametric L-bracket in OpenSCAD (.scad), rendered it via CLI to STL and PNG, overrode parameters via `-D` flags, then systematically broke it: degenerate dimensions, non-manifold geometry, self-intersecting polyhedra, zero-thickness features. Tested `assert()` gates, `echo()` warnings, `--hardwarnings`, `--quiet`, `--summary-file` JSON output, and compared CGAL vs Manifold render backends. Built a complete gate pipeline script simulating automated agent validation.

**Environment:** OpenSCAD 2025.02.28 (built from source), macOS, CLI-only (no GUI used).

## What Worked Well

1. **`-D` parameter overrides are perfect for agent workflows.** `openscad -o out.stl file.scad -D 'width=80' -D 'holes=3'` — clean, composable, no temp file generation needed. An agent can write a .scad file once and explore the design space purely through CLI flags.

2. **`assert()` is a real gate mechanism.** Returns exit code 1, custom error message with interpolated values, file/line reference. Grep-able pattern: `GATE FAIL: .*`. First-fail semantics (only first assertion fires). This is structurally identical to how orch spawn gates work.

3. **`--quiet` mode is gate-pipeline-ready.** Suppresses all output EXCEPT errors. Combined with exit code checking, you get a clean pass/fail signal with zero noise on success.

4. **`--summary-file` produces machine-readable JSON.** Bounding box, vertex/facet counts, render time — all parseable. Not produced on assert failure (which is fine — the exit code is the signal).

5. **Manifold backend is transformative.** Complex bracket: CGAL 12.5s → Manifold 54ms (232x speedup). For agent iteration loops, this is the difference between "tolerable" and "instant feedback".

6. **PNG preview via `--autocenter --viewall`** gives visual verification without a GUI. An agent can render, read the image, and confirm the geometry looks right.

## What Didn't Work

1. **OpenSCAD has zero awareness of physical printability.** Wall thickness 0.1mm, negative dimensions, zero-radius holes — all produce valid STL with exit 0 and "Simple: yes". The geometry is garbage but OpenSCAD declares it sound. **An agent that trusts exit code alone will ship unprintable parts.**

2. **`--hardwarnings` doesn't catch manifold warnings.** Edge-sharing cubes produce an EXPORT-WARNING about non-manifold geometry, but `--hardwarnings` still exits 0. The flag only catches language-level warnings (deprecated syntax, etc.), not geometry warnings. This is a misleading name.

3. **Self-intersecting polyhedra pass silently.** No warning, no error, exit 0. The CGAL backend doesn't detect them; the Manifold backend silently "fixes" them. Neither tells you the input was malformed.

4. **No summary JSON on failure.** When assert() fires, `--summary-file` is not created. You can't get structured error output — only stderr text parsing.

5. **First-fail assert semantics.** Only the first failing assertion runs. If you want to report ALL parameter violations at once, you need to build a custom validation module that collects errors before asserting. This is a design limitation for batch validation.

## What Surprised Me

1. **Negative dimensions produce geometry.** `cube([-10, 20, 30])` doesn't error — it creates an inside-out shape that renders to STL. The geometry is reflected/inverted but OpenSCAD treats it as valid. This is mathematically consistent (negative extrusion) but practically dangerous.

2. **Manifold backend silently repairs non-manifold geometry.** Edge-sharing cubes that CGAL flags as non-manifold (Simple: no) become clean geometry under Manifold (Status: NoError). The backend fixes the problem without telling you it existed. Good for production, bad for quality gates — you lose the signal.

3. **`echo()` is a viable soft warning channel.** echo() output goes to stderr, is prefixed with `ECHO:`, and doesn't affect the exit code. Combined with assert() for hard stops, you get a two-tier feedback system: warnings that don't block, errors that do.

4. **Render time scaling is non-linear with CGAL.** Simple bracket: 46ms. Complex bracket (15 high-$fn cylinders, boolean grid): 12.5s. That's 270x slower for maybe 10x more geometry. CGAL boolean operations are the bottleneck. Manifold doesn't have this problem.

5. **The STL format itself has no parameter metadata.** Once rendered, the STL carries zero information about what parameters produced it. For traceability, you'd need to emit parameter snapshots alongside the STL — the summary JSON partially does this (bounding box, counts) but not the input parameters.

## Would I Use This Again?

**Yes — this is one of the best headless CAD tools for agent workflows.** Here's why:

**What's mechanically gateable right now (no extra tooling):**
- Parameter validation via `assert()` — hard stops with parseable errors, exit code 1
- Soft warnings via `echo()` — non-blocking stderr output
- Geometry metadata via `--summary-file` — JSON with dimensions, vertex counts, render time
- Visual verification via PNG export — agent can inspect renders
- Non-manifold detection (CGAL only) — warns on `Simple: no`

**What fails silently and needs external analysis:**
- Physical printability (wall thickness, overhang angles, bridging distances) — needs slicer integration (PrusaSlicer CLI can analyze STL)
- Self-intersecting geometry — OpenSCAD doesn't detect it
- Structural integrity — no FEA in OpenSCAD, would need external tools
- Negative dimension nonsense — needs assert() guards (not built-in)
- Manifold-repaired geometry losing the "was malformed" signal

**Agent workflow friction:**
- **Low:** CLI ergonomics are excellent. `-D` flags, `--quiet`, `--summary-file`, exit codes — all work as expected.
- **Medium:** Error messages are text-only (no structured JSON errors). Parseable but requires grep/regex.
- **Low:** Render times are negligible with Manifold backend (<100ms for complex parts). CGAL is prohibitively slow for iteration.
- **None:** No GUI dependency, no X11/display needed, no interactive prompts.

**Honest assessment for agent orchestration with structural enforcement:**

This is an excellent domain. The gate structure maps almost 1:1 to what orch already does:
- `assert()` in .scad = spawn gates (blocking, first-fail, parseable errors)
- `echo()` warnings = coaching signals (non-blocking, advisory)
- `--summary-file` JSON = event telemetry (structured metadata)
- exit code = completion gate (binary pass/fail)
- `-D` flags = parameterized spawning (design space exploration without code changes)

The gap is **physical printability validation**, which requires an external tool (slicer). But that's a second gate in the pipeline, not a blocker. The architecture would be: OpenSCAD assert gates → render → slicer validation gate → STL output.

**Recommended backend: Manifold.** The 232x speedup over CGAL makes iterative agent workflows practical. The only downside is losing the non-manifold warning signal, which can be mitigated by running a separate CGAL validation pass when needed.
