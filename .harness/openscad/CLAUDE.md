# OpenSCAD Harness

Parametric 3D modeling with structural enforcement gates. Agent-driven design workflow using OpenSCAD CLI.

## Architecture

```
agent (Claude Code)
    │
    ├── openscad CLI          ← render, validate, export
    ├── prusa-slicer CLI      ← printability analysis (Layer 3)
    └── intent-gate script    ← LLM alignment check (Layer 4)
```

## Directory Structure

```
.harness/openscad/
├── CLAUDE.md              # This file — project constraints and workflow
├── parts/                 # .scad source files (one file per part)
├── lib/                   # Shared validation modules
│   └── validate.scad      # Layer 1 assert gate library
├── exports/               # Rendered outputs (STL, PNG, summary JSON)
├── sweeps/                # Parameter sweep results
├── gates/                 # Gate scripts (Layer 2-4)
├── test/                  # Test .scad files for gate validation
└── skills/                # Skill templates
    ├── design-part/       # Create new parametric parts
    └── iterate-design/    # Explore parameter space of existing parts
```

## Workflow

1. Write/modify `.scad` files in `parts/`
2. Include parameter validation: `use <../lib/validate.scad>`
3. Render with Manifold for fast iteration: `openscad -o exports/out.stl --backend manifold parts/file.scad`
4. Validate with CGAL for manifold correctness: `openscad -o /dev/null --backend cgal --quiet parts/file.scad`
5. Export PNG for visual check: `openscad -o exports/preview.png --autocenter --viewall parts/file.scad`
6. Run geometry gate: `./gates/geometry-check.sh parts/file.scad`
7. (When available) Run printability check: `prusa-slicer --export-gcode --info exports/out.stl`
8. (When available) Run intent gate: `./gates/intent-check.sh exports/out.stl design-spec.md`

## Constraints

### Render Backends

- **Manifold** for iteration (232x faster than CGAL). Use `-D` flags for parameter exploration.
- **CGAL** for final validation only. CGAL detects non-manifold geometry that Manifold silently repairs.
- Always run a CGAL validation pass before declaring a design complete.

### Parameter Validation (Layer 1 — assert gates)

Every `.scad` file MUST include parameter validation at the top using `lib/validate.scad`:

```openscad
use <../lib/validate.scad>

validate_part(width, height, depth, wall_thickness, fn, holes=[hole_diameter]);
validate_range("fillet_radius", fillet_radius, 0, min(wall_thickness, 5));
```

Pattern: `GATE FAIL:` prefix for grep-able error extraction. Exit code 1 on violation.

For custom validations, use `assert()` directly:
```openscad
assert(wall_thickness >= 0.8, str("GATE FAIL: wall_thickness=", wall_thickness, " < 0.8mm minimum"));
```

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
    openscad -o "exports/out_${thickness}.stl" parts/file.scad \
        -D "wall_thickness=${thickness}" --backend manifold --quiet
done
```

### $fn Budget

`$fn` controls facet count for curves. It explodes polygon count exponentially.

| Purpose | $fn | Use When |
|---------|-----|----------|
| Fast iteration | 32 | Developing geometry, testing parameters |
| Visual review | 64 | Checking appearance, screenshots |
| Export quality | 128 | Final STL for printing |

Gate: warn if `$fn > 128`, block if `$fn > 256`.

### Common Gotchas

- **Negative dimensions produce geometry** — always assert positivity
- **Manifold silently repairs non-manifold** — always CGAL-validate before export
- **$fn controls facet count exponentially** — $fn=360 on cylinder = 720 facets
- **First-fail assert semantics** — only first assertion fires; use `validate_part()` for batch checking
- **No printability awareness** — OpenSCAD exit 0 does NOT mean printable
- **STL has no parameter metadata** — emit parameter snapshots alongside exports
- **No summary JSON on failure** — `--summary-file` is not created when assert fires; rely on stderr parsing
- **`--hardwarnings` doesn't catch manifold warnings** — only language-level warnings, not geometry

## Gate Stack (4 layers)

| Layer | Type | Tool | Blocks? | Status |
|-------|------|------|---------|--------|
| 1 — Parameter validation | Execution (hard) | `assert()` in .scad via `lib/validate.scad` | Yes | Implemented + tested (9 tests) |
| 2 — Geometry validation | Execution (hard) | `gates/geometry-check.sh` | Yes | Implemented + tested (5 tests) |
| 3 — Printability | Execution (hard) | PrusaSlicer CLI | Yes | Not yet integrated |
| 4 — Intent alignment | Judgment (soft) | LLM-as-gate | Advisory only | Not yet implemented |

## Required Outputs per Design

- `parts/<name>.scad` — parametric source with assert gates
- `exports/<name>.stl` — rendered mesh (CGAL-validated)
- `exports/<name>.png` — visual preview (`--autocenter --viewall`)
- `exports/<name>-summary.json` — render metadata (`--summary-file`)
