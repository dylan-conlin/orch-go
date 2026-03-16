# design-part

Create a parametric OpenSCAD part from a design specification.

## When to Use

- New part from scratch based on a design spec or description
- Porting an existing design to parametric OpenSCAD

## Workflow

1. **Read the design spec** from SPAWN_CONTEXT or attached spec file
2. **Enumerate requirements**: dimensions, features, constraints, purpose
3. **Write `.scad` file** in `parts/` with parameter validation using `lib/validate.scad`
4. **Render with Manifold** for fast iteration: `openscad -o exports/<name>.stl --backend manifold parts/<name>.scad`
5. **Iterate parameters** using `-D` flags — do NOT edit .scad for parameter exploration
6. **Add advisory warnings** via `echo()` for non-blocking guidance (thin walls, high $fn)
7. **Validate with CGAL**: `openscad -o /dev/null --backend cgal --quiet parts/<name>.scad`
8. **Run geometry gate**: `./gates/geometry-check.sh parts/<name>.scad` (if available)
9. **Export final outputs**: STL + PNG preview + summary JSON
10. **Record parameters** — document default values and valid ranges in .scad header comments

## .scad File Template

```openscad
// <name>.scad — <brief description>
// Parameters override via: openscad -D 'param=value'

// --- Parameters ---
width = 40;           // mm
height = 30;          // mm
depth = 20;           // mm
wall_thickness = 3;   // mm (minimum 0.8mm for FDM)
fn = 32;              // facet count for curves

// --- Layer 1: Parameter Validation ---
use <../lib/validate.scad>

validate_part(width, height, depth, wall_thickness, fn);
// Add part-specific validations:
// validate_range("param_name", value, min, max);
// validate_positive("param_name", value);

// Advisory warnings (non-blocking)
if (wall_thickness < 1.2)
    echo("WARN: wall_thickness=", wall_thickness, "mm is thin for structural parts");
if (fn > 128)
    echo("WARN: $fn=", fn, " will produce high polygon count");

// --- Geometry ---
module part_name() {
    // Build geometry here
}

part_name();
```

## Required Outputs

| File | Purpose |
|------|---------|
| `parts/<name>.scad` | Parametric source with assert gates |
| `exports/<name>.stl` | CGAL-validated mesh |
| `exports/<name>.png` | Visual preview (`--autocenter --viewall`) |
| `exports/<name>-summary.json` | Render metadata (`--summary-file`) |

## Constraints

- Every `.scad` file starts with `use <../lib/validate.scad>` and parameter validation
- Use Manifold backend for iteration, CGAL for final validation
- Polygon count < 200K for single parts
- All parameters must have sensible defaults and comments with units
- No magic numbers — use named variables
- Prefer `$fn=32` during development, `$fn=128` for final export only
- Design for `-D` flag overrides — parameters at file top, not buried in modules

## Validation Checklist

Before marking complete:

- [ ] All parameters have assert gates (via `validate_part()` or direct `assert()`)
- [ ] Renders successfully with CGAL backend (manifold check)
- [ ] Polygon count within budget (check `--summary-file` JSON)
- [ ] PNG preview looks correct (`--autocenter --viewall`)
- [ ] Parameters are overridable via `-D` flags (test at least one)
- [ ] No negative dimensions or degenerate geometry possible with default params
